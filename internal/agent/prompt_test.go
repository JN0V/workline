package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
)

// Every role's output contract says how to write code and quotes so that
// they read (#138), and an answer asked for again repeats it.
func TestPromptAsksForBlockScalars(t *testing.T) {
	dir, run := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Review."), 0o644)
	os.MkdirAll(filepath.Join(run, "in"), 0o755)
	os.MkdirAll(filepath.Join(run, "out"), 0o755)
	os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("The change."), 0o644)
	req := Request{RunDir: run, Repo: t.TempDir(), Role: &role.Role{Name: "reviewer", Dir: dir, Intentions: []string{"finding"}}}
	_, user, err := Prompt(req)
	if err != nil || !strings.Contains(user, BlockScalars) {
		t.Errorf("the output contract does not ask for block scalars: %v\n%s", err, user)
	}
}

// An intention the engine applies but the agent is never shown the shape
// of is one no agent proposes: the product owner's refine went unproposed
// on DomoticsCore for that.
func TestEveryIntentionHasItsShape(t *testing.T) {
	for kind := range intent.Catalogue {
		if contracts[kind] == "" {
			t.Errorf("%s: no shape in the output contract (contracts, prompt.go)", kind)
		}
	}
}
