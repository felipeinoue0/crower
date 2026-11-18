# 🤝 `share` · `export` · `import`

**Move commands between machines, or pull a pack from a registry.**

```bash
crower export [name] [-o file]
crower import <file> [--overwrite]
crower import --from-shell [file|-] [--dry-run]
crower share list
crower share pull <pack>
```

## 🚩 Flags

| Command | Flag | Effect |
| --- | --- | --- |
| `export` | `-o`, `--out` | Output file path |
| `import` | `--overwrite` | Replace a command with the same name; drop aliases that conflict |
| `import` | `--from-shell` | Import your shell's aliases instead of a pack |
| `import` | `-n`, `--dry-run` | With `--from-shell`: list what would be imported, write nothing |

## 💡 Examples

```bash
crower export -o my-workflows.toml     # everything
crower export deploy -o deploy.toml    # one command
crower import my-workflows.toml
crower import team.toml --overwrite
crower share list
crower share pull docker-setup
```

## 🐚 Import your shell aliases

```bash
crower import --from-shell --dry-run      # what would come in
crower import --from-shell                # read every startup file
crower import --from-shell ~/.bash_aliases
alias | crower import --from-shell -      # the output of `alias`, any shell
```

With no file, crower **reads — never runs** — the alias definitions in
`~/.bashrc`, `~/.bash_aliases`, `~/.bash_profile`, `~/.profile`, `~/.zshrc`,
`~/.zsh_aliases`, fish's `config.fish`, `conf.d/` and saved alias functions, and
the PowerShell profiles (`Set-Alias` / `New-Alias`). Quoting (`'\''`, double
quotes), comments, `;` chains and several aliases on one line are understood;
zsh suffix aliases (`alias -s`) and shell functions are not imported.

An alias crower already has with the same line is skipped, so running it again
changes nothing. Imported commands land in the group **imported**, with a
description naming the file they came from. Pair it with the
[shell hook](Shell-Integration) and the aliases keep working by name.

## 🔀 Name and alias collisions

Import never lets two commands share a name or an alias. If a pack entry's name
or alias is already taken:

- **without `--overwrite`**, crower asks once per command; accepting renames the
  conflicting name or alias to `x_imported` (numbered if needed), declining
  skips that command;
- **with `--overwrite`**, a command with the same *name* is replaced, and a
  conflicting *alias* is dropped — taking it would silently edit the command
  that owns it.

Entries are processed in name order. Within one pack, a later entry that
collides with an earlier one is renamed the same way, even with `--overwrite`,
so two entries can never share a key.

`share pull` uses the non-overwrite path after its own confirmation.

## 🛰️ Registry protocol

Any HTTP server can host packs: `index.json` plus `packs/<name>.toml`. See
[crower-resources](https://github.com/fiwon123/crower-resources) for examples.

> [!NOTE]
> A pack name is a registry slug (`A-Z a-z 0-9 . _ -`). `share pull` rejects a
> name containing a separator or `..`, and the downloaded pack is written to a
> program-named temporary file only **after** you confirm the import.

> [!WARNING]
> An exported file contains what you typed — hosts, paths, sometimes secrets.
> Read it before you commit or send it, and prefer `{1}`-style parameters over
> baked-in addresses.

## 🔗 Related

[🤝 Sharing](Sharing) · [⚙️ Configuration](Configuration) · **[📖 Commands](How-to-Use)**
