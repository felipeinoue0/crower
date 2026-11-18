package listcontent

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fiwon123/crower/internal/domain/catalog"
	"github.com/fiwon123/crower/internal/domain/operation"
	"github.com/fiwon123/crower/internal/shell"
)

// Mode is what a request reports.
type Mode int

const (
	// ModeCommands lists the managed commands. It is the default.
	ModeCommands Mode = iota

	// ModeHistory lists the recorded operations.
	ModeHistory

	// ModeProcess lists the running processes.
	ModeProcess

	// ModeFolder lists a folder.
	ModeFolder

	// ModeSystem lists the environment variables.
	ModeSystem

	// ModePath lists the PATH entries.
	ModePath
)

// Request is one listing.
type Request struct {
	Mode Mode
	Args []string

	// Group keeps only the commands of that group (ModeCommands).
	Group string

	// Sort orders the commands: "" keeps the saved order, "name", "runs"
	// (most used first) or "recent" (last run first).
	Sort string
}

// Sort orders a command listing accepts.
var Sorts = []string{"name", "runs", "recent"}

// ListedCommand is one command as the list shows it: its saved row, and how
// often and when it last ran, counted from the history.
type ListedCommand struct {
	Row     int
	Command catalog.Command
	Runs    int
	LastRun time.Time
}

// State is the command set and the history the operation is recorded in.
type State interface {
	// Commands returns the current command set.
	Commands() catalog.Catalog

	// Record appends a history entry and persists the index.
	Record(op operation.Operation, commandName, note string) error
}

// Output is the result stream and the format it is asked for.
type Output interface {
	// Table renders the managed command table.
	Table(commands catalog.Catalog) string

	// CommandList renders the command listing a person reads.
	CommandList(rows []ListedCommand) string

	// Info writes a diagnostic line.
	Info(message string, data ...any)

	// Emit writes text in text mode and payload as JSON in structured mode.
	Emit(text string, payload any) error

	// Structured reports whether the caller asked for a JSON result.
	Structured() bool
}

// History is the operation log, as text and as rows.
//
// The method names carry the History prefix because one adapter implements the
// state and output ports too, and their Table already renders the command table.
type History interface {
	// HistoryTable renders the log newest first.
	HistoryTable() string

	// HistoryRows returns the same entries newest first, for a structured
	// result.
	HistoryRows() []HistoryRow
}

// Processes lists the running processes.
type Processes interface {
	List(partName string) ([]ProcessRow, error)
}

// Files lists a folder.
type Files interface {
	// FolderText runs the platform's listing command.
	FolderText(path string) (string, error)

	// FolderEntries reads the directory for a structured result.
	FolderEntries(path string) ([]FolderEntry, error)
}

// Environment reads the environment the shell starts with.
type Environment interface {
	// Variables returns the raw variable listing.
	Variables() (string, error)

	// Path returns the raw PATH value.
	Path() (string, error)
}

// Notes renders the history note for each mode.
type Notes interface {
	ListCommandsNote() string
	ListProcessNote(args []string) string
	ListHistoryNote() string
	ListFolderNote(args []string) string
	ListSystemNote() string
	ListPathNote() string
}

// UseCase runs one listing.
type UseCase struct {
	state       State
	output      Output
	history     History
	processes   Processes
	files       Files
	environment Environment
	notes       Notes
}

// New builds the use case from its seven dependencies.
func New(state State, output Output, history History, processes Processes, files Files, environment Environment, notes Notes) *UseCase {
	return &UseCase{
		state:       state,
		output:      output,
		history:     history,
		processes:   processes,
		files:       files,
		environment: environment,
		notes:       notes,
	}
}

// Run writes the listing the request names and records it.
func (u *UseCase) Run(request Request) error {
	switch request.Mode {
	case ModeProcess:
		return u.listProcess(request.Args)

	case ModeHistory:
		return u.listHistory()

	case ModeFolder:
		return u.listFolder(request.Args)

	case ModeSystem:
		return u.listSystem()

	case ModePath:
		return u.listPath()

	case ModeCommands:
		return u.listCommands(request)

	default:
		return fmt.Errorf("unknown list mode %d", request.Mode)
	}
}

