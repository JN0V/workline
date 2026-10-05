package backlog

import (
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// A day is counted in UTC, whatever the zone a forge writes it in; a link
// made the moment the label was set counts as started.
func TestWaitDays(t *testing.T) {
	now := time.Date(2026, 10, 5, 23, 30, 0, 0, time.UTC)
	if d := Day("2026-10-01T01:00:00+02:00"); d != "2026-09-30" {
		t.Errorf("day = %q", d)
	}
	if n := Days("2026-09-30", now); n != 5 {
		t.Errorf("days = %d", n)
	}
	if Day("yesterday") != "" {
		t.Error("a time that does not read gave a day")
	}
	at := "2026-09-01T10:00:00Z"
	if w, known := ReadyWait(4, forge.Trail{Labeled: at, Links: []forge.Link{{Kind: "commit", Ref: "1a2b3c4", At: at}}}); w != nil || !known {
		t.Errorf("a commit at the label's moment: %+v %v", w, known)
	}
	if _, known := ReadyWait(4, forge.Trail{}); known {
		t.Error("no day said, read as known")
	}
}

// Stuck lists the oldest first in each kind of wait, a wait for an issue
// closed or not past stuck-days left out, an announcement due listed
// whatever its age.
func TestBoardStuck(t *testing.T) {
	now := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	open := []forge.Issue{{ID: 1, Title: "a"}, {ID: 2, Title: "b"}, {ID: 3, Title: "c"}}
	waits := []Wait{
		{Issue: 2, Waits: WaitsAsked, Since: "2026-09-01"},
		{Issue: 1, Waits: WaitsAsked, Since: "2026-08-01"},
		{Issue: 3, Waits: WaitsAsked, Since: "2026-10-01"},
		{Issue: 3, Waits: WaitsObsolete, Since: "2026-10-04"},
		{Issue: 9, Waits: WaitsAsked, Since: "2026-01-01"},
	}
	b := MakeBoard(open, 0, waits, nil, Config{NextMax: 5, StuckDays: 14}, now)
	var got []int
	for _, w := range b.Stuck {
		got = append(got, w.Issue)
	}
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 || b.Stuck[2].Waits != WaitsObsolete {
		t.Errorf("stuck = %+v", b.Stuck)
	}
}
