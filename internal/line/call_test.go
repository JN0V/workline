package line

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A call is one per block (#2353):
// Given the tester's tries spent on an issue, blocked with a call, then
// lifted by a person and blocked again on the same body,
// When the line writes the second block,
// Then a second call is posted, a new comment that mails them; the same
// block written again posts none.
func TestCallOnePerBlock(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "") // in CI, the local forge refuses writes
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	l := &forge.Local{Repo: repo}
	id, err := l.OpenIssue("An issue", "Its body.", forge.Marker("run=1"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := lineConfig(t, "routing: {events: {issue: [tester]}}\n")
	block := func(since string) {
		t.Helper()
		w := StateWrite{Issue: id, Blocked: "tries spent", Call: "A person, the tester's tries are spent.",
			Record: Record{State: BlockedState, Body: "abc", Since: since}}
		if err := writeState(l, cfg, repo, w, "tester"); err != nil {
			t.Fatal(err)
		}
	}
	calls := func() int {
		t.Helper()
		comments, err := l.Comments(forge.Target{Kind: "issue", ID: id})
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, c := range comments {
			if strings.Contains(c, "the tester's tries are spent") {
				n++
			}
		}
		return n
	}
	block("2026-11-27T10:00:00Z")
	block("2026-11-27T10:00:00Z") // the same block, written again
	if n := calls(); n != 1 {
		t.Fatalf("one block: %d calls, want 1", n)
	}
	block("2026-11-28T10:00:00Z") // blocked again on the same body
	if n := calls(); n != 2 {
		t.Fatalf("a second block on the same body: %d calls, want 2", n)
	}
}
