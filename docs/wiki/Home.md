<div align="center">

# 🐦 crower

**One alias manager for every shell — with history, undo and a window.**

Save a command once and run it by name in bash, zsh, fish, PowerShell or cmd.
Every change is recorded and can be undone.

[![Release](https://img.shields.io/github/v/release/felipeinoue0/crower?style=for-the-badge&color=4C8BF5&label=release)](https://github.com/felipeinoue0/crower/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/felipeinoue0/crower/ci.go.yml?branch=main&style=for-the-badge&label=ci&color=22C55E)](https://github.com/felipeinoue0/crower/actions/workflows/ci.go.yml)
[![License](https://img.shields.io/badge/license-MIT-22C55E?style=for-the-badge)](https://github.com/felipeinoue0/crower/blob/main/LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-linux%20%7C%20macOS%20%7C%20windows-8250DF?style=for-the-badge)](How-to-Install)

</div>

---

## ⚡ Start in thirty seconds

```bash
crower init --apply             # once: PATH, native aliases, completion
crower import --from-shell      # bring the aliases you already have

# a shortcut for the command you keep retyping
crower create dps "docker ps --format 'table {{.Names}}\t{{.Status}}'"

dps                             # it's a command now — in any shell
cr                              # forgot the name? fuzzy finder
```

<p align="center">
  <a href="How-to-Install"><img src="https://img.shields.io/badge/📦_install-4C8BF5?style=for-the-badge" alt="Install"></a>
  <a href="Quick-Start"><img src="https://img.shields.io/badge/⚡_quick_start-F59E0B?style=for-the-badge" alt="Quick start"></a>
  <a href="How-to-Use"><img src="https://img.shields.io/badge/📖_commands-8250DF?style=for-the-badge" alt="Commands"></a>
</p>

## 🎯 Where to go

| I want to… | Read |
| --- | --- |
| install it | **[📦 Installation](How-to-Install)** |
| create my first shortcut | **[⚡ Quick start](Quick-Start)** |
| run commands by name in my shell | **[🐚 Shell integration](Shell-Integration)** |
| look up a command or flag | **[📖 Command reference](Command-Reference)** |
| copy a recipe into my shell | **[🍳 Recipes](Recipes)** |
| call crower from a script | **[🧩 Scripting](Scripting)** |
| share commands with my team | **[🤝 Sharing](Sharing)** |
| find where my data lives | **[⚙️ Configuration](Configuration)** |
| fix something | **[🚑 Troubleshooting](Troubleshooting)** |
| build or release it | **[🛠️ Development](Development)** |
| understand the code layout | **[🏛️ Architecture](Architecture)** |
| help prove a claim | **[🧪 Verifying claims](Verifying-claims)** |

## ✨ What it does

| | Capability |
| :--: | --- |
| 🔖 | **Shortcut anything** — any shell command becomes a name and an alias |
| 🐚 | **Native in every shell** — `deploy`, not `cr deploy`, with tab completion |
| 📥 | **Import** — the aliases already in your bash, zsh, fish or PowerShell config |
| 🔎 | **Find** — search, groups, run counts, and a fuzzy finder on bare `cr` |
| 🖥️ | **Interactive** — runs attached to your terminal; `--dry-run` shows the line first |
| ⚡ | **Chain steps** — multi-step commands with automatic error handling |
| 🔗 | **Nest commands** — call a crower command from another, with recursion protection |
| 📝 | **Annotate** — notes on commands, next to the history that shows what ran |
| ↩️ | **Undo** — every operation is reversible: `revert`, `restore` |
| 📁 | **Manage files** — create, copy, move, open, delete from one CLI |
| 🔍 | **Search** — files, folders, or the web |
| 📦 | **Share** — export, import, or pull a pack from a registry |

> [!NOTE]
> **Two ways in.** `crower gui` is the terminal menu; `crower-gui` (in the same
> archive) is the desktop window, light or dark. Both read the same
> configuration as the CLI.

## ⚠️ What is not proven

This project says what it verified. CI formats, vets, tests and builds on
**Linux/amd64**, then cross-compiles six archives on the same runner — no gate
executes what it built. macOS, Windows and arm64 binaries are **build claims,
not run claims**, and nothing draws a window in CI.

**[🧪 Verifying claims](Verifying-claims)** lists each one, the steps, and what
to record. Running the program and reporting what happened — including "it
worked" — is the most useful contribution right now.

---

<div align="center">

**[⬇️ Releases](https://github.com/felipeinoue0/crower/releases)** ·
**[🐛 Issues](https://github.com/felipeinoue0/crower/issues)** ·
**[🎯 Roadmap board](https://github.com/users/felipeinoue0/projects/5)** ·
**[🤝 Contributing](https://github.com/felipeinoue0/crower/blob/dev/CONTRIBUTING.md)**

<sub>MIT licensed · Apache-2.0 notices ship in the archives · built with an AI agent, reviewed by a human</sub>

</div>
