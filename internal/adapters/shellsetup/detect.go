// Package shellsetup finds the shell crower was started from and decides
// whether to offer installing the shell hook there.
//
// A program cannot change the folder or the variables of the shell that
// started it; only code running inside that shell can. The hook is that code,
// and this package is what lets a user get it without knowing it exists: it
// names the parent shell, and the cmd package installs the hook for it.
package shellsetup

import (
	"os"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// The shells crower can set up. Pwsh is PowerShell 7+; PowerShell is Windows
// PowerShell 5.1. They read different profile files but load the same hook.
const (
	Bash       = "bash"
	Zsh        = "zsh"
	Fish       = "fish"
	Pwsh       = "pwsh"
	PowerShell = "powershell"
	Cmd        = "cmd"
)

// HookEnv is the variable a loaded hook sets to its HookName, so crower can
// tell that the shell it runs in already has the hook.
const HookEnv = "CROWER_HOOK"

// ByName maps a process or program name -- "zsh", "/bin/bash", "pwsh.exe",
// "-zsh" for a login shell -- to the shell it is, or "" for anything else.
func ByName(name string) string {
	// Either separator: a Windows path is named the same on any platform.
	base := strings.TrimSpace(name)
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}

	base = strings.ToLower(base)
	base = strings.TrimPrefix(base, "-")
	base = strings.TrimSuffix(base, ".exe")

	switch base {
	case Bash, Zsh, Fish, Pwsh, Cmd:
		return base
	case PowerShell:
		return PowerShell
	default:
		return ""
	}
}

// HookName is the name a shell's hook is generated for (`crower init <name>`):
// both PowerShells share one.
func HookName(shell string) string {
	if shell == Pwsh {
		return PowerShell
	}

	return shell
}

// Parent is the shell that started this process, or "" when the parent is not
// a shell crower knows (a script runner, an IDE, sudo).
func Parent() string {
	p, err := process.NewProcess(int32(os.Getppid()))
	if err != nil {
		return ""
	}

	if name, err := p.Name(); err == nil {
		if shell := ByName(name); shell != "" {
			return shell
		}
	}

	exe, err := p.Exe()
	if err != nil {
		return ""
	}

	return ByName(exe)
}

// HookLoaded reports whether the shell's hook is loaded in the environment
// crower was started from.
func HookLoaded(shell string, getenv func(string) string) bool {
	return shell != "" && getenv(HookEnv) == HookName(shell)
}
