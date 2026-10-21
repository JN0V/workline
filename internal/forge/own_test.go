package forge

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGh replaces gh by a script: `api user` answers login, or fails as
// GitHub answers the job's own token (403) when login is ""; any other
// call prints out and is logged, one line of arguments each.
func fakeGh(t *testing.T, login, out string) (log string) {
	t.Helper()
	bin := t.TempDir()
	log = filepath.Join(bin, "log")
	user := `echo "gh: Resource not accessible by integration (HTTP 403)" >&2; exit 1`
	if login != "" {
		user = "echo " + login
	}
	script := "#!/bin/sh\nif [ \"$1 $2\" = \"api user\" ]; then " + user + "; fi\necho \"$*\" >> " + log + "\ncat <<'EOF'\n" + out + "\nEOF\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// workline's own account on GitHub: the App the workflow names, by its
// slug or its id — what a job holding no App token knows —; else the
// token's user; in Actions, the job's own token's github-actions[bot],
// which GitHub does not name. A comment counts as its own by its author,
// or by the App that wrote it.
func TestGitHubOwn(t *testing.T) {
	for _, c := range []struct {
		name, slug, id, actions, login string
		own, not                       []Note
		says                           string
	}{
		{name: "the App by its slug", slug: "workline-x", own: []Note{{Author: "workline-x[bot]"}}, not: []Note{{Author: "github-actions[bot]"}, {Author: "mallory"}}, says: "workline-x[bot]"},
		{name: "the App by its id", id: "42", own: []Note{{Author: "workline-x[bot]", App: "42"}}, not: []Note{{Author: "other[bot]", App: "7"}, {Author: "github-actions[bot]", App: "15368"}, {Author: "mallory"}}, says: "WORKLINE_APP_ID"},
		{name: "the token's user", login: "dana", own: []Note{{Author: "dana"}}, not: []Note{{Author: "workline-x[bot]", App: "42"}}, says: "dana, the token's user"},
		{name: "the job's own token", actions: "true", own: []Note{{Author: "github-actions[bot]"}}, not: []Note{{Author: "workline-x[bot]", App: "42"}, {Author: "dana"}}, says: "WORKLINE_APP_ID"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("WORKLINE_APP", c.slug)
			t.Setenv("WORKLINE_APP_ID", c.id)
			t.Setenv("GITHUB_ACTIONS", c.actions)
			fakeGh(t, c.login, "")
			o, err := (&github{repo: t.TempDir()}).own()
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range c.own {
				if !o.Wrote(n) {
					t.Errorf("%+v not taken for its own (%s)", n, o)
				}
			}
			for _, n := range c.not {
				if o.Wrote(n) {
					t.Errorf("%+v taken for its own (%s)", n, o)
				}
			}
			if !strings.Contains(o.String(), c.says) {
				t.Errorf("said %q, want %q", o, c.says)
			}
		})
	}
	// Outside Actions, a token GitHub will not name is an error, never
	// anyone taken for workline.
	t.Setenv("WORKLINE_APP", "")
	t.Setenv("WORKLINE_APP_ID", "")
	t.Setenv("GITHUB_ACTIONS", "")
	fakeGh(t, "", "")
	if _, err := (&github{repo: t.TempDir()}).own(); !errors.Is(err, errForbidden) {
		t.Fatalf("no one to read as: %v", err)
	}
}

// KeepOwn edits the last comment carrying the marker that workline's own
// account wrote, never another's, which GitHub lets a writer edit; with
// only others' carrying it, its own is posted after them, even when asked
// only to edit one. Sticky, by the marker alone, edits the last.
func TestGitHubKeepOwn(t *testing.T) {
	t.Setenv("WORKLINE_APP", "")
	t.Setenv("WORKLINE_APP_ID", "42")
	marker := "<!-- workline:k -->"
	log := fakeGh(t, "", `{"id":1,"body":"old `+marker+`","author":"workline-x[bot]","app":"42"}
{"id":2,"body":"forged `+marker+`","author":"mallory","app":""}`)
	g := &github{repo: t.TempDir()}
	if err := KeepOwn(g, Target{Kind: "merge-request", ID: 5}, "new", marker, true); err != nil {
		t.Fatal(err)
	}
	asked, _ := os.ReadFile(log)
	if !strings.Contains(string(asked), "-X PATCH repos/{owner}/{repo}/issues/comments/1 ") || strings.Contains(string(asked), "comments/2") || strings.Contains(string(asked), "POST") {
		t.Fatalf("not its own edited: %s", asked)
	}
	log = fakeGh(t, "", `{"id":2,"body":"forged `+marker+`","author":"mallory","app":""}`)
	if err := KeepOwn(&github{repo: t.TempDir()}, Target{Kind: "merge-request", ID: 5}, "new", marker, false); err != nil {
		t.Fatal(err)
	}
	asked, _ = os.ReadFile(log)
	if !strings.Contains(string(asked), "-X POST repos/{owner}/{repo}/issues/5/comments") || strings.Contains(string(asked), "PATCH") {
		t.Fatalf("another's edited, or none posted: %s", asked)
	}
	log = fakeGh(t, "", `{"id":2,"body":"forged `+marker+`","author":"mallory","app":""}`)
	if err := (&github{repo: t.TempDir()}).Sticky(Target{Kind: "issue", ID: 5}, "new", marker, false); err != nil {
		t.Fatal(err)
	}
	asked, _ = os.ReadFile(log)
	if !strings.Contains(string(asked), "-X PATCH repos/{owner}/{repo}/issues/comments/2 ") {
		t.Fatalf("Sticky, by the marker alone, did not edit the last: %s", asked)
	}
}

