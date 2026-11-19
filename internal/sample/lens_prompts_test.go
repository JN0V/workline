package sample

import (
	"os"
	"path/filepath"
	"testing"
)

// A lens's question is read where the reviewer reads its lens: under
// prompts/ once the role holds one, the project's own first.
func TestLensQuestionReadsAMovedRolesPrompts(t *testing.T) {
	roles, repo := t.TempDir(), t.TempDir()
	lens := "---\njudge:\n  question: Is it tested?\n---\nLook for tests.\n"
	path := filepath.Join(roles, "reviewer", "prompts", "lenses", "tests.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(lens), 0o644); err != nil {
		t.Fatal(err)
	}
	if q := lensQuestion(repo, roles, "tests"); q != "Is it tested?" {
		t.Errorf("a moved role's lens: question %q", q)
	}
}
