//go:build fyne

package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// ellipsisText is one line of text that fits the width it is given, ending in
// "…" when it is cut. Its minimum width is zero, so a long command line never
// widens the card or the window; canvas.Text alone takes its full width.
//
// It is not tappable, so a click on it reaches the card header around it.
type ellipsisText struct {
	widget.BaseWidget

	full  string
	style fyne.TextStyle
	size  float32
	color func() color.Color
	text  *canvas.Text
}

func newEllipsisText(full string, size float32, style fyne.TextStyle, colour func() color.Color) *ellipsisText {
	e := &ellipsisText{full: full, style: style, size: size, color: colour}
	e.text = canvas.NewText(full, colour())
	e.text.TextSize = size
	e.text.TextStyle = style
	e.ExtendBaseWidget(e)

	return e
}

// SetText replaces the full text; it is cut again at the next layout.
func (e *ellipsisText) SetText(full string) {
	e.full = full
	e.Refresh()
}

// CreateRenderer hands the text to the canvas.
func (e *ellipsisText) CreateRenderer() fyne.WidgetRenderer {
	return &ellipsisRenderer{owner: e}
}

type ellipsisRenderer struct {
	owner *ellipsisText
}

func (r *ellipsisRenderer) Layout(size fyne.Size) {
	t := r.owner.text
	t.Text = fitText(r.owner.full, size.Width, r.owner.size, r.owner.style)
	t.Resize(fyne.NewSize(size.Width, t.MinSize().Height))
	t.Move(fyne.NewPos(0, (size.Height-t.MinSize().Height)/2))
}

func (r *ellipsisRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, fyne.MeasureText("Mg", r.owner.size, r.owner.style).Height)
}

func (r *ellipsisRenderer) Refresh() {
	r.owner.text.Color = r.owner.color()
	r.Layout(r.owner.Size())
	r.owner.text.Refresh()
}

func (r *ellipsisRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.owner.text} }
func (r *ellipsisRenderer) Destroy()                     {}

// fitText is the longest prefix of full that, with "…", fits width.
func fitText(full string, width, size float32, style fyne.TextStyle) string {
	if width <= 0 || fyne.MeasureText(full, size, style).Width <= width {
		return full
	}

	runes := []rune(full)
	low, high := 0, len(runes)

	for low < high {
		mid := (low + high + 1) / 2
		if fyne.MeasureText(string(runes[:mid])+"…", size, style).Width <= width {
			low = mid
		} else {
			high = mid - 1
		}
	}

	if low == 0 {
		return ""
	}

	return string(runes[:low]) + "…"
}
