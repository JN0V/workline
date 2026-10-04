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
