package reviewer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// A comment's key: lines added above keep it, a line changed gets a new
// one, two lines reading alike stay apart by which one they are, two
// lenses apart by their names; a line not read is keyed by its number, so
// two of a file gone at the head do not share a key (#277).
func TestLineKey(t *testing.T) {
	before := []string{"package a", "", "func A() int {", "\treturn 1", "}", "func B() int {", "\treturn 1", "}"}
	moved := append([]string{"// A comment above.", ""}, before...)
	changed := append([]string{}, before...)
	changed[3] = "\treturn 2"
	key := func(lines []string, n int, by string) string { return lineKeyOf(lines, "a.go", n, by) }
	if key(before, 4, "correctness") != key(moved, 6, "correctness") {
		t.Error("a line moved by lines added above got a new key")
	}
	if key(before, 4, "correctness") == key(changed, 4, "correctness") {
		t.Error("a line changed kept its key")
	}
	if key(before, 4, "correctness") == key(before, 7, "correctness") {
		t.Error("two lines reading alike share a key")
	}
	if key(before, 4, "correctness") == key(before, 4, "tests") {
		t.Error("two lenses on one line share a key")
	}
	if key(nil, 3, "correctness") == key(nil, 9, "correctness") {
		t.Error("two lines of a file not read share a key")
	}
	if k := key(before, 4, "a--lens"); strings.Contains(k, "--") || !strings.HasPrefix(k, "a-lens/") {
		t.Errorf("a key holding a run of dashes would end its marker: %s", k)
	}
}

// The summary comment's review region holds each finding in full, for an
// engine that posts no comment on a line; one that does replaces it.
func TestSummaryReviewRegion(t *testing.T) {
	lines := []forge.LineComment{{Path: "a.go", Line: 4, Key: "correctness/x", Body: "**Important: A returns 1**\n\nwhy it fails"}}
	body := summaryComment(Review{Summary: "1 findings on the change"}, 0, nil, lines)
	from, end := strings.Index(body, forge.ReviewHere), strings.Index(body, forge.ReviewEnd)
	if from < 0 || end < from || !strings.Contains(body[from:end], "> why it fails") || !strings.Contains(body[from:end], "`a.go:4`") {
		t.Fatalf("the region does not hold the finding in full:\n%s", body)
	}
	filled := forge.FillReview(body, "**On their lines**: 1 comment")
	if strings.Contains(filled, "why it fails") || strings.Contains(filled, "workline:review") || !strings.Contains(filled, "**On their lines**: 1 comment\n\n| ") && !strings.Contains(filled, "**On their lines**: 1 comment\n\nNo finding") {
		t.Errorf("filled:\n%s", filled)
	}
}

// A nit is taken apart by the very finding it became: an important one
// of the same lens on the same line, warning, stays in the table
// (ADR-0051, cut 4).
func TestApartNitsKeepsAnImportantFindingAlike(t *testing.T) {
	nit := Finding{Lens: "tests", Where: "a.go:3", Severity: "nit", Title: "named better", Why: "it reads poorly"}
	important := Finding{Lens: "tests", Where: "a.go:3", Severity: "important", Title: "no test fails", Why: "nothing asserts it"}
	v := Review{Change: []Finding{nit}, Findings: []verdict.Finding{
		{Rule: "tests", Where: "a.go:3", Level: "warn", Message: message(important)},
		{Rule: "tests", Where: "a.go:3", Level: "warn", Message: message(nit)},
	}}
	rest, nits := apartNits(v)
	if len(rest.Findings) != 1 || rest.Findings[0].Message != message(important) {
		t.Errorf("the table holds %v, want the important finding only", rest.Findings)
	}
	if len(nits) != 1 || nits[0].Message != message(nit) {
		t.Errorf("the nits are %v, want the nit only", nits)
	}
}
