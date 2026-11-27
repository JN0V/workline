package line

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// TestBacklogDueUrgent: an Urgent issue (priority 1) with no state or in
// to-refine, labelled since the last backlog run started, makes one due,
// Ready full; never once read, further on, at priority 2, while a run
// waits or the runs are unread, nor with no product owner on schedule
// (#2322).
func TestBacklogDueUrgent(t *testing.T) {
	on := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 1}}\n"
	off := "routing: {events: {issue: [tester, developer]}, wip: {ready: 1}}\n"
	urgent := func(labels ...string) []forge.Issue {
		return []forge.Issue{{ID: 1, Labels: []string{"workline:ready"}}, {ID: 41, Labels: labels}}
	}
	at10 := map[int][]forge.LabelEvent{41: {{Label: "workline:priority/1", Added: true, Note: forge.Note{Created: "2026-11-16T10:00:00Z"}}}}
	at10p2 := map[int][]forge.LabelEvent{41: {{Label: "workline:priority/2", Added: true, Note: forge.Note{Created: "2026-11-16T10:00:00Z"}}}}
	before := "2026-11-16T09:00:00Z/2026-11-16T09:05:00Z"
	for _, c := range []struct {
		name   string
		config string
		open   []forge.Issue
		events map[int][]forge.LabelEvent
		last   string
		want   bool
	}{
		{"no state, no run before", on, urgent("workline:priority/1"), at10, "none", true},
		{"no state, labelled since the last run", on, urgent("workline:priority/1"), at10, before, true},
		{"to-refine, labelled since the last run", on, urgent("workline:priority/1", "workline:to-refine"), at10, before, true},
		{"read by the run since", on, urgent("workline:priority/1"), at10, "2026-11-16T10:05:00Z/2026-11-16T10:15:00Z", false},
		{"in ready", on, urgent("workline:priority/1", "workline:ready"), at10, before, false},
		{"in tests", on, urgent("workline:priority/1", "workline:needs-tests"), at10, before, false},
		{"priority 2", on, urgent("workline:priority/2"), at10p2, before, false},
		{"a run waits", on, urgent("workline:priority/1"), at10, "waiting", false},
		{"the runs unread", on, urgent("workline:priority/1"), at10, "unread", false},
		{"no product owner on schedule", off, urgent("workline:priority/1"), at10, before, false},
	} {
		due, said := BacklogDue(lineConfig(t, c.config), &labelBook{events: c.events}, c.open, c.last)
		if due != c.want || c.want && !strings.Contains(said, "#41") {
			t.Errorf("%s: BacklogDue = %v, %q; want %v, naming #41", c.name, due, said, c.want)
		}
	}
}
