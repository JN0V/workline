package forge

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ghStub puts a gh on the PATH answering by its arguments, each call
// logged: the case's body is sh's.
func ghStub(t *testing.T, cases string) string {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	script := "#!/bin/sh\necho \"$*\" >> " + log + "\ncase \"$*\" in\n" + cases + "*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// GitHub's closing references are asked of many pull requests in one
// GraphQL call; an issue of another repository is left out (ADR-0068 §8).
func TestGitHubCloses(t *testing.T) {
	answer := `{"data":{"repository":{"nameWithOwner":"o/r",` +
		`"p10":{"closingIssuesReferences":{"nodes":[{"number":5,"state":"OPEN","repository":{"nameWithOwner":"o/r"},"labels":{"nodes":[{"name":"bug"}]}},` +
		`{"number":6,"state":"CLOSED","repository":{"nameWithOwner":"x/y"},"labels":{"nodes":[]}}]}},` +
		`"p11":{"closingIssuesReferences":{"nodes":[]}},"p12":null}}}`
	log := ghStub(t, "*graphql*) echo '"+answer+"' ;;\n")
	got, err := (&github{repo: t.TempDir()}).Closes([]int{10, 11})
	calls, _ := os.ReadFile(log)
	if err != nil || len(got[10]) != 1 || got[10][0].ID != 5 || got[10][0].Closed || got[10][0].Labels[0] != "bug" || len(got[11]) != 0 ||
		strings.Count(string(calls), "graphql") != 1 || !strings.Contains(string(calls), "p11: pullRequest(number: 11)") {
		t.Errorf("%+v, %v; calls:\n%s", got, err, calls)
	}
	// A pull request GitHub answers null for is an error, never one
	// closing nothing.
	if _, err := (&github{repo: t.TempDir()}).Closes([]int{10, 11, 12}); err == nil || !strings.Contains(err.Error(), "#12") {
		t.Errorf("p12 null: %v", err)
	}
}

// The cord is read in one listing, pull requests told from issues.
func TestGitHubLabelled(t *testing.T) {
	ghStub(t, "*labels=step-back*) printf '%s\\n%s\\n' '{\"id\":7,\"pr\":false}' '{\"id\":9,\"pr\":true}' ;;\n")
	got, err := (&github{repo: t.TempDir()}).Labelled("step-back")
	if err != nil || len(got) != 2 || got[0] != (Target{Kind: "issue", ID: 7}) || got[1] != (Target{Kind: "merge-request", ID: 9}) {
		t.Errorf("%+v, %v", got, err)
	}
}

// GitLab's closing links are each merge request's closes_issues; the cord,
// its open issues then its open merge requests bearing the label.
func TestGitLabWoken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case p == "/merge_requests/10/closes_issues":
			_, _ = w.Write([]byte(`[{"iid":5,"state":"opened","labels":["type::bug"]}]`))
		case p == "/merge_requests/11/closes_issues":
			_, _ = w.Write([]byte(`[]`))
		case p == "/issues" && r.URL.Query().Get("labels") == "step-back" && r.URL.Query().Get("state") == "opened":
			_, _ = w.Write([]byte(`[{"iid":7}]`))
		case p == "/merge_requests" && r.URL.Query().Get("labels") == "step-back":
			_, _ = w.Write([]byte(`[{"iid":9}]`))
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
	closes, err := g.Closes([]int{10, 11})
	if err != nil || len(closes[10]) != 1 || closes[10][0].ID != 5 || closes[10][0].Closed || len(closes[11]) != 0 {
		t.Errorf("closes %+v, %v", closes, err)
	}
	items, err := g.Labelled("step-back")
	if err != nil || len(items) != 2 || items[0] != (Target{Kind: "issue", ID: 7}) || items[1] != (Target{Kind: "merge-request", ID: 9}) {
		t.Errorf("labelled %+v, %v", items, err)
	}
}

// With no App named, the yes is not read: never taken for any app's.
func TestGitHubAppStatusNoAppNamed(t *testing.T) {
	t.Setenv("WORKLINE_APP", "")
	log := ghStub(t, "*commits/c0ffee/statuses*) echo 42 ;;\n")
	yes, err := (&github{repo: t.TempDir()}).AppStatus("c0ffee", Accepted)
	calls, _ := os.ReadFile(log)
	if yes || !errors.Is(err, ErrUnreachable) || len(calls) != 0 {
		t.Errorf("%v, %v; calls:\n%s", yes, err, calls)
	}
}
