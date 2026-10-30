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

// fakeClaude puts on PATH a claude that runs script, and a run to call it in.
func fakeClaude(t *testing.T, script string) Request {
	t.Helper()
	bin, dir, run := t.TempDir(), t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(run, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, text := range map[string]string{filepath.Join(dir, "instruction.md"): "Code.", filepath.Join(run, "in", "task.md"): "The issue."} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Request{RunDir: run, Repo: t.TempDir(), MaxTurns: 12,
		Role: &role.Role{Name: "developer", Dir: dir, Intentions: []string{"patch"}, Model: role.Model{Timeout: "5s"}}}
}

// claude, worktree or not, is killed whole once it answered; stopped at
// its --max-turns with no answer written, it is a cap reached, its
// answer none.
func TestClaudeMaxTurnsKilledWhole(t *testing.T) {
	pid := filepath.Join(t.TempDir(), "pid")
	req := fakeClaude(t, `cat >/dev/null; sleep 30 >/dev/null 2>&1 & echo $! > `+pid+`
echo '{"subtype":"error_max_turns","is_error":true,"num_turns":12,"result":""}'; exit 1`)
	call, err := claude{}.Propose(req)
	if err != nil {
		t.Fatalf("err = %v, want a cap reached, not a failure", err)
	}
	if call.Turns != 12 {
		t.Errorf("turns = %d, want 12", call.Turns)
	}
	if got, _ := os.ReadFile(filepath.Join(req.RunDir, "out", "intentions.yaml")); strings.TrimSpace(string(got)) != "[]" {
		t.Errorf("intentions = %q, want none", got)
	}
	data, err := os.ReadFile(pid)
	if err != nil {
		t.Fatal(err)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(child, 0); err == nil {
		_ = syscall.Kill(child, syscall.SIGKILL)
		t.Errorf("its child %d still runs: claude's group was not killed", child)
	}
}

// Its max turns said in a stdout cut short, or by a claude killed, is a
// failure: nothing proves it stopped at its cap.
func TestClaudeMaxTurnsPartialIsAFailure(t *testing.T) {
	for name, script := range map[string]string{
		"cut short": `cat >/dev/null; printf '{"subtype":"error_max_turns","num_turns":12,"res'; exit 1`,
		"killed":    `cat >/dev/null; echo '{"subtype":"error_max_turns","is_error":true,"num_turns":12}'; kill -9 $$`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := (claude{}).Propose(fakeClaude(t, script)); !errors.Is(err, ErrUnavailable) {
				t.Errorf("err = %v, want unavailable", err)
			}
		})
	}
}
