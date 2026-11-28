package backlog

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A review that could not be asked on a body, the reviewer's prompt over
// its budget (#2296), records that body: read back, it holds the issue as
// over budget, said with what unblocks it, on that body alone; the body
// changed, the spec waits on a review again.
func TestSpecOverBudgetHeldOnItsBodyAlone(t *testing.T) {
	is := forge.Issue{Body: "## Need\n\nA.\n\n## Scope\n\na.go"}
	digest := BodyDigest(is.Body)
	r := SpecReview{Round: 1, Over: digest, Stopped: false}
	got, err := ReadSpecReview([]string{r.String() + "\n\n" + SpecMarker})
	if err != nil || got == nil || got.Over != digest || got.Body != "" || got.Round != 1 || got.Stopped {
		t.Fatalf("read back %+v (%v) from %q", got, err, r.String())
	}
	stopped := SpecReview{Body: "3f9a1c0e2b7d", Over: digest, Stopped: true, In: []string{"Scope"}}
	if got, err := ReadSpecReview([]string{stopped.String() + "\n\n" + SpecMarker}); err != nil || got == nil || got.Over != digest || !got.Stopped {
		t.Fatalf("read back %+v (%v) from %q", got, err, stopped.String())
	}
	comments := []string{r.String() + "\n\n" + SpecMarker}
	rule, why := SpecHold(is, comments)
	if rule != SpecOverBudget {
		t.Fatalf("held %q, want %q", rule, SpecOverBudget)
	}
	for _, s := range []string{"spec review", "over budget", "smaller issue", "split", "larger budget"} {
		if !strings.Contains(why, s) {
			t.Errorf("why %q does not say %q", why, s)
		}
	}
	if _, cleared := SpecCleared(is, comments); cleared {
		t.Error("an over-budget review cleared the spec")
	}
	edited := forge.Issue{Body: is.Body + "\n\nb.go"}
	if rule, _ := SpecHold(edited, comments); rule != "spec-not-reviewed" {
		t.Errorf("its body changed: held %q, want spec-not-reviewed", rule)
	}
	if rule, _ := SpecHold(forge.Issue{Body: is.Body, Labels: []string{LabelAccepted}}, comments); rule != "" {
		t.Errorf("accepted by a person: held %q", rule)
	}
	// A record with no over= written as before.
	old := "<!-- workline:spec-review body=d424ff914f28 round=1 open=2 in=Verification,Scope -->"
	if s := (SpecReview{Body: "d424ff914f28", Round: 1, Open: 2, In: []string{"Verification", "Scope"}}).String(); s != old {
		t.Fatalf("written %q, want %q", s, old)
	}
}
