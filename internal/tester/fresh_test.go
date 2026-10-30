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

// gitRepo is a repository of one commit holding files, init with args;
// its run runs git in a dir.
func gitRepo(t *testing.T, files map[string]string, initArgs ...string) (string, func(dir string, args ...string) string) {
	t.Helper()
	repo := t.TempDir()
	home := t.TempDir()
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "GIT_CONFIG_NOSYSTEM=1"}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(repo, append([]string{"init", "-q", "-b", "main"}, initArgs...)...)
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run(repo, "add", ".")
	run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "start")
	return repo, run
}

// noLeftover sets the temporary folder to an empty one, and checks at the
// end that freshTree left nothing in it.
func noLeftover(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Cleanup(func() {
		if left, _ := os.ReadDir(tmp); len(left) > 0 {
			t.Errorf("a fresh tree left behind: %v", left)
		}
	})
}

// A sha256 repository: the fresh tree takes its object format.
func TestFreshTreeTakesTheObjectFormat(t *testing.T) {
	repo, run := gitRepo(t, map[string]string{"run.sh": "real\n"}, "--object-format=sha256")
	tree, at, done, f := freshTree(repo, "HEAD")
	if f != nil {
		t.Fatal(f.Message)
	}
	defer done()
	if want := run(repo, "rev-parse", "HEAD"); at != want || len(at) != 64 {
		t.Errorf("at = %s, want %s", at, want)
	}
	if got, err := os.ReadFile(filepath.Join(tree, "run.sh")); err != nil || string(got) != "real\n" {
		t.Errorf("run.sh = %q, %v", got, err)
	}
}

// What a fresh tree cannot check out, or give whole, is a finding, the
// tree removed: never a tree the tests run on, red or green.
func TestFreshTreeFindings(t *testing.T) {
	cases := map[string]struct {
		repo func(t *testing.T) string
		rev  string
		want string
	}{
		"rev missing": {func(t *testing.T) string {
			repo, _ := gitRepo(t, map[string]string{"a": "a\n"})
			return repo
		}, "no-such-rev", "could not be read"},
		"object missing": {func(t *testing.T) string {
			repo, run := gitRepo(t, map[string]string{"a": "a\n"})
			root := run(repo, "rev-parse", "HEAD^{tree}")
			if err := os.Remove(filepath.Join(repo, ".git", "objects", root[:2], root[2:])); err != nil {
				t.Fatal(err)
			}
			return repo
		}, "HEAD", "could not be checked out"},
		"lfs": {func(t *testing.T) string {
			repo, _ := gitRepo(t, map[string]string{".gitattributes": "*.bin filter=lfs diff=lfs merge=lfs -text\n", "x.bin": "pointer\n"})
			return repo
		}, "HEAD", "x.bin names filter lfs"},
		"submodule": {func(t *testing.T) string {
			repo, run := gitRepo(t, map[string]string{"a": "a\n"})
			sub := run(repo, "rev-parse", "HEAD")
			run(repo, "update-index", "--add", "--cacheinfo", "160000,"+sub+",lib")
			run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "sub")
			return repo
		}, "HEAD", "submodule lib"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			repo := c.repo(t)
			noLeftover(t)
			_, _, done, f := freshTree(repo, c.rev)
			if f == nil {
				done()
				t.Fatal("no finding")
			}
			if f.Rule != "check-error" || !strings.Contains(f.Message, c.want) {
				t.Errorf("finding %s: %s; want check-error holding %q", f.Rule, f.Message, c.want)
			}
		})
	}
}
