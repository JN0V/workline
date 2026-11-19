package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// Merge wait said
// Given a green pull request from the developer's branch, whose commit
// is not workline's own (no trailer naming a role),
// When the merge job decides it waits,
// Then the pull request says why, in a comment.
func TestWeighSaysTheWaitOfACommitNotWorklines(t *testing.T) {
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
	run("commit", "-q", "-m", "fix: b") // no trailer: a hand's commit
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

	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeNormal}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	line, err := weigh(l, l, o, true, mr, &Result{}, map[string]kept{}, n)
	if err != nil {
		t.Fatal(err)
	}
	if line != nil {
		t.Fatalf("a commit not workline's own was accepted: %+v", line)
	}
	all, err := l.Comments(target)
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	for _, c := range all {
		if !strings.Contains(c, "workline:sticky=developer/") {
			said = append(said, c)
		}
	}
	if len(said) != 1 {
		t.Fatalf("the pull request carries %d comments of the wait, want one: %q", len(said), said)
	}
	if !strings.Contains(said[0], "not workline's own") {
		t.Errorf("the comment does not say why it waits: %q", said[0])
	}
}
