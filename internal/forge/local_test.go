package forge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The local forge keeps what a forge would: an issue, its comments in place
// by their marker, its labels, read back as written, in .git/workline.
func TestLocalRoundTrip(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "") // in CI, the local forge refuses writes
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	l := &Local{Repo: repo}
	id, err := l.OpenIssue("A title: with a colon", "The body.\n\n---\n\nStill the body.", Marker("run=1"))
	if err != nil || id != 1 {
		t.Fatalf("OpenIssue = %d, %v", id, err)
	}
	// The same title again comments on the open issue, as on GitHub.
	if again, _ := l.OpenIssue("A title: with a colon", "Again.", Marker("run=1")); again != 1 {
		t.Fatalf("the same title opened #%d", again)
	}
	is := Target{Kind: "issue", ID: 1}
	for _, body := range []string{"first\n\nwith a blank line", "second"} {
		if err := l.Sticky(is, body, Marker("sticky=x"), true); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Comment(is, "other", Marker("run=2")); err != nil {
		t.Fatal(err)
	}
	if err := l.Label(is, []string{"a", "b"}, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	it, err := l.Item("issue", 1)
	if err != nil {
		t.Fatal(err)
	}
	if it.Title != "A title: with a colon" || !strings.Contains(it.Body, "Still the body.") || len(it.Comments) != 3 ||
		!strings.HasPrefix(it.Comments[1], "second") || strings.Join(it.Labels, ",") != "b" {
		t.Fatalf("read back %+v", it)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "workline", "issues", "1.md")); err != nil {
		t.Fatal(err)
	}
	if n, _ := l.KeepIssue("Kept", "v1", false); n != 0 {
		t.Fatal("update-only opened an issue")
	}
	n, _ := l.KeepIssue("Kept", "v1", true)
	if m, _ := l.KeepIssue("Kept", "v2", true); m != n {
		t.Fatalf("kept issue moved from #%d to #%d", n, m)
	}
	if got, _ := l.Issue(n); got.Body != "v2" {
		t.Fatalf("kept body = %q", got.Body)
	}
}

// A merge request closed by a role: its comment and its state in one
// write, so it is never said closed and left open; a write that fails is
// said, the merge request still open.
func TestLocalCloseMergeRequest(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "first")
	l := &Local{Repo: repo}
	for _, b := range []string{"workline/documentalist/suspect", "workline/documentalist/stale"} {
		run("checkout", "-q", "-b", b, "main")
		run("commit", "-q", "--allow-empty", "-m", b)
		if _, err := l.OpenMergeRequest(b, "main", "docs: "+b, "body"); err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Join(repo, ".git", "workline", "merge-requests")
	os.Chmod(dir, 0o555)
	err := l.CloseMergeRequest("workline/documentalist/stale", "Closed: nothing is left to merge.", Marker("closed"))
	os.Chmod(dir, 0o755)
	if err == nil {
		t.Fatal("a close that could not be written passed")
	}
	if open, _ := l.OpenMergeRequests("workline/"); len(open) != 2 {
		t.Fatalf("after a failed close, open: %v", open)
	}
	for range 2 { // again: one comment, still closed
		if err := l.CloseMergeRequest("workline/documentalist/stale", "Closed: nothing is left to merge.", Marker("closed")); err != nil {
			t.Fatal(err)
		}
	}
	it, _ := l.Item("merge-request", 2)
	if l.StateOf(it) != "closed" || len(it.Comments) != 1 || !strings.Contains(it.Comments[0], "nothing is left to merge") {
		t.Fatalf("closed: %+v", it)
	}
	if open, _ := l.OpenMergeRequests("workline/"); len(open) != 1 || open[0] != "workline/documentalist/suspect" {
		t.Fatalf("open: %v", open)
	}
}

// The backlog of the local forge: the open issues listed, their comments
// read back, a closing that lists the issue no more, closing again nothing.
func TestLocalBacklog(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	l := &Local{Repo: repo}
	for _, title := range []string{"one", "two"} {
		if _, err := l.OpenIssue(title, "body", Marker("run="+title)); err != nil {
			t.Fatal(err)
		}
	}
	l.Comment(Target{Kind: "issue", ID: 2}, "a comment", Marker("c"))
	if c, err := l.Comments(Target{Kind: "issue", ID: 2}); err != nil || len(c) != 1 || !strings.Contains(c[0], "a comment") {
		t.Fatalf("Comments = %q, %v", c, err)
	}
	for range 2 {
		if err := l.Close(1, 0); err != nil {
			t.Fatal(err)
		}
	}
	open, err := l.Issues()
	if err != nil || len(open) != 1 || open[0].ID != 2 {
		t.Fatalf("Issues = %+v, %v", open, err)
	}
	if is, _ := l.Issue(1); !is.Closed {
		t.Fatal("#1 not read as closed")
	}
	if err := l.SetMilestone(2, "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if ms, err := l.Milestones(); err != nil || len(ms) != 1 || ms[0].Title != "v1.0.0" {
		t.Fatalf("Milestones = %v, %v", ms, err)
	}
}
