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

// ping runs the script on the `ping` operation, as the doctor does, against
// the forge at url; its output and its exit are returned.
func ping(t *testing.T, url string) (string, error) {
	t.Helper()
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	cmd := exec.Command("sh", "workline-forge.sh")
	cmd.Stdin = strings.NewReader(`{"operation":"ping"}`)
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=ping",
		"FORGEJO_URL=" + url, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// pingRefused says whether the script, asked to ping, said the forge is
// not reachable: it exited other than 0, or answered an error that is not
// the unknown-operation one.
func pingRefused(out string, err error) (string, bool) {
	if err != nil {
		return out, true
	}
	var got struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(out), &got) != nil || got.Error == "" {
		return out, false
	}
	return got.Error, !strings.Contains(got.Error, "unknown operation")
}

// Forge down: given a URL nothing listens on, when the script is asked to
// ping, then it says the forge is not reachable: an exit other than 0, or
// an error answered, never `{}` nor "unknown operation".
func TestPingForgeDown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	out, err := ping(t, url)
	if said, ok := pingRefused(out, err); !ok {
		t.Errorf("a forge nobody listens on was not said unreachable: %q (exit %v)", said, err)
	}
}

// Bad token: given a token the forge refuses, when the script is asked to
// ping, then it says the token is refused.
func TestPingBadToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message": "token is invalid"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()
	out, err := ping(t, srv.URL)
	said, ok := pingRefused(out, err)
	if !ok {
		t.Fatalf("a token refused was not said: %q (exit %v)", said, err)
	}
	if !strings.Contains(strings.ToLower(out), "token") {
		t.Errorf("the refusal does not name the token: %q", out)
	}
}

// All set: given a live Forgejo and a valid token, when the script is asked
// to ping, then it answers {}: the forge is reachable.
func TestPingAllSet(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/user":
			_, _ = fmt.Fprint(w, `{"id": 11, "login": "ci"}`)
		case "/api/v1/repos/owner/name":
			_, _ = fmt.Fprint(w, `{"id": 1, "full_name": "owner/name"}`)
		default:
			http.Error(w, "not mocked", http.StatusNotFound)
		}
	}))
	defer srv.Close()
	out, err := ping(t, srv.URL)
	if err != nil || out != "{}" {
		t.Errorf("ping = %q (exit %v), want {}; asked %q", out, err, asked)
	}
}
