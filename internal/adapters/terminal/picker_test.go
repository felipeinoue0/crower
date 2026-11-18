package terminal

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestScoreRanksWordStartsAndRuns(t *testing.T) {
	tests := []struct {
		better, worse string
	}{
		{"deploy", "node-pre-load-yaml"}, // a prefix beats scattered letters
		{"git-push", "the-gimp-utility"}, // a word start beats a mid-word hit
		{"dep", "deploy"},                // the shorter of two equal matches
	}

	for _, tt := range tests {
		b, okB := Score("dep", tt.better)
		w, okW := Score("dep", tt.worse)
		if tt.better == "git-push" {
			b, okB = Score("gp", tt.better)
			w, okW = Score("gp", tt.worse)
		}

		if !okB || !okW || b <= w {
			t.Errorf("score(%q)=%d,%v must beat score(%q)=%d,%v", tt.better, b, okB, tt.worse, w, okW)
		}
	}

	if _, ok := Score("xyz", "deploy"); ok {
		t.Error("a query whose letters are absent matched")
	}

	if _, ok := Score("ypd", "deploy"); ok {
		t.Error("letters out of order matched")
	}

	if s, ok := Score("", "anything"); !ok || s != 0 {
		t.Errorf("empty query = %d, %v; want 0, true", s, ok)
	}
}

func items() []Item {
	return []Item{
		{Title: "greet", Detail: "echo hello", Search: "greet hi echo hello"},
		{Title: "deploy", Detail: "ssh prod", Search: "deploy d ssh prod"},
		{Title: "logs", Detail: "journalctl", Search: "logs journalctl"},
	}
}

// pick types keys into the finder and returns its choice and what it drew.
func pick(t *testing.T, keys string) (int, string) {
	t.Helper()

	var out bytes.Buffer

	choice, err := run(bufio.NewReader(strings.NewReader(keys)), &out, items(), 80, 5)
	if err != nil {
		t.Fatalf("run(%q): %v", keys, err)
	}

	return choice, out.String()
}

func TestPickerKeys(t *testing.T) {
	tests := []struct {
		name string
		keys string
		want int
	}{
		{"enter takes the first", "\r", 0},
		{"typing filters", "dep\r", 1},
		{"arrow down", "\x1b[B\r", 1},
		{"ss3 arrow down", "\x1bOB\x1bOB\r", 2},
		{"down then up", "\x1b[B\x1b[B\x1b[A\r", 1},
		{"ctrl+n and ctrl+p", "\x0e\x0e\x10\r", 1},
		{"tab moves down", "\t\r", 1},
		{"down stops at the end", "\x1b[B\x1b[B\x1b[B\x1b[B\r", 2},
		{"backspace widens again", "logx\x7f\r", 2},
		{"ctrl+u clears", "zzz\x15\r", 0},
		{"esc leaves", "\x1b", -1},
		{"ctrl+c leaves", "dep\x03", -1},
		{"enter with no match does nothing", "zzz\r\x03", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := pick(t, tt.keys); got != tt.want {
				t.Errorf("keys %q chose %d, want %d", tt.keys, got, tt.want)
			}
		})
	}
}

// The finder erases what it drew and gives the cursor back, so the terminal
// is left as it was found.
func TestPickerCleansUp(t *testing.T) {
	_, drawn := pick(t, "\x1b")

	if !strings.HasPrefix(drawn, "\x1b[?25l") || !strings.HasSuffix(drawn, "\r\x1b[J\x1b[?25h") {
		t.Errorf("finder did not hide/restore the cursor and erase:\n%q", drawn)
	}

	if !strings.Contains(drawn, "3/3") || !strings.Contains(drawn, "esc quit") {
		t.Errorf("finder lacks its count or hint:\n%q", drawn)
	}
}

// Input that ends mid-run is an error, not a silent choice.
func TestPickerEndOfInput(t *testing.T) {
	var out bytes.Buffer

	if _, err := run(bufio.NewReader(strings.NewReader("de")), &out, items(), 80, 5); err == nil {
		t.Error("end of input returned no error")
	}
}
