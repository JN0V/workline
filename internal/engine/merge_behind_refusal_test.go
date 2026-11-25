package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A refusal says why: given an accepted pull request behind its base, its
// auto-merge on, that the engine does not update (a check of its head is
// still running), when `workline follow` weighs it, then it prints one
// decision line on it, naming the reason, and pushes nothing.
func TestFollowSaysWhyABehindPullRequestIsNotUpdated(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	repo, run, write := gitRepo(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	run("init", "-q", "--bare", "-b", "main", origin)
	run("remote", "add", "origin", origin)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("push", "-q", "origin", "main")
	run("checkout", "-q", "-b", "workline/developer/3")
	write("b.txt", "b\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "feat: b", "-m", "Workline-Role: developer")
	run("push", "-q", "origin", "workline/developer/3")
	run("checkout", "-q", "main")
	write("more.txt", "more\n")
	run("add", "more.txt")
	run("commit", "-q", "-m", "docs: more")
	run("push", "-q", "origin", "main")
	branchTip := run("rev-parse", "workline/developer/3")

	state := `{"issues": [],
		"statuses": [{"commit": "workline/developer/3", "name": "workline/verdict", "state": "success", "description": "pass: the line passed, every agent's review whole", "by": "workline-bot"}],
		"merge-requests": [{"id": 7, "branch": "workline/developer/3", "base": "main", "author": "dana",
			"labels": ["workline:accepted"],
			"label-events": [{"label": "workline:accepted", "added": true, "author": "dana", "insider": true, "writer": true, "created": "2026-01-04T10:00:00Z"}],
			"auto-merge": true, "auto-merge-at": "2026-01-04T10:05:00Z", "auto-merge-by": "workline-bot",
			"check-runs": [{"name": "test", "state": "pending"}],
			"behind": true,
			"pushes": [{"head": "workline/developer/3", "at": "2026-01-04T09:00:00Z"}],
			"comments": []}]}`
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path, Repo: repo}

	res := &Result{}
	o := FollowOptions{Repo: repo, Base: "main", Merge: "cautious"}
	if _, err := mergePolicy(f, o, false, res, map[string]kept{}); err != nil {
		t.Fatal(err)
	}

	where := forge.MergeRef(f, 7)
	var said []string
	for _, x := range res.Findings {
		if x.Where == where {
			said = append(said, x.Rule+": "+x.Message)
		}
	}
	if len(said) != 1 {
		t.Fatalf("decision lines on %s = %d, want 1: %q", where, len(said), said)
	}
	reason := strings.ToLower(said[0])
	if !strings.Contains(reason, "check") && !strings.Contains(reason, "test") && !strings.Contains(reason, "pending") && !strings.Contains(reason, "running") {
		t.Errorf("the decision line does not name why it was not updated (a check still running): %q", said[0])
	}
	if got := run("ls-remote", "origin", "refs/heads/workline/developer/3"); !strings.HasPrefix(got, branchTip) {
		t.Errorf("the branch moved though it was not to be updated: %q, was %s", got, branchTip)
	}
}
