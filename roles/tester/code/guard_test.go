package tester

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/work"
)

// guardBase is a repository whose base holds a library with `add` and
// its one passing test, and no routed gate: only the red proof runs.
// Git's global config is pointed at the null device.
func guardBase(t *testing.T) (repo, base string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("WORKLINE_EVENT", "")
	t.Setenv("WORKLINE_JUDGED", "")
	repo, git := gitRepo(t, map[string]string{"lib.sh": "add() { echo $(($1 + $2)); }\n"})
	for name, body := range map[string]string{
		"app/test_old.sh":       ". ./lib.sh\n[ \"$(add 1 1)\" = 2 ]\n",
		".workline/config.yaml": "routing: {events: {merge-request: []}}\n",
	} {
		p := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(repo, "add", ".")
	git(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "base: a library and its test")
	return repo, git(repo, "rev-parse", "HEAD")
}

// guardPost runs the tester's post on the scenarios of an issue adding
// mul beside add, kept as today, with the agent's intentions.
func guardPost(t *testing.T, intents []intent.Intention) (code int, runDir string) {
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
	issue := task{Issue: 7, Title: "Multiply", Base: base, Text: "Add a multiplication; addition stays as it is.", Scenarios: []work.Scenario{
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
	mulTest  = intent.Intention{Kind: "patch", Value: map[string]any{"file": "app/test_mul.sh", "content": ". ./lib.sh\n[ \"$(mul 2 3)\" = 6 ]\n"}}
	addGuard = intent.Intention{Kind: "patch", Value: map[string]any{"file": "app/test_add.sh", "content": ". ./lib.sh\n[ \"$(add 2 3)\" = 5 ]\n"}}
	kept     = intent.Intention{Kind: "covers", Value: map[string]any{"scenario": "Addition kept", "tests": []any{"app.test_add"}, "reason": "kept as today"}}
)

// Guard written: given a scenario kept as today that no existing test
// checks, and a scenario the change alters, when the tester's tests are
// proven on the base, then the test written to guard the kept scenario,
// green on the base, is accepted beside the red one, and the Draft names
// it for its scenario in the Tests table, the kept scenario listed
// nowhere as not tested.
func TestGuardTestGreenOnTheBaseIsKept(t *testing.T) {
	code, runDir := guardPost(t, []intent.Intention{mulTest, addGuard,
		{Kind: "covers", Value: map[string]any{"scenario": "Multiply two numbers", "tests": []any{"app.test_mul"}}}, kept})
	if code != 0 {
		said, _ := os.ReadFile(filepath.Join(runDir, "out", "verdict.yaml"))
		t.Fatalf("Post = %d, want 0: a guard test green on the base is accepted; its verdict:\n%s", code, said)
	}
	raw, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml"))
	if err != nil {
		t.Fatalf("no Draft named: %v", err)
	}
	var draft map[string]any
	if err := yaml.Unmarshal(raw, &draft); err != nil {
		t.Fatal(err)
	}
	body, _ := draft["body"].(string)
	for _, want := range []string{"| Multiply two numbers | `app.test_mul` |", "| Addition kept | `app.test_add` |"} {
		if !strings.Contains(body, want) {
			t.Errorf("the Draft's Tests table lacks %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "**Not tested**") {
		t.Errorf("the guarded scenario is listed as not tested:\n%s", body)
	}
}

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
