package developer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/verdict"
)

// An open pull request saying `Implements #4` on GitHub: GitHub closes
// nothing on that word, so #4 is not skipped as taken. Given a ready issue
// #4 and an open pull request whose body says `Implements #4` on GitHub
// (gh replaced by a script answering as GitHub does), When the developer
// picks (with no agent), Then it names #4 as the issue it would take and
// passes no issue over.
func TestPickIgnoresImplementsOnGitHub(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" + `case "$*" in
*"/pulls?state=open"*) echo '{"id":9,"branch":"feat/four","base":"main","here":true,"author":"dev","bot":false,"title":"feat: four","body":"Implements #4","labels":[],"draft":false,"auto":false}' ;;
*"/issues?state=open"*) echo '{"id":4,"title":"Four","body":"","labels":["` + backlog.LabelReady + `"],"milestone":"","author":"maintainer","association":"OWNER","closed":false,"reason":"","assignees":[],"blocked":0,"subs":0,"repo":"","fields":{}}' ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("WORKLINE_FORGE", "github")
	t.Setenv("WORKLINE_AI", "none")
	t.Setenv("WORKLINE_OPEN_MERGE_REQUESTS", "1")
	t.Setenv("WORKLINE_EVENT", "")
	t.Setenv("WORKLINE_TARGET", "")

	runDir := t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := Pre(runDir, t.TempDir()); code != exitNothing {
		t.Fatalf("exit %d, want %d: #4 was passed over, or the run failed", code, exitNothing)
	}
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != verdict.Pass || !strings.Contains(v.Summary, "would take #4") {
		t.Errorf("verdict %+v, want a pass saying it would take #4", v)
	}
	for _, f := range v.Findings {
		if f.Rule == "issue-passed" {
			t.Errorf("an issue was passed over: %+v", f)
		}
	}
}
