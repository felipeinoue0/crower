// Package shellhook writes the script a shell evaluates to call every managed
// command by its own name: `deploy` instead of `crower deploy`.
//
// The script is generated, never installed: `eval "$(crower init zsh)"` in the
// startup file runs it on every new shell, so it always reflects the current
// command set, and a crower create/update/delete in that shell re-syncs it.
//
// A name that already means something to the shell -- a binary on PATH, a
// builtin, a keyword, the user's own alias or function -- is skipped at the
// moment the shell defines it. A crower command can never shadow `git` or `cd`.
package shellhook

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Shells lists the shells a hook can be written for, in the order help shows.
var Shells = []string{"bash", "zsh", "fish", "powershell", "cmd"}

// mutating are the subcommands that can change the command set. Running one of
// them through the shell wrapper re-syncs the aliases in that shell.
var mutating = []string{"create", "update", "delete", "import", "revert", "restore", "reset", "setup"}

// validName is the set of names every supported shell accepts as a bare
// command word without quoting.
var validName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+-]*$`)

// reserved are names the hook itself owns.
var reserved = []string{"crower", "cr"}

// Usable filters names down to the ones a shell can define: valid words, not
// crower's own names, without duplicates, in a stable order.
func Usable(names []string) []string {
	out := make([]string, 0, len(names))

	for _, name := range names {
		if !validName.MatchString(name) || slices.Contains(reserved, name) || slices.Contains(out, name) {
			continue
		}

		out = append(out, name)
	}

	slices.Sort(out)

	return out
}

// Options is what a script needs besides the names.
type Options struct {
	// Bin is the program the shell calls: "crower" or "cr", as invoked.
	Bin string
	// Exe is the absolute path to the program, for PowerShell, whose wrapper
	// function would otherwise call itself.
	Exe string
	// Completion is the shell's completion script, embedded so the hook costs
	// one process at startup, not two.
	Completion string
	// Dirs maps a name to the folder its command only changes into (see
	// ChangeDir). Such a name is defined to move the shell itself, because a
	// cd run by `crower execute` happens in a child process and is lost.
	Dirs map[string]string
	// Reserved are crower's own subcommand names and aliases. `crower <word>`
	// runs the subcommand when word is one of them, so the wrapper never
	// treats such a word as a stored cd command; `crower execute <word>` still
	// does.
	Reserved []string
}

// reservedWords is Reserved filtered to words every shell can match
// literally, sorted.
func (o Options) reservedWords() []string {
	out := make([]string, 0, len(o.Reserved))

	for _, word := range o.Reserved {
		if validName.MatchString(word) && !slices.Contains(out, word) {
			out = append(out, word)
		}
	}

	slices.Sort(out)

	return out
}

// cdNames are the usable names whose command only changes folder, sorted.
func cdNames(names []string, opts Options) []string {
	var out []string

	for _, name := range Usable(names) {
		if _, ok := opts.Dirs[name]; ok {
			out = append(out, name)
		}
	}

	return out
}

// Script is the full hook for shell: the wrapper, the sync function, the
// aliases and the completion.
func Script(shell string, names []string, opts Options) (string, error) {
	if opts.Bin == "" {
		opts.Bin = "crower"
	}

	switch shell {
	case "bash", "zsh":
		return posixScript(shell, names, opts), nil
	case "fish":
		return fishScript(names, opts), nil
	case "powershell":
		return powershellScript(names, opts), nil
	case "cmd":
		return Aliases(shell, names, opts)
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(Shells, ", "))
	}
}

// Aliases is only the alias block, which the sync function evaluates again
// after a command changed the set.
func Aliases(shell string, names []string, opts Options) (string, error) {
	if opts.Bin == "" {
		opts.Bin = "crower"
	}

	var b strings.Builder

	for _, name := range Usable(names) {
		if dir, ok := opts.Dirs[name]; ok {
			if err := writeChangeDir(&b, shell, name, dir, opts); err != nil {
				return "", err
			}

			continue
		}

		switch shell {
		case "bash", "zsh":
			fmt.Fprintf(&b, "__crower_def %s\n", name)
		case "fish":
			fmt.Fprintf(&b, "if not type -q %[1]s\n    function %[1]s --description 'crower command'\n        command %[2]s execute %[1]s -- $argv\n    end\n    set -ga __crower_names %[1]s\nend\n", name, opts.Bin)
		case "powershell":
			fmt.Fprintf(&b, "if (-not (Get-Command '%[1]s' -ErrorAction SilentlyContinue)) { function global:%[1]s { & $global:__crower_exe execute '%[1]s' '--' @args }; $global:__crower_names += '%[1]s' }\n", name)
		case "cmd":
			fmt.Fprintf(&b, "%[1]s=%[2]s execute %[1]s -- $*\n", name, opts.Bin)
		default:
			return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(Shells, ", "))
		}
	}

	if shell == "cmd" {
		writeCmdWrappers(&b, names, opts)
	}

	return b.String(), nil
}

// writeCmdWrappers defines `crower` and `cr` as doskey macros when any command
// only changes folder. cmd has no functions, so each macro is one `if` chain:
// `cr execute <name>` and `cr <name>` (unless name is a subcommand) run
// crower and then `cd /d` in this window; anything else falls through to the
// program, which a doskey macro never expands into itself.
func writeCmdWrappers(b *strings.Builder, names []string, opts Options) {
	dirs := cdNames(names, opts)
	if len(dirs) == 0 {
		return
	}

	reserved := opts.reservedWords()

	for _, bin := range []string{"crower", "cr"} {
		var chain []string

		for _, name := range dirs {
			jump := fmt.Sprintf("(%s execute %s -- && cd /d %s)", opts.Bin, name, cmdPath(opts.Dirs[name]))
			chain = append(chain, fmt.Sprintf(`if "$1 $2"=="execute %s" %s`, name, jump))

			if !slices.Contains(reserved, name) {
				chain = append(chain, fmt.Sprintf(`if "$1"=="%s" %s`, name, jump))
			}
		}

		fmt.Fprintf(b, "%s=%s else %s $*\n", bin, strings.Join(chain, " else "), bin)
	}
}

// writeChangeDir defines name to run its command through crower -- which
// records the run and fails on a missing folder -- and then, only on success,
// to change the calling shell's directory. Arguments are not forwarded: a cd
// command takes none.
func writeChangeDir(b *strings.Builder, shell, name, dir string, opts Options) error {
	switch shell {
	case "bash", "zsh":
		fmt.Fprintf(b, "__crower_cdef %s %s\n", name, posixQuote(posixWord(dir)))
	case "fish":
		fmt.Fprintf(b, "set -ga __crower_cdnames %[1]s\nset -ga __crower_cddirs %[3]s\nif not type -q %[1]s\n    function %[1]s --description 'crower command'\n        command %[2]s execute %[1]s --; and cd %[3]s\n    end\n    set -ga __crower_names %[1]s\nend\n", name, opts.Bin, fishWord(dir))
	case "powershell":
		fmt.Fprintf(b, "$global:__crower_dirs['%[1]s'] = %[2]s\nif (-not (Get-Command '%[1]s' -ErrorAction SilentlyContinue)) { function global:%[1]s { & $global:__crower_exe execute '%[1]s' '--'; if ($LASTEXITCODE -eq 0) { Set-Location -LiteralPath %[2]s } }; $global:__crower_names += '%[1]s' }\n", name, psPath(dir))
	case "cmd":
		fmt.Fprintf(b, "%[1]s=%[2]s execute %[1]s -- && cd /d %[3]s\n", name, opts.Bin, cmdPath(dir))
	default:
		return fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(Shells, ", "))
	}

	return nil
}

func posixScript(shell string, names []string, opts Options) string {
	// zsh does not split an unquoted parameter on spaces; ${=var} asks it to.
	split := "${__crower_names-}"
	if shell == "zsh" {
		split = "${=__crower_names-}"
	}

	aliases, _ := Aliases(shell, names, opts)

	var b strings.Builder

	fmt.Fprintf(&b, `# crower shell integration for %[1]s.
