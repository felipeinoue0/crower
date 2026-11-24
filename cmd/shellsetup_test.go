package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipeinoue0/crower/internal/adapters/shellsetup"
)

// Setting up bash, zsh or fish adds the hook line to the startup file once,
// and hookInstalled sees it.
func TestInstallHookPosix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX startup files")
	}

	for _, shell := range []string{shellsetup.Bash, shellsetup.Zsh, shellsetup.Fish} {
		t.Run(shell, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			if hookInstalled(shell) {
				t.Fatal("installed before setup")
			}

			first, err := installHook(rootCmd, shell)
			if err != nil {
				t.Fatalf("installHook: %v", err)
			}

			second, err := installHook(rootCmd, shell)
			if err != nil {
				t.Fatalf("installHook again: %v", err)
			}

			if !first.Changed || second.Changed {
				t.Errorf("changed = %v then %v, want true then false", first.Changed, second.Changed)
			}

			if !strings.HasPrefix(first.Where, home) {
				t.Errorf("wrote %q, want a file under %q", first.Where, home)
			}

			body, err := os.ReadFile(first.Where)
			if err != nil {
				t.Fatal(err)
			}

			if strings.Count(string(body), "init "+shell) != 1 {
				t.Errorf("%s holds the hook line %d times:\n%s", first.Where, strings.Count(string(body), "init "+shell), body)
			}

			if !hookInstalled(shell) {
				t.Error("hookInstalled = false after setup")
			}
		})
	}
}

func TestPolicyBlocksProfile(t *testing.T) {
	for policy, want := range map[string]bool{"Restricted": true, "AllSigned": true, "allsigned": true, "RemoteSigned": false, "Unrestricted": false, "Bypass": false, "": false} {
		if got := policyBlocksProfile(policy); got != want {
			t.Errorf("policyBlocksProfile(%q) = %v, want %v", policy, got, want)
		}
	}
}

func TestMutatingRun(t *testing.T) {
	tests := map[string]bool{
		"create a ls": true, "--config x delete a": true, "setup use work": true,
		"list": false, "a": false, "execute create": false, "": false, "--version": false,
	}

	for line, want := range tests {
		if got := mutatingRun(strings.Fields(line)); got != want {
			t.Errorf("mutatingRun(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestMacroFilePathIsUnderCrowerHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	file, err := macroFilePath()
	if err != nil {
		t.Fatal(err)
	}

	if want := filepath.Join(home, "crower", "crower.doskey"); file != want {
		t.Errorf("macroFilePath = %q, want %q", file, want)
	}
}

// cmd's macro file holds the user's commands, read from the default
// configuration. (runWith has already executed the tree, so this cannot catch
// unmerged flags; TestInstallHookCmd on Windows runs from a fresh root.)
func TestWriteMacroFileHoldsTheCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CROWER_SETUP", "")

	cfg := filepath.Join(home, "crower", "crower.yaml")
	if _, stderr, err := runWith(t, cfg, "", "create", "desk", `cd C:\Users\Ann\Desktop`); err != nil {
		t.Fatalf("create: %v\n%s", err, stderr)
	}

	resetFlags()

	file, err := writeMacroFile(rootCmd)
	if err != nil {
		t.Fatalf("writeMacroFile: %v", err)
	}

	body, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`desk=call `,
		`cd /d "C:\Users\Ann\Desktop"`,
		`doskey /macrofile="` + file + `"`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("macro file lacks %q:\n%s", want, body)
		}
	}
}
