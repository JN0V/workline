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

// TestersToWrite says whether the tester writes an issue's Verification
// in place of the one there: none, its own, or one the product owner
// wrote before ADR-0066, as it wrote it (its state po's `wrote`). A
// person's, or one a person edited, is kept.
func TestersToWrite(body string, po *State) bool {
	t := work.ByMeaning(body)[work.Verification].Text
	return strings.TrimSpace(t) == "" || strings.Contains(t, TesterMarker) || mine(po, work.Verification, t)
}

// WithVerification is a body with text as its Verification, the tester's:
// added where it is missing or empty, its own or the product owner's
// (TestersToWrite) rewritten; a person's is kept word for word, and
// changed is false.
func WithVerification(body, text string, po *State) (out string, changed bool) {
	if !TestersToWrite(body, po) {
		return body, false
	}
	old := work.ByMeaning(body)[work.Verification]
	c := Proposal{Verification: TesterMarker + "\n\n" + strings.TrimSpace(text)}
	if strings.TrimSpace(old.Text) != "" {
		c.Revise = []string{old.Heading}
	}
	out, added, _ := Refine(body, c, TesterRole)
	return out, len(added) > 0
}
