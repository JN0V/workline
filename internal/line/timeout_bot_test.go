package line

import (
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	wlclock "github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
)

// asked is the product owner's question to the reporter ann, written at at.
func askedAt(at time.Time) forge.Note {
	return forge.Note{
		Body:    backlog.Ask("ann", "What did you expect?", 1) + "\n\n" + backlog.AskMarker(backlog.Keeper, 1),
		Created: at.Format(time.RFC3339),
	}
}

// Bot after the question: Given a question to the reporter and then a
// bot's comment, When the delay passes, Then the product owner writes its
// assumption: the line wakes it for the question (ask=1).
func TestBotAfterTheQuestionDoesNotEndTheWait(t *testing.T) {
	written := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	defer func(c func() time.Time) { wlclock.Now = c }(wlclock.Now)
	wlclock.Now = func() time.Time { return written.Add(48 * time.Hour) } // the default delay is a day
	is := forge.Issue{ID: 5, Author: "ann", Title: "A vague issue", Body: "Something is wrong."}
	notes := []forge.Note{
		askedAt(written),
		{Body: "Thanks for the report, a build ran on this issue.", Author: "ci-bot", Bot: true, Created: written.Add(time.Hour).Format(time.RFC3339)},
	}
	key, _, _, _, err := timedOut(nil, newTally(), is, notes, Record{})
	if err != nil {
		t.Fatal(err)
	}
	if key != "ask=1" {
		t.Fatalf("timedOut key = %q, want ask=1: a bot's comment ended the wait", key)
	}
}

// Stranger after the question: Given a question and then an outsider's
// "+1", When the product owner runs, Then the question still waits: the
// line says it waits on its reporter, and wakes nothing.
func TestStrangerAfterTheQuestionLeavesItWaiting(t *testing.T) {
	written := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	defer func(c func() time.Time) { wlclock.Now = c }(wlclock.Now)
	wlclock.Now = func() time.Time { return written.Add(2 * time.Hour) } // within the day's delay
	is := forge.Issue{ID: 5, Author: "ann", Title: "A vague issue", Body: "Something is wrong."}
	notes := []forge.Note{
		askedAt(written),
		{Body: "+1", Author: "stranger", Created: written.Add(time.Hour).Format(time.RFC3339)},
	}
	key, _, waits, _, err := timedOut(nil, newTally(), is, notes, Record{})
	if err != nil {
		t.Fatal(err)
	}
	if key != "" {
		t.Errorf("timedOut key = %q, want none within the delay", key)
	}
	if !strings.Contains(waits, "its question of 2026-10-01 10:00 UTC waits on its reporter") {
		t.Fatalf("the question does not wait after an outsider's +1: waits = %q", waits)
	}
}
