package shellhook

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestChangeDir(t *testing.T) {
	tests := []struct {
		line string
		want string
		ok   bool
	}{
		{"cd /mnt/c/Users/Felipe/Desktop", "/mnt/c/Users/Felipe/Desktop", true},
		{"  cd   /srv/app  ", "/srv/app", true},
		{"cd '/mnt/c/My Folder'", "/mnt/c/My Folder", true},
		{`cd "C:\Users\Felipe\Desktop"`, `C:\Users\Felipe\Desktop`, true},
		{`cd /d D:\work`, `D:\work`, true},
		{`CD /D "D:\my work"`, `D:\my work`, true},
		{`chdir C:/tools`, `C:/tools`, true},
		{`cd \\server\share`, `\\server\share`, true},
		{"cd ~", "~", true},
		{"cd ~/projects", "~/projects", true},
		{"cd", "", false},
		{"cd ..", "", false},
		{"cd src", "", false},
		{"cd ~bob", "", false},
		{"cd /a b", "", false},
		{"cd /app && git pull", "", false},
		{"cd /app; ls", "", false},
		{"cd $HOME/x", "", false},
		{"cd %USERPROFILE%", "", false},
		{"cd {}", "", false},
		{"cd /tmp/*", "", false},
		{"cd '/it'\"s'", "", false},
		{"cdx /tmp", "", false},
		{"ls /tmp", "", false},
		{"Set-Location C:\\x", "", false},
	}

	for _, tt := range tests {
		got, ok := ChangeDir(tt.line)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ChangeDir(%q) = %q, %v; want %q, %v", tt.line, got, ok, tt.want, tt.ok)
		}
	}
}

func TestChangeDirPerShell(t *testing.T) {
	opts := Options{Bin: "cr", Dirs: map[string]string{
		"desk": "/mnt/c/Users/Felipe/Desktop",
		"win":  `C:\Users\o'brien\Desk top`,
		"home": "~/projects",
	}}
	names := []string{"desk", "win", "home", "deploy"}

	tests := []struct {
		shell string
		want  []string
	}{
		{"bash", []string{
			`__crower_cdef desk ''\''/mnt/c/Users/Felipe/Desktop'\'''` + "\n",
			`__crower_cdef home '~/'\''projects'\'''` + "\n",
			"__crower_def deploy\n",
			`command cr execute "$1" -- && builtin cd -- "$2"`,
		}},
		{"fish", []string{
			"command cr execute desk --; and cd '/mnt/c/Users/Felipe/Desktop'",
			`command cr execute win --; and cd 'C:\\Users\\o\'brien\\Desk top'`,
			"command cr execute home --; and cd ~/'projects'",
			"command cr execute deploy -- $argv",
		}},
		{"powershell", []string{
			"if ($LASTEXITCODE -eq 0) { Set-Location -LiteralPath '/mnt/c/Users/Felipe/Desktop' }",
			`Set-Location -LiteralPath 'C:\Users\o''brien\Desk top'`,
			"Set-Location -LiteralPath (Join-Path $HOME 'projects')",
		}},
		{"cmd", []string{
			`desk=call cr execute desk -- && cd /d "/mnt/c/Users/Felipe/Desktop"` + "\n",
			`win=call cr execute win -- && cd /d "C:\Users\o'brien\Desk top"` + "\n",
			`home=call cr execute home -- && cd /d "%USERPROFILE%\projects"` + "\n",
			"deploy=cr execute deploy -- $*\n",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			script, err := Script(tt.shell, names, opts)
			if err != nil {
				t.Fatal(err)
			}

			for _, want := range tt.want {
				if !strings.Contains(script, want) {
					t.Errorf("script lacks %q:\n%s", want, script)
				}
			}
		})
	}
}

