package engine

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// manyDrafts is a clone, on the local forge, with n tester drafts open
// behind main: each a pull request of its issue, one commit of workline's
// own, main having moved since. It gives the clone, the drafts' branches,
// the commit of main's tip, and a way to run git in it.
func manyDrafts(t *testing.T, n int) (repo string, branches []string, baseTip string, run func(args ...string) string) {
	t.Helper()
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
	for i := 1; i <= n; i++ {
		id, err := l.Open(fmt.Sprintf("issue %d", i), "A need.", nil)
		if err != nil {
			t.Fatal(err)
		}
		branch := fmt.Sprintf("workline/tester/issue-%d", id)
		run("checkout", "-q", "-b", branch, "main")
		file := fmt.Sprintf("test_%d.txt", id)
		write(file, "a test\n")
		run("add", file)
		run("commit", "-q", "-m", fmt.Sprintf("test: issue %d", id), "-m", OwnTrailer+": tester")
		run("checkout", "-q", "main")
		if _, err := l.OpenMergeRequest(branch, "main", fmt.Sprintf("test: issue %d", id), fmt.Sprintf("Tests of #%d", id), true); err != nil {
			t.Fatal(err)
		}
		branches = append(branches, branch)
	}
	write("other.txt", "main moved\n")
	run("add", "other.txt")
	run("commit", "-q", "-m", "feat: elsewhere")
	return repo, branches, run("rev-parse", "main"), run
}

// Accepted merges during a mass rebuild; Few runs per rebuild: given 25
// tester drafts open behind main, when the gardening run rebuilds them,
// then it pushes far fewer than 25 of them: each push starts the project's
// full checks, and 25 at once keep a pull request a person accepted waiting
// for its own, so the accepted pull request does not wait behind them.
func TestFollowRebuildsFewTesterDraftsAtOnce(t *testing.T) {
	repo, branches, baseTip, run := manyDrafts(t, 25)
	before := map[string]string{}
	for _, b := range branches {
		before[b] = run("rev-parse", b)
	}
	res := Follow(FollowOptions{Repo: repo, Forge: "local", Base: "main", Gardening: true})
	if res.Status == verdict.Block {
		t.Fatalf("the run blocked: %+v", res.Findings)
	}
	pushed := 0
	for _, b := range branches {
		if run("rev-parse", b) != before[b] {
			pushed++
		}
	}
	if pushed >= 10 {
		t.Errorf("%d of 25 tester drafts pushed in one gardening run, want far fewer than 25 (at most 9); summary: %s", pushed, res.Summary)
	}
	if got := run("rev-parse", "main"); got != baseTip {
		t.Errorf("main moved to %s", got)
	}
}

// Drafts stay rebuilt: given tester drafts behind main, when the gardening
// run ends, then each draft is on main's tip, or is due for its rebuild
// later, said in the run's summary: the rebuild is spread over runs, none
// is dropped without a word.
func TestFollowSaysTheTesterDraftsDueLater(t *testing.T) {
	repo, branches, baseTip, run := manyDrafts(t, 25)
	res := Follow(FollowOptions{Repo: repo, Forge: "local", Base: "main", Gardening: true})
	behind := 0
	for _, b := range branches {
		if _, err := git(repo, nil, "merge-base", "--is-ancestor", baseTip, run("rev-parse", b)); err != nil {
			behind++
		}
	}
	if behind == 0 {
		t.Fatalf("all 25 drafts rebuilt in one run: none left for a later one; summary: %s", res.Summary)
	}
	if !strings.Contains(res.Summary, "later") || !strings.Contains(res.Summary, fmt.Sprint(behind)) {
		t.Errorf("%d drafts left behind main, due for a later rebuild: the summary does not say so (the word `later` and their number): %q", behind, res.Summary)
	}
}
