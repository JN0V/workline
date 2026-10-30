package tester

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A fresh tree is the commit as it was: no replacement, no attribute
// filter, no ignored file of the repository it borrows objects from.
func TestFreshTreeReadsNothingButTheCommit(t *testing.T) {
	repo := t.TempDir()
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir(), "GIT_CONFIG_NOSYSTEM=1"}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run(repo, "init", "-q", "-b", "main")
	write("run.sh", "real\n")
	run(repo, "add", ".")
	run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "start")
	start := run(repo, "rev-parse", "HEAD")
	// What a worktree's agent could leave in the store it shares.
	write("run.sh", "patched\n")
	run(repo, "add", "run.sh")
	fake := run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit-tree", run(repo, "write-tree"), "-m", "fake")
	run(repo, "replace", "-f", start, fake)
	run(repo, "config", "filter.evil.smudge", "echo smudged")
	write(filepath.Join(".git", "info", "attributes"), "* filter=evil\n")
	write("cache.txt", "left\n")
	if got := run(repo, "show", "HEAD:run.sh"); got != "patched" {
		t.Fatalf("the replacement is not read by git itself: HEAD:run.sh = %q", got)
	}

	tree, at, done, f := freshTree(repo, "HEAD")
	if f != nil {
		t.Fatal(f.Message)
	}
	defer done()
	if at != start {
		t.Errorf("at = %s, want the start commit %s", at, start)
	}
	got, err := os.ReadFile(filepath.Join(tree, "run.sh"))
	if err != nil || string(got) != "real\n" {
		t.Errorf("run.sh = %q, %v; want the commit's own", got, err)
	}
	if _, err := os.Stat(filepath.Join(tree, "cache.txt")); err == nil {
		t.Error("cache.txt, not in the commit, is in the fresh tree")
	}
	done()
	if _, err := os.Stat(tree); err == nil {
		t.Error("done left the fresh tree")
	}
}
