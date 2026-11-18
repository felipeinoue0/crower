package terminal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// Item is one entry the picker offers.
type Item struct {
	// Title is what the entry is called: "🚀 deploy".
	Title string

	// Detail is shown dimmed beside the title: the line it runs.
	Detail string

	// Search is the text the query is matched against.
	Search string
}

// Pick draws an inline fuzzy finder under the cursor and returns the index of
// the chosen item, or -1 when the user left with Esc or Ctrl+C.
//
// in must be the terminal; out is where the finder is drawn (stderr, so a
// command's own output on stdout stays clean). The finder takes only the lines
// it needs and erases them when it is done.
func Pick(in, out *os.File, items []Item, height int) (int, error) {
	restore, err := MakeRaw(in)
	if err != nil {
		return -1, err
	}
	defer restore()

	enableANSI(out)

	return run(bufio.NewReader(in), out, items, Width(out), height)
}

// run is the finder over any reader and writer, so a test can type into it.
func run(in *bufio.Reader, out io.Writer, items []Item, width, height int) (int, error) {
	if width <= 0 {
		width = 80
	}

	p := newPicker(items, height)

	fmt.Fprint(out, "\x1b[?25l") // hide the cursor; the finder draws its own
	defer fmt.Fprint(out, "\x1b[?25h")

	drawn := 0

	for {
		lines := p.view(width)

		if drawn > 1 {
			fmt.Fprintf(out, "\r\x1b[%dA", drawn-1)
		}

		fmt.Fprint(out, "\r\x1b[J"+strings.Join(lines, "\r\n"))
		drawn = len(lines)

		k, err := readKey(in)
		if err != nil {
			erase(out, drawn)
			return -1, err
		}

		if done, choice := p.handle(k); done {
			erase(out, drawn)
			return choice, nil
		}
	}
}

// erase removes the finder's lines and leaves the cursor where it started.
func erase(out io.Writer, drawn int) {
	if drawn > 1 {
		fmt.Fprintf(out, "\r\x1b[%dA", drawn-1)
	}

	fmt.Fprint(out, "\r\x1b[J")
}

// key is one keystroke, decoded.
type key struct {
	kind keyKind
	r    rune
}

type keyKind int

const (
	keyRune keyKind = iota
	keyEnter
	keyCancel
	keyUp
	keyDown
	keyBackspace
	keyClear
	keyDeleteWord
	keyIgnore
)

// readKey decodes one keystroke from a raw terminal: the control keys, the
// arrow escape sequences (CSI and SS3 forms), and any printable rune.
func readKey(in *bufio.Reader) (key, error) {
	r, _, err := in.ReadRune()
	if err != nil {
		return key{}, err
	}

	switch r {
	case '\r', '\n':
		return key{kind: keyEnter}, nil
	case 3, 4: // Ctrl+C, Ctrl+D
		return key{kind: keyCancel}, nil
	case 0x7f, 0x08:
		return key{kind: keyBackspace}, nil
	case 0x10, 0x0b: // Ctrl+P, Ctrl+K
		return key{kind: keyUp}, nil
	case 0x0e, '\t': // Ctrl+N, Tab
		return key{kind: keyDown}, nil
	case 0x15: // Ctrl+U
		return key{kind: keyClear}, nil
	case 0x17: // Ctrl+W
		return key{kind: keyDeleteWord}, nil
	case 0x1b:
		// A lone Esc leaves; an escape sequence arrives in the same read, so
		// its bytes are already buffered.
		if in.Buffered() == 0 {
			return key{kind: keyCancel}, nil
		}

		next, _, _ := in.ReadRune()
		if next != '[' && next != 'O' {
			return key{kind: keyIgnore}, nil
		}

		final, _, _ := in.ReadRune()
		for final >= '0' && final <= '9' || final == ';' {
			final, _, _ = in.ReadRune()
		}

		switch final {
		case 'A':
			return key{kind: keyUp}, nil
		case 'B':
			return key{kind: keyDown}, nil
		}

		return key{kind: keyIgnore}, nil
	}

	if unicode.IsPrint(r) {
		return key{kind: keyRune, r: r}, nil
	}

	return key{kind: keyIgnore}, nil
}

// picker is the finder's state: the query, the matching items in rank order,
// and the selection.
type picker struct {
	items   []Item
	height  int
	query   []rune
	matches []int
	cursor  int
	offset  int
}

func newPicker(items []Item, height int) *picker {
	if height <= 0 {
		height = 10
	}

	p := &picker{items: items, height: min(height, max(len(items), 1))}
	p.filter()

	return p
}

