APP_NAME := crower
APP_NAME_SHORT := cr
GUI_NAME := crower-gui
GUI_PACKAGE = $(BUILD_DIR)/gui-package
BUILD_DIR := build
SCRIPTS_DIR := scripts
SCRIPT_NAME := add_to_path
INJECT_VERSION:= github.com/fiwon123/crower/cmd.Version

# The Fyne window is a GUI-subsystem program on Windows. Without -H windowsgui
# the loader gives it a console window, so double-clicking the executable opens
# a terminal instead of the window. The icon, manifest and version are a .syso
# in the package directory — the only hook Go offers for a Windows resource.
GUI_LDFLAGS = -X $(INJECT_VERSION)=$(VERSION)

ifeq ($(GOOS),windows)
GUI_LDFLAGS += -H windowsgui
endif

# Build tools run on the host even when GOOS points at the release target: a
# GOOS=windows `go run` would produce a Windows binary the Linux runner cannot
# execute. CGO is off for them too, so the cross compiler in CC is not asked to
# build the host's runtime/cgo.
HOST_GO := GOOS=$$(go env GOHOSTOS) GOARCH=$$(go env GOHOSTARCH) CGO_ENABLED=0

VERSION ?= $(shell git describe --tags --always)
DIST_DIR := $(BUILD_DIR)/dist

WINDOWS_BIN := $(BUILD_DIR)/$(APP_NAME).exe
SHORT_WINDOWS_BIN := $(BUILD_DIR)/$(APP_NAME_SHORT).exe
WINDOWS_ZIP := $(BUILD_DIR)/$(APP_NAME)_$(VERSION)_windows.zip
WINDOWS_SCRIPT :=  $(SCRIPTS_DIR)/$(SCRIPT_NAME).bat

LINUX_BIN := $(BUILD_DIR)/$(APP_NAME)
SHORT_LINUX_BIN := $(BUILD_DIR)/$(APP_NAME_SHORT)
LINUX_TAR := $(BUILD_DIR)/$(APP_NAME)_$(VERSION)_linux.tar.gz
LINUX_SCRIPT :=  $(SCRIPTS_DIR)/$(SCRIPT_NAME).sh


.PHONY: all windows linux zip_linux zip_windows clean release check gui-deps \
	help dev-up dev-down dev-shell dev-opencode dev-opencode-auto dev-log dev-build reset \
	preflight prepare-agent-mounts check-identity \
	shell opencode opencode-auto release-platform release-gui-platform package-cli package-gui verify-release

all: release

# Create build folder if missing
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)
	mkdir -p $(DIST_DIR)

# Build Windows binary
windows: $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X $(INJECT_VERSION)=$(VERSION)" -o $(WINDOWS_BIN)
	cp $(WINDOWS_BIN) $(SHORT_WINDOWS_BIN)

# Build Linux binary
linux: $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X $(INJECT_VERSION)=$(VERSION)" -o $(LINUX_BIN)
	cp $(LINUX_BIN) $(SHORT_LINUX_BIN)

# Compress Windows binary
zip_windows: windows
	zip -j $(WINDOWS_ZIP) $(WINDOWS_BIN) $(SHORT_WINDOWS_BIN) $(WINDOWS_SCRIPT) README.md LICENSE LICENSE-APACHE NOTICE

# Compress Linux binary
zip_linux: linux
	tar -czvf $(LINUX_TAR) \
	          -C $(BUILD_DIR) $(notdir $(LINUX_BIN)) $(notdir $(SHORT_LINUX_BIN)) \
	          -C ../$(SCRIPTS_DIR) $(notdir $(LINUX_SCRIPT)) ../README.md ../LICENSE ../LICENSE-APACHE ../NOTICE

# Clean build folder
clean:
	rm -rf $(BUILD_DIR)

# Build & compress everything
release: zip_windows zip_linux
	@echo "Release ready: $(VERSION)"

