package line

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// Same on both: given the guarded config and the tuning file beside it,
// the tuning one not committed yet, when a role runs locally (in the
// clone) and in the tree a review lays its head out in, then both read
// the same values from the two files.
func TestTuningFileReadSameInCloneAndReviewTree(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "main")
	dir := filepath.Join(repo, ".workline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("forge: local\nroles:\n  tester:\n    settings:\n      test-command: go test ./...\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "first")
	head := run("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(dir, "tuning.yaml"), []byte("roles:\n  tester:\n    settings:\n      suite-max-seconds: 900\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree, done, err := HeadTree(repo, head)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	for name, root := range map[string]string{"the clone": repo, "the review tree": tree} {
		c, err := role.LoadProjectConfig(root)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s := c.Roles["tester"].Settings
		if s["test-command"] != "go test ./..." {
			t.Errorf("%s: test-command = %v, want the guarded file's", name, s["test-command"])
		}
		if s["suite-max-seconds"] != 900 {
			t.Errorf("%s: suite-max-seconds = %v, want 900 from the tuning file", name, s["suite-max-seconds"])
		}
	}
}
