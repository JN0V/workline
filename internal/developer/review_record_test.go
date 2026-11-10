package developer

import "testing"

// The record the reviewer hides in its comment, one JSON list of its findings.
const findingsRecord = "<!-- workline:findings [{\"lens\":\"design\",\"holds\":\"warns\",\"where\":\"lib.sh:1\",\"severity\":\"Important\",\"text\":\"mul adds — it gives 5. Fix: multiply.\"}] -->\n"

// Record read: Given a reviewer comment with a record holding one
// Important finding, When the developer runs a round, Then it works on
// that finding.
func TestReviewRowsReadTheRecord(t *testing.T) {
	asked, nits := reviewRows(findingsRecord)
	if nits != 0 || len(asked) != 1 {
		t.Fatalf("asked %+v, nits %d: want the record's one finding", asked, nits)
	}
	if a := asked[0]; a.Holds != "warns" || a.Lens != "design" || a.Where != "lib.sh:1" || a.Key != "design|lib.sh|mul adds" {
		t.Errorf("finding: %+v", a)
	}
}

// Reworded table: Given the same record and a table with other words,
// When the round runs, Then the result is the same.
func TestReviewRowsIgnoreTheTablesWords(t *testing.T) {
	reworded := "| Level | Area | Place | Remark |\n|---|---|---|---|\n" +
		"| high | design | `lib.sh:1` | something else entirely |\n" + findingsRecord
	want, _ := reviewRows(findingsRecord)
	got, _ := reviewRows(reworded)
	if len(want) != 1 || len(got) != 1 || got[0].Key != want[0].Key || got[0].Holds != want[0].Holds || got[0].Where != want[0].Where {
		t.Errorf("reworded table: %+v, want %+v", got, want)
	}
}

// No record: Given a comment with no record, When the round runs, Then
// the developer does nothing and says why.
func TestReviewRowsWithNoRecordAskNothing(t *testing.T) {
	table := "| | | Where | Finding |\n|---|---|---|---|\n" +
		"| warns | design | `lib.sh:1` | Important: mul adds — it gives 5. |\n"
	asked, nits := reviewRows(table)
	if len(asked) != 0 || nits != 0 {
		t.Errorf("asked %+v, nits %d from a table alone: want nothing", asked, nits)
	}
}
