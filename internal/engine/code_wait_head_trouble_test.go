package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// codeWaitComment weighs a developer's fix pull request at `normal`, main
// moved under it (in conflict with it, or not), its head red or not, and
// the reviewer's summary as given ("" for none, "findings" for one with an
// open finding, "read" for one that read the head with no finding). It
// gives the wait comment written on it, the one with the marker's key.
func codeWaitComment(t *testing.T, conflict, red bool, review, key string) string {
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
	if conflict {
		write("b.txt", "b from main\n")
		run("add", "b.txt")
	} else {
		write("c.txt", "c\n")
		run("add", "c.txt")
	}
	run("commit", "-q", "-m", "main moves on")

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
	summary := ""
	switch review {
	case "findings":
		summary = "**workline reviewer** — verdict pass.\n\n" +
			`<!-- workline:findings [{"lens":"correctness","holds":"blocks","where":"b.txt:1","severity":"Important","text":"Wrong result on empty input"}] -->`
	case "read":
		summary = "**workline reviewer** — verdict pass.\n\n<!-- workline:at head=" + head + " -->"
	}
	if summary != "" {
		if err := l.Comment(target, summary, forge.Marker("sticky=reviewer/summary")); err != nil {
			t.Fatal(err)
		}
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	hand := map[string]kept{}
	if red {
		hand[head] = kept{class: verdictRed, why: "the line's verdict on its head is block (tester): tester-test-wrong", rules: []string{"tester-test-wrong"}}
	}
	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeNormal}
	if _, err := weigh(l, l, o, true, mr, &Result{}, hand, n); err != nil {
		t.Fatal(err)
	}
	comments, err := l.Comments(target)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range comments {
		if strings.Contains(c, forge.Marker("merge-waits="+key)) {
			return c
		}
	}
	t.Fatalf("no merge-waits=%s comment on the pull request; comments: %q", key, comments)
	return ""
}

// Open findings and a red head
// Given a pull request with open reviewer findings and a red verdict on its head,
// When the line writes its waiting comment,
// Then the comment says the findings are open and also that the verdict is red and the label alone will not merge it.
func TestFindingsWaitSaysRedVerdict(t *testing.T) {
	got := codeWaitComment(t, false, true, "findings", "findings")
	if !strings.Contains(got, "the reviewer's findings are open") || !strings.Contains(got, "Wrong result on empty input") {
		t.Fatalf("not the open-findings wait: %q", got)
	}
	if !strings.Contains(got, "**The verdict on this head is red**: the line's verdict on its head is block (tester): tester-test-wrong.") {
		t.Errorf("the wait does not say the verdict is red: %q", got)
	}
	if !strings.Contains(got, "The label alone will not merge it") {
		t.Errorf("the wait does not say the label alone will not merge it: %q", got)
	}
}

// Open findings and a conflict
// Given a pull request with open reviewer findings that conflicts with main,
// When the line writes its waiting comment,
// Then the comment also says it conflicts with main.
func TestFindingsWaitSaysConflict(t *testing.T) {
	got := codeWaitComment(t, true, false, "findings", "findings")
	if !strings.Contains(got, "the reviewer's findings are open") {
		t.Fatalf("not the open-findings wait: %q", got)
	}
	if !strings.Contains(got, "**It conflicts with main**") {
		t.Errorf("the wait does not say the head conflicts with main: %q", got)
	}
	if !strings.Contains(got, "the label alone will not merge it") {
		t.Errorf("the wait does not say the label alone will not merge it: %q", got)
	}
}

// Unreviewed and unreproduced waits on a conflict
// Given a conflicting head on a wait for a review or a reproduction,
// When the line writes its waiting comment,
// Then it also says the head conflicts with main.
func TestUnreviewedAndUnreproducedWaitsSayConflict(t *testing.T) {
	unreviewed := codeWaitComment(t, true, false, "", devmark.Unreviewed)
	if !strings.Contains(unreviewed, "no reviewer's verdict is on it") {
		t.Fatalf("not the unreviewed wait: %q", unreviewed)
	}
	if !strings.Contains(unreviewed, "**It conflicts with main**") {
		t.Errorf("the unreviewed wait does not say the head conflicts with main: %q", unreviewed)
	}
	unreproduced := codeWaitComment(t, true, false, "read", devmark.Unreproduced)
	if !strings.Contains(unreproduced, "no test red on the base shows the bug") {
		t.Fatalf("not the unreproduced wait: %q", unreproduced)
	}
	if !strings.Contains(unreproduced, "**It conflicts with main**") {
		t.Errorf("the unreproduced wait does not say the head conflicts with main: %q", unreproduced)
	}
}
