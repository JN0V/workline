package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Only draft repeats the text: Given an issue whose only draft section is an
// Example that repeats the text above it, When the refine runs, Then its
// result says the draft was left out for repeating what is above, not that
// the sections already exist.
func TestOnlyDraftRepeatsTheText(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	top := "Exports lose their last row when the day has three sales, and the file ends one row short."
	path := filepath.Join(t.TempDir(), "forge.json")
	state := forge.FakeState{Issues: []forge.FakeItem{{ID: 9, Author: "zed", Insider: true, Labels: []string{}, Body: top}}}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	p := &Plan{issues: map[int]forge.Issue{9: {ID: 9, Author: "zed", Insider: true, Body: top}}}
	c := Proposal{Do: "refine", Issue: 9, Example: "Exports lose their last row when the day has three sales: the file ends one row short."}
	rule, why := p.checkRefining(f, t.TempDir(), "product-owner", &c)
	if rule != "nothing-to-refine" {
		t.Fatalf("rule = %q (%s), want nothing-to-refine", rule, why)
	}
	if strings.Contains(why, "in the issue already") || !strings.Contains(why, "repeat") || !strings.Contains(why, "above") {
		t.Errorf("result = %q, want the draft's repetition of what is above named as the reason", why)
	}
}
