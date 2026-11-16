package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// No empty runs without an agent: given no agent set and a developer
// pull request in conflict, when the reconcile runs on two following
// hours (the first starts the round, which says it waits for an agent),
// then no round is due on the second: no CI run is started for that
// conflict after the first.
func TestRoundDueNoAgentConflictNotDueAgain(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("WORKLINE_AI", "none")
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
	act := Act{Act: "code", Issue: 1}
	record := "**The developer's record**\n\n" + act.Line()
	if err := l.Comment(forge.Target{Kind: "merge-request", ID: id}, record, forge.Marker("sticky=developer/"+RecordKey)); err != nil {
		t.Fatal(err)
	}

	// First hour: the reconcile finds the round due, and the run it starts
	// has no agent.
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	due, err := RoundDue(l, repo, mr)
	if err != nil || !due {
		t.Fatalf("first hour: RoundDue = %v, %v; want a round due for the conflict", due, err)
	}
	runDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(runDir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	head, base := run("rev-parse", "workline/developer/1"), run("rev-parse", "main")
	if code := conflictPre(runDir, repo, Settings{}, l, l, mr, act, head, base); code != exitNothing {
		t.Fatalf("the round with no agent exited %d, want %d", code, exitNothing)
	}

	// Second hour: the same conflict, still no agent.
	mr, err = l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	due, err = RoundDue(l, repo, mr)
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Errorf("second hour: a round is due again for the same conflict, with no agent: a CI run started for nothing")
	}
}
