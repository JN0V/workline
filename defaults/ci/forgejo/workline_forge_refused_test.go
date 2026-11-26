package forgejo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

// tryDown runs the script on one operation against a forge that
// answers nothing: a server that was listening, then closed.
func tryDown(t *testing.T, op string, req any) (string, error) {
	t.Helper()
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	in, _ := json.Marshal(req)
	cmd := exec.Command("sh", "workline-forge.sh")
	cmd.Stdin = strings.NewReader(string(in))
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=" + op,
		"FORGEJO_URL=" + url, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Forge down: Given the forge's URL answers nothing, When a read runs
// (the one `workline follow` makes), Then the script exits other than 0,
// which the engine says as the forge unreachable, and curl's error is
// on its output.
func TestForgeDownStopsReads(t *testing.T) {
	for _, op := range []string{"all-issues", "merge-requests", "open-merge-requests"} {
		req := map[string]any{"prefix": "workline/"}
		out, err := tryDown(t, op, req)
		if err == nil {
			t.Errorf("%s: a forge down was read as empty: exit 0, answer %q", op, out)
		}
		if !strings.Contains(out, "curl:") {
			t.Errorf("%s: curl's error not on the output: %q", op, out)
		}
	}
}

// Issue not found: Given issue 99 does not exist, When the script reads
// the issue, Then it exits other than 0, never an empty answer taken for
// an issue with an empty Need.
func TestIssueNotFoundFails(t *testing.T) {
	out, err := try(t, &mock{}, "issue", map[string]any{"id": 99})
	if err == nil {
		t.Errorf("an issue not found was read as an answer: exit 0, %q", out)
	}
	if strings.Contains(out, `"title"`) {
		t.Errorf("an issue answered for an id not found: %q", out)
	}
}

// Label refused: Given the forge is down, When a role labels an issue,
// Then the act fails, whether it adds or removes a label.
func TestLabelFailsWhenForgeDown(t *testing.T) {
	for name, req := range map[string]map[string]any{
		"add":    {"target": map[string]any{"kind": "issue", "id": 3}, "add": []string{"needs review"}},
		"remove": {"target": map[string]any{"kind": "issue", "id": 3}, "remove": []string{"workline: step 0"}},
	} {
		if out, err := tryDown(t, "label", req); err == nil {
			t.Errorf("label %s with the forge down passed: %q", name, out)
		}
	}
}
