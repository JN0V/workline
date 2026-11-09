package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// An agent's failure, quoted in the agent-unavailable finding, holds ***
// in place of a token's value (#1215).
func TestAgentFailureMasked(t *testing.T) {
	t.Setenv("NPM_TOKEN", "fake-value-for-test-1215")
	if got := lastLine("starting\nauth failed for fake-value-for-test-1215\n"); got != "auth failed for ***" {
		t.Errorf("lastLine = %q", got)
	}
	a, err := Parse(`cmd:echo "auth failed for $NPM_TOKEN" >&2; exit 1`)
	if err != nil {
		t.Fatal(err)
	}
	dir, run := t.TempDir(), t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Judge."), 0o644)
	_ = os.MkdirAll(filepath.Join(run, "in"), 0o755)
	_ = os.MkdirAll(filepath.Join(run, "out"), 0o755)
	_ = os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("The change."), 0o644)
	_, err = a.Propose(Request{RunDir: run, Repo: t.TempDir(), Role: &role.Role{Name: "committer", Dir: dir, Intentions: []string{"note"}}})
	if err == nil || !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "fake-value") || !strings.Contains(err.Error(), "auth failed for ***") {
		t.Errorf("Propose = %v", err)
	}
}
