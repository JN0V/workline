package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Doctor names each: Given flat files in both folders, When a person runs
// `workline doctor`, Then each file is listed with its new path.
func TestDoctorNamesEachFlatPromptOverride(t *testing.T) {
	isolate(t)
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	write := func(path, text string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The reviewer's shipped prompts are under prompts/.
	roles := t.TempDir()
	write(filepath.Join(roles, "reviewer", "role.yaml"), "contract: 1\nname: reviewer\n")
	write(filepath.Join(roles, "reviewer", "prompts", "instruction.md"), "Review.")

	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	write(filepath.Join(repo, ".workline", "config.yaml"), "forge: local\nrouting:\n  events:\n    issue: [reviewer]\n")
	project := filepath.Join(repo, ".workline", "roles", "reviewer")
	person := filepath.Join(cfgDir, "workline", "roles", "reviewer")
	write(filepath.Join(project, "policy.md"), "Our policy.")
	write(filepath.Join(project, "persona.md"), "Our persona.")
	write(filepath.Join(person, "instruction.md"), "My instruction.")
	write(filepath.Join(person, "knowledge", "go.md"), "My Go notes.")

	r := Run(Options{Repo: repo, RolesDir: roles})
	var all strings.Builder
	for _, c := range r.Checks {
		all.WriteString(filepath.ToSlash(c.Where + "\n" + c.Message + "\n" + c.Fix + "\n"))
	}
	text := all.String()
	for _, want := range []string{
		".workline/roles/reviewer/prompts/policy.md",
		".workline/roles/reviewer/prompts/persona.md",
		"workline/roles/reviewer/prompts/instruction.md",
		"workline/roles/reviewer/prompts/knowledge/go.md",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("doctor does not list the new path %s:\n%s", want, text)
		}
	}
}
