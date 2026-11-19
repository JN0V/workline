package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// movedRoleTask runs sharedPre on a role named name with no pre, whose
// folder holds only prompts/instruction.md, and returns the task it wrote
// and the error it gave.
func movedRoleTask(t *testing.T, name, text string) (string, error) {
	t.Helper()
	dir, run := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prompts", "instruction.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	err := sharedPre(&role.Role{Name: name, Dir: dir}, []string{"WORKLINE_RUN_DIR=" + run})
	got, _ := os.ReadFile(filepath.Join(run, "in", "task.md"))
	return string(got), err
}

// Judge moved
// Given the judge's instruction.md in roles/judge/prompts/,
// When the judge runs,
// Then it reads that instruction and runs.
func TestSharedPreReadsTheMovedJudgesInstruction(t *testing.T) {
	const text = "Answer the question below about what the role produced.\n"
	got, err := movedRoleTask(t, "judge", text)
	if err != nil {
		t.Fatalf("the judge, its instruction in prompts/, does not run: %v", err)
	}
	if got != text {
		t.Errorf("in/task.md holds %q, want the instruction in prompts/: %q", got, text)
	}
}

// Auditor moved
// Given the auditor's instruction.md in roles/auditor/prompts/,
// When the auditor runs,
// Then it reads that instruction and runs.
func TestSharedPreReadsTheMovedAuditorsInstruction(t *testing.T) {
	const text = "Audit what the project holds against its spec.\n"
	got, err := movedRoleTask(t, "auditor", text)
	if err != nil {
		t.Fatalf("the auditor, its instruction in prompts/, does not run: %v", err)
	}
	if got != text {
		t.Errorf("in/task.md holds %q, want the instruction in prompts/: %q", got, text)
	}
}