# Platform release for a CLI-only archive (local use).
#
# The release workflow builds one combined archive per platform with
# release-gui-platform; this target stays for a local CLI-only build.
#
# The binary inside the archive is always named $(APP_NAME)$(EXT) /
# $(APP_NAME_SHORT)$(EXT) — never arch-suffixed. The platform and architecture
# are already in the archive filename, and the documentation tells users to run
# `crower` and `cr`, so a suffixed binary is unreachable by the name it ships.
# verify-release enforces that.
release-platform: $(BUILD_DIR)
	mkdir -p $(DIST_DIR)
	@echo "Building $(APP_NAME) for $(GOOS)/$(GOARCH) with version $(VERSION)"
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags "-X $(INJECT_VERSION)=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)$(EXT)
	cp $(BUILD_DIR)/$(APP_NAME)$(EXT) $(BUILD_DIR)/$(APP_NAME_SHORT)$(EXT)
	@$(MAKE) package-cli GOOS=$(GOOS) GOARCH=$(GOARCH)
	@$(MAKE) verify-release

# Platform release for a platform that carries the Fyne GUI.
#
# Fyne needs CGO, so this target runs on the target OS runner (Linux, Windows
# or macOS), not as a cross-compile. It builds the cgo-free `crower`/`cr` pair
# and `crower-gui`, then packages all three into the one platform archive.
release-gui-platform: $(BUILD_DIR)
	mkdir -p $(DIST_DIR) $(GUI_PACKAGE)
	@echo "Building $(APP_NAME), $(APP_NAME_SHORT) and $(GUI_NAME) for $(GOOS)/$(GOARCH) with version $(VERSION)"
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags "-X $(INJECT_VERSION)=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)$(EXT)
	cp $(BUILD_DIR)/$(APP_NAME)$(EXT) $(BUILD_DIR)/$(APP_NAME_SHORT)$(EXT)
ifeq ($(GOOS),windows)
	@$(HOST_GO) go run ./internal/tools/winres -icon internal/fyneui/logo.png \
		-out cmd/crowergui/rsrc_windows_$(GOARCH).syso -arch $(GOARCH) \
		-name $(GUI_NAME) -version "$(VERSION)"
endif
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -tags fyne -ldflags "$(GUI_LDFLAGS)" -o $(BUILD_DIR)/$(GUI_NAME)$(EXT) ./cmd/crowergui
ifeq ($(GOOS),windows)
	@rm -f cmd/crowergui/rsrc_windows_$(GOARCH).syso
	@$(HOST_GO) go run ./internal/tools/winres -verify $(BUILD_DIR)/$(GUI_NAME).exe
	@$(HOST_GO) go run ./internal/tools/winres -verify-console $(BUILD_DIR)/$(APP_NAME).exe
	@$(HOST_GO) go run ./internal/tools/winres -verify-console $(BUILD_DIR)/$(APP_NAME_SHORT).exe
	@# The mingw cross-link uses msvcrt, the C runtime every Windows has; a
	@# native build would import the UCRT API sets instead, which is the heap of
	@# small DLLs a machine can be missing. Keep the exe on msvcrt (#468).
	@if command -v x86_64-w64-mingw32-objdump >/dev/null 2>&1; then \
		if x86_64-w64-mingw32-objdump -p $(BUILD_DIR)/$(GUI_NAME).exe | grep -q 'api-ms-win-crt'; then \
			echo "ERROR: $(GUI_NAME).exe imports the UCRT API sets; the cross-link must stay on msvcrt (#468)" >&2; \
			exit 1; \
		fi; \
		if ! x86_64-w64-mingw32-objdump -p $(BUILD_DIR)/$(GUI_NAME).exe | grep -q 'msvcrt.dll'; then \
			echo "ERROR: $(GUI_NAME).exe does not import msvcrt.dll; the cross-link changed runtime (#468)" >&2; \
			exit 1; \
		fi; \
	fi
endif
	@$(MAKE) package-gui GOOS=$(GOOS) GOARCH=$(GOARCH)
	@$(MAKE) verify-release GUI=1

# Fail the build if an archive does not carry the names users are told to run.
# GOOS/GOARCH are pinned to the host: this target inspects archives, so it must
# run on the machine doing the build rather than be cross-compiled to the
# archive's target. GUI=1 additionally requires crower-gui in every archive.
ARCHIVEVERIFY_FLAGS := $(if $(filter 1,$(GUI)),-gui,)

verify-release:
	@GOOS=$$(go env GOHOSTOS) GOARCH=$$(go env GOHOSTARCH) CGO_ENABLED=0 go run ./internal/tools/archiveverify $(ARCHIVEVERIFY_FLAGS) $(DIST_DIR)

