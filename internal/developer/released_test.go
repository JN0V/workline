package developer

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
)

// A claim let go takes its card out of Doing, back to the board's ready
// column (ADR-0057, ADR-0059): GitLab's column labels swapped in the same
// call, a GitHub project's Status set; with no board, the labels alone.
// Each puts the ready label back on, which the claim took off (#1081).
func TestReleasedCardLeavesDoing(t *testing.T) {
	s := SettingsFrom(nil)
	is := forge.Issue{ID: 4, Labels: []string{s.ClaimLabel, "Doing", "bug"}}
	v := released(is, s, &board.Move{Board: "Dev", Column: "To Do", Off: []string{"Doing", "Done"}})
	if add, _ := v["add"].([]string); !slices.Equal(add, []string{backlog.LabelReady, "To Do"}) || !slices.Equal(v["remove"].([]string), []string{s.ClaimLabel, "Doing"}) {
		t.Errorf("GitLab: %v", v)
	}
	v = released(is, s, &board.Move{Board: "Dev", Column: "Todo", Project: 3})
	if st, _ := v["status"].(map[string]any); st["project"] != 3 || st["option"] != "Todo" || !slices.Equal(v["remove"].([]string), []string{s.ClaimLabel}) {
		t.Errorf("GitHub: %v", v)
	}
	if v = released(is, s, nil); !slices.Equal(v["add"].([]string), []string{backlog.LabelReady}) || !slices.Equal(v["remove"].([]string), []string{s.ClaimLabel}) || v["status"] != nil {
		t.Errorf("no board: %v", v)
	}
}

// The claim moves its issue from Ready to Doing, one state at a time
// (ADR-0059, §2; #1081): the ready label off in the same call, with or
// without a board.
func TestClaimTakesReadyOff(t *testing.T) {
	s := SettingsFrom(nil)
	in := claimLabel(Act{Issue: 2}, s)
	v, _ := in.Value.(map[string]any)
	if !slices.Equal(v["add"].([]string), []string{s.ClaimLabel}) || !slices.Equal(v["remove"].([]string), []string{backlog.LabelReady}) {
		t.Errorf("no board: %v", v)
	}
	in = claimLabel(Act{Issue: 2, Board: &board.Move{Board: "Dev", Column: "Doing", Off: []string{"To Do"}}}, s)
	v, _ = in.Value.(map[string]any)
	if !slices.Equal(v["add"].([]string), []string{s.ClaimLabel, "Doing"}) || !slices.Equal(v["remove"].([]string), []string{backlog.LabelReady, "To Do"}) {
		t.Errorf("GitLab: %v", v)
	}
	in = claimLabel(Act{Issue: 2, Board: &board.Move{Board: "Dev", Column: "In Progress", Project: 3}}, s)
	v, _ = in.Value.(map[string]any)
	if !slices.Equal(v["remove"].([]string), []string{backlog.LabelReady}) || v["status"] == nil {
		t.Errorf("GitHub: %v", v)
	}
}
