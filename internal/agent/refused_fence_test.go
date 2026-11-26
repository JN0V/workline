package agent

import (
	"os"
	"path/filepath"

	"github.com/JN0V/workline/internal/intent"
	"strings"
	"testing"
)

func TestFenceOutlastsBackticks(t *testing.T) {
	if f := fence("a ```` b ``` c"); f != "`````" {
		t.Errorf("fence = %q", f)
	}
	if f := fence("none"); f != "```" {
		t.Errorf("fence = %q", f)
	}
}

func TestRefusedAnswerCapsAndFencesADiff(t *testing.T) {
	run := t.TempDir()
	if err := writeRefused(run, "--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n"+strings.Repeat("+x\n", 40000)); err != nil {
		t.Fatal(err)
	}
	got := refusedAnswerMax(run, maxRefusedChars)
	if !strings.Contains(got, "```diff\n--- a/x\n+++ b/x") || len(got) > maxRefusedChars+200 {
		t.Errorf("diff not fenced or not capped (%d chars): %.200s", len(got), got)
	}
}

func writeRefused(run, diff string) error {
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		return err
	}
	return intent.Write(filepath.Join(run, "out", "refused-1.yaml"), []intent.Intention{{Kind: "patch", Value: diff}})
}
