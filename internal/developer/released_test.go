package developer

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
)

// A claim let go takes its card out of Doing, back to the board's ready
// column (ADR-0057, ADR-0059): GitLab's column labels swapped in the same
// call, a GitHub project's Status set; with no board, the claim alone.
func TestReleasedCardLeavesDoing(t *testing.T) {
	s := SettingsFrom(nil)
	is := forge.Issue{ID: 4, Labels: []string{s.ClaimLabel, "Doing", "bug"}}
	v := released(is, s, &board.Move{Board: "Dev", Column: "To Do", Off: []string{"Doing", "Done"}})
	if add, _ := v["add"].([]string); !slices.Equal(add, []string{"To Do"}) || !slices.Equal(v["remove"].([]string), []string{s.ClaimLabel, "Doing"}) {
		t.Errorf("GitLab: %v", v)
	}
	v = released(is, s, &board.Move{Board: "Dev", Column: "Todo", Project: 3})
	if st, _ := v["status"].(map[string]any); st["project"] != 3 || st["option"] != "Todo" || !slices.Equal(v["remove"].([]string), []string{s.ClaimLabel}) {
		t.Errorf("GitHub: %v", v)
	}
	if v = released(is, s, nil); v["add"] != nil || v["status"] != nil {
		t.Errorf("no board: %v", v)
	}
}
