# 🐚 Shell integration

**Every crower command, a command of its own — in bash, zsh, fish, PowerShell and cmd.**

```bash
crower init --apply      # once; then open a new terminal
deploy                   # instead of: crower deploy
```

## ⚡ Set it up

`crower init --apply` detects your shell and writes the line for you. By hand:

| Shell | File | Line |
| --- | --- | --- |
| bash | `~/.bashrc` | `eval "$(crower init bash)"` |
| zsh | `~/.zshrc` | `eval "$(crower init zsh)"` |
| fish | `~/.config/fish/config.fish` | `crower init fish \| source` |
| PowerShell 7 | `$PROFILE` | `crower init powershell \| Out-String \| Invoke-Expression` |
| cmd.exe | — | see [cmd.exe](#-cmdexe) below |

`--apply` writes a guarded form (`command -v crower … && eval …`), so a shell
without crower installed still starts cleanly.

## 🧩 What the hook does

- ➕ **One alias per name and alias.** `crower create greet "echo hi" -a hi`
  gives you `greet` and `hi`. Arguments pass through untouched —
  `greet -x 'two words'` runs `crower execute greet -- -x 'two words'`.
- 🛡️ **Never shadows.** If the shell already knows the name — a program on
  `PATH`, a builtin or keyword, your own alias or function — crower skips it.
  A command named `ls` or `git` stays reachable as `cr ls`.
- 🔄 **Stays in sync.** The hook wraps `crower` and `cr`: after `create`,
  `update`, `delete`, `import`, `revert`, `restore`, `reset` or `setup`, the
  aliases are rebuilt in that shell — a new command works on the next line, a
  deleted one is gone.
- ⌨️ **Completes.** `cr <Tab>`, `crower execute <Tab>`, `update`, `delete` and
  `export` offer command names; zsh, fish and PowerShell show each one's
  description (or the line it runs).
- 🪶 **Costs one process** at shell start — a few milliseconds — and never
  checks for updates.

Names the hook can define: letters, digits, `_ . + -`, not starting with `-`.
`crower` and `cr` themselves are reserved.

## 🪟 cmd.exe

cmd has no startup script, so the hook is a doskey macro file:

```bat
crower init cmd > "%USERPROFILE%\crower.doskey"
doskey /macrofile="%USERPROFILE%\crower.doskey"
```

To load it in every cmd window, set the `AutoRun` value of
`HKCU\Software\Microsoft\Command Processor` to the `doskey` line. cmd does not
re-sync: run the first line again after adding commands.

## 🧪 Check it

```bash
type deploy              # bash/zsh: "deploy is aliased to `command crower execute deploy --'"
crower init zsh | head   # see exactly what the hook defines
```

## 🚑 Troubleshooting

| Symptom | Cause |
| --- | --- |
| `deploy: command not found` right after `create` in a script | Aliases apply to lines read *after* they are defined; in an interactive shell the next prompt has it. |
| `echo y \| crower delete x` keeps the alias (bash) | A function at the end of a pipe runs in a subshell; run it without the pipe. zsh is unaffected. |
| A command is not defined | Its name is already taken in the shell — `type <name>` shows by what. Use `cr <name>`. |
| zsh: no completion | `compinit` must run before the hook line. |

## 🔗 Related

[`init`](init) · [`import`](share) · [⚡ Quick start](Quick-Start) · **[📖 Commands](How-to-Use)**
