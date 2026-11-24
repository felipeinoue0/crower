<div align="center">

# 📖 Command reference

**Every command, every flag, and the things a script can rely on.**

</div>

`crower <command> --help` prints the same information for one command. This page
is the version you can read without a terminal.

---

## 🧭 The contract

Four rules hold for every command. A script may depend on all of them.

| | |
| --- | --- |
| **Results** go to **stdout** | the table of `list`, the rows of `search` |
| **Diagnostics** go to **stderr** | prompts, status lines, warnings, errors |
| **Exit 0** | the work happened — or the user declined, and nothing was the answer |
| **Exit 1** | the work did not happen |

```bash
# only the table reaches the file
crower list > commands.txt

# a failure is visible to the script
if ! crower execute deploy; then echo "deploy failed" >&2; fi
```

A prompt with **no input available** — a closed stdin, or `--no-input` — is an
error, not an empty answer. The run fails instead of waiting.

## ⚙️ Global flags

These work on every command.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--config <path>` | | configuration file (default `$HOME/crower/crower.yaml`) |
| `--no-input` | | never prompt; fail instead of waiting for a person |
| `--version` | `-v` | print the version |
| `--upgrade` | | upgrade to the latest release |
| `--check` | | check whether a newer release exists |

`--config` and `--no-input` are persistent: every subcommand inherits them.

`--setup <name>` selects a named command setup (`~/crower/setups/<name>/`).
It applies when `--config` is not given; `CROWER_SETUP` sets the same default.
The default installation keeps its old path and history.

## 🤖 Machine-readable results

`list` and `search` accept `--json`: the result becomes **one JSON object** on
stdout, diagnostics stay on stderr, and the exit status is unchanged.

```bash
crower list --json | jq -r '.commands[].name'
```

| Command | Object |
| --- | --- |
| `crower list` | `{"commands":[{"row":0,"name":"gs","aliases":["g"],"exec":"git status"}]}` |
| `crower list --history` | `{"history":[{"version":1,"file":"00001.yaml","timestamp":"2026-01-02T15:04:05Z","operation":"create","command":"gs","note":"…"}]}` |
| `crower list --folder <dir>` | `{"entries":[{"name":"a.txt","dir":false,"size":12,"mode":"-rw-r--r--","modified":"…"}]}` |
| `crower list --process` | `{"processes":[…]}` |
| `crower list --system` | `{"variables":[{"name":"PATH","values":["…"]}]}` |
| `crower list --syspath` | `{"variable":{"name":"PATH","values":["…"]}}` |
| `crower search` | `{"search":{"kind":"file","query":"main","path":".","results":["…"]}}` |

Any other command rejects `--json` as an unknown flag: a global switch a command
silently ignored would report success without honouring it.

---

`crower list --json` rows carry `description`, `params`, `shell`, `cwd`, `env` and `timeout` when the command declares them. The first two are metadata; the rest say how it runs.

## ➕ crower create

```text
crower create [name] [exec]
```

Create a command, a file or a folder. With no arguments, crower asks for each
field.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--alias <a,b>` | `-a` | add aliases (comma-separated, repeatable) |
| `--icon <marker>` | | a short marker (an emoji) shown beside the command in the window |
| `--group <name>` | | the group the command belongs to |
| `--description <text>` | | what the command is for |
| `--params <a,b>` | | positional parameters (`env,region`) |
| `--cwd <dir>` | | working directory the command runs in |
| `--shell <name>` | | shell that runs the command (`sh`, `pwsh`, `cmd`) |
| `--timeout <dur>` | | longest the command may run (`120s`) |
| `--here` | | run in your own shell through the shell hook (`cd`, `export`, `source` stay in effect) |
| `--process <name>` | `-p` | create the command from a running process; the argument is its name or pid |
| `--file` | `-f` | treat every argument as a file path to create |
| `--folder` | `-o` | treat every argument as a folder path to create |
| `--script` | | create a script file that the command runs |
| `--system` | | create a system variable |
| `--syspath` | | add a directory to the system `PATH` |
| `--execute` | | create a command from the last command you ran |

```bash
crower create greet "echo hello"
crower create greet "echo hello" --alias hi
crower create --file notes.txt
crower create --folder reports
crower create -p server nginx
```

A command runs through `sh -c` on Linux and macOS, and `cmd /c` on Windows,
unless it declares its own `Shell`. A Steps command runs its steps instead of
its `exec`; `execute` refuses to run one with parameters it cannot apply.

