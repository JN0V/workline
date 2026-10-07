package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// An item imported from a file is a person's words: only a role's finding
// is marked as that role's, for the product owner to read as its draft.
func TestOpenedByOnlyOnFindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(`{"issues": [], "merge-requests": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	o, err := NewOpenings(f, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []Opening{
		{Role: "product-owner", Key: "import=ROADMAP.md:abc", Title: "Imported", Body: "A person's item.", From: " from ROADMAP.md", Commit: "0000000"},
		{Role: "reviewer", Key: CodeKey("a.go", "x := 1"), Title: "Found", Body: "A finding.", Commit: "0000000", Triage: true},
	} {
		if _, _, err := o.Open(op); err != nil {
			t.Fatal(err)
		}
	}
	all, err := f.AllIssues()
	if err != nil || len(all) != 2 {
		t.Fatalf("issues: %v, %v", all, err)
	}
	if by := OpenedBy(all[0].Body); by != "" || !strings.Contains(all[0].Body, "Opened from ROADMAP.md by the product-owner role.") {
		t.Errorf("an import: opened by %q, body %q", by, all[0].Body)
	}
	if by := OpenedBy(all[1].Body); by != "reviewer" {
		t.Errorf("a finding: opened by %q, want reviewer", by)
	}
}

// A run stopped after opening an issue, before its label and state, is
// completed when resumed: the issue found open gets what it lacks.
func TestOpenResumedCompletesTheIssue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	key := CodeKey("a.go", "x := 1")
	state := `{"issues": [{"id": 1, "title": "Found", "labels": [], "comments": [], "body": "A finding.\n\nOpened by the reviewer role.\n\n<!-- workline:opened-by=reviewer -->\n\n<!-- workline:` + key + ` -->"}], "merge-requests": []}`
	if err := os.WriteFile(path, []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	o, _ := NewOpenings(f, 3)
	op := Opening{Role: "reviewer", Key: key, Title: "Found", Body: "A finding.", Sources: []string{"a.go"}, Commit: "0000000", Triage: true}
	for i := 0; i < 2; i++ { // the second run finds it whole, and writes nothing
		if got, id, err := o.Open(op); err != nil || got != StillOpen || id != 1 {
			t.Fatalf("run %d: %s #%d, %v", i, got, id, err)
		}
	}
	is, _ := f.Issue(1)
	comments, _ := f.Comments(forge.Target{Kind: "issue", ID: 1})
	if len(is.Labels) != 1 || is.Labels[0] != LabelTriage || len(comments) != 1 || !strings.Contains(comments[0], "sources: [a.go]") {
		t.Errorf("labels %v, comments %q", is.Labels, comments)
	}
}

// A person's issue: no key on a line of its own, as Openings writes it —
// one quoted or inside a sentence is their text —, and no bot author.
func TestByPerson(t *testing.T) {
	for _, c := range []struct {
		is   forge.Issue
		want bool
	}{
		{forge.Issue{Body: "A tester role.", Author: "ann"}, true},
		{forge.Issue{Body: "Text.\n\n<!-- workline:import=BACKLOG.md:0a1b2c3d4e5f -->", Author: "ann"}, false},
		{forge.Issue{Body: "Found.\n\n<!-- workline:opened-by=reviewer -->\n\n<!-- workline:issue=a.go#12345678 -->"}, false},
		{forge.Issue{Body: "As #12 said:\n> <!-- workline:import=BACKLOG.md:0a1b2c3d4e5f -->", Author: "ann"}, true},
		{forge.Issue{Body: "It writes `<!-- workline:issue=a.go#1 -->` at the end.", Author: "ann"}, true},
		{forge.Issue{Body: "Bump it.", Author: "renovate[bot]"}, false},
		{forge.Issue{Body: "Bump it.", Author: "project_12_bot_ab12"}, false},
	} {
		if got := ByPerson(c.is); got != c.want {
			t.Errorf("ByPerson(%q by %q) = %v, want %v", c.is.Body, c.is.Author, got, c.want)
		}
	}
}
