package cmd

import (
	"github.com/felipeinoue0/crower/internal/adapters/notes"
	"github.com/felipeinoue0/crower/internal/adapters/system"
	"github.com/felipeinoue0/crower/internal/application/executecommand"
	"github.com/felipeinoue0/crower/internal/domain/catalog"
	"github.com/felipeinoue0/crower/internal/domain/operation"
	"github.com/felipeinoue0/crower/internal/session"

	"github.com/spf13/cobra"
)

var last bool

// dryRunFlag prints what a run would execute, and runs nothing.
var dryRunFlag bool

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute <command> [params...]",
	Short: "Run a managed command.",
	Long: `Run a managed command, by name or by alias.

	crower execute greet
	crower greet

--dry-run (-n) prints the expanded line and runs nothing:
	crower execute greet Ann --dry-run

The last touched command can be run instead:
	crower execute --last
	crower execute --create
	crower execute --update

Arguments after the command are applied to its steps.`,
	Example: `	crower execute greet
	crower execute greet --help
	crower execute greet Ann --dry-run
	crower execute --last`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := newSession(cmd)
		if err != nil {
			return err
		}

		request := executecommand.Request{DryRun: dryRunFlag}

		// The flags decide which command to run; the use case owns how.
		switch {
		case last:
			request.Last, request.Params = operation.Execute, args
		case createFlag:
			request.Last, request.Params = operation.Create, args
		case updateFlag:
			request.Last, request.Params = operation.Update, args
		case len(args) > 0:
			request.Key, request.Params = args[0], args[1:]
		case dryRunFlag:
			return missingArgs(cmd, "a command to show")
		}

		session := newExecuteSession(cfg)

		return executecommand.New(session, session, session, session, session).Run(request)
	},
}

// executeSession adapts the configuration to the ports the execute use case
// declares.
//
// It embeds sessionPorts for the catalog, the diagnostics stream, the result
// stream, the debug log, the confirmation and the history entry, and adds the
// shell runner and the two execution notes.
type executeSession struct {
	sessionPorts

	runner *system.Runner
}

// newExecuteSession wires the configuration and a runner attached to the
// terminal: the CLI runs a command the way the shell would, so ssh, vim and a
// prompt work and a long build shows its output as it goes.
func newExecuteSession(cfg *session.Session) executeSession {
	return executeSession{
		sessionPorts: newSessionPorts(cfg),
		runner:       system.NewLiveRunner(system.Streams{In: streams.In, Out: streams.Out, Err: streams.Err}),
	}
}

// newCapturingExecuteSession wires a runner that captures the output, for an
// interface that displays the output itself.
func newCapturingExecuteSession(cfg *session.Session) executeSession {
	return executeSession{
		sessionPorts: newSessionPorts(cfg),
		runner:       system.NewRunner(),
	}
}

// Run runs one shell line in the command's target.
func (s executeSession) Run(line string, target executecommand.Target) (string, error) {
	return s.runner.Run(line, system.Target{
		Shell:   target.Shell,
		Dir:     target.Dir,
		Env:     target.Env,
		Timeout: target.Timeout,
	})
}

// RunNested runs a line that invokes crower again, with the depth guard.
func (s executeSession) RunNested(line string, target executecommand.Target) (string, error) {
	return s.runner.RunNested(line, system.Target{
		Shell:   target.Shell,
		Dir:     target.Dir,
		Env:     target.Env,
		Timeout: target.Timeout,
	})
}

// ExecuteCommandNote is the history note for `execute <command>`.
func (s executeSession) ExecuteCommandNote(command catalog.Command, params []string) string {
	return notes.ExecuteCommandNote(command, params)
}

// ExecuteLastNote is the history note for the --last forms.
func (s executeSession) ExecuteLastNote(op operation.Operation, command catalog.Command, args []string) string {
	return notes.ExecuteLastNote(op, command, args)
}

func init() {
	executeCmd.Flags().BoolVarP(&last, "last", "l", false, "execute recent executed command")
	executeCmd.Flags().BoolVarP(&createFlag, "create", "c", false, "execute recent created command")
	executeCmd.Flags().BoolVarP(&updateFlag, "update", "u", false, "execute recent updated command")
	executeCmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "print what would run, and run nothing")
}
