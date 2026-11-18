package developer

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A p1 is not held by a p2.
// Given a priority-2 issue held with an open pull request changing a
// file, when a ready priority-1 issue names that file in its Scope and
// the developer runs, then the developer takes the priority-1 issue.
func TestPickTakesPriorityOneBesideAHeldPriorityTwo(t *testing.T) {
	s := SettingsFrom(nil)
	issues := []forge.Issue{
		{ID: 1, Labels: []string{"workline:ready", s.ClaimLabel, "workline:priority/2"}, Body: "## Scope\n\n- `internal/doctor/doctor.go`\n"},
		{ID: 2, Labels: []string{"workline:ready", "workline:priority/1"}, Body: "## Scope\n\n- `internal/doctor/doctor.go`\n"},
	}
	coded := []forge.MergeRequest{{ID: 7, Title: "feat: doctor", Body: "Closes #1"}}
	files := []string{"internal/doctor/doctor.go"}
	id, passed := pick(issues, coded, s, 0, false, nil, nil, files, nil)
	if id != 2 {
		t.Errorf("pick = #%d, passed %v: want the priority-1 issue #2", id, passed)
	}
	if len(passed) != 0 {
		t.Errorf("passed over %v: want none", passed)
	}
}
