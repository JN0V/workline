package engine

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A commit is a record of workline's when, against its one parent, it
// changes only the lines workline writes in the docs: a header's checked,
// judged, judged-in-parts or verified, in a front matter or a README's
// header, and derived blocks reading what their commands give. Anything
// else — a doc's text, another key, a doc added, a mode, a file no doc,
// nothing at all, a merge — is not.
func TestOwnRecord(t *testing.T) {
	repo, run, write := gitRepo(t)
	doc := "---\nsources: [a.go]\nchecked: 111\nverified: agent:x\n---\n# A\n\n<!-- workline:derive cases -->3<!-- workline:end -->\n\nOne hour.\n"
	readme := "<!-- workline\nsources: [a.go]\nchecked: 111\n-->\n# R\n\nHello.\n"
	plain := "---\nchecked: 111\n---\nNot a doc of the documentalist's.\n"
	write("doc.md", doc)
	write("README.md", readme)
	write("plain.md", plain)
	write("a.go", "package a\n")
	run("add", ".")
	run("commit", "-q", "-m", "docs: one hour")
	base := run("rev-parse", "HEAD")
	isDoc := func(p string) bool { return strings.HasSuffix(p, ".md") && p != "plain.md" }
	var asked []string
	asGiven := func(commit string, docs []string) bool { asked = docs; return true }
	notAsGiven := func(string, []string) bool { return false }
	for _, c := range []struct {
		name        string
		files       map[string]string
		exec        string
		regenerated func(string, []string) bool
		want        bool
		wantAsked   []string
	}{
		{"judged added, verified changed", map[string]string{"doc.md": strings.Replace(doc, "verified: agent:x\n", "judged: 222\nverified: agent:documentalist\n", 1)}, "", nil, true, nil},
		{"checked moved", map[string]string{"doc.md": strings.Replace(doc, "checked: 111", "checked: 333", 1)}, "", nil, true, nil},
		{"judged-in-parts in a README's header", map[string]string{"README.md": strings.Replace(readme, "checked: 111\n", "checked: 111\njudged-in-parts: 444\n", 1)}, "", nil, true, nil},
		{"a derived block as given", map[string]string{"doc.md": strings.Replace(doc, ">3<", ">4<", 1)}, "", asGiven, true, []string{"doc.md"}},
		{"a derived block not as given", map[string]string{"doc.md": strings.Replace(doc, ">3<", ">4<", 1)}, "", notAsGiven, false, nil},
		{"a derived block, no command to tell", map[string]string{"doc.md": strings.Replace(doc, ">3<", ">4<", 1)}, "", nil, false, nil},
		{"its text", map[string]string{"doc.md": strings.Replace(strings.Replace(doc, "checked: 111", "checked: 333", 1), "One hour.", "Two hours.", 1)}, "", asGiven, false, nil},
		{"its sources", map[string]string{"doc.md": strings.Replace(doc, "[a.go]", "[b.go]", 1)}, "", asGiven, false, nil},
		{"a doc added", map[string]string{"new.md": "---\nchecked: 111\n---\nNew.\n"}, "", asGiven, false, nil},
		{"made executable", nil, "doc.md", asGiven, false, nil},
		{"a file no doc", map[string]string{"plain.md": strings.Replace(plain, "checked: 111", "checked: 333", 1)}, "", asGiven, false, nil},
		{"nothing", nil, "", asGiven, false, nil},
	} {
		asked = nil
		run("checkout", "-q", "-B", "work", base)
		for p, s := range c.files {
			write(p, s)
		}
		if c.exec != "" {
			if err := os.Chmod(filepath.Join(repo, c.exec), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		run("add", ".")
		run("commit", "-q", "--allow-empty", "-m", "docs: "+c.name)
		head := run("rev-parse", "HEAD")
		if got := ownRecord(repo, head, isDoc, c.regenerated); got != c.want || !slices.Equal(asked, c.wantAsked) {
			t.Errorf("%s: %v, asked of %v; want %v, asked of %v", c.name, got, asked, c.want, c.wantAsked)
		}
	}
	// No docs known: no record. A merge, even of records alone: OwnMerge's,
	// never a record.
	run("checkout", "-q", "-B", "side", base)
	write("README.md", strings.Replace(readme, "checked: 111", "checked: 555", 1))
	run("commit", "-q", "-am", "docs: side")
	if ownRecord(repo, run("rev-parse", "HEAD"), nil, asGiven) {
		t.Error("with no docs known, a record")
	}
	run("checkout", "-q", "-B", "work", base)
	write("doc.md", strings.Replace(doc, "checked: 111", "checked: 666", 1))
	run("commit", "-q", "-am", "docs: work")
	run("merge", "-q", "--no-ff", "side", "-m", "Merge branch 'side' into work")
	if ownRecord(repo, run("rev-parse", "HEAD"), isDoc, asGiven) {
		t.Error("a merge taken for a record")
	}
}
