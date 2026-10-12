package engine

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A gardening branch a person committed to, main moved under it: named
// when git finds a conflict; when git cannot tell, said as such
// (`conflict-unread`), never as a conflict it may not be; left without a
// word when it merges.
func TestFollowAPersonsBranch(t *testing.T) {
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	write := func(p, s string) {
		if err := os.WriteFile(filepath.Join(repo, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("doc.md", "---\nchecked: 111\n---\nOne hour.\n")
	run("add", "doc.md")
	run("commit", "-q", "-m", "docs: one hour")
	run("checkout", "-q", "-b", "workline/documentalist/suspect")
	write("doc.md", "---\nchecked: 111\n---\nTwo hours.\n")
	run("commit", "-q", "-am", "docs: two hours", "-m", OwnTrailer+": documentalist")
	write("note.md", "A person's note.\n")
	run("add", "note.md")
	run("commit", "-q", "-m", "docs: a person's note")
	run("checkout", "-q", "main")
	write("other.md", "Elsewhere.\n")
	run("add", "other.md")
	run("commit", "-q", "-m", "docs: elsewhere")

	branch := "workline/documentalist/suspect"
	if f, err := follow(repo, "main", branch, "documentalist", true, true); err != nil || f.Rule != "" {
		t.Errorf("merging still: %+v, %v; want nothing said", f, err)
	}
	was := conflicts
	defer func() { conflicts = was }()
	conflicts = func(string, string, string) (bool, error) { return false, errors.New("git merge-tree: unknown option") }
	if f, err := follow(repo, "main", branch, "documentalist", true, true); err != nil || f.Rule != "conflict-unread" || f.Level != "warn" {
		t.Errorf("git failing: %+v, %v; want conflict-unread, a warning", f, err)
	}
	conflicts = was
	write("doc.md", "---\nchecked: 111\n---\nNinety minutes.\n")
	run("commit", "-q", "-am", "docs: ninety minutes")
	if f, err := follow(repo, "main", branch, "documentalist", true, true); err != nil || f.Rule != "not-rebuilt" {
		t.Errorf("in conflict: %+v, %v; want not-rebuilt", f, err)
	}
}

// A doc's header lines recording when it was read are left out, in a
// front matter or a README's header; the same words in its body, or in a
// doc with no header, are kept: they say what the doc says.
func TestUnrecorded(t *testing.T) {
	cases := []struct{ doc, want string }{
		{"---\nsources: [a.go]\nchecked: 111\njudged: 222\njudged-in-parts: 333\nverified: agent:x\n---\n# T\n\njudged: kept\n",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\njudged: kept\n"},
		{"<!-- workline\nsources: [a.go]\njudged: 222\n-->\n# T\n", "<!-- workline\nsources: [a.go]\n-->\n# T\n"},
		{"# T\n\njudged: 222\n", "# T\n\njudged: 222\n"},
		{"---\njudged: 222\nno end\n", "---\njudged: 222\nno end\n"},
	}
	for _, c := range cases {
		if got := unrecorded(c.doc); got != c.want {
			t.Errorf("unrecorded(%q) = %q, want %q", c.doc, got, c.want)
		}
	}
}

func TestMergeDocFrontMatterByKey(t *testing.T) {
	old := "---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n"
	cases := []struct {
		name, ours, theirs, want string
		ok                       bool
	}{
		{"base moved checked, the branch added judged beside it",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n",
			"---\nsources: [a.go]\nchecked: 111\njudged: 333\n---\n# T\n\nTwo hours.\n",
			"---\nsources: [a.go]\nchecked: 222\njudged: 333\n---\n# T\n\nTwo hours.\n", true},
		{"both moved checked: base's, the later judgement",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n",
			"---\nsources: [a.go]\nchecked: 333\n---\n# T\n\nOne hour.\n",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n", true},
		{"the branch added a key first",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n",
			"---\ntype: reference\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n",
			"---\ntype: reference\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n", true},
		{"both changed the same sentence",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nNinety minutes.\n",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nTwo hours.\n",
			"", false},
	}
	for _, c := range cases {
		got, ok := mergeDoc(t.TempDir(), old, c.ours, c.theirs)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: got %v %q, want %v %q", c.name, ok, got, c.ok, c.want)
		}
	}
}
