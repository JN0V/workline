package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// gitRepo is a repository for a test, and a way to run git in it.
func gitRepo(t *testing.T) (string, func(args ...string) string, func(p, s string)) {
	t.Helper()
	repo := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(p, s string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	return repo, run, write
}

// A routine merge request's kind is told by what its change is: only the
// lines recording when docs were read, records; derived blocks besides,
// derived; any other line of a doc, or a doc added, docs.
func TestRoutineKind(t *testing.T) {
	repo, run, write := gitRepo(t)
	doc := "---\nsources: [a.go]\nchecked: 111\n---\n# A\n\n<!-- workline:derive cases -->3<!-- workline:end -->\n\nOne hour.\n"
	write("doc.md", doc)
	run("add", "doc.md")
	run("commit", "-q", "-m", "docs: one hour")
	base := run("rev-parse", "HEAD")
	for _, c := range []struct {
		name, doc, add, want string
	}{
		{"a record", strings.Replace(doc, "checked: 111\n", "checked: 111\njudged: 222\n", 1), "", kindRecords},
		{"a derived block", strings.Replace(strings.Replace(doc, ">3<", ">4<", 1), "checked: 111\n", "checked: 111\njudged: 222\n", 1), "", kindDerived},
		{"its text", strings.Replace(doc, "One hour.", "Two hours.", 1), "", kindDocs},
		{"its checked", strings.Replace(doc, "checked: 111", "checked: 333", 1), "", kindDocs},
		{"a doc added", doc, "new.md", kindDocs},
	} {
		run("checkout", "-q", "-B", "work", base)
		write("doc.md", c.doc)
		if c.add != "" {
			write(c.add, "New.\n")
		}
		run("add", ".")
		run("commit", "-q", "--allow-empty", "-m", "docs: "+c.name)
		got, err := routineKind(repo, base, run("rev-parse", "HEAD"))
		if err != nil || got != c.want {
			t.Errorf("%s: %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

// A revert names what it reverts in git's words, GitLab's or GitHub's; the
// merge request is the one the reverted merge names, else the one the
// revert's subject quotes; GitHub's own revert, `Revert "<title>"` alone,
// the merge of that title it undoes exactly, else the newest of that title.
func TestRevertOf(t *testing.T) {
	repo, run, write := gitRepo(t)
	write("doc.md", "One.\n")
	run("add", "doc.md")
	run("commit", "-q", "-m", "docs: record the docs judged (#9)")
	squash := run("rev-parse", "HEAD")
	write("doc.md", "Two.\n")
	run("commit", "-q", "-am", "Merge pull request #12 from acme/workline/documentalist/derived")
	merge := run("rev-parse", "HEAD")
	// Two merges of the same title, by merge commits; the older reverted.
	same := func(n, line string) string {
		run("checkout", "-q", "-b", "b"+n)
		write("doc"+n+".md", line)
		run("add", "doc"+n+".md")
		run("commit", "-q", "-m", "docs: bring the docs in line")
		run("checkout", "-q", "main")
		run("merge", "-q", "--no-ff", "b"+n, "-m", "Merge pull request #"+n+" from acme/workline/documentalist/release", "-m", "docs: bring the docs in line")
		return run("rev-parse", "HEAD")
	}
	same("20", "Three.\n")
	older := run("rev-parse", "HEAD")
	same("21", "Four.\n")
	run("revert", "--no-edit", "-m", "1", older)
	reverted := run("rev-parse", "HEAD")
	run("commit", "-q", "--amend", "-m", "Revert \"docs: bring the docs in line\"") // GitHub's: no body
	githubs := run("rev-parse", "HEAD")
	if githubs == reverted {
		t.Fatal("the revert's message was not rewritten")
	}
	history, err := commits(repo, "HEAD", 100)
	if err != nil {
		t.Fatal(err)
	}
	none := func() []logged { return nil }
	for _, c := range []struct {
		msg    string
		want   int
		revert bool
	}{
		{"Revert \"docs: record the docs judged (#9)\"\n\nThis reverts commit " + squash + ".\n", 9, true},
		{"Revert \"docs: x\" (#14)\n\n* Revert \"docs: x\"\n\nThis reverts commit " + squash + ".\n", 9, true},
		{"Revert \"Merge pull request #12\"\n\nThis reverts commit " + merge + ".\n", 12, true},
		{"Revert \"docs: regenerate the derived blocks\"\n\nThis reverts merge request !21\n", 21, true},
		{"Revert \"docs: condense a page\" (#30)\n\nReverts acme/workline#27\n", 27, true},
		{"Revert \"docs: condense a page (#27)\" (#30)\n\nThis reverts commit 0123456789abcdef.\n", 27, true},
		{"Revert \"feat: a commit pushed straight to main\"\n", 0, true},
		{"docs: the word Revert in a subject (#5)\n", 0, false},
		{"feat: a feature (#6)\n", 0, false},
	} {
		if got, revert := revertOf(repo, logged{"", c.msg}, none); got != c.want || revert != c.revert {
			t.Errorf("%q: %d, %v; want %d, %v", c.msg, got, revert, c.want, c.revert)
		}
	}
	if got, _ := revertOf(repo, history[0], func() []logged { return history[1:] }); got != 20 {
		t.Errorf("GitHub's revert of the older of two merges of one title: %d, want 20, the one it undoes", got)
	}
	if n, title := requestOf("Merge branch 'derived' into 'main'\n\ndocs: regenerate the derived blocks\n\nSee merge request acme/workline!21"); n != 21 || title != "docs: regenerate the derived blocks" {
		t.Errorf("GitLab's merge: %d %q", n, title)
	}
}

// The engine's own record on a merge request is a comment ending with its
// marker, by a bot or by no one the forge names; a person's comment ending
// with the same marker is theirs, and never read as one.
func TestOwnNote(t *testing.T) {
	notes := []forge.Note{
		{Body: "Merging, I say.\n\n<!-- workline:merge-alone=docs -->", Author: "dana"},
		{Body: "**Merging with no person's yes**\n\n<!-- workline:merge-alone=records -->", Author: "workline-app[bot]", Bot: true},
		{Body: "Quoting <!-- workline:merge-demoted=abc --> in a sentence.", Author: ""},
	}
	if kind, ok := ownNote(notes, "merge-alone=", "workline-app[bot]"); !ok || kind != "records" {
		t.Errorf("merge-alone: %q, %v; want records, the bot's", kind, ok)
	}
	if kind, ok := ownNote(notes, "merge-alone=", "dana"); !ok || kind != "docs" {
		t.Errorf("merge-alone, the token dana's: %q, %v; want docs, written as the token's user", kind, ok)
	}
	if _, ok := ownNote(notes, "merge-demoted=abc", ""); ok {
		t.Errorf("a marker inside a sentence read as the engine's")
	}
}
