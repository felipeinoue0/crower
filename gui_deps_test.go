package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// scripts/gui-deps.sh is the one place the Fyne build's system packages live,
// and it answers differently per distro: fyne.yml and release.yml pipe its
// Debian names into apt-get, while a Fedora host needs dnf names and a
// developer without headers needs the install command rather than a C error.
//
// None of that is reachable from a parse-the-Makefile test and none of it runs
// in CI's own jobs, so it is exercised here instead — the distro is a file and
// pkg-config is a binary, and both are pointed at fixtures.

// aptPackages is the exact string two workflows install. It is copied here
// rather than read from the script on purpose: a test that reads the value it
// asserts cannot notice a change to it.
const aptPackages = "build-essential libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev wayland-protocols"

// runGuiDeps executes the script with the given environment overrides and
// returns its streams and exit code.
func runGuiDeps(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	cmd := exec.Command("sh", append([]string{"scripts/gui-deps.sh"}, args...)...)
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var out, errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running gui-deps.sh: %v", err)
		}
		code = exitError.ExitCode()
	}

	return strings.TrimRight(out.String(), "\n"), errOut.String(), code
}

// fakeOSRelease writes an os-release fixture and returns its path.
func fakeOSRelease(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "os-release")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing os-release fixture: %v", err)
	}

	return path
}

// fakePkgConfig writes a pkg-config stand-in with the given exit code.
func fakePkgConfig(t *testing.T, exitCode int) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "pkg-config")
	script := "#!/bin/sh\nexit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing pkg-config stub: %v", err)
	}

	return path
}

// TestGuiDepsListMatchesTheAptContract pins the Debian list exactly.
//
// fyne.yml and release.yml run `sudo apt-get install -y $(make -s gui-deps)`,
// so a reordered or renamed package here is a CI break that no Go test would
// otherwise see — the workflows are not executed by make check.
func TestGuiDepsListMatchesTheAptContract(t *testing.T) {
	osRelease := fakeOSRelease(t, "ID=ubuntu\nID_LIKE=debian\n")

	stdout, stderr, code := runGuiDeps(t, map[string]string{"OS_RELEASE": osRelease}, "list")
	if code != 0 {
		t.Fatalf("list on ubuntu exited %d, stderr: %s", code, stderr)
	}
	if stdout != aptPackages {
		t.Errorf("list on ubuntu = %q, want the exact apt list %q", stdout, aptPackages)
	}
}

// TestGuiDepsListPerDistro covers the branch a developer on another distro
// relies on: same script, that distro's package names.
func TestGuiDepsListPerDistro(t *testing.T) {
	tests := []struct {
		name      string
		osRelease string
		wantSubs  []string
		wantCode  int
	}{
		{
			name:      "debian",
			osRelease: "ID=debian\n",
			wantSubs:  []string{"build-essential", "libwayland-dev"},
		},
		{
			name:      "fedora",
			osRelease: "ID=fedora\n",
			wantSubs:  []string{"wayland-devel", "libxkbcommon-devel", "pkgconf-pkg-config"},
		},
		{
			name:      "rhel family via ID_LIKE",
			osRelease: "ID=rocky\nID_LIKE=\"rhel centos fedora\"\n",
			wantSubs:  []string{"wayland-devel", "libXxf86vm-devel"},
		},
		{
			name:      "arch",
			osRelease: "ID=arch\n",
			wantSubs:  []string{"wayland", "libxkbcommon", "libxxf86vm"},
		},
		{
			name:      "unknown distro is an error, not a guess",
			osRelease: "ID=plan9\n",
			wantCode:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osRelease := fakeOSRelease(t, tt.osRelease)

			stdout, stderr, code := runGuiDeps(t, map[string]string{"OS_RELEASE": osRelease}, "list")
			if code != tt.wantCode {
				t.Fatalf("list exited %d, want %d (stderr: %s)", code, tt.wantCode, stderr)
			}

			if tt.wantCode != 0 {
				if stdout != "" {
					t.Errorf("list on an unknown distro wrote %q to stdout; a workflow piping it would install nonsense", stdout)
				}
				if !strings.Contains(stderr, "unrecognised distribution") {
					t.Errorf("stderr = %q, want it to say the distribution is unrecognised", stderr)
				}
				return
			}

			for _, want := range tt.wantSubs {
				if !strings.Contains(stdout, want) {
					t.Errorf("list = %q, want it to contain %q", stdout, want)
				}
			}
		})
	}
}

