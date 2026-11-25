package documentalist

import (
	"os"
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