## ✏️ crower update

```text
crower update [command]
```

Change a command's name, aliases, exec or execution metadata. The command is
found by name or alias.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--name <name>` | `-n` | new name |
| `--alias <a,b>` | `-a` | new alias list, replacing the old one |
| `--exec <line>` | `-e` | new command line |
| `--icon <marker>` | | new icon; an empty value removes it |
| `--group <name>` | | new group; an empty value removes it |
| `--description <text>` | | what the command is for; empty removes it |
| `--params <a,b>` | | positional parameters; empty removes them |
| `--cwd <dir>` | | working directory; empty removes it |
| `--shell <name>` | | shell that runs the command; empty removes it |
| `--timeout <dur>` | | longest the command may run; empty removes it |
| `--here` | | run in your own shell through the shell hook; `--here=false` turns it off |
| `--last` | `-l` | update the last executed command |
| `--create` | `-c` | update the last created command |
| `--execute` | `-x` | update the last updated command |

```bash
crower update greet --exec "echo hi"
crower update greet --name hello --alias hi
crower update --last
```

With no field given, crower asks for each one. A Steps command keeps its steps:
`update` has no flag for them, so it never replaces them.

## 🗑️ crower delete

```text
crower delete [command]
```

Delete a command, a file or a folder.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--create` | `-c` | delete the last created command |
| `--update` | `-u` | delete the last updated command |
| `--execute` | `-x` | delete the last executed command |
| `--history` | | delete one history entry and its snapshot |
| `--file` | `-f` | treat every argument as a file path |
| `--folder` | `-o` | treat every argument as a folder path |
| `--system` | | delete a system variable |
| `--syspath` | | remove a directory from the system `PATH` |

```bash
crower delete greet
crower delete hi
crower delete --file notes.txt
crower delete --create
```

With no arguments crower asks which command to delete. Deleting is recorded in
the history, so `revert` can bring the command back.

## 📋 crower list

```text
crower list [search] [flags]
```

| Flag | Short | Meaning |
| --- | --- | --- |
| `--group <name>` | `-g` | only the commands of that group (case-insensitive) |
| `--sort <order>` | `-s` | `name`, `runs` (most used first) or `recent` (last run first) |
| `--process` | `-p` | list running processes |
| `--history` | `-i` | list the history, newest first |
| `--folder <dir>` | `-o` | list the files in a folder |
| `--system` | | list system variables |
| `--syspath` | | list the system `PATH` entries |
| `--json` | | one JSON object on stdout instead of a table |

```bash
crower list
crower list docker
crower list --group work --sort runs
crower list --history
crower list --folder ~/projects
crower list --json
```

With no flag, crower lists the managed commands in the order they were saved:
`#` (the saved row), icon and name, aliases, group, how many times it ran and
when it last ran (counted from the history), and the line it runs with its
description. A search keeps the commands whose name, alias, line, description
or group contains it. Columns no command uses are left out.

On a terminal the list is coloured and cut to the window's width — the optional
columns give way before the command does. Piped, with `NO_COLOR`, or with
`TERM=dumb`, it is plain and every line is whole. `--json` adds `runs` and
`last_run` to each command.

**Bare `crower` / `cr`** in an interactive terminal opens a fuzzy finder over the
same commands, most used first: type to filter, `↑`/`↓` (or `Ctrl+P`/`Ctrl+N`,
`Tab`) to move, `Enter` to run, `Esc` or `Ctrl+C` to leave. A command with
declared `Params` asks for each; one whose line has a placeholder asks for its
arguments. Without a terminal, with `--no-input`, or with no commands, bare
`crower` prints the help.

## ▶️ crower execute

```text
crower execute <command> [params...]
```

