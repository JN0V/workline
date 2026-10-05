package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// GitHub's timeline says when the label was last set and what names the
// issue: a pull request linked or naming it, a commit naming it; another
// label, and an issue naming it, are not read; another repository's pull
// request is named with its repository (ADR-0031). The answer is
// the one asked of JN0V/workline #79, 2026-10-05, a commit added, and the
// label set, taken off and set again: the last setting is read.
func TestGitHubTrail(t *testing.T) {
	bin := t.TempDir()
	args := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + args + "\n" + `cat <<'EOF'
{"data":{"repository":{"issue":{"timelineItems":{"nodes":[
{"__typename":"LabeledEvent","createdAt":"2026-10-01T08:00:00Z","label":{"name":"workline:ready"}},
{"__typename":"UnlabeledEvent","createdAt":"2026-10-02T08:00:00Z","label":{"name":"workline:ready"}},
{"__typename":"CrossReferencedEvent","createdAt":"2026-10-04T08:00:00Z","source":{"__typename":"PullRequest","number":112}},
{"__typename":"LabeledEvent","createdAt":"2026-10-05T09:01:54Z","label":{"name":"workline:priority/3"}},
{"__typename":"LabeledEvent","createdAt":"2026-10-05T09:02:26Z","label":{"name":"workline:ready"}},
{"__typename":"CrossReferencedEvent","createdAt":"2026-10-05T10:00:00Z","source":{"__typename":"Issue"}},
{"__typename":"ConnectedEvent","createdAt":"2026-10-06T10:00:00Z","subject":{"__typename":"PullRequest","number":180}},
{"__typename":"CrossReferencedEvent","createdAt":"2026-10-06T11:00:00Z","isCrossRepository":true,"source":{"__typename":"PullRequest","number":4,"repository":{"nameWithOwner":"JN0V/other"}}},
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
		{Kind: "pull-request", Ref: "JN0V/other#4", At: "2026-10-06T11:00:00Z"},
		{Kind: "commit", Ref: "1a2b3c4", At: "2026-10-07T10:00:00Z"},
	}}
	if !reflect.DeepEqual(tr, want) {
		t.Errorf("trail = %+v\nwant %+v", tr, want)
	}
	// What the answer is read from is asked: the canned answer above would
	// pass whatever the query.
	asked, err := os.ReadFile(args)
	if err != nil {
		t.Fatalf("gh never ran: %v", err)
	}
	// Whole words, whatever the layout: REFERENCED_EVENT is in
	// CROSS_REFERENCED_EVENT too, and 79 in 790.
	for _, w := range []string{"LABELED_EVENT", "CONNECTED_EVENT", "CROSS_REFERENCED_EVENT", "REFERENCED_EVENT", "isCrossRepository", "nameWithOwner", "abbreviatedOid", "number=79"} {
		if !regexp.MustCompile(`\b` + w + `\b`).Match(asked) {
			t.Errorf("gh not asked for %q: %s", w, asked)
		}
	}
}

// GitLab says when a label was added in its label events, and what names
// an issue in its system notes, another project's named with its path; a
// person's note saying the same words is not read.
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
				{"body": "mentioned in merge request group/other!12", "system": true, "created_at": "2026-09-06T11:00:00Z"},
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
		{Kind: "commit", Ref: "group/other@1a2b3c4d5e", At: "2026-09-06T10:00:00Z"},
		{Kind: "pull-request", Ref: "group/other!12", At: "2026-09-06T11:00:00Z"},
	}}
	if !reflect.DeepEqual(tr, want) {
		t.Errorf("trail = %+v\nwant %+v", tr, want)
	}
}
