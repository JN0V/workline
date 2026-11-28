package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// On GitLab, the roles' sections are one note of the bot's, edited in
// place; an older note of the bot's is folded in and deleted, a person's
// left; Notes and Comments read each section as the note it was.
func TestGitLabStateComment(t *testing.T) {
	type note struct {
		ID     int               `json:"id"`
		Body   string            `json:"body"`
		Author map[string]string `json:"author"`
	}
	var mu sync.Mutex
	notes := []note{
		{1, "Old record.\n\n" + Marker("sticky=reviewer/acts"), map[string]string{"username": "project_1_bot_0a1b"}},
		{2, "By hand.\n\n" + Marker("sticky=tester/acts"), map[string]string{"username": "dana"}},
	}
	next, posted, deleted := 3, 0, []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/1")
		_ = r.ParseForm()
		switch {
		case p == "/api/v4/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "project_1_bot_0a1b"})
		case r.Method == "GET" && p == "/merge_requests/5/notes":
			_ = json.NewEncoder(w).Encode(notes)
		case r.Method == "GET" && strings.HasPrefix(p, "/members"):
			_ = json.NewEncoder(w).Encode([]any{})
		case r.Method == "POST" && p == "/merge_requests/5/notes":
			notes = append(notes, note{next, r.Form.Get("body"), map[string]string{"username": "project_1_bot_0a1b"}})
			next, posted = next+1, posted+1
			_, _ = w.Write([]byte(`{"id": 1}`))
		case r.Method == "PUT" && strings.HasPrefix(p, "/merge_requests/5/notes/"):
			for i := range notes {
				if strconv.Itoa(notes[i].ID) == strings.TrimPrefix(p, "/merge_requests/5/notes/") {
					notes[i].Body = r.Form.Get("body")
				}
			}
			_, _ = w.Write([]byte(`{"id": 1}`))
		case r.Method == "DELETE" && strings.HasPrefix(p, "/merge_requests/5/notes/"):
			id := strings.TrimPrefix(p, "/merge_requests/5/notes/")
			deleted = append(deleted, id)
			for i := range notes {
				if strconv.Itoa(notes[i].ID) == id {
					notes = append(notes[:i], notes[i+1:]...)
					break
				}
			}
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "1")
	t.Setenv("GITLAB_TOKEN", "bot-token")
	t.Setenv("WORKLINE_GITLAB_TOKEN", "")
	mr := Target{Kind: "merge-request", ID: 5}
	g := &gitlab{repo: t.TempDir()}
	for _, w := range [][2]string{{"developer", "Developer opened it."}, {"tester", "Tests ran."}, {"developer", "Developer fixed it."}} {
		if err := g.Sticky(mr, w[1], Marker("sticky="+w[0]+"/acts"), true); err != nil {
			t.Fatal(err)
		}
	}
	if posted != 1 || len(notes) != 2 || notes[0].ID != 2 || strings.Join(deleted, ",") != "1" {
		t.Fatalf("posted %d, deleted %v, notes %+v: want the person's note left and one of the bot's", posted, deleted, notes)
	}
	body := notes[1].Body
	for _, w := range []string{"Old record.", "Tests ran.", "Developer fixed it.", "### developer", "UTC developer: Developer fixed it."} {
		if !strings.Contains(body, w) {
			t.Errorf("%q not in the state note:\n%s", w, body)
		}
	}
	if secs, _, _ := strings.Cut(body, historyMark); strings.Contains(secs, "Developer opened it.") {
		t.Errorf("the developer's section not replaced:\n%s", body)
	}
	bodies, err := g.Comments(mr)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"By hand.\n\n" + Marker("sticky=tester/acts"), "Old record.\n\n" + Marker("sticky=reviewer/acts"), "Developer fixed it.\n\n" + Marker("sticky=developer/acts"), "Tests ran.\n\n" + Marker("sticky=tester/acts")}
	if strings.Join(bodies, "|") != strings.Join(want, "|") {
		t.Fatalf("read as %q, want %q", bodies, want)
	}
}

// A write reads the comment back: its section lost to another writer
// meanwhile, it writes once more; lost again, it fails.
func TestWriteStateReadsBack(t *testing.T) {
	own := Own{logins: []string{"bot"}, name: "bot"}
	var body string
	lose := 0
	ops := stateOps{
		list: func() ([]Note, error) {
			if body == "" {
				return nil, nil
			}
			return []Note{{ID: "1", Body: body, Author: "bot"}}, nil
		},
		post: func(b string) error { body = b; return nil },
		edit: func(_, b string) error {
			if lose > 0 { // another's write lands over this one
				lose--
				return nil
			}
			body = b
			return nil
		},
		drop: func(string) error { return nil },
	}
	if err := writeState(ops, own, "First.", Marker("sticky=reviewer/acts"), true); err != nil {
		t.Fatal(err)
	}
	lose = 1
	if err := writeState(ops, own, "Second.", Marker("sticky=tester/acts"), true); err != nil || !strings.Contains(body, "Second.") || !strings.Contains(body, "First.") {
		t.Fatalf("a lost write not written again: %v\n%s", err, body)
	}
	lose = 2
	if err := writeState(ops, own, "Third.", Marker("sticky=developer/acts"), true); err == nil || !strings.Contains(err.Error(), "lost") {
		t.Fatalf("lost twice, not said: %v", err)
	}
}

// A section alone past the forge's limit is cut and says so, its marker
// kept: the record is still found.
func TestStateSectionCut(t *testing.T) {
	notes := []Note{}
	p, ok := planState(notes, Own{unsigned: true}, strings.Repeat("x", 70000), Marker("sticky=tester/acts"), true, time.Time{})
	if !ok || len([]rune(p.main)) > stateLimit || !strings.Contains(p.main, "Cut:") || p.over != "" {
		t.Fatalf("%v, %d characters, overflow %d", ok, len([]rune(p.main)), len(p.over))
	}
	if got := splitState([]Note{{Body: p.main}}); len(got) != 1 || !strings.HasSuffix(got[0].Body, Marker("sticky=tester/acts")) {
		t.Fatalf("the cut section not read back: %d", len(got))
	}
}
