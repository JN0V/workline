package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The runs going are read in each status not completed, requested too
// (#1761), page after page while a page is full.
func TestGitHubRunsGoing(t *testing.T) {
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls")
	script := "#!/bin/sh\necho \"$2\" >> " + calls + `
case "$2" in
*status=requested*) echo '{"name":"workline gardening backlog","workflow":"workline-gardening.yml","status":"requested","started":"2026-11-16T02:15:00Z"}' ;;
*page=1\&status=in_progress*) i=0; while [ $i -lt 100 ]; do echo '{"name":"workline merge","workflow":"workline-merge.yml","status":"in_progress","started":"2026-11-16T02:00:00Z"}'; i=$((i+1)); done ;;
*page=2\&status=in_progress*) echo '{"name":"workline sample","workflow":"workline-sample.yml","status":"in_progress","started":"2026-11-16T01:00:00Z"}' ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	runs, err := (&github{repo: t.TempDir()}).RunsGoing()
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 102 || runs[0].Status != "requested" || runs[101].Name != "workline sample" {
		t.Errorf("%d runs going, first %+v: want the requested one and both pages of in_progress", len(runs), runs[0])
	}
	sent, _ := os.ReadFile(calls)
	if n := strings.Count(string(sent), "\n"); n != 6 {
		t.Errorf("%d calls, want 6 (5 statuses, a second page of in_progress):\n%s", n, sent)
	}
}
