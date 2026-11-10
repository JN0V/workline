package tester

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestParseObjection(t *testing.T) {
	o, missing := parseObjection("workline:objection\n- **test:** `TestTwoByThree`\n- Scenario: Two by three\n- kind: Wrong interface\n- why: mul takes a list,\nnot two numbers.")
	if len(missing) > 0 || o.Test != "TestTwoByThree" || o.Scenario != "Two by three" || kindOf(o.Kind) != "interface" || o.Why != "mul takes a list,\nnot two numbers." {
		t.Fatalf("%+v, missing %v", o, missing)
	}
	_, missing = parseObjection("workline:objection\ntest: x\nwhy: too strict")
	if !slices.Equal(missing, []string{"scenario", "kind (one of: " + kindNames() + ")"}) {
		t.Fatalf("missing %v", missing)
	}
	if o, _ := parseObjection("workline:objection\nwhy: use the `validate`"); o.Why != "use the `validate`" {
		t.Fatalf("why %q", o.Why)
	}
	if o, _ := parseObjection("workline:objection\nscenario: **Two** by **three**"); o.Scenario != "**Two** by **three**" {
		t.Fatalf("scenario %q", o.Scenario)
	}
	if k := kindOf("the need itself"); k != "need" {
		t.Fatalf("kind %q", k)
	}
}

// The answer's task gives each scenario's declared tests, and how a
// rewrite that renames them says so: a covers entry.
func TestAnswerTaskGivesDeclaredTests(t *testing.T) {
	runDir, repo := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(runDir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "test_console.py"), []byte("def test_old(): pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := answer{Issue: 3, File: "test_console.py", Kind: "over-specified", Scenario: Scenario{Name: "Restart", Text: "Given …"},
		Record: record{Covers: map[string][]string{"Restart": {"tests.test_old"}, "Enabled console": {"tests.test_old"}}}}
	if code := writeAnswerTask(runDir, repo, a, forge.Thread{}, &forge.Issue{Title: "Console"}, Settings{}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "task.md"))
	if err != nil {
		t.Fatal(err)
	}
	task := string(data)
	for _, want := range []string{"- **Enabled console:** `tests.test_old`", "- **Restart:** `tests.test_old`", "- covers: {scenario: <its name>"} {
		if !strings.Contains(task, want) {
			t.Errorf("the task lacks %q:\n%s", want, task)
		}
	}
	if strings.Contains(task, "names the scenario") {
		t.Errorf("the task still asks the test to name its scenario:\n%s", task)
	}
}
