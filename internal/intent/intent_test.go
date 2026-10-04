package intent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeKeepsFallbackPatchesOfOtherFiles(t *testing.T) {
	fallback := []Intention{
		{"patch", map[string]any{"file": "README.md", "content": "regenerated"}},
		{"patch", map[string]any{"file": "docs/a.md", "content": "regenerated"}},
		{"comment", "generated"},
	}
	agent := []Intention{
		{"patch", "--- a/docs/a.md\n+++ b/docs/a.md\n@@ -1 +1 @@\n-x\n+y\n"},
		{"comment", "the agent's"},
	}
	got := Merge(fallback, agent)
	var summary []string
	for _, i := range got {
		summary = append(summary, fmt.Sprint(i.Kind, PatchFiles(i.Value)))
	}
	want := "[patch[README.md] patch[docs/a.md] comment[]]"
	if fmt.Sprint(summary) != want {
		t.Fatalf("Merge = %v, want %s: the agent's patch replaces the fallback for its own file only, and its comment replaces the fallback comment", summary, want)
	}
	if got[1].Value == fallback[1].Value {
		t.Fatal("the patch kept for docs/a.md must be the agent's")
	}
}

func TestNormalizeDiff(t *testing.T) {
	in := "--- a/docs/a.md\n+++ b/docs/a.md\n@@ -1 +1 @@\n-x\n+y\n--- /dev/null\n+++ b/docs/new.md\n@@ -0,0 +1 @@\n+z\n"
	want := "diff --git a/docs/a.md b/docs/a.md\n--- a/docs/a.md\n+++ b/docs/a.md\n@@ -1 +1 @@\n-x\n+y\n" +
		"diff --git a/docs/new.md b/docs/new.md\nnew file mode 100644\n--- /dev/null\n+++ b/docs/new.md\n@@ -0,0 +1 @@\n+z\n"
	if got := NormalizeDiff(in); got != want {
		t.Fatalf("NormalizeDiff =\n%s\nwant\n%s", got, want)
	}
	if got := NormalizeDiff(want); got != want {
		t.Fatal("a diff that has its headers is left as it is")
	}
}

func TestMergeKeepsFallbackDiffsBeside(t *testing.T) {
	derived := Intention{"patch", "--- a/README.md\n+++ b/README.md\n@@ -19 +19 @@\n-61\n+63\n"}
	agent := []Intention{{"patch", "--- a/README.md\n+++ b/README.md\n@@ -3 +3 @@\n-checked: a\n+checked: b\n"}}
	got := Merge([]Intention{derived}, agent)
	if len(got) != 2 || got[0].Value != agent[0].Value || got[1].Value != derived.Value {
		t.Fatalf("Merge = %v: a fallback diff stays, after the agent's patch of the same file", got)
	}
}

func TestPatchFilesNamesADeletedFile(t *testing.T) {
	diff := "--- a/docs/cards/a.md\n+++ /dev/null\n@@ -1,2 +0,0 @@\n-# A\n-Short.\n--- a/docs/cards/b.md\n+++ b/docs/cards/b.md\n@@ -1 +1,2 @@\n # B\n+Short.\n"
	if got := strings.Join(PatchFiles(diff), ","); got != "docs/cards/a.md,docs/cards/b.md" {
		t.Fatalf("PatchFiles = %s, want the deleted file by its own path", got)
	}
}

func TestMergeKeepsStickyProposalsBesideTheAgents(t *testing.T) {
	tracking := Intention{Kind: "issue", Value: map[string]any{"title": "Docs due", "body": "- a.md", "sticky": true}}
	plain := Intention{Kind: "issue", Value: map[string]any{"title": "Something else", "body": "x"}}
	agent := []Intention{{Kind: "issue", Value: map[string]any{"title": "The code disagrees with a.md", "body": "y"}}}
	got := Merge([]Intention{tracking, plain}, agent)
	if len(got) != 2 || got[0].Value.(map[string]any)["title"] != "Docs due" {
		t.Fatalf("Merge = %v: the role's sticky issue stays beside the agent's; its plain one gives way", got)
	}
}

func TestNormalizeDiffMergesOverlappingHunks(t *testing.T) {
	// Lines 1-6 of a doc; hunks at 1, 3 (overlapping on line 3) and 6 (apart).
	in := "diff --git a/d.md b/d.md\n--- a/d.md\n+++ b/d.md\n" +
		"@@ -1,3 +1,3 @@\n-a\n+A\n b\n c\n" +
		"@@ -3,2 +3,2 @@\n c\n-d\n+D\n" +
		"@@ -6,1 +6,1 @@\n-f\n+F\n"
	want := "diff --git a/d.md b/d.md\n--- a/d.md\n+++ b/d.md\n" +
		"@@ -1,4 +1,4 @@\n-a\n+A\n b\n c\n-d\n+D\n" +
		"@@ -6,1 +6,1 @@\n-f\n+F\n"
	if got := NormalizeDiff(in); got != want {
		t.Fatalf("NormalizeDiff =\n%s\nwant\n%s", got, want)
	}
	// Overlapping on a line one hunk changes: not the same change twice, left to git.
	clash := "diff --git a/d.md b/d.md\n--- a/d.md\n+++ b/d.md\n@@ -1,2 +1,2 @@\n a\n-b\n+B\n@@ -2,1 +2,1 @@\n-b\n+X\n"
	if got := NormalizeDiff(clash); got != clash {
		t.Fatalf("hunks disagreeing were merged:\n%s", got)
	}
}

func TestWriteReadsBackATabbedQuote(t *testing.T) {
	in := []Intention{{Kind: "finding", Value: map[string]any{"quote": "\t\tif !ok {\n\t\t\treturn w"}}}
	file := filepath.Join(t.TempDir(), "intentions.yaml")
	if err := Write(file, in); err != nil {
		t.Fatal(err)
	}
	back, err := Read(file)
	if err != nil || len(back) != 1 || back[0].Value.(map[string]any)["quote"] != "\t\tif !ok {\n\t\t\treturn w" {
		t.Errorf("read back: %v %v", back, err)
	}
}