Run a managed command by name or alias. `crower <name>` is the same thing.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--last` | `-l` | run the last executed command |
| `--create` | `-c` | run the last created command |
| `--update` | `-u` | run the last updated command |
| `--dry-run` | `-n` | print the expanded line(s) and the target; run and record nothing |

```bash
crower execute greet
crower greet
crower greet Ada --dry-run
crower execute --last
```

The command runs **attached to your terminal**: it reads your keyboard, writes
as it goes and keeps its colours, so `ssh`, `vim`, `htop` and prompts work
through an alias. Ctrl+C goes to the command, not to crower, which waits for
it and reports how it ended. A command with a `Timeout` runs in its own process group, gets no
stdin, and Ctrl+C cancels it. (`crower gui` and the window capture the output
and show it instead.)

`--dry-run` writes the expanded steps to stdout — `crower x --dry-run | sh` is a
legitimate use — and the shell, folder, env keys and timeout to stderr.

Parameters after the command fill `{1}`, `{2}`, … in its steps; `{}` takes the
rest. Each parameter is quoted before it reaches the shell.

A command may declare the shell it needs and the folder it runs in. `Shell` is
`sh`, `pwsh` or `cmd` (the named shell must be installed); empty keeps the
platform's own shell (`sh` on Linux and macOS, `cmd` on Windows). `Cwd` is the
working directory, and a leading `~` is the user's home. Every step of a Steps
command inherits both. An unknown shell, or a folder that does not exist, fails
before anything runs. Set them with `create`/`update` (`--shell`, `--cwd`,
`--timeout`), in the configuration file, or in a pack; `Steps` and `Env` are
set in the file or a pack.

`Env` merges entries over the process environment. A value that is exactly
`${VAR}` reads `VAR` from the environment, and an unset variable is an error
rather than an empty value. A literal under a secret-looking key (`*TOKEN*`,
`*SECRET*`, `*PASSWORD*`, `*_KEY*`, …) is refused on export, import, save and
execute: a pack carries placeholders, never secrets. A hand-edited config with
one still loads, with a warning, and `list --json` redacts it. `Timeout` is a Go
duration (`120s`, `2m`) applied to **each** step; an overrun kills the step and
the error names the limit.

## 📂 crower open

```text
crower open [path...]
```

| Flag | Short | Meaning |
| --- | --- | --- |
| `--file` | `-f` | open a file with the system opener |
| `--folder` | `-o` | open a folder with the system opener |
| `--system` | `-s` | open the system settings |

```bash
crower open                 # the configuration file
crower open --file notes.txt
crower open --folder .
```

## 📄 crower copy

```text
crower copy <source...> <destination>
```

Copy files or folders. The last argument is the destination.

```bash
crower copy notes.txt ~/backup
crower copy a.txt b.txt ~/backup
crower copy ./reports ~/backup
```

## 🚚 crower move

```text
crower move <source...> <destination>
```

Move files or folders. The last argument is the destination.

```bash
crower move notes.txt ~/archive
crower move ./reports ~/archive
```

## 🔍 crower search

```text
crower search <term> [path]
```

| Flag | Short | Meaning |
| --- | --- | --- |
| `--file` | `-f` | search for a file name |
| `--folder` | `-o` | search for a folder name |
| `--browser` | `-b` | search the web in the default browser |
| `--json` | | one JSON object on stdout instead of a table |

```bash
crower search report ./notes
crower search --folder projects ~
crower search --browser "golang context"
```

## 🗜️ crower extract

```text
crower extract <archive...>
```

Unpack `.zip`, `.tar`, `.tar.gz`, `.7z`, `.rar` and friends.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--out <dir>` | `-o` | extract into this folder instead of the archive's own |

```bash
crower extract files.zip
crower extract files.zip --out ./build
crower extract "*.zip"
```

## ↩️ crower revert

```text
crower revert <steps>
```

Undo operations, newest first. The argument is how many operations to roll back:
`revert 1` undoes the last one, `revert 3` undoes the last three.

```bash
crower revert 1
crower revert 3
```

`crower list --history` shows what is there. Reverting writes the restored
command map back and drops the entries it passed. A snapshot records the state
*after* its operation, so undoing one operation restores the state before it —
which is why `revert 1` after a delete brings the command back.

## ♻️ crower restore

```text
crower restore <command>
```

Bring back a command from the snapshot of a past operation, by name or alias.
This is how a deleted or overwritten command comes back without undoing
everything after it.

```bash
crower restore greet
```

## 🔄 crower reset

```text
crower reset
```

Delete every managed command. The table is shown first and nothing happens
unless you confirm. History snapshots are left alone: `crower list --history`
still works and `crower revert` can bring the commands back.

```bash
crower reset
```

## 📤 crower export

```text
crower export [name]
```