ifeq ($(GOOS),windows)
EXT := .exe
SCRIPT_EXT := .bat
CLI_ARCHIVE := $(DIST_DIR)/$(APP_NAME)_$(VERSION)_$(GOOS)_$(GOARCH).zip
else
EXT :=
SCRIPT_EXT := .sh
CLI_ARCHIVE := $(DIST_DIR)/$(APP_NAME)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz
endif

package-cli: $(BUILD_DIR)
	@mkdir -p $(DIST_DIR)
ifeq ($(GOOS),windows)
	zip -j $(CLI_ARCHIVE) $(BUILD_DIR)/$(APP_NAME)$(EXT) $(BUILD_DIR)/$(APP_NAME_SHORT)$(EXT) $(SCRIPTS_DIR)/add_to_path$(SCRIPT_EXT) README.md LICENSE LICENSE-APACHE NOTICE
else
	tar -czvf $(CLI_ARCHIVE) -C $(BUILD_DIR) $(APP_NAME)$(EXT) $(APP_NAME_SHORT)$(EXT) -C ../$(SCRIPTS_DIR) add_to_path$(SCRIPT_EXT) ../README.md ../LICENSE ../LICENSE-APACHE ../NOTICE
endif
	@echo "CLI package: $(CLI_ARCHIVE)"

package-gui: $(BUILD_DIR)
	@mkdir -p $(DIST_DIR) $(GUI_PACKAGE)
	cp $(BUILD_DIR)/$(APP_NAME)$(EXT) $(BUILD_DIR)/$(APP_NAME_SHORT)$(EXT) $(BUILD_DIR)/$(GUI_NAME)$(EXT) \
		$(SCRIPTS_DIR)/$(SCRIPT_NAME)$(SCRIPT_EXT) README.md NOTICE LICENSE LICENSE-APACHE LICENSE-FYNE $(GUI_PACKAGE)/
ifeq ($(GOOS),windows)
	cp $(SCRIPTS_DIR)/WINDOWS.txt $(GUI_PACKAGE)/
	cd $(GUI_PACKAGE) && zip -j $(abspath $(CLI_ARCHIVE)) *
else
	tar -czvf $(CLI_ARCHIVE) -C $(GUI_PACKAGE) $(APP_NAME)$(EXT) $(APP_NAME_SHORT)$(EXT) $(GUI_NAME)$(EXT) $(SCRIPT_NAME)$(SCRIPT_EXT) README.md NOTICE LICENSE LICENSE-APACHE LICENSE-FYNE
endif
	@echo "Platform package: $(CLI_ARCHIVE)"


# ---------------------------------------------------------------------------
# Host prerequisites for the Fyne window
#
# `crower-gui` is a CGO build behind `-tags fyne`, so the C toolchain needs the
# X11/GL and Wayland development headers. The dev sandbox and every CI job
# install them; a developer's machine usually has not, and the failure is a
# missing C header (`wayland-client-core.h`) rather than anything Go reports —
# so it reads like a broken checkout when it is a missing apt package.
#
# The list lives here and nowhere else: `fyne.yml` and `release.yml` call this
# target instead of repeating it, so CI and a host install the same packages.
#
# Stdout is the bare package list, so a workflow can consume it with
# `$(make -s gui-deps)`; the hint for a human goes to stderr.
# ---------------------------------------------------------------------------
GUI_DEPS := build-essential libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev wayland-protocols

gui-deps: ## Print the Linux packages the Fyne GUI needs (INSTALL=1 installs them)
ifeq ($(INSTALL),1)
	sudo apt-get update
	sudo apt-get install -y $(GUI_DEPS)
else
	@echo $(GUI_DEPS)
	@echo "Install them with: sudo make gui-deps INSTALL=1" >&2
	@echo "Then build with:   CGO_ENABLED=1 go build -tags fyne ./cmd/crowergui" >&2
endif


