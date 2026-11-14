package line

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	wlclock "github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
)

// Given a question on a local forge unanswered past the delay,
// When reconcile runs (the line's timeout check),
// Then an assumption is due: the question is dated, and the line
// wakes the product owner once for it (ask=1).
func TestLocalQuestionTimesOut(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "") // in CI, the local forge refuses writes
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	l := &forge.Local{Repo: repo}
	id, err := l.OpenIssue("A vague issue", "Something is wrong.", forge.Marker("run=1"))
	if err != nil {
		t.Fatal(err)
	}
	written := time.Now().UTC()
	defer func(c func() time.Time) { wlclock.Now = c }(wlclock.Now)
	wlclock.Now = func() time.Time { return written }
	question := backlog.Ask("alice", "What did you expect?", 1)
	if err := l.Comment(forge.Target{Kind: "issue", ID: id}, question, backlog.AskMarker(backlog.Keeper, 1)); err != nil {
		t.Fatal(err)
	}
	notes, err := l.Notes(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		t.Fatal(err)
	}
	s, open := backlog.Unanswered(notes, backlog.Keeper)
	if !open || !s.Dated {
		t.Fatalf("the local forge's question is not dated: open=%v dated=%v", open, s.Dated)
	}

	// The default delay is a day: two days later it has passed.
	wlclock.Now = func() time.Time { return written.Add(48 * time.Hour) }
	key, _, _, _, err := timedOut(l, newTally(), forge.Issue{ID: id}, notes, Record{})
	if err != nil {
		t.Fatal(err)
	}
	if key != "ask=1" {
		t.Fatalf("timedOut key = %q, want ask=1", key)
	}
}
