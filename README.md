<p align="center">
  <img src="assets/logo.svg?v=2" alt="crower logo — a crow perched on a terminal cursor" width="128" height="128">
</p>

<h1 align="center">crower</h1>

<p align="center">
  <strong>One alias manager for every shell — with history, undo and a window.</strong><br>
  <em>Save a command once. Run it by name in bash, zsh, fish, PowerShell or cmd.</em>
</p>

<p align="center">
  <a href="https://github.com/felipeinoue0/crower/actions/workflows/ci.go.yml"><img src="https://github.com/felipeinoue0/crower/actions/workflows/ci.go.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/felipeinoue0/crower/releases"><img src="https://img.shields.io/github/v/release/felipeinoue0/crower" alt="Release"></a>
  <a href="https://github.com/felipeinoue0/crower/blob/main/LICENSE"><img src="https://img.shields.io/github/license/felipeinoue0/crower" alt="License"></a>
</p>

<p align="center">
  <a href="#-quick-start">Quick start</a> ·
  <a href="#-installation">Install</a> ·
  <a href="#-shell-integration">Shell integration</a> ·
  <a href="#-everyday-use">Everyday use</a> ·
  <a href="https://github.com/felipeinoue0/crower/wiki">Wiki</a>
</p>

---

```console
$ crower create deploy "ssh prod 'cd /app && git pull && systemctl restart app'" -a d --icon 🚀
$ deploy                      # a real command in your shell — no prefix
$ crower list
#  NAME       ALIASES  GROUP  RUNS  LAST RUN  COMMAND
0  🚀 deploy  d        ops      12  2h ago    ssh prod 'cd /app && git pull && systemctl restart app'
1  👋 greet   hi                 3  just now  echo hello {1}
$ cr                          # nothing else? a fuzzy finder opens — type, ↑↓, Enter
```

## ✨ Why crower

| | |
|---|---|
| 🐚 **Native in every shell** | `deploy`, not `cr deploy` — one hook for bash, zsh, fish, PowerShell and cmd, with tab completion. |
| 📥 **Bring what you have** | `crower import --from-shell` reads the aliases already in your `.bashrc`, `.zshrc`, fish config or PowerShell profile. |
| 🔎 **Find it again** | Search, groups, icons, run counts — and a fuzzy finder when you only remember half a name. |
| ⚡ **Runs like you typed it** | Attached to your terminal: `ssh`, `vim`, prompts and colours just work. `--dry-run` shows the line first. |
| 🧠 **Parameters and steps** | `{1}`, `{2}`, `{}` placeholders, quoted safely; multi-step commands with a folder, shell, env and timeout. |
| ↩️ **Undo anything** | Every change is recorded. `crower revert` steps back; `crower restore` brings a command back. |
| 🖥️ **A window too** | `crower-gui` — the same commands, history and file tools in a desktop window, light or dark. |
| 🌍 **Everywhere** | Linux, macOS and Windows, amd64 and arm64. One static binary for the CLI. |

## 🚀 Quick start

```bash
# 1. Install (Linux / macOS) — Windows: see Installation below
curl -fsSL https://raw.githubusercontent.com/felipeinoue0/crower/main/scripts/install.sh | sh

# 2. Open a new terminal: the installer already hooked crower into your shells
#    (on a machine set up another way, the first `cr` you run offers to)

# 3. Bring your existing aliases, or create new ones
crower import --from-shell
crower create greet "echo Hello {1}!" --alias hi

# 4. Run them by name
hi Ada                       # → Hello Ada!
```

## 📦 Installation

### 🐧 Linux · 🍎 macOS — one command

```bash
curl -fsSL https://raw.githubusercontent.com/felipeinoue0/crower/main/scripts/install.sh | sh
```

It picks the newest release, verifies it against `checksums.txt`, and installs
`crower` and `cr` into `~/.local/bin`. Pin a version or a folder with
`sh -s -- --version v0.21.0 --dir ~/.local/bin`.

### 🪟 Windows

