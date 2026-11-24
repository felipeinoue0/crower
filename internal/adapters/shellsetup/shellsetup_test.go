package shellsetup

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestByName(t *testing.T) {
	tests := map[string]string{
		"zsh": Zsh, "-zsh": Zsh, "/usr/bin/bash": Bash, "fish": Fish,
		"pwsh.exe": Pwsh, "PowerShell.exe": PowerShell, "cmd.exe": Cmd, `C:\Windows\System32\cmd.exe`: Cmd,
		"sh": "", "sudo": "", "code": "", "": "",
	}

	for name, want := range tests {
		if got := ByName(name); got != want {
			t.Errorf("ByName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestHookLoaded(t *testing.T) {
	env := func(value string) func(string) string {
		return func(key string) string {
			if key == HookEnv {
				return value
			}

			return ""
		}
	}

	tests := []struct {
		shell, env string
		want       bool
	}{
		{Zsh, "zsh", true},
		{Bash, "zsh", false}, // a bash started from zsh inherits the variable
		{Pwsh, "powershell", true},
		{PowerShell, "powershell", true},
		{Cmd, "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		if got := HookLoaded(tt.shell, env(tt.env)); got != tt.want {
			t.Errorf("HookLoaded(%q, %q) = %v, want %v", tt.shell, tt.env, got, tt.want)
		}
	}
}

func TestShouldOffer(t *testing.T) {
	base := Situation{Args: []string{"a"}, Interactive: true, Shell: Bash}

	tests := []struct {
		name string
		edit func(*Situation)
		want bool
	}{
		{"a person in a shell without the hook", func(*Situation) {}, true},
		{"no arguments", func(s *Situation) { s.Args = nil }, true},
		{"flags before the command", func(s *Situation) { s.Args = []string{"--config", "x", "list"} }, true},
		{"not interactive", func(s *Situation) { s.Interactive = false }, false},
		{"unknown shell", func(s *Situation) { s.Shell = "" }, false},
		{"hook loaded", func(s *Situation) { s.HookLoaded = true }, false},
		{"already installed", func(s *Situation) { s.Installed = true }, false},
		{"declined", func(s *Situation) { s.Declined = true }, false},
		{"opted out", func(s *Situation) { s.OptOut = true }, false},
		{"the hook calling init", func(s *Situation) { s.Args = []string{"init", "bash"} }, false},
		{"init after a config flag", func(s *Situation) { s.Args = []string{"--config", "x", "init", "bash"} }, false},
		{"tab completion", func(s *Situation) { s.Args = []string{"__complete", "ex"} }, false},
		{"emit", func(s *Situation) { s.Args = []string{"execute", "a", "--emit"} }, false},
		{"no input", func(s *Situation) { s.Args = []string{"a", "--no-input"} }, false},
		{"version", func(s *Situation) { s.Args = []string{"--version"} }, false},
		{"json", func(s *Situation) { s.Args = []string{"list", "--json"} }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := base
			tt.edit(&s)

			if got := ShouldOffer(s); got != tt.want {
				t.Errorf("ShouldOffer(%+v) = %v, want %v", s, got, tt.want)
			}
		})
	}
}

func TestAnswersRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "shell-setup.json")

	if Declined(path, Bash) || Installed(path, Pwsh) {
		t.Fatal("a missing file has answers")
	}

	for _, err := range []error{Decline(path, Bash), Decline(path, Bash), Remember(path, Pwsh), Decline(path, Cmd)} {
		if err != nil {
			t.Fatal(err)
		}
	}

	if !Declined(path, Bash) || !Declined(path, Cmd) || Declined(path, Zsh) {
		t.Error("Declined does not reflect what was declined")
	}

	if !Installed(path, Pwsh) || Installed(path, Bash) {
		t.Error("Installed does not reflect what was remembered")
	}
}

func TestWithAutoRun(t *testing.T) {
	file := `C:\Users\Ann\crower\crower.doskey`
	entry := AutoRunEntry(file)

	tests := []struct {
		existing, want string
		changed        bool
	}{
		{"", entry, true},
		{"  ", entry, true},
		{`@chcp 65001>nul`, `@chcp 65001>nul& ` + entry, true},
		{`set X=1 `, `set X=1& ` + entry, true},
		{`@chcp 65001>nul& ` + entry, `@chcp 65001>nul& ` + entry, false},
		{strings.ToUpper(entry), strings.ToUpper(entry), false},
	}

	for _, tt := range tests {
		got, changed := WithAutoRun(tt.existing, file)
		if got != tt.want || changed != tt.changed {
			t.Errorf("WithAutoRun(%q) = %q, %v; want %q, %v", tt.existing, got, changed, tt.want, tt.changed)
		}
	}

	if entry != `doskey /macrofile="C:\Users\Ann\crower\crower.doskey" & set "CROWER_HOOK=cmd"` {
		t.Errorf("AutoRunEntry = %q", entry)
	}
}
