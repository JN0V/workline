package sample

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/role"
)

// The titles the engine gives the merge requests it opens, the weekly
// sample's and each role's default, pass the committer's rules once a
// squash adds " (#N)" (ADR-0051). The documentalist's: its own test.
func TestEngineTitlesPassTheSquashRule(t *testing.T) {
	r, err := role.Load("../../roles", "committer")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(r.MergedSettings(&role.ProjectConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	var s committer.Settings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	titles := []string{ResetTitle}
	roles, err := os.ReadDir("../../roles")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range roles {
		if d.IsDir() {
			titles = append(titles, engine.DefaultTitle(d.Name()))
		}
	}
	for _, title := range titles {
		found, err := committer.Check(title+" (#12345)", s)
		if err != nil {
			t.Fatal(err)
		}
		if len(found) > 0 {
			t.Errorf("%q fails the committer's rules once squashed: %v", title, found)
		}
	}
}
