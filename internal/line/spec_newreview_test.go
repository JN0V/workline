package line

import (
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// New review still wakes:
// Given an issue to refine whose step ran after the reviewer's review,
// When the reviewer writes a new review,
// Then the spec hash changes and the step is due again.
func TestNewReviewStillWakes(t *testing.T) {
	first := backlog.SpecMarker + "\nThe spec lacks a failing case."
	quote := "> " + backlog.SpecMarker + "\n> The spec lacks a failing case."
	second := backlog.SpecMarker + "\nThe spec now lacks a limit."

	_, _, before, _, err := readRecordSpec(commentsBacklog{comments: []string{first, quote}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, _, after, _, err := readRecordSpec(commentsBacklog{comments: []string{first, quote, second}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if before == "" || after == "" || after == before {
		t.Fatalf("spec hash before = %q, after a new review = %q, want them to differ", before, after)
	}

	s := routing.State{Name: refineState}
	is := forge.Issue{ID: 1, Body: "the body"}
	rec := Record{State: refineState, Body: bodyHash(is.Body), Labeled: "t", Spec: before}
	if _, ok := due(s, is, "t", after, rec, true); !ok {
		t.Errorf("the step is not due after a new review")
	}
}
