// Package terminal answers the questions an interactive feature asks about the
// terminal -- is this a terminal, how wide is it, may it show colour -- and
// puts it in raw mode for the picker.
//
// It uses golang.org/x/sys directly, the same calls golang.org/x/term makes,
// so crower needs no further module.
package terminal

import (
	"os"
)

// ColorEnabled reports whether f may receive ANSI colour: it is a terminal,
// NO_COLOR (https://no-color.org) is not set, TERM is not "dumb", and on
// Windows the console accepted virtual-terminal sequences.
func ColorEnabled(f *os.File) bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}

	if os.Getenv("TERM") == "dumb" {
		return false
	}

	return IsTerminal(f) && enableANSI(f)
}

// Width is the terminal's column count, or 0 when f is not a terminal.
func Width(f *os.File) int {
	return width(f)
}

// IsTerminal reports whether f is a terminal.
func IsTerminal(f *os.File) bool {
	return isTerminal(f)
}

// MakeRaw puts the terminal into raw mode -- no echo, no line buffering, no
// signal keys -- and returns the function that restores it.
func MakeRaw(f *os.File) (func(), error) {
	return makeRaw(f)
}
