package documentalist

import (
	"sort"
	"strings"
	"testing"
)

func archivedProblems(docs map[string]string) []string {
	files := map[string]bool{}
	for p := range docs {
		files[p] = true
	}
	b := Budgets{DocLines: 5, SectionWords: 1000, FolderLines: 1000, RootAgentFileLines: 100}
	b.CardWords.Min, b.CardWords.Max = 1, 1000
	var got []string
	for _, p := range Hygiene(Tree{Docs: docs, Files: files}, b, Duplicates{MinWords: 10, Similarity: 0.7}) {
		got = append(got, p.Rule+" "+p.Where)
	}
	sort.Strings(got)
	return got
}

func TestHygieneSkipsArchivedBudgetsAndDuplicates(t *testing.T) {
	para := "A session lasts as long as its token, and a token lasts one hour unless it is refreshed, after which the user signs in again.\n"
	long := strings.Repeat("line\n", 10)
	docs := map[string]string{
		"docs/old.md": "---\narchived: 2026-10-04\n---\n# Old\n\n" + long + "\n" + para,
		"docs/new.md": "# New\n\n" + para,
	}
	if got := archivedProblems(docs); len(got) != 0 {
		t.Fatalf("archived doc reported: %v", got)
	}
	docs["docs/old.md"] = strings.Replace(docs["docs/old.md"], "archived: 2026-10-04", "archived: off", 1)
	got := strings.Join(archivedProblems(docs), "|")
	if !strings.Contains(got, "doc-too-long docs/old.md") || !strings.Contains(got, "duplicate") {
		t.Fatalf("a doc not archived must be gardened, got %s", got)
	}
}

// Condensing and splitting pick their doc among the budget problems: an
// archived doc, reported none, is never condensed; the same doc unmarked is.
func TestArchivedDocIsNeverCondensed(t *testing.T) {
	long := strings.Repeat("line\n", 10)
	docs := map[string]string{
		"docs/old.md": "---\narchived: 2026-10-04\n---\n# Old\n\n" + long,
	}
	files := map[string]bool{"docs/old.md": true}
	b := Budgets{DocLines: 5, SectionWords: 1000, FolderLines: 1000, RootAgentFileLines: 100}
	b.CardWords.Min, b.CardWords.Max = 1, 1000
	d := Duplicates{MinWords: 10, Similarity: 0.7}
	cands := func() []*condenseTask {
		return condenseCandidates(Hygiene(Tree{Docs: docs, Files: files}, b, d))
	}
	if got := cands(); len(got) != 0 {
		t.Fatalf("archived doc offered to condense: %v", got[0].Doc)
	}
	docs["docs/old.md"] = strings.Replace(docs["docs/old.md"], "archived: 2026-10-04", "archived: off", 1)
	if got := cands(); len(got) != 1 || got[0].Doc != "docs/old.md" {
		t.Fatalf("a doc not archived must be condensed, got %v", got)
	}
}

func TestHygieneChecksLinksOfArchivedAndToIt(t *testing.T) {
	docs := map[string]string{
		"docs/old.md": "---\narchived: 2026-10-04\n---\n# Old\n\nSee [gone](gone.md).\n",
		"docs/new.md": "# New\n\nSee [old](old.md).\n",
	}
	got := strings.Join(archivedProblems(docs), "|")
	if got != "dead-link docs/old.md" {
		t.Fatalf("Hygiene = %s, want only the dead link of the archived doc", got)
	}
}

func TestArchivedRootAgentFileNotOverBudget(t *testing.T) {
	long := strings.Repeat("Read a doc.\n", 120)
	docs := map[string]string{"AGENTS.md": "---\narchived: true\n---\n# For agents\n\n" + long}
	for _, p := range archivedProblems(docs) {
		if strings.HasPrefix(p, "agent-file-too-long") {
			t.Fatalf("archived agent file reported: %s", p)
		}
	}
	docs["AGENTS.md"] = "# For agents\n\n" + long
	if got := strings.Join(archivedProblems(docs), "|"); !strings.Contains(got, "agent-file-too-long AGENTS.md") {
		t.Fatalf("an agent file not archived must be reported, got %s", got)
	}
}