Export one command, or all of them, to a TOML file for sharing.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--out <path>` | `-o` | write here instead of the default file |

```bash
crower export
crower export greet
crower export --out ./commands.toml
```

## 📥 crower import

```text
crower import [file] [flags]
```

Import commands from a TOML file, or the aliases your shell already has.

| Flag | Short | Meaning |
| --- | --- | --- |
| `--overwrite` | | replace a command that already exists with the same name |
| `--from-shell` | | import shell aliases: from the startup files, a file, or `-` (stdin) |
| `--dry-run` | `-n` | with `--from-shell`, list what would be imported and write nothing |

```bash
crower import ./commands.toml
crower import ./commands.toml --overwrite
crower import --from-shell --dry-run
crower import --from-shell
crower import --from-shell ~/.bash_aliases
alias | crower import --from-shell -
```

`--from-shell` with no file **reads, never runs**: `~/.bashrc`,
`~/.bash_aliases`, `~/.bash_profile`, `~/.profile`, `~/.zshrc`,
`~/.zsh_aliases`, fish's `config.fish`, `conf.d/*.fish` and the functions
`alias --save` writes, and the PowerShell profiles (`Set-Alias`, `New-Alias`).
From stdin it also accepts zsh's bare `name=value` output. An alias crower
already has with the same line is skipped, so importing twice changes nothing;
imported commands get the group `imported` and a description naming the file.

Names and aliases share one namespace: an import that would collide is renamed
rather than silently shadowing a command.

## 🤝 crower share

List and pull command packs from a registry.

### crower share list

List the packs the registry publishes.

```bash
crower share list
```

### crower share pull

Pull a pack into the command set.

```bash
crower share pull tools
```

## 🎛️ crower gui

Open the interactive terminal UI.

```bash
crower gui
```

## 🖼️ crower-gui

Every platform archive contains `crower-gui`, the Fyne desktop application. It
is not a `crower` subcommand; extract the archive and launch it. The CLI
binaries are cgo-free and need no display.

```bash
crower-gui                   # open the window
crower-gui --config FILE     # another configuration file
crower-gui --setup NAME      # a named setup
```

It opens the window whatever the arguments are: an unknown argument is ignored,
because a window has no subcommands and a double-click must never reach a command
parser. `--help` prints the three lines above and exits.

---

## 🗂️ crower setup

```bash
crower setup list                 # the default setup and every named one
crower setup new streamer         # create ~/crower/setups/streamer/
crower setup use streamer         # make it the default for future runs
crower setup show                 # active setup, config path, command count
crower setup delete gaming --force
```

Each setup keeps its own commands and its own history. The default setup keeps
`~/crower/crower.yaml`; nothing migrates.

### crower setup list

Prints `default` and every folder under `~/crower/setups/`, marking the active
one.

### crower setup new

Creates an empty named setup. The name must be a slug (letters, digits, `-`, `_`,
`.`).

### crower setup use

Persists the default setup for future runs (the same as `--setup` and
`CROWER_SETUP`). The setup must exist.

### crower setup show

Prints the active setup, its configuration path and how many commands it holds.

### crower setup delete

Removes a named setup and its history. The active setup is protected unless
`--force` is passed; the default setup is not a folder and is never deleted here.

## 🩺 crower doctor

```bash
crower doctor
crower doctor --json
```

Shows the configuration file and folder, whether the folder is writable, how
many commands and history entries exist, the update-check mode, the detected
shell and editor, whether the diagnostics stream is a terminal, and the
platform. `--json` prints the same as one object.

It reads only: nothing is created beyond the config folder the run needs.

## 🚀 crower init

```text
crower init [bash|zsh|fish|powershell|cmd] [flags]
```

```bash
crower init                 # show what your shell needs
crower init --apply         # write it into the startup file
eval "$(crower init zsh)"   # the hook itself, for a startup file
```

With no shell, prints the PATH line and the hook line for the detected shell
and where they go; `--apply` appends both once (a second run is a no-op), the
PowerShell profile included. With a shell, prints the **hook**: one alias per
command name and alias, a re-sync after `create`/`update`/`delete`/`import`/
`revert`/`restore`/`reset`/`setup`, and tab completion. A name the shell
already knows is skipped. `cmd` prints doskey macros. See
[🐚 Shell integration](Shell-Integration).

## 🧾 Exit codes

| Status | Meaning |
| --- | --- |
| `0` | the work happened, help was printed, or the user declined a confirmation |
| `1` | the command failed; the reason is one line on stderr |

Nothing else is used: a script can test the status without decoding it.

## 🔤 A note on shorthands

`-o` means `--folder` on `create`, `delete`, `list`, `search` and `open`, and
`--out` on `extract` and `export`. It is the one shorthand that changes meaning
between commands; write the long flag in scripts.
