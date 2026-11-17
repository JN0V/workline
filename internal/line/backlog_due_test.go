package line

import (
	"errors"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// labelBook is a backlog whose issues' label events are given: the
// rest of forge.Backlog is never called.
type labelBook struct {
	forge.Backlog
	events map[int][]forge.LabelEvent
	err    error
	read   []int
}

func (l *labelBook) LabelEvents(id int) ([]forge.LabelEvent, error) {
	l.read = append(l.read, id)
	return l.events[id], l.err
}

func readyEvent(added bool, at string) forge.LabelEvent {
	return forge.LabelEvent{Label: "workline:ready", Added: added, Note: forge.Note{Created: at}}
}

// TestBacklogDue: with Ready below its target, a backlog run is due when
// Ready dropped since the last one started, or that run readied an issue;
// not when it readied none and none left Ready since, said so (#1549).
func TestBacklogDue(t *testing.T) {
	cfg := lineConfig(t, "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n")
	open := []forge.Issue{
		{ID: 1, Labels: []string{"workline:ready"}},
		{ID: 2, Labels: []string{"workline:developing"}},
		{ID: 3, Labels: []string{"workline:to-refine"}}, // upstream of Ready: never read
	}
	last := "2026-11-16T10:00:00Z/2026-11-16T10:20:00Z"
	for _, c := range []struct {
		name   string
		events map[int][]forge.LabelEvent
		last   string
		err    error
		want   bool
		said   string
	}{
		{"Ready dropped since the last run", map[int][]forge.LabelEvent{
			1: {readyEvent(true, "2026-11-01T09:00:00Z")},
			2: {readyEvent(true, "2026-11-01T09:00:00Z"), readyEvent(false, "2026-11-16T11:00:00Z")},
		}, last, nil, true, "dropped"},
		{"the last run readied one, Ready still below", map[int][]forge.LabelEvent{
			1: {readyEvent(true, "2026-11-16T10:10:00Z")},
		}, last, nil, true, "readied an issue"},
		{"the last run readied none, Ready unchanged", map[int][]forge.LabelEvent{
			1: {readyEvent(true, "2026-11-01T09:00:00Z")},
			2: {readyEvent(true, "2026-11-01T09:00:00Z"), readyEvent(false, "2026-11-16T09:00:00Z")},
		}, last, nil, false, "readied none, and none has left Ready since"},
		{"no backlog run before", nil, "none", nil, true, "no backlog run before"},
		{"one waits already", nil, "waiting", nil, false, "waits already"},
		{"the runs not listed", nil, "unread", nil, false, "could not be read"},
		{"the labels not read", nil, last, errors.New("API rate limit"), false, "could not be read"},
	} {
		b := &labelBook{events: c.events, err: c.err}
		due, said := backlogDue(cfg, b, open, c.last)
		if due != c.want || !strings.Contains(said, c.said) {
			t.Errorf("%s: backlogDue = %v, %q; want %v, saying %q", c.name, due, said, c.want, c.said)
		}
		for _, id := range b.read {
			if id == 3 {
				t.Errorf("%s: #3, upstream of Ready, read", c.name)
			}
		}
	}
}
