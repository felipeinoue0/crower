// Package shellalias reads the aliases a user already has -- in bash, zsh, fish
// and PowerShell startup files, or in the output of `alias` -- so they can be
// imported into crower in one step.
//
// It reads definitions and never runs the startup files: sourcing a .bashrc to
// ask the shell would execute everything in it.
package shellalias

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Alias is one definition found in a source.
type Alias struct {
	Name   string
	Value  string
	Source string
}

var validName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+-]*$`)

// Parse reads every alias definition in r.
//
// lenient also accepts a bare `name=value` line, which is how zsh prints its
// aliases; in a startup file that line is a variable assignment, so it is only
// accepted from piped `alias` output. A later definition of a name replaces an
// earlier one, as it does in the shell.
func Parse(r io.Reader, source string, lenient bool) ([]Alias, error) {
	var found []Alias

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		for _, alias := range parseLine(scanner.Text(), lenient) {
			alias.Source = source
			found = upsert(found, alias)
		}
	}

	return found, scanner.Err()
}

// upsert replaces an alias of the same name or appends a new one, keeping the
// first position so the order follows the file.
func upsert(list []Alias, alias Alias) []Alias {
	for i := range list {
		if list[i].Name == alias.Name {
			list[i] = alias
			return list
		}
	}

	return append(list, alias)
}

func parseLine(line string, lenient bool) []Alias {
	words := splitWords(line)
	if len(words) == 0 {
		return nil
	}

	head := strings.ToLower(words[0])

	switch {
	case head == "alias":
		return parseAlias(words[1:])
	case head == "set-alias" || head == "new-alias":
		return parsePowerShell(words[1:])
	case head == "function" && len(words) > 1:
		return parseFishFunction(words[1:])
	case lenient && strings.Contains(words[0], "=") && len(words) == 1:
		return parseAssignments(words)
	}

	return nil
}

// parseAlias handles bash/zsh `alias a='x' b=y` and fish `alias a 'x'`.
func parseAlias(words []string) []Alias {
	var rest []string

	for _, word := range words {
		if word == "--" {
			continue
		}

		// zsh -s is a suffix alias (`alias -s pdf=open`): the name is a file
		// extension, not a command.
		if word == "-s" || word == "--save" {
			return nil
		}

		if strings.HasPrefix(word, "-") && len(rest) == 0 {
			continue
		}

		rest = append(rest, word)
	}

	if len(rest) == 0 {
		return nil
	}

	if strings.Contains(rest[0], "=") {
		return parseAssignments(rest)
	}

	// fish: alias name value...
	if len(rest) < 2 {
		return nil
	}

	return keep(Alias{Name: rest[0], Value: strings.Join(rest[1:], " ")})
}

func parseAssignments(words []string) []Alias {
	var out []Alias

	for _, word := range words {
		name, value, ok := strings.Cut(word, "=")
		if !ok {
			continue
		}

		out = append(out, keep(Alias{Name: name, Value: value})...)
	}

	return out
}

// parsePowerShell handles `Set-Alias ll Get-ChildItem` and the named form
// `Set-Alias -Name ll -Value Get-ChildItem -Scope Global`.
func parsePowerShell(words []string) []Alias {
	var name, value string
	var positional []string

	for i := 0; i < len(words); i++ {
		word := words[i]

		if !strings.HasPrefix(word, "-") {
			positional = append(positional, word)
			continue
		}

		flag := strings.ToLower(strings.TrimLeft(word, "-"))
		flag, inline, hasInline := strings.Cut(flag, ":")

		next := func() string {
			if hasInline {
				return inline
			}

			if i+1 < len(words) {
				i++
				return words[i]
			}

			return ""
		}

		switch flag {
		case "name", "n":
			name = next()
		case "value", "v":
			value = next()
		case "scope", "option", "description", "s", "o", "d":
			next()
		}
	}

	if name == "" && len(positional) > 0 {
		name, positional = positional[0], positional[1:]
	}

	if value == "" && len(positional) > 0 {
		value = positional[0]
	}

	return keep(Alias{Name: name, Value: value})
}

// parseFishFunction reads the function `alias --save` writes:
//
//	function ll --wraps='ls -l' --description 'alias ll=ls -l'
func parseFishFunction(words []string) []Alias {
	for i, word := range words {
		description := ""

		switch {
		case strings.HasPrefix(word, "--description="):
			description = strings.TrimPrefix(word, "--description=")
		case (word == "--description" || word == "-d") && i+1 < len(words):
			description = words[i+1]
		default:
			continue
		}

		body, ok := strings.CutPrefix(description, "alias ")
		if !ok {
			return nil
		}

		name, value, ok := strings.Cut(body, "=")
		if !ok || name != words[0] {
			return nil
		}

		return keep(Alias{Name: name, Value: value})
	}

	return nil
}

func keep(alias Alias) []Alias {
	alias.Value = strings.TrimSpace(alias.Value)

	if !validName.MatchString(alias.Name) || alias.Value == "" {
		return nil
	}

	return []Alias{alias}
}

// splitWords splits a line into shell words: single quotes are literal, double
// quotes and a backslash escape, an unquoted # starts a comment, and an
// unquoted ; | & ends the command -- what follows is another command.
func splitWords(line string) []string {
	var (
		words   []string
		current strings.Builder
		inWord  bool
		single  bool
		double  bool
	)

	runes := []rune(line)

	flush := func() {
		if inWord {
			words = append(words, current.String())
			current.Reset()
			inWord = false
		}
	}

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		switch {
		case single:
			if r == '\'' {
				single = false
			} else {
				current.WriteRune(r)
			}
		case double:
			switch {
			case r == '"':
				double = false
			case r == '\\' && i+1 < len(runes) && strings.ContainsRune("\"\\$`", runes[i+1]):
				i++
				current.WriteRune(runes[i])
			default:
				current.WriteRune(r)
			}
		case r == '\'':
			single, inWord = true, true
		case r == '"':
			double, inWord = true, true
		case r == '\\' && i+1 < len(runes):
			i++
			current.WriteRune(runes[i])
			inWord = true
		case r == '#' && !inWord:
			flush()
			return words
		case r == ';' || r == '|' || r == '&':
			flush()
			return words
		case r == ' ' || r == '\t':
			flush()
		default:
			current.WriteRune(r)
			inWord = true
		}
	}

	flush()

	return words
}

// StartupFiles lists the files a shell reads aliases from, for every shell
// crower supports, whether or not each exists.
func StartupFiles(home string) []string {
	files := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_aliases"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".zsh_aliases"),
		filepath.Join(home, ".config", "fish", "config.fish"),
		filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	}

	for _, pattern := range []string{
		filepath.Join(home, ".config", "fish", "conf.d", "*.fish"),
		filepath.Join(home, ".config", "fish", "functions", "*.fish"),
	} {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}

	return files
}

// ReadStartupFiles parses every existing startup file. A later file's
// definition of a name replaces an earlier one. It returns the files it read.
func ReadStartupFiles(home string) ([]Alias, []string, error) {
	var (
		all  []Alias
		read []string
	)

	for _, path := range StartupFiles(home) {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, read, err
		}

		found, err := Parse(file, path, false)
		file.Close()

		if err != nil {
			return nil, read, err
		}

		read = append(read, path)

		for _, alias := range found {
			all = upsert(all, alias)
		}
	}

	return all, read, nil
}
