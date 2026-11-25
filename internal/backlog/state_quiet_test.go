package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Quiet by default
// Given a need opened by a person, read by the role before, that it proposes to refine with no question to ask and no assumption taken,
// When the role writes its state and labels the issue for the person,
// Then the role's comment is edited in place, no new comment posted on the person's issue.
func TestPersistEditsTheStateInPlaceWhenNoOneIsAsked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	state := forge.FakeState{Issues: []forge.FakeItem{{ID: 9, Title: "Rows lost", Body: "Some rows are lost.", Author: "zed", Insider: true, Labels: []string{},
		Comments: []forge.FakeComment{{Body: FormatState(State{Confirmed: "abc"}) + "\n\n" + StateMarker(role)}}}}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	refine := Pending{Issue: 9, Act: "refine", Line: "Refine #9.", Proposal: &Proposal{Do: "refine", Issue: 9, Need: "Every row exported."}}
	p := &Plan{Visit: []int{9}, Record: Record{Proposed: []Pending{refine}}, ledger: &Ledger{}}

	if err := p.Persist(f, role); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got forge.FakeState
	if err := json.Unmarshal(after, &got); err != nil {
		t.Fatal(err)
	}
	it := got.Issues[0]
	if it.Posted != 0 {
		t.Errorf("%d comments posted anew on the issue, want none: the state is edited in place", it.Posted)
	}
	if len(it.Comments) != 1 || !strings.Contains(it.Comments[0].Body, LabelProposed) {
		t.Errorf("the role's comment is not the one kept, edited with its new state: %+v", it.Comments)
	}
}
