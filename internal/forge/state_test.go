package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// stateFake is a simulated forge holding issues 1 and 3 and merge
// request 2, none with a comment.
func stateFake(t *testing.T) *Fake {
	t.Helper()
	path := filepath.Join(t.TempDir(), "forge.json")
	data := `{"issues":[{"id":1,"title":"one"},{"id":3,"title":"three"}],"merge-requests":[{"id":2,"branch":"b"}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return &Fake{Path: path}
}

// stateItem is the item as the simulated forge's file holds it: its
// comments as written, not as a role reads them.
func stateItem(t *testing.T, f *Fake, tg Target) (*FakeState, *FakeItem) {
	t.Helper()
	s, err := f.loadState()
	if err != nil {
		t.Fatal(err)
	}
	it, err := s.item(tg)
	if err != nil {
		t.Fatal(err)
	}
	return s, it
}

func stateMarker(role string) string { return Marker("sticky=" + role + "/state") }

func stateWrite(t *testing.T, f *Fake, tg Target, role, text string) {
	t.Helper()
	if err := f.Sticky(tg, text, stateMarker(role), true); err != nil {
		t.Fatalf("%s writes: %v", role, err)
	}
}

// historyLines are the lines of a body that are one line of the dated
// history: the date, the time, the role and a sentence.
func historyLines(body string) []string {
	return regexp.MustCompile(`(?m)^(?:[-*] )?\d{4}-\d{2}-\d{2} \d{2}:\d{2}(?: UTC)? \S+: \S.*$`).FindAllString(body, -1)
}

// One comment: Given an issue the product owner, the reviewer, the
// tester and the developer each wrote on, When the line has run on it,
// Then it holds one comment of workline's, with a section for each
// role; each role still reads its section as the comment it was.
func TestStateOneComment(t *testing.T) {
	f := stateFake(t)
	issue := Target{Kind: "issue", ID: 1}
	roles := []string{"product-owner", "reviewer", "tester", "developer"}
	for _, r := range roles {
		stateWrite(t, f, issue, r, "The "+strings.ToUpper(r[:1])+" record of this run.")
	}
	_, it := stateItem(t, f, issue)
	if len(it.Comments) != 1 || it.Posted != 1 {
		t.Fatalf("%d comments, %d posted; want one comment, posted once", len(it.Comments), it.Posted)
	}
	for _, r := range roles {
		if !strings.Contains(it.Comments[0].Body, r) {
			t.Errorf("no section for %s in:\n%s", r, it.Comments[0].Body)
		}
	}
	k, err := ReadKept(f, issue)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles {
		body, ok := k.Last(stateMarker(r))
		if !ok || !strings.Contains(body, "The "+strings.ToUpper(r[:1])+" record of this run.") {
			t.Errorf("%s reads %q, %v from its section", r, body, ok)
		}
	}
}

// Edited, not posted: Given an issue with workline's comment, When
// another role writes on it, Then the comment is edited, none is
// posted, and no new comment appears.
func TestStateEditedNotPosted(t *testing.T) {
	f := stateFake(t)
	issue := Target{Kind: "issue", ID: 1}
	stateWrite(t, f, issue, "reviewer", "The reviewer's first record.")
	stateWrite(t, f, issue, "tester", "The tester's record.")
	_, it := stateItem(t, f, issue)
	if len(it.Comments) != 1 || it.Posted != 1 {
		t.Fatalf("%d comments, %d posted after another role wrote; want the one comment edited, none posted", len(it.Comments), it.Posted)
	}
	body := it.Comments[0].Body
	if !strings.Contains(body, "The reviewer's first record.") || !strings.Contains(body, "The tester's record.") {
		t.Errorf("a section lost:\n%s", body)
	}
}

// Dated history: Given a comment edited by 3 runs, When a person reads
// it, Then its end lists 3 lines, each with the date, the time, the
// role and one sentence.
func TestStateHistoryDated(t *testing.T) {
	f := stateFake(t)
	issue := Target{Kind: "issue", ID: 1}
	roles := []string{"reviewer", "tester", "developer"}
	for _, r := range roles {
		stateWrite(t, f, issue, r, "A sentence from the "+r+" about this run.")
	}
	_, it := stateItem(t, f, issue)
	if len(it.Comments) != 1 {
		t.Fatalf("%d comments, want one", len(it.Comments))
	}
	lines := historyLines(it.Comments[0].Body)
	if len(lines) != 3 {
		t.Fatalf("%d history lines, want 3:\n%s", len(lines), it.Comments[0].Body)
	}
	for _, r := range roles {
		n := 0
		for _, l := range lines {
			if strings.Contains(l, " "+r+": ") {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%d history lines for %s, want 1: %q", n, r, lines)
		}
	}
}

// History kept short: Given 12 edits, When a person reads it, Then the
// last 10 lines show; every section stays. (GitHub's "edited" menu is
// the forge's own: the simulated forge keeps no versions of a comment.)
func TestStateHistoryKeptShort(t *testing.T) {
	f := stateFake(t)
	issue := Target{Kind: "issue", ID: 1}
	names := []string{"r01", "r02", "r03", "r04", "r05", "r06", "r07", "r08", "r09", "r10", "r11", "r12"}
	for _, r := range names {
		stateWrite(t, f, issue, r, "A sentence from "+r+" about this run.")
	}
	_, it := stateItem(t, f, issue)
	if len(it.Comments) != 1 {
		t.Fatalf("%d comments, want one", len(it.Comments))
	}
	body := it.Comments[0].Body
	lines := historyLines(body)
	if len(lines) != 10 {
		t.Fatalf("%d history lines, want 10:\n%s", len(lines), body)
	}
	for i, r := range names {
		in := strings.Contains(strings.Join(lines, "\n"), " "+r+": ")
		if want := i >= 2; in != want {
			t.Errorf("%s in the history: %v, want %v", r, in, want)
		}
		if !strings.Contains(body, "A sentence from "+r+" about this run.") {
			t.Errorf("the section of %s is gone", r)
		}
	}
}

// Older comments folded: Given a pull request with 10 comments from an
// earlier workline, written by workline's App, When the line writes on
// it, Then they become one comment and the others are gone.
func TestStateFoldsOlderComments(t *testing.T) {
	f := stateFake(t)
	mr := Target{Kind: "merge-request", ID: 2}
	s, it := stateItem(t, f, mr)
	for _, n := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"} {
		it.Comments = append(it.Comments, FakeComment{Body: "Finding number " + n + " said.\n\n" + Marker("sticky=reviewer/lens-"+n), Author: fakeBot, Bot: true})
	}
	if err := f.save(s); err != nil {
		t.Fatal(err)
	}
	stateWrite(t, f, mr, "developer", "The developer's record.")
	_, it = stateItem(t, f, mr)
	if len(it.Comments) != 1 {
		t.Fatalf("%d comments, want the 10 older ones and the new section as one", len(it.Comments))
	}
	for _, n := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"} {
		if !strings.Contains(it.Comments[0].Body, "Finding number "+n+" said.") {
			t.Errorf("older comment %s not folded in:\n%s", n, it.Comments[0].Body)
		}
	}
}

// A person's token left alone: Given a role's comment written under a
// person's account by a local run, When the line writes on the item,
// Then that comment stays as it is, and workline's own is one comment.
func TestStatePersonsTokenLeftAlone(t *testing.T) {
	f := stateFake(t)
	mr := Target{Kind: "merge-request", ID: 2}
	s, it := stateItem(t, f, mr)
	persons := FakeComment{Body: "The person's local run said this.\n\n" + stateMarker("tester"), Author: "dana", Insider: true}
	it.Comments = append(it.Comments, persons)
	if err := f.save(s); err != nil {
		t.Fatal(err)
	}
	if err := KeepOwn(f, mr, "The reviewer's record.", stateMarker("reviewer"), true); err != nil {
		t.Fatal(err)
	}
	if err := KeepOwn(f, mr, "The tester's record.", stateMarker("tester"), true); err != nil {
		t.Fatal(err)
	}
	_, it = stateItem(t, f, mr)
	own := 0
	found := false
	for _, c := range it.Comments {
		if c.Author == "dana" {
			found = true
			if c.Body != persons.Body || c.Editor != "" {
				t.Errorf("the person's comment changed: %+v", c)
			}
			continue
		}
		own++
		if !strings.Contains(c.Body, "The reviewer's record.") && own == 1 {
			t.Errorf("workline's comment lacks the reviewer's section:\n%s", c.Body)
		}
	}
	if !found {
		t.Error("the person's comment is gone")
	}
	if own != 1 {
		t.Errorf("%d comments of workline's, want one holding both sections", own)
	}
}

// Too long, cut and said: Given sections that together pass 65,536
// characters, When the line writes, Then the history is cut first, then
// the largest section is cut, it ends with a line saying it was cut,
// and the write succeeds.
func TestStateTooLongCutAndSaid(t *testing.T) {
	f := stateFake(t)
	issue := Target{Kind: "issue", ID: 3}
	stateWrite(t, f, issue, "tester", "The tester's small record.")
	stateWrite(t, f, issue, "developer", "The developer's small record.")
	huge := "A long record.\n\n" + strings.Repeat("Z", 70000)
	if err := f.Sticky(issue, huge, stateMarker("reviewer"), true); err != nil {
		t.Fatalf("the write failed: %v", err)
	}
	_, it := stateItem(t, f, issue)
	if len(it.Comments) != 1 {
		t.Fatalf("%d comments, want one: no overflow comment here", len(it.Comments))
	}
	body := it.Comments[0].Body
	if n := utf8.RuneCountInString(body); n > 65536 {
		t.Errorf("the comment holds %d characters, over 65,536", n)
	}
	if z := strings.Count(body, "Z"); z < 1000 || z >= 70000 {
		t.Errorf("the largest section holds %d of its 70000 characters; want it cut, not dropped", z)
	}
	if !strings.Contains(body, "The tester's small record.") || !strings.Contains(body, "The developer's small record.") {
		t.Errorf("a small section was cut")
	}
	if h := historyLines(body); len(h) != 0 {
		t.Errorf("the history stays, cut first: %q", h)
	}
	said := regexp.MustCompile(`(?mi)^.*\bcut\b.*$`).FindStringIndex(body)
	if said == nil || said[0] < strings.LastIndex(body, "Z") {
		t.Errorf("no line after the section saying it was cut (%v)", said)
	}
}

// Two writers at once: Given two roles read the comment, then both
// write, one after the other, When both are done, Then both sections
// are in the comment. The simulated GitLab answers the first two reads
// of the notes together, before any write: read A, read B, write A,
// write B.
func TestStateTwoWriters(t *testing.T) {
	type note struct {
		ID   int
		Body string
	}
	var (
		mu      sync.Mutex
		store   []note
		next    = 1
		gated   bool
		arrived int
	)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		full := r.URL.EscapedPath()
		p := strings.TrimPrefix(full, "/api/v4/projects/1")
		switch {
		case full == "/api/v4/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "project_1_bot_0a1b"})
		case p == "/members/all":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == "GET" && p == "/merge_requests/5/notes":
			mu.Lock()
			var list []map[string]any
			for _, n := range store {
				list = append(list, map[string]any{"id": n.ID, "body": n.Body, "author": map[string]any{"username": "project_1_bot_0a1b"}})
			}
			wait := gated && arrived < 2
			if wait {
				arrived++
				if arrived == 2 {
					close(release)
				}
			}
			mu.Unlock()
			if wait {
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == "POST" && p == "/merge_requests/5/notes":
			_ = r.ParseForm()
			mu.Lock()
			store = append(store, note{ID: next, Body: r.PostForm.Get("body")})
			next++
			mu.Unlock()
			_, _ = w.Write([]byte(`{"id": 9}`))
		case r.Method == "PUT" && strings.HasPrefix(p, "/merge_requests/5/notes/"):
			_ = r.ParseForm()
			mu.Lock()
			for i := range store {
				if p == "/merge_requests/5/notes/"+itoa(store[i].ID) {
					store[i].Body = r.PostForm.Get("body")
				}
			}
			mu.Unlock()
			_, _ = w.Write([]byte(`{"id": 9}`))
		case r.Method == "DELETE" && strings.HasPrefix(p, "/merge_requests/5/notes/"):
			mu.Lock()
			var kept []note
			for _, n := range store {
				if p != "/merge_requests/5/notes/"+itoa(n.ID) {
					kept = append(kept, n)
				}
			}
			store = kept
			mu.Unlock()
			_, _ = w.Write([]byte(`{}`))
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
	if err := (&gitlab{repo: t.TempDir()}).Sticky(mr, "The reviewer's record.", stateMarker("reviewer"), true); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	gated = true
	mu.Unlock()
	roles := []string{"tester", "developer"}
	repos := []string{t.TempDir(), t.TempDir()}
	errs := make(chan error, len(roles))
	var wg sync.WaitGroup
	for i, r := range roles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- (&gitlab{repo: repos[i]}).Sticky(mr, "The "+r+"'s record.", stateMarker(r), true)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(store) != 1 {
		t.Fatalf("%d notes, want one comment", len(store))
	}
	for _, r := range []string{"reviewer", "tester", "developer"} {
		if !strings.Contains(store[0].Body, stateMarker(r)) || !strings.Contains(store[0].Body, "The "+r+"'s record.") {
			t.Errorf("the section of %s is lost:\n%s", r, store[0].Body)
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// A person's edit: Given a person edited one section of the comment,
// When a role reads its own section, Then the other sections' records
// are still read, and the edited one is said unread.
func TestStatePersonsEdit(t *testing.T) {
	f := stateFake(t)
	issue := Target{Kind: "issue", ID: 1}
	stateWrite(t, f, issue, "reviewer", "The reviewer-record-alpha stands.")
	stateWrite(t, f, issue, "tester", "The tester-record-alpha stands.")
	s, it := stateItem(t, f, issue)
	if len(it.Comments) != 1 {
		t.Fatalf("%d comments, want one comment with the two sections", len(it.Comments))
	}
	if !strings.Contains(it.Comments[0].Body, "tester-record-alpha") {
		t.Fatalf("the tester's section is not there:\n%s", it.Comments[0].Body)
	}
	it.Comments[0].Body = strings.Replace(it.Comments[0].Body, "tester-record-alpha", "tester-record-changed-by-a-person", 1)
	it.Comments[0].Editor = "dana"
	if err := f.save(s); err != nil {
		t.Fatal(err)
	}
	k, err := ReadKept(f, issue)
	if err != nil {
		t.Fatal(err)
	}
	if body, ok := k.Last(stateMarker("reviewer")); !ok || !strings.Contains(body, "reviewer-record-alpha") {
		t.Errorf("the reviewer's section: %q, %v; want it read", body, ok)
	}
	if err := k.Unread(stateMarker("reviewer")); err != nil {
		t.Errorf("the reviewer's section said unread: %v", err)
	}
	if body, ok := k.Last(stateMarker("tester")); ok {
		t.Errorf("the edited section read: %q", body)
	}
	if err := k.Unread(stateMarker("tester")); err == nil {
		t.Error("the edited section not said unread")
	}
}

// Waits keep their mail: Given a feature's pull request waits for a
// person's yes, When the line says it, Then that wait is still a
// comment of its own, as today, apart from the state comment.
func TestStateWaitsKeepTheirMail(t *testing.T) {
	f := stateFake(t)
	mr := Target{Kind: "merge-request", ID: 2}
	wait := Marker("merge-waits=feature")
	if err := f.Sticky(mr, "This waits for a person's yes.", wait, true); err != nil {
		t.Fatal(err)
	}
	stateWrite(t, f, mr, "reviewer", "The reviewer's record.")
	if err := f.Sticky(mr, "This still waits for a person's yes.", wait, true); err != nil {
		t.Fatal(err)
	}
	_, it := stateItem(t, f, mr)
	if len(it.Comments) != 2 {
		t.Fatalf("%d comments, want the wait's and the state's", len(it.Comments))
	}
	for _, c := range it.Comments {
		isWait := strings.Contains(c.Body, wait)
		if isWait == strings.Contains(c.Body, stateMarker("reviewer")) {
			t.Errorf("the wait and the state share a comment, or neither is there:\n%s", c.Body)
		}
		if isWait && !strings.Contains(c.Body, "This still waits for a person's yes.") {
			t.Errorf("the wait was not edited in place:\n%s", c.Body)
		}
	}
}
