package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// readTreeRun sets a fake claude printing script on PATH and a run folder
// with a read tree, and returns the folder and Propose's error.
func readTreeRun(t *testing.T, script string, tree bool) (string, error) {
	t.Helper()
	bin := t.TempDir()
	must(t, os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\ncat > /dev/null\n"+script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir, runDir := t.TempDir(), t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Read."), 0o644))
	dirs := []string{"in", "out"}
	if tree {
		dirs = append(dirs, ReadTreeDir)
	}
	for _, d := range dirs {
		must(t, os.MkdirAll(filepath.Join(runDir, d), 0o755))
	}
	must(t, os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("The issue."), 0o644))
	_, err := claude{}.Propose(Request{RunDir: runDir, Repo: t.TempDir(),
		Role: &role.Role{Name: "product-owner", Dir: dir, Intentions: []string{"comment"}}})
	return runDir, err
}

// Raw output not posted, read tree (#1245): Given an agent in a read tree
// whose output is not Claude Code's JSON (no result message, a file's
// contents in it), When its answer is read, Then none of that output is
// kept as its answer nor said in the error, and the run says the agent's
// words could not be read.
func TestRawOutputNotPostedWithoutWorktree(t *testing.T) {
	runDir, err := readTreeRun(t, "echo 'API_PASSWORD=hunter2-from-a-file'\necho 'not json at all'\n", true)
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
}

// Raw output not quoted on failure, read tree (#1245): Given an agent in a
// read tree whose stream ends, with no result message, on a file's
// contents, When claude exits non-zero, Then the error says it failed with
// its stderr, and quotes nothing of the stream.
func TestRawStreamNotQuotedOnFailure(t *testing.T) {
	script := `echo '{"type":"system","subtype":"init"}'
echo '{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"config/.env"}}]}}'
echo '{"type":"user","message":{"content":[{"type":"tool_result","content":"DB_PASSWORD_IN_FILE=s3cr3t-from-env-file"}]}}'
echo 'killed' >&2
exit 1
`
	_, err := readTreeRun(t, script, true)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("not an agent unavailable: %v", err)
	}
	if strings.Contains(err.Error(), "s3cr3t") || strings.Contains(err.Error(), "tool_result") {
		t.Errorf("the error quotes the raw stream: %v", err)
	}
	if !strings.Contains(err.Error(), "killed") {
		t.Errorf("the error does not say claude's stderr: %v", err)
	}
}

// Plain words read, no tools: Given an agent with no tools (a judge) whose
// output is plain text rather than Claude Code's JSON, When its answer is
// read, Then that text is kept as its answer: nothing but its words can be
// in it.
func TestPlainOutputReadWithNoTools(t *testing.T) {
	runDir, err := readTreeRun(t, "echo 'yes: the doc matches'\n", false)
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("plain text read as YAML: %v", err)
	}
	data, rerr := os.ReadFile(filepath.Join(runDir, "out", "agent-answer.txt"))
	if rerr != nil || !strings.Contains(string(data), "yes: the doc matches") {
		t.Errorf("the plain answer is not kept: %q, %v", data, rerr)
	}
}
