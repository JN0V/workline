package backlog

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// What waits on a person is what Persist labels: a proposal for a person,
// or a draft of the role's not ready; a question held for a later run, a
// label left on nothing, an issue set aside wait on no one (#430).
func TestWaiting(t *testing.T) {
	ask := Pending{Act: "ask", Capped: true}
	refine := Pending{Act: "refine"}
	draft := "## Need\n\n" + DraftMarker + "\n\nA need."
	l := &Ledger{States: map[int]*State{
		1: {Label: LabelProposed, Proposed: []Pending{refine}},
		2: {Label: LabelProposed, Proposed: []Pending{ask}},
		3: {Label: LabelProposed},
		4: {Label: LabelProposed},
		5: {Label: LabelProposed, Proposed: []Pending{refine}},
		6: {Proposed: []Pending{refine}},
	}, Aside: map[int]bool{5: true}}
	open := []forge.Issue{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4, Body: draft}, {ID: 5}, {ID: 6}}
	if got := l.Waiting(open); got != 2 {
		t.Errorf("Waiting = %d, want 2: #1 for its proposal, #4 for its draft", got)
	}
}
