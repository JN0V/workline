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
		// Python: a string standing as a statement is a docstring, a comment;
		// a string a value takes is code (docs/research/documentalist-genericity.md).
		{"a.py", "\"\"\"Tokens last one hour.\"\"\"\nTTL = 3600\n", "Tokens last one hour.", true, false, 1},
		{"a.py", "def ttl():\n    \"\"\"Tokens last\n    one hour.\n    \"\"\"\n    return 3600\n", "Tokens last one hour.", true, false, 2},
		{"a.py", "class T:\n    r'''Raw docstring.'''\n", "Raw docstring.", true, false, 2},
		{"a.py", "HELP = \"\"\"Tokens last one hour.\"\"\"\n", "Tokens last one hour.", true, true, 1},
		{"a.py", "HELP = (\n    \"\"\"Tokens last one hour.\"\"\"\n)\n", "Tokens last one hour.", true, true, 2},
		{"a.py", "x = f(a,\n      '''one hour''')\n", "one hour", true, true, 2},
		{"a.py", "s = \"# not a comment\"\n", "# not a comment", true, true, 1},
		{"a.pyi", "def f() -> int: ...  # one hour\n", "one hour", true, false, 1},
		// A script without an extension, by its first line.
		{"bin/run", "#!/usr/bin/env python3\n\"\"\"Runs it hourly.\"\"\"\nEVERY = 3600\n", "Runs it hourly.", true, false, 2},
		{"bin/run", "#!/bin/sh\n# runs hourly\nsleep 3600\n", "runs hourly", true, false, 2},
		{"bin/run", "#!/bin/sh\n# runs hourly\nsleep 3600\n", "sleep 3600", true, true, 3},
		// SQL, Lua, templates.
		{"a.sql", "-- one hour\nSELECT 3600; /* two */\n", "one hour", true, false, 1},
		{"a.sql", "-- one hour\nSELECT 3600; /* two */\n", "SELECT 3600;", true, true, 2},
		{"a.lua", "--[[ one\nhour ]]\nlocal ttl = 3600 -- two\n", "one hour", true, false, 1},
		{"a.hbs", "{{!-- one hour --}}{{! two }}<p>{{ttl}}</p>\n", "one hour", true, false, 1},
		{"a.hbs", "{{!-- one hour --}}{{! two }}<p>{{ttl}}</p>\n", "two", true, false, 1},
		{"a.hbs", "{{!-- one hour --}}{{! two }}<p>{{ttl}}</p>\n", "<p>{{ttl}}</p>", true, true, 1},
		{"page.html", "{# one hour #}\n{% comment %}two{% endcomment %}\n<p>3600</p>\n", "one hour", true, false, 1},
		{"page.html", "{# one hour #}\n{% comment %}two{% endcomment %}\n<p>3600</p>\n", "two", true, false, 2},
		{"a.vue", "<a href=\"x\">http://a.b</a>\n<script>\n// one hour\nconst ttl = 3600\n</script>\n", "one hour", true, false, 3},
		{"a.vue", "<a href=\"x\">http://a.b</a>\n", "http://a.b", true, true, 1},
	} {
		found, code, line := quoteIn(c.path, c.content, c.quote)
		if found != c.found || code != c.code || line != c.line {
			t.Errorf("quoteIn(%s, %q) = %v, %v, %d; want %v, %v, %d", c.path, c.quote, found, code, line, c.found, c.code, c.line)
		}
	}
}

