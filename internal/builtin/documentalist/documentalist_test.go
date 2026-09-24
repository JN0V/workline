package documentalist

import (
	"sort"
	"strings"
	"testing"
)

func TestSection(t *testing.T) {
	doc := "---\nchecked: abc\n---\n# Auth\n\nIntro.\n\n## Token refresh\n\nOne hour.\n\n### Detail\n\nMore.\n\n## Sign out\n\nBye.\n"
	got, ok := Section(doc, "token-refresh")
	if !ok || got != "## Token refresh\n\nOne hour.\n\n### Detail\n\nMore." {
		t.Fatalf("Section = %q, %v", got, ok)
	}
	if _, ok := Section(doc, "missing"); ok {
		t.Fatal("a missing anchor must be reported")
	}
	a, _ := Section("---\nchecked: abc\n---\n# T\n\nSame.\n", "")
	b, _ := Section("---\nchecked: def\n---\n# T\n\nSame.\n", "")
	if a != b {
		t.Fatal("changing only the frontmatter is not a change")
	}
}

func TestSplitSource(t *testing.T) {
	for _, c := range []struct{ in, repo, path, anchor string }{
		{"src/a.go", "", "src/a.go", ""},
		{"docs/tech/auth.md#token-refresh", "", "docs/tech/auth.md", "token-refresh"},
		{"api:docs/x.md#part", "api", "docs/x.md", "part"},
	} {
		r, p, a := splitSource(c.in)
		if r != c.repo || p != c.path || a != c.anchor {
			t.Errorf("splitSource(%q) = %q %q %q", c.in, r, p, a)
		}
	}
}

func TestJudgeHunks(t *testing.T) {
	doc := "---\nchecked: abc\n---\n# T\n\nOne hour.\n<!-- workline:derive cmd=\"x\" -->\n3 roles\n<!-- workline:end -->\n"
	diff := "--- a/d.md\n+++ b/d.md\n@@ -5,2 +5,2 @@\n\n-One hour.\n+Two hours.\n"
	files, err := parseDiff(diff)
	if err != nil || len(files) != 1 || files[0].path != "d.md" {
		t.Fatalf("parseDiff = %v, %v", files, err)
	}
	if why := misquoted(doc, files[0]); why != "" {
		t.Fatalf("a blank context line whose space was trimmed is still a quote: %s", why)
	}
	if got := applyHunks(doc, files[0]); !strings.Contains(got, "\nTwo hours.\n") || lineCount(got) != lineCount(doc) {
		t.Fatalf("applyHunks = %q", got)
	}
	shifted, _ := parseDiff(strings.Replace(diff, "@@ -5,2 +5,2 @@", "@@ -4,2 +4,2 @@", 1))
	if misquoted(doc, shifted[0]) == "" {
		t.Fatal("a hunk citing the wrong lines must be refused, even if git apply would find them")
	}
	derived, _ := parseDiff("--- a/d.md\n+++ b/d.md\n@@ -7,3 +7,3 @@\n <!-- workline:derive cmd=\"x\" -->\n-3 roles\n+4 roles\n <!-- workline:end -->\n")
	if !touchesDerived(doc, derived[0]) || touchesDerived(doc, files[0]) {
		t.Fatal("only a change between derive markers touches a derived block")
	}
	if !checkedMatches("---\nsources: [a]\nchecked: 1a8e5a4\n---\n", map[string]string{"": "1a8e5a4238f3"}) ||
		checkedMatches("---\nsources: [a]\nchecked: 1a8\n---\n", map[string]string{"": "1a8e5a4238f3"}) {
		t.Fatal("checked must name the commit by at least 7 characters")
	}
}

