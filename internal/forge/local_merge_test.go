package forge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// localMergeRepo is a clone with main checked out and the branch
// workline/developer/1 one commit past it, opened as !1 of the local forge.
func localMergeRepo(t *testing.T) (*Local, string) {
	t.Helper()
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	repo := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "main")
	run("config", "user.name", "Ada")
	run("config", "user.email", "ada@example.invalid")
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-qm", "first")
	run("checkout", "-qb", "workline/developer/1")
	write("b.txt", "b\n")
	run("add", "b.txt")
	run("commit", "-qm", "fix: b", "-m", "Workline-Role: developer")
	run("checkout", "-q", "main")
	l := &Local{Repo: repo}
	if _, err := l.OpenMergeRequest("workline/developer/1", "main", "fix: b", "Closes #1", false); err != nil {
		t.Fatal(err)
	}
	return l, run("rev-parse", "workline/developer/1")
}

// The local forge's merge is a merge commit naming the merge request, its
// base checked out moved with its files; it is the head weighed or
// nothing (#1322).
func TestLocalAutoMergeMerges(t *testing.T) {
	l, head := localMergeRepo(t)
	mr, _ := l.MergeRequest(1)
	if mr.Author != LocalApp {
		t.Fatalf("a role's merge request is authored by %q", mr.Author)
	}
	if merged, err := l.AutoMerge(mr, strings.Repeat("0", 40), true); merged || !Refused(err) || !strings.Contains(Said(err), "moved since") {
		t.Fatalf("another head merged: %v, %v", merged, err)
	}
	merged, err := l.AutoMerge(mr, head, true)
	if !merged || err != nil {
		t.Fatalf("AutoMerge = %v, %v", merged, err)
	}
	msg, _ := exec.Command("git", "-C", l.Repo, "log", "-1", "--format=%B", "main").Output()
	if !strings.Contains(string(msg), "See merge request !1") {
		t.Fatalf("the merge's message: %s", msg)
	}
	if _, err := os.Stat(filepath.Join(l.Repo, "b.txt")); err != nil {
		t.Fatal("the base checked out did not move with its files")
	}
	it, _ := l.Item("merge-request", 1)
	if l.StateOf(it) != "merged" {
		t.Fatalf("state %s", l.StateOf(it))
	}
}

// A base checked out with uncommitted changes, or a change that conflicts
// with it, merges nothing, said; the worktree is never left mid-merge.
func TestLocalAutoMergeRefuses(t *testing.T) {
	l, head := localMergeRepo(t)
	mr, _ := l.MergeRequest(1)
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", l.Repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(l.Repo, "a.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if merged, err := l.AutoMerge(mr, head, true); merged || !Refused(err) || !strings.Contains(Said(err), "uncommitted changes") {
		t.Fatalf("dirty base: %v, %v", merged, err)
	}
	git("commit", "-qam", "change a")
	git("checkout", "-q", "workline/developer/1")
	if err := os.WriteFile(filepath.Join(l.Repo, "a.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("commit", "-qam", "fix: a", "-m", "Workline-Role: developer")
	head2, _ := exec.Command("git", "-C", l.Repo, "rev-parse", "HEAD").Output()
	// main is checked out nowhere now: its ref alone would move.
	if merged, err := l.AutoMerge(mr, strings.TrimSpace(string(head2)), true); merged || !Refused(err) || !strings.Contains(Said(err), "conflicts with main") {
		t.Fatalf("conflict: %v, %v", merged, err)
	}
	if _, err := os.Stat(filepath.Join(l.Repo, ".git", "MERGE_HEAD")); err == nil {
		t.Fatal("left mid-merge")
	}
}

// A yes on a merge request is recorded with the head it was given on,
// and read back by the person of the clone; a verdict is kept per head.
func TestLocalMergeStateAndVerdict(t *testing.T) {
	l, head := localMergeRepo(t)
	if err := l.Label(Target{Kind: "merge-request", ID: 1}, []string{"workline:accepted"}, nil); err != nil {
		t.Fatal(err)
	}
	mr, _ := l.MergeRequest(1)
	st, err := l.MergeState(mr)
	if err != nil || st.Head != head || len(st.Labels) != 1 || !st.Labels[0].Writer || st.Labels[0].Author != "Ada" ||
		len(st.Pushes) != 1 || st.Pushes[0].Head != head {
		t.Fatalf("state %+v, %v", st, err)
	}
	if c, _, _ := l.Verdict(head); c != "" {
		t.Fatalf("a verdict before one was kept: %s", c)
	}
	if err := l.KeepVerdict(head, "pass", "the line passed"); err != nil {
		t.Fatal(err)
	}
	if c, why, err := l.Verdict(head); c != "pass" || why != "the line passed" || err != nil {
		t.Fatalf("Verdict = %s, %s, %v", c, why, err)
	}
	if _, err := l.Item("issue", 9); !NotFound(err) {
		t.Fatalf("an issue not here: %v", err)
	}
}
