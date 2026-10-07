package backlog

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestSpecReviewReadsBack(t *testing.T) {
	r := SpecReview{Body: "3f9a1c0e2b7d", Round: 2, Open: 1, In: []string{"Verification", "Scope"}, Stopped: true}
	got := ReadSpecReview([]string{"state", "findings\n\n" + r.String() + "\n\n" + SpecMarker})
	if got == nil || got.Body != r.Body || got.Round != 2 || got.Open != 1 || !slices.Equal(got.In, r.In) || !got.Stopped {
		t.Fatalf("read back %+v from %q", got, r.String())
	}
	if ReadSpecReview([]string{r.String()}) != nil {
		t.Fatal("a record without the reviewer's marker read")
	}
}

func TestSectionAt(t *testing.T) {
	body := "Intro.\n\n## Need\n\nA.\n\n### Verification\n\nB."
	for line, want := range map[int]string{1: "", 3: "Need", 5: "Need", 7: "Verification", 9: "Verification"} {
		if got := SectionAt(body, line); got != want {
			t.Errorf("line %d: %q, want %q", line, got, want)
		}
	}
}

func TestRevisableOnlyTheRolesOwn(t *testing.T) {
	is := forge.Issue{Body: "## Need\n\n" + DraftLine("product-owner") + "\n\nA.\n\n## Verification\n\nB.\n\n## Validation\n\nC.\n\n## Scope\n\nD."}
	st := &State{Wrote: map[string]string{"Verification": BodyDigest("B."), "Scope": BodyDigest("not D.")}}
	r := &SpecReview{In: []string{"Need", "Verification", "Validation", "Scope"}}
	if got := Revisable(is, st, r); !slices.Equal(got, []string{"Need", "Verification"}) {
		t.Fatalf("revisable %v, want the draft and the text as written", got)
	}
}
