package shellsetup

import (
	"strings"
)

// AutoRunKey and AutoRunValue locate the command cmd.exe runs whenever it
// starts: the one place a cmd window can load crower's macros by itself.
const (
	AutoRunKey   = `Software\Microsoft\Command Processor`
	AutoRunValue = "AutoRun"
)

// AutoRunEntry is what crower adds to AutoRun: load the macro file, and mark
// the window as having the hook.
func AutoRunEntry(macroFile string) string {
	return `doskey /macrofile="` + macroFile + `" & set "` + HookEnv + `=` + Cmd + `"`
}

// HasAutoRun reports whether an AutoRun value already loads the macro file.
func HasAutoRun(existing, macroFile string) bool {
	return strings.Contains(strings.ToLower(existing), strings.ToLower(macroFile))
}

// WithAutoRun is the AutoRun value that also loads the macro file. The user's
// own value is kept and runs first; it reports false when nothing changes.
func WithAutoRun(existing, macroFile string) (string, bool) {
	if HasAutoRun(existing, macroFile) {
		return existing, false
	}

	entry := AutoRunEntry(macroFile)

	if strings.TrimSpace(existing) == "" {
		return entry, true
	}

	return strings.TrimRight(existing, " ") + " & " + entry, true
}
