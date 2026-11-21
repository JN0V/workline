package tick

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A succeeded `workline gardening docs and backlog` run is the template's
// Sunday run: it is a docs run too, so docs is not listed again after it.
func TestDueDocsAndBacklogRunCountsAsDocs(t *testing.T) {
	po := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
	sunday := succeeded(run("workline gardening docs and backlog", "workline-gardening.yml", "completed", "2026-11-15T18:10:00Z"))
	var runs = slices.DeleteFunc(slices.Clone(done), func(r forge.Run) bool { return r.Name == "workline gardening docs" })
	runs = append(runs, sunday)
	r := Due(setUp(t, po, 10, runs, ""))
	if got := jobsOf(r); slices.Contains(got, "docs") {
		t.Errorf("jobs %v, want no docs; said %v, unread %v", got, r.Said, r.Unread)
	}
}
