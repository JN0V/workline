package forgejo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// openedPull runs open-merge-request for a branch with no pull request
// open yet, and returns the JSON body the script posted to create it.
func openedPull(t *testing.T, draft bool) map[string]any {
	t.Helper()
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	const repo = "/api/v1/repos/owner/name"
	var mu sync.Mutex
	var posted string
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls = append(calls, strings.TrimSpace(r.Method+" "+r.URL.EscapedPath()+" "+string(body)))
		mu.Unlock()
		switch {
		case r.Method == "GET" && r.URL.Path == repo+"/pulls":
			_, _ = fmt.Fprint(w, `[]`)
		case r.Method == "POST" && r.URL.Path == repo+"/pulls":
			mu.Lock()
			posted = string(body)
			mu.Unlock()
			_, _ = fmt.Fprint(w, `{"number": 12}`)
		default:
			http.Error(w, "not mocked", http.StatusNotFound)
		}
	}))
	defer srv.Close()
	in, _ := json.Marshal(map[string]any{"branch": "feat/new", "base": "main",
		"title": "Add the thing", "body": "What it does.", "draft": draft})
	cmd := exec.Command("sh", "workline-forge.sh")
	cmd.Stdin = strings.NewReader(string(in))
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=open-merge-request",
		"FORGEJO_URL=" + srv.URL, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		t.Fatalf("open-merge-request: %v\n%s\ncalls: %q", err, out, calls)
	}
	if got := strings.TrimSpace(string(out)); got != `{"id":12}` {
		t.Errorf("answer = %s, want {\"id\":12}", got)
	}
	var pull map[string]any
	if err := json.Unmarshal([]byte(posted), &pull); err != nil {
		t.Fatalf("no pull request created, or its body unreadable %q: %v; calls: %q", posted, err, calls)
	}
	return pull
}

// Given a pushed branch, when the engine opens a merge request with
// `draft` true, Forgejo and Gitea show it as a draft: its title opens
// with their `WIP:` or `[WIP]` prefix, and still says the title asked.
// With `draft` false, the title is sent as it is, with no such prefix.
func TestOpenMergeRequestDraft(t *testing.T) {
	title := func(p map[string]any) string {
		s, _ := p["title"].(string)
		return s
	}
	isDraft := func(s string) bool {
		s = strings.ToUpper(strings.TrimSpace(s))
		return strings.HasPrefix(s, "WIP:") || strings.HasPrefix(s, "[WIP]")
	}

	d := title(openedPull(t, true))
	if !isDraft(d) {
		t.Errorf("a draft opened with the title %q: no WIP: or [WIP] prefix, so not a draft", d)
	}
	if !strings.Contains(d, "Add the thing") {
		t.Errorf("a draft's title = %q, lost the title asked, %q", d, "Add the thing")
	}

	r := title(openedPull(t, false))
	if r != "Add the thing" {
		t.Errorf("a pull request not a draft opened with the title %q, want %q", r, "Add the thing")
	}
}
