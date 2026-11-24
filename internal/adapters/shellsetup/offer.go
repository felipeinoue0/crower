package shellsetup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
)

// OptOutEnv turns the offer off for good when set to anything.
const OptOutEnv = "CROWER_NO_SETUP"

// Situation is everything the offer depends on, gathered by the caller so the
// decision itself is a plain function.
type Situation struct {
	// Args are the command-line arguments after the program name.
	Args []string
	// Interactive is true when both stdin and stderr are terminals: someone
	// can see the question and answer it.
	Interactive bool
	// Shell is the parent shell (Parent), "" when unknown.
	Shell string
	// HookLoaded is true when the hook is active in this shell already.
	HookLoaded bool
	// Installed is true when the shell's startup file (or cmd's AutoRun)
	// already loads the hook; the open window just predates it.
	Installed bool
	// Declined is true when the user said no for this shell before.
	Declined bool
	// OptOut is true when CROWER_NO_SETUP is set.
	OptOut bool
}

// skipCommands never ask: the hook itself calls init, and completion runs on
// every Tab.
var skipCommands = []string{"init", "completion", "__complete", "__completeNoDesc", "help"}

// skipFlags never ask: a script said it cannot answer, the hook is asking for
// a line to evaluate, or the user only wants help or the version.
var skipFlags = []string{"--no-input", "--emit", "--version", "--help", "-h", "--json"}

// ShouldOffer reports whether to ask the user to set up their shell.
//
// It asks only a person (both streams are terminals), in a shell crower knows,
// that has neither the hook loaded nor installed, and that was not declined.
// It never asks while the hook itself, or a script, is running crower.
func ShouldOffer(s Situation) bool {
	if !s.Interactive || s.OptOut || s.Shell == "" || s.HookLoaded || s.Installed || s.Declined {
		return false
	}

	for _, arg := range s.Args {
		if slices.Contains(skipFlags, arg) {
			return false
		}
	}

	for i := 0; i < len(s.Args); i++ {
		arg := s.Args[i]

		// A root flag's value is not the command: --config x init.
		if arg == "--config" || arg == "--setup" {
			i++
			continue
		}

		if len(arg) > 0 && arg[0] == '-' {
			continue
		}

		return !slices.Contains(skipCommands, arg)
	}

	return true
}

// answers is the file that remembers which shells the user said no for, and
// which were set up where checking would cost a process (PowerShell).
type answers struct {
	Declined  []string `json:"declined,omitempty"`
	Installed []string `json:"installed,omitempty"`
}

// Declined reports whether the user said no to setting up shell.
func Declined(path, shell string) bool {
	return slices.Contains(read(path).Declined, shell)
}

// Decline remembers that the user said no for shell, so crower never asks
// again there.
func Decline(path, shell string) error {
	a := read(path)
	if slices.Contains(a.Declined, shell) {
		return nil
	}

	a.Declined = append(a.Declined, shell)
	slices.Sort(a.Declined)

	return write(path, a)
}

// Installed reports whether shell was set up before (see Remember).
func Installed(path, shell string) bool {
	return slices.Contains(read(path).Installed, shell)
}

// Remember records that shell was set up.
func Remember(path, shell string) error {
	a := read(path)
	if slices.Contains(a.Installed, shell) {
		return nil
	}

	a.Installed = append(a.Installed, shell)
	slices.Sort(a.Installed)

	return write(path, a)
}

func write(path string, a answers) error {
	body, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, append(body, '\n'), 0o644)
}

// read is the stored answers; a missing or damaged file is "no answers", which
// at worst asks once more.
func read(path string) answers {
	var a answers

	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || err != nil {
		return a
	}

	_ = json.Unmarshal(body, &a)

	return a
}
