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
