package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fiwon123/crower/internal/adapters/github"
	"github.com/fiwon123/crower/internal/adapters/shellalias"
	"github.com/fiwon123/crower/internal/adapters/storage"
	"github.com/fiwon123/crower/internal/application/sharecommands"
	"github.com/fiwon123/crower/internal/domain/catalog"
	"github.com/fiwon123/crower/internal/session"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export [name]",
	Short: "Export commands to a file.",
	Long:  `Export one command, or all of them, to a TOML file for sharing.`,
	Example: `	crower export
	crower export greet
	crower export --out ./commands.toml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := newSession(cmd)
		if err != nil {
			return err
		}

		name := ""
		if len(args) > 0 {
			name = args[0]
		}

		out, err := cmd.Flags().GetString("out")
		if err != nil {
			return err
		}

		return newShareUseCase(cfg).Export(name, out)
	},
}

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import commands from a file, or your existing shell aliases.",
	Long: `Import commands from a TOML file. --overwrite replaces a command that
already exists with the same name.

--from-shell imports the aliases you already have. With no file it reads the
startup files of bash, zsh, fish and PowerShell (~/.bashrc, ~/.bash_aliases,
~/.zshrc, config.fish and fish functions, the PowerShell profiles); with a
file it reads that file; with "-" it reads the output of "alias" from stdin.
The files are read, never run. Importing twice changes nothing.

	crower import --from-shell --dry-run
	crower import --from-shell
	alias | crower import --from-shell -`,
	Example: `	crower import ./commands.toml
	crower import ./commands.toml --overwrite
	crower import --from-shell --dry-run
	alias | crower import --from-shell -`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromShell, err := cmd.Flags().GetBool("from-shell")
		if err != nil {
			return err
		}

		if len(args) == 0 && !fromShell {
			return missingArgs(cmd, "a file to import")
		}

		overwrite, err := cmd.Flags().GetBool("overwrite")
		if err != nil {
			return err
		}

		if fromShell {
			preview, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return err
			}

			aliases, err := readShellAliases(args)
			if err != nil {
				return err
			}

			cfg, err := newSession(cmd)
			if err != nil {
				return err
			}

			return newShareUseCase(cfg).ImportAliases(aliases, overwrite, preview)
		}

		cfg, err := newSession(cmd)
		if err != nil {
			return err
		}

		return newShareUseCase(cfg).Import(args[0], overwrite)
	},
}

// readShellAliases reads the aliases --from-shell imports, as commands: from
// the startup files, one file, or stdin ("-").
func readShellAliases(args []string) ([]catalog.Command, error) {
	var (
		found []shellalias.Alias
		err   error
	)

	switch {
	case len(args) == 1 && args[0] == "-":
		found, err = shellalias.Parse(streams.In, "alias output", true)
	case len(args) == 1:
		var file *os.File

		file, err = os.Open(args[0])
		if err != nil {
			return nil, err
		}
		defer file.Close()

		found, err = shellalias.Parse(file, args[0], false)
	default:
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, homeErr
		}

		var read []string

		found, read, err = shellalias.ReadStartupFiles(home)
		if len(read) == 0 && err == nil {
			fmt.Fprintln(streams.Err, "No shell startup file found. Try: alias | crower import --from-shell -")
		}

		for _, path := range read {
			fmt.Fprintln(streams.Err, "Read", path)
		}
	}

	if err != nil {
		return nil, err
	}

	commands := make([]catalog.Command, 0, len(found))

	for _, alias := range found {
		commands = append(commands, catalog.Command{
			Name:        alias.Name,
			Exec:        alias.Value,
			Description: "imported from " + displayPath(alias.Source),
			Group:       "imported",
		})
	}

	return commands, nil
}

// displayPath shortens a path under the home directory to ~/...
func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}

	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") && filepath.IsAbs(path) {
		return filepath.ToSlash(filepath.Join("~", rel))
	}

	return path
}

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share commands through a registry.",
	Long:  `List the packs a registry publishes, and pull one into the command set.`,
	Example: `	crower share list
	crower share pull tools`,
}

var shareListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the packs available in the registry.",
	Long:    `List the packs available in the registry.`,
	Example: `	crower share list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := newSession(cmd)
		if err != nil {
			return err
		}

		return newShareUseCase(cfg).List()
	},
}

var sharePullCmd = &cobra.Command{
	Use:     "pull <name>",
	Short:   "Pull a pack from the registry.",
	Long:    `Pull a pack from the registry into the command set.`,
	Example: `	crower share pull tools`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return missingArgs(cmd, "a pack name")
		}

		cfg, err := newSession(cmd)
		if err != nil {
			return err
		}

		return newShareUseCase(cfg).Pull(args[0])
	},
}

// shareSession adapts the configuration to the ports the share use case
// declares.
type shareSession struct {
	sessionPorts

	packs    *storage.Packs
	registry *github.Registry
}

// newShareSession wires the configuration, the pack files and the registry.
func newShareSession(cfg *session.Session) shareSession {
	return shareSession{
		sessionPorts: newSessionPorts(cfg),
		packs:        storage.NewPacks(),
		registry:     github.NewRegistry(),
	}
}

// Write stores a pack at path.
func (s shareSession) Write(path string, commands []catalog.Command) error {
	return s.packs.Write(path, commands)
}

// Read loads the pack at path.
func (s shareSession) Read(path string) ([]catalog.Command, error) {
	return s.packs.Read(path)
}

// Index returns the registry's packs.
func (s shareSession) Index() ([]sharecommands.Pack, error) {
	return s.registry.Index()
}

// Pack fetches one registry pack.
func (s shareSession) Pack(name string) ([]catalog.Command, string, error) {
	return s.registry.Pack(name)
}

// newShareUseCase builds the use case over the configuration.
func newShareUseCase(cfg *session.Session) *sharecommands.UseCase {
	session := newShareSession(cfg)

	return sharecommands.New(session, session, session, session, session)
}

func init() {
	exportCmd.Flags().StringP("out", "o", "", "Output file path")
	importCmd.Flags().Bool("overwrite", false, "Overwrite existing commands")
	importCmd.Flags().Bool("from-shell", false, "import your existing shell aliases")
	importCmd.Flags().BoolP("dry-run", "n", false, "with --from-shell, show what would be imported")

	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(shareCmd)
	shareCmd.AddCommand(shareListCmd)
	shareCmd.AddCommand(sharePullCmd)
}
