package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeinoue0/crower/internal/domain/catalog"
)

func TestShellName(t *testing.T) {
	tests := map[string]string{
		"/bin/bash":           "bash",
		"/usr/bin/zsh":        "zsh",
		"/usr/local/bin/fish": "fish",
		"/opt/pwsh":           "powershell",
		"C:\\Program Files\\PowerShell\\pwsh.exe": "powershell",
		"/bin/sh": "",
		"":        "",
	}

	for shell, want := range tests {
		if got := shellName(shell); got != want {
			t.Errorf("shellName(%q) = %q, want %q", shell, got, want)
		}
	}
}

func TestInitTarget(t *testing.T) {
	home := "/home/u"

	tests := []struct {
		shell string
		file  string
		line  string
	}{
		{"bash", filepath.Join(home, ".bashrc"), `export PATH="$HOME/.local/bin:$PATH"`},
		{"zsh", filepath.Join(home, ".zshrc"), `export PATH="$HOME/.local/bin:$PATH"`},
		{"fish", filepath.Join(home, ".config", "fish", "config.fish"), `fish_add_path "$HOME/.local/bin"`},
		{"powershell", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			file, line := initTarget(tt.shell, home)
			if file != tt.file || line != tt.line {
				t.Errorf("initTarget(%q) = %q, %q; want %q, %q", tt.shell, file, line, tt.file, tt.line)
			}
		})
	}
}

func TestApplyInitIsIdempotent(t *testing.T) {
	home := t.TempDir()
	rc := filepath.Join(home, ".bashrc")
	line := `export PATH="$HOME/.local/bin:$PATH"`

	changed, err := applyInit(rc, line)
	if err != nil || !changed {
		t.Fatalf("first apply = %v, %v; want changed", changed, err)
	}

	changed, err = applyInit(rc, line)
	if err != nil || changed {
		t.Fatalf("second apply = %v, %v; want unchanged", changed, err)
	}

	body, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if strings.Count(string(body), line) != 1 {
		t.Errorf("rc = %q, want the line exactly once", body)
	}
}

// A missing startup file is created, including its folder.
func TestApplyInitCreatesTheFile(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".config", "fish", "config.fish")

	changed, err := applyInit(rc, "fish_add_path")
	if err != nil || !changed {
		t.Fatalf("apply = %v, %v", changed, err)
	}

	if _, err := os.Stat(rc); err != nil {
		t.Errorf("startup file: %v", err)
	}
}

func TestHookTarget(t *testing.T) {
	home := "/home/u"

	tests := []struct {
		shell, goos, file, contains string
	}{
		{"bash", "linux", filepath.Join(home, ".bashrc"), `eval "$(crower init bash)"`},
		{"zsh", "darwin", filepath.Join(home, ".zshrc"), `eval "$(crower init zsh)"`},
		{"fish", "linux", filepath.Join(home, ".config", "fish", "config.fish"), "crower init fish | source"},
		{"powershell", "windows", filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"), "Invoke-Expression"},
		{"powershell", "linux", filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"), "Invoke-Expression"},
		{"", "linux", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.shell+"/"+tt.goos, func(t *testing.T) {
			file, line := hookTarget(tt.shell, home, tt.goos)
			if file != tt.file || !strings.Contains(line, tt.contains) {
				t.Errorf("hookTarget(%q, %q) = %q, %q; want %q containing %q", tt.shell, tt.goos, file, line, tt.file, tt.contains)
			}
		})
	}
}

// The alias block is built from the saved command set, names and aliases both.
func TestInitAliasesListsEveryNameAndAlias(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "crower.yaml")

	if _, _, err := runWith(t, cfg, "", "create", "greet", "echo hi", "--alias", "hi,hey"); err != nil {
		t.Fatalf("create: %v", err)
	}

	out, _, err := runWith(t, cfg, "", "init", "zsh", "--aliases")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	for _, want := range []string{"__crower_def greet\n", "__crower_def hi\n", "__crower_def hey\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("aliases lack %q:\n%s", want, out)
		}
	}

	full, _, err := runWith(t, cfg, "", "init", "bash")
	if err != nil {
		t.Fatalf("init bash: %v", err)
	}

	if !strings.Contains(full, "__start_crower") || !strings.Contains(full, "__crower_def greet") {
		t.Errorf("hook lacks completion or aliases:\n%.400s", full)
	}
}

func TestInitRejectsAnUnknownShell(t *testing.T) {
	if _, _, err := run(t, "init", "nushell"); err == nil {
		t.Error("init nushell succeeded, want an error")
	}
}

// `cr gr<Tab>` and `crower delete <Tab>` offer the saved names and aliases.
func TestCommandNamesComplete(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "crower.yaml")

	if _, _, err := runWith(t, cfg, "", "create", "greet", "echo hi", "--alias", "hi", "--description", "say hello"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, _, err := runWith(t, cfg, "", "create", "deploy", "echo go"); err != nil {
		t.Fatalf("create: %v", err)
	}

	out, err := complete(t, cfg, "delete", "")
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	for _, want := range []string{"deploy\techo go", "greet\tsay hello", "hi\tsay hello"} {
		if !strings.Contains(out, want) {
			t.Errorf("completion lacks %q:\n%s", want, out)
		}
	}

	out, err = complete(t, cfg, "gr")
	if err != nil {
		t.Fatalf("complete root: %v", err)
	}

	if !strings.Contains(out, "greet") || strings.Contains(out, "deploy") {
		t.Errorf("root completion for %q = %s", "gr", out)
	}
}

// complete runs cobra's hidden completion command, which writes to cobra's own
// output rather than crower's streams.
func complete(t *testing.T, cfg string, args ...string) (string, error) {
	t.Helper()

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	_, _, err := runWith(t, cfg, "", append([]string{"__complete"}, args...)...)

	return buf.String(), err
}

func TestChangeDirs(t *testing.T) {
	commands := catalog.New(nil, []catalog.Command{
		{Name: "desk", Aliases: []string{"d"}, Exec: "cd /mnt/c/Users/me/Desktop"},
		{Name: "deploy", Exec: "cd /app && git pull"},
		{Name: "steps", Exec: "cd /tmp", Steps: []string{"cd /tmp", "ls"}},
	})

	got := changeDirs(commands)
	want := map[string]string{"desk": "/mnt/c/Users/me/Desktop", "d": "/mnt/c/Users/me/Desktop"}

	if len(got) != len(want) {
		t.Fatalf("changeDirs = %v, want %v", got, want)
	}

	for name, dir := range want {
		if got[name] != dir {
			t.Errorf("changeDirs[%q] = %q, want %q", name, got[name], dir)
		}
	}
}

func TestHereWords(t *testing.T) {
	commands := catalog.New(nil, []catalog.Command{
		{Name: "act", Aliases: []string{"a"}, Exec: "source .venv/bin/activate", Here: true},
		{Name: "plain", Exec: "ls"},
		{Name: "broken", Exec: "ls", Cwd: "/tmp", Here: true},
	})

	got := hereWords(commands)
	if len(got) != 2 || !got["act"] || !got["a"] {
		t.Errorf("hereWords = %v, want act and its alias only", got)
	}
}
