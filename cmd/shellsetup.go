package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/felipeinoue0/crower/internal/adapters/shellhook"
	"github.com/felipeinoue0/crower/internal/adapters/shellsetup"
	"github.com/felipeinoue0/crower/internal/adapters/terminal"
	"github.com/felipeinoue0/crower/internal/domain/catalog"

	"github.com/spf13/cobra"
)

// Setting up a shell means making it load the hook at start: a line in its
// startup file, or for cmd a macro file that AutoRun loads. The hook is the
// only way a crower command can change the terminal's folder or variables,
// and these functions are what let a user get it without knowing it exists.

// crowerHome is the folder crower keeps its own files in.
func crowerHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "crower"), nil
}

// macroFilePath is the doskey file cmd's AutoRun loads.
func macroFilePath() (string, error) {
	dir, err := crowerHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "crower.doskey"), nil
}

// answersPath remembers the shells the user declined.
func answersPath() (string, error) {
	dir, err := crowerHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "shell-setup.json"), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// setupResult says what setting up one shell did.
type setupResult struct {
	// Where is the file (or registry value) that now loads the hook.
	Where string
	// Changed is false when it already did.
	Changed bool
	// Warning is a step the user must still take, or "".
	Warning string
}

// errCannotSetUp is returned for a shell this platform has no setup for.
var errCannotSetUp = errors.New("cannot be set up on this system")

// hookFile is the startup file of a POSIX shell or fish, and the line that
// loads the hook from it.
func hookFile(shell string) (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	file, line := hookTarget(shell, home, runtime.GOOS)
	if file == "" {
		return "", "", fmt.Errorf("%s: %w", shell, errCannotSetUp)
	}

	return file, line, nil
}

// powershellProfile asks the PowerShell itself where its profile is. The path
// differs between Windows PowerShell and PowerShell 7, and Documents may be
// redirected (OneDrive), so guessing it misses real setups.
func powershellProfile(exe string) (string, error) {
	out, err := osexec.Command(exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "$PROFILE.CurrentUserCurrentHost").Output()
	if err != nil {
		return "", fmt.Errorf("ask %s for its profile: %w", exe, err)
	}

	profile := strings.TrimSpace(string(out))
	if profile == "" {
		return "", fmt.Errorf("%s reported no profile path", exe)
	}

	return profile, nil
}

