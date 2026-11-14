package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Not kept
// Given the forge refuses the edit,
// When the wake runs in CI,
// Then a warning shows on the run, not only a log line.
func TestWakeIssueNotKeptWarnsInCI(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "workline")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	repo := t.TempDir()
	env := append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@example.com",
		"GITHUB_ACTIONS=true", "GITHUB_REPOSITORY=", "GITHUB_TOKEN=", "GH_TOKEN=",
		"GITLAB_CI=", "GITLAB_TOKEN=", "CI=true")
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=" + os.DevNull}, args...)...)
		cmd.Dir = repo
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("commit", "-q", "--allow-empty", "-m", "init")
	if err := os.MkdirAll(filepath.Join(repo, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "adr", "0001-x.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-q", "-m", "an adr")
	script := `case "$WORKLINE_FORGE_OPERATION" in keep-issue) echo '{"error":"body too long"}';; *) echo '{}';; esac`
	cmd := exec.Command(bin, "wake", "--forge", "cmd:"+script, "--range", "HEAD~1..HEAD", "--issue")
	cmd.Dir = repo
	cmd.Env = env
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "::warning") {
		t.Errorf("no ::warning annotation on the run:\n%s", out)
	}
}
