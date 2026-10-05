package backlog

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestParts(t *testing.T) {
	is := forge.Issue{ID: 9, Children: []int{12, 12, 9}, Body: "Text.\n\n- [ ] #30 not a part\n\n## Sub-issues\n\n- [x] #11\n- [ ] #12\n* [ ] #9"}
	if got := Parts(is); !slices.Equal(got, []int{11, 12}) {
		t.Errorf("Parts = %v, want [11 12]: the relation and the Sub-issues list; itself and other lists left out", got)
	}
	if got := SplitInto(is, &State{Split: []int{13, 11}}); !slices.Equal(got, []int{11, 12, 13}) {
		t.Errorf("SplitInto = %v, want [11 12 13]: the parts and the state's split", got)
	}
	if got := Parts(forge.Issue{ID: 5, Children: []int{7, 7}}); !slices.Equal(got, []int{7}) {
		t.Errorf("Parts = %v, want [7]: a child the forge lists twice is one part", got)
	}
	if got := Parts(forge.Issue{ID: 5, Children: []int{7, 7}}); !slices.Equal(got, []int{7}) {
		t.Errorf("Parts = %v, want [7]: a child the forge lists twice is one part", got)
	}
	if got := Parts(forge.Issue{ID: 4, Body: "No parts."}); len(got) != 0 {
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

func TestCloserNotReadIsSaid(t *testing.T) {
	part := Part{ID: 11, Issue: forge.Issue{Closed: true, Reason: "completed", Title: "Keep the last row"}, Unread: errors.New("the forge refused closers: unknown operation")}
	ev := ReadEvidence(forge.Issue{ID: 9}, []Part{part}, "product-owner")
	if !strings.Contains(ev.Body, "| #11 Keep the last row | closed as completed | not read: the forge did not say |") || strings.Contains(ev.Body, "by hand") {
		t.Errorf("a closer the forge refused to say is said not read, never by hand:\n%s", ev.Body)
	}
	if strings.Contains(ev.Body, "unknown operation") {
		t.Errorf("the forge's error stays out of the comment, in the finding:\n%s", ev.Body)
	}
	if len(ev.Unread) != 1 || ev.Unread[0].ID != 11 {
		t.Errorf("Unread = %v, want #11", ev.Unread)
	}
}
