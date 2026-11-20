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

// alive reports whether the process still runs. A killed child may linger
// as a zombie until reaped: where /proc exists, a zombie does not count.
func alive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		return err == syscall.EPERM
	}
	if st, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil && strings.Contains(string(st), ") Z ") {
		return false
	}
	return true
}

// Nothing left
// Given a test command that starts a long-lived child (a build daemon),
// When the command ends and the run with it, with a deadline far off or none,
// Then the child is gone within a few seconds: no process from it is left.
func TestTestCommandLeavesNoChildBehind(t *testing.T) {
	for name, deadline := range map[string]time.Time{
		"no deadline":  {},
		"far deadline": time.Now().Add(time.Hour),
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			pid := filepath.Join(dir, "pid")
			if _, _, err := shell(dir, "sleep 60 >/dev/null 2>&1 </dev/null & echo $! > "+pid, deadline); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(pid)
			if err != nil {
				t.Fatal(err)
			}
			child, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = syscall.Kill(child, syscall.SIGKILL) }()
			gone := false
			for end := time.Now().Add(3 * time.Second); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
				if !alive(child) {
					gone = true
					break
				}
			}
			if !gone {
				t.Errorf("the child %d the command started still runs after the run ended", child)
			}
		})
	}
}
