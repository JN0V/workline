package reviewer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
)

// A spec review not asked, its prompt over the reviewer's budget (#2296):
// the record marks the body, the rounds kept, and the comment says why it
// waits and what unblocks it. A review not whole for another reason, or a
// whole one, carries no mark, a prior one dropped.
func TestSpecOverBudgetRecordedAndSaid(t *testing.T) {
	prior := &backlog.SpecReview{Body: "aaaaaaaaaaaa", Round: 2, Open: 1, In: []string{"Scope"}, Over: "cccccccccccc"}
	st := state{Spec: &Spec{Path: "#4", Text: "## Scope\n\nx", Digest: "bbbbbbbbbbbb", Round: 3, Prior: prior}}
	r := overRecord(specRecord(st, Review{Complete: false}), st, true)
	if r.Over != "bbbbbbbbbbbb" || r.Body != "" || r.Round != 2 || r.Open != 1 {
		t.Fatalf("over budget recorded %+v, want over=bbbbbbbbbbbb, round 2, 1 open, no body", r)
	}
	c := specComment(Review{Summary: "0 findings"}, r, 5, nil)
	for _, s := range []string{"spec review", "over budget", "smaller issue", "split", "larger budget", "over=bbbbbbbbbbbb"} {
		if !strings.Contains(c, s) {
			t.Errorf("comment does not say %q:\n%s", s, c)
		}
	}
	if strings.Contains(c, "reads it again at the next run") {
		t.Errorf("comment says it is read again:\n%s", c)
	}
	if r := overRecord(specRecord(st, Review{Complete: false}), st, false); r.Over != "" {
		t.Errorf("not whole for another reason: recorded over=%s", r.Over)
	}
	if c := specComment(Review{}, overRecord(specRecord(st, Review{Complete: false}), st, false), 5, nil); !strings.Contains(c, "reads it again at the next run") {
		t.Errorf("not whole for another reason, not said read again:\n%s", c)
	}
	if r := overRecord(specRecord(st, Review{Complete: true}), st, true); r.Over != "" || r.Body != "bbbbbbbbbbbb" {
		t.Errorf("whole review recorded %+v", r)
	}
}
