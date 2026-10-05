package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// GitHub's timeline says when the label was last set and what names the
// issue: a pull request linked or naming it, a commit naming it; another
// label, and an issue naming it, are not read (ADR-0031). The answer is
// the one asked of JN0V/workline #79, 2026-10-05, a commit added.
func TestGitHubTrail(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" + `cat <<'EOF'
{"data":{"repository":{"issue":{"timelineItems":{"nodes":[
{"__typename":"CrossReferencedEvent","createdAt":"2026-10-04T08:00:00Z","source":{"__typename":"PullRequest","number":112}},
{"__typename":"LabeledEvent","createdAt":"2026-10-05T09:01:54Z","label":{"name":"workline:priority/3"}},
{"__typename":"LabeledEvent","createdAt":"2026-10-05T09:02:26Z","label":{"name":"workline:ready"}},
{"__typename":"CrossReferencedEvent","createdAt":"2026-10-05T10:00:00Z","source":{"__typename":"Issue"}},
{"__typename":"ConnectedEvent","createdAt":"2026-10-06T10:00:00Z","subject":{"__typename":"PullRequest","number":180}},
{"__typename":"ReferencedEvent","createdAt":"2026-10-07T10:00:00Z","commit":{"abbreviatedOid":"1a2b3c4"}}
]}}}}}
EOF
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	tr, err := (&github{repo: t.TempDir()}).Trail(79, "workline:ready")
	if err != nil {
		t.Fatal(err)
	}
	want := Trail{Labeled: "2026-10-05T09:02:26Z", Links: []Link{
		{Kind: "pull-request", Ref: "#112", At: "2026-10-04T08:00:00Z"},
		{Kind: "pull-request", Ref: "#180", At: "2026-10-06T10:00:00Z"},
		{Kind: "commit", Ref: "1a2b3c4", At: "2026-10-07T10:00:00Z"},
	}}
	if !reflect.DeepEqual(tr, want) {
		t.Errorf("trail = %+v\nwant %+v", tr, want)
	}
}

// GitLab says when a label was added in its label events, and what names
// an issue in its system notes; a person's note saying the same words is
// not read.
func TestGitLabTrail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj") {
		case "/issues/4/resource_label_events":
			json.NewEncoder(w).Encode([]map[string]any{
				{"action": "add", "created_at": "2026-09-01T10:00:00Z", "label": map[string]any{"name": "workline:ready"}},
				{"action": "remove", "created_at": "2026-09-02T10:00:00Z", "label": map[string]any{"name": "workline:ready"}},
				{"action": "add", "created_at": "2026-09-03T10:00:00Z", "label": map[string]any{"name": "workline:ready"}},
				{"action": "add", "created_at": "2026-09-04T10:00:00Z", "label": map[string]any{"name": "bug"}},
			})
		case "/issues/4/notes":
			json.NewEncoder(w).Encode([]map[string]any{
				{"body": "mentioned in merge request !7", "system": true, "created_at": "2026-09-05T10:00:00Z"},
				{"body": "mentioned in commit group/other@1a2b3c4d5e", "system": true, "created_at": "2026-09-06T10:00:00Z"},
				{"body": "mentioned in commit 1a2b3c4d", "system": false, "created_at": "2026-09-07T10:00:00Z"},
				{"body": "changed the description", "system": true, "created_at": "2026-09-08T10:00:00Z"},
			})
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	tr, err := (&gitlab{repo: t.TempDir()}).Trail(4, "workline:ready")
	if err != nil {
		t.Fatal(err)
	}
	want := Trail{Labeled: "2026-09-03T10:00:00Z", Links: []Link{
		{Kind: "pull-request", Ref: "!7", At: "2026-09-05T10:00:00Z"},
		{Kind: "commit", Ref: "1a2b3c4d5e", At: "2026-09-06T10:00:00Z"},
	}}
	if !reflect.DeepEqual(tr, want) {
		t.Errorf("trail = %+v\nwant %+v", tr, want)
	}
}