// handle applies a key. It reports whether the finder is done, and the chosen
// index (-1 for none).
func (p *picker) handle(k key) (bool, int) {
	switch k.kind {
	case keyEnter:
		if len(p.matches) == 0 {
			return false, -1
		}

		return true, p.matches[p.cursor]
	case keyCancel:
		return true, -1
	case keyUp:
		if p.cursor > 0 {
			p.cursor--
		}
	case keyDown:
		if p.cursor < len(p.matches)-1 {
			p.cursor++
		}
	case keyBackspace:
		if len(p.query) > 0 {
			p.query = p.query[:len(p.query)-1]
			p.filter()
		}
	case keyClear:
		p.query = p.query[:0]
		p.filter()
	case keyDeleteWord:
		trimmed := strings.TrimRight(string(p.query), " ")
		if i := strings.LastIndex(trimmed, " "); i >= 0 {
			p.query = []rune(trimmed[:i+1])
		} else {
			p.query = p.query[:0]
		}

		p.filter()
	case keyRune:
		p.query = append(p.query, k.r)
		p.filter()
	}

	// Keep the selection on screen.
	if p.cursor < p.offset {
		p.offset = p.cursor
	}

	if p.cursor >= p.offset+p.height {
		p.offset = p.cursor - p.height + 1
	}

	return false, -1
}

// filter re-ranks the items against the query and resets the selection.
func (p *picker) filter() {
	query := strings.ToLower(string(p.query))

	type ranked struct {
		index, score int
	}

	var found []ranked

	for i, item := range p.items {
		if s, ok := Score(query, item.Search); ok {
			found = append(found, ranked{i, s})
		}
	}

	slices.SortStableFunc(found, func(a, b ranked) int { return b.score - a.score })

	p.matches = p.matches[:0]
	for _, f := range found {
		p.matches = append(p.matches, f.index)
	}

	p.cursor, p.offset = 0, 0
}

// view is the finder's lines: the query, the visible matches, and a hint. It
// always has the same number of lines, so a redraw covers the last one.
func (p *picker) view(width int) []string {
	const (
		reset = "\x1b[0m"
		dim   = "\x1b[2m"
		bold  = "\x1b[1m"
		cyan  = "\x1b[36m"
		inv   = "\x1b[7m"
	)

	lines := make([]string, 0, p.height+2)

	count := fmt.Sprintf("%d/%d", len(p.matches), len(p.items))
	prompt := cyan + bold + "❯ " + reset + string(p.query) + inv + " " + reset
	pad := width - 3 - runewidth.StringWidth(string(p.query)) - len(count) - 1
	lines = append(lines, prompt+strings.Repeat(" ", max(1, pad))+dim+count+reset)

	titleWidth := 0
	for _, index := range p.matches {
		titleWidth = max(titleWidth, runewidth.StringWidth(p.items[index].Title))
	}

	titleWidth = min(titleWidth, 32)

	for row := 0; row < p.height; row++ {
		at := p.offset + row
		if at >= len(p.matches) {
			lines = append(lines, "")
			continue
		}

		item := p.items[p.matches[at]]
		title := runewidth.FillRight(runewidth.Truncate(item.Title, titleWidth, "…"), titleWidth)
		room := width - 2 - titleWidth - 2 - 1
		detail := runewidth.Truncate(strings.Join(strings.Fields(item.Detail), " "), max(room, 0), "…")

		if at == p.cursor {
			lines = append(lines, cyan+bold+"▌ "+title+reset+"  "+dim+detail+reset)
		} else {
			lines = append(lines, "  "+title+"  "+dim+detail+reset)
		}
	}

	if len(p.matches) == 0 {
		lines[1] = dim + "  no match" + reset
	}

	lines = append(lines, dim+"  ↑↓ move · enter run · esc quit"+reset)

	return lines
}

// Score is a fuzzy match of query (lower case) against text: every query
// rune must appear in order. Consecutive runs, word starts and a match at the
// very start score higher. An empty query matches everything equally.
func Score(query, text string) (int, bool) {
	if query == "" {
		return 0, true
	}

	q := []rune(query)
	t := []rune(strings.ToLower(text))

	score, qi, last := 0, 0, -2

	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if t[ti] != q[qi] {
			continue
		}

		score++

		if ti == last+1 {
			score += 5
		}

		if ti == 0 {
			score += 10
		} else if strings.ContainsRune(" -_./:", t[ti-1]) {
			score += 6
		}

		last = ti
		qi++
	}

	if qi < len(q) {
		return 0, false
	}

	// Among equal matches, the shorter text is the closer one.
	return score*100 - len(t), true
}