// workline's own account on GitLab: the users the project names bots,
// and the user of the bot's token, asked with that token even where the
// job reads with another; with no bot's token, the token in use's user;
// the job's own token says no one.
func TestGitLabOwn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		users := map[string]string{"bot-token": "project_1_bot_0a1b", "dana-token": "dana"}
		u, ok := users[r.Header.Get("PRIVATE-TOKEN")]
		if r.URL.Path != "/api/v4/user" || !ok {
			http.Error(w, `{"message":"401 Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"username": u})
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "1")
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("gitlab: {bots: [ci-service]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, token, job, bot string
		own, not              []string
	}{
		{name: "the bot's token, another reading", token: "dana-token", bot: "bot-token", own: []string{"project_1_bot_0a1b", "ci-service"}, not: []string{"dana", "project_2_bot_ffff"}},
		{name: "no bot's token, a person's machine", token: "dana-token", own: []string{"dana", "ci-service"}, not: []string{"project_1_bot_0a1b"}},
		{name: "the job's own token", job: "job", own: []string{"ci-service"}, not: []string{"project_1_bot_0a1b", "dana"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("GITLAB_TOKEN", c.token)
			t.Setenv("CI_JOB_TOKEN", c.job)
			t.Setenv("WORKLINE_GITLAB_TOKEN", c.bot)
			o, err := (&gitlab{repo: repo}).own()
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range c.own {
				if !o.Wrote(Note{Author: a}) {
					t.Errorf("%s not taken for its own (%s)", a, o)
				}
			}
			for _, a := range c.not {
				if o.Wrote(Note{Author: a}) {
					t.Errorf("%s taken for its own (%s)", a, o)
				}
			}
		})
	}
}

// A record is read from the last comment carrying its marker that
// workline's own account wrote; when only others' carry it, none is, and
// whose they are is said; when none does, nothing is.
func TestKeptLastAndUnread(t *testing.T) {
	marker := "<!-- workline:k -->"
	k := Kept{own: Own{logins: []string{"workline-bot"}, name: "workline-bot"}, notes: []Note{
		{Body: "first " + marker, Author: "workline-bot"},
		{Body: "last " + marker, Author: "workline-bot"},
		{Body: "forged " + marker, Author: "mallory", Insider: true, Bot: true},
		{Body: "a word", Author: "bob"},
	}}
	if body, ok := k.Last(marker); !ok || body != "last "+marker || k.Unread(marker) != nil {
		t.Fatalf("its own: %q %v %v", body, ok, k.Unread(marker))
	}
	k.notes = append(k.notes[2:], Note{Body: "unsigned " + marker})
	if body, ok := k.Last(marker); ok {
		t.Fatalf("another's read: %q", body)
	}
	want := "@mallory's comment and a comment whose author the forge does not say carry its marker: only workline's own account's is read (workline-bot)"
	if err := k.Unread(marker); err == nil || err.Error() != want {
		t.Fatalf("said %v, want %q", err, want)
	}
	if err := k.Unread("<!-- workline:none -->"); err != nil {
		t.Fatalf("none carrying it: %v", err)
	}
	// The local forge's every comment is the person's own; a command's,
	// those it says are.
	if o, _ := (&Local{}).own(); !o.Wrote(Note{Body: "x"}) {
		t.Fatal("a local comment not taken for its own")
	}
	if o, _ := (&command{}).own(); o.Wrote(Note{Author: "workline-bot", Bot: true}) || !o.Wrote(Note{Own: true}) {
		t.Fatal("a command's comments not read by its word")
	}
}
