package backlog

import (
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// The tester writes an issue's Verification from its acceptance tests, as
// a human tester writes the test plan (ADR-0066): which test proves which
// scenario. The product owner writes none.

// TesterMarker opens the Verification the tester wrote: a reader does
// not see it; the tester rewrites its own, never another's.
var TesterMarker = forge.Marker("verification=tester")

// TestersVerification says whether an issue's Verification is the
// tester's own: written by it, marked so.
func TestersVerification(body string) bool {
	return strings.Contains(work.ByMeaning(body)[work.Verification].Text, TesterMarker)
}

// WithVerification is a body with text as its Verification, the tester's:
// added where it is missing or empty, its own rewritten; a Verification
// another wrote — a person, or a product owner before ADR-0066 — is kept
// word for word, and changed is false.
func WithVerification(body, text string) (out string, changed bool) {
	old := work.ByMeaning(body)[work.Verification]
	if t := strings.TrimSpace(old.Text); t != "" && !strings.Contains(t, TesterMarker) {
		return body, false
	}
	c := Proposal{Verification: TesterMarker + "\n\n" + strings.TrimSpace(text)}
	if strings.TrimSpace(old.Text) != "" {
		c.Revise = []string{old.Heading}
	}
	out, added, _ := Refine(body, c, TesterRole)
	return out, len(added) > 0
}
