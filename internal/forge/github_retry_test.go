package forge

import (
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// githubAt is a GitHub client whose gh, the real one, talks to srv: a
// fake GitHub Enterprise over TLS, its certificate trusted, any token.
// Skipped where gh is not installed.
func githubAt(t *testing.T, srv *httptest.Server) *github {
	t.Helper()
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh is not installed")
	}
	ca := filepath.Join(t.TempDir(), "ca.pem")
	f, err := os.Create(ca)
	if err != nil {
		t.Fatal(err)
	}
	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	f.Close()
	host := srv.Listener.Addr().String()
	t.Setenv("GH_HOST", host)
	t.Setenv("GH_REPO", host+"/o/r")
	t.Setenv("GH_ENTERPRISE_TOKEN", "secret")
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_NO_UPDATE_NOTIFIER", "1")
	t.Setenv("GH_PROMPT_DISABLED", "1")
	t.Setenv("SSL_CERT_FILE", ca)
	return &github{repo: t.TempDir()}
}

// fakeGitHub answers each call with the next of its answers, the last one
// for good; it counts the calls and when each came.
type fakeGitHub struct {
	mu      sync.Mutex
	answers []func(w http.ResponseWriter, r *http.Request)
	calls   []string
	at      []time.Time
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	n := len(f.calls)
	f.calls = append(f.calls, r.Method+" "+r.URL.RequestURI())
	f.at = append(f.at, time.Now())
	a := f.answers[min(n, len(f.answers)-1)]
	f.mu.Unlock()
	a(w, r)
}

func (f *fakeGitHub) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func status(code int, header map[string]string, body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		for k, v := range header {
			w.Header().Set(k, v)
		}
		w.WriteHeader(code)
		fmt.Fprint(w, body)
	}
}

// dropped closes the connection with no answer, as a proxy that fails.
func dropped(w http.ResponseWriter, r *http.Request) {
	c, _, err := w.(http.Hijacker).Hijack()
	if err == nil {
		c.Close()
	}
}

const issueOne = `{"number":1,"title":"Rows lost","body":"","state":"open","labels":[],"user":{"login":"mia"}}`

// A read GitHub fails for a moment is tried again, and read (#491): a
// 502, a connection dropped, its secondary rate limit (403) and a 429,
// each once; the rate's after the time GitHub asks, not sooner.
func TestGitHubReadTriedAgain(t *testing.T) {
	quick(t)
	secondary := `{"message":"You have exceeded a secondary rate limit. Please wait a few minutes before you try again."}`
	for _, c := range []struct {
		name  string
		first func(http.ResponseWriter, *http.Request)
		wait  time.Duration
	}{
		{"502", status(502, nil, `{"message":"Server Error"}`), 0},
		{"502 not JSON", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(502)
			fmt.Fprint(w, "<html>Unicorn</html>")
		}, 0},
		{"dropped", dropped, 0},
		{"secondary rate limit", status(403, map[string]string{"Retry-After": "1"}, secondary), time.Second},
		{"429, none left", func(w http.ResponseWriter, r *http.Request) {
			reset := map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": fmt.Sprint(time.Now().Add(2 * time.Second).Unix())}
			status(429, reset, `{"message":"API rate limit exceeded"}`)(w, r)
		}, time.Second},
	} {
		t.Run(c.name, func(t *testing.T) {
			fake := &fakeGitHub{answers: []func(http.ResponseWriter, *http.Request){c.first, status(200, nil, issueOne)}}
			srv := httptest.NewTLSServer(fake)
			defer srv.Close()
			is, err := githubAt(t, srv).Issue(1)
			if err != nil {
				t.Fatalf("%v, calls %q", err, fake.calls)
			}
			if is.Title != "Rows lost" || fake.count() != 2 {
				t.Errorf("issue %+v, calls %q; want it read on the second call", is, fake.calls)
			}
			if got := fake.at[1].Sub(fake.at[0]); got < c.wait {
				t.Errorf("tried again after %v, GitHub asked %v", got, c.wait)
			}
		})
	}
}

