package documentalist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// uncoveredMessage commits a doc whose one line is docLine, with
// ci/w.yml holding source as its only source, lets its one part answer that
// it says nothing of the doc, and returns the "uncovered" finding's message.
func uncoveredMessage(t *testing.T, source, docLine string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
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
	write("ci/w.yml", source)
	write("docs/env.md", "---\nsources: ci/w.yml\n---\n# Env\n\n"+docLine+"\n")
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "add", "."},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "x"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
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
	return msg
}

// Name in a source: Given a doc line naming `workline`, which its source
// holds and no part cites, When the parts are read, Then the finding says
// no part cited it.
func TestNameInASource(t *testing.T) {
	msg := uncoveredMessage(t, "steps:\n  - {name: workline, action: prepare}\n", "The job runs `workline` here.")
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

// Name in no source: Given a doc line naming `prepare_all`, which its source
// does not hold, When the parts are read, Then the finding says it is
// renamed or gone from the code.
func TestNameInNoSource(t *testing.T) {
	msg := uncoveredMessage(t, "steps:\n  - {name: workline, action: prepare}\n", "The job runs `prepare_all` here.")
	if !strings.Contains(msg, "`prepare_all`") {
		t.Errorf("the finding does not name prepare_all: %s", msg)
	}
	if !strings.Contains(msg, "renamed or gone from the code") {
		t.Errorf("a name no source holds is not said renamed or gone: %s", msg)
	}
	if strings.Contains(msg, "no part cited") {
		t.Errorf("a name no source holds is said held: %s", msg)
	}
}

// Whole identifier: Given a doc line naming `run`, and a source holding
// only `running`, When the parts are read, Then the name is said renamed
// or gone: a longer word does not hold it.
func TestNameOnlyInsideALongerWord(t *testing.T) {
	msg := uncoveredMessage(t, "steps:\n  - {name: running, action: prepare}\n", "The job does `run` here.")
	if !strings.Contains(msg, "renamed or gone from the code") {
		t.Errorf("a name held only inside a longer word is not said renamed or gone: %s", msg)
	}
	if strings.Contains(msg, "no part cited") {
		t.Errorf("a name held only inside a longer word is said held: %s", msg)
	}
}
