package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A conflict seen with no agent is not a failed round: nothing is due
// while no agent is set, and a run with an agent on the same head takes it.
func TestNoAgentConflictIsNotAFailedRound(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, "p.md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("one\n")
	run("add", "p.md")
	run("commit", "-q", "-m", "docs: p")
	run("checkout", "-q", "-b", "workline/developer/1")
	write("branch\n")
	run("commit", "-q", "-am", "fix: p")
	head := run("rev-parse", "HEAD")
	run("checkout", "-q", "main")
	write("main\n")
	run("commit", "-q", "-am", "docs: moved")

	l := &forge.Local{Repo: repo}
	if _, err := l.Open("an issue", "A bug.", []string{"bug"}); err != nil {
		t.Fatal(err)
	}
	id, err := l.OpenMergeRequest("workline/developer/1", "main", "fix: p", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	kept, err := forge.ReadKept(l, forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		t.Fatal(err)
	}
	// What the no-agent round records.
	a := Act{Act: "code", Issue: 1, NoAgent: head}

	t.Setenv("WORKLINE_AI", "none")
	if _, due := conflictNow(l, kept, a, repo, mr, head); due {
		t.Error("a round is due with no agent, at a head already seen so")
	}
	t.Setenv("WORKLINE_AI", "some-agent")
	if _, due := conflictNow(l, kept, a, repo, mr, head); !due {
		t.Error("with an agent, the conflict seen with none is not taken: it reads as a failed round")
	}
}