# ---------------------------------------------------------------------------
# Local verification gate
#
# gofmt, vet, test, build — in that order, because a formatting failure hides
# everything after it.
#
# Two properties this target deliberately has:
#
#   - A missing toolchain FAILS the target. It does not skip. A gate that
#     cannot fail is worse than no gate: it reads green and proves nothing.
#   - After the run it names the packages that have no test files. `go test`
#     prints "[no test files]" for those and still exits 0, so a green run
#     would otherwise look like coverage that does not exist.
#
# The two cross builds exist because several packages choose their behaviour
# with build tags (`handler_windows.go`, `handler_darwin.go`, ...). A host build
# on Linux never compiles those files, so a Windows-only break would reach a
# release: CI builds Linux on every pull request, and the release matrix is the
# first job that compiles the other five targets. These builds are cheap and
# they can fail.
#
# Tests use isolated temp directories (internal/helper/tests.go), so parallel
# runs are not racing over a shared fixture (#47) — a red run is a regression
# until proven otherwise, not a baseline to discount.
# ---------------------------------------------------------------------------
# set -e matters: without it the recipe's status is the last command's (the
# note), so a failing vet, test or build still exited 0 — a gate that could not
# fail. The `|| true` keeps an empty untested-package list from tripping it.
check: ## gofmt, vet, test, build — the local verification gate
	@set -e; \
	if ! command -v go >/dev/null 2>&1; then \
		echo "ERROR: go is not installed — the check cannot run, so nothing is verified" >&2; \
		exit 1; \
	fi; \
	out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then echo "unformatted files:"; echo "$$out"; exit 1; fi; \
	go vet ./...; \
	go test ./...; \
	go build ./...; \
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...; \
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./...; \
	notested="$$(go list -f '{{if and (not .TestGoFiles) (not .XTestGoFiles)}}{{.ImportPath}}{{end}}' ./... | grep -v '^$$' || true)"; \
	if [ -n "$$notested" ]; then \
		echo ""; \
		echo "NOTE: these packages have no test files — this run does not cover them:"; \
		echo "$$notested" | sed 's/^/  /'; \
	fi

# ---------------------------------------------------------------------------
# Dev sandbox
#
# A pinned toolchain and a shell that behave the same on every machine. The
# sandbox is optional: the project still builds and tests with a plain host
# Go toolchain (see the `check` target above).
#
# Cross-platform note: this Makefile targets POSIX hosts — Linux, macOS and
# WSL2 — where `make` runs recipes under a POSIX shell. Native Windows users
# should use `python scripts/dev-up.py dev-up` (or `scripts/dev-up.ps1`), which
# provides the same targets without a POSIX shell. Everyone can run the project
# with a plain `docker compose up`, which needs no Make at all.
# ---------------------------------------------------------------------------

# --- Host identity -----------------------------------------------------------
# Run bind-mounted dev services as the host developer instead of root. Derived
# from the workspace OWNER, not the current process, so invoking make inside an
# already-running bind-mounted container still sends the host identity.
# GNU stat (-c) first, then BSD/macOS stat (-f), then plain `id`. The `id`
# fallback is what makes this work everywhere `make` does.
HOST_UID ?= $(shell stat -c '%u' . 2>/dev/null || stat -f '%u' . 2>/dev/null || id -u)
HOST_GID ?= $(shell stat -c '%g' . 2>/dev/null || stat -f '%g' . 2>/dev/null || id -g)
HOST_HOME ?= $(HOME)
export HOST_UID HOST_GID HOST_HOME

# --- In-container AI agents --------------------------------------------------
# opencode, Claude Code and Codex are installed in the dev image (pinned in
# Dockerfile.template), so they run on every host OS and nothing here depends on
# a host install. Each one asks for its own login on first run — an API key you
# paste, or an account you sign in with — and keeps it in a named volume
# (docker-compose.yml), so you sign in once per project, not once per start.
#
# Every agent has a plain target and an `-auto` target. The command lines are
# variables so a project can change one without touching the recipe:
#
#   opencode --auto            approves what is not explicitly denied; the
#                              secret-file deny rules in opencode.json still apply
#   claude --dangerously-skip-permissions
#                              no prompts — the container is the boundary, as for
#                              codex-auto. Claude's Read/Edit deny rules for `.env`
#                              in .claude/settings.json still apply (verified). It
#                              asks once to accept bypass mode; the answer is kept
#                              in its volume. For a reviewed middle ground:
#                              CLAUDE_AUTO_CMD="claude --permission-mode auto"
#   codex --sandbox danger-full-access
#                              Codex's own sandbox (bubblewrap) cannot create a
#                              namespace inside an unprivileged container, so the
#                              CONTAINER is the sandbox and Codex still asks first
#   codex --dangerously-bypass-approvals-and-sandbox
#                              no prompts at all — the flag Codex documents for an
#                              externally sandboxed environment, which this is.
#                              Codex has no file deny list: in this mode nothing
#                              stops it reading `.env`. Use it where that is fine.
#
# Extra flags for one run: `make claude AGENT_ARGS="--model sonnet"`.
OPENCODE_CMD      ?= opencode
OPENCODE_AUTO_CMD ?= opencode --auto
CLAUDE_CMD        ?= claude
CLAUDE_AUTO_CMD   ?= claude --dangerously-skip-permissions
CODEX_CMD         ?= codex --sandbox danger-full-access
CODEX_AUTO_CMD    ?= codex --dangerously-bypass-approvals-and-sandbox
AGENT_ARGS        ?=

