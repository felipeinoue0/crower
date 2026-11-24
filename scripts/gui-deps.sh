#!/bin/sh
# The system packages the Fyne window needs on Linux — in one place, for every
# caller: `make gui_linux` (so a missing header is an install command, not a C
# error), `make gui-deps`, and fyne.yml / release.yml, which consume
# `make -s gui-deps` and must keep seeing the Debian names on an Ubuntu runner.
#
# The package lists and the pkg-config module list are separate on purpose: a
# distro package can carry several modules (wayland-devel ships wayland-client,
# wayland-cursor and wayland-egl), and a module can be missing while its
# package looks installed. `check` asks pkg-config, because pkg-config is what
# the CGO link actually resolves.
#
# Usage:
#   gui-deps.sh list       the package list for this distro (stdout, one line)
#   gui-deps.sh command    the install command, printed and never run
#   gui-deps.sh install    run that command (asks for sudo)
#   gui-deps.sh check      fail if a pkg-config module the link needs is missing
#
# OS_RELEASE overrides the file the distro is read from and PKG_CONFIG the
# pkg-config binary; the tests use both, so neither has to be faked globally.

set -eu

OS_RELEASE=${OS_RELEASE:-/etc/os-release}
PKG_CONFIG=${PKG_CONFIG:-pkg-config}

# The Debian list is the canonical one: two workflows pipe it into apt-get, so
# its order and content are a contract — TestGuiDepsListMatchesTheAptContract
# pins it.
APT_PACKAGES="build-essential libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev wayland-protocols"
DNF_PACKAGES="gcc pkgconf-pkg-config mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel wayland-devel libxkbcommon-devel wayland-protocols-devel"
PACMAN_PACKAGES="base-devel pkgconf mesa libx11 libxcursor libxrandr libxinerama libxi libxxf86vm wayland libxkbcommon wayland-protocols"

# What `-tags fyne` links on Linux, taken from go-gl/glfw's #cgo lines:
#   -lX11 -lXrandr -lXxf86vm -lXi -lXcursor -lXinerama
#   -lwayland-client -lwayland-cursor -lwayland-egl -lxkbcommon -lGL
# plus wayland-protocols for the generated protocol code. Leaving a module out
# of this list does not remove the need for it, it only moves the failure back
# to a linker error — so this tracks the link line, not the apt list.
PC_MODULES="x11 xcursor xinerama xi xxf86vm xrandr gl wayland-client wayland-cursor wayland-egl xkbcommon wayland-protocols"

# Reads ID and ID_LIKE rather than probing for a package manager: `dnf` exists
# on a Debian box that happens to have it installed, and the answer we want is
# which distro this is, not which binary is on PATH.
detect_distro() {
    id=""
    like=""

    if [ -r "$OS_RELEASE" ]; then
        id=$(sed -n 's/^ID=//p' "$OS_RELEASE" | head -n 1 | tr -d '"')
        like=$(sed -n 's/^ID_LIKE=//p' "$OS_RELEASE" | head -n 1 | tr -d '"')
    fi

    case " $id $like " in
        *debian* | *ubuntu*) echo apt ;;
        *fedora* | *rhel* | *centos* | *rocky* | *alma*) echo dnf ;;
        *arch* | *manjaro*) echo pacman ;;
        *) echo unknown ;;
    esac
}

DISTRO=$(detect_distro)

packages() {
    case "$DISTRO" in
        apt) echo "$APT_PACKAGES" ;;
        dnf) echo "$DNF_PACKAGES" ;;
        pacman) echo "$PACMAN_PACKAGES" ;;
        *) return 1 ;;
    esac
}

install_command() {
    pkgs=$(packages) || return 1

    case "$DISTRO" in
        apt) echo "sudo apt-get update && sudo apt-get install -y $pkgs" ;;
        dnf) echo "sudo dnf install -y $pkgs" ;;
        pacman) echo "sudo pacman -S --needed --noconfirm $pkgs" ;;
    esac
}

unknown_hint() {
    echo "gui-deps: unrecognised distribution in $OS_RELEASE." >&2
    echo "          Install the X11/GL and Wayland development headers by hand;" >&2
    echo "          the three package lists at the top of $0 are the set to mirror." >&2
}

hint_install() {
    if cmd=$(install_command 2>/dev/null); then
        echo "          Install them with: $cmd" >&2
    else
        unknown_hint
    fi
}

# Exits non-zero when a module is missing, so `make gui_linux` stops before the
# compiler does — with a package manager command instead of
# "wayland-client-core.h: No such file or directory".
check_headers() {
    if ! command -v "$PKG_CONFIG" >/dev/null 2>&1; then
        echo "gui-deps: $PKG_CONFIG is not installed, so the headers cannot be checked." >&2
        hint_install
        return 1
    fi

    missing=""
    for module in $PC_MODULES; do
        if ! "$PKG_CONFIG" --exists "$module" 2>/dev/null; then
            missing="$missing $module"
        fi
    done

    if [ -n "$missing" ]; then
        echo "gui-deps: missing development headers:$missing" >&2
        hint_install
        return 1
    fi
}

case "${1:-list}" in
    list)
        if ! packages; then
            unknown_hint
            exit 2
        fi
        ;;
    command)
        if ! install_command; then
            unknown_hint
            exit 2
        fi
        ;;
    install)
        if ! cmd=$(install_command); then
            unknown_hint
            exit 2
        fi
        # Echoed first: the sudo prompt gives no other clue what is about to run.
        echo "+ $cmd" >&2
        sh -c "$cmd"
        ;;
    check)
        check_headers || exit 1
        ;;
    *)
        echo "usage: $0 [list|command|install|check]" >&2
        exit 64
        ;;
esac
