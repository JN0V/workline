package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A Draft behind main, main having moved a file since (#1941): the agent's
// code starts from the Draft with main merged in, which holds main's
// paths, so its diff applies on main's checkout; made again, it is the
// same commit. A Draft holding main starts from itself; one main does not
// merge into names the files.
func TestDraftOnBase(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	git := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(dir, name, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	repo := t.TempDir()
	git(repo, "init", "-q", "-b", "main")
	write(repo, "internal/tester/proof.go", "package tester\n")
	git(repo, "add", ".")
	git(repo, "commit", "-q", "-m", "first")
	git(repo, "checkout", "-q", "-b", "workline/tester/issue-3")
	write(repo, "tests/proof_test.go", "package tests\n")
	git(repo, "add", ".")
	git(repo, "commit", "-q", "-m", "test: the acceptance tests of #3")
	draft := git(repo, "rev-parse", "HEAD")
	git(repo, "checkout", "-q", "main")

	if from, files, err := draftOnBase(repo, draft, git(repo, "rev-parse", "HEAD")); err != nil || from != draft || files != nil {
		t.Fatalf("a Draft holding main: got %q %v %v, want the Draft itself", from, files, err)
	}

	git(repo, "mv", "internal/tester", "roles")
	git(repo, "commit", "-q", "-m", "refactor: move the tester")
	main := git(repo, "rev-parse", "HEAD")

	from, files, err := draftOnBase(repo, draft, main)
	if err != nil || len(files) > 0 || from == draft || from == "" {
		t.Fatalf("a Draft behind main: got %q %v %v, want a merge", from, files, err)
	}
	if parents := git(repo, "rev-list", "--parents", "-n", "1", from); parents != from+" "+draft+" "+main {
		t.Errorf("the merge's parents: %q, want the Draft then main", parents)
	}
	if got := git(repo, "ls-tree", "-r", "--name-only", from); got != "roles/proof.go\ntests/proof_test.go" {
		t.Errorf("the start tree holds %q, want main's paths and the Draft's tests", got)
	}
	if git(repo, "rev-parse", "refs/heads/workline/tester/issue-3") != draft {
		t.Error("the tester's branch moved")
	}
	// Its agent's edit, as its diff against the start commit: it applies on main.
	tree := t.TempDir()
	git(repo, "worktree", "add", "-q", "--detach", tree, from)
	write(tree, "roles/proof.go", "package tester\n\nfunc Proof() {}\n")
	diff := git(tree, "diff", from)
	cmd := exec.Command("git", "-C", repo, "apply", "--check", "-")
	cmd.Stdin = strings.NewReader(diff + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("its diff does not apply on main: %s", out)
	}

	// Post finds the commit pre started from again.
	if again, _, err := draftOnBase(repo, draft, main); err != nil || again != from {
		t.Errorf("made again: %q %v, want %s", again, err, from)
	}

	// Main changing a file the Draft changed too: no start, the file named.
	git(repo, "checkout", "-q", "main")
	write(repo, "tests/proof_test.go", "package other\n")
	git(repo, "add", ".")
	git(repo, "commit", "-q", "-m", "test: another")
	from, files, err = draftOnBase(repo, draft, git(repo, "rev-parse", "HEAD"))
	if err != nil || from != "" || !slices.Equal(files, []string{"tests/proof_test.go"}) {
		t.Errorf("a Draft main conflicts with: got %q %v %v, want the file in conflict", from, files, err)
	}
}
