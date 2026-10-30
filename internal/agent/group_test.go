//go:build !windows

package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/role"
)

// A cmd: agent working in a worktree past its time is killed whole
// (ADR-0056): a child it started, left running, would edit the worktree
// while the engine bounds it. It is told its caps; the turns it reports
// are read.
func TestWorktreeAgentKilledWholeAtItsTime(t *testing.T) {
	dir, run := t.TempDir(), t.TempDir()
	for _, d := range []string{"in", "out", WorktreeDir} {
		if err := os.MkdirAll(filepath.Join(run, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, text := range map[string]string{filepath.Join(dir, "instruction.md"): "Code.", filepath.Join(run, "in", "task.md"): "The issue."} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pid, caps := filepath.Join(t.TempDir(), "pid"), filepath.Join(t.TempDir(), "caps")
	script := `cat >/dev/null; echo "$WORKLINE_MAX_TURNS $WORKLINE_MAX_SECONDS" > ` + caps +
		`; printf 'turns: 4\n' > "$WORKLINE_CALL"; sleep 30 & echo $! > ` + pid + `; wait`
	req := Request{RunDir: run, Repo: t.TempDir(), MaxTurns: 12,
		Role: &role.Role{Name: "developer", Dir: dir, Intentions: []string{"patch"}, Model: role.Model{Timeout: "1s"}}}
	start := time.Now()
	call, err := command{script: script}.Propose(req)
	if !errors.Is(err, ErrUnavailable) || time.Since(start) > 5*time.Second {
		t.Fatalf("err = %v after %s, want unavailable at its 1 s", err, time.Since(start))
	}
	if call.Turns != 4 {
		t.Errorf("turns = %d, want the 4 it reported", call.Turns)
	}
	if told, _ := os.ReadFile(caps); strings.TrimSpace(string(told)) != "12 1" {
		t.Errorf("caps told = %q, want WORKLINE_MAX_TURNS 12, WORKLINE_MAX_SECONDS 1", told)
	}
	data, err := os.ReadFile(pid)
	if err != nil {
		t.Fatal(err)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(child, 0); err == nil {
		_ = syscall.Kill(child, syscall.SIGKILL)
		t.Errorf("its child %d still runs: the agent's group was not killed", child)
	}
}
