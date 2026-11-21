package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// refusedWaitComment weighs a developer's feature pull request at
// `enterprising` with a release candidate refused, main moved under it
// (in conflict with it, or not), and gives the wait comment written on it.
func refusedWaitComment(t *testing.T, conflict, red bool) string {
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
	run("commit", "-q", "-m", "feat: b", "-m", OwnTrailer+": developer")
	head := run("rev-parse", "HEAD")
	run("checkout", "-q", "main")
	if conflict {
		write("b.txt", "b from main\n")
		run("add", "b.txt")
	} else {
		write("c.txt", "c\n")
		run("add", "c.txt")
	}
	run("commit", "-q", "-m", "main moves on")

	l := &forge.Local{Repo: repo}
	if _, err := l.Open("a feature", "A feature.", []string{"enhancement"}); err != nil {
		t.Fatal(err)
	}
	id, err := l.OpenMergeRequest("workline/developer/1", "main", "feat: b", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	target := forge.Target{Kind: "merge-request", ID: id}
	if err := l.Comment(target, "**The developer's record**\n\n<!-- workline:developer-act {\"act\":\"code\",\"issue\":1} -->", forge.Marker("sticky=developer/code")); err != nil {
		t.Fatal(err)
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	hand := map[string]kept{}
	if red {
		hand[head] = kept{class: verdictRed, why: "the line's verdict on its head is block (tester): tester-test-wrong", rules: []string{"tester-test-wrong"}}
	}
	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }, demoted: map[string]string{kindFeature: "#7"}}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeEnterprising}
	if _, err := weigh(l, l, o, true, mr, &Result{}, hand, n); err != nil {
		t.Fatal(err)
	}
	comments, err := l.Comments(target)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range comments {
		if strings.Contains(c, forge.Marker("merge-waits=feature")) {
			return c
		}
	}
	t.Fatalf("no merge-waits=feature comment on the pull request; comments: %q", comments)
	return ""
}

// Red verdict said
// Given a refused candidate and a red head,
// When the wait comment is written,
// Then it says the verdict is red and that the label alone will not merge it.
func TestRefusedWaitSaysRedVerdict(t *testing.T) {
	got := refusedWaitComment(t, false, true)
	if !strings.Contains(got, "a release candidate was refused") {
		t.Fatalf("not the refused-candidate wait: %q", got)
	}
	if !strings.Contains(got, "**The verdict on this head is red**: the line's verdict on its head is block (tester): tester-test-wrong.") {
		t.Errorf("the wait does not say the verdict is red: %q", got)
	}
	if !strings.Contains(got, "The label alone will not merge it") {
		t.Errorf("the wait does not say the label alone will not merge it: %q", got)
	}
}

// Conflict said
// Given a refused candidate and a head conflicting with main,
// When the comment is written,
// Then it says so.
func TestRefusedWaitSaysConflict(t *testing.T) {
	got := refusedWaitComment(t, true, false)
	if !strings.Contains(got, "a release candidate was refused") {
		t.Fatalf("not the refused-candidate wait: %q", got)
	}
	if !strings.Contains(got, "**It conflicts with main**") {
		t.Errorf("the wait does not say the head conflicts with main: %q", got)
	}
	if !strings.Contains(got, "the label alone will not merge it") {
		t.Errorf("the wait does not say the label alone will not merge it: %q", got)
	}
}
