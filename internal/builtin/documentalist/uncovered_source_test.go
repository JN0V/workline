package documentalist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// Name in a source: Given a doc line naming `workline`, which its source
// holds and no part cites, When the parts are read, Then the finding says
// no part cited it.
func TestNameInASource(t *testing.T) {
	repo, runDir := t.TempDir(), t.TempDir()
	write := func(p, text string) {
		t.Helper()
		full := filepath.Join(repo, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ci/w.yml", "steps:\n  - {name: workline, action: prepare}\n")
	write("docs/env.md", "---\nsources: ci/w.yml\n---\n# Env\n\nThe job runs `workline` here.\n")
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "add", "."},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "x"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// The one part answers that it says nothing of the doc.
	part := filepath.Join(runDir, "in", "parts", "01-"+partSlug("docs/env.md")+"-1")
	if err := os.MkdirAll(part, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(part, "answer.yaml"), []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKLINE_PARTS", "answered")

	sd := &suspectDoc{doc: &Doc{Path: "docs/env.md", Sources: []string{"ci/w.yml"}}}
	var findings []verdict.Finding
	total := 0
	if _, _, _, _, _, err := partsFor(runDir, repo, []*suspectDoc{sd}, Settings{}, &total, &findings); err != nil {
		t.Fatal(err)
	}
	var msg string
	for _, f := range findings {
		if f.Rule == "uncovered" {
			msg = f.Message
		}
	}
	if msg == "" {
		t.Fatalf("no uncovered finding: %v", findings)
	}
	if !strings.Contains(msg, "`workline`") {
		t.Errorf("the finding does not name workline: %s", msg)
	}
	if !strings.Contains(msg, "no part cited") {
		t.Errorf("the finding does not say no part cited it: %s", msg)
	}
	if strings.Contains(msg, "renamed or gone from the code") {
		t.Errorf("a name its source holds is not said renamed or gone: %s", msg)
	}
}
