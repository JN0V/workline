package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/routing"
)

func behindChecks(t *testing.T, rel, tplPath, extra string) []Check {
	t.Helper()
	tpl, err := os.ReadFile(filepath.Join("..", "..", tplPath))
	if err != nil {
		t.Fatal(err)
	}
	engine := string(pinLine.FindSubmatch(tpl)[1])
	repo := t.TempDir()
	dst := filepath.Join(repo, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, append(tpl, []byte(extra)...), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &Report{}
	ciTemplates(r, Options{Repo: repo, Engine: engine}, &routing.Config{Events: map[string][]string{}})
	return r.Checks
}

func TestWorkflowBehindCleanCustomisedAndGitLab(t *testing.T) {
	gh := ".github/workflows/workline.yml"
	ghTpl := "defaults/ci/github/workline.yml"
	if c := behindChecks(t, gh, ghTpl, ""); len(c) != 0 {
		t.Errorf("clean copy warned: %+v", c)
	}
	if c := behindChecks(t, gh, ghTpl, "\n# workline:customised\n# extra\n"); len(c) != 0 {
		t.Errorf("acknowledged copy warned: %+v", c)
	}
	if c := behindChecks(t, ".gitlab-ci.yml", "defaults/ci/gitlab/workline.gitlab-ci.yml", "\n# extra\n"); len(c) != 0 {
		t.Errorf("gitlab pipeline with other content warned: %+v", c)
	}
}
