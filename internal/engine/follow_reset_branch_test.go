package engine

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// Reset loop: given a tester loop reset that closes its pull request (main
// holds the Draft's tests already, so nothing is left to merge), when the
// run ends, then no workline/tester/ branch is left without an open pull
// request; the Draft still open keeps its branch.
func TestFollowLeavesNoTesterBranchWithoutAPullRequest(t *testing.T) {
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
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	l := &forge.Local{Repo: repo}
	for _, file := range []string{"t1.txt", "t2.txt"} {
		id, err := l.Open("issue "+file, "A need.", nil)
		if err != nil {
			t.Fatal(err)
		}
		branch := fmt.Sprintf("workline/tester/issue-%d", id)
		run("checkout", "-q", "-b", branch, "main")
		write(file, "a test\n")
		run("add", file)
		run("commit", "-q", "-m", "test: "+file, "-m", OwnTrailer+": tester")
		run("checkout", "-q", "main")
		if _, err := l.OpenMergeRequest(branch, "main", "test: "+file, "Its tests.", true); err != nil {
			t.Fatal(err)
		}
	}
	// Main moved, and holds the first Draft's tests as they are.
	write("t1.txt", "a test\n")
	write("other.txt", "main moved\n")
	run("add", "t1.txt", "other.txt")
	run("commit", "-q", "-m", "feat: elsewhere")

	res := Follow(FollowOptions{Repo: repo, Forge: "local", Base: "main", Gardening: true})
	if res.Status == verdict.Block {
		t.Fatalf("the run blocked: %+v", res.Findings)
	}
	left := strings.Fields(run("for-each-ref", "--format=%(refname:short)", "refs/heads/workline/tester/"))
	open, err := l.OpenMergeRequests("workline/tester/")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(left) != fmt.Sprint(open) || fmt.Sprint(open) != "[workline/tester/issue-2]" {
		t.Errorf("tester branches left: %v, open pull requests: %v; want the one open Draft, workline/tester/issue-2, alone; summary: %s", left, open, res.Summary)
	}
}
