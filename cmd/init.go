package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipeinoue0/crower/internal/adapters/shellhook"
	"github.com/felipeinoue0/crower/internal/adapters/shellsetup"
	"github.com/felipeinoue0/crower/internal/domain/catalog"

	"github.com/spf13/cobra"
)

var initApply bool
var initAll bool
var initAliasesOnly bool

// initCmd explains, or applies, the environment crower needs: the install
// directory on PATH, and the hook that makes every managed command a command of
// its own in the shell.
var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "Set up your shell: PATH, native aliases and completion.",
	Long: `Set up your shell so every crower command can be run by its own name.

With no shell, crower shows what your shell needs. With a shell, it prints the
hook that shell evaluates at startup: one alias per command name and alias, a
re-sync after create/update/delete, and tab completion.

	bash        eval "$(crower init bash)"          in ~/.bashrc
	zsh         eval "$(crower init zsh)"           in ~/.zshrc
	fish        crower init fish | source           in config.fish
	PowerShell  crower init powershell | Out-String | Invoke-Expression   in $PROFILE
	cmd.exe     crower init cmd > "%USERPROFILE%\crower.doskey"
	            doskey /macrofile="%USERPROFILE%\crower.doskey"

--apply writes the right line into your startup file for you. A command whose
name is already taken in the shell (a program on PATH, a builtin, your own
alias) is skipped, so crower never hides git or cd.`,
	Example: `  crower init
  crower init --apply
  eval "$(crower init zsh)"
  crower init powershell | Out-String | Invoke-Expression`,
	Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	ValidArgs: shellhook.Shells,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			// The hook runs on every new shell: it must never wait on the
			// network for the update notice.
			skipUpdateNotice = true
			return printHook(cmd, args[0])
		}

		if initAll {
			return setUpAll(cmd)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		// The shell that started crower, which may differ from $SHELL (a
		// bash opened from zsh, cmd or either PowerShell on Windows).
		shell := shellsetup.Parent()
		if shell == "" {
			shell = detectShell()
		}

		switch shell {
		case shellsetup.Cmd, shellsetup.PowerShell, shellsetup.Pwsh:
			return initWindowsShell(cmd, shell)
		}

		target, line := initTarget(shell, home)
		hookFile, hookLine := hookTarget(shell, home, runtime.GOOS)

		if target == "" && hookFile == "" {
			fmt.Fprintln(streams.Out, "Could not recognise your shell.")
			fmt.Fprintf(streams.Out, "Run \"crower init <shell>\" with one of: %s.\n", strings.Join(shellhook.Shells, ", "))
			return nil
		}

		if !initApply {
			fmt.Fprintf(streams.Out, "Shell:   %s\n", shell)
			if target != "" {
				fmt.Fprintf(streams.Out, "PATH:    %s\n", line)
			}
			fmt.Fprintf(streams.Out, "Hook:    %s\n", hookLine)
			fmt.Fprintf(streams.Out, "Startup: %s\n", hookFile)
			fmt.Fprintln(streams.Out)
			fmt.Fprintln(streams.Out, "The hook turns every crower command into a command of its own, with tab")
			fmt.Fprintln(streams.Out, `completion. Run "crower init --apply" to add these lines now.`)
			return nil
		}

		if target != "" {
			changed, err := applyInit(target, line)
			if err != nil {
				return err
			}

			if changed {
				fmt.Fprintf(streams.Out, "Added PATH to %s\n", target)
			}
		}

		changed, err := applyInit(hookFile, hookLine)
		if err != nil {
			return err
		}

		if changed {
			fmt.Fprintf(streams.Out, "Added the crower hook to %s\n", hookFile)
			fmt.Fprintln(streams.Out, "Open a new terminal (or source the file) and run any command by its name.")
		} else {
			fmt.Fprintf(streams.Out, "%s already loads crower. Nothing changed.\n", hookFile)
		}

		return nil
	},
}