# The overlay carries only the host's gh config and ~/.gitconfig, and
# `prepare-agent-mounts` creates both sources first, so it is always merged.
COMPOSE_UP := docker compose -f docker-compose.yml -f docker-compose.agent.yml

# Python runs the agent-pin helper (`agents-upgrade`). Starting the sandbox
# never needs it. Resolved with fallbacks: a Windows Python often provides only
# `python` or the `py` launcher.
PYTHON ?= $(shell command -v python3 2>/dev/null || command -v python 2>/dev/null || command -v py 2>/dev/null)

# --- Is the dev image current? -----------------------------------------------
# `dev-up` does not rebuild, so an edited Dockerfile keeps running the old image
# until someone remembers `make dev-build`. The image therefore carries a label
# with a hash of the files it was built from, and the entry points refuse an
# image whose label does not match the files on disk.
#
# IMAGE_INPUTS lists the Dockerfile and EVERY file it COPYs. Add a file here
# when you add a COPY — `make check` compares the two. The hash is the first 16
# hex digits of SHA-256 over the files in this order; scripts/dev-up.py computes
# the same value, so both entry points agree on what "current" means.
IMAGE_INPUTS ?= Dockerfile scripts/dev-sandbox-rc.sh
IMAGE_INPUTS_HASH = $(shell cat $(IMAGE_INPUTS) 2>/dev/null | { sha256sum 2>/dev/null || shasum -a 256; } | cut -c1-16)
# Compose names a built image `<project>-<service>`; the project is `name:`.
DEV_IMAGE ?= $(shell sed -n 's/^name:[[:space:]]*//p' docker-compose.yml | head -n 1)-dev


help: ## List the available targets
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@printf '\n  \033[36m%-30s\033[0m %s\n' "docker compose up" "run the sandbox with no host prerequisites (any OS)"
	@printf '  \033[36m%-30s\033[0m %s\n' "scripts/dev-up.py" "native Windows: python scripts/dev-up.py dev-up"

# --- Preflight ---------------------------------------------------------------
# Refuses to start as root, and pre-creates the read-only mount sources the
# agent overlay needs. Missing mount sources are fatal on Windows and create
# root-owned directories on Linux, so they are created here rather than assumed.
preflight: check-identity prepare-agent-mounts

check-identity:
	@test "$(HOST_UID)" != "0" || { echo "ERROR: refusing to run the dev sandbox as UID 0; repair workspace ownership or set HOST_UID to the host user." >&2; exit 1; }

prepare-agent-mounts:
	@mkdir -p "$(HOST_HOME)/.config/gh" && touch "$(HOST_HOME)/.gitconfig"

# Refuses a dev image that was built from different files than the ones on
# disk (see IMAGE_INPUTS). A missing image is built — by `dev-build`, so it
# carries the label. A mismatch is an error, not a rebuild: `dev-up` never
# rebuilds behind your back, it tells you to.
check-image:
	@have="$$(docker image inspect --format '{{ index .Config.Labels "dev.image-inputs" }}' '$(DEV_IMAGE)' 2>/dev/null || true)"; \
	if [ -z "$$have" ]; then \
	  echo "[sandbox] no dev image yet — building it (this runs once)."; \
	  $(MAKE) --no-print-directory dev-build; \
	elif [ "$$have" != "$(IMAGE_INPUTS_HASH)" ]; then \
	  echo "ERROR: the dev image is out of date: it was not built from the current $(IMAGE_INPUTS)." >&2; \
	  echo "       (an edited Dockerfile, a bumped agent or toolchain pin, or an image built outside make dev-build)" >&2; \
	  echo "       Rebuild it: make dev-build   — agent logins are kept, they live in volumes." >&2; \
	  exit 1; \
	fi

# The value the image label is compared with. Printed for the launcher parity
# check and for debugging a mismatch.
image-inputs-hash:
	@echo "$(IMAGE_INPUTS_HASH)"

