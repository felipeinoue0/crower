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

	return b.String(), nil
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
		fmt.Fprintf(b, "if not type -q %[1]s\n    function %[1]s --description 'crower command'\n        command %[2]s execute %[1]s --; and cd %[3]s\n    end\n    set -ga __crower_names %[1]s\nend\n", name, opts.Bin, fishWord(dir))
	case "powershell":
		fmt.Fprintf(b, "if (-not (Get-Command '%[1]s' -ErrorAction SilentlyContinue)) { function global:%[1]s { & $global:__crower_exe execute '%[1]s' '--'; if ($LASTEXITCODE -eq 0) { Set-Location -LiteralPath %[2]s } }; $global:__crower_names += '%[1]s' }\n", name, psPath(dir))
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
  if ! type "$1" >/dev/null 2>&1; then
    alias "$1=__crower_cd $1 $2"
    __crower_names="${__crower_names-} $1"
  fi
}

__crower_cd() {
  command %[2]s execute "$1" -- && builtin cd -- "$2"
}

__crower_sync() {
  local __crower_n
  for __crower_n in %[3]s; do unalias "$__crower_n" 2>/dev/null; done
  __crower_names=""
  eval "$(command %[2]s init %[1]s --aliases 2>/dev/null)"
}

crower() {
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
%[5]s`, shell, opts.Bin, split, strings.Join(mutating, "|"), aliases)

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
    command %[1]s init fish --aliases 2>/dev/null | source
end

function crower --wraps crower --description 'crower, re-syncing commands after a change'
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
%[3]s`, opts.Bin, strings.Join(mutating, " "), aliases)

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

function global:__crower_sync {
    foreach ($n in $global:__crower_names) { Remove-Item -Path "Function:\$n" -ErrorAction SilentlyContinue }
    $global:__crower_names = @()
    (& $global:__crower_exe init powershell --aliases 2>$null) | Out-String | Invoke-Expression
}

function global:crower {
    & $global:__crower_exe @args
    $status = $LASTEXITCODE
    if ($args | Where-Object { $_ -in @(%[3]s) }) { __crower_sync }
    $global:LASTEXITCODE = $status
}

function global:cr { crower @args }

%[4]s`, opts.Bin, psQuote(exe), strings.Join(quoted, ", "), aliases)

	if opts.Completion != "" {
		b.WriteString("\n# Tab completion for crower, cr and every command name.\n")
		b.WriteString(opts.Completion)
		b.WriteString("\nRegister-ArgumentCompleter -CommandName 'cr' -ScriptBlock ${__crowerCompleterBlock}\n")
	}

	return b.String()
}

// psQuote escapes a value for a single-quoted PowerShell string.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