// powershellPolicy is the execution policy in effect for the current user.
func powershellPolicy(exe string) string {
	out, err := osexec.Command(exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "Get-ExecutionPolicy").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// policyBlocksProfile reports whether an execution policy stops a profile the
// user wrote from running.
func policyBlocksProfile(policy string) bool {
	return strings.EqualFold(policy, "Restricted") || strings.EqualFold(policy, "AllSigned")
}

// allowProfileScripts sets the current user's policy to RemoteSigned: scripts
// written on this machine run, downloaded ones must be signed. It is the
// default PowerShell 7 already uses on Windows.
func allowProfileScripts(exe string) error {
	return osexec.Command(exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned -Force").Run()
}

const powershellHookLine = `if (Get-Command crower -ErrorAction SilentlyContinue) { crower init powershell | Out-String | Invoke-Expression }`

// hookInstalled reports whether the shell already loads the hook at start.
// It does not start a PowerShell: an earlier setup is remembered instead.
func hookInstalled(shell string) bool {
	switch shell {
	case shellsetup.Bash, shellsetup.Zsh, shellsetup.Fish:
		file, _, err := hookFile(shell)
		if err != nil {
			return false
		}

		body, err := os.ReadFile(file)

		return err == nil && strings.Contains(string(body), "init "+shell)
	case shellsetup.Cmd:
		file, err := macroFilePath()
		if err != nil {
			return false
		}

		existing, err := readAutoRun()

		return err == nil && shellsetup.HasAutoRun(existing, file)
	default:
		path, err := answersPath()
		return err == nil && shellsetup.Installed(path, shell)
	}
}

// installHook makes shell load the hook at start. It is idempotent.
func installHook(cmd *cobra.Command, shell string) (setupResult, error) {
	switch shell {
	case shellsetup.Bash, shellsetup.Zsh, shellsetup.Fish:
		file, line, err := hookFile(shell)
		if err != nil {
			return setupResult{}, err
		}

		changed, err := applyInit(file, line)

		return setupResult{Where: file, Changed: changed}, err

	case shellsetup.Pwsh, shellsetup.PowerShell:
		profile, err := powershellProfile(shell)
		if err != nil {
			return setupResult{}, err
		}

		if err := os.MkdirAll(filepath.Dir(profile), 0o755); err != nil {
			return setupResult{}, err
		}

		changed, err := applyInit(profile, powershellHookLine)
		if err != nil {
			return setupResult{}, err
		}

		if path, err := answersPath(); err == nil {
			_ = shellsetup.Remember(path, shell)
		}

		result := setupResult{Where: profile, Changed: changed}
		if policy := powershellPolicy(shell); policyBlocksProfile(policy) {
			result.Warning = fmt.Sprintf("PowerShell's execution policy (%s) stops the profile from running", policy)
		}

		return result, nil

	case shellsetup.Cmd:
		if runtime.GOOS != "windows" {
			return setupResult{}, fmt.Errorf("cmd: %w", errCannotSetUp)
		}

		file, err := writeMacroFile(cmd)
		if err != nil {
			return setupResult{}, err
		}

		existing, err := readAutoRun()
		if err != nil {
			return setupResult{}, err
		}

		value, changed := shellsetup.WithAutoRun(existing, file)
		if changed {
			if err := writeAutoRun(value); err != nil {
				return setupResult{}, err
			}
		}

		return setupResult{Where: `HKCU\` + shellsetup.AutoRunKey + `\` + shellsetup.AutoRunValue, Changed: changed}, nil

	default:
		return setupResult{}, fmt.Errorf("%s: %w", shell, errCannotSetUp)
	}
}

// writeMacroFile writes cmd's macros for the current commands and returns the
// file. It is also how a cmd window learns about a new command: the macros
// reload this file after every crower run.
func writeMacroFile(cmd *cobra.Command) (string, error) {
	file, err := macroFilePath()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}

	// Create it first: buildHook only points the macros at a file that
	// exists.
	if !fileExists(file) {
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			return "", err
		}
	}

	script, err := buildHook(cmd, shellsetup.Cmd, false)
	if err != nil {
		return "", err
	}

	return file, os.WriteFile(file, []byte(script), 0o644)
}

// refreshMacroFile rewrites cmd's macro file after a run that may have changed
// the commands, when cmd has been set up on this machine.
func refreshMacroFile(cmd *cobra.Command) {
	file, err := macroFilePath()
	if err != nil || !fileExists(file) {
		return
	}

	_, _ = writeMacroFile(cmd)
}

// installedShells are the shells on this machine crower can set up.
func installedShells() []string {
	var candidates []string

	if runtime.GOOS == "windows" {
		candidates = []string{shellsetup.Cmd, shellsetup.PowerShell, shellsetup.Pwsh}
	} else {
		candidates = []string{shellsetup.Bash, shellsetup.Zsh, shellsetup.Fish}
	}

	var found []string

	for _, shell := range candidates {
		if shell == shellsetup.Cmd {
			found = append(found, shell)
			continue
		}

		if _, err := osexec.LookPath(shell); err == nil {
			found = append(found, shell)
		}
	}

	return found
}

// reloadHint is how to load the hook in the terminal that is already open.
func reloadHint(shell, where string) string {
	switch shell {
	case shellsetup.Bash, shellsetup.Zsh:
		return "open a new terminal, or run: source " + where
	case shellsetup.Fish:
		return "open a new terminal, or run: source " + where
	case shellsetup.Pwsh, shellsetup.PowerShell:
		return "open a new terminal, or run: . $PROFILE"
	case shellsetup.Cmd:
		file, _ := macroFilePath()
		return `open a new cmd window, or run: doskey /macrofile="` + file + `"`
	default:
		return "open a new terminal"
	}
}

// displayName is how a shell is named to the user.
func displayName(shell string) string {
	switch shell {
	case shellsetup.Pwsh:
		return "PowerShell 7"
	case shellsetup.PowerShell:
		return "Windows PowerShell"
	default:
		return shell
	}
}

// interactive reports whether a person can see and answer a question.
func interactive() bool {
	in, ok := streams.In.(*os.File)
	return ok && terminal.IsTerminal(in) && stderrIsTerminal()
}

// offerSetup asks, once per shell, to set up the shell crower was started
// from, and does it on yes. It runs before the command and never fails it.
func offerSetup(cmd *cobra.Command, args []string) {
	situation := shellsetup.Situation{
		Args:        args,
		Interactive: interactive(),
		OptOut:      os.Getenv(shellsetup.OptOutEnv) != "",
	}

	// The cheap conditions first: the parent process is looked up only when
	// a person is there to ask.
	if !shellsetup.ShouldOffer(withShell(situation, "?")) {
		return
	}

	situation.Shell = shellsetup.Parent()
	situation.HookLoaded = shellsetup.HookLoaded(situation.Shell, os.Getenv)

	if !shellsetup.ShouldOffer(situation) {
		return
	}

	answers, err := answersPath()
	if err != nil {
		return
	}

	situation.Declined = shellsetup.Declined(answers, situation.Shell)
	if !situation.Declined {
		situation.Installed = hookInstalled(situation.Shell)
	}

	if !shellsetup.ShouldOffer(situation) {
		return
	}

	shell := situation.Shell
	name := displayName(shell)

	fmt.Fprintf(streams.Err, "crower: set up %s so commands like \"cd\" work in this terminal? It adds one line to %s's startup. [Y/n] ", name, name)

	if !answerYes(true) {
		_ = shellsetup.Decline(answers, shell)
		fmt.Fprintf(streams.Err, "crower: OK, not asking again for %s. Run \"crower init --apply\" any time.\n", name)

		return
	}

	result, err := installHook(cmd, shell)
	if err != nil {
		fmt.Fprintf(streams.Err, "crower: could not set up %s: %v\n", name, err)
		return
	}

	fmt.Fprintf(streams.Err, "crower: %s now loads the hook (%s). To use it here, %s.\n", name, result.Where, reloadHint(shell, result.Where))

	if result.Warning != "" {
		offerPolicyFix(shell, result.Warning)
	}
}

// offerPolicyFix asks to let PowerShell run the profile crower just wrote.
func offerPolicyFix(shell, warning string) {
	fmt.Fprintf(streams.Err, "crower: %s. Allow scripts written on this computer for your user (RemoteSigned)? [Y/n] ", warning)

	if !answerYes(true) {
		fmt.Fprintln(streams.Err, "crower: left as is. To allow it later: Set-ExecutionPolicy -Scope CurrentUser RemoteSigned")
		return
	}

	if err := allowProfileScripts(shell); err != nil {
		fmt.Fprintln(streams.Err, "crower: could not change it:", err)
		return
	}

	fmt.Fprintln(streams.Err, "crower: done.")
}

// withShell is s with a placeholder shell, for the checks that do not need the
// real one.
func withShell(s shellsetup.Situation, shell string) shellsetup.Situation {
	s.Shell = shell
	return s
}

// answerYes reads one line; empty is the default.
func answerYes(def bool) bool {
	line, err := bufio.NewReader(streams.In).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(streams.Err)
		return false
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return def
	case "y", "yes", "s", "sim":
		return true
	default:
		return false
	}
}

// hintNoHook explains, on stderr, why a command that has to act on the
// terminal cannot when crower runs without the hook.
func hintNoHook(commands catalog.Catalog, key string) {
	command, found := commands.Resolve(key)
	if !found || !stderrIsTerminal() {
		return
	}

	_, isDir := shellhookChangeDir(command)
	if !isDir && !command.Here {
		return
	}

	shell := shellsetup.Parent()
	if shell == "" || shellsetup.HookLoaded(shell, os.Getenv) {
		return
	}

	what := "changes folder"
	if !isDir {
		what = "runs in your shell (--here)"
	}

	name := displayName(shell)

	if hookInstalled(shell) {
		where := ""
		if file, _, err := hookFile(shell); err == nil {
			where = file
		}

		fmt.Fprintf(streams.Err, "crower: %q %s, which only the shell hook can do, and this %s window started before it was set up: %s.\n", command.Name, what, name, reloadHint(shell, where))

		return
	}

	fmt.Fprintf(streams.Err, "crower: %q %s, which only the shell hook can do. Run \"crower init --apply\", then open a new terminal.\n", command.Name, what)
}

// setUpAll sets up every shell installed on this machine, for the installers
// and `crower init --all`.
func setUpAll(cmd *cobra.Command) error {
	var failed []string

	for _, shell := range installedShells() {
		result, err := installHook(cmd, shell)
		if err != nil {
			fmt.Fprintf(streams.Err, "%-18s not set up: %v\n", displayName(shell), err)
			failed = append(failed, shell)

			continue
		}

		state := "already set up"
		if result.Changed {
			state = "set up"
		}

		fmt.Fprintf(streams.Out, "%-18s %s (%s)\n", displayName(shell), state, result.Where)

		if result.Warning != "" {
			fmt.Fprintf(streams.Out, "%-18s %s; to allow it: Set-ExecutionPolicy -Scope CurrentUser RemoteSigned\n", "", result.Warning)
		}
	}

	fmt.Fprintln(streams.Out, "Open a new terminal to use it.")

	if len(failed) > 0 && slices.Equal(failed, installedShells()) {
		return fmt.Errorf("no shell could be set up")
	}

	return nil
}

// shellhookChangeDir is the folder a command only changes into, as the hook
// sees it: a Steps command never is one.
func shellhookChangeDir(command catalog.Command) (string, bool) {
	if len(command.Steps) > 0 {
		return "", false
	}

	return shellhook.ChangeDir(command.Exec)
}

// mutatingRun reports whether the command line may have changed the commands,
// so cmd's macro file must be rewritten.
func mutatingRun(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// A root flag's value is not the command: --config x delete a.
		if arg == "--"+flagConfig || arg == "--"+flagSetup {
			i++
			continue
		}

		if strings.HasPrefix(arg, "-") {
			continue
		}

		return slices.Contains([]string{"create", "update", "delete", "import", "revert", "restore", "reset", "setup"}, arg)
	}

	return false
}