// TestGuiDepsCommandMatchesList keeps the printed install command and the bare
// list from drifting: `command` is what a developer is told to run, `list` is
// what CI installs, and they must name the same packages.
func TestGuiDepsCommandMatchesList(t *testing.T) {
	osRelease := fakeOSRelease(t, "ID=fedora\n")

	list, _, code := runGuiDeps(t, map[string]string{"OS_RELEASE": osRelease}, "list")
	if code != 0 {
		t.Fatalf("list exited %d", code)
	}

	command, stderr, code := runGuiDeps(t, map[string]string{"OS_RELEASE": osRelease}, "command")
	if code != 0 {
		t.Fatalf("command exited %d, stderr: %s", code, stderr)
	}

	if !strings.HasPrefix(command, "sudo dnf install -y ") {
		t.Errorf("command = %q, want it to start with the dnf install line", command)
	}
	if !strings.HasSuffix(command, list) {
		t.Errorf("command = %q, want the list %q as its arguments", command, list)
	}
}

// TestGuiDepsCheckFailsWithTheInstallCommand is the property this target
// exists for: a missing header must produce a package manager command, not
// "wayland-client-core.h: No such file or directory" at the end of a compile.
func TestGuiDepsCheckFailsWithTheInstallCommand(t *testing.T) {
	osRelease := fakeOSRelease(t, "ID=fedora\n")
	pkgConfig := fakePkgConfig(t, 1)

	stdout, stderr, code := runGuiDeps(t, map[string]string{
		"OS_RELEASE": osRelease,
		"PKG_CONFIG": pkgConfig,
	}, "check")

	if code != 1 {
		t.Fatalf("check with missing headers exited %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("check wrote %q to stdout, want nothing", stdout)
	}
	for _, want := range []string{"missing development headers", "wayland-client", "sudo dnf install -y"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to contain %q", stderr, want)
		}
	}
}

// TestGuiDepsCheckPassesWhenHeadersResolve proves the check can pass — a gate
// that always fails is as useless as one that always passes.
func TestGuiDepsCheckPassesWhenHeadersResolve(t *testing.T) {
	osRelease := fakeOSRelease(t, "ID=fedora\n")
	pkgConfig := fakePkgConfig(t, 0)

	stdout, stderr, code := runGuiDeps(t, map[string]string{
		"OS_RELEASE": osRelease,
		"PKG_CONFIG": pkgConfig,
	}, "check")

	if code != 0 {
		t.Fatalf("check with resolved headers exited %d, want 0 (stderr: %s)", code, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("check wrote stdout=%q stderr=%q on success, want silence", stdout, stderr)
	}
}

// TestGuiDepsCheckWithoutPkgConfig covers the other way the precheck can fail:
// pkg-config itself absent, which would otherwise read as "everything is
// missing" with no clue why.
func TestGuiDepsCheckWithoutPkgConfig(t *testing.T) {
	osRelease := fakeOSRelease(t, "ID=fedora\n")

	_, stderr, code := runGuiDeps(t, map[string]string{
		"OS_RELEASE": osRelease,
		"PKG_CONFIG": filepath.Join(t.TempDir(), "no-such-pkg-config"),
	}, "check")

	if code != 1 {
		t.Fatalf("check without pkg-config exited %d, want 1", code)
	}
	if !strings.Contains(stderr, "is not installed") {
		t.Errorf("stderr = %q, want it to say pkg-config is not installed", stderr)
	}
}

// TestGuiDepsScriptIsRunnable guards the property `make` relies on: the shebang
// and the exec bit. A checkout that loses the mode fails every target that
// calls the script, and a Go test that only ran it through `sh` would not
// notice.
func TestGuiDepsScriptIsRunnable(t *testing.T) {
	info, err := os.Stat("scripts/gui-deps.sh")
	if err != nil {
		t.Fatalf("stat scripts/gui-deps.sh: %v", err)
	}

	if info.Mode()&0o111 == 0 {
		t.Errorf("scripts/gui-deps.sh is not executable (mode %v); make would fail to run it", info.Mode())
	}

	body := readRepositoryFile(t, "scripts/gui-deps.sh")
	if !strings.HasPrefix(body, "#!/bin/sh") {
		t.Errorf("scripts/gui-deps.sh does not start with a sh shebang; it is executed by make, not sourced")
	}
}

// TestGuiDepsDefaultsToList keeps `gui-deps.sh` with no argument behaving like
// `list`, which is the mode `make -s gui-deps` relies on.
func TestGuiDepsDefaultsToList(t *testing.T) {
	osRelease := fakeOSRelease(t, "ID=debian\n")

	stdout, stderr, code := runGuiDeps(t, map[string]string{"OS_RELEASE": osRelease})
	if code != 0 {
		t.Fatalf("no-argument run exited %d, stderr: %s", code, stderr)
	}
	if stdout != aptPackages {
		t.Errorf("no-argument run = %q, want the apt list", stdout)
	}
}

// TestGuiDepsRejectsUnknownMode keeps a typo in the Makefile loud instead of
// silently doing nothing.
func TestGuiDepsRejectsUnknownMode(t *testing.T) {
	_, stderr, code := runGuiDeps(t, nil, "instal")

	if code == 0 {
		t.Fatal("an unknown mode exited 0; a typo in the Makefile would print nothing and report success")
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("stderr = %q, want a usage line", stderr)
	}
}
