package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/routing"
)

// A repository set up at v0.33.0, checked by a v0.71.0 engine: the pin is
// said with its file, and `issue` routed with no workflow running it; with
// the templates of the running release, nothing (#1016).
func TestWorkflowsOutdatedOrMissing(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, pin, routes string) {
		t.Helper()
		body := "env:\n  WORKLINE_VERSION: " + pin + "          # a release of workline\njobs:\n  line:\n    steps:\n"
		for _, e := range strings.Fields(routes) {
			body += "      - run: workline route " + e + "\n"
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	line := &routing.Config{Events: map[string][]string{
		"commit-msg": {"committer"}, "merge-request": {"committer"}, "main": {"documentalist"},
		"schedule": {"documentalist"}, "release": {}, "issue": {"product-owner", "developer"}}}
	run := func(engine string) map[string]Check {
		r := &Report{}
		ciTemplates(r, Options{Repo: repo, Engine: engine}, line)
		got := map[string]Check{}
		for _, c := range r.Checks {
			got[c.Rule+" "+c.Where] = c
		}
		return got
	}

	write("workline.yml", "v0.33.0", "merge-request main")
	write("workline-gardening.yml", "v0.71.0", "schedule")
	got := run("v0.71.0")
	old, ok := got["workflow-outdated .github/workflows/workline.yml"]
	if !ok || !strings.Contains(old.Message, "runs workline v0.33.0, this engine is v0.71.0") || !strings.Contains(old.Fix, "blob/v0.71.0/ci/github/workline.yml") {
		t.Errorf("an old pin: %+v; want workline.yml said, v0.33.0 against v0.71.0", got)
	}
	if _, ok := got["workflow-outdated .github/workflows/workline-gardening.yml"]; ok {
		t.Errorf("a pin of this release said outdated: %+v", got)
	}
	miss, ok := got["workflow-missing .github/workflows/workline-develop.yml"]
	if !ok || !strings.Contains(miss.Fix, "blob/v0.71.0/ci/github/workline-develop.yml") || len(got) != 2 {
		t.Errorf("issue routed, not run: %+v; want workline-develop.yml to copy, and nothing else", got)
	}
	if got := run("(devel)"); len(got) != 1 {
		t.Errorf("a build that is no release compares no pin: %+v", got)
	}

	write("workline.yml", "v0.71.0", "merge-request main")
	write("workline-develop.yml", "v0.71.0", "issue")
	if got := run("v0.71.0"); len(got) != 0 {
		t.Errorf("every template of this release: %+v; want nothing said", got)
	}
}
