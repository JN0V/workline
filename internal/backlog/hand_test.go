package backlog

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// The kinds back to propose: the record's, a closing reopened since, an act
// undone since — each once.
func TestDemoted(t *testing.T) {
	h := &Hand{
		Record: Record{
			Propose: []string{"close-duplicate"},
			Closed:  []Closing{{Issue: 3, Act: "close-duplicate"}, {Issue: 4, Act: "close-obsolete"}, {Issue: 5, Act: "close-obsolete"}},
		},
		Undone: []Undo{{Issue: 7, Act: "rename"}, {Issue: 8, Act: "rename"}, {Issue: 9, Act: "close-duplicate"}},
	}
	open := []forge.Issue{{ID: 3}, {ID: 4}, {ID: 7}}
	got := h.Demoted(open)
	slices.Sort(got)
	if want := []string{"close-duplicate", "close-obsolete", "rename"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := (&Hand{}).Demoted(nil); len(got) != 0 {
		t.Errorf("nothing demoted: got %v", got)
	}
}
