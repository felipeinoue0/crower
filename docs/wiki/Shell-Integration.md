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
- 📂 **Jumps folders.** A command that is only `cd <folder>` moves the
  terminal you typed it in — `crower create desk "cd /mnt/c/Users/me/Desktop"`
  then `desk`, `cr desk` or `cr execute desk`. A cd run by `crower execute` happens in a child process and is
  lost, so for these the hook runs crower first (the run is still counted, and
  a missing folder stops the jump) and then changes folder in your shell:
  `cd` in bash, zsh and fish, `Set-Location` in PowerShell, `cd /d` in cmd.
  The folder must be absolute or start with `~`, written as one word (quote it
  if it has spaces); `cd /d` and `chdir` are accepted. A line with anything
  more — `cd /app && git pull`, `cd $DIR`, `cd src` — runs as before.
  `cr <name>` follows crower's own rule: a name that is also a subcommand
  (`list`, `create`, …) runs the subcommand, so use `cr execute <name>`.
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

When a command only changes folder, the file also defines `cr` and `crower`
macros, so `cr desk` moves the window too; every other `cr …` line is passed to
the program unchanged. A first or second word that contains a double quote
(`cr "my cmd"`) confuses cmd's `if`; use the bare name there.

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
| A `cd` command runs but the folder does not change | The hook is not loaded (`type cr` should say it is a function; in cmd, `doskey /macros` should list `cr=`), the terminal predates the command's change (`exec $SHELL`), or the line is not a plain `cd <absolute folder>` — see "Jumps folders" above. The program on its own can never move your terminal. |
| A command is not defined | Its name is already taken in the shell — `type <name>` shows by what. Use `cr <name>`. |
| zsh: no completion | `compinit` must run before the hook line. |

## 🔗 Related

[`init`](init) · [`import`](share) · [⚡ Quick start](Quick-Start) · **[📖 Commands](How-to-Use)**
