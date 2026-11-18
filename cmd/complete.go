package cmd

import (
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// completeCommandNames offers the managed command names and aliases for the
// first positional argument, so `cr dep<Tab>` and `crower delete dep<Tab>`
// finish the word. Only the first argument is a name; parameters after it are
// the command's own and complete as files.
func completeCommandNames(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	cfg, err := newSession(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	commands := cfg.Catalog()

	var out []cobra.Completion

	for _, word := range commandWords(commands) {
		if !strings.HasPrefix(word, toComplete) {
			continue
		}

		// A description makes zsh, fish and PowerShell show what the name
		// runs; bash drops it.
		description := ""
		name := word
		if owner, ok := commands.AliasOwner(word); ok {
			name = owner
		}

		if command, ok := commands.Get(name); ok {
			description = command.Description
			if description == "" {
				description = command.Exec
				if description == "" && len(command.Steps) > 0 {
					description = strings.Join(command.Steps, " && ")
				}
			}
		}

		out = append(out, cobra.CompletionWithDesc(word, oneLine(description)))
	}

	slices.Sort(out)

	return out, cobra.ShellCompDirectiveNoFileComp
}

// oneLine trims a description to one short line for a completion menu.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")

	const max = 60
	if len([]rune(s)) > max {
		s = string([]rune(s)[:max-1]) + "…"
	}

	return s
}

func init() {
	for _, c := range []*cobra.Command{rootCmd, executeCmd, updateCmd, deleteCmd, exportCmd} {
		c.ValidArgsFunction = completeCommandNames
	}
}
