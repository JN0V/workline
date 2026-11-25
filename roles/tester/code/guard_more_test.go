package tester

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/work"
)

// guardPost runs the tester's post on the scenarios of an issue adding
// mul beside add, kept as today, with the agent's intentions.
func guardPost(t *testing.T, intents []intent.Intention, labels ...string) (code int, runDir string) {
	t.Helper()
	repo, base := guardBase(t)
	runDir = t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	settings, err := json.Marshal(map[string]any{"test-command": gradleLike("1"), "test-report": "junit.xml", "test-paths": []string{"**/test_*.sh"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "settings.json"), settings, 0o644); err != nil {
		t.Fatal(err)
	}
	issue := task{Issue: 7, Title: "Multiply", Base: base, Labels: labels, Text: "Add a multiplication; addition stays as it is.", Scenarios: []work.Scenario{
		{Name: "Multiply two numbers", Text: "Given 2 and 3, When mul is called, Then it gives 6."},
		{Name: "Addition kept", Text: "Given 2 and 3, When add is called, Then it gives 5, as today."},
	}}
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", taskFile), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		t.Fatal(err)
	}
	return Post(runDir, repo), runDir
}

var (
	addGuard = intent.Intention{Kind: "patch", Value: map[string]any{"file": "app/test_add.sh", "content": ". ./lib.sh\n[ \"$(add 2 3)\" = 5 ]\n"}}
	kept     = intent.Intention{Kind: "covers", Value: map[string]any{"scenario": "Addition kept", "tests": []any{"app.test_add"}, "reason": "kept as today"}}
)

// Guards alone: given only a guard test, its other scenario said not
// tested, when the tests are proven on the base, then the guard is
// refused as any test green there: nothing shows what the base lacks.
func TestGuardsAloneAreGreenOnTheBase(t *testing.T) {
	code, runDir := guardPost(t, []intent.Intention{addGuard,
		{Kind: "covers", Value: map[string]any{"scenario": "Multiply two numbers", "reason": "a person reads it"}}, kept})
	said, _ := os.ReadFile(filepath.Join(runDir, "out", "verdict.yaml"))
	if !strings.Contains(string(said), "test-green-on-base") {
		t.Errorf("Post = %d, a guard alone not refused green on the base:\n%s", code, said)
	}
	if _, err := os.Stat(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		t.Error("a Draft opened with a guard alone")
	}
}

// A bug not shown: given a bug whose test of its scenario passes on the
// base, beside a guard, when the tests are proven, then it is not
// reproduced, as with no guard: marked needs-repro, no Draft.
func TestBugGreenBesideAGuardIsNotReproduced(t *testing.T) {
	greenMul := intent.Intention{Kind: "patch", Value: map[string]any{"file": "app/test_mul.sh", "content": ". ./lib.sh\n[ \"$(add 3 3)\" = 6 ]\n"}}
	_, runDir := guardPost(t, []intent.Intention{greenMul, addGuard,
		{Kind: "covers", Value: map[string]any{"scenario": "Multiply two numbers", "tests": []any{"app.test_mul"}}}, kept}, "bug")
	out, _ := os.ReadFile(filepath.Join(runDir, "out", "intentions.yaml"))
	if !strings.Contains(string(out), "workline:needs-repro") {
		t.Errorf("a bug with no red test beside its guard is not marked not reproduced:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		t.Error("a Draft opened for a bug not shown")
	}
}
