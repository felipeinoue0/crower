package shellhook

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestUsable(t *testing.T) {
	got := Usable([]string{"greet", "crower", "cr", "has space", "a;b", "$(x)", "greet", "deploy-prod", "k8s.get", "", "-flag"})
	want := []string{"deploy-prod", "greet", "k8s.get"}

	if !slices.Equal(got, want) {
		t.Errorf("Usable = %q, want %q", got, want)
	}
}

func TestScriptPerShell(t *testing.T) {
	names := []string{"deploy", "hi"}
	opts := Options{Bin: "crower", Exe: `C:\Program Files\o'brien\crower.exe`}

	tests := []struct {
		shell string
		want  []string
	}{
		{"bash", []string{"__crower_def deploy\n", "__crower_def hi\n", `alias "$1=command crower execute $1 --"`, "for __crower_n in ${__crower_names-};"}},
		{"zsh", []string{"__crower_def deploy\n", "for __crower_n in ${=__crower_names-};"}},
		{"fish", []string{"if not type -q deploy\n", "command crower execute deploy -- $argv", "set -ga __crower_names deploy"}},
		{"powershell", []string{"function global:deploy { & $global:__crower_exe execute 'deploy' '--' @args }", `'C:\Program Files\o''brien\crower.exe'`}},
		{"cmd", []string{"deploy=crower execute deploy -- $*\n", "hi=crower execute hi -- $*\n"}},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			script, err := Script(tt.shell, names, opts)
			if err != nil {
				t.Fatalf("Script: %v", err)
			}

			for _, want := range tt.want {
				if !strings.Contains(script, want) {
					t.Errorf("script lacks %q:\n%s", want, script)
				}
			}
		})
	}
}

// Every mutating subcommand re-syncs; a run of one that cannot change the set
// (list) does not appear in the trigger list.
func TestWrapperResyncsOnlyAfterAChange(t *testing.T) {
	script, err := Script("bash", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(script, "create|update|delete|import|revert|restore|reset|setup) __crower_sync") {
		t.Errorf("trigger list missing:\n%s", script)
	}

	if strings.Contains(script, "list") {
		t.Errorf("list must not trigger a re-sync:\n%s", script)
	}
}

func TestUnsupportedShell(t *testing.T) {
	if _, err := Script("nushell", nil, Options{}); err == nil {
		t.Error("Script(nushell) succeeded, want an error")
	}

	if _, err := Aliases("nushell", []string{"x"}, Options{}); err == nil {
		t.Error("Aliases(nushell) succeeded, want an error")
	}
}

// The script must parse in the real shell. A shell that is not installed is
// reported as skipped, which is not a pass.
func TestPosixScriptParses(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}

			script, err := Script(shell, []string{"deploy", "hi"}, Options{Bin: "crower", Completion: "true"})
			if err != nil {
				t.Fatal(err)
			}

			file := filepath.Join(t.TempDir(), "hook")
			if err := os.WriteFile(file, []byte(script), 0o600); err != nil {
				t.Fatal(err)
			}

			if out, err := exec.Command(path, "-n", file).CombinedOutput(); err != nil {
				t.Errorf("%s -n: %v\n%s", shell, err, out)
			}
		})
	}
}

// End to end in bash: the alias forwards its arguments, and a name the shell
// already knows is left alone.
func TestBashAliasesForwardAndNeverShadow(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}

	dir := t.TempDir()
	fake := filepath.Join(dir, "crower")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho \"ran: $*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	script, err := Script("bash", []string{"deploy", "echo"}, Options{Bin: "crower"})
	if err != nil {
		t.Fatal(err)
	}

	body := "shopt -s expand_aliases\n" + script + "\ndeploy -x 'two words'\necho plain\n"

	cmd := exec.Command(bash, "--norc", "-c", body)
	cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "ran: execute deploy -- -x two words") {
		t.Errorf("deploy did not forward its arguments:\n%s", got)
	}

	if !strings.Contains(got, "plain") || strings.Contains(got, "ran: execute echo") {
		t.Errorf("the builtin echo was shadowed:\n%s", got)
	}
}
