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

// A finding that says where it was found ends with "Found by the … role
// …", not "Opened by": a run stopped before its label and state is
// completed all the same when resumed, the same line or one said at
// another commit; its description's digest is kept.
func TestOpenResumedCompletesAFoundIssue(t *testing.T) {
	key := CodeKey("a.go", "x := 1")
	for _, found := range []string{"while reviewing abc1234, outside that change", "while reviewing def5678, outside that change"} {
		path := filepath.Join(t.TempDir(), "forge.json")
		state := `{"issues": [{"id": 1, "title": "Found", "labels": [], "comments": [], "body": "Rows go missing.\n\nFound by the reviewer role while reviewing abc1234, outside that change.\n\n<!-- workline:opened-by=reviewer -->\n\n<!-- workline:` + key + ` -->"}], "merge-requests": []}`
		if err := os.WriteFile(path, []byte(state), 0o644); err != nil {
			t.Fatal(err)
		}
		f := &forge.Fake{Path: path}
		o, _ := NewOpenings(f, 3)
		op := Opening{Role: "reviewer", Key: key, Title: "Found", Body: "Rows go missing.", Sources: []string{"a.go"}, Commit: "0000000", Triage: true, Found: found}
		if got, id, err := o.Open(op); err != nil || got != StillOpen || id != 1 {
			t.Fatalf("%s: %s #%d, %v", found, got, id, err)
		}
		is, _ := f.Issue(1)
		comments, _ := f.Comments(forge.Target{Kind: "issue", ID: 1})
		if len(is.Labels) != 1 || is.Labels[0] != LabelTriage || len(comments) != 1 ||
			!strings.Contains(comments[0], "sources: [a.go]") || !strings.Contains(comments[0], "Description: "+BodyDigest("Rows go missing.")) {
			t.Errorf("%s: labels %v, comments %q", found, is.Labels, comments)
		}
	}
}

// A new finding that says where it was found ends with that line.
func TestOpenSaysWhereItWasFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(`{"issues": [], "merge-requests": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	o, _ := NewOpenings(f, 3)
	op := Opening{Role: "reviewer", Key: CodeKey("a.go", "x := 1"), Title: "Found", Body: "Rows go missing.", Commit: "0000000", Triage: true, Found: "while reviewing abc1234, outside that change."}
	if got, _, err := o.Open(op); err != nil || got != Opened {
		t.Fatalf("%s, %v", got, err)
	}
	is, _ := f.Issue(1)
	if !strings.Contains(is.Body, "Rows go missing.\n\nFound by the reviewer role while reviewing abc1234, outside that change.\n\n<!-- workline:opened-by=reviewer -->") || strings.Contains(is.Body, "Opened by") {
		t.Errorf("body %q", is.Body)
	}
}
