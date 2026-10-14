package forge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The pull requests merged since a moment: read page by page, the last
// updated first, until a page ends before it; one closed and not merged,
// or merged before, left out. Each with the commit it became and its head.
func TestGitHubMergedSince(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	page := func(n, from int, updated string) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, `{"id":%d,"merged":"","as":"","updated":"%s","branch":"b","base":"main","here":true,"title":"t","author":"a","bot":false}`+"\n", from-i, updated)
		}
		return b.String()
	}
	first := `{"id":300,"merged":"2026-10-06T10:00:00Z","as":"abc","updated":"2026-10-07T00:00:00Z","branch":"feat","base":"main","here":true,"title":"Feat","author":"dev","bot":false}` + "\n" +
		`{"id":299,"merged":"2026-10-01T10:00:00Z","as":"old","updated":"2026-10-06T00:00:00Z","branch":"old","base":"main","here":true,"title":"Old","author":"dev","bot":false}` + "\n" +
		page(98, 298, "2026-10-05T12:00:00Z")
	second := page(100, 200, "2026-10-04T00:00:00Z")
	os.WriteFile(filepath.Join(bin, "page1"), []byte(first), 0o644)
	os.WriteFile(filepath.Join(bin, "page2"), []byte(second), 0o644)
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\nd=" + bin + "\n" + `case "$*" in
*"page=1 "*) cat "$d/page1" ;;
*"page=2 "*) cat "$d/page2" ;;
*) echo "asked for a third page" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	got, err := g.MergedSince(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 300 || got[0].MergedAs != "abc" || got[0].Head != "refs/pull/300/head" || got[0].MergedAt != "2026-10-06T10:00:00Z" {
		t.Fatalf("merged: %+v", got)
	}
	calls, _ := os.ReadFile(log)
	if n := strings.Count(string(calls), "pulls?state=closed&sort=updated&direction=desc"); n != 2 {
		t.Errorf("pages asked: %d, want 2 (the second ends before since):\n%s", n, calls)
	}
}

// GitLab's merge requests merged since a moment, asked updated since: each
// with its merge commit, else its squash commit, else its head (a
// fast-forward); one merged before since left out.
func TestGitLabMergedSince(t *testing.T) {
	var asked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		if p != "/merge_requests" {
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
			return
		}
		asked = r.URL.RawQuery
		json.NewEncoder(w).Encode([]map[string]any{
			{"iid": 7, "source_branch": "feat", "target_branch": "main", "source_project_id": 3, "target_project_id": 3, "title": "Feat",
				"merged_at": "2026-10-06T10:00:00.000Z", "merge_commit_sha": "m7", "squash_commit_sha": "s7", "sha": "h7"},
			{"iid": 8, "source_branch": "fix", "target_branch": "main", "source_project_id": 3, "target_project_id": 3, "title": "Fix",
				"merged_at": "2026-10-07T10:00:00Z", "squash_commit_sha": "s8", "sha": "h8"},
			{"iid": 9, "source_branch": "ff", "target_branch": "main", "source_project_id": 3, "target_project_id": 3, "title": "FF",
				"merged_at": "2026-10-08T10:00:00Z", "sha": "h9"},
			{"iid": 5, "source_branch": "old", "target_branch": "main", "source_project_id": 3, "target_project_id": 3, "title": "Old",
				"merged_at": "2026-10-01T10:00:00Z", "merge_commit_sha": "m5"},
		})
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}
	got, err := g.MergedSince(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var as []string
	for _, m := range got {
		as = append(as, fmt.Sprintf("!%d %s %s", m.ID, m.MergedAs, m.Head))
	}
	want := []string{"!7 m7 refs/merge-requests/7/head", "!8 s8 refs/merge-requests/8/head", "!9 h9 refs/merge-requests/9/head"}
	if fmt.Sprint(as) != fmt.Sprint(want) {
		t.Errorf("merged:\n%q\nwant\n%q", as, want)
	}
	if !strings.Contains(asked, "state=merged") || !strings.Contains(asked, "updated_after=2026-10-05T00%3A00%3A00Z") {
		t.Errorf("asked: %s", asked)
	}
}
