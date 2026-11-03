package engine

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Every variable the engine hands a role's pre and post is the line's
// own: a project's command run from post, its proof, never gets one
// (#953). A variable added to scriptEnv and not to forge.LineEnv fails
// here, before a proof's engines take it for theirs.
func TestScriptEnvIsTheLinesOwn(t *testing.T) {
	t.Setenv(forge.RunsDirEnv, "/runs")
	own := len(os.Environ())
	env := scriptEnv("/runs/1-developer", "developer", Options{
		Event: "issue", AI: "claude", RolesDir: "/roles", Forge: "github", aiWaits: AIWaitsReady,
		Target: &forge.Target{Kind: "issue", ID: 3},
	})
	added := env[own:] // what scriptEnv adds to the process's own
	if len(added) < 9 {
		t.Fatalf("scriptEnv added %d variables, want 9: %q", len(added), added)
	}
	for _, kv := range added {
		if name, _, _ := strings.Cut(kv, "="); !slices.Contains(forge.LineEnv, name) {
			t.Errorf("scriptEnv hands post %s, not in forge.LineEnv: a project's command would read it", name)
		}
	}
	for _, kv := range forge.ProjectEnv(env) {
		if strings.HasPrefix(kv, forge.RunsDirEnv+"=") || strings.HasPrefix(kv, "WORKLINE_RUN_DIR=") {
			t.Errorf("ProjectEnv kept %s", kv)
		}
	}
}
