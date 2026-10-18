package documentalist

import (
	"strings"
	"testing"
)

// A patch changing only the lines recording when a doc was read changes
// nothing it says; `checked`, which vouches, is not one of them.
func TestRecordOnly(t *testing.T) {
	was := "---\nsources: [a.go]\nchecked: 1111111\n---\n# A\n"
	for _, c := range []struct {
		now  string
		want bool
	}{
		{"---\nsources: [a.go]\nchecked: 1111111\njudged: 2222222\n---\n# A\n", true},
		{"---\nsources: [a.go]\nchecked: 1111111\njudged: 2222222\nverified: agent:documentalist\n---\n# A\n", true},
		{"---\nsources: [a.go]\nchecked: 3333333\n---\n# A\n", false},
		{"---\nsources: [a.go]\nchecked: 1111111\njudged: 2222222\n---\n# B\n", false},
		{was, false},
	} {
		if got := recordOnly(was, c.now); got != c.want {
			t.Errorf("recordOnly(%q) = %v, want %v", c.now, got, c.want)
		}
	}
	readme := "<!-- workline\nsources: [a.go]\nchecked: 1111111\n-->\n# R\n"
	if !recordOnly(readme, strings.Replace(readme, "-->", "judged: 2222222\n-->", 1)) {
		t.Error("a README's header comment: recording `judged` is a record only")
	}
}

func TestWithHeaderValue(t *testing.T) {
	doc := "---\nsources: [a.go]\nchecked: 1111111\n---\n# A\n"
	got, ok := withHeaderValue(doc, "judged", "2222222")
	if !ok || got != "---\nsources: [a.go]\nchecked: 1111111\njudged: 2222222\n---\n# A\n" {
		t.Errorf("added: %q, %v", got, ok)
	}
	if again, _ := withHeaderValue(got, "judged", "3333333"); again != strings.Replace(got, "2222222", "3333333", 1) {
		t.Errorf("replaced: %q", again)
	}
	if _, ok := withHeaderValue("# no header\n", "judged", "2222222"); ok {
		t.Error("a doc with no header has nowhere to record it")
	}
}

// A record that does not read is none, never a doc held by guess.
func TestParseKept(t *testing.T) {
	comment := "x\n" +
		`<!-- workline:doc-judged {"doc":"docs/a.md","key":"judged","at":"abc","ids":{"docs/a.md":"1"}} -->` + "\n" +
		`<!-- workline:doc-judged {"doc":"docs/b.md","key":"checked","at":"abc","ids":{"docs/b.md":"1"}} -->` + "\n" +
		`<!-- workline:doc-judged {"doc":"docs/c.md","key":"judged","at":"abc"} -->` + "\n" +
		`<!-- workline:doc-judged {not json} -->` + "\n" +
		"<!-- workline:doc-judged " + hiddenJSON(keptRecord{Doc: "docs/my-doc.md", Key: "judged-in-parts", At: "def", IDs: map[string]string{"docs/my-doc.md": "2"}}) + " -->\n"
	got := parseKept(comment)
	if len(got) != 2 || got["docs/a.md"].At != "abc" || got["docs/my-doc.md"].Key != "judged-in-parts" {
		t.Errorf("parseKept = %v", got)
	}
	if strings.Contains(hiddenJSON(keptRecord{Doc: "a-->b"}), "-->") {
		t.Error("a record must not end the comment hiding it")
	}
}

func TestDocPaths(t *testing.T) {
	got, ok := docPaths(&Doc{Path: "docs/a.md", Sources: []string{"src/a/", "./src/b.go#part", "src/a", "."}})
	if !ok || strings.Join(got, ",") != "docs/a.md,src/a,src/b.go," {
		t.Errorf("docPaths = %q, %v", got, ok)
	}
	if _, ok := docPaths(&Doc{Path: "docs/a.md", Sources: []string{"api:docs/x.md"}}); ok {
		t.Error("a source in another repository: its state is not this one's")
	}
}

func TestPersonsTitle(t *testing.T) {
	for _, c := range []struct {
		fixed, recorded []string
		want            string
	}{
		{[]string{"docs/a.md"}, nil, "docs: bring docs/a.md in line with the code"},
		{[]string{"docs/a.md", "docs/b.md"}, []string{"docs/c.md"}, "docs: bring docs/a.md and 1 more in line with the code"},
		{nil, []string{"docs/c.md"}, "docs: record when docs/c.md was judged"},
		{[]string{"docs/" + strings.Repeat("x", 60) + ".md"}, nil, "docs: bring the docs in line with the code"},
		{nil, nil, ""},
	} {
		if got := personsTitle(c.fixed, c.recorded); got != c.want {
			t.Errorf("personsTitle(%v, %v) = %q, want %q", c.fixed, c.recorded, got, c.want)
		}
	}
}
