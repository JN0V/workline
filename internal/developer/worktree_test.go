package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The worktree's diff: one patch a text file, untracked files in, a link
// refused, a link out of the tree said so, the worktree removed after.
func TestWorktreeCode(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=x", "-c", "user.email=x@example.invalid", "commit", "-q", "--allow-empty", "-m", "start"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	at, _ := gitOut(repo, "rev-parse", "HEAD")
	run := t.TempDir()
	if err := openWorktree(run, repo, at); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(run, "worktree")
	for _, err := range []error{
		os.WriteFile(filepath.Join(tree, "a.sh"), []byte("a\n"), 0o755),
		os.Symlink("/etc/hosts", filepath.Join(tree, "out")),
		os.Symlink("a.sh", filepath.Join(tree, "in")),
		os.WriteFile(filepath.Join(tree, "b.bin"), []byte{0, 1, 2, 0}, 0o644),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	patches, refused, err := worktreeCode(run, repo, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 1 || !strings.Contains(patches[0].Value.(string), "+++ b/a.sh") || strings.Contains(patches[0].Value.(string), "100755") {
		t.Errorf("patches = %v, want a.sh's alone, its mode not read", patches)
	}
	want := map[string]string{"out": "symlink-out", "in": "not-text", "b.bin": "not-text"}
	for _, r := range refused {
		if want[r.File] != r.Rule {
			t.Errorf("%s refused by %s, want %q", r.File, r.Rule, want[r.File])
		}
		delete(want, r.File)
	}
	if len(want) > 0 {
		t.Errorf("not refused: %v", want)
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Errorf("the worktree is still there: %v", err)
	}
}
