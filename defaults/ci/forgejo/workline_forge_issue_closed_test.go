package forgejo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

// readOneIssue runs the `issue` operation on issue 7 against a Forgejo
// API that holds it in the state given, and returns the answer as a map.
func readOneIssue(t *testing.T, state string) map[string]any {
	t.Helper()
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/repos/owner/name/issues/7" {
			_, _ = fmt.Fprintf(w, `{"number": 7, "title": "Rows lost", "body": "Need.", "state": %q, "labels": [{"id": 1, "name": "bug"}]}`, state)
			return
		}
		http.Error(w, "not mocked", http.StatusNotFound)
	}))
	defer srv.Close()
	cmd := exec.Command("sh", "workline-forge.sh")
	cmd.Stdin = strings.NewReader(`{"id": 7}`)
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=issue",
		"FORGEJO_URL=" + srv.URL, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("issue: %v\n%s", err, out)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &got); err != nil {
		t.Fatalf("answer %q: %v", out, err)
	}
	return got
}

// Closed issue: Given a closed issue on Forgejo, When the line routes the
// issue event on it, Then the issue is read as closed (`closed` true),
// which the line says as "is closed" and wakes no role.
func TestIssueClosedReadAsClosed(t *testing.T) {
	got := readOneIssue(t, "closed")
	if got["closed"] != true {
		t.Errorf("a closed issue read with closed = %v, want true; answer %v", got["closed"], got)
	}
	if got["id"] != float64(7) || got["title"] != "Rows lost" {
		t.Errorf("the issue's other fields changed: %v", got)
	}
}

// Open issue: Given an open issue on Forgejo, When it is read, Then
// `closed` is false, and present in the answer.
func TestIssueOpenReadAsOpen(t *testing.T) {
	got := readOneIssue(t, "open")
	closed, present := got["closed"]
	if !present || closed != false {
		t.Errorf("an open issue read with closed = %v (present %v), want false; answer %v", closed, present, got)
	}
}
