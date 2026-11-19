package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// writeMovedFiles writes each file (a slash path, its text) under base.
func writeMovedFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		p := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// askMovedReviewer runs Prompt for a reviewer whose shipped prompts are under
// prompts/, with the project's and the person's files given, each by its path
// in the role's folder.
func askMovedReviewer(t *testing.T, project, person map[string]string) error {
	t.Helper()
	defer func(c, h func() (string, error)) { userConfigDir, userHomeDir = c, h }(userConfigDir, userHomeDir)
	cfg, home, dir, run, repo := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	userConfigDir = func() (string, error) { return cfg, nil }
	userHomeDir = func() (string, error) { return home, nil }
	writeMovedFiles(t, dir, map[string]string{
		"prompts/persona.md":              "You review.",
		"prompts/instruction.md":          "Review the change.",
		"prompts/policy.md":               "Shipped policy.",
		"prompts/knowledge/go.md":         "Go notes.",
		"prompts/skills/review/SKILL.md":  "---\nname: review\ndescription: x\n---\nReview well.",
	})
	writeMovedFiles(t, run, map[string]string{"in/task.md": "The change."})
	writeMovedFiles(t, filepath.Join(repo, ".workline", "roles", "reviewer"), project)
	writeMovedFiles(t, filepath.Join(cfg, "workline", "roles", "reviewer"), person)
	r := &role.Role{Name: "reviewer", Dir: dir, Intentions: []string{"note"}}
	r.Context.Knowledge = []string{"go.md"}
	_, _, err := Prompt(Request{RunDir: run, Repo: repo, Role: r})
	return err
}

// The run refuses: Given the reviewer moved under prompts/, and the
// project's .workline/roles/reviewer/policy.md, When it runs, Then it stops
// and names .workline/roles/reviewer/prompts/policy.md.
func TestPromptRefusesAFlatProjectOverrideOfAMovedRole(t *testing.T) {
	err := askMovedReviewer(t, map[string]string{"policy.md": "Our own policy."}, nil)
	if err == nil {
		t.Fatal("the run went on with a flat override left at its old path")
	}
	if want := ".workline/roles/reviewer/prompts/policy.md"; !strings.Contains(filepath.ToSlash(err.Error()), want) {
		t.Errorf("the refusal does not name %s: %v", want, err)
	}
}

// Every prompt file: Given the same moved role, and a flat persona.md,
// instruction.md, knowledge/go.md or skills/review/SKILL.md in the project's
// folder, When it runs, Then it stops the same way, naming each file's path
// under prompts/.
func TestPromptRefusesEveryFlatPromptFileOfAMovedRole(t *testing.T) {
	err := askMovedReviewer(t, map[string]string{
		"persona.md":              "Our persona.",
		"instruction.md":          "Our instruction.",
		"knowledge/go.md":         "Our Go notes.",
		"skills/review/SKILL.md":  "Our skill.",
	}, nil)
	if err == nil {
		t.Fatal("the run went on with flat overrides left at their old paths")
	}
	msg := filepath.ToSlash(err.Error())
	for _, f := range []string{"persona.md", "instruction.md", "knowledge/go.md", "skills/review/SKILL.md"} {
		if want := ".workline/roles/reviewer/prompts/" + f; !strings.Contains(msg, want) {
			t.Errorf("the refusal does not name %s: %v", want, err)
		}
	}
}

// Both paths: Given a flat policy.md and one under prompts/, When the role
// runs, Then it stops, naming the flat one to remove, even though the new
// one exists.
func TestPromptRefusesAFlatFileEvenWhenTheNewOneExists(t *testing.T) {
	err := askMovedReviewer(t, map[string]string{
		"policy.md":         "The old one.",
		"prompts/policy.md": "The new one.",
	}, nil)
	if err == nil {
		t.Fatal("the run went on with a flat override beside the one under prompts/")
	}
	if want := ".workline/roles/reviewer/policy.md"; !strings.Contains(filepath.ToSlash(err.Error()), want) {
		t.Errorf("the refusal does not name the flat %s to remove: %v", want, err)
	}
}

// The person's folder: Given the same flat file in the person's config
// folder, When the role runs, Then it stops the same way, naming the new
// path.
func TestPromptRefusesAFlatOverrideInThePersonsFolder(t *testing.T) {
	err := askMovedReviewer(t, nil, map[string]string{"policy.md": "My own policy."})
	if err == nil {
		t.Fatal("the run went on with a flat override in the person's folder")
	}
	if want := "workline/roles/reviewer/prompts/policy.md"; !strings.Contains(filepath.ToSlash(err.Error()), want) {
		t.Errorf("the refusal does not name %s: %v", want, err)
	}
}
