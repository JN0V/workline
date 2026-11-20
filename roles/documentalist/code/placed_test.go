package documentalist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The reader task's patch of #547, on docs/weekly-sample.md as the agent
// read it (e5f16ced): its one context line holds two spaces too many, and
// the patch ends without a newline. Placed by the 6 lines it removes,
// found once in the page, the fix applies as placed: never withheld as a
// diff git would read otherwise (#592).
func TestApplyDocAsPlaced(t *testing.T) {
	page, err := os.ReadFile("testdata/issue-592/weekly-sample.md.txt")
	if err != nil {
		t.Fatal(err)
	}
	diff, err := os.ReadFile("testdata/issue-592/reader.diff")
	if err != nil {
		t.Fatal(err)
	}
	files, err := parseDiff(string(diff))
	if err != nil {
		t.Fatal(err)
	}
	var f fileDiff
	for _, fd := range files {
		if fd.path == "docs/weekly-sample.md" {
			f = fd
		}
	}
	now, rule, why := applyDoc(string(page), f, string(diff))
	if rule != "" {
		t.Fatalf("withheld (%s): %s", rule, why)
	}
	want := "  unearned nor put back.\n" +
		"- **What a product owner's act was done on**:\n" +
		"  - the issue's text it read, found by the digest its record keeps — the body now, else the version GitHub's edit history keeps; for a refine, the issue as it reads now beside it;\n" +
		"  - the code at the commit it read;\n" +
		"  - a text gone from the forge is not read, for you: GitLab keeps no edit history;\n" +
		"  - an act recorded before this record is read as its issue reads now, and named.\n" +
		"- **A squash**: a merge request's commits are read on its head, which\n"
	if !strings.Contains(now, want) {
		t.Errorf("the page as fixed lacks the 4 bullets in place:\n%s", now)
	}
	if strings.Count(now, "\n") != strings.Count(string(page), "\n")-1 {
		t.Errorf("the page went from %d lines to %d, want one fewer", strings.Count(string(page), "\n"), strings.Count(now, "\n"))
	}
}

// A hunk whose removed lines the doc holds twice, its context quoted
// wrong, is in doubt: no place is guessed, the fix is refused as today.
func TestApplyDocPlaceInDoubt(t *testing.T) {
	doc := "# Guide\n\nRun it.\nThen stop.\n\nRun it.\nThen stop.\n"
	diff := "--- a/docs/guide.md\n+++ b/docs/guide.md\n@@ -1,4 +1,3 @@\n  # Guide\n \n-Run it.\n-Then stop.\n+Run it, then stop.\n"
	files, err := parseDiff(diff)
	if err != nil {
		t.Fatal(err)
	}
	if now, rule, _ := applyDoc(doc, files[0], diff); rule != "misquoted" {
		t.Errorf("rule %q, want misquoted; the doc became:\n%s", rule, now)
	}
}

// A patch moving text, git cannot apply as written: its part changing a
// doc is written anew from its hunks as placed, its part creating a doc
// kept as written, and git applies the whole so. A part whose place is in
// doubt leaves the patch refused, with git's own reason (#592).
func TestPlacedPatch(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "# Guide\n\n## Sessions\n\nA session lasts one hour.\nIt is renewed on each request.\n\n## Again\n\nRun it.\nRun it.\n"
	if err := os.WriteFile(filepath.Join(repo, "docs", "guide.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	docs := map[string]string{"docs/guide.md": doc}
	moved := "--- a/docs/guide.md\n+++ b/docs/guide.md\n@@ -3,5 +3,3 @@\n  ## Sessions\n \n-A session lasts one hour.\n-It is renewed on each request.\n+See [sessions](sessions.md).\n" +
		"--- /dev/null\n+++ b/docs/sessions.md\n@@ -0,0 +1,3 @@\n+# Sessions\n+\n+A session lasts one hour.\n"
	got, err := placedPatch(repo, moved, docs)
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	if !strings.Contains(got, "new file mode 100644\n--- /dev/null\n+++ b/docs/sessions.md\n@@ -0,0 +1,3 @@\n+# Sessions\n") {
		t.Errorf("the new doc's part not kept as written:\n%s", got)
	}
	if !strings.Contains(got, " ## Sessions\n \n-A session lasts one hour.\n") {
		t.Errorf("the doc's part not written anew, quoting the doc:\n%s", got)
	}
	if _, err := gitIn(repo, got, "apply", "--check", "-"); err != nil {
		t.Errorf("git cannot apply it: %v", err)
	}
	asWritten := "--- a/docs/guide.md\n+++ b/docs/guide.md\n@@ -1,1 +1,1 @@\n-# Guide\n+# The guide\n"
	if same, err := placedPatch(repo, asWritten, docs); err != nil || same != asWritten {
		t.Errorf("a patch git applies as written: %q, %v; want it as written", same, err)
	}
	doubt := "--- a/docs/guide.md\n+++ b/docs/guide.md\n@@ -9,3 +9,2 @@\n  \n-Run it.\n+Run it once.\n"
	if _, err := placedPatch(repo, doubt, docs); err == nil || !strings.Contains(err.Error(), "patch failed") {
		t.Errorf("a hunk whose place is in doubt: %v, want git's refusal", err)
	}
}
