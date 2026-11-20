package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// Conflict after main moved: given a developer pull request with no
// conflict when its round ended, when a merge into main makes it conflict
// (no refused-merge record on it yet, no reviewer finding on its head),
// then the reconcile's check finds a developer round due, to merge main in.
func TestRoundDueWhenMainMovedIntoConflict(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("policy.md", "one\n")
	run("add", "policy.md")
	run("commit", "-q", "-m", "docs: policy")
	run("checkout", "-q", "-b", "workline/developer/1")
	write("policy.md", "branch\n")
	run("commit", "-q", "-am", "fix: policy", "-m", "Workline-Role: developer")
	run("checkout", "-q", "main")
	write("policy.md", "main\n")
	run("commit", "-q", "-am", "docs: policy moved on main")

	l := &forge.Local{Repo: repo}
	if _, err := l.Open("an issue", "A bug.", []string{"bug"}); err != nil {
		t.Fatal(err)
	}
	id, err := l.OpenMergeRequest("workline/developer/1", "main", "fix: policy", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	record := "**The developer's record**\n\n" + devmark.Act{Act: "code", Issue: 1}.Line()
	if err := l.Comment(forge.Target{Kind: "merge-request", ID: id}, record, forge.Marker("sticky=developer/"+devmark.RecordKey)); err != nil {
		t.Fatal(err)
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	due, err := RoundDue(l, repo, mr)
	if err != nil || !due {
		t.Errorf("RoundDue = %v, %v; want a round due: the branch conflicts with main", due, err)
	}
}
