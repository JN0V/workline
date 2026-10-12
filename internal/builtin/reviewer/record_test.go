package reviewer

import (
	"os"
	"os/exec"
	"path/filepath"
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