// listCommands reports the managed commands: all of them in their saved
// order, or those matching the search and group, in the order asked for.
func (u *UseCase) listCommands(request Request) error {
	if request.Sort != "" && !slices.Contains(Sorts, request.Sort) {
		return fmt.Errorf("unknown sort %q (use %s)", request.Sort, strings.Join(Sorts, ", "))
	}

	commands := u.state.Commands()
	query := strings.ToLower(strings.Join(request.Args, " "))
	usage := Usage(u.history.HistoryRows())

	rows := make([]ListedCommand, 0, commands.Len())

	for i, command := range commands.Commands() {
		if request.Group != "" && !strings.EqualFold(command.Group, request.Group) {
			continue
		}

		if query != "" && !Matches(command, query) {
			continue
		}

		stat := usage[command.Name]
		rows = append(rows, ListedCommand{Row: i, Command: command, Runs: stat.Runs, LastRun: stat.Last})
	}

	SortListed(rows, request.Sort)

	payload := CommandsResult{Commands: make([]CommandRow, 0, len(rows))}

	for _, row := range rows {
		command := row.Command

		aliases := command.Aliases
		if aliases == nil {
			aliases = []string{}
		}

		last := ""
		if !row.LastRun.IsZero() {
			last = row.LastRun.UTC().Format(time.RFC3339)
		}

		payload.Commands = append(payload.Commands, CommandRow{
			Row:         row.Row,
			Name:        command.Name,
			Aliases:     aliases,
			Exec:        command.Exec,
			Description: command.Description,
			Params:      command.Params,
			Icon:        command.Icon,
			Group:       command.Group,
			Shell:       command.Shell,
			Cwd:         command.Cwd,
			Env:         redactEnv(command.Env),
			Timeout:     command.Timeout,
			Runs:        row.Runs,
			LastRun:     last,
		})
	}

	switch {
	case commands.Len() == 0:
		u.output.Info(`No commands yet. Create one: crower create greet "echo hello"`)
	case len(rows) == 0:
		u.output.Info("No command matches.")
	}

	text := ""
	if len(rows) > 0 {
		text = u.output.CommandList(rows)
	}

	return u.emit(text, payload, u.notes.ListCommandsNote())
}

// Stat is how often a command ran and when it last did.
type Stat struct {
	Runs int
	Last time.Time
}

// Usage counts the execute entries in the history, per command name.
func Usage(history []HistoryRow) map[string]Stat {
	usage := map[string]Stat{}

	for _, row := range history {
		if row.Operation != string(operation.Execute) || row.Command == "" {
			continue
		}

		stat := usage[row.Command]
		stat.Runs++

		if when, err := time.Parse(time.RFC3339, row.Timestamp); err == nil && when.After(stat.Last) {
			stat.Last = when
		}

		usage[row.Command] = stat
	}

	return usage
}

// Matches reports whether a command contains query (lower case) in its name,
// an alias, its line or steps, its description or its group.
func Matches(command catalog.Command, query string) bool {
	fields := append([]string{command.Name, command.Exec, command.Description, command.Group}, command.Aliases...)
	fields = append(fields, command.Steps...)

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}

	return false
}

// SortListed orders rows in place. The sort is stable, so ties keep the saved
// order.
func SortListed(rows []ListedCommand, by string) {
	switch by {
	case "name":
		slices.SortStableFunc(rows, func(a, b ListedCommand) int {
			return strings.Compare(strings.ToLower(a.Command.Name), strings.ToLower(b.Command.Name))
		})
	case "runs":
		slices.SortStableFunc(rows, func(a, b ListedCommand) int { return b.Runs - a.Runs })
	case "recent":
		slices.SortStableFunc(rows, func(a, b ListedCommand) int { return b.LastRun.Compare(a.LastRun) })
	}
}

// redactEnv hides a literal under a secret-looking key from the JSON result.
//
// The write path refuses such a value, but an old or hand-edited config still
// loads (with a warning), and a result a script can read must not be the one
// place the literal is printed. A ${VAR} reference is the placeholder itself
// and stays readable.
func redactEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}

	redacted := make(map[string]string, len(env))

	for key, value := range env {
		if _, reference := shell.EnvReference(value); !reference && shell.LooksSecret(key) {
			redacted[key] = "${REDACTED}"
			continue
		}

		redacted[key] = value
	}

	return redacted
}

