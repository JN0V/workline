package documentalist

import (
	"strings"
	"testing"
)

// At a push, the hygiene checks keep what they found in the push's docs:
// their own budgets, the folder holding them, their links. A setting missing
// is still said; passages written twice are not looked for (ADR-0050).
func TestHygieneOfKeepsThePushsDocs(t *testing.T) {
	long := "# A doc\n\n" + strings.Repeat("A line.\n", 5)
	passage := strings.Repeat("the same words written in two places ", 8) + "\n"
	tree := Tree{
		Docs: map[string]string{
			"docs/pushed.md": long + "[gone](nowhere.md)\n",
			"docs/other.md":  long + "[gone](nowhere.md)\n" + passage,
			"notes/far.md":   long + passage,
		},
		Files: map[string]bool{"docs/pushed.md": true, "docs/other.md": true, "notes/far.md": true},
	}
	b := Budgets{DocLines: 3, SectionWords: 400, FolderLines: 10, RootAgentFileLines: 200}
	b.CardWords.Min, b.CardWords.Max = 80, 400
	got := map[string]bool{}
	for _, p := range HygieneOf(tree, b, Duplicates{}, map[string]bool{"docs/pushed.md": true}) {
		got[p.Rule+" "+p.Where] = true
	}
	for _, want := range []string{"doc-too-long docs/pushed.md", "folder-too-long docs", "dead-link docs/pushed.md", "setting-missing "} {
		if !got[want] {
			t.Errorf("missing %q, got %v", want, got)
		}
	}
	for _, bad := range []string{"doc-too-long docs/other.md", "dead-link docs/other.md", "doc-too-long notes/far.md", "folder-too-long notes"} {
		if got[bad] {
			t.Errorf("%q is not the push's, got %v", bad, got)
		}
	}
	for _, p := range HygieneOf(tree, b, Duplicates{MinWords: 10, Similarity: 0.8}, map[string]bool{"docs/other.md": true}) {
		if p.Rule == "duplicate" || p.Rule == "setting-missing" {
			t.Errorf("%s at a push: %s", p.Rule, p.Message)
		}
	}
}
