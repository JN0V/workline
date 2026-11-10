package reviewer

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// The summary comment hides each finding the table shows, and each nit,
// with its severity and text apart, for the developer's rounds to read
// whatever the table's words (#705). A rule's finding has no severity; a
// decision is asked apart, never in the record.
func TestSummaryHoldsTheFindingsRecord(t *testing.T) {
	important := Finding{Lens: "correctness", Where: "lib.sh:2", Severity: "important", Title: "mul adds", Why: "mul 2 3 gives 5", Fix: "multiply"}
	nit := Finding{Lens: "style", Where: "lib.sh:2", Severity: "nit", Title: "name the operands", Why: "$1 reads poorly"}
	v := Review{Summary: "2 findings on the change", Change: []Finding{important, nit}, Findings: []verdict.Finding{
		{Rule: "correctness", Where: "lib.sh:2", Level: "block", Message: message(important)},
		{Rule: "style", Where: "lib.sh:2", Level: "warn", Message: message(nit)},
		{Rule: "gate:lint", Where: "a.go:9", Level: "block", Message: "gate lint did not pass --> 1 new finding"},
		{Rule: "decision", Where: "lib.sh:1", Level: "warn", Message: "which rounding?"},
	}}
	body := summaryComment(v, 0, nil, nil)
	m := regexp.MustCompile(`(?m)^<!-- workline:findings (\[.*\]) -->$`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no findings record:\n%s", body)
	}
	var got []findingJSON
	if err := json.Unmarshal([]byte(m[1]), &got); err != nil {
		t.Fatalf("the record does not parse: %v\n%s", err, m[1])
	}
	want := []findingJSON{
		{Lens: "correctness", Holds: "blocks", Where: "lib.sh:2", Severity: "Important", Text: "mul adds — mul 2 3 gives 5 Fix: multiply"},
		{Lens: "gate:lint", Holds: "blocks", Where: "a.go:9", Text: "gate lint did not pass --> 1 new finding"},
		{Lens: "style", Holds: "warns", Where: "lib.sh:2", Severity: "Nit", Text: "name the operands — $1 reads poorly"},
	}
	if len(got) != len(want) {
		t.Fatalf("record %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("finding %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	if strings.Contains(m[1], "-->") {
		t.Errorf("the record ends its comment early: %s", m[1])
	}
}
