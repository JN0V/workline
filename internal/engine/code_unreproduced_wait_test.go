package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// unreproducedWaitAfter weighs a developer's fix pull request at `normal`
// (the reviewer's summary read its head, no test red on the base shows the
// bug), once for each verdict given in order, true for a red head and false
// for a green one, on the same pull request. It gives the unreproduced wait
// comment as it stands after each weighing.
func unreproducedWaitAfter(t *testing.T, reds ...bool) []string {
	t.Helper()
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	repo, run, write := gitRepo(t)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("checkout", "-q", "-b", "workline/developer/1")
	write("b.txt", "b from the branch\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "fix: b", "-m", OwnTrailer+": developer")
	head := run("rev-parse", "HEAD")
	run("checkout", "-q", "main")

	l := &forge.Local{Repo: repo}
	if _, err := l.Open("a bug", "A bug.", []string{"bug"}); err != nil {
		t.Fatal(err)
	}
	id, err := l.OpenMergeRequest("workline/developer/1", "main", "fix: b", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	target := forge.Target{Kind: "merge-request", ID: id}
	record := "**The developer's record**\n\n" + devmark.Act{Act: "code", Issue: 1}.Line()
	if err := l.Comment(target, record, forge.Marker("sticky=developer/"+devmark.RecordKey)); err != nil {
		t.Fatal(err)
	}
	summary := "**workline reviewer** — verdict pass.\n\n<!-- workline:at head=" + head + " -->"
	if err := l.Comment(target, summary, forge.Marker("sticky=reviewer/summary")); err != nil {
		t.Fatal(err)
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeNormal}
	var waits []string
	for _, red := range reds {
		hand := map[string]kept{}
		if red {
			hand[head] = kept{class: verdictRed, why: "the line's verdict on its head is block (tester): tester-test-wrong", rules: []string{"tester-test-wrong"}}
		}
		if _, err := weigh(l, l, o, true, mr, &Result{}, hand, n); err != nil {
			t.Fatal(err)
		}
		comments, err := l.Comments(target)
		if err != nil {
			t.Fatal(err)
		}
		found := ""
		for _, c := range comments {
			if strings.Contains(c, forge.Marker("merge-waits="+devmark.Unreproduced)) {
				found = c
			}
		}
		if found == "" {
			t.Fatalf("no merge-waits=%s comment after weighing %d; comments: %q", devmark.Unreproduced, len(waits)+1, comments)
		}
		waits = append(waits, found)
	}
	return waits
}

// Green after red; Still red
// Given the unreproduced wait said "a person's yes neither" on a red head (the line blocks it, a yes does not help),
// When a later head is green and the fix is still unreproduced,
// Then the comment says a person's yes is what the merge request waits for, edited in place.
func TestUnreproducedWaitFollowsTheHead(t *testing.T) {
	got := unreproducedWaitAfter(t, false, true, false)

	red := got[1]
	if !strings.Contains(red, "the line blocks its head") || !strings.Contains(red, "a person's yes neither") {
		t.Fatalf("on a red head the comment does not say the line blocks it and a yes does not help: %q", red)
	}

	green := got[2]
	if strings.Contains(green, "a person's yes neither") || strings.Contains(green, "the line blocks its head") {
		t.Errorf("on a later green head the comment still says the line blocks it: %q", green)
	}
	if !strings.Contains(green, "**Waiting for a person's yes**") || !strings.Contains(green, "no test red on the base shows the bug") {
		t.Errorf("on a later green head the comment does not say a person's yes is what it waits for: %q", green)
	}
	if !strings.Contains(green, "To accept this one, put the label") {
		t.Errorf("on a later green head the comment does not say how to give the yes: %q", green)
	}
}
