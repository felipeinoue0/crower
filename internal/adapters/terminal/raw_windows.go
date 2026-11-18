package terminal

import (
	"os"

	"golang.org/x/sys/windows"
)

func isTerminal(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}

func width(f *os.File) int {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(f.Fd()), &info); err != nil {
		return 0
	}

	return int(info.Window.Right-info.Window.Left) + 1
}

// enableANSI turns on virtual-terminal processing, which Windows 10 and later
// support; a console that refuses it gets no colour.
func enableANSI(f *os.File) bool {
	handle := windows.Handle(f.Fd())

	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}

	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}

	return windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}

func makeRaw(f *os.File) (func(), error) {
	handle := windows.Handle(f.Fd())

	var saved uint32
	if err := windows.GetConsoleMode(handle, &saved); err != nil {
		return nil, err
	}

	// Virtual-terminal input makes the arrow keys arrive as the same escape
	// sequences a Unix terminal sends.
	raw := saved &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_LINE_INPUT)
	raw |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT

	if err := windows.SetConsoleMode(handle, raw); err != nil {
		return nil, err
	}

	return func() { _ = windows.SetConsoleMode(handle, saved) }, nil
}
