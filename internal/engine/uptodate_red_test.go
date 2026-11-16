package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// Red fixed on main; No person's token: given a developer pull request
// red at the tester (frozen-test-missing, as #1638 was; not
// proof-over-time), when a fix merges into main, then the line brings the
// branch up to date by itself, main merged in as workline's own commit (a
// trailer of its own), so the checks run again on the new head; and it
// does not wait for a person's label because of it.
func TestWeighTakesMainInForATesterBlock(t *testing.T) {
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
	write("b.txt", "b\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "fix: b", "-m", OwnTrailer+": developer")
	head := run("rev-parse", "HEAD")
	run("checkout", "-q", "main")
	write("c.txt", "the tester's fix\n")
	run("add", "c.txt")
	run("commit", "-q", "-m", "fix: the tester's bug")
	baseTip := run("rev-parse", "HEAD")

	l := &forge.Local{Repo: repo}
	if _, err := l.Open("an issue", "A bug.", []string{"bug"}); err != nil {
		t.Fatal(err)
	}
	id, err := l.OpenMergeRequest("workline/developer/1", "main", "fix: b", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Comment(forge.Target{Kind: "merge-request", ID: id}, "**The developer's record**\n\n<!-- workline:developer-act {\"act\":\"code\",\"issue\":1} -->", forge.Marker("sticky=developer/code")); err != nil {
		t.Fatal(err)
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	// The verdict that kept pull request #1638 red, as its workline/verdict
	// status kept it: the tester's frozen-test-missing, with the
	// documentalist's findings that ride along.
	rules := []string{"derived-behind", "folder-too-long", "suspect", "links-not-checked", "section-too-long", "doc-too-long", "sources-too-wide", "frozen-test-missing"}
	hand := map[string]kept{head: {class: verdictRed, why: "the line's verdict on its head is block (tester block): " + strings.Join(rules, ", "), rules: rules}}
	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeNormal}
	res := &Result{}
	if _, err := weigh(l, l, o, true, mr, res, hand, n); err != nil {
		t.Fatal(err)
	}

	tip := run("rev-parse", "workline/developer/1")
	if tip == head {
		t.Fatalf("the branch stayed at %s: main, which moved, was not merged in; findings %+v", head, res.Findings)
	}
	if _, err := git(repo, nil, "merge-base", "--is-ancestor", baseTip, tip); err != nil {
		t.Errorf("main (%s) is not in the new head %s: %v", baseTip, tip, err)
	}
	if _, err := git(repo, nil, "merge-base", "--is-ancestor", head, tip); err != nil {
		t.Errorf("the old head %s is not kept in the new one: nothing is to be rewritten", head)
	}
	if ownTrailer(repo, tip) == "" {
		t.Errorf("the merge commit %s carries no trailer of workline's own: it reads as a person's push", tip)
	}
	for _, f := range res.Findings {
		if f.Rule == "waits-for-a-yes" || f.Rule == "label-ignored" {
			t.Errorf("finding %s (%s): the pull request waits for a person's label because of the update", f.Rule, f.Message)
		}
		if f.Level == verdict.Block {
			t.Errorf("blocking finding %+v", f)
		}
	}
}