# What the RUNNING sandbox has: each agent's version and whether it is signed
# in, then (with Python) each pin next to the vendor's latest release.
agents-version: ## Show agent versions and logins in the sandbox, and the latest releases
	@test -n "$$($(COMPOSE_UP) ps -q dev 2>/dev/null)" || { echo "ERROR: the dev sandbox is not running — start it with 'make dev-up'." >&2; exit 1; }
	@$(COMPOSE_UP) exec -T dev bash /workspace/scripts/agents-status.sh
	@if [ -n "$(PYTHON)" ]; then echo; $(PYTHON) scripts/agents-pin.py show; fi

# Upgrading an agent = rewrite its pin in the Dockerfile to the vendor's latest
# release (or AGENT_VERSION=X.Y.Z for one agent), rebuild, and re-create a
# running sandbox on the new image. The pin change is an ordinary diff to
# commit, so the whole project moves together. Logins live in volumes and
# survive all of it.
AGENTS_TO_UPGRADE ?=
AGENT_VERSION ?=

agents-upgrade: preflight ## Upgrade every agent to its latest release (keeps logins)
	@test -n "$(PYTHON)" || { echo "ERROR: agents-upgrade needs python3 on the host to look up and rewrite the pins." >&2; exit 1; }
	@$(PYTHON) scripts/agents-pin.py bump $(AGENTS_TO_UPGRADE) $(if $(AGENT_VERSION),--version $(AGENT_VERSION))
	@$(MAKE) --no-print-directory dev-build
	@if [ -n "$$($(COMPOSE_UP) ps -q dev 2>/dev/null)" ]; then \
	  echo "[agents] re-creating the running sandbox on the new image (logins are kept)..."; \
	  token="$$(scripts/resolve-gh-token.sh 2>/dev/null)"; \
	  GH_TOKEN="$$token" GITHUB_TOKEN="$$token" $(COMPOSE_UP) up -d --force-recreate dev; \
	  $(MAKE) --no-print-directory agents-version; \
	fi
	@echo "[agents] commit the Dockerfile pin change so the whole project upgrades with you."

opencode-upgrade: AGENTS_TO_UPGRADE = opencode
opencode-upgrade: agents-upgrade ## Upgrade opencode in the sandbox (AGENT_VERSION=X.Y.Z to pin one)
claude-upgrade: AGENTS_TO_UPGRADE = claude
claude-upgrade: agents-upgrade ## Upgrade Claude Code in the sandbox (AGENT_VERSION=X.Y.Z to pin one)
codex-upgrade: AGENTS_TO_UPGRADE = codex
codex-upgrade: agents-upgrade ## Upgrade Codex in the sandbox (AGENT_VERSION=X.Y.Z to pin one)

# One look at everything the sandbox depends on. A failed item names its fix.
# Logins are checked by each agent's own status command (exit code only) — no
# credential is ever printed.
doctor: ## Check Docker, the dev image, the sandbox, the GitHub token and each agent's login
	@status=0; \
	ok()   { printf '  ok    %s\n' "$$1"; }; \
	bad()  { printf '  FAIL  %s\n' "$$1"; status=1; }; \
	note() { printf '  --    %s\n' "$$1"; }; \
	if docker info >/dev/null 2>&1; then ok "Docker is running"; else bad "Docker is not reachable — start Docker Desktop or the docker service"; fi; \
	have="$$(docker image inspect --format '{{ index .Config.Labels "dev.image-inputs" }}' '$(DEV_IMAGE)' 2>/dev/null || true)"; \
	if [ -z "$$have" ]; then note "no dev image yet — make dev-up builds it"; \
	elif [ "$$have" = "$(IMAGE_INPUTS_HASH)" ]; then ok "dev image is current"; \
	else bad "dev image is out of date — make dev-build"; fi; \
	if [ -n "$$(scripts/resolve-gh-token.sh 2>/dev/null)" ]; then ok "GitHub token available (gh works in the sandbox)"; \
	else note "no GitHub token — gh auth login on the host"; fi; \
	if [ -n "$$($(COMPOSE_UP) ps -q dev 2>/dev/null)" ]; then \
	  ok "sandbox is running"; \
	  $(COMPOSE_UP) exec -T dev bash /workspace/scripts/agents-status.sh || status=1; \
	else note "sandbox is not running — make dev-up (agent logins are checked while it runs)"; fi; \
	exit $$status

