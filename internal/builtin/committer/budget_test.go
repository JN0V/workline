package committer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
)

// A refused message's task, its diff cut at maxDiff, fits the role's
// context budget at the measured ratio (#235), with room for the message,
// a stat of a hundred files and an answer asked again: a refused prompt
// leaves the commit-msg hook with no rewrite.
func TestLargestTaskFitsTheBudget(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // the role's own facets, none of this machine's
	r, err := role.Load("../../../roles", "committer")
	if err != nil {
		t.Fatal(err)
	}
	run := t.TempDir()
	os.MkdirAll(filepath.Join(run, "in"), 0o755)
	os.MkdirAll(filepath.Join(run, "out"), 0o755)
	stat := strings.Repeat(" internal/builtin/committer/committer.go | 12 +++++++-----\n", 100)
	task := strings.Repeat("m", 1000) + stat + capText(strings.Repeat("+\tif err != nil {\n", maxDiff), maxDiff)
	os.WriteFile(filepath.Join(run, "in", "task.md"), []byte(task), 0o644)
	os.WriteFile(filepath.Join(run, "out", "feedback.md"), []byte(strings.Repeat("x", 1000)), 0o644)
	if _, _, err := agent.Prompt(agent.Request{RunDir: run, Repo: t.TempDir(), Role: r}); err != nil {
		t.Errorf("the largest task is refused: %v", err)
	}
}
