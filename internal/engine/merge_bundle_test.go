package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// The apply finds the bundle the merge round writes, where it writes it;
// a run with no bundle (an older engine's) applies as before (#1704).
func TestOwnMergeFindsTheRoundsBundle(t *testing.T) {
	run := t.TempDir()
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "out", "merge-request.yaml"), []byte("merge: abc\ntree: def\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if m := ownMerge(run); m == nil || m.Bundle != "" {
		t.Fatalf("no bundle: want none named, got %+v", m)
	}
	b := filepath.Join(run, developer.MergeBundle)
	if err := os.WriteFile(b, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if m := ownMerge(run); m == nil || m.Bundle != b {
		t.Fatalf("want the bundle %s, got %+v", b, m)
	}
}

// The apply's clone lacks the merged tree, as review-apply's fresh
// checkout does: commitMerge reads it from the round's bundle and commits
// the merge, two parents; with no bundle it refuses as before (#1704).
func TestCommitMergeReadsTheRoundsBundle(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	g := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	tmp := t.TempDir()
	origin, review := filepath.Join(tmp, "origin.git"), filepath.Join(tmp, "review")
	g(tmp, "init", "-q", "--bare", "-b", "main", origin)
	g(tmp, "clone", "-q", origin, review)
	put := func(text, msg string) {
		if err := os.WriteFile(filepath.Join(review, "a.txt"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		g(review, "add", "a.txt")
		g(review, "commit", "-q", "-m", msg)
	}
	put("one\n", "first")
	g(review, "push", "-q", "origin", "HEAD:main")
	g(review, "checkout", "-q", "-b", "fix")
	put("branch\n", "branch change")
	head := g(review, "rev-parse", "HEAD")
	g(review, "push", "-q", "origin", "fix")
	g(review, "checkout", "-q", "main")
	put("main\n", "main change")
	base := g(review, "rev-parse", "HEAD")
	g(review, "push", "-q", "origin", "main")
	// The round's settled tree, in the review clone only, and its bundle.
	blob := g(review, "hash-object", "-w", "--stdin", "--path", "a.txt") // empty stdin: an empty file
	tree := strings.TrimSpace(gIn(t, review, "100644 blob "+blob+"\ta.txt\n", "mktree"))
	at := g(review, "commit-tree", tree, "-p", head, "-p", base, "-m", "settled")
	g(review, "update-ref", "refs/workline/merge-bundle", at)
	bundle := filepath.Join(tmp, "merge.bundle")
	g(review, "bundle", "create", "-q", bundle, "refs/workline/merge-bundle", "^"+head, "^"+base)

	f := &forge.Fake{Path: filepath.Join(tmp, "state.json")}
	mr := forge.MergeRequest{ID: 2, Branch: "fix", Base: "main"}
	apply := func(b string) (string, error) {
		dir := filepath.Join(t.TempDir(), "apply")
		g(tmp, "clone", "-q", origin, dir)
		g(dir, "checkout", "-q", "fix")
		g(dir, "config", "user.name", "T")
		g(dir, "config", "user.email", "t@example.invalid")
		err := commitMerge(f, runState{Repo: dir}, mr, "Merge main", baseMerge{Base: base, Tree: tree, Bundle: b})
		return dir, err
	}
	if _, err := apply(""); err == nil || !strings.Contains(err.Error(), "is not in this clone") {
		t.Fatalf("no bundle: want the refusal of today, got %v", err)
	}
	dir, err := apply(bundle)
	if err != nil {
		t.Fatalf("with the round's bundle: %v", err)
	}
	if got := g(dir, "log", "-1", "--format=%T %P"); got != tree+" "+head+" "+base {
		t.Errorf("the merge committed: want tree and parents %s %s %s, got %s", tree, head, base, got)
	}
}

func gIn(t *testing.T, dir, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return string(out)
}
