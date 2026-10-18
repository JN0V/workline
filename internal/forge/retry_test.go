package forge

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// quick makes the pauses short for a test, then puts them back.
func quick(t *testing.T) {
	saved := retry
	retry.pause = time.Millisecond
	t.Cleanup(func() { retry = saved })
}

// gitlabAt is a GitLab client on the API at base.
func gitlabAt(t *testing.T, base string) *gitlab {
	t.Setenv("CI_API_V4_URL", base+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "1")
	t.Setenv("GITLAB_TOKEN", "secret")
	return &gitlab{repo: t.TempDir()}
}

// What the conformance cases cannot wait for: a Retry-After longer than
// workline waits is not waited for, said; a 501 is no moment's failure.
func TestRetryNotWaited(t *testing.T) {
	quick(t)
	for _, c := range []struct {
		status int
		after  string
		says   string
	}{
		{429, "3600", "GitLab asks to wait 3600 s, longer than workline waits (60 s): not tried again"},
		{501, "", "501 Not Implemented"},
	} {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if c.after != "" {
				w.Header().Set("Retry-After", c.after)
			}
			w.WriteHeader(c.status)
		}))
		_, err := gitlabAt(t, srv.URL).api("projects/:id/issues")
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), c.says) || strings.Contains(err.Error(), "tried 3 times") || calls != 1 {
			t.Errorf("%d: %d calls, %v; want 1 call, %q", c.status, calls, err, c.says)
		}
	}
}

// A write that could not be sent — no connection made — never reached
// GitLab: it is sent again, bounded.
func TestRetryUnsentWrite(t *testing.T) {
	quick(t)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close() // nothing listens there now: the connection is refused
	_, err = gitlabAt(t, "http://"+addr).api("-X", "POST", "projects/:id/issues/9/notes", "-f", "body=hello")
	if err == nil || !strings.Contains(err.Error(), "(tried 3 times)") {
		t.Errorf("a refused connection: %v, want tried 3 times", err)
	}
}

// Retry-After is seconds, or an HTTP date.
func TestRetryAfter(t *testing.T) {
	if d, ok := retryAfter("7"); !ok || d != 7*time.Second {
		t.Errorf("seconds: %v %v", d, ok)
	}
	if d, ok := retryAfter(time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)); !ok || d < 28*time.Second || d > 31*time.Second {
		t.Errorf("a date: %v %v", d, ok)
	}
	if _, ok := retryAfter("soon"); ok {
		t.Error("an unreadable value is read")
	}
}
