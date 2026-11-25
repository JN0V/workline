package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The last merge request from a branch, closed unmerged with a comment
// carrying workline's marker, is workline's close; one merged, one whose
// comments lack the marker, or none from the branch, is not (#592). gh is
// replaced by a script answering as GitHub does, the newest first.
func TestGitHubClosedWith(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" + `case "$*" in
*"head={owner}:workline%2Fdocumentalist%2Freader-a&"*) echo '{"id":5,"state":"closed","merged":""}' ;;
*"head={owner}:workline%2Fdocumentalist%2Freader-b&"*) echo '{"id":6,"state":"closed","merged":"2026-10-20T07:00:00Z"}' ;;
*"head={owner}:workline%2Fdocumentalist%2Freader-d&"*) echo '{"id":8,"state":"closed","merged":""}' ;;
*"head={owner}:workline%2Fdocumentalist%2Freader-e&"*) echo '{"id":9,"state":"open","merged":""}' ;;
*/issues/5/comments*) echo 'Closed by workline: main changed its lines.'; echo '<!-- workline:closed -->' ;;
*/issues/8/comments*) echo 'Not what I want.' ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	for branch, want := range map[string]bool{"a": true, "b": false, "c": false, "d": false, "e": false} {
		got, err := g.ClosedWith("workline/documentalist/reader-"+branch, Marker("closed"))
		if err != nil || got != want {
			t.Errorf("reader-%s: %v, %v; want %v", branch, got, err, want)
		}
	}
}

// GitLab lists the merge requests from a source branch, the newest first:
// one `closed`, its notes carrying the marker, is workline's close; a
// merged one is `merged` there.
func TestGitLabClosedWith(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		q := r.URL.Query()
		switch {
		case p == "/merge_requests" && q.Get("order_by") == "created_at" && q.Get("sort") == "desc":
			states := map[string]map[string]any{
				"workline/documentalist/reader-a": {"iid": 5, "state": "closed"},
				"workline/documentalist/reader-b": {"iid": 6, "state": "merged"},
				"workline/documentalist/reader-d": {"iid": 8, "state": "closed"},
			}
			out := []map[string]any{}
			if s, ok := states[q.Get("source_branch")]; ok {
				out = append(out, s)
			}
			_ = json.NewEncoder(w).Encode(out)
		case p == "/merge_requests/5/notes":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "body": "Closed by workline.\n\n<!-- workline:closed -->"}})
		case p == "/merge_requests/8/notes":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "body": "Not what I want."}})
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}
	for branch, want := range map[string]bool{"a": true, "b": false, "c": false, "d": false} {
		got, err := g.ClosedWith("workline/documentalist/reader-"+branch, Marker("closed"))
		if err != nil || got != want {
			t.Errorf("reader-%s: %v, %v; want %v", branch, got, err, want)
		}
	}
}

// The local and the simulated forge read their own merge requests: the
// last from the branch, by number.
func TestLocalAndFakeClosedWith(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "first")
	l := &Local{Repo: repo}
	for _, b := range []string{"workline/documentalist/reader-a", "workline/documentalist/reader-b"} {
		run("checkout", "-q", "-b", b, "main")
		run("commit", "-q", "--allow-empty", "-m", b)
		if _, err := l.OpenMergeRequest(b, "main", "docs: "+b, "body", false); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.CloseMergeRequest("workline/documentalist/reader-a", "Closed: main changed its lines.", Marker("closed")); err != nil {
		t.Fatal(err)
	}
	for branch, want := range map[string]bool{"a": true, "b": false, "c": false} {
		if got, err := l.ClosedWith("workline/documentalist/reader-"+branch, Marker("closed")); err != nil || got != want {
			t.Errorf("local reader-%s: %v, %v; want %v", branch, got, err, want)
		}
	}

	state := filepath.Join(t.TempDir(), "forge.json")
	data, _ := json.Marshal(FakeState{MergeRequests: []FakeItem{
		{ID: 1, Branch: "workline/documentalist/reader-a", Closed: true, Comments: []FakeComment{{Body: "Closed by workline.\n\n<!-- workline:closed -->"}}},
		{ID: 2, Branch: "workline/documentalist/reader-b", Closed: true, Comments: []FakeComment{{Body: "Closed by workline.\n\n<!-- workline:closed -->"}}},
		{ID: 3, Branch: "workline/documentalist/reader-b", Closed: true, Comments: []FakeComment{{Body: "Not what I want."}}},
		{ID: 4, Branch: "workline/documentalist/reader-d", Closed: true, Merged: true, Comments: []FakeComment{{Body: "<!-- workline:closed -->"}}},
	}})
	if err := os.WriteFile(state, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &Fake{Path: state, Repo: repo}
	for branch, want := range map[string]bool{"a": true, "b": false, "c": false, "d": false} {
		if got, err := f.ClosedWith("workline/documentalist/reader-"+branch, Marker("closed")); err != nil || got != want {
			t.Errorf("fake reader-%s: %v, %v; want %v", branch, got, err, want)
		}
	}
}
