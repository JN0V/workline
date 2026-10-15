package reviewer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// gitSteps runs git in repo, and commits a file there, each returning what
// git printed: the new commit's id for a commit.
func gitSteps(t *testing.T, repo string) (run func(args ...string) string, commit func(path, text string) string) {
	run = func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	commit = func(path, text string) string {
		t.Helper()
		os.WriteFile(filepath.Join(repo, path), []byte(text), 0o644)
		run("add", path)
		run("commit", "-qm", "change "+path)
		return run("rev-parse", "HEAD")
	}
	return run, commit
}

// A change is known again by its patch id on a new base (#286); the same
// change but for an indent is not: in some languages an indent is code.
func TestSinceKnowsAChangeNotItsIndent(t *testing.T) {
	repo := gitRepo(t, map[string]string{"README.md": "Start.\n"})
	run, commit := gitSteps(t, repo)
	base := run("rev-parse", "HEAD")
	run("checkout", "-qb", "feature")
	commit("job.py", "if ready:\n    run()\n")
	first, err := Record{}.since(repo, base, "HEAD")
	if err != nil || first.Unread != nil || len(first.Patches) != 1 {
		t.Fatalf("the first review: %+v, %v", first, err)
	}
	rec := Record{}.add(first.Commits, first.Patches)

	run("checkout", "-q", base)
	moved := commit("README.md", "Start, and more.\n")
	head := commit("job.py", "if ready:\n    run()\n")
	if sp, _ := rec.since(repo, moved, head); sp.From != head {
		t.Errorf("rebased, the same change: reviewed from %s, want all of it reviewed (%s)", sp.From, head)
	}
	run("checkout", "-q", moved)
	head = commit("job.py", "if ready:\n  run()\n")
	if sp, _ := rec.since(repo, moved, head); sp.From != moved {
		t.Errorf("rebased, the indent changed: reviewed from %s, want the change reviewed again (from %s)", sp.From, moved)
	}
}

// A commit moving a file whole has a patch id, read without renames: on a
// new base, it is known again (#358).
func TestSinceKnowsAMove(t *testing.T) {
	repo := gitRepo(t, map[string]string{"README.md": "Start.\n", "a.go": "package a\n"})
	run, commit := gitSteps(t, repo)
	base := run("rev-parse", "HEAD")
	run("checkout", "-qb", "feature")
	run("mv", "a.go", "b.go")
	run("commit", "-qm", "move")
	first, err := Record{}.since(repo, base, "HEAD")
	if err != nil || first.Unread != nil || len(first.Patches) != 1 {
		t.Fatalf("the first review: %+v, %v", first, err)
	}
	rec := Record{}.add(first.Commits, first.Patches)
	run("checkout", "-q", base)
	moved := commit("README.md", "Start, and more.\n")
	run("mv", "a.go", "b.go")
	run("commit", "-qm", "move")
	head := run("rev-parse", "HEAD")
	if sp, _ := rec.since(repo, moved, head); sp.From != head {
		t.Errorf("rebased, the same move: reviewed from %s, want all of it reviewed (%s)", sp.From, head)
	}
}

// Two commits reviewed, then a branch of one commit merged in: the merge and
// that commit are new, whatever order git lists them in.
func TestSinceCountsTheCommitsNotReviewed(t *testing.T) {
	repo := gitRepo(t, map[string]string{"README.md": "Start.\n"})
	run, commit := gitSteps(t, repo)
	base := run("rev-parse", "HEAD")
	run("checkout", "-qb", "side")
	commit("side.go", "package side\n")
	run("checkout", "-qb", "feature", base)
	commit("a.go", "package a\n")
	reviewed := commit("b.go", "package b\n")
	first, _ := Record{}.since(repo, base, "HEAD")
	rec := Record{}.add(first.Commits, first.Patches)
	run("merge", "-q", "--no-edit", "side")
	sp, err := rec.since(repo, base, "HEAD")
	if err != nil || sp.From != reviewed || sp.New != 2 || len(sp.Commits) != 4 {
		t.Errorf("after the merge: from %s, %d new of %d, %v; want from %s, 2 new of 4", sp.From, sp.New, len(sp.Commits), err, reviewed)
	}
}

