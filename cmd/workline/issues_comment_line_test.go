package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// TestIssuesCommentOnALine
//
// Scenario: Line comment
// Given a local merge request that changes `bin/sc-event`,
// When I run `workline issues comment '!N' --body <text> --line 40`
// with the file `bin/sc-event`,
// Then the comment is kept on line 40 of that file, as a thread of the
// merge request, which the developer's round reads and answers there.
func TestIssuesCommentOnALine(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "") // in CI, the local forge refuses writes
	}
	sinceEnv(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	var script strings.Builder
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&script, "echo line %d\n", i)
	}
	if err := os.MkdirAll(filepath.Join(repo, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("forge: local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "bin", "sc-event"), []byte(script.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "first")
	gitIn(t, repo, "checkout", "-q", "-b", "feature")
	changed := strings.Replace(script.String(), "echo line 40\n", "echo line forty\n", 1)
	if err := os.WriteFile(filepath.Join(repo, "bin", "sc-event"), []byte(changed), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-am", "change sc-event")

	l := &forge.Local{Repo: repo}
	id, err := l.OpenMergeRequest("feature", "main", "Change sc-event", "It changes bin/sc-event.", false)
	if err != nil {
		t.Fatal(err)
	}
	mr := fmt.Sprintf("!%d", id)

	// The option naming the file is not in the issue's scope: `--file`
	// or `--path` are both taken for it.
	code := 64
	for _, name := range []string{"file", "path"} {
		_, code = captureStdout(t, func() int {
			return dispatch([]string{"issues", "comment", mr, "--body", "This quoting breaks on a space.",
				"--" + name, "bin/sc-event", "--line", "40", "--repo", repo})
		})
		if code == 0 {
			break
		}
	}
	if code != 0 {
		t.Fatalf("workline issues comment %s --line 40: exit %d, want 0", mr, code)
	}
	threads, err := l.Threads(id)
	if err != nil || len(threads) != 1 {
		t.Fatalf("Threads = %+v, %v; want the comment kept as one thread", threads, err)
	}
	th := threads[0]
	if th.Path != "bin/sc-event" || th.Line != 40 {
		t.Errorf("thread on %s:%d, want bin/sc-event:40", th.Path, th.Line)
	}
	if len(th.Comments) != 1 || !strings.HasPrefix(th.Comments[0].Body, "This quoting breaks on a space.") {
		t.Errorf("thread comments = %+v, want the text said", th.Comments)
	}
}
