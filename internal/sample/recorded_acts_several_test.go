package sample

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
)

// Several records in one comment
// Given an issue comment holding three act records of a role without a
// draw of its own,
// When the weekly sample reads the week's acts,
// Then all three acts are in the pool it draws from.
func TestRecordedActsSeveralInOneComment(t *testing.T) {
	kinds := []string{"plan", "split", "close"}
	body := "**The architect's record**\n"
	for _, k := range kinds {
		a := engine.Act{Role: "architect", Kind: k, Run: "run-1", Day: "2026-03-02",
			Commit: "c0ffee-" + k, Models: []string{"m1"}, Items: []engine.ActItem{}}
		body += "\n" + a.Line()
	}
	state := forge.FakeState{Issues: []forge.FakeItem{{
		ID: 7, Title: "t", Labels: []string{},
		Comments: []forge.FakeComment{{Body: body}},
	}}, MergeRequests: []forge.FakeItem{}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path, Repo: t.TempDir()}

	acts, unread, err := recordedActs(f, f, "2026-03-01", "2026-03-08")
	if err != nil {
		t.Fatalf("recordedActs: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("unread: %v, want none", unread)
	}
	if len(acts) != len(kinds) {
		t.Fatalf("recordedActs gave %d acts, want %d: %+v", len(acts), len(kinds), acts)
	}
	for i, k := range kinds {
		a := acts[i]
		if a.Role != "architect" || a.Kind != k || a.Ref != "#7" || a.Day != "2026-03-02" || a.Commit != "c0ffee-"+k {
			t.Errorf("act %d = %+v, want architect %s on #7 of 2026-03-02 at commit c0ffee-%s", i, a, k, k)
		}
	}
	keys := map[string]bool{}
	for _, a := range acts {
		keys[a.key] = true
	}
	if len(keys) != len(kinds) {
		t.Errorf("the acts share a key, so a draw would take them for one: %+v", acts)
	}
}
