package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A hook to the project's trigger URL that GitLab disabled, or that starts
// pipelines on another branch, is said by setup with where to mend it;
// nothing is created nor changed, its token masked (ADR-0059, decision 8).
func TestSetIssueHookSaysABrokenHook(t *testing.T) {
	for _, c := range []struct {
		name, url, alert, want string
	}{
		{"disabled", "https://x/api/v4/projects/1/ref/main/trigger/pipeline?token={token}&variables[WORKLINE_TASK]=issue", "disabled", "GitLab disabled it"},
		{"other branch", "https://x/api/v4/projects/1/ref/dev/trigger/pipeline?token={token}&variables[WORKLINE_TASK]=issue", "executable", `starts pipelines on "dev", not on the default branch main`},
	} {
		t.Run(c.name, func(t *testing.T) {
			var writes []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					writes = append(writes, r.Method+" "+r.URL.Path)
				}
				switch strings.TrimPrefix(r.URL.Path, "/api/v4/projects/1") {
				case "":
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "path_with_namespace": "g/p", "default_branch": "main"})
				case "/hooks":
					_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 4, "url": c.url, "issues_events": true, "alert_status": c.alert}})
				default:
					http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
				}
			}))
			defer srv.Close()
			t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
			t.Setenv("CI_PROJECT_ID", "1")
			t.Setenv("GITLAB_TOKEN", "secret")
			did, err := (&gitlab{repo: t.TempDir()}).SetIssueHook()
			if err != nil || len(did) != 1 || !strings.Contains(did[0], c.want) || !strings.Contains(did[0], "Settings → Webhooks") {
				t.Fatalf("said %q, %v; want %q", did, err, c.want)
			}
			if len(writes) > 0 {
				t.Fatalf("wrote %q, want nothing", writes)
			}
		})
	}
}
