package cmd

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/fiwon123/crower/internal/adapters/terminal"
	"github.com/fiwon123/crower/internal/application/executecommand"
	"github.com/fiwon123/crower/internal/application/listcontent"
	"github.com/fiwon123/crower/internal/domain/catalog"
	"github.com/fiwon123/crower/internal/session"

	"github.com/spf13/cobra"
)

// placeholder finds a {1} / {} in a line, which means the command expects
// arguments even when it declares no named params.
var placeholder = regexp.MustCompile(`\{\d*\}`)

// pickOrHelp is bare `crower`: in a terminal, with commands to choose from, a
// fuzzy finder that runs the chosen one; anywhere else, the help, as before.
func pickOrHelp(cmd *cobra.Command, cfg *session.Session) error {
	commands := cfg.Catalog()

	in, inOK := streams.In.(*os.File)
	errOut, errOK := streams.Err.(*os.File)
	noInput, _ := cmd.Flags().GetBool(flagNoInput)

	if !inOK || !errOK || noInput || commands.Len() == 0 || !terminal.IsTerminal(in) || !terminal.IsTerminal(errOut) {
		return cmd.Help()
	}

	ordered := byUse(commands, listcontent.Usage(newListSession(cfg).HistoryRows()))

	items := make([]terminal.Item, len(ordered))
	for i, command := range ordered {
		items[i] = pickItem(command, hasIcons(ordered))
	}

	choice, err := terminal.Pick(in, errOut, items, 10)
	if err != nil {
		return err
	}

	if choice < 0 {
		return nil
	}

	chosen := ordered[choice]

	params, err := askParams(cfg, chosen)
	if err != nil {
		return err
	}

	line := strings.TrimSpace("crower " + chosen.Name + " " + strings.Join(params, " "))
	if terminal.ColorEnabled(errOut) {
		line = "\x1b[2m› " + line + "\x1b[0m"
	} else {
		line = "› " + line
	}

	fmt.Fprintln(streams.Err, line)

	session := newExecuteSession(cfg)

	return executecommand.New(session, session, session, session, session).Run(
		executecommand.Request{Key: chosen.Name, Params: params},
	)
}

// byUse orders the commands most used first; ties keep the saved order, so a
// new set reads like `crower list`.
func byUse(commands catalog.Catalog, usage map[string]listcontent.Stat) []catalog.Command {
	ordered := commands.Commands()

	slices.SortStableFunc(ordered, func(a, b catalog.Command) int {
		return usage[b.Name].Runs - usage[a.Name].Runs
	})

	return ordered
}

func hasIcons(commands []catalog.Command) bool {
	return slices.ContainsFunc(commands, func(c catalog.Command) bool { return c.Icon != "" })
}

// pickItem is how one command reads in the finder: its icon and name, then
// what it runs and what it is for.
func pickItem(command catalog.Command, icons bool) terminal.Item {
	title := command.Name
	if len(command.Aliases) > 0 {
		title += " (" + strings.Join(command.Aliases, ", ") + ")"
	}

	if icons {
		icon := command.Icon
		if icon == "" {
			icon = "  "
		}

		title = icon + " " + title
	}

	line := command.Exec
	if len(command.Steps) > 0 {
		line = strings.Join(command.Steps, " && ")
	}

	detail := line
	if command.Description != "" {
		detail = line + "  · " + command.Description
	}

	search := strings.Join(append([]string{command.Name, command.Group, command.Description, line}, command.Aliases...), " ")

	return terminal.Item{Title: title, Detail: detail, Search: search}
}

// askParams asks for the values a command needs: each declared param by
// name, or one line of arguments when its line has a placeholder.
func askParams(cfg *session.Session, command catalog.Command) ([]string, error) {
	if len(command.Params) > 0 {
		values := make([]string, 0, len(command.Params))

		for _, name := range command.Params {
			fmt.Fprintf(streams.Err, "  %s: ", name)

			value, err := cfg.ReadLine()
			if err != nil {
				return nil, err
			}

			values = append(values, strings.TrimSpace(value))
		}

		return values, nil
	}

	lines := append([]string{command.Exec}, command.Steps...)
	if !slices.ContainsFunc(lines, placeholder.MatchString) {
		return nil, nil
	}

	fmt.Fprint(streams.Err, "  arguments: ")

	value, err := cfg.ReadLine()
	if err != nil {
		return nil, err
	}

	return strings.Fields(value), nil
}
