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
// derived only when they read what their commands give; any other line of
// a doc — text after a marker with no end of its own too —, or a doc
// added, docs.
func TestRoutineKind(t *testing.T) {
	repo, run, write := gitRepo(t)
	doc := "---\nsources: [a.go]\nchecked: 111\n---\n# A\n\n<!-- workline:derive cases -->3<!-- workline:end -->\n\nOne hour.\n"
	unclosed := "---\nchecked: 111\n---\n# B\n\n<!-- workline:derive cases -->3\n\nOne hour.\n\n<!-- workline:derive more -->1<!-- workline:end -->\n"
	write("doc.md", doc)
	write("unclosed.md", unclosed)
	run("add", "doc.md", "unclosed.md")
	run("commit", "-q", "-m", "docs: one hour")
	base := run("rev-parse", "HEAD")
	asGiven := func([]string) bool { return true }
	notAsGiven := func([]string) bool { return false }
	for _, c := range []struct {
		name, file, doc, add string
		regenerated          func([]string) bool
		want                 string
	}{
		{"a record", "doc.md", strings.Replace(doc, "checked: 111\n", "checked: 111\njudged: 222\n", 1), "", asGiven, kindRecords},
		{"a derived block as given", "doc.md", strings.Replace(strings.Replace(doc, ">3<", ">4<", 1), "checked: 111\n", "checked: 111\njudged: 222\n", 1), "", asGiven, kindDerived},
		{"a derived block not as given", "doc.md", strings.Replace(doc, ">3<", ">4<", 1), "", notAsGiven, kindDocs},
		{"a derived block, no command to tell", "doc.md", strings.Replace(doc, ">3<", ">4<", 1), "", nil, kindDocs},
		{"text after an unclosed marker", "unclosed.md", strings.Replace(unclosed, "One hour.", "Every change approved.", 1), "", asGiven, kindDocs},
		{"its text", "doc.md", strings.Replace(doc, "One hour.", "Two hours.", 1), "", asGiven, kindDocs},
		{"its checked", "doc.md", strings.Replace(doc, "checked: 111", "checked: 333", 1), "", asGiven, kindDocs},
		{"a doc added", "doc.md", doc, "new.md", asGiven, kindDocs},
	} {
		run("checkout", "-q", "-B", "work", base)
		write(c.file, c.doc)
		if c.add != "" {
			write(c.add, "New.\n")
		}
		run("add", ".")
		run("commit", "-q", "--allow-empty", "-m", "docs: "+c.name)
		got, err := routineKind(repo, base, run("rev-parse", "HEAD"), c.regenerated)
		if err != nil || got != c.want {
			t.Errorf("%s: %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

// The pages agents are told to read are never workline's to change alone:
// an agent file, a page it links to, a folder it links to whole, a file it
// imports and what that one links to, a prompt or a template by name.
func TestSteering(t *testing.T) {
	repo, run, write := gitRepo(t)
	for p, s := range map[string]string{
		"AGENTS.md":             "Read [the principles](docs/PRINCIPLES.md) first, then [the decisions](docs/adr/).\n\n[ref]: docs/spec/contract.md\n",
		"CLAUDE.md":             "@team/RULES.md\n",
		"team/RULES.md":         "See [style](style.md#words).\n",
		"docs/guide.md":         "A guide.\n",
		"docs/PRINCIPLES.md":    "1. People at both ends.\n",
		"docs/adr/0001-x.md":    "An ADR.\n",
		"docs/spec/other.md":    "Other.\n",
		"docs/review-prompt.md": "A prompt.\n",
	} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(p)), 0o755); err != nil {
			t.Fatal(err)
		}
		write(p, s)
	}
	run("add", ".")
	run("commit", "-q", "-m", "docs: agents read these")
	steers, err := steering(repo, run("rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]bool{
		"AGENTS.md": true, "docs/PRINCIPLES.md": true, "docs/adr/0001-x.md": true, "docs/spec/contract.md": true,
		"team/RULES.md": true, "team/style.md": true, "docs/review-prompt.md": true, "sub/claude.md": true,
		"docs/guide.md": false, "docs/spec/other.md": false, "docs/adr-notes.md": false,
	} {
		if got := steers(p); got != want {
			t.Errorf("%s steers agents: %v, want %v", p, got, want)
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
	history, err := commits(repo, "HEAD", "", 100)
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

// The engine's own record on a merge request is a comment of its App's
// ending with its marker; anyone else's — a person's, another bot's, no
// one named — is never read as one, nor a marker inside a sentence.
func TestOwnNote(t *testing.T) {
	notes := []forge.Note{
		{Body: "Merging, I say.\n\n<!-- workline:merge-alone=records -->", Author: "dana"},
		{Body: "Me too.\n\n<!-- workline:merge-alone=records -->", Author: "claude[bot]", Bot: true},
		{Body: "Nobody.\n\n<!-- workline:merge-alone=records -->"},
		{Body: "Quoting <!-- workline:merge-alone=derived --> in a sentence.", Author: "workline-app[bot]", Bot: true},
		{Body: "**Merging with no person's yes**\n\n<!-- workline:merge-alone=docs -->", Author: "workline-app[bot]", Bot: true},
	}
	if kind, ok := ownNote(notes, "merge-alone=", "workline-app[bot]"); !ok || kind != "docs" {
		t.Errorf("merge-alone: %q, %v; want docs, the App's", kind, ok)
	}
	if _, ok := ownNote(notes, "merge-alone=", ""); ok {
		t.Errorf("read with no App")
	}
}

// Each revert is read once, however far back: the record on the forge says
// the last base commit read, and the next run reads every commit past it —
// a revert 120 commits down is found, and once read, never again.
func TestRevertsPastTheRecordRead(t *testing.T) {
	repo, run, write := gitRepo(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("%s", out)
	}
	run("remote", "add", "origin", origin)
	write("doc.md", "---\nchecked: 111\n---\nOne hour.\n")
	run("add", "doc.md")
	run("commit", "-q", "-m", "docs: one hour")
	run("push", "-q", "origin", "main")
	if err := writeRecord(repo, record{read: run("rev-parse", "HEAD")}); err != nil {
		t.Fatal(err)
	}
	write("doc.md", "---\nchecked: 111\njudged: 222\n---\nOne hour.\n")
	run("commit", "-q", "-am", "docs: record the docs judged (#9)")
	run("revert", "--no-edit", "HEAD")
	for i := 0; i < 120; i++ {
		run("commit", "-q", "--allow-empty", "-m", "chore: and then")
	}
	run("push", "-q", "origin", "main")
	state := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(state, []byte(`{"issues": [], "merge-requests": [{"id": 9, "branch": "workline/documentalist/suspect", "base": "main", "closed": true, "labels": [],
		"comments": [{"body": "Merging.\n\n<!-- workline:merge-alone=records -->", "author": "workline-bot", "bot": true}]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: state, Repo: repo}
	tip := run("rev-parse", "HEAD")
	res := &Result{}
	demoted, err := demotions(f, FollowOptions{Repo: repo}, tip, "workline-bot", res)
	if err != nil || demoted[kindRecords] != "#1" {
		t.Fatalf("120 commits past the record: %v, %v, %+v; want records demoted, #1", demoted, err, res.Findings)
	}
	rec, err := readRecord(repo)
	if err != nil || rec.read != tip || len(rec.demoted) != 1 || rec.demoted[0].issue != 1 || rec.demoted[0].request != 9 {
		t.Fatalf("the record written back: %+v, %v", rec, err)
	}
	res = &Result{}
	if demoted, err = demotions(f, FollowOptions{Repo: repo}, tip, "workline-bot", res); err != nil || demoted[kindRecords] != "#1" || len(res.Findings) != 0 {
		t.Errorf("read again: %v, %v, %+v; want #1 still held, nothing said, no second issue", demoted, err, res.Findings)
	}
}