Download `crower_<tag>_windows_amd64.zip` (or `_arm64`) from the
[releases page](https://github.com/felipeinoue0/crower/releases), extract it, and run
`add_to_path.bat` from the extracted folder — it also sets up cmd and PowerShell,
so open a new window and every command works there. Double-click
**`crower-gui.exe`** for the window.

### 📥 Any platform — from a release archive

Archives are named `crower_<tag>_<os>_<arch>` and contain `crower`, `cr` and
`crower-gui`. Swap `linux_amd64` for `linux_arm64`, `darwin_amd64` or `darwin_arm64`:

```bash
VERSION=v0.21.0
curl -LO "https://github.com/felipeinoue0/crower/releases/download/$VERSION/crower_${VERSION}_linux_amd64.tar.gz"
tar -xzf "crower_${VERSION}_linux_amd64.tar.gz"
sudo mv crower cr crower-gui /usr/local/bin/
```

No `sudo`? Run `./add_to_path.sh` from the extracted folder instead.

### 🛠️ From source

```bash
git clone https://github.com/felipeinoue0/crower.git && cd crower
go build -o crower . && cp crower cr      # Go 1.25.4+
```

Check any install with `crower --version`. Update in place with `crower --upgrade`.

## 🐚 Shell integration

One line in your startup file makes every crower command a command of its own,
keeps it in sync when you create or delete one, turns on tab completion, and lets
a command change your terminal's folder (`cd`) or run in it (`--here`).

**You don't add it yourself.** The installers run `crower init --all`, which sets
up every shell on the machine; and the first time you run crower in a shell
without it, crower asks once (Enter = yes) and does it. `crower init --apply`
sets up the current shell, `CROWER_NO_SETUP=1` turns the question off. By hand:

| Shell | Add to | Line |
|---|---|---|
| bash | `~/.bashrc` | `eval "$(crower init bash)"` |
| zsh | `~/.zshrc` | `eval "$(crower init zsh)"` |
| fish | `~/.config/fish/config.fish` | `crower init fish \| source` |
| PowerShell (5.1 and 7) | `$PROFILE` | `crower init powershell \| Out-String \| Invoke-Expression` |
| cmd.exe | `AutoRun` (registry) | `crower init --apply` from cmd writes `%USERPROFILE%\crower\crower.doskey` and loads it from `AutoRun` |

- 🛡️ **Never shadows anything.** A name your shell already knows — a program on
  `PATH`, a builtin, your own alias — is skipped, so crower can't hide `git` or `cd`.
- 🔄 **Always current.** `crower create`, `update`, `delete`, `import`, `revert`…
  re-sync the aliases in the shell you ran them from.
- ⌨️ **Completes names.** `cr dep<Tab>` and `crower delete <Tab>` offer your commands
  with their descriptions.
- ⚡ **Cheap.** One process at shell start, a few milliseconds.

## 🧭 Everyday use

### ➕ Create and organise

```bash
crower create build "go build ./..."
crower create test "go test ./..." --alias t --group dev --icon 🧪
crower create seek "grep -rn {1} {2}" --description "search code" --params pattern,path
crower update test --exec "go test -race ./..."
crower delete build
```

### 📥 Import your shell aliases

```bash
crower import --from-shell --dry-run      # see what it found
crower import --from-shell                # bash, zsh, fish and PowerShell files
alias | crower import --from-shell -      # or pipe the output of `alias`
```

Files are read, never executed. Importing twice changes nothing.

### 🔎 Find and run

```bash
crower list                    # every command, with runs and last run
crower list docker             # search name, alias, line, description, group
crower list -g dev -s runs     # one group, most used first
cr                             # fuzzy finder: type, ↑↓, Enter
seek TODO internal/            # run by name (with the shell hook)
crower seek TODO . --dry-run   # print the expanded line, run nothing
```

### 🧠 Parameters and multi-step commands

`{1}`, `{2}` are positional, `{}` takes the rest, and every value is quoted
before it reaches the shell. With no placeholder, arguments are appended — like
a shell alias.

```bash
crower create greet "echo Hello {1}, welcome to {2}!"
greet Ada "the team"           # Hello Ada, welcome to the team!
```

Chain with `&&`, or give a command `Steps` in the configuration file; each step
runs in the command's folder, shell, env and timeout:

```toml
[release]
Steps   = ["go test ./...", "go build -o app .", "./app --version"]
Cwd     = "~/projects/app"
Timeout = "5m"
```

### ↩️ History and undo

```bash
crower list --history          # everything crower did, newest first
crower revert 2                # undo the last two operations
crower restore deploy          # bring back a deleted command
```

### 🗂️ Files and folders

```bash
crower create --file notes.txt && crower create --folder reports
crower copy notes.txt ~/backup && crower move old.txt archive/
crower open --folder . && crower extract release.tar.gz
crower search report ./notes
```

## 🖥️ The window

`crower-gui` (or double-click it) opens the same commands in a desktop window:
search and group them, run with parameters, edit, browse and restore the
history, copy/move/search/extract files, and update crower — in light or dark.
`crower gui` is the keyboard menu for a terminal without a display.

## 🧩 Scripting

- 📤 **Results on stdout, diagnostics on stderr** — piping never picks up prompts.
- ✅ **Exit status says whether the work happened** — `1` with one `error:` line when it didn't.
- 🚫 **Never hangs** — `--no-input` (or closed stdin) fails instead of prompting.
- 🧾 **JSON** — `crower list --json`, `crower list -i --json`, `crower doctor --json`.

```bash
crower --no-input delete old-command || exit 1
crower list --json | jq -r '.commands[] | select(.runs > 10) | .name'
```

`--config <path>` points a run at another command file, so a script can carry
its own set.

## 🤝 Sharing

```bash
crower export -o team.toml                 # all commands (or: crower export deploy)
crower import team.toml                    # merge, asking on conflicts
crower share list && crower share pull docker-setup   # from a registry
```

A registry is any HTTP server with `index.json` + `packs/<name>.toml` — see
[crower-resources](https://github.com/felipeinoue0/crower-resources).

## ⚙️ Configuration

Everything lives in `~/crower/`: `crower.yaml` (your commands, TOML),
`history/` and `historyhelper.json` (undo), `crower.log`, `settings.json`
(`update_check`: `daily`, `weekly` or `never`). Override with `--config`.

**Setups** keep separate command sets with separate history — work, home, a
stream:

```bash
crower --setup streamer create obs "obs"
CROWER_SETUP=streamer crower list
```

`crower doctor` shows every path, permission and setting crower depends on.

## 🛠️ Development

```bash
make check          # the local gate: gofmt, vet, test, build
make release        # Windows and Linux release archives into build/
make help           # every target
go run ./ --help
```

`make dev-up` (or `docker compose up`) starts a sandbox with the pinned Go
toolchain; on native Windows use `python scripts/dev-up.py dev-up`. Architecture
and conventions: [AGENTS.md](AGENTS.md) · [PROJECT_CONTEXT.md](PROJECT_CONTEXT.md).

## 📚 Documentation

| I want to… | Read |
|---|---|
| 🐚 set up my shell | [Shell integration](https://github.com/felipeinoue0/crower/wiki/Shell-Integration) |
| 📖 look up a command or flag | [Command reference](https://github.com/felipeinoue0/crower/wiki/Command-Reference) |
| 🍳 copy a working example | [Recipes](https://github.com/felipeinoue0/crower/wiki/Recipes) |
| 🧩 script against crower | [Scripting](https://github.com/felipeinoue0/crower/wiki/Scripting) |
| 🚑 fix something | [Troubleshooting](https://github.com/felipeinoue0/crower/wiki/Troubleshooting) |
| 🏛️ understand the code | [Architecture](https://github.com/felipeinoue0/crower/wiki/Architecture) |
| 🎯 see what's planned | [Roadmap board](https://github.com/users/felipeinoue0/projects/5) |

## 💬 Contributing

Issues, ideas and pull requests are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
One feature per pull request; say what ran and what did not.

## ⚠️ What is not proven

This project says what it verified. CI tests and builds on Linux/amd64 and
cross-compiles the rest — no gate executes the other builds.

- 🍎 **macOS** and 🪟 **Windows** builds are cross-compiled, never started by CI
  (macOS shell operations: [#129](https://github.com/felipeinoue0/crower/issues/129)).
- 💪 **arm64** builds are never executed on ARM hardware.
- 🐟 **fish, PowerShell and cmd hooks** are unit-tested for shape; bash and zsh
  were run in real sessions.
- ⌨️ **The fuzzy finder on Windows** uses the console's VT input mode, untested
  on a real console.

Running it and reporting what happened — "it worked" included — is the most
useful contribution right now: [Verifying claims](https://github.com/felipeinoue0/crower/wiki/Verifying-claims).

## 📄 License

**MIT** — see [LICENSE](LICENSE). The archives also carry [NOTICE](NOTICE)
(attribution for bundled dependencies) and [LICENSE-APACHE](LICENSE-APACHE),
which the Apache-2.0 dependencies require to travel with a distribution; they do
not change crower's own licence.
