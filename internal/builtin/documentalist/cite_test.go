package documentalist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A quote is evidence when some of it lies outside the file's comments,
// read by the file's type (ADR-0014, step 2).
func TestQuoteIn(t *testing.T) {
	for _, c := range []struct {
		path, content, quote string
		found, code          bool
		line                 int
	}{
		{"a.go", "package a\n\n// Tokens last one hour.\nconst TTL = 7200\n", "Tokens last one hour.", true, false, 3},
		{"a.go", "package a\n\n// Tokens last one hour.\nconst TTL = 7200\n", "const TTL = 7200", true, true, 4},
		{"a.go", "x := 1 // the default is on\n", "the default is on", true, false, 1},
		{"a.go", "x := 1 // the default is on\n", "x := 1", true, true, 1},
		{"a.go", "/* the default\n   is on */\nx := 1\n", "the default is on", true, false, 1},
		{"a.go", "u := \"https://example.com\" // a link\n", "https://example.com", true, true, 1},
		{"a.go", "s := `a raw\n// not a comment`\n", "// not a comment", true, true, 2},
		{"a.go", "const TTL = 7200\n", "const TTL = 3600", false, false, 0},
		{"a.cpp", "#define VERSION \"1.4.1\"\n// returns 1.4.0\n", "#define VERSION \"1.4.1\"", true, true, 1},
		{"a.cpp", "#define VERSION \"1.4.1\"\n// returns 1.4.0\n", "returns 1.4.0", true, false, 2},
		{"ci.yml", "# Not tried live yet.\nimage: ghcr.io/x:v1 # pinned\n", "Not tried live yet.", true, false, 1},
		{"ci.yml", "# Not tried live yet.\nimage: ghcr.io/x:v1 # pinned\n", "image: ghcr.io/x:v1", true, true, 2},
		{"ci.yml", "url: \"http://a#b\"\n", "http://a#b", true, true, 1},
		{"run.sh", "echo $# # count\n", "echo $#", true, true, 1},
		{"Makefile", "# build it\nall:\n", "build it", true, false, 1},
		{"README.md", "<!-- old -->\nThe new way.\n", "old", true, false, 1},
		{"README.md", "<!-- old -->\nThe new way.\n", "The new way.", true, true, 2},
		{"library.json", "{\"version\": \"2.12.0\" // no comments in JSON\n}", "no comments in JSON", true, true, 1},
		{"a.go", "x := 'a' // it's a rune\n", "it's a rune", true, false, 1},
	} {
		found, code, line := quoteIn(c.path, c.content, c.quote)
		if found != c.found || code != c.code || line != c.line {
			t.Errorf("quoteIn(%s, %q) = %v, %v, %d; want %v, %v, %d", c.path, c.quote, found, code, line, c.found, c.code, c.line)
		}
	}
}

// Words a fix takes out: replaced, dropped; never rewrapped or only added.
func TestWordsTakenOut(t *testing.T) {
	for _, c := range []struct {
		removed, added []string
		want           string
	}{
		{[]string{"Access tokens last one hour."}, []string{"Access tokens last two hours."}, "one hour"},
		{[]string{"| GitHub (comments, a comment edited in place, and labels tried live) |"}, []string{"| GitHub (comments and labels tried live) |"}, "a comment edited in place"},
		{[]string{"Version 2.0.0"}, []string{"Version 2.12.0"}, "2.0.0"},
		{[]string{"In a repository, `workline init` has the documentalist run"}, []string{"In a repository, `workline init` has the committer and the documentalist run"}, ""},
		{[]string{"one two", "three"}, []string{"one", "two three"}, ""},
		{nil, []string{"A new line."}, ""},
	} {
		got := strings.Join(wordsTakenOut(citedBlock{removed: c.removed, added: c.added}), " ")
		if got != c.want {
			t.Errorf("wordsTakenOut(%q → %q) = %q, want %q", c.removed, c.added, got, c.want)
		}
	}
}

