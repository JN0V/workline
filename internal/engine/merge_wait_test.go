package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// The wait is said; A blocking finding still holds: given a developer's
// fix whose reviewer's summary holds a finding from a lens that blocks,
// when the merge pass runs, then it waits (nothing accepted, a
// waits-for-a-yes finding), and the pull request carries one comment
// saying why and what the person does, updated in place: a second pass
// posts no second one.
func TestWeighSaysTheWaitOnABlockingFinding(t *testing.T) {
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
	record := "**The developer's record**\n\n" + developer.Act{Act: "code", Issue: 1}.Line()
	if err := l.Comment(target, record, forge.Marker("sticky=developer/"+developer.RecordKey)); err != nil {
		t.Fatal(err)
	}
	summary := "**workline reviewer** — verdict pass.\n\n" +
		`<!-- workline:findings [{"lens":"correctness","holds":"blocks","where":"b.txt:1","severity":"Important","text":"Wrong result on empty input"}] -->`
	if err := l.Comment(target, summary, forge.Marker("sticky=reviewer/summary")); err != nil {
		t.Fatal(err)
	}

	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeNormal}
	hand := map[string]kept{}
	weighOnce := func() *Result {
		t.Helper()
		mr, err := l.MergeRequest(id)
		if err != nil {
			t.Fatal(err)
		}
		res := &Result{}
		line, err := weigh(l, l, o, true, mr, res, hand, n)
		if err != nil {
			t.Fatal(err)
		}
		if line != nil {
			t.Fatalf("a fix with a blocking finding open was accepted: %+v", line)
		}
		return res
	}
	// said is the comments workline's wait wrote: neither its record nor
	// the reviewer's.
	said := func() []string {
		t.Helper()
		all, err := l.Comments(target)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, c := range all {
			if !strings.Contains(c, "workline:sticky=developer/") && !strings.Contains(c, "workline:sticky=reviewer/") {
				out = append(out, c)
			}
		}
		return out
	}

	res := weighOnce()
	waited := false
	for _, f := range res.Findings {
		if f.Rule == "waits-for-a-yes" {
			waited = true
		}
	}
	if !waited {
		t.Errorf("no waits-for-a-yes finding: %+v", res.Findings)
	}
	first := said()
	if len(first) != 1 {
		t.Fatalf("the pull request carries %d comments of the wait, want one: %q", len(first), first)
	}
	if !strings.Contains(first[0], "Wrong result on empty input") {
		t.Errorf("the comment does not say why (the blocking finding): %q", first[0])
	}
	if !strings.Contains(first[0], backlog.AcceptedLabel(l)) {
		t.Errorf("the comment does not say what the person does (put the label %s): %q", backlog.AcceptedLabel(l), first[0])
	}

	weighOnce()
	second := said()
	if len(second) != 1 || second[0] != first[0] {
		t.Errorf("the second pass posted again or rewrote it differently: %q, then %q", first, second)
	}
}