// A type whose comments the engine does not know is said, never taken
// silently as code; a script names its type on its first line.
func TestStyleOf(t *testing.T) {
	for _, c := range []struct {
		path, content, kind string
		known               bool
	}{
		{"a.go", "", ".go", true},
		{"data.json", "", ".json", true},
		{"Makefile", "", "makefile", true},
		{"bin/run", "#!/usr/bin/env -S python3 -u\n", "#!python3", true},
		{"bin/run", "#!/usr/bin/node\n", "#!node", true},
		{"bin/run", "#!/usr/bin/wish\n", "#!wish", false},
		{"a.qqq", "x\n", ".qqq", false},
		{"VERSION", "1.2.3\n", "version", true},
		{"tools/release", "set -e\n", "release", false},
	} {
		st, kind := styleOf(c.path, c.content)
		if kind != c.kind || st.known != c.known {
			t.Errorf("styleOf(%s) = %q, known %v; want %q, %v", c.path, kind, st.known, c.kind, c.known)
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
		{[]string{"It's 1 000 \u201clines\u201d."}, []string{"It\u2019s 1\u202f000 \"lines\"."}, ""},
		{[]string{"It holds 1 000 lines."}, []string{"It holds 1000 lines."}, ""},
	} {
		got := strings.Join(wordsTakenOut(citedBlock{removed: c.removed, added: c.added}, "en"), " ")
		if got != c.want {
			t.Errorf("wordsTakenOut(%q → %q) = %q, want %q", c.removed, c.added, got, c.want)
		}
	}
}

// A line reworded taking out only its language's glue needs no claim: an
// article mended in French, "du" become "de l'"; a French "a" (has) or a
// negation does.
func TestGlueByLanguage(t *testing.T) {
	for _, c := range []struct {
		removed, added, lang string
		want             string
	}{
		{"Le connexion reste ouverte.", "La connexion reste ouverte.", "fr", ""},
		{"La fiche du employeur.", "La fiche de l\u2019employeur.", "fr", ""},
		{"Le jeton ne se renouvelle pas.", "Le jeton se renouvelle.", "fr", "ne pas"},
		{"Il a un jeton.", "Il un jeton.", "fr", "a"},
		{"Le connexion reste ouverte.", "La connexion reste ouverte.", "en", "Le"},
		{"It runs on a board.", "It runs a board.", "en", ""},
	} {
		b := citedBlock{from: 1, to: 1, removed: []string{c.removed}, added: []string{c.added}}
		if got := strings.Join(uncited(b, nil, c.lang), " "); got != c.want {
			t.Errorf("uncited(%q → %q, %s) = %q, want %q", c.removed, c.added, c.lang, got, c.want)
		}
	}
	for text, want := range map[string]string{
		"Le jeton d'accès dure une heure, et la session reste ouverte.": "fr",
		"Access tokens last one hour, and the session stays open.":     "en",
		"1.4.1": "en",
	} {
		if got := docLanguage(text); got != want {
			t.Errorf("docLanguage(%q) = %s, want %s", text, got, want)
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
// (ADR-0014, step 3), and the words said around it anew (step 4). Another
// number, a fact beside the count (a version, a name, a negation), or other
// words elsewhere, still need a claim, but for glue taken out of a line
// reworded.
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
		{"the rest said anew", "`Clock.h` is 524 lines long. Watch it.", "`Clock.h` is 569 lines: under the limit.", ""},
		{"watch, said anew", "`Clock.h` is currently ~630 lines (includes full `String` class stub). Watch the 800-line limit.", "`Clock.h` is currently ~569 lines (includes full `String` class stub). Over the 800-line hard limit.", ""},
		{"a version beside the count", "`Clock.h` is 524 lines since 1.4.1.", "`Clock.h` is 569 lines since 1.5.0.", "removal-uncited"},
		{"another number beside the count", "`Clock.h` is 524 lines, 12 tests.", "`Clock.h` is 569 lines, 9 tests.", "removal-uncited"},
		{"a negation beside the count", "`Clock.h` is 524 lines, not split.", "`Clock.h` is 569 lines, split.", "removal-uncited"},
		{"a tense beside the count", "`Clock.h` is 524 lines.", "`Clock.h` was 569 lines.", "removal-uncited"},
		{"a code name beside the count", "`Clock.h` is 524 lines, with `tick`.", "`Clock.h` is 569 lines, with `beat`.", "removal-uncited"},
		{"a proper name beside the count", "`Clock.h` is 524 lines, tried on Arduino.", "`Clock.h` is 569 lines, tried on boards.", "removal-uncited"},
		{"other words, the count left", "`Clock.h` is 524 lines long.", "`Clock.h` is 524 lines.", "removal-uncited"},
		{"glue only", "The clock keeps the time, per board.", "The clock keeps time, a board.", ""},
		{"a negation", "The clock is not synced.", "The clock is synced.", "removal-uncited"},
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
