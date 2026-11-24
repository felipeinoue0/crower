package output

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"github.com/felipeinoue0/crower/internal/application/listcontent"
)

// ListStyle is how the command listing is drawn.
type ListStyle struct {
	// Color turns on ANSI colour; only for a terminal that accepts it.
	Color bool

	// Width is the terminal width. Lines are cut to it with an ellipsis; 0
	// (a pipe or a file) keeps them whole, so grep sees every character.
	Width int

	// Now is the clock "last run" is measured against.
	Now time.Time
}

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiCyan    = "\x1b[36m"
	ansiYellow  = "\x1b[33m"
	ansiMagenta = "\x1b[35m"
	ansiGreen   = "\x1b[32m"
)

// paint wraps s in the codes when colour is on.
func (s ListStyle) paint(text string, codes ...string) string {
	if !s.Color || text == "" {
		return text
	}

	return strings.Join(codes, "") + text + ansiReset
}

// column is one column of the listing: its title, each row's cell, and how a
// cell is painted.
type column struct {
	title string
	cells []string
	codes []string
	right bool
}

// CommandList renders the command listing: one row per command, columns sized
// to their content, empty columns left out, the last column cut to the
// terminal width.
func CommandList(rows []listcontent.ListedCommand, style ListStyle) string {
	if len(rows) == 0 {
		return ""
	}

	if style.Now.IsZero() {
		style.Now = time.Now()
	}

	hasIcon := false
	for _, row := range rows {
		if row.Command.Icon != "" {
			hasIcon = true
			break
		}
	}

	num := column{title: "#", codes: []string{ansiDim}, right: true}
	name := column{title: "NAME", codes: []string{ansiBold, ansiCyan}}
	aliases := column{title: "ALIASES", codes: []string{ansiYellow}}
	group := column{title: "GROUP", codes: []string{ansiMagenta}}
	runs := column{title: "RUNS", codes: []string{ansiGreen}, right: true}
	last := column{title: "LAST RUN", codes: []string{ansiDim}}

	commandCells := make([]string, len(rows))
	descriptions := make([]string, len(rows))

	for i, row := range rows {
		command := row.Command

		num.cells = append(num.cells, strconv.Itoa(row.Row))

		label := command.Name
		if hasIcon {
			icon := command.Icon
			if icon == "" {
				icon = " "
			}

			label = padRight(icon, 2) + " " + label
		}

		name.cells = append(name.cells, label)
		aliases.cells = append(aliases.cells, strings.Join(command.Aliases, ", "))
		group.cells = append(group.cells, command.Group)

		if row.Runs > 0 {
			runs.cells = append(runs.cells, strconv.Itoa(row.Runs))
		} else {
			runs.cells = append(runs.cells, "")
		}

		last.cells = append(last.cells, Ago(row.LastRun, style.Now))

		line := command.Exec
		if len(command.Steps) > 0 {
			line = strings.Join(command.Steps, " && ")
		}

		commandCells[i] = strings.Join(strings.Fields(line), " ")
		descriptions[i] = command.Description
	}

	columns := []column{num, name}
	for _, optional := range []column{aliases, group, runs, last} {
		if !allEmpty(optional.cells) {
			columns = append(columns, optional)
		}
	}

	const gap = "  "

	// On a narrow terminal, drop the optional columns -- last run first --
	// until the command has room to be read.
	const minCommand = 20

	var widths []int

	prefixWidth := 0

	for {
		widths = make([]int, len(columns))
		prefixWidth = 0

		for c, col := range columns {
			widths[c] = runewidth.StringWidth(col.title)
			for _, cell := range col.cells {
				widths[c] = max(widths[c], runewidth.StringWidth(cell))
			}

			prefixWidth += widths[c] + len(gap)
		}

		if style.Width == 0 || style.Width-prefixWidth >= minCommand || len(columns) == 2 {
			break
		}

		columns = columns[:len(columns)-1]
	}

	var b strings.Builder

	// Header.
	var header strings.Builder
	for c, col := range columns {
		header.WriteString(align(col.title, widths[c], col.right))
		header.WriteString(gap)
	}

	header.WriteString("COMMAND")
	b.WriteString(style.paint(strings.TrimRight(header.String(), " "), ansiBold, ansiDim))
	b.WriteString("\n")

	for i := range rows {
		var line strings.Builder

		for c, col := range columns {
			cell := align(col.cells[i], widths[c], col.right)
			// Paint the text, not the padding, so the codes never change
			// the alignment.
			trimmed := strings.TrimRight(cell, " ")
			line.WriteString(strings.Replace(cell, trimmed, style.paint(trimmed, col.codes...), 1))
			line.WriteString(gap)
		}

		text := commandCells[i]
		description := descriptions[i]

		if style.Width > 0 {
			room := style.Width - prefixWidth
			text, description = fit(text, description, room)
		}

		line.WriteString(text)

		if description != "" {
			line.WriteString(style.paint("  # "+description, ansiDim))
		}

		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteString("\n")
	}

	return b.String()
}

// fit cuts the command and its description into room columns: the command
// first, the description only with what is left.
func fit(text, description string, room int) (string, string) {
	if room < 8 {
		room = 8
	}

	if runewidth.StringWidth(text) > room {
		return runewidth.Truncate(text, room, "…"), ""
	}

	left := room - runewidth.StringWidth(text) - len("  # ")
	if description == "" || left < 8 {
		return text, ""
	}

	return text, runewidth.Truncate(description, left, "…")
}

// Ago is a short, human "last run": "just now", "5m ago", "3h ago", "2d ago",
// or the date after a month. A zero time is "".
func Ago(when, now time.Time) string {
	if when.IsZero() {
		return ""
	}

	d := now.Sub(when)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return when.Local().Format("2006-01-02")
	}
}

func align(s string, width int, right bool) string {
	if right {
		return strings.Repeat(" ", max(0, width-runewidth.StringWidth(s))) + s
	}

	return padRight(s, width)
}

func padRight(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-runewidth.StringWidth(s)))
}

func allEmpty(cells []string) bool {
	for _, cell := range cells {
		if cell != "" {
			return false
		}
	}

	return true
}
