package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The Makefile declares its targets in a .PHONY list. When a target is listed
// there but its recipe is deleted, `make` treats it as an up-to-date target
// with no commands: it prints "Nothing to be done" and exits 0. That happened
// to `check` (and the whole dev-sandbox layer) in issue #90, and it meant a
// gate that reported success while running nothing.
//
// These tests are the guard: every declared target must have a rule, and the
// gate must still run gofmt, vet, test and build.

// phonyTargets returns the target names declared in the file's .PHONY list,
// following backslash continuations.
func phonyTargets(text string) []string {
	lines := strings.Split(text, "\n")
	var names []string

	for i, line := range lines {
		if !strings.HasPrefix(line, ".PHONY:") {
			continue
		}

		// The declaration may span several lines; keep consuming while the
		// current line ends with a continuation.
		joined := strings.TrimSuffix(line, "\\")
		for j := i; strings.HasSuffix(strings.TrimRight(lines[j], " \t"), "\\"); j++ {
			joined += " " + strings.TrimSuffix(strings.TrimRight(lines[j+1], " \t"), "\\")
		}

		joined = strings.TrimPrefix(joined, ".PHONY:")
		names = append(names, strings.Fields(joined)...)
	}

	return names
}

// missingRules returns the declared targets that have no rule in the file.
func missingRules(text string) []string {
	var missing []string

	for _, name := range phonyTargets(text) {
		// A rule starts at column 0 with "name:" — a recipe line is indented,
		// so it cannot match.
		if !regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `:`).MatchString(text) {
			missing = append(missing, name)
		}
	}

	return missing
}

func TestMakefileDeclaredTargetsHaveRules(t *testing.T) {
	data, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}

	if missing := missingRules(string(data)); len(missing) > 0 {
		t.Fatalf("declared in .PHONY but has no recipe (make would silently do nothing): %s",
			strings.Join(missing, ", "))
	}
}

func TestMakefileCheckTargetRunsTheGate(t *testing.T) {
	data, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}

	text := string(data)
	if missing := missingRules(text); contains(missing, "check") {
		t.Fatalf("the check target has no recipe — `make check` would report success while doing nothing")
	}

	// All four gate steps must still be present, in order.
	checkBody := ruleBody(text, "check")
	for _, want := range []string{"gofmt -l", "go vet ./...", "go test ./...", "go build ./..."} {
		if !strings.Contains(checkBody, want) {
			t.Errorf("check recipe is missing %q; the gate no longer covers it", want)
		}
	}
}

// TestMakefileGuiLinuxTargetBuildsTheTagger guards the one-command GUI build.
//
// `make gui_linux` only works because of four things that a bare `go build`
// does not carry: the fyne tag (without it the package's files are all
// excluded and the command is not even there), CGO (Fyne's Linux driver is
// cgo), the ldflags that stamp the version, and .PHONY (a file named
// `gui_linux` in the tree would otherwise make make report success and build
// nothing — the issue #90 failure mode).
func TestMakefileGuiLinuxTargetBuildsTheTagger(t *testing.T) {
	text := readRepositoryFile(t, "Makefile")

	if missing := missingRules(text); contains(missing, "gui_linux") {
		t.Fatalf("gui_linux has no recipe — `make gui_linux` would print \"Nothing to be done\" and exit 0")
	}

	if !contains(phonyTargets(text), "gui_linux") {
		t.Errorf("gui_linux is not declared in .PHONY; a file of that name would silently skip the build")
	}

	guiBody := ruleBody(text, "gui_linux")
	buildLine := guiBuildCommand(t, guiBody)
	for _, want := range []string{"-tags fyne", "CGO_ENABLED=1", "./cmd/crowergui", "$(GUI_LDFLAGS)"} {
		if !strings.Contains(buildLine, want) {
			t.Errorf("gui_linux's build command is missing %q (line: %s); the target no longer builds the tagged, version-stamped window", want, strings.TrimSpace(buildLine))
		}
	}
}

