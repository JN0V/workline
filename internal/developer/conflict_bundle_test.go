package developer

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/devmark"
	"go.yaml.in/yaml/v3"
)

// Conflict lands: Given an accepted pull request in conflict, When the CI
// develop round settles it (the review job's merge round, whose merged tree
// is written in that job's own .git), Then review-apply, which starts from
// a fresh checkout and gets only the run directory, can push the merge
// commit to its branch: the merged tree `tree:` names in
// out/merge-request.yaml is in that fresh checkout once the run
// directory's git bundle of the merge's objects is read there.
func TestConflictLands(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("WORKLINE_ROLES_DIR", t.TempDir()) // no tester role: the proof is unproven, never red
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(dir, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The review job's checkout: a branch and a main in conflict on a.txt.
	repo := t.TempDir()
	git(repo, "init", "-q", "-b", "main")
	write(repo, "a.txt", "one\n")
	git(repo, "add", "a.txt")
	git(repo, "commit", "-q", "-m", "first")
	git(repo, "checkout", "-q", "-b", "workline/developer/3")
	write(repo, "a.txt", "branch\n")
	git(repo, "commit", "-q", "-am", "branch change")
	head := git(repo, "rev-parse", "HEAD")
	git(repo, "checkout", "-q", "main")
	write(repo, "a.txt", "main\n")
	git(repo, "commit", "-q", "-am", "main change")
	base := git(repo, "rev-parse", "HEAD")

	// review-apply's fresh checkout, made before the round settles anything.
	fresh := filepath.Join(t.TempDir(), "fresh")
	git(repo, "clone", "-q", repo, fresh)

	// The agent's settled merge, written in the review job's .git only.
	git(repo, "checkout", "-q", "--detach", head)
	write(repo, "a.txt", "branch and main\n")
	git(repo, "commit", "-q", "-am", "settled")
	settledAt := git(repo, "rev-parse", "HEAD")

	runDir := t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	bounds, err := json.Marshal(Bounds{Issue: 3})
	if err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(runDir, "in"), boundsFile, string(bounds))
	r := round{MR: 4, Branch: "workline/developer/3", Record: Act{Act: "code", Issue: 3}, At: head, Merge: base, Base: settledAt, Conflicts: []string{"a.txt"},
		Asked: []devmark.Asked{{Key: "conflict|" + base, Where: "main", Holds: conflictHolds, Text: "conflicts with main in a.txt"}}}
	if code := mergePost(runDir, repo, Settings{}, r); code != 0 {
		t.Fatalf("mergePost exited %d", code)
	}

	data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var mr struct {
		Tree string `yaml:"tree"`
	}
	if err := yaml.Unmarshal(data, &mr); err != nil || mr.Tree == "" {
		t.Fatalf("out/merge-request.yaml names no tree: %q, %v", data, err)
	}
	if out, err := exec.Command("git", "-C", fresh, "cat-file", "-e", mr.Tree).CombinedOutput(); err == nil {
		t.Fatalf("the fresh checkout already holds the tree %s: %s", mr.Tree, out)
	}

	// Only the run directory travels between the jobs: any git bundle in
	// it is read in the fresh checkout, as review-apply reads it.
	err = filepath.WalkDir(runDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		_ = exec.Command("git", "-C", fresh, "bundle", "unbundle", path).Run() // not a bundle: skipped
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	kind, err := exec.Command("git", "-C", fresh, "cat-file", "-t", mr.Tree).Output()
	if err != nil || strings.TrimSpace(string(kind)) != "tree" {
		t.Errorf("the merged tree %s is not in a fresh checkout after the run directory's bundle is read: review-apply refuses the merge, and nothing lands", mr.Tree)
	}
}
