package developer

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// A claim let go puts the ready label back on, which the claim took off
// (#1081).
func TestReleasedPutsReadyBack(t *testing.T) {
	s := SettingsFrom(nil)
	v := released(forge.Issue{ID: 4, Labels: []string{s.ClaimLabel, "bug"}}, s)
	if !slices.Equal(v["add"].([]string), []string{backlog.LabelReady}) || !slices.Equal(v["remove"].([]string), []string{s.ClaimLabel}) {
		t.Errorf("released: %v", v)
	}
}

// The claim moves its issue from Ready to Doing, one state at a time
// (#1081): the ready label off in the same call.
func TestClaimTakesReadyOff(t *testing.T) {
	s := SettingsFrom(nil)
	v, _ := claimLabel(devmark.Act{Issue: 2}, s).Value.(map[string]any)
	if !slices.Equal(v["add"].([]string), []string{s.ClaimLabel}) || !slices.Equal(v["remove"].([]string), []string{backlog.LabelReady}) {
		t.Errorf("claim: %v", v)
	}
}