// initWindowsShell shows, or with --apply performs, the setup of cmd or a
// PowerShell, whose startup is not a file crower can name in advance.
func initWindowsShell(cmd *cobra.Command, shell string) error {
	if !initApply {
		fmt.Fprintf(streams.Out, "Shell:   %s\n", displayName(shell))
		fmt.Fprintln(streams.Out)
		fmt.Fprintln(streams.Out, "The hook turns every crower command into a command of its own, with tab")
		fmt.Fprintln(streams.Out, `completion. Run "crower init --apply" to set it up, or "crower init --all"`)
		fmt.Fprintln(streams.Out, "for every shell on this computer.")

		return nil
	}

	result, err := installHook(cmd, shell)
	if err != nil {
		return err
	}

	if result.Changed {
		fmt.Fprintf(streams.Out, "%s now loads the crower hook (%s).\n", displayName(shell), result.Where)
	} else {
		fmt.Fprintf(streams.Out, "%s already loads crower (%s). Nothing changed.\n", displayName(shell), result.Where)
	}

	if result.Warning != "" {
		fmt.Fprintf(streams.Out, "%s. To allow it: Set-ExecutionPolicy -Scope CurrentUser RemoteSigned\n", result.Warning)
	}

	fmt.Fprintf(streams.Out, "To use it here, %s.\n", reloadHint(shell, result.Where))

	return nil
}

// printHook writes the hook, or only its alias block with --aliases, for shell.
func printHook(cmd *cobra.Command, shell string) error {
	script, err := buildHook(cmd, shell, initAliasesOnly)
	if err != nil {
		return err
	}

	fmt.Fprint(streams.Out, script)

	return nil
}

// buildHook is the hook for shell, or only its alias block.
func buildHook(cmd *cobra.Command, shell string, aliasesOnly bool) (string, error) {
	opts := shellhook.Options{Bin: invokedAs(), Reserved: subcommandWords(rootCmd)}

	// Outside a run (first-run setup, the refresh after a change) cobra has
	// not merged the global flags into this command yet, and newSession
	// reads --config and --setup from them.
	_ = cmd.LocalFlags()

	// cmd's macros reload their file after every crower run, which is how a
	// cmd window picks up a new command; only a file crower manages qualifies.
	if shell == shellsetup.Cmd {
		if file, err := macroFilePath(); err == nil && fileExists(file) {
			opts.MacroFile = file
		}
	}

	if exe, err := os.Executable(); err == nil {
		opts.Exe = exe
	}

	// A configuration that cannot be read still yields a working hook -- the
	// wrapper and completion -- so a broken file never breaks the shell.
	var names []string
	if cfg, err := newSession(cmd); err == nil {
		// Only the catalog is needed; release the log file, which Windows
		// would otherwise keep locked for the rest of the run.
		defer func() { _ = cfg.Logger.Close() }()

		names = commandWords(cfg.Catalog())
		opts.Dirs = changeDirs(cfg.Catalog())
		opts.Here = hereWords(cfg.Catalog())
	} else {
		fmt.Fprintln(streams.Err, "crower: no command aliases loaded:", err)
	}

	if aliasesOnly {
		return shellhook.Aliases(shell, names, opts)
	}

	var completion bytes.Buffer

	switch shell {
	case "bash":
		_ = rootCmd.GenBashCompletionV2(&completion, true)
	case "zsh":
		_ = rootCmd.GenZshCompletion(&completion)
	case "fish":
		_ = rootCmd.GenFishCompletion(&completion, true)
	case "powershell":
		_ = rootCmd.GenPowerShellCompletionWithDesc(&completion)
	}

	opts.Completion = completion.String()

	return shellhook.Script(shell, names, opts)
}

// commandWords is every name and alias a shell can call.
func commandWords(commands catalog.Catalog) []string {
	words := commands.Names()

	for alias := range commands.AliasIndex() {
		words = append(words, alias)
	}

	return words
}

// changeDirs maps every name and alias of a command that only changes folder
// to that folder, so the hook can move the shell instead of a child process.
// A Steps command is never one: its exec line is not what runs.
func changeDirs(commands catalog.Catalog) map[string]string {
	dirs := map[string]string{}

	for _, command := range commands.Commands() {
		if len(command.Steps) > 0 {
			continue
		}

		dir, ok := shellhook.ChangeDir(command.Exec)
		if !ok {
			continue
		}

		dirs[command.Name] = dir
		for _, alias := range command.Aliases {
			dirs[alias] = dir
		}
	}

	return dirs
}

