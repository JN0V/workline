package reviewer

import "testing"

// The patch a verdict is kept for (ADR-0050, step 6): the same after a
// rebase onto a base that changed another file; a new one once the branch
// changes the file again; the blob of a file the change leaves.
func TestPatchOfKeepsARebaseNotAChange(t *testing.T) {
	repo := gitRepo(t, map[string]string{"a.go": "package a\n\nfunc A() int { return 1 }\n", "b.go": "package a\n"})
	run, commit := gitSteps(t, repo)
	main := run("rev-parse", "HEAD")
	run("checkout", "-qb", "feature")
	commit("a.go", "package a\n\nfunc A() int { return 2 }\n")
	before := patchOf(repo, main, "HEAD", "a.go")
	if before == "" {
		t.Fatal("no patch id for a.go")
	}
	run("checkout", "-q", main)
	commit("b.go", "package a\n\n// B is more.\n")
	base := run("rev-parse", "HEAD")
	run("checkout", "-q", "feature")
	run("rebase", "-q", base)
	if got := patchOf(repo, base, "HEAD", "a.go"); got != before {
		t.Errorf("after a rebase: patch %q, want %q", got, before)
	}
	commit("a.go", "package a\n\n// A is two.\nfunc A() int { return 2 }\n")
	if got := patchOf(repo, base, "HEAD", "a.go"); got == before || got == "" {
		t.Errorf("a.go changed again: patch %q, want a new one", got)
	}
	if got, blob := patchOf(repo, base, "HEAD", "b.go"), run("rev-parse", "HEAD:b.go"); got != blob {
		t.Errorf("b.go left by the change: %q, want its blob %q", got, blob)
	}
}

// The verdict kept is the last on the same finding on the same patch, by
// its lens; a yes below judge-at-least is judged again, and a line written
// before the keys never matches.
func TestJudgedBeforeTheLastAtTheFloor(t *testing.T) {
	same := Answered{Patch: "p1", Finding: "f1"}
	rec := Record{Judged: []Answered{
		{Lens: "correctness", Where: "a.go:3"},
		{Lens: "correctness", Patch: "p1", Finding: "f1", Yes: false, Why: "first"},
		{Lens: "correctness", Patch: "p1", Finding: "f1", Yes: true, Level: "model"},
		{Lens: "edge-cases", Patch: "p1", Finding: "f1", Yes: false},
	}}
	if j, ok := judgedBefore(rec, "correctness", same, "context"); !ok || !j.Yes {
		t.Errorf("got %+v %v, want the last, a yes", j, ok)
	}
	if _, ok := judgedBefore(rec, "correctness", same, "provider"); ok {
		t.Error("a yes at model kept under judge-at-least provider")
	}
	if _, ok := judgedBefore(rec, "correctness", Answered{Patch: "p2", Finding: "f1"}, ""); ok {
		t.Error("kept on another patch")
	}
	if _, ok := judgedBefore(rec, "correctness", Answered{}, ""); ok {
		t.Error("kept with no key")
	}
}
