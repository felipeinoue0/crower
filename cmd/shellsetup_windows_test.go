//go:build windows

package cmd

import (
	"bytes"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeinoue0/crower/internal/adapters/shellsetup"
	"github.com/felipeinoue0/crower/internal/session"
)

// Windows PowerShell answers where its profile is; the path is the one it
// loads, wherever Documents lives.
func TestPowerShellProfileQuery(t *testing.T) {
	profile, err := powershellProfile(shellsetup.PowerShell)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.EqualFold(filepath.Base(profile), "Microsoft.PowerShell_profile.ps1") || !strings.Contains(strings.ToLower(profile), "windowspowershell") {
		t.Errorf("profile = %q, want Windows PowerShell's", profile)
	}

	if policy := powershellPolicy(shellsetup.PowerShell); policy == "" {
		t.Error("no execution policy reported")
	}
}

// Setting up cmd writes the macro file and adds it to AutoRun, keeping what
// was there; a new cmd then runs it. The user's AutoRun is restored after.
func TestInstallHookCmd(t *testing.T) {
	saved, err := readAutoRun()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := writeAutoRun(saved); err != nil {
			t.Errorf("restore AutoRun: %v", err)
		}
	})

	if err := writeAutoRun("set CROWER_TEST_MINE=1"); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	// Run outside any command, as the first-run offer does: the commands
	// must load (cobra has not merged --config into the root yet).
	errOut := &bytes.Buffer{}
	previous := streams
	streams = session.Streams{In: previous.In, Out: previous.Out, Err: errOut}
	t.Cleanup(func() { streams = previous })

	result, err := installHook(rootCmd, shellsetup.Cmd)
	if err != nil {
		t.Fatalf("installHook: %v", err)
	}

	if strings.Contains(errOut.String(), "no command aliases loaded") {
		t.Errorf("the macro file was built without the commands:\n%s", errOut.String())
	}

	file := filepath.Join(home, "crower", "crower.doskey")

	body, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("macro file: %v", err)
	}

	if !strings.Contains(string(body), "cr=") || !strings.Contains(string(body), `doskey /macrofile="`+file+`"`) {
		t.Errorf("macro file lacks the reloading cr macro:\n%s", body)
	}

	value, err := readAutoRun()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(value, "set CROWER_TEST_MINE=1& ") || !shellsetup.HasAutoRun(value, file) || !result.Changed {
		t.Errorf("AutoRun = %q (changed %v), want the old value kept and the macro file added", value, result.Changed)
	}

	if !hookInstalled(shellsetup.Cmd) {
		t.Error("hookInstalled(cmd) = false after setup")
	}

	again, err := installHook(rootCmd, shellsetup.Cmd)
	if err != nil || again.Changed {
		t.Errorf("second setup changed = %v, err %v; want no change", again.Changed, err)
	}

	// A new cmd runs AutoRun: both the user's command and crower's mark run.
	out, err := osexec.Command("cmd", "/c", "set CROWER_").CombinedOutput()
	if err != nil {
		t.Fatalf("cmd: %v\n%s", err, out)
	}

	got := strings.ReplaceAll(string(out), "\r\n", "\n")
	for _, want := range []string{"CROWER_HOOK=cmd\n", "CROWER_TEST_MINE=1\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("a new cmd lacks %q:\n%s", want, got)
		}
	}
}
