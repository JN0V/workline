package backlog

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// The kinds back to propose: the record's, a closing reopened since — not
// one still closed —, an act undone since (found undone already: no issue
// state read here) — each once.
func TestDemoted(t *testing.T) {
	h := &Hand{
		Record: Record{
			Propose: []string{"split"},
			Closed: []Closing{
				{Issue: 3, Act: "close-duplicate"}, {Issue: 4, Act: "close-duplicate"}, // both reopened: once
				{Issue: 5, Act: "close-obsolete"}, // still closed: not demoted
				{Issue: 6, Act: "split"},          // reopened, its kind demoted already: once
			},
		},
		Undone: []Undo{{Issue: 7, Act: "rename"}, {Issue: 8, Act: "rename"}, {Issue: 9, Act: "split"}},
	}
	open := []forge.Issue{{ID: 3}, {ID: 4}, {ID: 6}, {ID: 7}}
	got := h.Demoted(open)
	slices.Sort(got)
	if want := []string{"close-duplicate", "rename", "split"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := (&Hand{}).Demoted(nil); len(got) != 0 {
		t.Errorf("nothing demoted: got %v", got)
	}
}
