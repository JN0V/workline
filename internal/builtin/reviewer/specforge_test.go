package reviewer

import (
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
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

// A finding in a Verification a person wrote is theirs to answer: the
// comment never says the product owner rewrites it (#847).
func TestSpecCommentPersonsSectionIsTheirs(t *testing.T) {
	body := "## Need\n\nA.\n\n## Verification\n\nThe mean works well.\n\n## Validation\n\nC.\n\n## Scope\n\ncalc/calc.go\n"
	mine := revisable(forge.Issue{ID: 4, Body: body}, &backlog.State{})
	if len(mine) != 0 {
		t.Fatalf("a person's issue: the product owner's sections %v, want none", mine)
	}
	c := specComment(Review{Summary: "1 finding"}, backlog.SpecReview{Body: "b", Round: 1, Open: 1, In: []string{"Verification"}}, 5, mine)
	for _, want := range []string{"Verification, a person's: the product owner never rewrites it and asks its reporter", "accepts the issue as it reads with `workline:accepted`"} {
		if !strings.Contains(c, want) {
			t.Errorf("comment lacks %q:\n%s", want, c)
		}
	}
	if strings.Contains(c, "rewrites it at its next refine") {
		t.Errorf("comment says the product owner rewrites a person's section:\n%s", c)
	}
}

// A finding in a Verification the product owner wrote, as it wrote it, is
// its own to rewrite at its next refine, as before (#847).
func TestSpecCommentOwnersSectionIsRewritten(t *testing.T) {
	verification := "\nThe mean works well.\n\n"
	body := "## Need\n\nA.\n\n## Verification\n" + verification + "## Validation\n\nC.\n\n## Scope\n\ncalc/calc.go\n"
	st := &backlog.State{}
	st.SetWrote("Verification", backlog.BodyDigest(verification))
	mine := revisable(forge.Issue{ID: 4, Body: body}, st)
	if !slices.Equal(mine, []string{"Verification"}) {
		t.Fatalf("the product owner's sections %v, want Verification", mine)
	}
	c := specComment(Review{Summary: "1 finding"}, backlog.SpecReview{Body: "b", Round: 1, Open: 1, In: []string{"Verification"}}, 5, mine)
	if want := "Verification, the product owner's: it rewrites it at its next refine"; !strings.Contains(c, want) {
		t.Errorf("comment lacks %q:\n%s", want, c)
	}
	if strings.Contains(c, "a person's") {
		t.Errorf("comment says a person answers the product owner's section:\n%s", c)
	}
}
