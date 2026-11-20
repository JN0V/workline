package productowner

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// No question after the yes:
// Given an issue a person accepted, the reviewer's finding open in its
// Validation, a person's section,
// When the product owner is told the reviewer's findings,
// Then it is not told to ask the reporter about that hold.
func TestAcceptedHoldAsksTheReporterNothing(t *testing.T) {
	body := "## Need\n\nA calculator multiplies.\n\n## Verification\n\nThe product is right.\n\n## Validation\n\n- **Two by three:** Given 2 and 3, When multiplied, Then 6.\n\n## Scope\n\ncalc/calc.go\n"
	is := forge.Issue{ID: 1356, Title: "Mean", Author: "reporter", Body: body, Labels: []string{backlog.LabelAccepted}}
	r := &backlog.SpecReview{Body: backlog.BodyDigest(body), Round: 1, Open: 1, In: []string{"Validation"}}
	comments := []string{backlog.SpecMarker + "\nThe Validation names no failing case."}

	got := specFindings(is, &backlog.State{}, comments, r)
	if strings.Contains(got, "`ask` its reporter") {
		t.Errorf("a person accepted the issue, yet the product owner is told to ask its reporter:\n%s", got)
	}
}
