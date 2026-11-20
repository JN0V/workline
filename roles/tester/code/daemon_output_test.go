//go:build !windows

package tester

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// Daemon holding the output
// Given a test command `echo ready; (sleep 30 &) ; exit 0` whose background
// sleep keeps the command's output open,
// When the tester runs it,
// Then the run returns in under 2 seconds and the test is reported as
// passed with no error.
func TestDaemonHoldingTheOutput(t *testing.T) {
	start := time.Now()
	out, late, err := shell(t.TempDir(), "echo ready; (sleep 30 &) ; exit 0", time.Time{})
	took := time.Since(start)
	if took >= 2*time.Second {
		t.Errorf("the run took %s, want under 2s", took)
	}
	if err != nil {
		t.Errorf("err = %v, want none: the command exited 0", err)
	}
	if late {
		t.Error("late = true, want false: no deadline was set")
	}
	if !strings.Contains(out.String(), "ready") {
		t.Errorf("output %q lost what the command printed", out.String())
	}
}

// Output kept and masked
// Given a test command that prints `before-exit` and the line
// `token=SECRET123` (a secret the project masks), starts a background daemon
// sharing its output, and exits 0,
// When the tester runs it,
// Then the reported output holds `before-exit` exactly once, and the secret
// is masked as forge.Mask renders it.
func TestDaemonRunOutputKeptAndMasked(t *testing.T) {
	t.Setenv("GH_TOKEN", "SECRET123")
	start := time.Now()
	out, _, err := shell(t.TempDir(), "echo before-exit; echo token=SECRET123; (sleep 30 &) ; exit 0", time.Time{})
	took := time.Since(start)
	if took >= 2*time.Second {
		t.Errorf("the run took %s, want under 2s", took)
	}
	if err != nil {
		t.Errorf("err = %v, want none: the command exited 0", err)
	}
	got := out.String()
	if n := strings.Count(got, "before-exit"); n != 1 {
		t.Errorf("before-exit appears %d times in %q, want once", n, got)
	}
	if strings.Contains(got, "SECRET123") {
		t.Errorf("the secret is not masked: %q", got)
	}
	if want := forge.Mask("token=SECRET123"); !strings.Contains(got, want) {
		t.Errorf("output %q lacks %q, the secret's line as forge.Mask renders it", got, want)
	}
}

// Output after the kill is dropped
// Given a test command that starts a background daemon which waits 3
// seconds and then prints `late` to the shared output, and the command exits
// 0 at once,
// When the tester runs it and returns in under 2 seconds,
// Then the reported output does not hold `late`, and the daemon is no longer
// running.
func TestDaemonOutputAfterTheKillDropped(t *testing.T) {
	dir := t.TempDir()
	pid := filepath.Join(dir, "pid")
	start := time.Now()
	out, _, _ := shell(dir, "(sleep 3; echo late) & echo $! > "+pid+"; exit 0", time.Time{})
	took := time.Since(start)
	if took >= 2*time.Second {
		t.Errorf("the run took %s, want under 2s", took)
	}
	if strings.Contains(out.String(), "late") {
		t.Errorf("output %q holds what the daemon printed after the kill", out.String())
	}
	data, err := os.ReadFile(pid)
	if err != nil {
		t.Fatal(err)
	}
	daemon, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.Kill(daemon, syscall.SIGKILL) }()
	gone := false
	for end := time.Now().Add(time.Second); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
		if !alive(daemon) {
			gone = true
			break
		}
	}
	if !gone {
		t.Errorf("the daemon %d still runs after the run returned", daemon)
	}
}

// Failing command with a daemon
// Given a test command that starts a background daemon sharing its output
// and exits 1,
// When the tester runs it,
// Then it returns in under 2 seconds and the test is reported as failed.
func TestFailingCommandWithADaemon(t *testing.T) {
	start := time.Now()
	_, _, err := shell(t.TempDir(), "echo failing; (sleep 30 &) ; exit 1", time.Time{})
	took := time.Since(start)
	if took >= 2*time.Second {
		t.Errorf("the run took %s, want under 2s", took)
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Errorf("err = %v, want the command's exit status 1", err)
	}
}
