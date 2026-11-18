//go:build fyne

package fyneui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// The embedded catalogue is the full gemoji set, with every entry usable.
func TestEmojiCatalogue(t *testing.T) {
	entries := loadEmojiCatalogue()

	// gemoji has ~1,390 simple glyphs; ~300 are newer than the bundled emoji
	// font and are dropped (#543).
	if len(entries) < 1000 {
		t.Fatalf("catalogue = %d entries, want the renderable gemoji set", len(entries))
	}

	drawable := bundledGlyphs()

	categories := make(map[string]bool)
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.Emoji == "" {
			t.Fatal("the catalogue has an entry without a glyph")
		}

		if !simpleEmoji(entry.Emoji) {
			t.Errorf("%q is a complex sequence the picker should have dropped", entry.Emoji)
		}

		if !drawable(entry.Emoji) {
			t.Errorf("%q is in no bundled font, so the picker would show an empty cell", entry.Emoji)
		}

		if seen[entry.Emoji] {
			t.Errorf("%q appears twice", entry.Emoji)
		}

		seen[entry.Emoji] = true
		categories[entry.Category] = true
	}

	for _, want := range emojiCategories {
		if !categories[want] {
			t.Errorf("category %q is missing from the catalogue", want)
		}
	}
}

// The picker keeps the glyphs the bundled fonts draw and drops the ones that
// would fall back to a system font the painter cannot draw (#543).
func TestEmojiCatalogueDropsGlyphsNoBundledFontDraws(t *testing.T) {
	drawable := bundledGlyphs()

	for _, glyph := range []string{"🚀", "📦", "🏷️", "❤️", "😀", "#️⃣"} {
		if !drawable(glyph) {
			t.Errorf("%q should be drawable from the bundled fonts", glyph)
		}
	}

	offered := make(map[string]bool)
	for _, entry := range loadEmojiCatalogue() {
		offered[entry.Emoji] = true
	}

	// Emoji 11+ glyphs and a subdivision flag, none in EmojiOne.
	for _, glyph := range []string{"🧠", "🥰", "🟢", "🧰", "🏴\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f"} {
		if drawable(glyph) {
			t.Errorf("%q should not be drawable from the bundled fonts", glyph)
		}

		if offered[glyph] {
			t.Errorf("the picker offers %q, which would render as an empty cell", glyph)
		}
	}

	if !offered["🚀"] {
		t.Error("the picker dropped 🚀, which the bundled emoji font draws")
	}
}

// The picker filters by name, alias or tag, and by category.
func TestFilterEmoji(t *testing.T) {
	entries := loadEmojiCatalogue()

	all := filterEmoji(entries, "", "All")
	if len(all) != len(entries) {
		t.Errorf("all = %d, want %d", len(all), len(entries))
	}

	smiles := filterEmoji(entries, "SMILE", "All")
	if len(smiles) == 0 {
		t.Error("query SMILE matched nothing")
	}

	found := false
	for _, entry := range smiles {
		if entry.Emoji == "😀" {
			found = true
		}
	}

	if !found {
		t.Error("query SMILE did not match the grinning face")
	}

	rocket := filterEmoji(entries, "🚀", "All")
	if len(rocket) == 0 {
		t.Error("searching for the emoji glyph matched nothing")
	}

	flagMatches := filterEmoji(entries, "flag", "Flags")
	if len(flagMatches) == 0 {
		t.Error("category+query matched nothing")
	}

	for _, entry := range flagMatches {
		if entry.Category != "Flags" || !emojiMatches(entry, "flag") {
			t.Errorf("%q does not satisfy the category and query", entry.Emoji)
		}
	}

	flags := filterEmoji(entries, "", "Flags")
	if len(flags) == 0 {
		t.Fatal("the Flags category is empty")
	}

	for _, entry := range flags {
		if entry.Category != "Flags" {
			t.Errorf("%q is in %q, want Flags", entry.Emoji, entry.Category)
		}
	}

	if got := filterEmoji(entries, "definitely-not-an-emoji", "All"); len(got) != 0 {
		t.Errorf("nonsense query = %d entries, want none", len(got))
	}

	if got := emojiRows(0); got != 0 {
		t.Errorf("emojiRows(0) = %d, want 0", got)
	}

	if got := emojiRows(emojiColumns + 1); got != 2 {
		t.Errorf("emojiRows(%d) = %d, want 2", emojiColumns+1, got)
	}
}

// The icon field is one large emoji button — no separate Clear — and the
// catalogue offers the empty choice, so a row stays a row.
func TestIconFieldIsOneButtonWithAnEmptyChoice(t *testing.T) {
	front, window := newTestUI(t, testModel(), stubEditor{}, stubReverter{})

	choice, field := front.iconField("🚀")

	if choice.button.text.Text != "🚀" {
		t.Errorf("display = %q, want the emoji", choice.button.text.Text)
	}

	if hasEntry(field) {
		t.Error("the icon field contains an editable entry")
	}

	if findButton(field, "Clear") != nil {
		t.Error("the icon field still carries a Clear button")
	}

	choice.button.Tapped(&fyne.PointEvent{})

	top := window.Canvas().Overlays().Top()
	if top == nil {
		t.Fatal("the emoji button did not open the catalogue")
	}

	noIcon := findButton(top, "No icon")
	if noIcon == nil {
		t.Fatal("the catalogue has no empty choice")
	}

	noIcon.OnTapped()

	if choice.value != "" || choice.button.text.Text != "Pick" {
		t.Errorf("cleared field = %q/%q, want Pick", choice.value, choice.button.text.Text)
	}
}