# Load it from your startup file with:  eval "$(%[2]s init %[1]s)"
# Every crower command becomes a command of its own: run "deploy", not "%[2]s deploy".

__crower_def() {
  if ! type "$1" >/dev/null 2>&1; then
    alias "$1=command %[2]s execute $1 --"
    __crower_names="${__crower_names-} $1"
  fi
}

__crower_cdef() {
  __crower_cmap="${__crower_cmap-}$1 $2
"
  if ! type "$1" >/dev/null 2>&1; then
    alias "$1=__crower_cd $1 $2"
    __crower_names="${__crower_names-} $1"
  fi
}

__crower_cd() {
  command %[2]s execute "$1" -- && builtin cd -- "$2"
}

# __crower_cdir sets __crower_w to the folder word of a cd command, if $1 is one.
__crower_cdir() {
  local __crower_l
  __crower_w=""
  while IFS= read -r __crower_l; do
    case "$__crower_l" in
      "$1 "*) __crower_w="${__crower_l#"$1 "}"; return 0 ;;
    esac
  done <<__CROWER_MAP__
${__crower_cmap-}
__CROWER_MAP__
  return 1
}

__crower_sync() {
  local __crower_n
  for __crower_n in %[3]s; do unalias "$__crower_n" 2>/dev/null; done
  __crower_names=""
  __crower_cmap=""
  eval "$(command %[2]s init %[1]s --aliases 2>/dev/null)"
}

