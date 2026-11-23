package engine

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A person's comment is answered by their own yes put at or after it,
// still so once a bot (the line, on a push) took that yes off; not once
// a person, or someone the forge does not name, took it off; never by a
// yes put before it, nor by another person's.
func TestAnsweredByTheirYes(t *testing.T) {
	yes := func(by, at string) forge.LabelEvent {
		return forge.LabelEvent{Label: "workline:accepted", Added: true, Writer: true,
			Note: forge.Note{Author: by, Insider: true, Created: at}}
	}
	off := func(by string, bot bool, at string) forge.LabelEvent {
		return forge.LabelEvent{Label: "workline:accepted", Note: forge.Note{Author: by, Bot: bot, Created: at}}
	}
	whole := func(by, at string) forge.LabelEvent {
		e := yes(by, at)
		e.Label = "workline:accepted-whole"
		return e
	}
	trace := forge.Note{Author: "dana", Insider: true, Created: "2026-01-04T09:59:58Z"}
	for _, c := range []struct {
		name   string
		labels []forge.LabelEvent
		want   bool
	}{
		{"their yes after it, on", []forge.LabelEvent{yes("dana", "2026-01-04T10:00:00Z")}, true},
		{"taken off by the line on a push", []forge.LabelEvent{yes("dana", "2026-01-04T10:00:00Z"), off("workline-bot", true, "2026-01-04T10:30:00Z")}, true},
		{"taken off by them", []forge.LabelEvent{yes("dana", "2026-01-04T10:00:00Z"), off("dana", false, "2026-01-04T10:30:00Z")}, false},
		{"taken off by whom the forge does not say", []forge.LabelEvent{yes("dana", "2026-01-04T10:00:00Z"), off("", false, "2026-01-04T10:30:00Z")}, false},
		{"taken off by them, put again", []forge.LabelEvent{yes("dana", "2026-01-04T10:00:00Z"), off("dana", false, "2026-01-04T10:30:00Z"), yes("dana", "2026-01-04T10:40:00Z")}, true},
		{"swapped for the whole yes by them", []forge.LabelEvent{yes("dana", "2026-01-04T10:00:00Z"), whole("dana", "2026-01-04T10:05:00Z"), off("dana", false, "2026-01-04T10:05:00Z")}, true},
		{"their yes before it", []forge.LabelEvent{yes("dana", "2026-01-04T09:00:00Z")}, false},
		{"another person's yes", []forge.LabelEvent{yes("lee", "2026-01-04T10:00:00Z")}, false},
		{"no yes", nil, false},
	} {
		if got := answeredByTheirYes(trace, c.labels); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