func TestHygiene(t *testing.T) {
	para := "A session lasts as long as its token, and a token lasts one hour unless it is refreshed, after which the user signs in again.\n"
	tree := Tree{
		Docs: map[string]string{
			"docs/a.md": "# A\n\n## Part one\n\n" + para + "\nSee [b](b.md#part-one), [again](b.md#part-one-1), [c](c.md), `code [x](nowhere.md)`.\n\n```\n[y](nowhere.md)\n```\n",
			"docs/b.md": "# B\n\n## Part one\n\n" + strings.Replace(para, "signs in again", "must sign in again", 1) + "\n## Part one\n\nAgain.\n",
		},
		Files: map[string]bool{"docs/a.md": true, "docs/b.md": true},
	}
	var got []string
	for _, p := range Hygiene(tree, Budgets{DocLines: 100, SectionWords: 100, FolderLines: 100, RootAgentFileLines: 100,
		CardWords: struct {
			Min int `json:"min"`
			Max int `json:"max"`
		}{1, 100}}, Duplicates{MinWords: 10, Similarity: 0.7}) {
		got = append(got, p.Rule+" "+p.Where)
	}
	sort.Strings(got)
	want := "dead-link docs/a.md|duplicate docs/a.md"
	if strings.Join(got, "|") != want {
		t.Fatalf("Hygiene = %v, want %s (repeated headings get -1, links in code are not links)", got, want)
	}
}

func TestNamed(t *testing.T) {
	got := named("Call `auth.RefreshToken()` and `Revoke`, not `go test ./...` nor `id`.\n\n```\n`Hidden`\n```\n")
	if strings.Join(got, ",") != "RefreshToken,Revoke" {
		t.Fatalf("named = %v", got)
	}
}

func TestParseDocKeepsCommitsAsWritten(t *testing.T) {
	for _, c := range []struct{ fm, repo, want string }{
		{"checked: 11180e1", "", "11180e1"}, // a number, read as YAML
		{"checked: 1234567", "", "1234567"},
		{"checked: {api: 0012e45}", "api", "0012e45"},
	} {
		d, err := ParseDoc("d.md", []byte("---\nsources: [a.go]\n"+c.fm+"\n---\n"))
		if err != nil || d.Checked[c.repo] != c.want {
			t.Errorf("%s: checked = %v, %v", c.fm, d.Checked, err)
		}
	}
}

func TestHeaderComment(t *testing.T) {
	doc := "<!-- workline\nsources: [cmd/workline]\nchecked: 11180e1\n-->\n<h1>Title</h1>\n\nText.\n"
	d, err := ParseDoc("README.md", []byte(doc))
	if err != nil || d == nil || d.Checked[""] != "11180e1" || d.Sources[0] != "cmd/workline" {
		t.Fatalf("ParseDoc = %+v, %v", d, err)
	}
	a, _ := Section(doc, "")
	b, _ := Section(strings.Replace(doc, "11180e1", "abcdef0", 1), "")
	if a != b || strings.Contains(a, "sources") {
		t.Fatalf("the header is not part of what the doc says: %q", a)
	}
	if lines := scan(doc); lines[0].text != "<h1>Title</h1>" || lines[0].n != 5 {
		t.Fatalf("scan starts at %+v, want line 5", lines[0])
	}
	if d, _ := ParseDoc("x.md", []byte("<!-- a plain comment -->\n# T\n")); d != nil {
		t.Fatal("only a `<!-- workline` comment is a header")
	}
}

func TestDeriveSkipsExamplesInCode(t *testing.T) {
	doc := "Real: <!-- workline:derive n -->0<!-- workline:end -->.\n\n" +
		"An example: `<!-- workline:derive name -->` and `<!-- workline:end -->`.\n\n" +
		"```\n<!-- workline:derive other -->x<!-- workline:end -->\n```\n\n" +
		"Block:\n<!-- workline:derive n -->\nold\n<!-- workline:end -->\n"
	findings, fixed := Derive(t.TempDir(), map[string]string{"d.md": doc}, map[string]string{"n": "echo 3"})
	if len(findings) != 1 || findings[0].Rule != "derived-stale" {
		t.Fatalf("findings = %v: examples in code are not blocks", findings)
	}
	want := strings.Replace(strings.Replace(doc, "-->0<!--", "-->3<!--", 1), "-->\nold\n<!--", "-->\n3\n<!--", 1)
	if fixed["d.md"] != want {
		t.Fatalf("fixed =\n%s\nwant\n%s", fixed["d.md"], want)
	}
}

func TestEditedInPlace(t *testing.T) {
	gained := []string{"intentions: [commit-message, note]   # subset of the catalogue in role-run.md"}
	if !editedInPlace("intentions: [commit-message, note]   # subset of the catalogue below", gained) {
		t.Error("a reference updated in place is not a rewrite")
	}
	if editedInPlace("Signing out ends the session.", []string{"A session is ended by signing out."}) {
		t.Error("a sentence reworded is a rewrite")
	}
}
