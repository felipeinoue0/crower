<div align="center">

# 🛠️ Development

**`make check` is the gate a change has to pass.**

</div>

## ⚙️ Commands

```bash
make check          # gofmt, vet, go test ./..., go build ./..., windows/darwin build
go test ./...       # tests only
make release        # cross-build the release archives into build/dist/
make help           # every target, one line each
```

`make check` fails when the toolchain is missing — a gate that cannot run reports
nothing, and reporting nothing must not look like success. It also lists the
packages that have **no test file**: `go test` prints `[no test files]` for them
and exits 0, so a green run is green for the packages that ran.

It builds for `windows/amd64` and `darwin/arm64` as well as the host. Several
packages pick their behaviour with build tags (`handler_windows.go`,
`handler_darwin.go`, …), and a host build on Linux never compiles those files, so
without the cross builds a Windows-only break would first surface in a release.

## 🐳 Dev sandbox

```bash
make dev-up         # the pinned toolchain, foreground
make dev-shell      # a shell inside it
make dev-opencode   # the agent inside it (Linux/WSL2)
make dev-down       # stop it
```

On any OS with Docker and no `make`: `docker compose up`. On native Windows:
`python scripts/dev-up.py dev-up`.

## 🖥️ The GUI on your host

`make check` does **not** compile the Fyne window: every file in
`internal/fyneui` is behind `//go:build fyne`, so the gate neither builds nor
vets it.

```bash
make gui_linux    # one command: build/crower-gui for this machine
```

The header check runs first, so a machine without them stops at the install
command for its distro — apt, dnf or pacman — instead of
`wayland-client-core.h: No such file or directory`, a C error that reads like a
broken checkout:

```bash
make gui-deps                 # the system packages, for your distro
sudo make gui-deps INSTALL=1  # install them, then re-run make gui_linux
```

It never runs `sudo` itself. `make dev-shell` needs none of this: the sandbox
image already carries the toolchain and the headers.

An editor without the tag reports the whole package as undefined
(`undefined: newGrowingEntry`). The committed `.vscode/settings.json` sets
`buildTags: fyne` and `CGO_ENABLED=1`; for another editor, give `gopls`
`buildFlags: ["-tags=fyne"]`.

`fyne.yml` is the only gate that builds and tests this package, and it is
manual dispatch:

```bash
gh workflow run fyne.yml --ref <branch>
```

## 🏛️ Layout

Four rings, one dependency direction, and a gate in `architecture_test.go` that
fails the build when an import points the wrong way:

```text
cmd/  ->  internal/application/  ->  internal/domain/
                 ^
        internal/adapters/

        internal/session/  (composition root, built by cmd)
```

`internal/session` is the composition root between `cmd` and the adapters.
**[Architecture](Architecture)** explains the rule, what each ring may import,
and where a new operation goes. Every operation has been ported (#215): there is
no `internal/legacy/`, `internal/app`, `internal/data` or `internal/helper` left
to shrink.

## 🌿 Branches

```text
main          ← release merges (dev → main, a pull request)
  └── dev     ← integration branch and the repository's default branch
        ├── feat/42-note-export
        ├── fix/46-test-fixture-race
        └── chore/44-agent-layer
```

`<type>/<issue>-<slug>`, conventional commits, one feature per pull request.
`AGENTS.md` holds the full rules.

## 🚀 Releasing

```bash
gh workflow run release.yml --ref main -f version=v0.6.0
```

The workflow tags `main`, builds one archive per platform — four natively and
two arm64 with `fyne-cross` — each containing `crower`, `cr` and `crower-gui`.
It verifies each binary's embedded revision, writes `checksums.txt`, and creates
a **draft** release — publish it by hand, or pass the `publish` input.

The published body follows `docs/release-notes-template.md` and ends with a
compare link, not a pasted PR list: `generate-notes` cannot see squash-merged
feature commits, and a pasted list goes stale. Highlights and Fixes are built
from the PRs merged into `dev` (`enhancement` and `bug`), and
`internal/tools/bodycheck` refuses a body that drops a section or reintroduces
a badge, so the dispatch fails before the draft exists.

## 🧪 What a green run proves

Formatting, vet, the tested packages on Linux/amd64, and that every archive
contains a binary built from the release commit. It does **not** run the darwin,
windows or arm64 binaries, and it does not draw a window. See
**[Verifying claims](Verifying-claims)**.

---

🐛 [Issues](https://github.com/fiwon123/crower/issues) ·
🎯 [Roadmap board](https://github.com/users/fiwon123/projects/5) ·
🤝 [Contributing](How-to-Contribute)
