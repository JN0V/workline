package backlog

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestParts(t *testing.T) {
	is := forge.Issue{ID: 9, Children: []int{12}, Body: "Text.\n\n- [ ] #30 not a part\n\n## Sub-issues\n\n- [x] #11\n- [ ] #12\n* [ ] #9"}
	if got := Parts(is, &State{Split: []int{13, 11}}); !slices.Equal(got, []int{11, 12, 13}) {
		t.Errorf("Parts = %v, want [11 12 13]: the relation, the Sub-issues list, the state; itself and other lists left out", got)
	}
	if got := Parts(forge.Issue{ID: 4, Body: "No parts."}, nil); len(got) != 0 {
		t.Errorf("Parts = %v, want none", got)
	}
}

func TestVerificationItems(t *testing.T) {
	cases := []struct {
		body string
		want []string
	}{
		{"## Verification\n\nConformance cases:\n- a parent gets one comment;\n1. A test passes.\n- [ ] A box item\n\n## Scope\n\nx", []string{"a parent gets one comment;", "A test passes.", "A box item"}},
		{"### Verification\n\nA test writes three rows\nand reads them back.", []string{"A test writes three rows and reads them back."}},
		{"## Verification\n\n_No response_", nil},
		{"No sections.", nil},
	}
	for _, c := range cases {
		if got := VerificationItems(c.body); !slices.Equal(got, c.want) {
			t.Errorf("VerificationItems(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

func TestProofIsAQuote(t *testing.T) {
	done := Part{ID: 11, Issue: forge.Issue{Closed: true, Reason: "completed", Body: "## Verification\n\nA **test** writes `three` rows."}}
	if proof("A test writes three rows;", []Part{done}) == "" {
		t.Error("a quote, emphasis, case and final punctuation aside, should prove the item")
	}
	if proof("A test writes four rows.", []Part{done}) != "" {
		t.Error("other words prove nothing")
	}
	refused := done
	refused.Issue.Reason = "not_planned"
	if proof("A test writes three rows.", []Part{refused}) != "" {
		t.Error("a part not delivered proves nothing")
	}
}
