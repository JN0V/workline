package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// must fails the test on a setup error.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// TestClaudeToolsOnlyInAWorktree: a role with no worktree keeps claude
// without tools, outside the repository; given one, claude runs there
// with its tools, none asking a person, and closing words that are no
// list read as nothing proposed, said, never a failure (ADR-0056).
func TestClaudeToolsOnlyInAWorktree(t *testing.T) {
	bin, dump := t.TempDir(), filepath.Join(t.TempDir(), "args")
	fake := "#!/bin/sh\n{ pwd; for a in \"$@\"; do echo \"[$a]\"; done; } > \"$WORKLINE_ARGS_DUMP\"\ncat > /dev/null\n" +
		"echo '{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"tool_use\",\"name\":\"Edit\"}]}}'\n" +
		"echo '{\"type\":\"result\",\"num_turns\":2,\"result\":\"Done: mul written.\",\"modelUsage\":{}}'\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tool := range claudeShellTools {
		must(t, os.WriteFile(filepath.Join(bin, tool), []byte("#!/bin/sh\n"), 0o755))
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("WORKLINE_ARGS_DUMP", dump)
	run := func(tree bool) (Call, string, error) {
		dir, runDir := t.TempDir(), t.TempDir()
		must(t, os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Code."), 0o644))
		for _, d := range []string{"in", "out"} {
			must(t, os.MkdirAll(filepath.Join(runDir, d), 0o755))
		}
		if tree {
			must(t, os.MkdirAll(filepath.Join(runDir, WorktreeDir), 0o755))
		}
		must(t, os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("The issue."), 0o644))
		call, err := claude{}.Propose(Request{RunDir: runDir, Repo: t.TempDir(), MaxTurns: 7,
			Role: &role.Role{Name: "developer", Dir: dir, Intentions: []string{"patch", "comment"}}})
		args, _ := os.ReadFile(dump)
		return call, string(args), err
	}
	if _, args, err := run(false); !errors.Is(err, ErrInvalidOutput) || !strings.Contains(args, "[--tools]\n[]\n") || strings.Contains(args, "dontAsk") {
		t.Errorf("no worktree: want no tools and prose refused, got %v:\n%s", err, args)
	}
	call, args, err := run(true)
	if err != nil {
		t.Fatalf("in a worktree, prose closing words failed the call: %v", err)
	}
	for _, want := range []string{"/" + WorktreeDir + "\n", "[--tools]\n[" + claudeTools + "]", "[--allowedTools]\n[" + claudeTools + "]",
		"[--permission-mode]\n[dontAsk]", "[--max-turns]\n[7]", "[--append-system-prompt]", "[stream-json]", "[--strict-mcp-config]"} {
		if !strings.Contains(args, want) {
			t.Errorf("in a worktree, claude lacks %q:\n%s", want, args)
		}
	}
	if strings.Contains(args, "[--system-prompt]") {
		t.Errorf("in a worktree, Claude Code's own prompt replaced:\n%s", args)
	}
	if call.Turns != 2 || call.Tools["Edit"] != 1 {
		t.Errorf("in a worktree: turns or tools not recorded: %+v", call)
	}
}

// TestClaudeShellToolsSaid: on Linux, an agent in a worktree whose shell
// lacks bubblewrap or socat is not asked: the agent unavailable, what is
// missing named, rather than code no command ran (#900). Elsewhere, or
// with no worktree, nothing is asked of the runner.
func TestClaudeShellToolsSaid(t *testing.T) {
	bin := t.TempDir()
	must(t, os.WriteFile(filepath.Join(bin, "bwrap"), []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", bin)
	err := shellTools("/run/worktree", "linux")
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), ", socat not installed") {
		t.Errorf("socat missing on Linux: %v", err)
	}
	if err := shellTools("", "linux"); err != nil {
		t.Errorf("no worktree, no shell: %v", err)
	}
	if err := shellTools("/run/worktree", "darwin"); err != nil {
		t.Errorf("not Linux: %v", err)
	}
	must(t, os.WriteFile(filepath.Join(bin, "socat"), []byte("#!/bin/sh\n"), 0o755))
	if err := shellTools("/run/worktree", "linux"); err != nil {
		t.Errorf("both installed: %v", err)
	}
}

// TestPromptInAWorktreeAsksForEdits: given a worktree, the answer's
// contract asks for edits, then a list without patches.
func TestPromptInAWorktreeAsksForEdits(t *testing.T) {
	dir, runDir := t.TempDir(), t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Code."), 0o644))
	must(t, os.MkdirAll(filepath.Join(runDir, "in"), 0o755))
	must(t, os.MkdirAll(filepath.Join(runDir, WorktreeDir), 0o755))
	must(t, os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("The issue."), 0o644))
	_, user, err := Prompt(Request{RunDir: runDir, Repo: t.TempDir(), Role: &role.Role{Name: "developer", Dir: dir, Intentions: []string{"patch", "comment"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(user, "edit its files there and run its tests") || strings.Contains(user, "- patch:") || !strings.Contains(user, "- comment:") {
		t.Errorf("the worktree's answer contract:\n%s", user)
	}
}
