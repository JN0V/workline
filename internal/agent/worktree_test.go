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
	if call.Turns != 2 || call.Tools["Edit"] != 1 || len(call.Mended) == 0 {
		t.Errorf("in a worktree: turns, tools or the mend not recorded: %+v", call)
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
