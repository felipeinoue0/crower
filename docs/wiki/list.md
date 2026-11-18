# 📋 `list`

**List commands (the default), history, processes, files, or variables.**

```bash
crower list [search] [flags]
```

## 🚩 Flags

| Flag | Effect |
| --- | --- |
| `-g`, `--group <name>` | Only that group's commands |
| `-s`, `--sort <order>` | `name`, `runs` (most used first) or `recent` |
| `-i`, `--history` | Operation history, newest first |
| `-p`, `--process` | Running processes |
| `-o`, `--folder` | Files in a folder (pass the path) |
| `--system` | System variables |
| `--syspath` | `PATH` entries |
| `--json` | One JSON object instead of the table ([shape](Scripting#-machine-readable-results---json)) |

## 💡 Examples

```bash
crower list                          # every stored command
crower list docker                   # name, alias, line, description or group contains "docker"
crower list -g work -s runs          # one group, most used first
crower list -i                       # what ran, and when
crower list -p                       # processes
crower list --folder ~/projects
crower list --system
crower list --json                     # the same list, as one JSON object
crower list -i --json | jq '.history[0]'
```

```text
#  NAME       ALIASES  GROUP  RUNS  LAST RUN  COMMAND
0  🚀 deploy  d        ops      12  2h ago    ssh prod 'cd /app && git pull'  # ship it
1  👋 greet   hi                 3  just now  echo hello {1}
```

`RUNS` and `LAST RUN` are counted from the history. On a terminal the list is
coloured and fits the window; piped (or with `NO_COLOR`) it is plain and whole.

> [!TIP]
> Don't remember the name? Run **`cr`** on its own: a fuzzy finder opens over
> your commands — type, `↑`/`↓`, `Enter`.

> [!NOTE]
> The table goes to **stdout**; status lines go to stderr. `crower list > file`
> captures the list and nothing else.

The history table has a `Command` column: the command an entry touched, or a
dash for an operation that touched a file or a variable.

## 🔗 Related

[`execute`](execute) · [`revert`](revert) · [🧩 Scripting](Scripting) · **[📖 Commands](How-to-Use)**
