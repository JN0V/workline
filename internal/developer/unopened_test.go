package developer

import (
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/intent"
)

// Its pull request not opening gives the claim back the first time, the
// reason on the issue, and blocks the issue the second (#856): retried
// once, never in a loop.
func TestUnopenedGivesBackThenBlocks(t *testing.T) {
	s := SettingsFrom(nil)
	repo := t.TempDir()
	first := unopened(Act{Issue: 159}, s, repo)
	label, _ := first[0]["label"].(map[string]any)
	comment, _ := first[1]["comment"].(map[string]any)
	body, _ := comment["body"].(string)
	if !slices.Equal(label["remove"].([]string), []string{s.ClaimLabel}) || !slices.Equal(label["add"].([]string), []string{backlog.LabelReady}) || label["card"] != "ready" {
		t.Errorf("first: label %v", label)
	}
	if comment["sticky"] != "claim" || !strings.Contains(body, "gives this issue back") || !strings.Contains(body, intent.WhyMark) {
		t.Errorf("first: comment %v", comment)
	}
	if a, ok := ReadAct(body); !ok || a.Act != "unopened" || a.Unopened != 1 {
		t.Errorf("first: record %+v in %q", a, body)
	}
	second := unopened(Act{Issue: 159, Unopened: 1}, s, repo)
	label, _ = second[0]["label"].(map[string]any)
	comment, _ = second[1]["comment"].(map[string]any)
	body, _ = comment["body"].(string)
	if !slices.Contains(label["add"].([]string), backlog.LabelBlocked) || !slices.Contains(label["remove"].([]string), s.ClaimLabel) || !slices.Contains(label["remove"].([]string), backlog.LabelReady) {
		t.Errorf("second: label %v", label)
	}
	if a, ok := ReadAct(body); !ok || a.Unopened != 2 || !strings.Contains(body, "blocks this issue") {
		t.Errorf("second: %+v in %q", a, body)
	}
}
