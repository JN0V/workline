package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/routing"
)

// Copied file behind: given a repository whose copied workflow differs from
// the template of the engine's release beyond the pin, when the person runs
// workline doctor, then it names that file and the template to re-copy.
func TestWorkflowCopiedBehindTheTemplate(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("..", "..", "defaults", "ci", "github", "workline.yml"))
	if err != nil {
		t.Fatal(err)
	}
	m := pinLine.FindSubmatch(template)
	if m == nil {
		t.Fatal("the template workline.yml has no WORKLINE_VERSION pin")
	}
	engine := string(m[1])

	repo := t.TempDir()
	dir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The copy is pinned to the engine's release, yet differs from the
	// template by a line: it was re-pinned, not re-copied.
	behind := string(template) + "\n# a step of an older copy\n  - run: echo this copy is behind its template\n"
	if err := os.WriteFile(filepath.Join(dir, "workline.yml"), []byte(behind), 0o644); err != nil {
		t.Fatal(err)
	}

	line := &routing.Config{Events: map[string][]string{}}
	r := &Report{}
	ciTemplates(r, Options{Repo: repo, Engine: engine}, line)

	var found *Check
	for i, c := range r.Checks {
		if c.Where == ".github/workflows/workline.yml" {
			found = &r.Checks[i]
		}
	}
	if found == nil {
		t.Fatalf("a copy that differs from its template beyond the pin: %+v; want .github/workflows/workline.yml said", r.Checks)
	}
	if found.Level != Warn {
		t.Errorf("level %q, want %q", found.Level, Warn)
	}
	said := found.Message + " " + found.Fix
	if !strings.Contains(said, ".github/workflows/workline.yml") || !strings.Contains(said, "defaults/ci/github/workline.yml") {
		t.Errorf("the check says %q; want the copied file and the template to re-copy named", said)
	}
}