// listProcess reports the running processes, optionally filtered.
func (u *UseCase) listProcess(args []string) error {
	partName := ""
	if len(args) > 0 {
		partName = args[0]
	}

	rows, err := u.processes.List(partName)
	if err != nil {
		return err
	}

	if rows == nil {
		rows = []ProcessRow{}
	}

	return u.emit(processTable(rows), ProcessesResult{Processes: rows}, u.notes.ListProcessNote(args))
}

// listHistory reports the recorded operations.
func (u *UseCase) listHistory() error {
	rows := u.history.HistoryRows()
	if rows == nil {
		rows = []HistoryRow{}
	}

	return u.emit(u.history.HistoryTable(), HistoryResult{History: rows}, u.notes.ListHistoryNote())
}

// listFolder reports a folder.
//
// The structured result reads the directory itself: the text rendering is
// `ls -la` (or `dir`), whose columns depend on platform and locale, and parsing
// that back into fields would be guesswork. That also means the JSON form needs
// no shell arm at all, while the text form still runs the platform command.
func (u *UseCase) listFolder(args []string) error {
	path := "./"
	if len(args) > 0 {
		path = args[0]
	}

	if u.output.Structured() {
		entries, err := u.files.FolderEntries(path)
		if err != nil {
			return err
		}

		if entries == nil {
			entries = []FolderEntry{}
		}

		return u.emit("", FolderResult{Folder: path, Entries: entries}, u.notes.ListFolderNote(args))
	}

	out, err := u.files.FolderText(path)
	if err != nil {
		return err
	}

	return u.emit(out, nil, u.notes.ListFolderNote(args))
}

// listSystem reports every environment variable.
func (u *UseCase) listSystem() error {
	out, err := u.environment.Variables()
	if err != nil {
		return err
	}

	return u.emit(formatVariables(out), variablesPayload(out), u.notes.ListSystemNote())
}

// listPath reports the PATH entries.
func (u *UseCase) listPath() error {
	out, err := u.environment.Path()
	if err != nil {
		return err
	}

	return u.emit(formatVariable("PATH", out), variablePayload("PATH", out), u.notes.ListPathNote())
}

// formatVariables renders every KEY=VALUE line as a named block.
func formatVariables(out string) string {
	var builder strings.Builder

	for _, sysVar := range strings.Split(out, "\n") {
		before, after, ok := strings.Cut(sysVar, "=")
		if !ok {
			continue
		}

		builder.WriteString(formatVariable(before, after))
		builder.WriteString("\n")
	}

	return builder.String()
}

func formatVariable(name string, paths string) string {
	outBuilder := strings.Builder{}
	splitted := strings.Split(paths, string(os.PathListSeparator))

	outBuilder.WriteString(name)
	outBuilder.WriteString("\n")

	for i, path := range splitted {
		outBuilder.WriteString(strconv.Itoa(i))
		outBuilder.WriteString("- ")
		outBuilder.WriteString(path)
		outBuilder.WriteString("\n")
	}

	return outBuilder.String()
}

// processTable renders the process table `list --process` prints.
func processTable(rows []ProcessRow) string {
	const (
		titleLayout   = "%-8s %-30s %s\n"
		contentLayout = "%-8d %-30s %s\n"
	)

	var builder strings.Builder

	fmt.Fprintf(&builder, titleLayout, "PID", "Name", "Path")
	builder.WriteString("-------------------------------------------------------------\n")

	for _, row := range rows {
		fmt.Fprintf(&builder, contentLayout, row.PID, row.Name, row.Path)
	}

	return builder.String()
}

// emit writes the result in the configured format and records the operation.
//
// A failure is returned instead of printed, so the command exits non-zero; a
// result is written only once the operation succeeded.
func (u *UseCase) emit(text string, payload any, note string) error {
	if err := u.output.Emit(text, payload); err != nil {
		return err
	}

	return u.state.Record(operation.List, "", note)
}
