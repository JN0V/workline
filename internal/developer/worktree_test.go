package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The worktree's diff: one patch a text file, untracked files in, a
// deletion a patch, a link refused, a link out of the tree said so, a
// name holding a link's mode no link; the worktree left to the engine.
func TestWorktreeCode(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(repo, "gone.sh"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "gone.sh"},
		{"-c", "user.name=x", "-c", "user.email=x@example.invalid", "commit", "-q", "-m", "start"},
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
		os.WriteFile(filepath.Join(tree, "mode 120000.txt"), []byte("b\n"), 0o644),
		os.Remove(filepath.Join(tree, "gone.sh")),
		os.Symlink("/etc/hosts", filepath.Join(tree, "out")),
		os.Symlink("a.sh", filepath.Join(tree, "in")),
		os.WriteFile(filepath.Join(tree, "b.bin"), []byte{0, 1, 2, 0}, 0o644),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	patches, refused, err := worktreeCode(run, at)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range patches {
		diff := p.Value.(string)
		if strings.Contains(diff, "100755") {
			t.Errorf("a mode read: %s", diff)
		}
		files, _ := touched(diff)
		got = append(got, strings.Join(files, ","))
		if strings.Contains(diff, "gone.sh") && !strings.Contains(diff, "deleted file mode") {
			t.Errorf("gone.sh's patch is no deletion: %s", diff)
		}
	}
	if want := "a.sh gone.sh mode 120000.txt"; strings.Join(got, " ") != want {
		t.Errorf("patches touch %q, want %q", strings.Join(got, " "), want)
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
	if _, err := os.Stat(tree); err != nil {
		t.Errorf("the worktree is gone, the engine's to remove: %v", err)
	}
}
