package engine

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// developerPullRequest sets up a developer pull request on the local
// forge, branch workline/developer/1 of issue #1 labelled label, workline's
// own commit on it, main checked out and not ahead of it yet.
func developerPullRequest(t *testing.T, label string) (l *forge.Local, repo string, run func(args ...string) string, write func(p, s string), id int) {
	t.Helper()
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	repo, run, write = gitRepo(t)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("checkout", "-q", "-b", "workline/developer/1")
	write("b.txt", "b\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "fix: b", "-m", OwnTrailer+": developer")
	run("checkout", "-q", "main")

	l = &forge.Local{Repo: repo}
	if _, err := l.Open("an issue", "An issue.", []string{label}); err != nil {
		t.Fatal(err)
	}
	var err error
	id, err = l.OpenMergeRequest("workline/developer/1", "main", "fix: b", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	record := "**The developer's record**\n\n<!-- workline:developer-act {\"act\":\"code\",\"issue\":1} -->"
	if err := l.Comment(forge.Target{Kind: "merge-request", ID: id}, record, forge.Marker("sticky=developer/code")); err != nil {
		t.Fatal(err)
	}
	return l, repo, run, write, id
}

// weighPullRequest runs the merge policy's weighing once on the pull
// request id, as the line does, with the verdicts in hand.
func weighPullRequest(t *testing.T, l *forge.Local, repo string, id int, hand map[string]kept) *Result {
	t.Helper()
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeNormal}
	res := &Result{}
	if _, err := weigh(l, l, o, true, mr, res, hand, n); err != nil {
		t.Fatal(err)
	}
	return res
}

// Red pull request bounded: given a developer pull request red for a
// rule main does not fix, when main moves three times, then the engine
// merges main in and re-runs the judge a bounded number of times (at
// least once, fewer than three), said on the pull request.
func TestWeighBoundsTheMergesInForARedHead(t *testing.T) {
	l, repo, run, write, id := developerPullRequest(t, "bug")
	first := run("rev-parse", "workline/developer/1")
	for i := 1; i <= 3; i++ {
		write(fmt.Sprintf("main%d.txt", i), "m\n")
		run("add", "-A")
		run("commit", "-q", "-m", fmt.Sprintf("docs: main moves %d", i))
		tip := run("rev-parse", "workline/developer/1")
		hand := map[string]kept{tip: {class: verdictRed, why: "the line's verdict on its head is block (tester): tester-test-wrong", rules: []string{"tester-test-wrong"}}}
		weighPullRequest(t, l, repo, id, hand)
	}
	merges, err := strconv.Atoi(run("rev-list", "--count", "--merges", first+"..workline/developer/1"))
	if err != nil {
		t.Fatal(err)
	}
	if merges < 1 || merges >= 3 {
		t.Errorf("main merged into the red branch %d times over three moves of main: want at least once, and a bound under three", merges)
	}
	comments, err := l.Comments(forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(strings.Join(comments, "\n")), "main") {
		t.Errorf("the bound is not said on the pull request: its comments %q", comments)
	}
}

// Waiting feature kept up to date: given a feature waiting for a person's
// yes and behind main, when the line runs, then main is merged into it
// once per tip of main, and it still waits for the yes.
func TestWeighKeepsAWaitingFeatureUpToDate(t *testing.T) {
	l, repo, run, write, id := developerPullRequest(t, "enhancement")
	waits := func(res *Result) bool {
		for _, f := range res.Findings {
			if f.Rule == "waits-for-a-yes" {
				return true
			}
		}
		return false
	}
	in := func(tip, head string) bool {
		_, err := git(repo, nil, "merge-base", "--is-ancestor", tip, head)
		return err == nil
	}
	moveMain := func(name string) string {
		write(name, "m\n")
		run("add", "-A")
		run("commit", "-q", "-m", "docs: "+name)
		return run("rev-parse", "main")
	}

	firstTip := moveMain("main1.txt")
	res := weighPullRequest(t, l, repo, id, map[string]kept{})
	head := run("rev-parse", "workline/developer/1")
	if !in(firstTip, head) {
		t.Fatalf("main (%s) was not merged into the waiting feature, at %s; findings %+v", firstTip, head, res.Findings)
	}
	if !waits(res) {
		t.Errorf("the feature does not wait for a person's yes after the update: %+v", res.Findings)
	}

	// The same tip of main: nothing merged again.
	res = weighPullRequest(t, l, repo, id, map[string]kept{})
	if again := run("rev-parse", "workline/developer/1"); again != head {
		t.Errorf("the branch moved from %s to %s with main at the same tip", head, again)
	}
	if !waits(res) {
		t.Errorf("the feature does not wait for a person's yes: %+v", res.Findings)
	}

	// A new tip of main: merged in once more.
	secondTip := moveMain("main2.txt")
	res = weighPullRequest(t, l, repo, id, map[string]kept{})
	if now := run("rev-parse", "workline/developer/1"); now == head || !in(secondTip, now) {
		t.Errorf("main's new tip %s was not merged into the feature: head %s", secondTip, now)
	}
	if !waits(res) {
		t.Errorf("the feature does not wait for a person's yes after the second update: %+v", res.Findings)
	}
}
