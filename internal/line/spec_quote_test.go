package line

import (
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// commentsBacklog is a backlog whose issue holds the comments given;
// the rest of the interface is never called.
type commentsBacklog struct {
	forge.Backlog
	comments []string
}

func (c commentsBacklog) Comments(forge.Target) ([]string, error) { return c.comments, nil }

// Quote wakes nothing:
// Given an issue to refine whose step ran after the reviewer's review,
// When a person's comment quoting that review is added,
// Then its step is not due.
func TestQuoteWakesNothing(t *testing.T) {
	review := backlog.SpecMarker + "\nThe spec lacks a failing case; say what happens on an empty list."
	quote := "I disagree with this:\n\n> " + backlog.SpecMarker + "\n> The spec lacks a failing case; say what happens on an empty list.\n\nIt is covered."

	_, _, before, _, err := readRecordSpec(commentsBacklog{comments: []string{review}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, _, after, _, err := readRecordSpec(commentsBacklog{comments: []string{review, quote}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if before == "" {
		t.Fatalf("the reviewer's review gave no spec hash")
	}
	if after != before {
		t.Errorf("spec hash after a person's quote = %q, want %q (the review's)", after, before)
	}

	s := routing.State{Name: refineState}
	is := forge.Issue{ID: 1, Body: "the body"}
	rec := Record{State: refineState, Body: bodyHash(is.Body), Labeled: "t", Spec: before}
	if _, ok := due(s, is, "t", after, rec, true); ok {
		t.Errorf("the step is due after a person's quote of the review")
	}
}
