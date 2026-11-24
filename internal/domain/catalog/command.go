package catalog

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Command is one managed command: the name it is invoked by, the aliases it
// also answers to, the line it runs, and the lines it runs instead when it is a
// Steps command.
type Command struct {
	Name    string
	Aliases []string
	Exec    string
	Steps   []string

	// Description and Params are metadata: what the command is for and what
	// its positional values mean. They never change how it runs.
	Description string
	Params      []string

	// Icon is a short marker -- an emoji -- shown beside the command so the
	// list is recognisable at a glance, and Group names the set it belongs
	// to. Both are presentation: they change no behaviour, and a file that
	// lacks them loads as empty.
	Icon  string
	Group string

	// Shell names the shell the command runs through ("sh", "pwsh", "cmd"),
	// and Cwd is the folder it runs in. Empty Shell keeps the platform's own
	// shell; empty Cwd keeps the caller's working directory. Every step of a
	// Steps command inherits both.
	Shell string
	Cwd   string

	// Env is merged over the process environment, and Timeout is the per-step
	// limit ("120s"). Empty Env adds nothing; empty Timeout means no limit.
	// Every step inherits both.
	Env     map[string]string
	Timeout string

	// Here asks the shell hook to run the exec line in the calling shell
	// rather than in a child process, so a cd, an export or a `source` stays
	// in effect. Without the hook the command runs as a subprocess like any
	// other. See CheckHere for what it cannot be combined with.
	Here bool
}

// ErrHereConflict is returned for an in-shell command that also declares
// something only a child process can honour.
var ErrHereConflict = errors.New("--here cannot be combined with")

// CheckHere reports why the command cannot run in the calling shell, or nil.
//
// Steps, Shell, Cwd, Env and Timeout all describe the child process crower
// starts; the calling shell has none of them to apply, so silently dropping
// them would run a different command than the one stored.
func (c Command) CheckHere() error {
	if !c.Here {
		return nil
	}

	var conflicts []string

	if len(c.Steps) > 0 {
		conflicts = append(conflicts, "steps")
	}

	if c.Shell != "" {
		conflicts = append(conflicts, "--shell")
	}

	if c.Cwd != "" {
		conflicts = append(conflicts, "--cwd")
	}

	if len(c.Env) > 0 {
		conflicts = append(conflicts, "env")
	}

	if c.Timeout != "" {
		conflicts = append(conflicts, "--timeout")
	}

	if len(conflicts) == 0 {
		return nil
	}

	return fmt.Errorf("%w %s (they need a child process)", ErrHereConflict, strings.Join(conflicts, ", "))
}

// clone returns a command whose slices do not share storage with the catalog.
func (c Command) clone() Command {
	c.Aliases = slices.Clone(c.Aliases)
	c.Steps = slices.Clone(c.Steps)
	c.Params = slices.Clone(c.Params)
	c.Env = maps.Clone(c.Env)
	return c
}
