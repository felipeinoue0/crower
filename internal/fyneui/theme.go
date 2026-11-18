//go:build fyne

package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const (
	// appTextSize is the body text size. The toolkit's default of 14 leaves an
	// emoji in a dropdown and a command's summary at the edge of legibility on a
	// 1080p screen; every label, button and select grows with it.
	appTextSize float32 = 16

	// appPadding is the window's base spacing unit. It is installed as the
	// theme's padding, so every box, border and form layout that Fyne builds
	// separates its children by exactly this much: buttons in a row, form rows,
	// and the gap around a tab.
	//
	// The toolkit's own 4 px is tight enough that two controls look joined; 8 px
	// reads as two controls that belong together. A hand-inserted gap() spacer
	// must never be added on top of it - a spacer is a normal child to a box
	// layout and brings two more paddings with it (#516).
	appPadding float32 = 8

	// appSection separates groups rather than controls: a card's title from its
	// body, a page from the window's edge, one toolbar group from the next. Two
	// base units, so the break is visible without a rule or a heading.
	appSection float32 = appPadding * 2
)

// appTheme is the default theme with a larger body text and one spacing scale.
type appTheme struct {
	fyne.Theme
}

func newAppTheme() fyne.Theme {
	return appTheme{Theme: theme.DefaultTheme()}
}

func (t appTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return appTextSize
	case theme.SizeNamePadding:
		return appPadding
	}

	return t.Theme.Size(name)
}

// colorNameCard is the surface a card is drawn on. It is the window's own
// name: the toolkit draws cards on the input background, so a field inside a
// card disappeared into it. With its own surface, a card sits one step above
// the page and its fields one step above the card.
const colorNameCard fyne.ThemeColorName = "crowerCard"

// palette is one variant's colours. A name it does not list falls back to the
// toolkit's own.
type palette map[fyne.ThemeColorName]color.Color

func hex(rgb uint32, alpha uint8) color.Color {
	return color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: alpha}
}

// darkPalette and lightPalette are the window's colours: layered neutral
// surfaces, one calm primary, and status colours muted enough to sit side by
// side -- the toolkit's #43f436 success green and pure orange read as alarms.
// Text on a coloured button keeps at least 4.5:1 contrast.
var darkPalette = palette{
	theme.ColorNameBackground:          hex(0x15171c, 0xff),
	colorNameCard:                      hex(0x1d2027, 0xff),
	theme.ColorNameInputBackground:     hex(0x262a33, 0xff),
	theme.ColorNameInputBorder:         hex(0x3a404c, 0xff),
	theme.ColorNameButton:              hex(0x2c313b, 0xff),
	theme.ColorNameDisabledButton:      hex(0x23272f, 0xff),
	theme.ColorNameSeparator:           hex(0x2e333d, 0xff),
	theme.ColorNameForeground:          hex(0xe7e9ee, 0xff),
	theme.ColorNamePlaceHolder:         hex(0x8b93a1, 0xff),
	theme.ColorNameDisabled:            hex(0x5c6370, 0xff),
	theme.ColorNamePrimary:             hex(0x2563eb, 0xff),
	theme.ColorNameForegroundOnPrimary: hex(0xffffff, 0xff),
	theme.ColorNameSuccess:             hex(0x238636, 0xff),
	theme.ColorNameForegroundOnSuccess: hex(0xffffff, 0xff),
	theme.ColorNameError:               hex(0xda3633, 0xff),
	theme.ColorNameForegroundOnError:   hex(0xffffff, 0xff),
	theme.ColorNameWarning:             hex(0xd4a72c, 0xff),
	theme.ColorNameForegroundOnWarning: hex(0x1b1f24, 0xff),
	theme.ColorNameHover:               hex(0xffffff, 0x14),
	theme.ColorNamePressed:             hex(0xffffff, 0x24),
	theme.ColorNameFocus:               hex(0x3b82f6, 0xb0),
	theme.ColorNameSelection:           hex(0x3b82f6, 0x55),
	theme.ColorNameHyperlink:           hex(0x6ca4f8, 0xff),
	theme.ColorNameOverlayBackground:   hex(0x1d2027, 0xff),
	theme.ColorNameMenuBackground:      hex(0x232731, 0xff),
	theme.ColorNameHeaderBackground:    hex(0x1d2027, 0xff),
	theme.ColorNameShadow:              hex(0x000000, 0x66),
	theme.ColorNameScrollBar:           hex(0xffffff, 0x33),
}

var lightPalette = palette{
	theme.ColorNameBackground:          hex(0xf3f5f8, 0xff),
	colorNameCard:                      hex(0xffffff, 0xff),
	theme.ColorNameInputBackground:     hex(0xe9edf2, 0xff),
	theme.ColorNameInputBorder:         hex(0xd0d7de, 0xff),
	theme.ColorNameButton:              hex(0xe7eaef, 0xff),
	theme.ColorNameDisabledButton:      hex(0xeef0f3, 0xff),
	theme.ColorNameSeparator:           hex(0xdde2e8, 0xff),
	theme.ColorNameForeground:          hex(0x1f2328, 0xff),
	theme.ColorNamePlaceHolder:         hex(0x6b7380, 0xff),
	theme.ColorNameDisabled:            hex(0xa0a7b3, 0xff),
	theme.ColorNamePrimary:             hex(0x2563eb, 0xff),
	theme.ColorNameForegroundOnPrimary: hex(0xffffff, 0xff),
	theme.ColorNameSuccess:             hex(0x1a7f37, 0xff),
	theme.ColorNameForegroundOnSuccess: hex(0xffffff, 0xff),
	theme.ColorNameError:               hex(0xcf222e, 0xff),
	theme.ColorNameForegroundOnError:   hex(0xffffff, 0xff),
	theme.ColorNameWarning:             hex(0xd4a72c, 0xff),
	theme.ColorNameForegroundOnWarning: hex(0x1f2328, 0xff),
	theme.ColorNameHover:               hex(0x000000, 0x0f),
	theme.ColorNamePressed:             hex(0x000000, 0x1f),
	theme.ColorNameFocus:               hex(0x2563eb, 0xa0),
	theme.ColorNameSelection:           hex(0x2563eb, 0x40),
	theme.ColorNameHyperlink:           hex(0x2563eb, 0xff),
	theme.ColorNameOverlayBackground:   hex(0xffffff, 0xff),
	theme.ColorNameMenuBackground:      hex(0xffffff, 0xff),
	theme.ColorNameHeaderBackground:    hex(0xffffff, 0xff),
	theme.ColorNameShadow:              hex(0x000000, 0x33),
	theme.ColorNameScrollBar:           hex(0x000000, 0x33),
}

// Color is the window's palette for the variant, and the toolkit's for any
// name the palette does not list.
func (t appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	colors := darkPalette
	if variant == theme.VariantLight {
		colors = lightPalette
	}

	if c, ok := colors[name]; ok {
		return c
	}

	return t.Theme.Color(name, variant)
}
