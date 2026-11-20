package backlog

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Accepted by a person:
// Given a bug's parent a person accepted, with no green proof of its
// reproduction tests,
// When the line closes it,
// Then its tests are not merged unproven, and why is said on the parent,
// in the comment that closes it.
func TestAcceptedBugParentSaysItsTestsAreNotMerged(t *testing.T) {
	red := func(int) (Symptom, bool, error) {
		return Symptom{Issue: 3, Parts: []int{6, 7}, Why: "red on main"}, true, nil
	}
	p := &Plan{issues: map[int]forge.Issue{3: {ID: 3, Labels: []string{"bug"}, Children: []int{6, 7}}}, config: Config{Symptom: red}}
	c := Proposal{Do: "close", Issue: 3, Parent: true, Ticked: "ann", Why: "its 2 parts are closed"}
	p.bugParent(&c, nil)
	if c.Symptom != "" {
		t.Fatalf("a red proof is no symptom gone: %q", c.Symptom)
	}
	got := Comment(c, "product-owner")
	if !strings.Contains(got, "red on main") {
		t.Errorf("the comment on the accepted parent does not say why its tests have no green proof:\n%s", got)
	}
	if !strings.Contains(strings.ToLower(got), "merge") {
		t.Errorf("the comment on the accepted parent does not say its tests are not merged:\n%s", got)
	}
}
