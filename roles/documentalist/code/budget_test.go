package documentalist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
)

// The largest task the role writes, at whole-chars' most, fits its context
// budget at the measured ratio (#235): a refused prompt would leave the
// night's gardening unjudged. A feedback of 5,000 characters is room for
// an answer asked again.
func TestLargestTaskFitsTheBudget(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // the role's own facets, none of this machine's
	r, err := role.Load("../../../roles", "documentalist")
	if err != nil {
		t.Fatal(err)
	}
	run := t.TempDir()
	os.MkdirAll(filepath.Join(run, "in"), 0o755)
	os.MkdirAll(filepath.Join(run, "out"), 0o755)
	line := "\tif n, err := f(x[i]); err != nil {\n"
	largest := taskChars(Settings{WholeChars: wholeCharsMax}) // whole-chars at the most it may be
	os.WriteFile(filepath.Join(run, "in", "task.md"), []byte(strings.Repeat(line, largest/len(line))), 0o644)
	os.WriteFile(filepath.Join(run, "out", "feedback.md"), []byte(strings.Repeat("x", 5000)), 0o644)
	if _, _, err := agent.Prompt(agent.Request{RunDir: run, Repo: t.TempDir(), Role: r}); err != nil {
		t.Errorf("the largest task is refused: %v", err)
	}
}
