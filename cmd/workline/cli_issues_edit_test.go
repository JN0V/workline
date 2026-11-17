package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// workline runs the built binary in repo with stdin, and returns what it
// printed and its exit code.
func workline(t *testing.T, bin, repo, stdin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repo
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@example.com",
		"GITHUB_ACTIONS=", "GITHUB_REPOSITORY=", "GITHUB_TOKEN=", "GH_TOKEN=",
		"GITLAB_CI=", "GITLAB_TOKEN=", "CI=")
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		code = 1
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}
	return string(out), code
}

// Edit on a local forge:
// Given an issue on `forge: local`,
// When a person runs `workline issues edit 1 --section Validation` with
// the new text on stdin,
// Then the Validation is the new text and the other sections stay
// byte-identical.
func TestIssuesEditChangesOneSectionOnTheLocalForge(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	bin := filepath.Join(t.TempDir(), "workline")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("forge: local\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	top := "## Need\n\nA calculator multiplies.\n\n## Verification\n\nThe product is right.\n\n"
	oldValidation := "- **Two by three:** Given 2 and 3, When multiplied, Then 6."
	newValidation := "- **Four by five:** Given 4 and 5, When multiplied, Then 20."
	bottom := "## Scope\n\ncalc/calc.go"
	body := top + "## Validation\n\n" + oldValidation + "\n\n" + bottom

	if out, code := workline(t, bin, repo, body, "issues", "open", "Mean", "--body-file", "-"); code != 0 {
		t.Fatalf("issues open: exit %d\n%s", code, out)
	}
	out, code := workline(t, bin, repo, newValidation, "issues", "edit", "1", "--section", "Validation")
	if code != 0 {
		t.Fatalf("issues edit 1 --section Validation: exit %d, want 0\n%s", code, out)
	}

	shown, code := workline(t, bin, repo, "", "issues", "show", "1")
	if code != 0 {
		t.Fatalf("issues show 1: exit %d\n%s", code, shown)
	}
	if !strings.Contains(shown, top+"## Validation\n\n"+newValidation) {
		t.Errorf("the Validation is not the new text, or what is above it changed:\n%s", shown)
	}
	if !strings.Contains(shown, "\n\n"+bottom) {
		t.Errorf("the Scope below the Validation changed:\n%s", shown)
	}
	if strings.Contains(shown, "Given 2 and 3") {
		t.Errorf("the old Validation is still there:\n%s", shown)
	}
}
