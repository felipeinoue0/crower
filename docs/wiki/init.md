# 🚀 `init`

**Set up your shell: `PATH`, native aliases for every command, and tab completion.**

```bash
crower init                 # show what your shell needs
crower init --apply         # write it into the startup file
crower init zsh             # print the hook (what the startup line evaluates)
```

With no shell, `init` detects bash, zsh, fish or PowerShell and prints the
`PATH` line, the hook line, and the file they belong in. `--apply` appends both
once — a second run changes nothing.

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
