//go:build !windows

package tester

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A test command that hangs ends at its deadline, its children with it,
// a build daemon's workers too; the finding names the setting it came
// of (#1197).
func TestTestCommandStopsWholeAtItsDeadline(t *testing.T) {
	dir := t.TempDir()
	pid := filepath.Join(dir, "pid")
	start := time.Now()
	_, late, _ := shell(dir, "sleep 30 & echo $! > "+pid+"; wait", time.Now().Add(300*time.Millisecond))
	if !late || time.Since(start) > 5*time.Second {
		t.Fatalf("late = %v after %s, want stopped at its 300 ms", late, time.Since(start))
	}
	data, err := os.ReadFile(pid)
	if err != nil {
		t.Fatal(err)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(child, 0); err == nil {
		_ = syscall.Kill(child, syscall.SIGKILL)
		t.Errorf("its child %d still runs: the command's group was not killed", child)
	}
	if f := overTime("on the base", "suite-max-seconds"); f.Rule != "proof-over-time" || f.Where != "suite-max-seconds" {
		t.Errorf("finding = %+v, want proof-over-time at suite-max-seconds", f)
	}
}
