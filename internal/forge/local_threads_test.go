package forge

import (
	"os/exec"
	"strings"
	"testing"
)

// Comments on a line of a local merge request (#1265): each its own
// comment, read back as a thread with its replies; one open is not posted
// again, one resolved is.
func TestLocalThreads(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "") // in CI, the local forge refuses writes
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	l := &Local{Repo: repo}
	id, err := l.OpenMergeRequest("workline/tester/issue-3", "main", "test: #3", "Its tests.", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Sticky(Target{Kind: "merge-request", ID: id}, "Its record.", Marker("sticky=tester/tests"), true); err != nil {
		t.Fatal(err)
	}
	c := LineComment{Path: "tests/t.sh", Line: 4, Key: "missing-1", Body: "workline:objection\n- kind: missing"}
	r, err := l.Review(id, "abc", "", []LineComment{c})
	if err != nil || len(r.Placed) != 1 || r.Placed[0].State != LinePosted {
		t.Fatalf("Review = %+v, %v", r, err)
	}
	if r, _ = l.Review(id, "abc", "", []LineComment{c}); r.Placed[0].State != LineOpen {
		t.Fatalf("posted again: %+v", r)
	}
	threads, err := l.Threads(id)
	if err != nil || len(threads) != 1 {
		t.Fatalf("Threads = %+v, %v", threads, err)
	}
	th := threads[0]
	if th.ID != "2" || th.Path != "tests/t.sh" || th.Line != 4 || !strings.HasPrefix(th.Comments[0].Body, c.Body) {
		t.Fatalf("thread = %+v", th)
	}
	// Its key read back as on a forge: the reviewer resolves and weighs
	// its own by it.
	if keys := LineKeys(th.Comments[0].Body); len(keys) != 1 || keys[0] != "missing-1" {
		t.Fatalf("keys = %v in %q", keys, th.Comments[0].Body)
	}
	if err := l.Reply(id, th.ID, "Added."); err != nil {
		t.Fatal(err)
	}
	if err := l.Reply(id, "9", "No."); err == nil {
		t.Fatal("a reply in no thread was taken")
	}
	threads, _ = l.Threads(id)
	if got := threads[0].Comments; len(got) != 2 || got[1].Body != "Added." {
		t.Fatalf("replies = %+v", got)
	}
	if err := l.Resolve(id, []string{th.ID}); err != nil {
		t.Fatal(err)
	}
	if threads, _ = l.Threads(id); !threads[0].Resolved {
		t.Fatal("not resolved")
	}
	if r, _ = l.Review(id, "def", "", []LineComment{c}); r.Placed[0].State != LinePosted {
		t.Fatalf("a resolved comment not posted again: %+v", r)
	}
}
