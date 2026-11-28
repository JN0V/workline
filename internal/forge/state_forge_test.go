package forge

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	next, posted, deleted, editor := 3, 0, []string{}, map[int]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/1")
		_ = r.ParseForm()
		switch {
		case r.URL.Path == "/api/graphql":
			var nodes []map[string]any
			for _, n := range notes {
				nd := map[string]any{"id": "gid://gitlab/Note/" + strconv.Itoa(n.ID), "author": n.Author}
				if e := editor[n.ID]; e != "" {
					nd["lastEditedBy"] = map[string]string{"username": e}
				}
				nodes = append(nodes, nd)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"project": map[string]any{"item": map[string]any{"notes": map[string]any{"nodes": nodes, "pageInfo": map[string]any{}}}}}})
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
					delete(editor, notes[i].ID) // the bot edited it last
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
	// A person edits the developer's section, its digest written again:
	// GitLab keeps no versions of a note, so the next write marks every
	// section of a note another edited, never taking theirs as its own.
	old, forged := "Developer fixed it.\n\n"+Marker("sticky=developer/acts"), "Developer says done.\n\n"+Marker("sticky=developer/acts")
	notes[1].Body = strings.Replace(strings.Replace(notes[1].Body, old, forged, 1), digest(old), digest(forged), 1)
	editor[notes[1].ID] = "dana"
	if err := g.Sticky(mr, "Tests ran again.", Marker("sticky=tester/acts"), true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes[1].Body, "<!-- workline:section edited -->\n### developer") {
		t.Errorf("the section a person edited written again as workline's:\n%s", notes[1].Body)
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

// Sections past what two comments hold: neither comment passes the
// forge's limit, the largest is cut and says so, every record kept.
func TestStateOverflowWithinLimit(t *testing.T) {
	own := Own{unsigned: true}
	var notes []Note
	for _, r := range []string{"reviewer", "tester", "developer", "product-owner"} {
		p, ok := planState(notes, own, strings.Repeat("x", 40000), Marker("sticky="+r+"/acts"), true, time.Time{})
		if !ok {
			t.Fatal("nothing written")
		}
		if n := len([]rune(p.main)); n > stateLimit {
			t.Fatalf("the main comment holds %d characters", n)
		}
		if n := len([]rune(p.over)); n > stateLimit {
			t.Fatalf("the overflow holds %d characters", n)
		}
		notes = []Note{{ID: "1", Body: p.main}}
		if p.over != "" {
			notes = append(notes, Note{ID: "2", Body: p.over})
		}
	}
	got := splitState(notes)
	if len(got) != 4 || !strings.Contains(notes[0].Body+notes[1].Body, "Cut:") {
		t.Fatalf("%d records read back, want 4, one cut", len(got))
	}
	for _, n := range got {
		if n.section != intact {
			t.Errorf("a cut section read as altered: %.60q", n.Body)
		}
	}
}

// A section moving to the overflow is written there before the main
// comment drops it; one leaving it stays there until the main comment
// holds it: a write failing between loses none.
func TestStateApplyOrder(t *testing.T) {
	own := Own{unsigned: true}
	big, small, dev := Marker("sticky=tester/acts"), Marker("sticky=reviewer/acts"), Marker("sticky=developer/acts")
	p, _ := planState(nil, own, strings.Repeat("s", 25000), small, true, time.Time{})
	p, _ = planState([]Note{{ID: "1", Body: p.main}}, own, strings.Repeat("d", 24000), dev, true, time.Time{})
	notes := []Note{{ID: "1", Body: p.main}}
	p, _ = planState(notes, own, strings.Repeat("b", 30000), big, true, time.Time{})
	var order []string
	fail := func(stop int) stateOps {
		return stateOps{
			post: func(b string) error { order = append(order, "post:"+b[:len(overMark)]); return nil },
			edit: func(id, b string) error {
				if order = append(order, id); len(order) == stop {
					return errors.New("refused")
				}
				return nil
			},
			drop: func(string) error { return nil },
		}
	}
	if err := p.apply(fail(0)); err != nil || len(order) != 2 || !strings.HasPrefix(order[0], "post:"+overMark) || order[1] != "1" {
		t.Fatalf("written in the order %q, %v: want the overflow first", order, err)
	}
	// The tester's section shrinks: the reviewer's moves to the overflow,
	// the tester's back to the main comment.
	notes = []Note{{ID: "1", Body: p.main}, {ID: "2", Body: p.over}}
	p, _ = planState(notes, own, strings.Repeat("t", 20000), big, true, time.Time{})
	if !strings.Contains(p.over, small) || strings.Contains(p.over, big) {
		t.Fatalf("the reviewer's section not moved to the overflow")
	}
	if p.stage == "" || !strings.Contains(p.stage, big) || !strings.Contains(p.stage, small) {
		t.Fatalf("no overflow staged holding both sections")
	}
	order = nil
	if err := p.apply(fail(2)); err == nil || strings.Join(order, ",") != "2,1" {
		t.Fatalf("written %q, %v: want the staged overflow, then the main comment refused", order, err)
	}
}

// A person who edits a section of the state comment on GitHub and
// writes its digest again is still told from workline: the section is
// read only as the comment's history says workline last wrote it.
func TestGitHubStateEditByHistory(t *testing.T) {
	t.Setenv("WORKLINE_APP", "workline-x")
	t.Setenv("WORKLINE_APP_ID", "")
	own := Own{logins: []string{"workline-x[bot]"}}
	rev, tes := Marker("sticky=reviewer/acts"), Marker("sticky=tester/acts")
	p, _ := planState(nil, own, "Reviewer found two.", rev, true, time.Time{})
	p, _ = planState([]Note{{ID: "7", Body: p.main, Author: "workline-x[bot]"}}, own, "Tester ran it.", tes, true, time.Time{})
	wrote := p.main
	old, forged := "Reviewer found two.\n\n"+rev, "Reviewer found none.\n\n"+rev
	body := strings.Replace(strings.Replace(wrote, old, forged, 1), digest(old), digest(forged), 1)
	note, _ := json.Marshal(map[string]string{"id": "7", "body": body, "author": "workline-x[bot]", "editor": "mallory"})
	last, _ := json.Marshal(map[string]string{"diff": wrote})
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in *userContentEdits*) cat <<'EOF'\n" + string(last) + "\nEOF\n;; *) cat <<'EOF'\n" + string(note) + "\nEOF\n;; esac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	k, err := ReadKept(&github{repo: t.TempDir()}, Target{Kind: "issue", ID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := k.Last(rev); ok {
		t.Errorf("the section a person edited is read: %q", got)
	}
	if got, ok := k.Last(tes); !ok || !strings.Contains(got, "Tester ran it.") {
		t.Errorf("the section left as workline wrote it: %q, %v, want it read", got, ok)
	}
}
