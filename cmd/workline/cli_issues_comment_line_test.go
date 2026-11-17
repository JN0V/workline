package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Line comment:
// Given a local merge request that changes `bin/sc-event`,
// When a person runs `workline issues comment '!<n>' --body <text>
// --file bin/sc-event --line 40`,
// Then the comment is kept on that file and line, as a thread the
// developer's next round reads and answers there.
func TestIssuesCommentOnALineOfALocalMergeRequest(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "") // in CI, the local forge refuses writes
	}
	bin := filepath.Join(t.TempDir(), "workline")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL="+os.DevNull,
			"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@example.com",
			"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("forge: local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(repo, "bin", "sc-event")
	if err := os.WriteFile(script, []byte(strings.Repeat("echo old\n", 50)), 0o755); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "chore: start")
	git("checkout", "-q", "-b", "workline/developer/6")
	if err := os.WriteFile(script, []byte(strings.Repeat("echo new\n", 50)), 0o755); err != nil {
		t.Fatal(err)
	}
	git("commit", "-q", "-am", "fix: sc-event")
	git("checkout", "-q", "main")

	l := &forge.Local{Repo: repo}
	id, err := l.OpenMergeRequest("workline/developer/6", "main", "fix: sc-event", "Its fix.", false)
	if err != nil {
		t.Fatal(err)
	}

	out, code := workline(t, bin, repo, "", "issues", "comment", fmt.Sprintf("!%d", id),
		"--body", "Quote $1 here.", "--file", "bin/sc-event", "--line", "40")
	if code != 0 {
		t.Fatalf("issues comment --file bin/sc-event --line 40: exit %d, want 0\n%s", code, out)
	}

	threads, err := l.Threads(id)
	if err != nil || len(threads) != 1 {
		t.Fatalf("Threads = %+v, %v: want one thread, on the line", threads, err)
	}
	th := threads[0]
	if th.Path != "bin/sc-event" || th.Line != 40 {
		t.Errorf("the comment is on %s:%d, want bin/sc-event:40", th.Path, th.Line)
	}
	if len(th.Comments) != 1 || !strings.HasPrefix(th.Comments[0].Body, "Quote $1 here.") {
		t.Errorf("the thread's first comment = %+v, want the text written", th.Comments)
	}

	// The developer's round answers it in the thread.
	if err := l.Reply(id, th.ID, "Quoted."); err != nil {
		t.Fatal(err)
	}
	threads, _ = l.Threads(id)
	if len(threads) != 1 || len(threads[0].Comments) != 2 || threads[0].Comments[1].Body != "Quoted." {
		t.Errorf("the answer is not in the thread: %+v", threads)
	}
}
