package engine

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/role"
)

// Each step on its own: Given a role with built-in pre code and no post,
// When it runs, Then its code runs for pre and the shared post runs
// after. The role folder holds no pre nor post file.
func TestScriptBuiltinPreThenSharedPost(t *testing.T) {
	const name = "builtin-pre-only-fixture"
	if !slices.Contains(contract.Builtins().Names(), name) {
		contract.Join(name, contract.Role{Pre: func(runDir, repo string) int {
			if err := os.MkdirAll(filepath.Join(runDir, "in"), 0o755); err != nil {
				return 1
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("from the built-in pre\n"), 0o644); err != nil {
				return 1
			}
			return 0
		}})
	}
	roleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(roleDir, "instruction.md"), []byte("the instruction file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &role.Role{Name: name, Dir: roleDir}
	runDir, repo := t.TempDir(), t.TempDir()
	env := append(os.Environ(), "WORKLINE_RUN_DIR="+runDir)

	code, err := script(r, "pre", repo, env)
	if err != nil || code != 0 {
		t.Fatalf("pre: code %d, error %v; want 0, none", code, err)
	}
	task, _ := os.ReadFile(filepath.Join(runDir, "in", "task.md"))
	if string(task) != "from the built-in pre\n" {
		t.Errorf("pre left in/task.md = %q, want what the built-in pre wrote, not the shared pre's instruction", task)
	}

	code, err = script(r, "post", repo, env)
	if err != nil || code != 0 {
		t.Fatalf("post: code %d, error %v; want 0, none", code, err)
	}
	verdict, _ := os.ReadFile(filepath.Join(runDir, "out", "verdict.yaml"))
	if string(verdict) != "status: pass\n" {
		t.Errorf("post left out/verdict.yaml = %q, want the shared post's pass", verdict)
	}
}
