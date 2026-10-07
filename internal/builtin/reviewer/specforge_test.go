package reviewer

import (
	"testing"

	"github.com/JN0V/workline/internal/backlog"
)

// A review not whole keeps the last record's round and findings open, so
// partial findings never reset the rounds (#128).
func TestSpecRecordNotWholeKeepsTheRounds(t *testing.T) {
	prior := &backlog.SpecReview{Body: "aaaaaaaaaaaa", Round: 3, Open: 2, In: []string{"Scope"}}
	st := state{Spec: &Spec{Path: "#4", Text: "## Scope\n\nx", Digest: "bbbbbbbbbbbb", Round: 4, Prior: prior}}
	r := specRecord(st, Review{Complete: false})
	if r.Body != "" || r.Round != 3 || r.Open != 2 || len(r.In) != 1 {
		t.Fatalf("record %+v, want round 3, 2 open, no body", r)
	}
	if r := specRecord(st, Review{Complete: true}); r.Body != "bbbbbbbbbbbb" || r.Round != 4 || r.Open != 0 {
		t.Fatalf("whole review recorded %+v", r)
	}
}
