package forge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGH puts a gh on PATH that logs each call's arguments to the file
// it returns and runs script.
func fakeGH(t *testing.T, script string) string {
	t.Helper()
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls")
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\necho \"$*\" >> "+calls+"\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return calls
}

func callsIn(t *testing.T, file string) []string {
	t.Helper()
	data, _ := os.ReadFile(file)
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// The runs going are one listing of the last day's runs, every workflow,
// those not completed kept, requested too (#1761, #1762): one call, and
// the next page only when its Link says one follows.
func TestGitHubRunsGoing(t *testing.T) {
	calls := fakeGH(t, `case "$2" in
*page=1\&*) printf 'HTTP/2.0 200 OK\r\nLink: <https://api.github.com/x?page=2>; rel="next"\r\n\r\n'
  echo '{"name":"workline gardening backlog","workflow":"workline-gardening.yml","status":"requested","started":"2026-11-16T02:15:00Z"}' ;;
*page=2\&*) printf 'HTTP/2.0 200 OK\r\n\r\n'
  echo '{"name":"workline merge","workflow":"workline-merge.yml","status":"in_progress","started":"2026-11-16T02:00:00Z"}' ;;
esac
`)
	runs, err := (&github{repo: t.TempDir()}).RunsGoing()
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 || runs[0].Status != "requested" || runs[1].Name != "workline merge" {
		t.Errorf("runs going %+v: want the requested one and the next page's", runs)
	}
	sent := callsIn(t, calls)
	if len(sent) != 2 || !strings.Contains(sent[0], "actions/runs?per_page=100&page=1&created=%3E%3D") || !strings.Contains(sent[0], `select(.status != "completed")`) {
		t.Errorf("calls %q: want one listing of every workflow's runs a page, the completed left out", sent)
	}
}

// A listing read again sends its last ETag; GitHub's 304 Not Modified
// costs no quota, and the answer kept, in the clone's git folder from one
// command to the next, is used (#1762).
func TestGitHubListingsReadConditionally(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	calls := fakeGH(t, `case "$*" in
*'If-None-Match: W/"e1"'*) printf 'HTTP/2.0 304 Not Modified\r\nEtag: W/"e1"\r\n\r\n'; echo "unexpected end of JSON input" >&2; exit 1 ;;
*) printf 'HTTP/2.0 200 OK\r\nEtag: W/"e1"\r\n\r\n'
   echo '{"name":"workline merge","workflow":"workline-merge.yml","status":"in_progress","started":"2026-11-16T02:00:00Z"}' ;;
esac
`)
	ReadConditionally(true)
	t.Cleanup(func() { ReadConditionally(false); answers.m, answers.repo, answers.path = nil, "", "" })
	answers.m, answers.repo, answers.path = nil, "", ""
	sent0, free0 := Calls()
	g := &github{repo: repo}
	for i := range 2 {
		runs, err := g.RunsGoing()
		if err != nil || len(runs) != 1 || runs[0].Name != "workline merge" {
			t.Fatalf("read %d: %+v, %v: want the run going, from the answer kept the second time", i+1, runs, err)
		}
	}
	// Another command, the answers read back from the clone.
	answers.m, answers.repo, answers.path = nil, "", ""
	if runs, err := g.RunsGoing(); err != nil || len(runs) != 1 {
		t.Fatalf("a next command: %+v, %v: want the answer kept in the clone", runs, err)
	}
	sent, free := Calls()
	if sent-sent0 != 3 || free-free0 != 2 {
		t.Errorf("%d calls, %d free: want 3, the last 2 answered 304", sent-sent0, free-free0)
	}
	if got := callsIn(t, calls); len(got) != 3 || strings.Contains(got[0], "If-None-Match") || !strings.Contains(got[2], `If-None-Match: W/"e1"`) {
		t.Errorf("calls %q: the first with no ETag, the next with the one kept", got)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "workline", "etags.json")); err != nil {
		t.Errorf("the answers are not kept in the clone: %v", err)
	}
}

// A change is read (#1762): after a 304, an answer 200 with a new ETag
// replaces the one kept, and its ETag is sent the next time.
func TestGitHubListingANewAnswerReplacesTheOneKept(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	state := filepath.Join(t.TempDir(), "changed")
	calls := fakeGH(t, `case "$*" in
*'If-None-Match: W/"e2"'*) printf 'HTTP/2.0 304 Not Modified\r\nEtag: W/"e2"\r\n\r\n'; exit 1 ;;
*'If-None-Match: W/"e1"'*) if [ -f `+state+` ]; then
     printf 'HTTP/2.0 200 OK\r\nEtag: W/"e2"\r\n\r\n'
     echo '{"name":"workline gardening docs","workflow":"workline-gardening.yml","status":"queued","started":"2026-11-16T02:30:00Z"}'
   else printf 'HTTP/2.0 304 Not Modified\r\nEtag: W/"e1"\r\n\r\n'; exit 1; fi ;;
*) printf 'HTTP/2.0 200 OK\r\nEtag: W/"e1"\r\n\r\n'
   echo '{"name":"workline merge","workflow":"workline-merge.yml","status":"in_progress","started":"2026-11-16T02:00:00Z"}' ;;
esac
`)
	ReadConditionally(true)
	t.Cleanup(func() { ReadConditionally(false); answers.m, answers.repo, answers.path = nil, "", "" })
	answers.m, answers.repo, answers.path = nil, "", ""
	g := &github{repo: repo}
	names := func() string {
		runs, err := g.RunsGoing()
		if err != nil {
			t.Fatal(err)
		}
		var n []string
		for _, r := range runs {
			n = append(n, r.Name)
		}
		return strings.Join(n, ",")
	}
	if a, b := names(), names(); a != "workline merge" || b != "workline merge" {
		t.Fatalf("read, then 304: %q, %q; want the merge run both times", a, b)
	}
	if err := os.WriteFile(state, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if c, d := names(), names(); c != "workline gardening docs" || d != "workline gardening docs" {
		t.Errorf("a new run, then 304 on its ETag: %q, %q; want the new answer both times", c, d)
	}
	if got := callsIn(t, calls); len(got) != 4 || !strings.Contains(got[3], `If-None-Match: W/"e2"`) {
		t.Errorf("calls %q: the last one with the new ETag", got)
	}
}
