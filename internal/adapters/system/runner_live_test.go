//go:build !windows

package system

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// A live runner hands the command its streams: stdin reaches it, its output is
// written where it happens rather than returned, and stderr stays separate.
func TestLiveRunnerAttachesTheStreams(t *testing.T) {
	var out, errOut bytes.Buffer

	runner := NewLiveRunner(Streams{In: strings.NewReader("Ann\n"), Out: &out, Err: &errOut})

	got, err := runner.Run(`read n; echo "hi $n"; echo oops >&2`, Target{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got != "" {
		t.Errorf("Run returned %q; a live run has already written its output", got)
	}

	if out.String() != "hi Ann\n" {
		t.Errorf("stdout = %q, want %q", out.String(), "hi Ann\n")
	}

	if errOut.String() != "oops\n" {
		t.Errorf("stderr = %q, want %q", errOut.String(), "oops\n")
	}
}

// The capturing runner, which the window and the terminal UI use, still returns
// the output.
func TestCapturingRunnerReturnsTheOutput(t *testing.T) {
	got, err := NewRunner().Run("echo captured", Target{})
	if err != nil || got != "captured\n" {
		t.Errorf("Run = %q, %v; want %q", got, err, "captured\n")
	}
}

// A timed command runs in its own process group, so it is a background job to
// the terminal. It must not get stdin -- a read would stop it with SIGTTIN and
// hang the run -- and its timeout still applies.
func TestLiveTimedCommandGetsNoStdin(t *testing.T) {
	var out bytes.Buffer

	runner := NewLiveRunner(Streams{In: strings.NewReader("should-not-arrive\n"), Out: &out, Err: &out})

	// The error is not the point: read on a closed stdin fails, and the echo
	// after it still runs. What matters is what read saw, and how long it took.
	start := time.Now()
	_, _ = runner.Run(`read n; echo "got:$n"`, Target{Timeout: "5s"})

	if strings.Contains(out.String(), "should-not-arrive") {
		t.Errorf("a timed command read stdin: %q", out.String())
	}

	if time.Since(start) > 4*time.Second {
		t.Errorf("a timed command waited on stdin for %s", time.Since(start))
	}
}