// The findings judged, every run's, read back as written; past the
// comment's size, the oldest go first, counted, and the comment stays
// within it (ADR-0046). A title holding what would end a hidden comment
// does not end it.
func TestJudgedLinesFitTheComment(t *testing.T) {
	var r Record
	for i := range 400 {
		r.Judged = append(r.Judged, Answered{Lens: "edge-cases", Where: fmt.Sprintf("a/f%03d.go:%d", i, i+1), At: strings.Repeat("ab", 20),
			Yes: i%2 == 0, Finder: "claude-sonnet-5", Judge: "claude-opus-5", Title: "A title -- with --> in it, <b>and</b> a quote \""})
	}
	r.Gone = 2
	all := r.judgedLines(1 << 20)
	back, gone, held := ReadJudged("text\n" + all + "\ntext")
	if !held || gone != 2 || !slices.Equal(back, r.Judged) {
		t.Fatalf("read back: held %v, gone %d, %d lines, equal %v", held, gone, len(back), slices.Equal(back, r.Judged))
	}
	if n := strings.Count(all, "-->"); n != 401 {
		t.Errorf("%d comments closed, want 401: a title ended one", n)
	}
	body := strings.Repeat("x", 20000)
	text := body + r.judgedLines(SummaryMax-len(body))
	back, gone, _ = ReadJudged(text)
	switch {
	case len(text) > SummaryMax:
		t.Errorf("the comment holds %d characters, over %d", len(text), SummaryMax)
	case len(back) == 0 || back[len(back)-1] != r.Judged[len(r.Judged)-1]:
		t.Errorf("the newest line went")
	case gone != 2+len(r.Judged)-len(back) || back[0] != r.Judged[len(r.Judged)-len(back)]:
		t.Errorf("%d kept, %d gone: want the oldest gone, counted with the 2 before", len(back), gone)
	case len(text)+300 < SummaryMax:
		t.Errorf("the comment holds %d characters: one more line would have fit", len(text))
	}
	if _, gone, _ := ReadJudged("<!-- workline:judged-lines gone=0 -->\n<!-- workline:judged {\"lens\": -->"); gone != 1 {
		t.Errorf("a line that does not read: %d gone, want 1", gone)
	}
	if _, _, held := ReadJudged("<!-- workline:record reviewed= runs=1 -->"); held {
		t.Errorf("a comment an engine before the lines wrote holds them")
	}
}

// A record that does not read keeps the findings judged, read apart; the
// machine's record never holds them.
func TestJudgedLinesOutliveARecordDamaged(t *testing.T) {
	r, err := parseRecord("<!-- workline:record runs=x -->\n<!-- workline:judged-lines gone=3 -->\n<!-- workline:judged {\"lens\":\"tests\",\"where\":\"a.go:1\",\"at\":\"ab\",\"yes\":true,\"finder\":\"\",\"judge\":\"\",\"title\":\"t\"} -->")
	if err == nil || len(r.Judged) != 1 || r.Gone != 3 {
		t.Errorf("a record damaged: %+v, %v; want an error, 1 line and 3 gone kept", r, err)
	}
	if s := (Record{Runs: 1, Judged: r.Judged}).String(); strings.Contains(s, "judged") {
		t.Errorf("the machine's record holds the findings judged: %s", s)
	}
}

// The summary comment a run writes merges the record the comment holds
// by then (#375): a run applied after this one read it keeps its commits,
// its run counted and its findings judged; what blocks is this run's. A
// comment no run wrote since, or one whose record does not read, is
// written over as it is.
func TestMergeSummaryKeepsTheRunAppliedSince(t *testing.T) {
	line := func(at string) Answered {
		return Answered{Lens: "correctness", Where: "a.go:1", At: at, Yes: true, Title: "t"}
	}
	comment := func(r Record) string {
		return "**workline reviewer** — x.\n\n" + r.String() + r.judgedLines(SummaryMax) + "\n\n<!-- workline:sticky=reviewer/summary -->"
	}
	was := Record{Reviewed: []string{"a1"}, Runs: 1, Judged: []Answered{line("a1")}}
	ours := Record{Reviewed: []string{"a1", "c3"}, Runs: 2, Judged: []Answered{line("a1"), line("c3")},
		Blocks: []Block{{Lens: "correctness", Where: "a.go:1", Title: "t"}}}
	body := "**workline reviewer** — ours.\n\n" + ours.String() + ours.judgedLines(SummaryMax)

	if got := MergeSummary(body, was.Read(), comment(was)); got != body {
		t.Errorf("no run applied since: the body changed:\n%s", got)
	}
	if got := MergeSummary(body, was.Read(), "<!-- workline:record runs=x -->"); got != body {
		t.Errorf("a record damaged: the body changed:\n%s", got)
	}
	now := Record{Reviewed: []string{"a1", "b2"}, Runs: 2, Judged: []Answered{line("a1"), line("b2")}}
	got := MergeSummary(body, was.Read(), comment(now))
	if !strings.HasPrefix(got, "**workline reviewer** — ours.") {
		t.Errorf("not this run's summary:\n%s", got)
	}
	m, err := parseRecord(got)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Reviewed, []string{"a1", "b2", "c3"}) || m.Runs != 3 || len(m.Blocks) != 1 ||
		!slices.Equal(m.Judged, []Answered{line("a1"), line("b2"), line("c3")}) {
		t.Errorf("merged: %+v; want a1 b2 c3, 3 runs, this run's block, every run's line once", m)
	}
}