// hereWords is every name and alias of a command that runs in the calling
// shell (--here). A command whose stored form cannot run there is left out, so
// the hook never routes it to `execute --emit`, which would refuse it.
func hereWords(commands catalog.Catalog) map[string]bool {
	words := map[string]bool{}

	for _, command := range commands.Commands() {
		if !command.Here || command.CheckHere() != nil {
			continue
		}

		words[command.Name] = true
		for _, alias := range command.Aliases {
			words[alias] = true
		}
	}

	return words
}

// subcommandWords is every word `crower <word>` resolves to a subcommand for:
// each subcommand's name and aliases, plus cobra's own help and completion.
func subcommandWords(root *cobra.Command) []string {
	words := []string{"help", "completion"}

	for _, sub := range root.Commands() {
		words = append(words, sub.Name())
		words = append(words, sub.Aliases...)
	}

	return words
}

// invokedAs is the program name the user typed, "crower" or "cr", so the hook
// calls the binary that is certainly on PATH.
func invokedAs() string {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(os.Args[0])), ".exe")
	if name == "cr" {
		return "cr"
	}

	return "crower"
}

// detectShell is the running shell. Windows sets no $SHELL; PowerShell is its
// interactive default.
func detectShell() string {
	if shell := shellName(os.Getenv("SHELL")); shell != "" {
		return shell
	}

	if runtime.GOOS == "windows" {
		return "powershell"
	}

	return ""
}

// hookTarget is the startup file and the line that loads the hook.
func hookTarget(shell, home, goos string) (string, string) {
	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc"), `command -v crower >/dev/null 2>&1 && eval "$(crower init bash)"`
	case "zsh":
		return filepath.Join(home, ".zshrc"), `command -v crower >/dev/null 2>&1 && eval "$(crower init zsh)"`
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), `command -q crower; and crower init fish | source`
	case "powershell":
		profile := filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
		if goos == "windows" {
			profile = filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		}

		return profile, `if (Get-Command crower -ErrorAction SilentlyContinue) { crower init powershell | Out-String | Invoke-Expression }`
	default:
		return "", ""
	}
}

// shellName turns $SHELL into the name the commands and the startup files use.
func shellName(shell string) string {
	base := strings.ToLower(filepath.Base(shell))

	switch {
	case strings.Contains(base, "zsh"):
		return "zsh"
	case strings.Contains(base, "bash"):
		return "bash"
	case strings.Contains(base, "fish"):
		return "fish"
	case strings.Contains(base, "pwsh"), strings.Contains(base, "powershell"):
		return "powershell"
	default:
		return ""
	}
}

// initTarget is the startup file and the line that puts the install directory
// on PATH for one shell.
func initTarget(shell, home string) (string, string) {
	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc"), `export PATH="$HOME/.local/bin:$PATH"`
	case "zsh":
		return filepath.Join(home, ".zshrc"), `export PATH="$HOME/.local/bin:$PATH"`
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), `fish_add_path "$HOME/.local/bin"`
	default:
		return "", ""
	}
}

// applyInit appends the line unless the file already contains it.
//
// It reports whether the file changed: a second run is a no-op, because a
// startup file that grows on every init is noise.
func applyInit(path, line string) (bool, error) {
	body, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if strings.Contains(string(body), line) {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "\n%s\n", line); err != nil {
		return false, err
	}

	return true, nil
}

func init() {
	initCmd.Flags().BoolVar(&initApply, "apply", false, "add the PATH line and the hook to your shell startup file")
	initCmd.Flags().BoolVar(&initAll, "all", false, "set up every shell installed on this computer (bash, zsh, fish, PowerShell, cmd)")
	initCmd.Flags().BoolVar(&initAliasesOnly, "aliases", false, "print only the alias block (used by the hook to re-sync)")
	_ = initCmd.Flags().MarkHidden("aliases")
	rootCmd.AddCommand(initCmd)
}