// The header is left out of what a fix changes; the body's lines keep the
// doc's numbers.
func TestBodyBlocks(t *testing.T) {
	old := "---\nsources: [a.go]\nchecked: abc1234\n---\n# T\n\nOne hour.\n\nKept.\n"
	diff := "--- a/d.md\n+++ b/d.md\n@@ -2,3 +2,3 @@\n sources: [a.go]\n-checked: abc1234\n+checked: def5678\n ---\n@@ -6,4 +6,4 @@\n \n-One hour.\n+Two hours.\n \n Kept.\n"
	files, err := parseDiff(diff)
	if err != nil {
		t.Fatal(err)
	}
	blocks := bodyBlocks(old, files[0])
	if len(blocks) != 1 || blocks[0].from != 7 || blocks[0].to != 7 || blocks[0].removed[0] != "One hour." {
		t.Fatalf("bodyBlocks = %+v", blocks)
	}
}

// A line count the engine reports off is brought to the engine's number with
// no claim, in each shape a doc gives it; the engine's count is the evidence
// (ADR-0014, step 3). Another number, or other words, still need a claim.
func TestCountFixNeedsNoClaim(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "Clock.h"), []byte(strings.Repeat("x\n", 569)), 0o644); err != nil {
		t.Fatal(err)
	}
	cc := &citeContext{repo: dir, files: map[string]bool{"src/Clock.h": true}, shown: map[string]string{}}
	for _, c := range []struct {
		name, old, new, rule string
	}{
		{"prose", "`Clock.h` is 524 lines long.", "`Clock.h` is 569 lines long.", ""},
		{"approx sign", "`Clock.h` is currently ~283 lines.", "`Clock.h` is currently 569 lines.", ""},
		{"approx word dropped", "`Clock.h` is approximately 283 lines.", "`Clock.h` is 569 lines.", ""},
		{"approx word kept", "`Clock.h` is about 283 lines.", "`Clock.h` is about 569 lines.", ""},
		{"listing", "    Clock.h           (524 lines)   # the clock", "    Clock.h           (569 lines)   # the clock", ""},
		{"thousands", "`Clock.h` is 1,008 lines.", "`Clock.h` is 569 lines.", ""},
		{"another number", "`Clock.h` is 524 lines long.", "`Clock.h` is 600 lines long.", "removal-uncited"},
		{"other words too", "`Clock.h` is 524 lines long.", "`Clock.h` is 569 lines.", "removal-uncited"},
	} {
		old := "---\nsources: [src/Clock.h]\nchecked: abc1234\n---\n# Clock\n\n" + c.old + "\n"
		diff := "--- a/d.md\n+++ b/d.md\n@@ -7 +7 @@\n-" + c.old + "\n+" + c.new + "\n"
		files, err := parseDiff(diff)
		if err != nil {
			t.Fatal(err)
		}
		f := placed(old, files[0])
		rule, msg, _ := cc.judgeCitations("d.md", old, f, nil, 0, true)
		if rule != c.rule {
			t.Errorf("%s: rule %q (%s), want %q", c.name, rule, msg, c.rule)
		}
	}
	// A table's cell, under its column of line counts.
	old := "---\nsources: [src/Clock.h]\nchecked: abc1234\n---\n# Clock\n\n| File | Lines |\n|---|---|\n| `Clock.h` | 524 |\n"
	files, err := parseDiff("--- a/d.md\n+++ b/d.md\n@@ -9 +9 @@\n-| `Clock.h` | 524 |\n+| `Clock.h` | 569 |\n")
	if err != nil {
		t.Fatal(err)
	}
	if rule, msg, _ := cc.judgeCitations("d.md", old, placed(old, files[0]), nil, 0, true); rule != "" {
		t.Errorf("table cell: rule %q (%s), want none", rule, msg)
	}
}
