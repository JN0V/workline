package role

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A kind of task may give the agent its own instruction and a few of the
// role's intentions (#878): the role's own stay whole, for what pre falls
// back on; one outside them, or a path for a facet, is refused at load.
func TestTaskInstructionAndIntentions(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "r"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(task string) {
		if err := os.WriteFile(filepath.Join(dir, "r", "role.yaml"), []byte("contract: 1\nname: r\nintentions: [kind, note, comment, close]\nmodel:\n  tier: standard\n  tasks:\n    kind: "+task+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("{instruction: tasks/kind.md, intentions: [kind, note], tier: light}")
	r, err := Load(dir, "r")
	if err != nil {
		t.Fatal(err)
	}
	k := r.ForTask("kind")
	if k.InstructionFacet != "tasks/kind.md" || k.Model.Tier != "light" || len(k.Intentions) != 4 {
		t.Errorf("for the kind task: facet %q, tier %q, intentions %v; want tasks/kind.md, light, the role's four", k.InstructionFacet, k.Model.Tier, k.Intentions)
	}
	if a := k.Asked(); !slices.Equal(a.Intentions, []string{"kind", "note"}) {
		t.Errorf("the agent asked answers with %v, want [kind note]", a.Intentions)
	}
	if o := r.ForTask("other"); o != r || o.Asked() != r {
		t.Error("a kind of task model.tasks does not name changes the role")
	}
	write("{intentions: [kind, ready]}")
	if _, err := Load(dir, "r"); err == nil || !strings.Contains(err.Error(), `model.tasks.kind.intentions names "ready"`) {
		t.Errorf("an intention outside the role's: %v, want it refused", err)
	}
	write("{instruction: ../x.md}")
	if _, err := Load(dir, "r"); err == nil || !strings.Contains(err.Error(), "is not a facet in tasks/") {
		t.Errorf("a path for a facet: %v, want it refused", err)
	}
}
