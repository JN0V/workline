package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// Raw output not posted: Given an agent in a worktree whose output is not
// Claude Code's JSON (no result message, a file's contents in it), When the
// developer opens its pull request, Then none of that output is posted (not
// in the closing words kept for a person, nor in what is proposed), and the
// run says the agent's words could not be read.
func TestRawOutputNotPosted(t *testing.T) {
	bin := t.TempDir()
	fake := "#!/bin/sh\ncat > /dev/null\necho 'API_PASSWORD=hunter2-from-a-file'\necho 'not json at all'\n"
	must(t, os.WriteFile(filepath.Join(bin, "claude"), []byte(fake), 0o755))
	for _, tool := range claudeShellTools {
		must(t, os.WriteFile(filepath.Join(bin, tool), []byte("#!/bin/sh\n"), 0o755))
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir, runDir := t.TempDir(), t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Code."), 0o644))
	for _, d := range []string{"in", "out", WorktreeDir} {
		must(t, os.MkdirAll(filepath.Join(runDir, d), 0o755))
	}
	must(t, os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("The issue."), 0o644))
	call, err := claude{}.Propose(Request{RunDir: runDir, Repo: t.TempDir(), MaxTurns: 7,
		Role: &role.Role{Name: "developer", Dir: dir, Intentions: []string{"patch", "comment"}}})
	var all []string
	all = append(all, call.Mended...)
	if err != nil {
		all = append(all, err.Error())
		if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("the error holds the raw output: %v", err)
		}
	}
	for _, f := range []string{ClosingUnread, "intentions.yaml"} {
		data, rerr := os.ReadFile(filepath.Join(runDir, "out", f))
		if rerr != nil {
			continue
		}
		if strings.Contains(string(data), "hunter2") {
			t.Errorf("%s holds the raw output: %q", f, data)
		}
		if f == ClosingUnread {
			all = append(all, string(data))
		}
	}
	if !strings.Contains(strings.Join(all, "\n"), "could not be read") {
		t.Errorf("the run does not say the agent's words could not be read: %q", all)
	}
}
