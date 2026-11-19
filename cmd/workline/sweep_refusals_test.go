package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// sweepBrokenRepo is a committed repository whose `.workline/config.yaml`
// is not valid YAML: every run of a role on it is refused.
func sweepBrokenRepo(t *testing.T) string {
	t.Helper()
	sinceEnv(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("foo: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	return repo
}

// sweepRun runs the built binary in dir, its stdout and its stderr apart.
func sweepRun(t *testing.T, bin, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // no /dev/tty
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return o.String(), e.String(), code
}

// Said in the log
// Given a run of a role that is refused (a config that is not valid YAML),
// When it ends with --json,
// Then its log (stderr) prints the reason, and stdout stays the JSON result.
func TestJSONRunSaysItsRefusalInTheLog(t *testing.T) {
	bin := sinceBuild(t)
	repo := sweepBrokenRepo(t)
	stdout, stderr, code := sweepRun(t, bin, repo, "docs", "--repo", repo, "--ai", "none", "--json")
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Errorf("stdout is not the JSON result: %v\n%s", err, stdout)
	}
	if !strings.Contains(stderr, "config-invalid") || !strings.Contains(stderr, "did not find expected") {
		t.Errorf("the log (stderr) lacks the reason of the refusal with --json:\n%s", stderr)
	}
}

// Said on the item
// Given a run of a role on an issue that is refused,
// When it ends, and again when it is run once more,
// Then the issue carries one comment saying what was refused and why,
// updated in place, never a second one.
func TestRefusedRunCommentsItsIssueOnce(t *testing.T) {
	bin := sinceBuild(t)
	repo := sweepBrokenRepo(t)
	state := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(state, []byte(`{"issues":[{"id":1,"title":"A need","labels":[],"comments":[]}],"merge-requests":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for round := 1; round <= 2; round++ {
		stdout, stderr, _ := sweepRun(t, bin, repo, "run-role", "documentalist", "--event", "pre-push",
			"--repo", repo, "--ai", "none", "--forge", "fake:"+state, "--target", "issue:1", "--json")
		data, err := os.ReadFile(state)
		if err != nil {
			t.Fatal(err)
		}
		var s struct {
			Issues []struct {
				Comments []json.RawMessage `json:"comments"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(data, &s); err != nil || len(s.Issues) != 1 {
			t.Fatalf("round %d: forge file unreadable: %v\n%s", round, err, data)
		}
		comments := s.Issues[0].Comments
		if len(comments) != 1 {
			t.Fatalf("round %d: the issue carries %d comments, want exactly 1\nforge: %s\nstdout:\n%s\nstderr:\n%s", round, len(comments), data, stdout, stderr)
		}
		if !strings.Contains(string(comments[0]), "did not find expected") {
			t.Errorf("round %d: the comment does not say why the run was refused: %s", round, comments[0])
		}
	}
}
