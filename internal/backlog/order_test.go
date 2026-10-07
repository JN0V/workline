package backlog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// The backlog's order: the nearest milestone in version order, none last;
// then the priority, none after 4; then the lowest number.
func TestOrder(t *testing.T) {
	issues := []forge.Issue{
		{ID: 1},
		{ID: 2, Labels: []string{PriorityLabel(1)}},
		{ID: 3, Milestone: "v1.10.0"},
		{ID: 4, Milestone: "v1.9.0", Labels: []string{PriorityLabel(4)}},
		{ID: 5, Milestone: "v1.9.0", Labels: []string{PriorityLabel(2)}},
		{ID: 6, Milestone: "v1.9.0"},
		{ID: 7, Milestone: "v1.9.0", Labels: []string{PriorityLabel(2)}},
	}
	Order(issues)
	var got []int
	for _, is := range issues {
		got = append(got, is.ID)
	}
	if want := "[5 7 4 6 3 2 1]"; fmt.Sprint(got) != want {
		t.Fatalf("order = %v, want %s", got, want)
	}
}

// Milestones ranked by their due date, the earlier first, one with a date
// before one without; the title, as a version, when neither has one or
// both the same.
func TestOrderByDueDate(t *testing.T) {
	issues := []forge.Issue{
		{ID: 1, Milestone: "v1.9.0"},
		{ID: 2, Milestone: "v1.10.0"},
		{ID: 3, Milestone: "Autumn", MilestoneDue: "2026-11-30"},
		{ID: 4, Milestone: "v2.0.0", MilestoneDue: "2026-10-31"},
		{ID: 5, Milestone: "v1.8.0", MilestoneDue: "2026-11-30"},
		{ID: 6},
	}
	Order(issues)
	if got, want := ids(issues), "[4 3 5 1 2 6]"; got != want {
		t.Fatalf("order = %s, want %s", got, want)
	}
	ms := []forge.Milestone{{Title: "v1.9.0"}, {Title: "v3.0.0", Due: "2026-12-01"}, {Title: "v2.0.0", Due: "2026-11-01"}}
	if got := NextMilestone(t.TempDir(), ms); got != "v2.0.0" {
		t.Fatalf("next milestone = %q, want v2.0.0", got)
	}
}

func ids(issues []forge.Issue) string {
	var got []int
	for _, is := range issues {
		got = append(got, is.ID)
	}
	return fmt.Sprint(got)
}

// A blocked issue comes after its open blockers, whatever its labels; a
// blocker not among them — closed — holds nothing back (ADR-0028).
func TestOrderBlocked(t *testing.T) {
	issues := []forge.Issue{
		{ID: 1, Labels: []string{PriorityLabel(1)}, BlockedBy: []int{3}},
		{ID: 2, Labels: []string{PriorityLabel(2)}, Body: "Blocked by #9."},
		{ID: 3, Labels: []string{PriorityLabel(4)}},
	}
	if cycles := Order(issues); len(cycles) > 0 || ids(issues) != "[2 3 1]" {
		t.Fatalf("order = %s, cycles %v; want [2 3 1], none", ids(issues), cycles)
	}
}

// A cycle is reported once and the order still ends, its first by Less
// placed first.
func TestOrderCycle(t *testing.T) {
	issues := []forge.Issue{
		{ID: 1, Body: "Blocked by #2."},
		{ID: 2, BlockedBy: []int{1}},
		{ID: 3, Body: "Blocked by #1"},
	}
	cycles := Order(issues)
	if fmt.Sprint(cycles) != "[[1 2 1]]" || ids(issues) != "[1 2 3]" {
		t.Fatalf("order = %s, cycles %v", ids(issues), cycles)
	}
	if CycleText(cycles[0]) != "#1 waits on #2, #2 waits on #1" {
		t.Fatal(CycleText(cycles[0]))
	}
	// An issue that only waits on a cycle is never placed to break it.
	issues = []forge.Issue{{ID: 1, BlockedBy: []int{2}}, {ID: 2, BlockedBy: []int{3}}, {ID: 3, BlockedBy: []int{2}}}
	if cycles := Order(issues); fmt.Sprint(cycles) != "[[2 3 2]]" || ids(issues) != "[2 1 3]" {
		t.Fatalf("order = %s, cycles %v", ids(issues), cycles)
	}
}

// The report's waiting part reads the backlog as the run leaves it: an
// issue the run closed holds nothing back, a link the run set holds.
func TestReportWaiting(t *testing.T) {
	p := &Plan{issues: map[int]forge.Issue{
		1: {ID: 1, Title: "a"}, 2: {ID: 2, Title: "b", BlockedBy: []int{1}},
		3: {ID: 3, Title: "c", Labels: []string{LabelReady}},
	}, added: map[int][]int{3: {2}},
		Decisions: []Decision{{Mode: Act, Act: Proposal{Do: "close", Issue: 1}}}}
	got := p.waiting()
	if !strings.Contains(got, "- #3 c waits on #2.") || strings.Contains(got, "#2 b waits") || strings.Contains(got, "**Next**") {
		t.Fatal(got)
	}
}

// The engine's line is rewritten with a blocker added; a person's is kept.
func TestWithBlockers(t *testing.T) {
	body := WithBlockers("Text.\n\nblocked by: #4", []int{7})
	body = WithBlockers(body, []int{5, 7})
	want := "Text.\n\nblocked by: #4\n\nBlocked by #5, #7. " + BlockedByMarker
	if body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
	if got := fmt.Sprint(Blockers(forge.Issue{ID: 9, Body: body, BlockedBy: []int{6}})); got != "[4 5 6 7]" {
		t.Fatal(got)
	}
}
