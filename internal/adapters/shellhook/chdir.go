package shellhook

import (
	"strings"
)

// A command whose whole line is "cd <folder>" cannot work through
// `crower execute`: the cd runs in a child shell, which exits, and the
// terminal that typed the name never moves. The hook runs inside that
// terminal's shell, so for such a command it defines the name to change the
// directory itself — after `crower execute` succeeds, so the run is still
// recorded and a folder that does not exist stops the jump with the shell's
// own error.

// ChangeDir reports the folder a command line only changes into.
//
// It recognises `cd <dir>`, `chdir <dir>` and cmd's `cd /d <dir>`, where
// <dir> is one word — bare, or wrapped in single or double quotes — that is
// an absolute path or starts with "~". Anything else is an ordinary command:
// a relative path (it depends on the folder the command runs in), a second
// command, a variable, a glob or a parameter placeholder. Refusing those keeps
// the hook from ever writing something into the shell that it would read
// differently than `crower execute` does.
func ChangeDir(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", false
	}

	verb := strings.ToLower(fields[0])
	if verb != "cd" && verb != "chdir" {
		return "", false
	}

	rest := strings.TrimSpace(line[strings.Index(line, fields[0])+len(fields[0]):])
	if strings.HasPrefix(strings.ToLower(rest), "/d ") {
		rest = strings.TrimSpace(rest[3:])
	}

	dir, ok := unquoteWord(rest)
	if !ok || dir == "" || strings.ContainsAny(dir, "\"'`$%{}*?!;&|<>()\r\n\x00") {
		return "", false
	}

	if !isAbsolute(dir) && !isHome(dir) {
		return "", false
	}

	return dir, true
}

// unquoteWord is the single word s spells: 'quoted', "quoted", or bare with
// no whitespace.
func unquoteWord(s string) (string, bool) {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1], true
	}

	if strings.ContainsAny(s, " \t") {
		return "", false
	}

	return s, true
}

// isAbsolute accepts a POSIX root, a Windows drive root and a UNC path, on any
// platform: the hook is generated for the shell, not for the build target.
func isAbsolute(dir string) bool {
	if strings.HasPrefix(dir, "/") || strings.HasPrefix(dir, `\\`) {
		return true
	}

	return len(dir) >= 3 && isLetter(dir[0]) && dir[1] == ':' && (dir[2] == '\\' || dir[2] == '/')
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isHome is "~" alone or "~/…" / "~\…". "~user" is refused: PowerShell and
// cmd have no equivalent.
func isHome(dir string) bool {
	return dir == "~" || strings.HasPrefix(dir, "~/") || strings.HasPrefix(dir, `~\`)
}

// splitHome returns the part of a "~" path after the home folder, without its
// leading separator, and whether dir was a "~" path at all.
func splitHome(dir string) (string, bool) {
	if !isHome(dir) {
		return "", false
	}

	return strings.TrimLeft(dir[1:], `/\`), true
}

// posixWord is dir as one sh/bash/zsh word, with a leading "~" left bare so the
// shell expands it.
func posixWord(dir string) string {
	if rest, ok := splitHome(dir); ok {
		if rest == "" {
			return "~"
		}

		return "~/" + posixQuote(rest)
	}

	return posixQuote(dir)
}

// fishWord is dir as one fish word; fish expands a bare leading "~" too.
func fishWord(dir string) string {
	if rest, ok := splitHome(dir); ok {
		if rest == "" {
			return "~"
		}

		return "~/" + fishQuote(rest)
	}

	return fishQuote(dir)
}

// psPath is dir as a PowerShell expression for Set-Location -LiteralPath,
// which does not expand "~" itself.
func psPath(dir string) string {
	if rest, ok := splitHome(dir); ok {
		if rest == "" {
			return "$HOME"
		}

		return "(Join-Path $HOME '" + psQuote(rest) + "')"
	}

	return "'" + psQuote(dir) + "'"
}

// cmdPath is dir as a quoted cmd.exe argument; "~" becomes %USERPROFILE%,
// which cmd expands when the macro runs.
func cmdPath(dir string) string {
	if rest, ok := splitHome(dir); ok {
		if rest == "" {
			return `"%USERPROFILE%"`
		}

		return `"%USERPROFILE%\` + strings.ReplaceAll(rest, "/", `\`) + `"`
	}

	return `"` + dir + `"`
}

func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}
