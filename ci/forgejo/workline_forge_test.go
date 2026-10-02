// Package forgejo tests the sample forge command against a mock of the
// Forgejo and Gitea API: what the script asks of it, request by request.
package forgejo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
)

// mock answers the API calls the label operation makes, and records them.
type mock struct {
	mu    sync.Mutex
	calls []string // "METHOD path body"
}

func (m *mock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	m.calls = append(m.calls, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(body)))
	m.mu.Unlock()
	const repo = "/api/v1/repos/owner/name"
	switch {
	case r.Method == "GET" && r.URL.Path == repo+"/labels":
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `[{"id": 1, "name": "needs review"}]`)
		} else {
			fmt.Fprint(w, `[]`)
		}
	case r.Method == "POST" && r.URL.Path == repo+"/labels":
		fmt.Fprint(w, `{"id": 2}`)
	case r.Method == "POST" && r.URL.Path == repo+"/issues/3/labels":
		fmt.Fprint(w, `[]`)
	case r.Method == "GET" && r.URL.Path == repo+"/issues/3/labels":
		fmt.Fprint(w, `[{"id": 5, "name": "workline: step 0"}, {"id": 6, "name": "workline:"}]`)
	case r.Method == "DELETE" && r.URL.Path == repo+"/issues/3/labels/5":
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "not mocked", http.StatusNotFound)
	}
}

// Label names holding spaces are kept whole: added, created when missing,
// and removed by their full name.
func TestLabelsWithSpaces(t *testing.T) {
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	m := &mock{}
	srv := httptest.NewServer(m)
	defer srv.Close()

	req, _ := json.Marshal(map[string]any{"target": map[string]any{"kind": "issue", "id": 3},
		"add": []string{"needs review", "to refine"}, "remove": []string{"workline: step 0"}})
	cmd := exec.Command("sh", "workline-forge.sh")
	cmd.Stdin = strings.NewReader(string(req))
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=label",
		"FORGEJO_URL=" + srv.URL, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("label: %v\n%s\ncalls: %q", err, out, m.calls)
	}
	if strings.TrimSpace(string(out)) != "{}" {
		t.Errorf("answer = %q, want {}", out)
	}
	for _, want := range []string{
		`POST /api/v1/repos/owner/name/labels {"name":"to refine","color":"#ededed"}`,
		`POST /api/v1/repos/owner/name/issues/3/labels {"labels":[1,2]}`,
		`DELETE /api/v1/repos/owner/name/issues/3/labels/5`,
	} {
		if !slices.Contains(m.calls, want) {
			t.Errorf("no call %q; calls: %q", want, m.calls)
		}
	}
	if slices.Contains(m.calls, `DELETE /api/v1/repos/owner/name/issues/3/labels/6`) {
		t.Errorf("a label named by a part of another's name was removed; calls: %q", m.calls)
	}
}

func lookPath() string {
	var dirs []string
	for _, tool := range []string{"sh", "curl", "jq", "git", "cat", "sed"} {
		if p, err := exec.LookPath(tool); err == nil {
			dirs = append(dirs, p[:strings.LastIndex(p, "/")])
		}
	}
	return strings.Join(slices.Compact(slices.Sorted(slices.Values(dirs))), ":")
}
