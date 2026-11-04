package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The workflow that applies the judge's findings grants checks: write; the
// answer, bounds and develop workflows also pass --from-merge-request, for
// their own run, and must not make doctor warn (#1033).
func TestApplyWorkflowReadsTheJudgeApplyWorkflow(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"workline.yml", "workline-apply.yml", "workline-answer.yml", "workline-bounds.yml", "workline-develop.yml"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "ci", "github", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if applies, granted := applyWorkflow(repo); !applies || !granted {
		t.Fatalf("templates as shipped: applies=%v granted=%v, want no warning", applies, granted)
	}

	apply := filepath.Join(dir, "workline-apply.yml")
	body, err := os.ReadFile(apply)
	if err != nil {
		t.Fatal(err)
	}
	old := string(body)
	trimmed := strings.Replace(old, "checks: write", "checks: read", 1)
	if trimmed == old {
		t.Fatal("workline-apply.yml has no checks: write to take out")
	}
	if err := os.WriteFile(apply, []byte(trimmed), 0o644); err != nil {
		t.Fatal(err)
	}
	if applies, granted := applyWorkflow(repo); !applies || granted {
		t.Fatalf("apply without checks: write: applies=%v granted=%v, want a warning", applies, granted)
	}
}
