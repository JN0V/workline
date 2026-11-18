package agent

import (
	"os"
	"path/filepath"
	"testing"

	"strings"

	"github.com/JN0V/workline/internal/role"
)

// promptsAsk assembles the prompt of a role named "committer" whose own
// folder is a copy of files (path relative to the role folder, text), with
// the project's repo and the person's config folder given empty ones, so
// the machine's own config never decides.
func promptsAsk(t *testing.T, files map[string]string, repo, cfg string) (system, user string, err error) {
	t.Helper()
	defer func(c, h func() (string, error)) { userConfigDir, userHomeDir = c, h }(userConfigDir, userHomeDir)
	userConfigDir = func() (string, error) { return cfg, nil }
	userHomeDir = func() (string, error) { return t.TempDir(), nil }
	dir, run := t.TempDir(), t.TempDir()
	for name, text := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(run, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("The task."), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &role.Role{Name: "committer", Dir: dir, Intentions: []string{"note"}}
	return Prompt(Request{RunDir: run, Repo: repo, Role: r})
}

// A moved role: Given a role whose instruction.md is in roles/<role>/prompts/,
// When it runs, Then the agent reads that file.
func TestPromptReadsAMovedRolesInstruction(t *testing.T) {
	system, _, err := promptsAsk(t, map[string]string{
		"prompts/instruction.md": "Commit what the task says.",
	}, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("a role with its instruction in prompts/ does not run: %v", err)
	}
	if !strings.Contains(system, "Commit what the task says.") {
		t.Errorf("the instruction in prompts/ is not read:\n%s", system)
	}
}

// A project override: Given a moved role and
// .workline/roles/<role>/prompts/policy.md in the project, When it runs,
// Then the project's policy is read.
func TestPromptReadsTheProjectsPolicyUnderPrompts(t *testing.T) {
	repo := t.TempDir()
	over := filepath.Join(repo, ".workline", "roles", "committer", "prompts")
	if err := os.MkdirAll(over, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(over, "policy.md"), []byte("The project's policy."), 0o644); err != nil {
		t.Fatal(err)
	}
	_, user, err := promptsAsk(t, map[string]string{
		"prompts/instruction.md": "Commit what the task says.",
		"prompts/policy.md":      "The shipped policy.",
	}, repo, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(user, "The project's policy.") || strings.Contains(user, "The shipped policy.") {
		t.Errorf("the project's policy under prompts/ is not the one read:\n%s", user)
	}
}

// A person's override: Given the same file in the person's config folder
// under workline/roles/<role>/prompts/, When it runs, Then the person's
// policy is read.
func TestPromptReadsThePersonsPolicyUnderPrompts(t *testing.T) {
	cfg := t.TempDir()
	over := filepath.Join(cfg, "workline", "roles", "committer", "prompts")
	if err := os.MkdirAll(over, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(over, "policy.md"), []byte("The person's policy."), 0o644); err != nil {
		t.Fatal(err)
	}
	_, user, err := promptsAsk(t, map[string]string{
		"prompts/instruction.md": "Commit what the task says.",
		"prompts/policy.md":      "The shipped policy.",
	}, t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(user, "The person's policy.") || strings.Contains(user, "The shipped policy.") {
		t.Errorf("the person's policy under prompts/ is not the one read:\n%s", user)
	}
}