// End to end in bash and zsh: a cd command moves the calling shell, only after
// crower succeeds, and never shadows a name the shell already knows.
func TestPosixChangeDirMovesTheShell(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}

			bin := t.TempDir()
			target := filepath.Join(t.TempDir(), "it's a folder")
			if err := os.Mkdir(target, 0o755); err != nil {
				t.Fatal(err)
			}

			// The fake crower fails for "broken", as a missing folder would.
			fake := "#!/bin/sh\necho \"ran: $*\" >&2\n[ \"$2\" = broken ] && exit 1\nexit 0\n"
			if err := os.WriteFile(filepath.Join(bin, "crower"), []byte(fake), 0o755); err != nil {
				t.Fatal(err)
			}

			opts := Options{Bin: "crower", Dirs: map[string]string{"go2": target, "broken": target, "home2": "~"}}
			script, err := Script(shell, []string{"go2", "broken", "home2"}, opts)
			if err != nil {
				t.Fatal(err)
			}

			start := t.TempDir()
			body := script + "\ncd " + posixQuote(start) + "\nbroken\necho \"after-broken=$PWD\"\ngo2 extra args\necho \"after-go=$PWD\"\nhome2\necho \"after-home=$PWD\"\n"
			if shell == "bash" {
				body = "shopt -s expand_aliases\n" + body
			}

			file := filepath.Join(t.TempDir(), "run")
			if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}

			home := t.TempDir()
			args := []string{"--norc", file}
			if shell == "zsh" {
				args = []string{"-f", file}
			}

			cmd := exec.Command(path, args...)
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "HOME="+home)

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: %v\n%s", shell, err, out)
			}

			got := string(out)
			for _, want := range []string{
				"ran: execute go2 --\n",
				"after-broken=" + start + "\n",
				"after-go=" + target + "\n",
				"after-home=" + home + "\n",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("output lacks %q:\n%s", want, got)
				}
			}
		})
	}
}

