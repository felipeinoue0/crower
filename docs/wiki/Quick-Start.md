<div align="center">

# ⚡ Quick start

**Five commands, one minute.**

</div>

## 0️⃣ Hook it into your shell

```bash
crower init --apply             # only if the installer or the first-run question did not
crower import --from-shell      # optional: bring the aliases you already have
```

Open a new terminal. From now on every command you save is a command of its own
— see [🐚 Shell integration](Shell-Integration).

## 1️⃣ Create a shortcut

```bash
crower create deploy "ssh user@server 'cd /app && git pull && systemctl restart app'"
```

`create` takes a name and the command to run later.

## 2️⃣ Run it

```bash
deploy                          # with the shell hook
cr deploy                       # anywhere
crower deploy --dry-run         # show the line, run nothing
```

`cr` is the same binary under a short name, and `execute` is optional when the
first argument is a stored name. Forgot the name? Run `cr` alone for a fuzzy
finder.

## 3️⃣ Add an alias

```bash
crower create backup -a bkp "tar -czf backup.tar.gz ~/important-data"
bkp
```

## 4️⃣ Pass parameters

```bash
crower create greet "echo Hello {1}, welcome to {2}!"
greet John Developer
```

`{1}`, `{2}`, … are positional; `{}` takes the rest. Arguments are quoted
before they reach the shell, so a value with a space stays one value.

---

## 🧭 What next

| | |
| --- | --- |
| 💬 **Don't want flags?** | `crower create` with no arguments asks, step by step |
| 🔎 **Find a command** | `crower list docker`, `crower list -s runs`, or just `cr` |
| 📜 **See what you did** | `crower list -i` — history, newest first |
| ↩️ **Undo the last operation** | `crower revert 1` |
| 🗑️ **Bring something back** | `crower restore <name>` |
| 🎛️ **Prefer a UI?** | `crower gui` (terminal) or `crower-gui` (window) |
| 🤝 **Share your set** | `crower export -o team.toml` |

> [!TIP]
> Every command that touches the filesystem or your command list appends to the
> history, and every entry can be reverted. You do not have to be careful to be
> productive.
