//go:build windows

package shellhook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// doskeyExpand does what doskey does when a macro is typed: $1..$9 become the
// words after the macro name, $* the rest of the line, $$ a dollar sign.
func doskeyExpand(body string, args []string) string {
	var b strings.Builder

	for i := 0; i < len(body); i++ {
		if body[i] != '$' || i+1 == len(body) {
			b.WriteByte(body[i])
			continue
		}

		next := body[i+1]
		switch {
		case next >= '1' && next <= '9':
			if n := int(next - '1'); n < len(args) {
				b.WriteString(args[n])
			}
		case next == '*':
			b.WriteString(strings.Join(args, " "))
		case next == '$':
			b.WriteByte('$')
		default:
			b.WriteByte(body[i])
			continue
		}

		i++
	}

	return b.String()
}

// The cmd macros, expanded as doskey would and run through the real cmd.exe:
// "cr a" and "cr execute a" end in the folder, "cr list" (a subcommand) and
// "cr deploy" reach the program and stay put. doskey itself only works in an
// interactive console, so this is as close as a test can get.
func TestCmdMacrosRunInCmd(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "cr.cmd"), []byte("@echo ran: %*\r\n@exit /b 0\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "my (build) dir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	opts := Options{Bin: "cr", Dirs: map[string]string{"a": target, "list": target}, Reserved: []string{"list"}}
	script, err := Script("cmd", []string{"a", "list", "deploy"}, opts)
	if err != nil {
		t.Fatal(err)
	}

	var macro string
	for _, line := range strings.Split(script, "\n") {
		if body, ok := strings.CutPrefix(line, "cr="); ok {
			macro = body
		}
	}

	if macro == "" {
		t.Fatalf("no cr macro:\n%s", script)
	}

	start := t.TempDir()

	tests := []struct {
		args    []string
		wantDir string
		wantRan string
	}{
		{[]string{"a"}, target, "ran: execute a --"},
		{[]string{"execute", "a"}, target, "ran: execute a --"},
		{[]string{"execute", "list"}, target, "ran: execute list --"},
		{[]string{"list"}, start, "ran: list"},
		{[]string{"deploy", "-x"}, start, "ran: deploy -x"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			// Parenthesised: cmd would bind a bare "& cd" to the last else only.
			line := "(" + doskeyExpand(macro, tt.args) + ") & cd"

			cmd := exec.Command("cmd")
			cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /d /s /c "` + line + `"`}
			cmd.Dir = start
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("cmd: %v\n%s", err, out)
			}

			lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")), "\n")
			if got := lines[len(lines)-1]; !endsInDir("cwd="+got, "cwd", tt.wantDir) {
				t.Errorf("ended in %q, want %q\n%s", got, tt.wantDir, out)
			}

			if !strings.Contains(string(out), tt.wantRan) {
				t.Errorf("output lacks %q:\n%s", tt.wantRan, out)
			}
		})
	}
}

// An in-shell command through the cr macro, run by the real cmd.exe: the lines
// crower emits run in this cmd -- a variable survives -- and `call` gives them
// the %VAR% expansion a typed line gets.
func TestCmdHereRunsInCmd(t *testing.T) {
	bin := t.TempDir()
	fake := "@if \"%3\"==\"--emit\" goto emit\r\n@echo ran: %*\r\n@exit /b 0\r\n:emit\r\n@echo set CROWER_T=hi\r\n@echo set CROWER_U=%%USERNAME%%x\r\n@exit /b 0\r\n"
	if err := os.WriteFile(filepath.Join(bin, "cr.cmd"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}

	opts := Options{Bin: "cr", Here: map[string]bool{"act": true}}
	script, err := Script("cmd", []string{"act"}, opts)
	if err != nil {
		t.Fatal(err)
	}

	macros := map[string]string{}
	for _, line := range strings.Split(script, "\n") {
		if name, body, ok := strings.Cut(line, "="); ok {
			macros[name] = body
		}
	}

	tests := []struct {
		macro string
		args  []string
	}{
		{"act", nil},
		{"cr", []string{"act"}},
		{"cr", []string{"execute", "act"}},
	}

	for _, tt := range tests {
		t.Run(tt.macro+" "+strings.Join(tt.args, " "), func(t *testing.T) {
			body, ok := macros[tt.macro]
			if !ok {
				t.Fatalf("no %s macro:\n%s", tt.macro, script)
			}

			line := "(" + doskeyExpand(body, tt.args) + ") & set CROWER_"

			cmd := exec.Command("cmd")
			cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /d /s /c "` + line + `"`}
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("cmd: %v\n%s", err, out)
			}

			got := strings.ReplaceAll(string(out), "\r\n", "\n")
			if !strings.Contains(got, "CROWER_T=hi\n") {
				t.Errorf("the emitted set did not stay in this cmd:\n%s", got)
			}

			want := "CROWER_U=" + os.Getenv("USERNAME") + "x\n"
			if !strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
				t.Errorf("output lacks %q (%%USERNAME%% not expanded):\n%s", want, got)
			}
		})
	}
}
