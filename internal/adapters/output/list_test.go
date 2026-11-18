package output_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"

	"github.com/fiwon123/crower/internal/adapters/output"
	"github.com/fiwon123/crower/internal/application/listcontent"
	"github.com/fiwon123/crower/internal/domain/catalog"
)

var now = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

func listed() []listcontent.ListedCommand {
	return []listcontent.ListedCommand{
		{Row: 0, Command: catalog.Command{Name: "deploy", Icon: "🚀", Aliases: []string{"d"}, Group: "ops", Exec: "ssh prod 'cd /app && git pull'", Description: "ship it"}, Runs: 12, LastRun: now.Add(-2 * time.Hour)},
		{Row: 1, Command: catalog.Command{Name: "which-command-long", Steps: []string{"echo a", "echo b"}}},
	}
}

func TestCommandListPlain(t *testing.T) {
	got := output.CommandList(listed(), output.ListStyle{Now: now})
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("%d lines, want header + 2:\n%s", len(lines), got)
	}

	if strings.Contains(got, "\x1b[") {
		t.Errorf("plain listing carries colour codes:\n%q", got)
	}

	for _, want := range []string{"NAME", "ALIASES", "GROUP", "RUNS", "LAST RUN", "COMMAND"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header %q lacks %q", lines[0], want)
		}
	}

	for _, want := range []string{"🚀 deploy", " d ", "ops", "12", "2h ago", "ssh prod 'cd /app && git pull'", "# ship it"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("row %q lacks %q", lines[1], want)
		}
	}

	if !strings.Contains(lines[2], "echo a && echo b") {
		t.Errorf("steps not joined: %q", lines[2])
	}

	// Every column starts at the same offset on every line: the command
	// column lines up although the names differ in length.
	if runeCol(lines[1], "ssh") != runeCol(lines[2], "echo") {
		t.Errorf("command column misaligned:\n%s", got)
	}

	for _, line := range lines {
		if strings.HasSuffix(line, " ") {
			t.Errorf("trailing space: %q", line)
		}
	}
}

// runeCol is the display column where sub starts.
func runeCol(line, sub string) int {
	return runewidth.StringWidth(line[:strings.Index(line, sub)])
}

func TestCommandListDropsEmptyColumns(t *testing.T) {
	rows := []listcontent.ListedCommand{{Row: 0, Command: catalog.Command{Name: "a", Exec: "echo a"}}}
	header := strings.SplitN(output.CommandList(rows, output.ListStyle{Now: now}), "\n", 2)[0]

	for _, absent := range []string{"ALIASES", "GROUP", "RUNS", "LAST RUN"} {
		if strings.Contains(header, absent) {
			t.Errorf("header %q shows the empty column %s", header, absent)
		}
	}
}

func TestCommandListFitsTheWidth(t *testing.T) {
	got := output.CommandList(listed(), output.ListStyle{Now: now, Width: 60})

	for _, line := range strings.Split(strings.TrimRight(got, "\n"), "\n")[1:] {
		if w := runeCol(line+"|", "|"); w > 60 {
			t.Errorf("line is %d columns, want ≤ 60: %q", w, line)
		}
	}

	if !strings.Contains(got, "…") {
		t.Errorf("a cut line has no ellipsis:\n%s", got)
	}

	// The optional columns made way for the command; the name never does.
	header := strings.SplitN(got, "\n", 2)[0]
	if strings.Contains(header, "LAST RUN") || !strings.Contains(header, "NAME") {
		t.Errorf("header at 60 columns = %q", header)
	}
}

func TestCommandListColour(t *testing.T) {
	got := output.CommandList(listed(), output.ListStyle{Now: now, Color: true})

	if !strings.Contains(got, "\x1b[1m\x1b[36m🚀 deploy\x1b[0m") {
		t.Errorf("name not painted:\n%q", got)
	}
}

func TestAgo(t *testing.T) {
	tests := map[time.Duration]string{
		10 * time.Second:    "just now",
		5 * time.Minute:     "5m ago",
		3 * time.Hour:       "3h ago",
		50 * time.Hour:      "2d ago",
		40 * 24 * time.Hour: now.Add(-40 * 24 * time.Hour).Local().Format("2006-01-02"),
	}

	for d, want := range tests {
		if got := output.Ago(now.Add(-d), now); got != want {
			t.Errorf("Ago(-%s) = %q, want %q", d, got, want)
		}
	}

	if output.Ago(time.Time{}, now) != "" {
		t.Error("a command that never ran has a last run")
	}
}
