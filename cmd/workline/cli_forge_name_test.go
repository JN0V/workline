package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// noForgeOutput builds the workline binary, makes a git repository with no
// `forge:` set and runs the binary in it with args; it returns the output.
func noForgeOutput(t *testing.T, args ...string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	bin := filepath.Join(t.TempDir(), "workline")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	repo := t.TempDir()
	env := append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@example.com",
		"GITHUB_ACTIONS=", "GITHUB_REPOSITORY=", "GITHUB_TOKEN=", "GH_TOKEN=",
		"GITLAB_CI=", "GITLAB_TOKEN=", "CI=")
	for _, c := range [][]string{
		{"init", "-q"},
		{"commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", c...)
		cmd.Dir = repo
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", c, err, out)
		}
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = repo
	cmd.Env = env
	out, _ := cmd.CombinedOutput()
	return string(out)
}

// Developer with no forge
// Given a repository with no `forge:` set,
// When I run `workline develop --ai none`,
// Then the message names `workline develop`, not `--open-merge-request`,
// and still says how to set a forge.
func TestDeveloperWithNoForge(t *testing.T) {
	out := noForgeOutput(t, "develop", "--ai", "none")
	if !strings.Contains(out, "workline develop needs a forge:") {
		t.Errorf("message does not start with `workline develop needs a forge:`:\n%s", out)
	}
	if strings.Contains(out, "--open-merge-request") {
		t.Errorf("message names --open-merge-request:\n%s", out)
	}
	for _, want := range []string{"forge:", ".workline/config.yaml", "--forge", "forge: local"} {
		if !strings.Contains(out, want) {
			t.Errorf("message does not say %q:\n%s", want, out)
		}
	}
}

// Tester with no forge
// Given the same repository,
// When I run `workline test write --issue 1 --ai none`,
// Then the message names `workline test write`, not `--open-merge-request`.
func TestTesterWithNoForge(t *testing.T) {
	out := noForgeOutput(t, "test", "write", "--issue", "1", "--ai", "none")
	if !strings.Contains(out, "workline test write needs a forge:") {
		t.Errorf("message does not start with `workline test write needs a forge:`:\n%s", out)
	}
	if strings.Contains(out, "--open-merge-request") {
		t.Errorf("message names --open-merge-request:\n%s", out)
	}
}
