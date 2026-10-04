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
