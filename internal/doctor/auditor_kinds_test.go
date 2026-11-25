package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// auditorKindsDoctor runs doctor on a local project holding the given
// files under .workline/, and returns everything it says, as one text.
func auditorKindsDoctor(t *testing.T, files ...string) string {
	t.Helper()
	isolate(t)
	write := func(path, text string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	roles := t.TempDir()
	write(filepath.Join(roles, "reviewer", "role.yaml"), "contract: 1\nname: reviewer\n")
	write(filepath.Join(roles, "reviewer", "prompts", "instruction.md"), "Review.")

	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	write(filepath.Join(repo, ".workline", "config.yaml"), "forge: local\nrouting:\n  events:\n    issue: [reviewer]\n")
	for _, f := range files {
		write(filepath.Join(repo, filepath.FromSlash(f)), "Our question.")
	}

	r := Run(Options{Repo: repo, RolesDir: roles})
	var all strings.Builder
	for _, c := range r.Checks {
		all.WriteString(filepath.ToSlash(c.Where + "\n" + c.Message + "\n" + c.Fix + "\n"))
	}
	return all.String()
}

// Old question named: Given a project with
// .workline/roles/auditor/kinds/ready.md, When a person runs
// `workline doctor`, Then it names that file and the path it goes to,
// .workline/roles/auditor/criteria/product-owner/ready.md.
func TestDoctorNamesAnOldAuditorQuestionUnderKinds(t *testing.T) {
	text := auditorKindsDoctor(t, ".workline/roles/auditor/kinds/ready.md")
	for _, want := range []string{
		".workline/roles/auditor/kinds/ready.md",
		".workline/roles/auditor/criteria/product-owner/ready.md",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("doctor does not name %s:\n%s", want, text)
		}
	}
}

// Both copies named: Given a project with the old file and its new copy
// both there, When a person runs `workline doctor`, Then it says to
// remove the old file.
func TestDoctorSaysToRemoveTheOldAuditorQuestionWhenItsCopyIsThere(t *testing.T) {
	text := auditorKindsDoctor(t,
		".workline/roles/auditor/kinds/ready.md",
		".workline/roles/auditor/criteria/product-owner/ready.md")
	if !strings.Contains(text, ".workline/roles/auditor/kinds/ready.md") {
		t.Errorf("doctor does not name the old file:\n%s", text)
	}
	low := strings.ToLower(text)
	if !strings.Contains(low, "remove") && !strings.Contains(low, "rm ") {
		t.Errorf("doctor does not say to remove the old file:\n%s", text)
	}
}
