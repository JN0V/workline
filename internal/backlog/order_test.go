package backlog

import (
	"fmt"
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
