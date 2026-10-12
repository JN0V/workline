package engine

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		{"README's header, merged as a front matter",
			"<!-- workline\nsources: [a.go]\nchecked: 222\n-->\n# T\n\nOne hour.\n",
			"<!-- workline\nsources: [a.go]\nchecked: 111\njudged: 333\n-->\n# T\n\nTwo hours.\n",
			"<!-- workline\nsources: [a.go]\nchecked: 222\njudged: 333\n-->\n# T\n\nTwo hours.\n", true},
	}
	for _, c := range cases {
		from := old
		if strings.HasPrefix(c.ours, "<!--") {
			from = "<!-- workline\nsources: [a.go]\nchecked: 111\n-->\n# T\n\nOne hour.\n"
		}
		got, ok := mergeDoc(t.TempDir(), from, c.ours, c.theirs)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: got %v %q, want %v %q", c.name, ok, got, c.ok, c.want)
		}
	}
}

// On a person's branch (ADR-0041), a doc is settled only where every
// conflict is in a line workline writes: a header key of its own both
// changed takes base's, another is a clash; a derived block is base's; the
// rest is git's own merge, a conflict there the person's.
func TestSettleDoc(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %s", out)
	}
	old := "---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n"
	cases := []struct {
		name, branch, base, want string
		ok                       bool
	}{
		{"checked both moved: base's; the branch's text kept",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nTwo hours.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n",
			"---\nsources: [a.go]\nchecked: 333\nverified: agent:x\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n",
			"---\nsources: [a.go]\nchecked: 333\nverified: agent:x\n---\n# T\n\nTwo hours.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n", true},
		{"a derived block both changed: base's",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->4<!-- workline:end --> cases.\n",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->5<!-- workline:end --> cases.\n",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->5<!-- workline:end --> cases.\n", true},
		{"sources both changed: the person's",
			"---\nsources: [a.go, b.go]\nchecked: 222\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n",
			"---\nsources: [a.go, c.go]\nchecked: 333\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n",
			"", false},
		{"a sentence both changed: the person's",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nTwo hours.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n",
			"---\nsources: [a.go]\nchecked: 333\n---\n# T\n\nNinety minutes.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n",
			"", false},
		{"the words beside a derived block both changed: the person's",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->4<!-- workline:end --> cases pass.\n",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n\n<!-- workline:derive n -->5<!-- workline:end --> cases run.\n",
			"", false},
	}
	for _, c := range cases {
		got, ok, err := settleDoc(repo, "docs/t.md", old, c.branch, c.base)
		if err != nil || ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: got %v %q (%v), want %v %q", c.name, ok, got, err, c.ok, c.want)
		}
	}
	// A derive marker with no end of its own is no block: the person's
	// text after it, up to another block's end, is never taken for base's.
	unclosed := "# T\n\n<!-- workline:derive a --> stray\n\nMy words.\n\n<!-- workline:derive n -->3<!-- workline:end --> cases.\n"
	onBase := "# T\n\n<!-- workline:derive a --> stray\n\nTheir words.\n\n<!-- workline:derive n -->5<!-- workline:end --> cases.\n"
	if got := asBase(unclosed, onBase); got != "# T\n\n<!-- workline:derive a --> stray\n\nMy words.\n\n<!-- workline:derive n -->5<!-- workline:end --> cases.\n" {
		t.Errorf("an unclosed marker: %q", got)
	}
	// A header not set one key a line is merged as text: both moving
	// `checked` beside a list of sources is left to the person.
	listed := "---\nsources:\n  - a.go\nchecked: 111\n---\n# T\n"
	if _, ok, err := settleDoc(repo, "docs/t.md", listed, strings.Replace(listed, "111", "222", 1), strings.Replace(listed, "111", "333", 1)); ok || err != nil {
		t.Errorf("a header of several lines a key: settled (%v), want it left", err)
	}
}
