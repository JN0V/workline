package engine

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Every variable the engine hands a role's pre and post is the line's
// own: a project's command run from post, its proof, never gets one
// (#953). Read from the engine's source, each "WORKLINE_X=" it sets, so
// one added there and not to forge.LineEnv fails here, whatever option
// sets it.
func TestScriptEnvIsTheLinesOwn(t *testing.T) {
	src, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	set := regexp.MustCompile(`"(WORKLINE_[A-Z_]+)=`).FindAllStringSubmatch(string(src), -1)
	if len(set) < 15 {
		t.Fatalf("engine.go sets %d WORKLINE_ variables, want 15 at least: the pattern no longer reads it", len(set))
	}
	for _, m := range set {
		if !slices.Contains(forge.LineEnv, m[1]) {
			t.Errorf("the engine hands its roles %s, not in forge.LineEnv: a project's command would read it", m[1])
		}
	}
	t.Setenv(forge.RunsDirEnv, "/runs")
	env := scriptEnv("/runs/1-developer", "developer", Options{Event: "issue", AI: "claude", Forge: "github",
		Target: &forge.Target{Kind: "issue", ID: 3}})
	for _, kv := range forge.ProjectEnv(env) {
		if name, _, _ := strings.Cut(kv, "="); slices.Contains(forge.LineEnv, name) {
			t.Errorf("ProjectEnv kept %s", kv)
		}
	}
}
