package line

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// TestReadyBelowTarget: Ready under its limit asks for a backlog run
// (#1549), at or above it does not; neither with Ready's limit lifted, nor
// with the product owner off the schedule line.
func TestReadyBelowTarget(t *testing.T) {
	ready := func(n int) []forge.Issue {
		open := []forge.Issue{{ID: 100, Labels: []string{"workline:developing"}}}
		for i := range n {
			open = append(open, forge.Issue{ID: i + 1, Labels: []string{"workline:ready"}})
		}
		return open
	}
	po := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
	for _, c := range []struct {
		name, config string
		ready        int
		want         bool
	}{
		{"9 of 10", po, 9, true},
		{"10 of 10", po, 10, false},
		{"11 of 10", po, 11, false},
		{"the shipped limit, 5", "routing: {events: {schedule: [product-owner], issue: [tester, developer]}}\n", 4, true},
		{"Ready's limit lifted", "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 0}}\n", 0, false},
		{"no product owner on the schedule", "routing: {events: {issue: [tester, developer]}, wip: {ready: 10}}\n", 0, false},
	} {
		if got := readyBelow(lineConfig(t, c.config), ready(c.ready)); got != c.want {
			t.Errorf("%s: readyBelow = %v, want %v", c.name, got, c.want)
		}
	}
}
