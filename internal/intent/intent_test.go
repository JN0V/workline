package intent

import (
	"fmt"
	"testing"
)

func TestMergeKeepsFallbackPatchesOfOtherFiles(t *testing.T) {
	fallback := []Intention{
		{"patch", map[string]any{"file": "README.md", "content": "regenerated"}},
		{"patch", map[string]any{"file": "docs/a.md", "content": "regenerated"}},
		{"release", map[string]any{"version": "v1.0.0"}},
	}
	agent := []Intention{
		{"patch", "--- a/docs/a.md\n+++ b/docs/a.md\n@@ -1 +1 @@\n-x\n+y\n"},
		{"release", map[string]any{"version": "v1.0.0", "notes": "n"}},
	}
	got := Merge(fallback, agent)
	var summary []string
	for _, i := range got {
		summary = append(summary, fmt.Sprint(i.Kind, PatchFiles(i.Value)))
	}
	want := "[patch[README.md] patch[docs/a.md] release[]]"
	if fmt.Sprint(summary) != want {
		t.Fatalf("Merge = %v, want %s: the agent's patch replaces the fallback for its own file only, and its release replaces the fallback release", summary, want)
	}
	if got[1].Value == fallback[1].Value {
		t.Fatal("the patch kept for docs/a.md must be the agent's")
	}
}