// The emoji button draws the selection large; an empty value says Pick.
func TestEmojiButtonShowsTheSelection(t *testing.T) {
	button := newEmojiButton("", nil)

	if button.text.Text != "Pick" {
		t.Errorf("empty button = %q, want Pick", button.text.Text)
	}

	button.SetEmoji("🚀")

	if button.text.Text != "🚀" || button.text.TextSize < 24 {
		t.Errorf("selected button = %q at %.1f pt, want the emoji large", button.text.Text, button.text.TextSize)
	}
}

// hasEntry reports whether an editable entry is reachable.
func hasEntry(object fyne.CanvasObject) bool {
	switch typed := object.(type) {
	case *widget.Entry:
		return true
	case *fyne.Container:
		for _, child := range typed.Objects {
			if hasEntry(child) {
				return true
			}
		}
	case fyne.Widget:
		renderer := test.WidgetRenderer(typed)
		if renderer == nil {
			return false
		}

		for _, child := range renderer.Objects() {
			if hasEntry(child) {
				return true
			}
		}
	}

	return false
}

// findButton returns the button with exactly this text, or nil.
func findButton(object fyne.CanvasObject, text string) *widget.Button {
	switch typed := object.(type) {
	case *appButton:
		if strings.TrimSpace(typed.Text) == text {
			return typed.Button
		}
	case *widget.Button:
		if strings.TrimSpace(typed.Text) == text {
			return typed
		}
	case *fyne.Container:
		for _, child := range typed.Objects {
			if button := findButton(child, text); button != nil {
				return button
			}
		}
	case fyne.Widget:
		renderer := test.WidgetRenderer(typed)
		if renderer == nil {
			return nil
		}

		for _, child := range renderer.Objects() {
			if button := findButton(child, text); button != nil {
				return button
			}
		}
	}

	return nil
}

// The picker cells draw the glyph at a readable size, not the theme's 14 pt.
func TestEmojiCellIsLarge(t *testing.T) {
	cell, ok := emojiCell().(*fyne.Container)
	if !ok || len(cell.Objects) == 0 {
		t.Fatalf("emojiCell = %T, want a container with a glyph", emojiCell())
	}

	text, ok := cell.Objects[0].(*canvas.Text)
	if !ok {
		t.Fatalf("cell object = %T, want a canvas text", cell.Objects[0])
	}

	if text.TextSize != emojiTextSize || emojiTextSize < 24 {
		t.Errorf("emoji text size = %.1f, want the catalog size (>= 24)", text.TextSize)
	}
}

// Selecting a cell picks that emoji and closes the pop-up.
func TestEmojiPickerPicksACell(t *testing.T) {
	front, window := newTestUI(t, testModel(), stubEditor{}, stubReverter{})

	picked := ""
	front.showEmojiPicker("", func(emoji string) { picked = emoji })

	overlays := window.Canvas().Overlays().List()
	if len(overlays) == 0 {
		t.Fatal("the picker did not open")
	}

	table := findTable(overlays[len(overlays)-1])
	if table == nil {
		t.Fatal("the picker has no table")
	}

	table.OnSelected(widget.TableCellID{Row: 0, Col: 0})

	if want := loadEmojiCatalogue()[0].Emoji; picked != want {
		t.Errorf("picked %q, want %q", picked, want)
	}
}

// findTable returns the first table in a rendered tree.
func findTable(object fyne.CanvasObject) *widget.Table {
	switch typed := object.(type) {
	case *widget.Table:
		return typed
	case *fyne.Container:
		for _, child := range typed.Objects {
			if table := findTable(child); table != nil {
				return table
			}
		}
	case fyne.Widget:
		renderer := test.WidgetRenderer(typed)
		if renderer == nil {
			return nil
		}

		for _, child := range renderer.Objects() {
			if table := findTable(child); table != nil {
				return table
			}
		}
	}

	return nil
}

// An icon set before #543 (from the CLI, an import or an older picker) can be a
// glyph no bundled font draws. The window leaves it out of titles and labels so
// the name keeps its left edge, and the button says an icon is set (#547).
func TestUndrawableIconIsLeftOutOfTheWindow(t *testing.T) {
	const brain = "🧠" // Emoji 10, not in the bundled EmojiOne set

	if got := drawableIcon(brain); got != "" {
		t.Fatalf("drawableIcon(%q) = %q, want it dropped", brain, got)
	}

	if got := drawableIcon(" 🚀 "); got != "🚀" {
		t.Errorf("drawableIcon(🚀) = %q, want it kept", got)
	}

	header := newCardHeader(brain, "think")
	if header.text.Text != "think" {
		t.Errorf("header = %q, want the name alone", header.text.Text)
	}

	// The caption is indented by the icon's width; an icon that is not drawn
	// must not indent it.
	plain := newCardHeader("", "think")
	header.setDetail("echo hi")
	plain.setDetail("echo hi")

	if plain.caption.MinSize() != header.caption.MinSize() {
		t.Errorf("caption indented by an icon that is not drawn: %v vs %v", header.caption.MinSize(), plain.caption.MinSize())
	}

	if got := groupLabel(brain, "ideas"); got != "ideas" {
		t.Errorf("groupLabel = %q, want the name alone", got)
	}

	button := newEmojiButton(brain, nil)
	if button.text.Text != "?" || button.value != brain {
		t.Errorf("button shows %q holding %q; want ? and the stored icon kept", button.text.Text, button.value)
	}
}