// guiBuildCommand returns the recipe line that actually runs the compiler.
//
// Asserting on the whole recipe body is not enough: the recipe also prints a
// build command in its non-Linux error message, so a body-wide `strings.Contains`
// stays green after the real line loses its tag — the guard would pass while
// the build it guards is broken. Only a non-echo line can be the command.
func guiBuildCommand(t *testing.T, body string) string {
	t.Helper()

	for _, line := range strings.Split(body, "\n") {
		command := strings.TrimPrefix(strings.TrimSpace(line), "@")
		if strings.HasPrefix(command, "echo") || strings.HasPrefix(command, "test ") {
			continue
		}
		if strings.Contains(command, "go build") {
			return command
		}
	}

	t.Fatalf("gui_linux has no `go build` line; the recipe does not build anything:\n%s", body)

	return ""
}

// TestMakefileGuiTargetsShareOneDepsSource keeps the package list in one place.
//
// fyne.yml and release.yml consume `make -s gui-deps`, which now resolves the
// list per distro; if gui_linux grew its own copy of the header check the two
// would drift, and the drift would show up as a build that passes the precheck
// and fails at the linker.
func TestMakefileGuiTargetsShareOneDepsSource(t *testing.T) {
	text := readRepositoryFile(t, "Makefile")

	for _, target := range []string{"gui-deps", "gui_linux"} {
		if !strings.Contains(ruleBody(text, target), "$(GUI_DEPS_SCRIPT)") {
			t.Errorf("%s does not call $(GUI_DEPS_SCRIPT); the distro logic would be duplicated", target)
		}
	}

	// The apt list moving back into the Makefile is the regression this
	// guards: two workflows install it, and only one distro would be right.
	if regexp.MustCompile(`(?m)^GUI_DEPS\s*:?=`).MatchString(text) {
		t.Errorf("the package list is back in the Makefile; it lives in scripts/gui-deps.sh so every distro resolves")
	}
}

// ruleBody returns the recipe text of the named target, up to the next
// top-level construct.
//
// A recipe may sit inside a make conditional (`gui-deps` is chosen with
// `ifeq ($(INSTALL),1)`), and those directives are themselves unindented — so
// they continue the body rather than ending it. Without this, a target whose
// only recipe is conditional reads as having no recipe at all.
func ruleBody(text, target string) string {
	lines := strings.Split(text, "\n")
	start := -1

	for i, line := range lines {
		if strings.HasPrefix(line, target+":") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}

	var body []string
	for _, line := range lines[start:] {
		if line != "" && !strings.HasPrefix(line, "\t") && !isConditional(line) {
			break
		}
		body = append(body, line)
	}

	return strings.Join(body, "\n")
}

// isConditional reports whether a line is a make conditional directive rather
// than the start of the next rule.
func isConditional(line string) bool {
	for _, directive := range []string{"ifeq", "ifneq", "ifdef", "ifndef", "else", "endif"} {
		if line == directive || strings.HasPrefix(line, directive+" ") ||
			strings.HasPrefix(line, directive+"(") {
			return true
		}
	}

	return false
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}

	return false
}

// TestMissingRulesCatchesTheRegression proves the guard itself can fail: it is
// given the exact shape that broke in issue #90 — a target declared in .PHONY
// with its recipe removed — and must report it.
func TestMissingRulesCatchesTheRegression(t *testing.T) {
	broken := strings.Join([]string{
		".PHONY: clean release check \\",
		"\tdev-up dev-shell",
		"",
		"clean:",
		"\trm -rf build",
		"",
		"release: clean",
		"\t@echo done",
		"",
	}, "\n")

	if got := missingRules(broken); !contains(got, "check") {
		t.Errorf("missingRules() = %v, want it to include \"check\"", got)
	}
	if got := missingRules(broken); !contains(got, "dev-up") {
		t.Errorf("missingRules() = %v, want it to include \"dev-up\"", got)
	}
	if got := missingRules(broken); contains(got, "clean") || contains(got, "release") {
		t.Errorf("missingRules() = %v, want defined targets excluded", got)
	}
}