// End to end in bash and zsh: "cr <name>", "crower <name>" and
// "crower execute <name>" move the shell for a cd command; a subcommand name
// still reaches the program, and a cd name the shell already owns (echo) still
// works through cr.
func TestPosixWrapperMovesTheShell(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}

			bin := t.TempDir()
			target := filepath.Join(t.TempDir(), "it's here")
			if err := os.Mkdir(target, 0o755); err != nil {
				t.Fatal(err)
			}

			fake := "#!/bin/sh\necho \"ran: $*\"\n"
			if err := os.WriteFile(filepath.Join(bin, "crower"), []byte(fake), 0o755); err != nil {
				t.Fatal(err)
			}

			dirs := map[string]string{"a": target, "list": target, "echo": target}
			opts := Options{Bin: "crower", Dirs: dirs, Reserved: []string{"list", "execute", "create"}}
			script, err := Script(shell, []string{"a", "list", "echo", "deploy"}, opts)
			if err != nil {
				t.Fatal(err)
			}

			start := t.TempDir()
			back := "cd " + posixQuote(start) + "\n"
			body := script + "\n" + back +
				"cr a\necho \"cr-a=$PWD\"\n" + back +
				"crower execute a\necho \"exec-a=$PWD\"\n" + back +
				"cr list\necho \"list=$PWD\"\n" + back +
				"cr execute list\necho \"exec-list=$PWD\"\n" + back +
				"cr echo\necho \"echo=$PWD\"\n" + back +
				"cr deploy -x\necho \"deploy=$PWD\"\n" +
				"cr\n"
			if shell == "bash" {
				body = "shopt -s expand_aliases\n" + body
			}

			file := filepath.Join(t.TempDir(), "run")
			if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}

			args := []string{"--norc", file}
			if shell == "zsh" {
				args = []string{"-f", file}
			}

			cmd := exec.Command(path, args...)
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: %v\n%s", shell, err, out)
			}

			got := string(out)
			for _, want := range []string{
				"cr-a=" + target + "\n",
				"exec-a=" + target + "\n",
				"ran: list\n", "list=" + start + "\n",
				"exec-list=" + target + "\n",
				"echo=" + target + "\n",
				"ran: deploy -x\n", "deploy=" + start + "\n",
				"ran: \n",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("output lacks %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestWrapperPerShell(t *testing.T) {
	opts := Options{
		Bin:      "cr",
		Dirs:     map[string]string{"a": `C:\Users\Felipe\Desktop\build`, "list": "/srv"},
		Reserved: []string{"list", "create", "bad word"},
	}
	names := []string{"a", "list", "deploy"}

	tests := []struct {
		shell string
		want  []string
	}{
		{"bash", []string{"    create|list|-*) ;;\n", `eval "__crower_cd \"\$__crower_key\" $__crower_w"`, "__crower_cmap=\"\"\n"}},
		{"fish", []string{"        case create list '-*' ''\n", "set -ga __crower_cdnames a\n", "and cd $__crower_cddirs[$__crower_i]"}},
		{"powershell", []string{
			"$args[0] -cnotin @('create', 'list')",
			`$global:__crower_dirs['a'] = 'C:\Users\Felipe\Desktop\build'`,
			"[System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)",
		}},
		{"cmd", []string{
			`crower=if "$1 $2"=="execute a" (call cr execute a -- && cd /d "C:\Users\Felipe\Desktop\build") else if "$1"=="a" (call cr execute a -- && cd /d "C:\Users\Felipe\Desktop\build") else if "$1 $2"=="execute list" (call cr execute list -- && cd /d "/srv") else crower $*` + "\n",
			`cr=if "$1 $2"=="execute a" (`,
			"else cr $*\n",
			"deploy=cr execute deploy -- $*\n",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			script, err := Script(tt.shell, names, opts)
			if err != nil {
				t.Fatal(err)
			}

			for _, want := range tt.want {
				if !strings.Contains(script, want) {
					t.Errorf("script lacks %q:\n%s", want, script)
				}
			}

			if strings.Contains(script, "bad word") {
				t.Errorf("an unsafe reserved word reached the script:\n%s", script)
			}
		})
	}
}

// cmd gets no cr/crower macros when no command changes folder, so the program
// is called directly exactly as before.
func TestCmdWithoutCdCommandsKeepsCr(t *testing.T) {
	script, err := Script("cmd", []string{"deploy"}, Options{Bin: "cr"})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(script, "\ncr=") || strings.HasPrefix(script, "cr=") || strings.Contains(script, "crower=") {
		t.Errorf("cmd defined a cr/crower macro without cd commands:\n%s", script)
	}
}

// End to end in PowerShell, where it is installed (the Windows CI runner):
// "a", "cr a" and "crower execute a" move the shell; "cr list" reaches the
// program because list is a subcommand.
func TestPowerShellWrapperMovesTheShell(t *testing.T) {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh not installed")
	}

	bin := t.TempDir()
	fake := filepath.Join(bin, "crower")
	body := "#!/bin/sh\necho \"ran: $*\"\n"
	if runtime.GOOS == "windows" {
		fake += ".cmd"
		body = "@echo ran: %*\r\n@exit /b 0\r\n"
	}

	if err := os.WriteFile(fake, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "it's here")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	opts := Options{Bin: "crower", Exe: fake, Dirs: map[string]string{"a": target, "list": target}, Reserved: []string{"list"}}
	script, err := Script("powershell", []string{"a", "list"}, opts)
	if err != nil {
		t.Fatal(err)
	}

	start := t.TempDir()
	back := "Set-Location -LiteralPath '" + psQuote(start) + "'\n"
	run := script + "\n" + back +
		"a\n\"bare=$((Get-Location).Path)\"\n" + back +
		"cr a\n\"cr-a=$((Get-Location).Path)\"\n" + back +
		"crower execute a\n\"exec-a=$((Get-Location).Path)\"\n" + back +
		"cr list\n\"list=$((Get-Location).Path)\"\n"

	file := filepath.Join(t.TempDir(), "run.ps1")
	if err := os.WriteFile(file, []byte(run), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-File", file).CombinedOutput()
	if err != nil {
		t.Fatalf("pwsh: %v\n%s", err, out)
	}

	got := strings.ReplaceAll(string(out), "\r\n", "\n")
	if !strings.Contains(got, "ran: list") {
		t.Errorf("cr list did not reach the program:\n%s", got)
	}

	for _, want := range []struct{ key, dir string }{
		{"bare", target}, {"cr-a", target}, {"exec-a", target}, {"list", start},
	} {
		if !endsInDir(got, want.key, want.dir) {
			t.Errorf("%s did not end in %q:\n%s", want.key, want.dir, got)
		}
	}
}

// endsInDir reports whether out has a "key=<path>" line naming dir. Only the
// last two path elements are compared, case-insensitively: Windows may print
// the temp folder by its long name where the test holds its 8.3 short name.
func endsInDir(out, key, dir string) bool {
	tail := strings.ToLower(filepath.Join(filepath.Base(filepath.Dir(dir)), filepath.Base(dir)))

	for _, line := range strings.Split(out, "\n") {
		if value, ok := strings.CutPrefix(line, key+"="); ok && strings.HasSuffix(strings.ToLower(value), tail) {
			return true
		}
	}

	return false
}
