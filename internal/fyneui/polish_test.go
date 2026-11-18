//go:build fyne

package fyneui

import (
	"image/color"
	"math"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/fiwon123/crower/internal/adapters/notes"
	"github.com/fiwon123/crower/internal/domain/catalog"
	"github.com/fiwon123/crower/internal/domain/operation"
)

func TestFitText(t *testing.T) {
	useAppTheme(t)

	style := fyne.TextStyle{Monospace: true}
	full := "git fetch --tags origin && go test ./..."
	width := fyne.MeasureText(full, 14, style).Width

	if got := fitText(full, width, 14, style); got != full {
		t.Errorf("a line that fits was cut: %q", got)
	}

	cut := fitText(full, width/2, 14, style)
	if !strings.HasSuffix(cut, "…") || !strings.HasPrefix(full, strings.TrimSuffix(cut, "…")) {
		t.Errorf("cut = %q, want a prefix of the line ending in …", cut)
	}

	if w := fyne.MeasureText(cut, 14, style).Width; w > width/2 {
		t.Errorf("cut is %.1f wide, want at most %.1f", w, width/2)
	}

	if got := fitText(full, 1, 14, style); got != "" {
		t.Errorf("no room = %q, want nothing", got)
	}
}

func TestCommandPreview(t *testing.T) {
	tests := []struct {
		command catalog.Command
		want    string
	}{
		{catalog.Command{Exec: "echo   hi\n"}, "echo hi"},
		{catalog.Command{Exec: "echo hi", Aliases: []string{"hi", "hey"}}, "echo hi  ·  aka hi, hey"},
		{catalog.Command{Exec: "unused", Steps: []string{"make", "make test"}}, "make && make test"},
	}

	for _, tt := range tests {
		if got := commandPreview(tt.command); got != tt.want {
			t.Errorf("commandPreview(%+v) = %q, want %q", tt.command, got, tt.want)
		}
	}
}

// A closed card already says what it runs: the caption is visible, under the
// title, and never wider than the card.
func TestClosedCardShowsWhatItRuns(t *testing.T) {
	long := strings.Repeat("echo something-long ", 40)
	front, _ := commandUI(t, stubExecutor{}, []catalog.Command{{Name: "greet", Icon: "🚀", Exec: long}})

	header := front.commandCards[0].header

	if !header.caption.Visible() || header.detail.full != strings.TrimSpace(strings.Join(strings.Fields(long), " ")) {
		t.Fatalf("caption visible=%v text=%q", header.caption.Visible(), header.detail.full)
	}

	driver := fyne.CurrentApp().Driver()
	title := driver.AbsolutePositionForObject(header.text)
	caption := driver.AbsolutePositionForObject(header.detail)

	if caption.Y <= title.Y {
		t.Errorf("caption y = %.1f, title y = %.1f: the caption is not under the title", caption.Y, title.Y)
	}

	cardRight := driver.AbsolutePositionForObject(front.commandCards[0]).X + front.commandCards[0].Size().Width
	if right := caption.X + header.detail.Size().Width; right > cardRight {
		t.Errorf("caption ends at %.1f, past the card's edge %.1f", right, cardRight)
	}

	if !strings.HasSuffix(header.detail.text.Text, "…") {
		t.Errorf("a long line is not ellipsised: %q", header.detail.text.Text)
	}
}

// A history card has no caption, so its header keeps its one-line height.
func TestHistoryCardHasNoCaption(t *testing.T) {
	note := notes.ExecuteCommandNote(catalog.Command{Name: "greet", Exec: "echo hi"}, nil)
	front, _ := historyEntryUI(t, stubExecutor{}, nil, operation.Execute, "greet", note)

	if front.historyCards[0].header.caption.Visible() {
		t.Error("a history card shows a caption")
	}
}

// An empty field in a history note is not shown as a label with nothing
// beside it.
func TestHistoryOmitsEmptyFields(t *testing.T) {
	note := notes.ExecuteCommandNote(catalog.Command{Name: "greet", Exec: "echo hi"}, nil)
	front, _ := historyEntryUI(t, stubExecutor{}, nil, operation.Execute, "greet", note)

	card := openHistoryCard(t, front, 0)
	labels := labelsIn(card.body)

	if slices.Contains(labels, "Aliases") {
		t.Errorf("an empty Aliases row is shown: %q", labels)
	}

	if !slices.Contains(labels, "Exec") {
		t.Errorf("the Exec row is missing: %q", labels)
	}
}

// Every Tools form shares one label column, so the fields of neighbouring
// cards start at the same x.
func TestToolFieldsShareOneColumn(t *testing.T) {
	front, _ := commandUI(t, stubExecutor{}, nil)
	front.tabs.SelectIndex(2)

	driver := fyne.CurrentApp().Driver()

	var xs []float32
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch typed := o.(type) {
		case *widget.Entry:
			xs = append(xs, driver.AbsolutePositionForObject(typed).X)
		case *widget.Select:
			xs = append(xs, driver.AbsolutePositionForObject(typed).X)
		case *fyne.Container:
			for _, child := range typed.Objects {
				walk(child)
			}
		case fyne.Widget:
			if r := test.WidgetRenderer(typed); r != nil {
				for _, child := range r.Objects() {
					walk(child)
				}
			}
		}
	}

	walk(front.tabs.Items[2].Content)

	if len(xs) < 6 {
		t.Fatalf("found %d fields, want every Tools field", len(xs))
	}

	for _, x := range xs {
		if math.Abs(float64(x-xs[0])) > 0.5 {
			t.Fatalf("field x positions differ: %v", xs)
		}
	}
}

// Text on every coloured button stays readable: at least 4.5:1 in both
// variants (WCAG AA for body text).
func TestButtonTextContrast(t *testing.T) {
	pairs := map[fyne.ThemeColorName]fyne.ThemeColorName{
		theme.ColorNamePrimary: theme.ColorNameForegroundOnPrimary,
		theme.ColorNameSuccess: theme.ColorNameForegroundOnSuccess,
		theme.ColorNameError:   theme.ColorNameForegroundOnError,
		theme.ColorNameWarning: theme.ColorNameForegroundOnWarning,
	}

	th := newAppTheme()

	for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		for fill, text := range pairs {
			ratio := contrast(th.Color(fill, variant), th.Color(text, variant))
			if ratio < 4.5 {
				t.Errorf("variant %d: %s on %s = %.2f:1, want ≥ 4.5", variant, text, fill, ratio)
			}
		}

		// Body text on the page and on a card.
		for _, surface := range []fyne.ThemeColorName{theme.ColorNameBackground, colorNameCard, theme.ColorNameInputBackground} {
			if ratio := contrast(th.Color(surface, variant), th.Color(theme.ColorNameForeground, variant)); ratio < 7 {
				t.Errorf("variant %d: foreground on %s = %.2f:1, want ≥ 7", variant, surface, ratio)
			}
		}
	}
}

func contrast(a, b color.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}

	return (la + 0.05) / (lb + 0.05)
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()

	channel := func(v uint32) float64 {
		s := float64(v) / 0xffff
		if s <= 0.03928 {
			return s / 12.92
		}

		return math.Pow((s+0.055)/1.055, 2.4)
	}

	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}
