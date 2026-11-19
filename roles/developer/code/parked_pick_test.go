package developer

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

const parkedValidation = "## Validation\n\n- **Two by three:** Given 2 and 3, When multiplied, Then 6.\n\n## Scope\n\n- `internal/doctor/doctor.go`\n"

// Parked ready issue.
// Given an open issue labelled workline:ready and workline:parked, and
// another ready one not parked, when the developer picks, then it
// takes the one not parked, and passes the parked one over for no
// reason: it is never offered.
func TestPickLeavesAParkedReadyIssue(t *testing.T) {
	s := SettingsFrom(nil)
	issues := []forge.Issue{
		{ID: 1, Labels: []string{"workline:ready", "workline:parked"}, Body: parkedValidation},
		{ID: 2, Labels: []string{"workline:ready"}, Body: parkedValidation},
	}
	id, passed := pick(issues, nil, s, 0, false, nil, nil, nil, nil)
	if id != 2 {
		t.Errorf("pick = #%d, passed %v: want #2, the parked #1 left alone", id, passed)
	}
}

// Unparked.
// Given a ready issue parked, when a person removes workline:parked,
// then the developer's next pick takes it as before: left alone while
// parked, taken once the label is off.
func TestPickTakesAnIssueAgainOnceUnparked(t *testing.T) {
	s := SettingsFrom(nil)
	parked := []forge.Issue{{ID: 1, Labels: []string{"workline:ready", "workline:parked"}, Body: parkedValidation}}
	if id, passed := pick(parked, nil, s, 0, false, nil, nil, nil, nil); id != 0 {
		t.Fatalf("pick = #%d, passed %v: the parked issue was taken", id, passed)
	}
	unparked := []forge.Issue{{ID: 1, Labels: []string{"workline:ready"}, Body: parkedValidation}}
	if id, passed := pick(unparked, nil, s, 0, false, nil, nil, nil, nil); id != 1 {
		t.Errorf("pick = #%d, passed %v: want #1 once workline:parked is off", id, passed)
	}
}
