# 🐚 Shell integration

**Every crower command, a command of its own — in bash, zsh, fish, PowerShell and cmd.**

```bash
deploy                   # instead of: crower deploy
```

## ⚡ Set it up

**Nothing to do.** The installers (`install.sh`, `add_to_path.sh`,
`add_to_path.bat`) run `crower init --all`, which sets up every shell on the
machine. On a machine set up another way, the first time you run crower in a
shell without the hook it asks once:

```text
crower: set up zsh so commands like "cd" work in this terminal? It adds one line to zsh's startup. [Y/n]
```

Enter sets it up; `n` is remembered for that shell (`~/crower/shell-setup.json`).
It never asks a script (stdin and stderr must be terminals), inside the hook, or
when `CROWER_NO_SETUP` is set. Either way, open a new terminal once afterwards —
a window that is already running cannot load it by itself.

| Command | Does |
| --- | --- |
| `crower init --apply` | set up the shell you run it from |
| `crower init --all` | set up every installed shell (bash, zsh, fish; cmd, Windows PowerShell, PowerShell 7) |
| `crower init` | show what the current shell needs |

PowerShell is asked where its profile is (`$PROFILE`), so Windows PowerShell 5.1,
PowerShell 7 and a OneDrive-redirected Documents folder all work. When the
execution policy blocks profile scripts (`Restricted`, the 5.1 default), crower
offers to set `RemoteSigned` for your user — scripts written on this computer
run, downloaded ones must be signed.

By hand:

| Shell | File | Line |
| --- | --- | --- |
| bash | `~/.bashrc` | `eval "$(crower init bash)"` |
| zsh | `~/.zshrc` | `eval "$(crower init zsh)"` |
| fish | `~/.config/fish/config.fish` | `crower init fish \| source` |
| PowerShell (5.1 and 7) | `$PROFILE` | `crower init powershell \| Out-String \| Invoke-Expression` |
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
- 🏠 **Runs here, when you ask.** A command created or updated with `--here`
  runs in your own shell: `crower create venv "source .venv/bin/activate" --here`
  then `venv`, `cr venv` or `cr execute venv` leaves the venv active. crower
  fills in the parameters and records the run (`crower execute <name> --emit`,
  which the hook calls), and the shell evaluates the line: `eval` in bash, zsh
  and fish, `Invoke-Expression` in PowerShell, `for /f … do call` in cmd.
  Without the hook — a script, the window, `crower execute` itself — the
  command runs as a subprocess like any other. Write the line for the shell you
  run it in; `exit` in it closes that shell. In fish and PowerShell a plain
  variable stays local to the hook's function: use `set -g` / `$global:`,
  while `cd` and environment variables (`export`, `$env:`) always reach the shell.
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

cmd has no startup script, so the hook is a doskey macro file that cmd's
`AutoRun` loads. Setting up cmd (the question, `init --apply` from cmd, or
`init --all`) writes `%USERPROFILE%\crower\crower.doskey` and adds
`doskey /macrofile="…" & set "CROWER_HOOK=cmd"` to
`HKCU\Software\Microsoft\Command Processor\AutoRun`, after any value that was
already there.

The file defines `cr` and `crower` macros: `cr desk` changes folder or runs
`--here` in the window, every other `cr …` line goes to the program, and then the
macro reloads the file — crower rewrites it after `create`, `update`, `delete`
and the other changes, so a new command works in the same window. A first or second word that contains a double quote
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
| crower says "only the shell hook can do" | This window has no hook: it predates the setup (open a new one), or the shell was never set up (`crower init --apply`). |
| A `cd` command runs but the folder does not change | The hook is not loaded (`type cr` should say it is a function; in cmd, `doskey /macros` should list `cr=`), the terminal predates the command's change (`exec $SHELL`), or the line is not a plain `cd <absolute folder>` — see "Jumps folders" above. The program on its own can never move your terminal. |
| `not an in-shell command …` from a name | The command lost `--here` in another terminal or through a pipe, and this shell's hook is stale. Open a new terminal. |
| A command is not defined | Its name is already taken in the shell — `type <name>` shows by what. Use `cr <name>`. |
| zsh: no completion | `compinit` must run before the hook line. |

## 🔗 Related

[`init`](init) · [`import`](share) · [⚡ Quick start](Quick-Start) · **[📖 Commands](How-to-Use)**