// A write GitHub answered with a 5xx may have been taken: it is not sent
// again, and its failure says so. One it refused for its rate, or that
// never reached it, is sent again.
func TestGitHubWriteTriedAgainOnlyUnsent(t *testing.T) {
	quick(t)
	fake := &fakeGitHub{answers: []func(http.ResponseWriter, *http.Request){status(502, nil, `{"message":"Server Error"}`), status(201, nil, `{"id":1}`)}}
	srv := httptest.NewTLSServer(fake)
	defer srv.Close()
	err := githubAt(t, srv).SetBody(1, "Rows lost on export.")
	if !errors.Is(err, ErrUnreachable) || !strings.Contains(err.Error(), "(HTTP 502) (a write: not sent again, GitHub may have taken it)") || fake.count() != 1 {
		t.Errorf("a write answered 502: %v, %d calls; want 1 call, said", err, fake.count())
	}

	fake = &fakeGitHub{answers: []func(http.ResponseWriter, *http.Request){status(429, map[string]string{"Retry-After": "1"}, `{"message":"slow down"}`), status(201, nil, `{"id":1}`)}}
	srv2 := httptest.NewTLSServer(fake)
	defer srv2.Close()
	if err := githubAt(t, srv2).SetBody(1, "Rows lost on export."); err != nil || fake.count() != 2 {
		t.Errorf("a write refused for the rate: %v, %d calls; want sent again", err, fake.count())
	}
}

// A read GitHub never answers fails after a bounded time, said; it never
// hangs (#491).
func TestGitHubNoAnswerBounded(t *testing.T) {
	quick(t)
	retry.answerMax = 300 * time.Millisecond
	done := make(chan struct{})
	fake := &fakeGitHub{answers: []func(http.ResponseWriter, *http.Request){func(w http.ResponseWriter, r *http.Request) { <-done }}}
	srv := httptest.NewTLSServer(fake)
	defer srv.Close()
	defer close(done)
	g := githubAt(t, srv)
	start := time.Now()
	_, err := g.Issue(1)
	if !errors.Is(err, ErrUnreachable) || !strings.Contains(err.Error(), "no answer within") || !strings.Contains(err.Error(), "(tried 3 times)") || fake.count() != 3 {
		t.Errorf("a read never answered: %v, %d calls; want 3 calls, said", err, fake.count())
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("gave up after %v", took)
	}
}

// A read of many pages GitHub fails on the second is read again whole;
// the headers gh prints before each page never reach what is read, nor a
// body that looks like them.
func TestGitHubPagesTriedAgain(t *testing.T) {
	quick(t)
	var srv *httptest.Server
	page2 := 0
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") != "2" {
			w.Header().Set("Link", "<"+srv.URL+"/api/v3/repos/o/r/labels?per_page=100&page=2>; rel=\"next\"")
			fmt.Fprint(w, `[{"name":"bug"}]`)
			return
		}
		if page2++; page2 == 1 {
			w.WriteHeader(502)
			fmt.Fprint(w, `{"message":"Server Error"}`)
			return
		}
		fmt.Fprint(w, `[{"name":"HTTP/1.1 200 OK"}]`)
	}))
	defer srv.Close()
	names, err := githubAt(t, srv).projectLabels()
	if err != nil || fmt.Sprint(names) != "[bug HTTP/1.1 200 OK]" || page2 != 2 {
		t.Errorf("labels %q, %v, page 2 asked %d times; want both pages, the second asked twice", names, err, page2)
	}
}

