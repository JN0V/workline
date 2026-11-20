package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// A role with no pre whose instruction.md is still at its root runs as today.
func TestSharedPreReadsTheRootInstructionWhenNotMoved(t *testing.T) {
	const text = "Answer the question below.\n"
	dir, run := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "instruction.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := sharedPre(&role.Role{Name: "judge", Dir: dir}, []string{"WORKLINE_RUN_DIR=" + run}); err != nil {
		t.Fatalf("a role with its instruction at the root does not run: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(run, "in", "task.md"))
	if err != nil {
		t.Fatal(err)
	}
	// What task.md holds is the engine's to choose (the instruction or a
	// pointer to it); here it must be written and name no error.
	if len(got) == 0 {
		t.Errorf("in/task.md is empty, want the task written")
	}
}

// With no instruction.md anywhere, the error names the file it looked for.
func TestSharedPreNamesTheMissingInstruction(t *testing.T) {
	dir, run := t.TempDir(), t.TempDir()
	err := sharedPre(&role.Role{Name: "judge", Dir: dir}, []string{"WORKLINE_RUN_DIR=" + run})
	if err == nil || !strings.Contains(err.Error(), "instruction.md") {
		t.Fatalf("error = %v, want one naming instruction.md", err)
	}
}
