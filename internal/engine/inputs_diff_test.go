package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// mergeRequestInputsEnv is the environment of a run on merge request 1,
// branch feature into main, with no range: a simulated forge holding it,
// git run apart from the machine's config.
func mergeRequestInputsEnv(t *testing.T, repo string) []string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	state := `{"issues": [], "merge-requests": [{"id": 1, "branch": "feature", "base": "main", "title": "Add b", "body": "Adds b.", "labels": [], "comments": []}]}`
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	return []string{"WORKLINE_REPO=" + repo, "WORKLINE_FORGE=fake:" + path, "WORKLINE_TARGET=merge-request:1"}
}

// The diff is built from origin/<branch>
// Given a run with no range on a merge request in a checkout with no
// local branch (the branch exists only as origin/feature),
// When the role reads its input,
// Then the diff is built from origin/feature and the run does not block.
func TestInputsDiffBuiltFromOriginBranch(t *testing.T) {
	env := mergeRequestInputsEnv(t, "")
	repo, run, write := gitRepo(t)
	env[0] = "WORKLINE_REPO=" + repo
	origin := filepath.Join(t.TempDir(), "origin.git")
	run("init", "-q", "--bare", "-b", "main", origin)
	run("remote", "add", "origin", origin)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("push", "-q", "origin", "main")
	run("checkout", "-q", "-b", "feature")
	write("b.txt", "unique-feature-line\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "feat: add b")
	run("push", "-q", "origin", "feature")
	run("checkout", "-q", "main")
	run("branch", "-q", "-D", "feature")

	out, _, err := readInputs(&role.Role{Inputs: []string{"target"}}, env)
	if err != nil {
		t.Fatalf("readInputs blocked with the branch only on origin: %v", err)
	}
	if !strings.Contains(out, "+unique-feature-line") {
		t.Errorf("the input does not hold the diff of origin/feature: %q", out)
	}
}

// It holds only the merge request's
// Given a merge request whose base moved on after the branch was cut,
// When the diff is built,
// Then it holds only the merge request's own changes.
func TestInputsDiffHoldsOnlyTheMergeRequestsChanges(t *testing.T) {
	repo, run, write := gitRepo(t)
	env := mergeRequestInputsEnv(t, repo)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("checkout", "-q", "-b", "feature")
	write("b.txt", "own-change-line\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "feat: add b")
	run("checkout", "-q", "main")
	write("c.txt", "moved-on-line\n")
	run("add", "c.txt")
	run("commit", "-q", "-m", "feat: the base moves on")

	out, _, err := readInputs(&role.Role{Inputs: []string{"target"}}, env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "+own-change-line") {
		t.Errorf("the input lacks the merge request's own change: %q", out)
	}
	if strings.Contains(out, "moved-on-line") {
		t.Errorf("the input holds what the base gained after the branch was cut: %q", out)
	}
}

// The same diff is not given
// Given a role declaring both target and commits on a merge request,
// When the input is built,
// Then the same diff is not given twice.
func TestInputsSameDiffNotGivenTwice(t *testing.T) {
	repo, run, write := gitRepo(t)
	env := mergeRequestInputsEnv(t, repo)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("checkout", "-q", "-b", "feature")
	write("b.txt", "given-once-line\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "feat: add b")
	run("checkout", "-q", "main")

	out, _, err := readInputs(&role.Role{Inputs: []string{"target", "commits"}}, env)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, "+given-once-line"); n != 1 {
		t.Errorf("the diff is given %d times, want once: %q", n, out)
	}
	if !strings.Contains(out, "feat: add b") {
		t.Errorf("the input lacks the commit's message: %q", out)
	}
}
