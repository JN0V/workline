package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Pack kept: Given init wrote a `packs:` line, When a person follows
// local-line.md step 2, Then the pack is still set: the step's config
// block keeps the `packs:` line, and does not tell the reader to replace
// the file init wrote.
func TestLocalLineStepTwoKeepsThePacksLine(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "local-line.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	start := strings.Index(doc, "## 2. Route the line")
	end := strings.Index(doc, "## 3.")
	if start < 0 || end < start {
		t.Fatal("docs/local-line.md has no step 2 before step 3")
	}
	step := doc[start:end]
	if !strings.Contains(step, "packs:") {
		t.Errorf("step 2 never mentions the `packs:` line init wrote:\n%s", step)
	}
	if strings.Contains(step, "replace it") {
		t.Errorf("step 2 tells the reader to replace the file init wrote, packs line included:\n%s", step)
	}
}

// No CI warning: Given `forge: local`, When doctor runs, Then it gives no
// CI template warning (neither the documentalist running in no CI, which
// says to copy the CI templates, nor a workflow outdated or missing).
func TestDoctorOnLocalForgeGivesNoCITemplateWarning(t *testing.T) {
	isolate(t)
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `forge: local
routing:
  events:
    pre-push: [committer, documentalist]
    merge-request: [committer, documentalist, tester, reviewer]
    issue: [product-owner, reviewer, tester, developer]
autonomy: normal
`
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Run(Options{Repo: repo, RolesDir: rolesDir, Engine: "v0.71.0"})
	for _, c := range r.Checks {
		if c.Level == Error && c.Rule == "config-invalid" {
			t.Fatalf("setup: the config does not load: %+v", c)
		}
	}
	for _, c := range r.Checks {
		if c.Level != Warn {
			continue
		}
		switch c.Rule {
		case "docs-judged-nowhere", "workflow-outdated", "workflow-missing":
			t.Errorf("forge: local gives a CI template warning: %+v", c)
		}
		if strings.Contains(c.Fix, "CI templates") {
			t.Errorf("forge: local gives a warning that sends to the CI templates: %+v", c)
		}
	}
}
