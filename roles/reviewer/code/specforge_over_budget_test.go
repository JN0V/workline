package reviewer

import (
	"os"
	"path/filepath"
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

// One lens over budget marks the body only when every lens without an
// answer was over it (#2296): another lens failing for a transient reason
// leaves the body to be read again.
func TestSpecOverBudgetOnlyWhenEveryUnansweredIs(t *testing.T) {
	for _, tc := range []struct {
		name string
		why  map[string]string
		want bool
	}{
		{"all over budget", map[string]string{"edge-cases": "over-budget: x", "intent": "over-budget: y"}, true},
		{"one over budget, one unavailable", map[string]string{"edge-cases": "over-budget: x", "intent": "unavailable: down"}, false},
		{"one over budget, one not asked", map[string]string{"edge-cases": "over-budget: x"}, false},
		{"none over budget", map[string]string{"edge-cases": "unavailable: down", "intent": "unavailable: down"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runDir := t.TempDir()
			st := state{Lenses: []string{"edge-cases", "intent"}, Spec: &Spec{Path: "#4", Digest: "bbbbbbbbbbbb"}}
			for _, lens := range st.Lenses {
				dir := filepath.Join(runDir, "in", "parts", sharePart(st, lens, 1))
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if why, ok := tc.why[lens]; ok {
					if err := os.WriteFile(filepath.Join(dir, "unanswered"), []byte(why), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			c, _, err := lensAnswers(runDir, t.TempDir(), Settings{}, st)
			if err != nil {
				t.Fatal(err)
			}
			if c.OverBudget != tc.want {
				t.Errorf("over budget %v, want %v (failed %v)", c.OverBudget, tc.want, c.Failed)
			}
		})
	}
}
