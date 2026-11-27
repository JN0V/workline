package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// inputsGit runs git in dir, isolated from the machine's own settings and
// with its own name and email on each commit; it returns what git printed.
func inputsGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Tester", "-c", "user.email=tester@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// inputsMR reads the inputs of a role with no range, on merge request 1 of
// the fake forge (branch feature, base main), in repo.
func inputsMR(t *testing.T, repo string, inputs ...string) (string, error) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "forge.json")
	state := `{"issues":[],"merge-requests":[{"id":1,"branch":"feature","base":"main","title":"Add b","labels":[],"comments":[]}]}`
	if err := os.WriteFile(file, []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"WORKLINE_FORGE=fake:" + file, "WORKLINE_REPO=" + repo, "WORKLINE_TARGET=merge-request:1"}
	got, _, err := readInputs(&role.Role{Name: "reader", Inputs: inputs}, env)
	return got, err
}

// The diff is built from origin/<branch>
// Given a run with no `range` on a merge request in a checkout with no
// local branch,
// When the role reads its input,
// Then the diff is built from `origin/<branch>` and the run does not block.
func TestInputsMergeRequestDiffFromOriginBranch(t *testing.T) {
	origin := t.TempDir()
	inputsGit(t, origin, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(origin, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputsGit(t, origin, "add", ".")
	inputsGit(t, origin, "commit", "-q", "-m", "base")
	inputsGit(t, origin, "checkout", "-q", "-b", "feature")
	if err := os.WriteFile(filepath.Join(origin, "b.txt"), []byte("feature change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputsGit(t, origin, "add", ".")
	inputsGit(t, origin, "commit", "-q", "-m", "add b")
	inputsGit(t, origin, "checkout", "-q", "main")

	clone := filepath.Join(t.TempDir(), "clone")
	inputsGit(t, t.TempDir(), "clone", "-q", origin, clone)
	if out := inputsGit(t, clone, "branch", "--list", "feature"); strings.TrimSpace(out) != "" {
		t.Fatalf("the clone holds a local feature branch: %q", out)
	}

	got, err := inputsMR(t, clone, "target")
	if err != nil {
		t.Fatalf("readInputs with the branch only as origin/feature: %v", err)
	}
	if !strings.Contains(got, "diff --git a/b.txt b/b.txt") || !strings.Contains(got, "+feature change") {
		t.Errorf("the diff does not hold the merge request's change:\n%s", got)
	}
}

// It holds only the merge request's
// Given a merge request whose base moved on after the branch was cut,
// When the diff is built,
// Then it holds only the merge request's own changes.
func TestInputsMergeRequestDiffHoldsOnlyItsOwn(t *testing.T) {
	repo := t.TempDir()
	inputsGit(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputsGit(t, repo, "add", ".")
	inputsGit(t, repo, "commit", "-q", "-m", "base")
	inputsGit(t, repo, "checkout", "-q", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte("feature change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputsGit(t, repo, "add", ".")
	inputsGit(t, repo, "commit", "-q", "-m", "add b")
	inputsGit(t, repo, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(repo, "c.txt"), []byte("main moved on\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputsGit(t, repo, "add", ".")
	inputsGit(t, repo, "commit", "-q", "-m", "main moves")

	got, err := inputsMR(t, repo, "target")
	if err != nil {
		t.Fatalf("readInputs: %v", err)
	}
	if !strings.Contains(got, "+feature change") {
		t.Errorf("the diff lacks the merge request's change:\n%s", got)
	}
	if strings.Contains(got, "c.txt") || strings.Contains(got, "main moved on") {
		t.Errorf("the diff holds what the base gained after the branch was cut:\n%s", got)
	}
}

// The same diff is not given
// Given a role declaring both `target` and `commits` on a merge request,
// When the input is built,
// Then the same diff is not given twice.
func TestInputsTargetAndCommitsGiveTheDiffOnce(t *testing.T) {
	repo := t.TempDir()
	inputsGit(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputsGit(t, repo, "add", ".")
	inputsGit(t, repo, "commit", "-q", "-m", "base")
	inputsGit(t, repo, "checkout", "-q", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte("feature change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputsGit(t, repo, "add", ".")
	inputsGit(t, repo, "commit", "-q", "-m", "add b")
	inputsGit(t, repo, "checkout", "-q", "main")

	got, err := inputsMR(t, repo, "target", "commits")
	if err != nil {
		t.Fatalf("readInputs: %v", err)
	}
	if n := strings.Count(got, "diff --git a/b.txt b/b.txt"); n != 1 {
		t.Errorf("the merge request's diff is given %d times, want once:\n%s", n, got)
	}
	if !strings.Contains(got, "add b") {
		t.Errorf("the commits' messages are missing:\n%s", got)
	}
}