crower() {
  # "crower <name>" and "crower execute <name>" for a cd command change this
  # shell's folder; a child process could not.
  local __crower_key=""
  case "${1-}" in
    execute) __crower_key="${2-}" ;;
    %[6]s) ;;
    *) __crower_key="${1-}" ;;
  esac
  if [ -n "$__crower_key" ] && __crower_cdir "$__crower_key"; then
    eval "__crower_cd \"\$__crower_key\" $__crower_w"
    return
  fi

  command %[2]s "$@"
  local __crower_status=$?
  local __crower_arg
  for __crower_arg in "$@"; do
    case "$__crower_arg" in
      %[4]s) __crower_sync; break ;;
    esac
  done
  return $__crower_status
}

cr() { crower "$@"; }

__crower_names=""
__crower_cmap=""
%[5]s`, shell, opts.Bin, split, strings.Join(mutating, "|"), aliases, strings.Join(append(opts.reservedWords(), "-*"), "|"))

	if opts.Completion != "" {
		b.WriteString("\n# Tab completion for crower, cr and every command name.\n")

		if shell == "zsh" {
			b.WriteString("if (( $+functions[compdef] )); then\n")
			b.WriteString(opts.Completion)
			b.WriteString("\ncompdef _crower crower cr\nfi\n")
		} else {
			b.WriteString(opts.Completion)
			b.WriteString("\ncomplete -o default -F __start_crower cr\n")
		}
	}

	return b.String()
}

func fishScript(names []string, opts Options) string {
	aliases, _ := Aliases("fish", names, opts)

	var b strings.Builder

	fmt.Fprintf(&b, `# crower shell integration for fish.
# Load it from config.fish with:  %[1]s init fish | source
# Every crower command becomes a command of its own: run "deploy", not "%[1]s deploy".

function __crower_sync
    for __crower_n in $__crower_names
        functions -e $__crower_n
    end
    set -g __crower_names
    set -g __crower_cdnames
    set -g __crower_cddirs
    command %[1]s init fish --aliases 2>/dev/null | source
end

