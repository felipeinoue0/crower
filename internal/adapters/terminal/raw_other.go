//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package terminal

import (
	"errors"
	"os"
)

func isTerminal(*os.File) bool { return false }
func width(*os.File) int       { return 0 }
func enableANSI(*os.File) bool { return false }
func makeRaw(*os.File) (func(), error) {
	return nil, errors.New("raw terminal mode is not supported on this platform")
}
