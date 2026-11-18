package shellalias

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func names(aliases []Alias) map[string]string {
	out := map[string]string{}
	for _, a := range aliases {
		out[a.Name] = a.Value
	}

	return out
}

func TestParseStartupSyntax(t *testing.T) {
	input := `
# a comment: alias commented='no'
export PATH="$HOME/bin:$PATH"
FOO=bar
alias ll='ls -la'
alias gs="git status"   # trailing comment
alias -- gd='git diff'
alias up=..; echo ignored
  alias indented='echo in a block'
alias quote='echo it'\''s here'
alias pipe='ps aux | grep -i "$1"'
alias a1='echo one' a2=two
alias ll='ls -lah'
alias -s pdf=open
alias 'bad name'=x
alias empty=''
alias fishy 'echo fish style'
alias grep='grep --color=auto'
Set-Alias np notepad.exe
Set-Alias -Name g -Value git -Scope Global
New-Alias -Name:k -Value:kubectl
function ll2 --wraps='ls -l' --description 'alias ll2=ls -l'
function regular() { echo no; }
`

	got, err := Parse(strings.NewReader(input), "rc", false)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"ll":       "ls -lah", // the later definition wins
		"gs":       "git status",
		"gd":       "git diff",
		"up":       "..",
		"indented": "echo in a block",
		"quote":    "echo it's here",
		"pipe":     `ps aux | grep -i "$1"`,
		"a1":       "echo one",
		"a2":       "two",
		"fishy":    "echo fish style",
		"grep":     "grep --color=auto",
		"np":       "notepad.exe",
		"g":        "git",
		"k":        "kubectl",
		"ll2":      "ls -l",
	}

	gotMap := names(got)

	for name, value := range want {
		if gotMap[name] != value {
			t.Errorf("%s = %q, want %q", name, gotMap[name], value)
		}
	}

	for name := range gotMap {
		if _, ok := want[name]; !ok {
			t.Errorf("unexpected alias %q = %q", name, gotMap[name])
		}
	}

	// The order follows the file; a redefinition keeps its first position.
	if got[0].Name != "ll" || got[0].Source != "rc" {
		t.Errorf("first = %+v, want ll from rc", got[0])
	}
}

// zsh prints its aliases as bare name=value lines. That form is accepted only
// from piped `alias` output: in a startup file it is a variable assignment.
func TestParseLenientAcceptsZshAliasOutput(t *testing.T) {
	input := "ll='ls -la'\nrun-help=man\nwhich-command=whence\n"

	lenient, err := Parse(strings.NewReader(input), "alias output", true)
	if err != nil {
		t.Fatal(err)
	}

	if got := names(lenient); got["ll"] != "ls -la" || got["run-help"] != "man" || len(got) != 3 {
		t.Errorf("lenient = %v", got)
	}

	strict, _ := Parse(strings.NewReader(input), "rc", false)
	if len(strict) != 0 {
		t.Errorf("a startup file's assignments were read as aliases: %v", strict)
	}
}

func TestReadStartupFilesReadsEveryShell(t *testing.T) {
	home := t.TempDir()

	write := func(rel, body string) {
		t.Helper()

		path := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write(".bashrc", "alias b='echo bash'\nalias shared='from bashrc'\n")
	write(".zshrc", "alias z='echo zsh'\nalias shared='from zshrc'\n")
	write(".config/fish/functions/f.fish", "function f --description 'alias f=echo fish'\n  echo fish $argv\nend\n")
	write("Documents/PowerShell/Microsoft.PowerShell_profile.ps1", "Set-Alias p Get-Process\n")

	got, read, err := ReadStartupFiles(home)
	if err != nil {
		t.Fatal(err)
	}

	if len(read) != 4 {
		t.Errorf("read %d files, want 4: %v", len(read), read)
	}

	m := names(got)
	for name, value := range map[string]string{"b": "echo bash", "z": "echo zsh", "f": "echo fish", "p": "Get-Process", "shared": "from zshrc"} {
		if m[name] != value {
			t.Errorf("%s = %q, want %q", name, m[name], value)
		}
	}
}

func TestReadStartupFilesWithNoFiles(t *testing.T) {
	got, read, err := ReadStartupFiles(t.TempDir())
	if err != nil || len(got) != 0 || len(read) != 0 {
		t.Errorf("empty home = %v, %v, %v", got, read, err)
	}
}
