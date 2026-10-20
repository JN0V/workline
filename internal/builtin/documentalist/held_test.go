package documentalist

import (
	"errors"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// Whom a doc due holding the release waits on is read from the note its
// finding ends with, each note a person must act on saying personJudges
// (#543): a note reworded without it would be counted as the agent's.
func TestHeldBy(t *testing.T) {
	held := "src/a.go changed since it was checked:\nabc1234 feat: a\n(made suspect since v1.0.0: the release waits until it is judged — by the agent, `workline docs`, or a person moving `checked`)"
	for _, c := range []struct {
		name, note, want string
	}{
		{"sources too wide", tooWideNote, heldForAPerson},
		{"a source that cannot be read", unreadNote(errors.New("gone")), heldForAPerson},
		{"too large, parts off", "(too large to be put before the agent — the doc with what changed in its sources, or with its sources as they are now: " + personJudges + ")", heldForAPerson},
		{"left for the next gardening", partsLater, heldForParts},
		{"not put before the agent in this round", "(not put before the agent in this round: over ai-max-calls or the task's size, or the round judges docs in parts or writes records)", heldForAgent},
		{"no note: the agent was not asked", "", heldForAgent},
	} {
		msg := held
		if c.note != "" {
			msg += "\n" + c.note
		}
		if got := heldBy(verdict.Finding{Rule: "suspect", Where: "docs/a.md", Level: "block", Message: msg}); got != c.want {
			t.Errorf("%s: heldBy = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestHeldSummary(t *testing.T) {
	got := heldSummary(map[string]string{"docs/c.md": heldForAgent, "docs/a.md": heldForAPerson, "docs/b.md": heldForAgent})
	want := "3 docs due hold the release: 1 for a person (docs/a.md), 2 for the agent, on a later round or run — see the findings"
	if got != want {
		t.Errorf("heldSummary = %q, want %q", got, want)
	}
	if got := heldSummary(map[string]string{"docs/a.md": heldForParts}); got != "1 doc due holds the release: 1 for the next gardening, in parts — see the findings" {
		t.Errorf("heldSummary, one doc = %q", got)
	}
}
