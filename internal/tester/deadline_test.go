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

// One run stops at suite-max-seconds, its proof's deadline far: the
// finding names the change's test file no report holds yet, and the
// command's last lines (#1425).
func TestRunStopsAtSuiteMaxAndSaysWhere(t *testing.T) {
	tree := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tree, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	done := `<testsuite name="app.DoneTest"><testcase classname="app.DoneTest" name="ends"/></testsuite>`
	s := Settings{Report: "out/*.xml", SuiteMaxSeconds: 1,
		Command:  "echo '" + done + "' > out/TEST-done.xml; echo 'HangTest > waits STARTED'; sleep 30",
		deadline: time.Now().Add(time.Hour), budget: "curator-proof-seconds",
		tests:    []string{"app/src/test/DoneTest.kt", "app/src/test/HangTest.kt"}}
	start := time.Now()
	_, f := runIn(tree, s, "on the head")
	if f == nil || f.Rule != "proof-over-time" || f.Where != "suite-max-seconds" || time.Since(start) > 10*time.Second {
		t.Fatalf("finding = %+v after %s, want proof-over-time at suite-max-seconds within its second", f, time.Since(start))
	}
	for _, want := range []string{"no report written by then holds `app/src/test/HangTest.kt`", "HangTest > waits STARTED"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("message %q lacks %q", f.Message, want)
		}
	}
	if strings.Contains(f.Message, "DoneTest.kt") {
		t.Errorf("message %q names a file its report holds", f.Message)
	}
}
