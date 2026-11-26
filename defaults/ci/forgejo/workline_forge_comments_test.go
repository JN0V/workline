package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// everyPage is a Forgejo or Gitea API as the real one is: the comments of
// an issue take `since` and `before` only, so every page asked answers the
// same comments.
func everyPage(comments string) http.Handler {
	const repo = "/api/v1/repos/owner/name"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == repo+"/issues/21/comments":
			_, _ = fmt.Fprint(w, comments)
		case r.Method == "GET" && r.URL.Path == "/api/v1/user":
			_, _ = fmt.Fprint(w, `{"id": 1, "login": "owner"}`)
		case r.Method == "GET" && r.URL.Path == repo+"/collaborators/dev/permission":
			_, _ = fmt.Fprint(w, `{"permission": "write"}`)
		case r.Method == "GET" && r.URL.Path == repo+"/collaborators/zed/permission":
			_, _ = fmt.Fprint(w, `{"permission": "read"}`)
		default:
			http.Error(w, "not mocked", http.StatusNotFound)
		}
	})
}

// routeComments runs `comments` on pull request 21 against a forge that
// answers every page the same, and returns its answer; the script must
// end within 5 seconds.
func routeComments(t *testing.T, comments string) string {
	t.Helper()
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	srv := httptest.NewServer(everyPage(comments))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "workline-forge.sh")
	cmd.WaitDelay = time.Second
	cmd.Stdin = strings.NewReader(`{"target": {"kind": "merge-request", "id": 21}}`)
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=comments",
		"FORGEJO_URL=" + srv.URL, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("the comments were not read within 5 seconds: %v", ctx.Err())
	}
	if err != nil {
		t.Fatalf("comments: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// Given a pull request with 2 comments on a local Forgejo, When the merge
// request is routed, Then both comments are read within 5 seconds.
func TestCommentsReadOnceOnForgejo(t *testing.T) {
	out := routeComments(t, `[{"id": 1, "body": "first", "user": {"id": 8, "login": "dev"}},
		{"id": 2, "body": "second", "user": {"id": 9, "login": "zed"}}]`)
	want := `{"comments":[{"body":"first","author":"dev","insider":true,"bot":false,"own":false},` +
		`{"body":"second","author":"zed","insider":false,"bot":false,"own":false}]}`
	if out != want {
		t.Errorf("answer = %s\nwant     %s", out, want)
	}
}

// Given the same on a local Gitea, When the merge request is routed,
// Then both comments are read within 5 seconds.
func TestCommentsReadOnceOnGitea(t *testing.T) {
	out := routeComments(t, `[{"id": 11, "html_url": "http://gitea/owner/name/pulls/21#issuecomment-11", "body": "first",
		"user": {"id": 2, "login": "dev"}, "created_at": "2026-10-01T09:00:00Z", "updated_at": "2026-10-01T09:00:00Z"},
		{"id": 12, "html_url": "http://gitea/owner/name/pulls/21#issuecomment-12", "body": "second",
		"user": {"id": 3, "login": "zed"}, "created_at": "2026-10-01T10:00:00Z", "updated_at": "2026-10-01T10:00:00Z"}]`)
	want := `{"comments":[{"body":"first","author":"dev","insider":true,"bot":false,"own":false,"created":"2026-10-01T09:00:00Z"},` +
		`{"body":"second","author":"zed","insider":false,"bot":false,"own":false,"created":"2026-10-01T10:00:00Z"}]}`
	if out != want {
		t.Errorf("answer = %s\nwant     %s", out, want)
	}
}
