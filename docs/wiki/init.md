# 🚀 `init`

**Set up your shell: `PATH`, native aliases for every command, and tab completion.**

```bash
crower init                 # show what your shell needs
crower init --apply         # set up the shell you run it from
crower init --all           # set up every shell on this computer
crower init zsh             # print the hook (what the startup line evaluates)
```

You rarely need it: the installers run `init --all`, and crower offers to set
up a shell the first time it runs there without the hook. With no shell, `init`
detects the shell it was started from — bash, zsh, fish, cmd, Windows PowerShell
or PowerShell 7 — and shows what it needs. `--apply` sets it up once (a second
run changes nothing): the startup file for bash, zsh and fish, the profile
PowerShell reports, or cmd's `AutoRun`. `--all` does that for every installed
shell.

```text
$ crower init
Shell:   zsh
PATH:    export PATH="$HOME/.local/bin:$PATH"
Hook:    command -v crower >/dev/null 2>&1 && eval "$(crower init zsh)"
Startup: /home/me/.zshrc
```

With a shell — `bash`, `zsh`, `fish`, `powershell` or `cmd` — `init` prints the
hook itself. What it does, per shell: **[🐚 Shell integration](Shell-Integration)**.

> [!NOTE]
> `--apply` changes a startup file. Read the printed lines first if you keep your
> shell configuration under version control.

## 🔗 Related

[🐚 Shell integration](Shell-Integration) · [📦 Installation](How-to-Install) · **[📖 Commands](How-to-Use)**