# --- Sandbox lifecycle -------------------------------------------------------
# HOST_UID/HOST_GID need no explicit export here: docker-compose.yml defaults
# them, and the exported values above supply the host's real identity when make
# is running on the host. Only the GitHub token is resolved at up-time.
dev-up: preflight check-image ## Start the dev sandbox (foreground; Ctrl+C stops it)
	@GH_TOKEN=$$(scripts/resolve-gh-token.sh 2>/dev/null) GITHUB_TOKEN=$$(scripts/resolve-gh-token.sh 2>/dev/null) $(COMPOSE_UP) up dev

dev-down: ## Stop the dev sandbox (keeps volumes)
	@docker compose down

dev-log: ## Tail dev sandbox logs
	@docker compose logs -f dev

dev-build: check-identity ## Rebuild the dev image after Dockerfile/dependency changes
	@docker compose build --build-arg DEV_IMAGE_INPUTS=$(IMAGE_INPUTS_HASH) dev

reset: ## Stop everything and wipe volumes (clean slate)
	@docker compose down -v

shell: dev-shell
opencode: dev-opencode
opencode-auto: dev-opencode-auto
claude: dev-claude
claude-auto: dev-claude-auto
codex: dev-codex
codex-auto: dev-codex-auto

dev-shell: preflight check-image ## Open a shell inside the dev sandbox
	@$(COMPOSE_UP) exec dev bash || $(COMPOSE_UP) run --rm dev bash

.PHONY: dev-agent dev-opencode dev-opencode-auto dev-claude dev-claude-auto dev-codex dev-codex-auto \
        opencode opencode-auto claude claude-auto codex codex-auto agents-version agents-upgrade \
        opencode-upgrade claude-upgrade codex-upgrade doctor check-image image-inputs-hash \
        preflight check-identity prepare-agent-mounts

# One recipe for every agent; the targets below only choose AGENT_CMD (a
# target-specific variable, which make passes down to this prerequisite).
#
# The sandbox is started if it is not running, WITHOUT --force-recreate: a
# running sandbox is left alone, so an agent already working in it keeps its
# session. The agent's exit status is the recipe's — no fallback re-runs it in
# another shell when it exits non-zero.
dev-agent: preflight check-image
	@test -n "$(AGENT_CMD)" || { echo "ERROR: run an agent target (make opencode, make claude-auto, ...), not dev-agent." >&2; exit 1; }
	@if [ -z "$$($(COMPOSE_UP) ps -q dev 2>/dev/null)" ]; then \
	  echo "[agent] starting the dev sandbox..."; \
	  token="$$(scripts/resolve-gh-token.sh 2>/dev/null)"; \
	  GH_TOKEN="$$token" GITHUB_TOKEN="$$token" $(COMPOSE_UP) up -d dev; fi
	@bin="$(firstword $(AGENT_CMD))"; \
	 tty="-T"; if [ -t 0 ]; then tty="-it"; fi; \
	 $(COMPOSE_UP) exec $$tty dev zsh -lc "cd /workspace && if ! command -v $$bin >/dev/null; then echo \"ERROR: $$bin is not installed in this image — add it to SANDBOX_AGENTS in the Dockerfile, then make dev-build.\" >&2; exit 127; fi; exec $(AGENT_CMD) $(AGENT_ARGS)"

dev-opencode: AGENT_CMD = $(OPENCODE_CMD)
dev-opencode: dev-agent ## Run opencode inside the dev sandbox
dev-opencode-auto: AGENT_CMD = $(OPENCODE_AUTO_CMD)
dev-opencode-auto: dev-agent ## Run opencode in auto mode inside the dev sandbox
dev-claude: AGENT_CMD = $(CLAUDE_CMD)
dev-claude: dev-agent ## Run Claude Code inside the dev sandbox
dev-claude-auto: AGENT_CMD = $(CLAUDE_AUTO_CMD)
dev-claude-auto: dev-agent ## Run Claude Code in auto mode inside the dev sandbox
dev-codex: AGENT_CMD = $(CODEX_CMD)
dev-codex: dev-agent ## Run Codex inside the dev sandbox (the container is its sandbox)
dev-codex-auto: AGENT_CMD = $(CODEX_AUTO_CMD)
dev-codex-auto: dev-agent ## Run Codex with no approval prompts inside the dev sandbox