// What gh is asked says whether a call only reads: a GET, said or not,
// without a field (gh sends a POST then); a GraphQL query.
func TestGitHubRead(t *testing.T) {
	for _, c := range []struct {
		args []string
		read bool
	}{
		{[]string{"api", "repos/{owner}/{repo}/issues/9"}, true},
		{[]string{"api", "--paginate", "repos/{owner}/{repo}/labels", "--jq", "graphql"}, true},
		{[]string{"api", "-X", "GET", "search/issues", "-f", "q=is:open"}, true},
		{[]string{"api", "--method=get", "search/issues", "-f", "q=is:open"}, true},
		{[]string{"api", "graphql", "--paginate", "-f", "query=query($n: Int!) { viewer { login } }", "-F", "n=1"}, true},
		{[]string{"api", "graphql", "-f", "query={ viewer { login } }"}, true},
		{[]string{"api", "graphql", "-f", "query=mutation($id: ID!) { disablePullRequestAutoMerge(input: {pullRequestId: $id}) { clientMutationId } }"}, false},
		{[]string{"api", "repos/{owner}/{repo}/issues/9/comments", "-f", "body=hi"}, false},
		{[]string{"api", "-X", "POST", "repos/{owner}/{repo}/check-runs", "--input", "-"}, false},
		{[]string{"api", "-X", "DELETE", "repos/{owner}/{repo}/issues/comments/1"}, false},
		{[]string{"api", "-XPATCH", "repos/{owner}/{repo}/issues/9"}, false},
	} {
		if got := ghRead(c.args); got != c.read {
			t.Errorf("%q: read %v, want %v", c.args, got, c.read)
		}
	}
}

// How gh says a call failed: Go's words for a request with no answer,
// gh's for a host it could not reach, an HTTP status.
func TestGitHubFailure(t *testing.T) {
	for _, c := range []struct {
		msg  string
		code int
		want failure
	}{
		{`Get "https://api.github.com/repos/o/r": dial tcp 140.82.121.6:443: connect: connection refused`, 0, failure{cut: true, unsent: true}},
		{`Post "https://api.github.com/graphql": net/http: TLS handshake timeout`, 0, failure{cut: true, unsent: true}},
		{`Get "https://api.github.com/repos/o/r": read tcp 10.0.0.2:5050->140.82.121.6:443: read: connection reset by peer`, 0, failure{cut: true}},
		{`Post "https://api.github.com/graphql": EOF`, 0, failure{cut: true}},
		{`Get "https://ghe.example/api/v3/repos/o/r": tls: failed to verify certificate: x509: certificate signed by unknown authority`, 0, failure{cut: true, lasting: true}},
		{"error connecting to api.github.com\ncheck your internet connection or https://githubstatus.com", 0, failure{cut: true, unsent: true}},
		{"gh: HTTP 502", 0, failure{code: 502}},
		{"gh: Server Error (HTTP 503)", 503, failure{code: 503}},
		{"gh: API rate limit exceeded for user ID 1. (HTTP 403)", 403, failure{code: 403, limited: true, least: time.Minute}},
		{"gh: Resource not accessible by integration (HTTP 403)", 403, failure{code: 403}},
		{"gh: error evaluating jq: unexpected EOF", 0, failure{}},
	} {
		got := ghFailure(false, c.msg, c.code, nil)
		got.header = nil
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%q: %+v, want %+v", c.msg, got, c.want)
		}
	}
}

// GitHub's rate: a minute waited when it asks nothing (its docs), the
// reset of a limit spent when it is near; a later one is not waited for.
func TestGitHubRateWaits(t *testing.T) {
	quick(t)
	if wait, note := again("GitHub", true, 1, failure{code: 403, limited: true, least: time.Minute, header: http.Header{}}); wait != time.Minute || note != "" {
		t.Errorf("a limit with no wait asked: %v %q, want a minute", wait, note)
	}
	spent := http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {fmt.Sprint(time.Now().Add(time.Hour).Unix())}}
	if wait, note := again("GitHub", true, 1, failure{code: 403, limited: true, least: time.Minute, header: spent}); wait >= 0 ||
		!strings.Contains(note, "GitHub asks to wait (X-RateLimit-Reset: in 3600 s), longer than workline waits (60 s): not tried again") {
		t.Errorf("a limit reset in an hour: %v %q, want not tried again, said", wait, note)
	}
	if wait, note := again("GitHub", false, 1, failure{code: 502, header: spent}); wait >= 0 || note != "a write: not sent again, GitHub may have taken it" {
		t.Errorf("a write answered 502: %v %q", wait, note)
	}
}
