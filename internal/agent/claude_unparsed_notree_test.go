package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// Raw output not posted, no worktree (#1245): Given an agent with no
// worktree (a read tree, or no tools at all) whose output is not Claude
// Code's JSON (no result message, a file's contents in it), When its
// answer is read, Then none of that output is kept as its answer nor said
// in the error, and the run says the agent's words could not be read.
func TestRawOutputNotPostedWithoutWorktree(t *testing.T) {
	for _, tc := range []struct {
		name string
		tree string // the run's folder the agent works in, "" for none
	}{
		{"read tree", ReadTreeDir},
		{"no tools", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			fake := "#!/bin/sh\ncat > /dev/null\necho 'API_PASSWORD=hunter2-from-a-file'\necho 'not json at all'\n"
			must(t, os.WriteFile(filepath.Join(bin, "claude"), []byte(fake), 0o755))
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			dir, runDir := t.TempDir(), t.TempDir()
			must(t, os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Read."), 0o644))
			dirs := []string{"in", "out"}
			if tc.tree != "" {
				dirs = append(dirs, tc.tree)
			}
			for _, d := range dirs {
				must(t, os.MkdirAll(filepath.Join(runDir, d), 0o755))
			}
			must(t, os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("The issue."), 0o644))
			_, err := claude{}.Propose(Request{RunDir: runDir, Repo: t.TempDir(),
				Role: &role.Role{Name: "product-owner", Dir: dir, Intentions: []string{"comment"}}})
			if err == nil {
				t.Fatal("an unreadable output was read as an answer")
			}
			if !errors.Is(err, ErrInvalidOutput) {
				t.Errorf("not an answer unread: %v", err)
			}
			if strings.Contains(err.Error(), "hunter2") {
				t.Errorf("the error holds the raw output: %v", err)
			}
			if !strings.Contains(err.Error(), "could not be read") {
				t.Errorf("the error does not say the agent's words could not be read: %v", err)
			}
			for _, f := range []string{"agent-answer.txt", "intentions.yaml"} {
				if data, rerr := os.ReadFile(filepath.Join(runDir, "out", f)); rerr == nil && strings.Contains(string(data), "hunter2") {
					t.Errorf("%s holds the raw output: %q", f, data)
				}
			}
		})
	}
}