function crower --wraps crower --description 'crower, re-syncing commands after a change'
    # "crower <name>" and "crower execute <name>" for a cd command change this
    # shell's folder; a child process could not.
    set -l __crower_key
    switch "$argv[1]"
        case execute
            set __crower_key $argv[2]
        case %[4]s
        case '*'
            set __crower_key $argv[1]
    end
    if test -n "$__crower_key"
        set -l __crower_i (contains -i -- $__crower_key $__crower_cdnames)
        if test -n "$__crower_i"
            command %[1]s execute $__crower_key --; and cd $__crower_cddirs[$__crower_i]
            return $status
        end
    end

    command %[1]s $argv
    set -l __crower_status $status
    for __crower_arg in $argv
        if contains -- $__crower_arg %[2]s
            __crower_sync
            break
        end
    end
    return $__crower_status
end

function cr --wraps crower
    crower $argv
end

set -g __crower_names
set -g __crower_cdnames
set -g __crower_cddirs
%[3]s`, opts.Bin, strings.Join(mutating, " "), aliases, strings.Join(append(opts.reservedWords(), "'-*'", "''"), " "))

	if opts.Completion != "" {
		b.WriteString("\n# Tab completion for crower, cr and every command name.\n")
		b.WriteString(opts.Completion)
		b.WriteString("\ncomplete -c cr -w crower\n")
	}

	return b.String()
}

func powershellScript(names []string, opts Options) string {
	aliases, _ := Aliases("powershell", names, opts)

	exe := opts.Exe
	if exe == "" {
		exe = opts.Bin
	}

	quoted := make([]string, len(mutating))
	for i, m := range mutating {
		quoted[i] = "'" + m + "'"
	}

	var b strings.Builder

	fmt.Fprintf(&b, `# crower shell integration for PowerShell.
# Load it from $PROFILE with:  %[1]s init powershell | Out-String | Invoke-Expression
# Every crower command becomes a command of its own: run "deploy", not "%[1]s deploy".

$global:__crower_exe = '%[2]s'
$global:__crower_names = @()
$global:__crower_dirs = [System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)

function global:__crower_sync {
    foreach ($n in $global:__crower_names) { Remove-Item -Path "Function:\$n" -ErrorAction SilentlyContinue }
    $global:__crower_names = @()
    $global:__crower_dirs = [System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)
    (& $global:__crower_exe init powershell --aliases 2>$null) | Out-String | Invoke-Expression
}

function global:crower {
    # "crower <name>" and "crower execute <name>" for a cd command change this
    # shell's folder; a child process could not.
    $key = $null
    if ($args.Count -ge 2 -and $args[0] -ceq 'execute') { $key = [string]$args[1] }
    elseif ($args.Count -ge 1 -and -not ([string]$args[0]).StartsWith('-') -and $args[0] -cnotin @(%[5]s)) { $key = [string]$args[0] }
    if ($key -and $global:__crower_dirs.ContainsKey($key)) {
        & $global:__crower_exe execute $key '--'
        if ($LASTEXITCODE -eq 0) { Set-Location -LiteralPath $global:__crower_dirs[$key] }
        return
    }

    & $global:__crower_exe @args
    $status = $LASTEXITCODE
    if ($args | Where-Object { $_ -in @(%[3]s) }) { __crower_sync }
    $global:LASTEXITCODE = $status
}

function global:cr { crower @args }

%[4]s`, opts.Bin, psQuote(exe), strings.Join(quoted, ", "), aliases, psList(opts.reservedWords()))

	if opts.Completion != "" {
		b.WriteString("\n# Tab completion for crower, cr and every command name.\n")
		b.WriteString(opts.Completion)
		b.WriteString("\nRegister-ArgumentCompleter -CommandName 'cr' -ScriptBlock ${__crowerCompleterBlock}\n")
	}

	return b.String()
}

// psList is words as a comma-separated list of single-quoted strings.
func psList(words []string) string {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = "'" + psQuote(w) + "'"
	}

	return strings.Join(quoted, ", ")
}

// psQuote escapes a value for a single-quoted PowerShell string.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
