package forge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// GitLab is reached through its REST API alone, with the token and the
// project CI gives: no CLI to install, on gitlab.com or an instance of one's
// own.
func TestGitLabAPI(t *testing.T) {
	var got []string // what the forge was asked to write, "METHOD path field=value"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "secret" {
			http.Error(w, `{"message":"401 Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case r.Method == "GET" && p == "/merge_requests/7/notes" && r.URL.Query().Get("page") != "2":
			w.Header().Set("X-Next-Page", "2")
			json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "body": "someone's"}})
		case r.Method == "GET" && p == "/merge_requests/7/notes":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "body": "old <!-- workline:k -->", "author": map[string]any{"username": "workline-bot"}}})
		case r.Method == "GET" && p == "/api/v4/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "workline-bot"})
		case r.Method == "GET" && p == "/merge_requests/7":
			json.NewEncoder(w).Encode(map[string]any{"source_branch": "feat", "target_branch": "main", "source_project_id": 3, "target_project_id": 3, "title": "Average", "description": "Closes #4"})
		case r.Method == "PUT" || r.Method == "POST":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			var fields []string
			for k := range form {
				fields = append(fields, k+"="+form.Get(k))
			}
			got = append(got, r.Method+" "+p+" "+strings.Join(fields, " "))
			w.Write([]byte(`{"iid": 9}`))
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}

	// A sticky note on the second page is edited, not added again.
	if err := g.Sticky(Target{Kind: "merge-request", ID: 7}, "new", "<!-- workline:k -->", true); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "PUT /merge_requests/7/notes/2 body=new\n\n<!-- workline:k -->" {
		t.Fatalf("sticky: %q", got)
	}
	// A merge request's branch, and whether it lives in this project.
	mr, err := g.MergeRequest(7)
	if err != nil || mr.Branch != "feat" || mr.Base != "main" || !mr.Here || mr.Title != "Average" || mr.Body != "Closes #4" {
		t.Fatalf("branch: %+v %v", mr, err)
	}
	// A refused token is the forge out of reach, not the role's fault.
	t.Setenv("GITLAB_TOKEN", "wrong")
	if _, err := (&gitlab{repo: t.TempDir()}).Issue(1); err == nil || !strings.Contains(err.Error(), ErrUnreachable.Error()) {
		t.Fatalf("a refused token: %v", err)
	}
}

// Who is of the project comes from its members' access level: Planner and
// above; a project access token's user is a bot. A note another user wrote
// cannot be edited (403): the sticky one is written anew after it.
func TestGitLabWho(t *testing.T) {
	var got []string
	members, written := true, false           // written: the bot wrote its own state note since
	refused, asked := http.StatusForbidden, 0 // the members' answer when not readable; the times they were asked
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		author := func(u string) map[string]any { return map[string]any{"username": u} }
		if p == "/members/all" {
			asked++
		}
		switch {
		case p == "/members/all" && !members:
			http.Error(w, fmt.Sprintf(`{"message":"%d %s"}`, refused, http.StatusText(refused)), refused)
		case p == "/members/all":
			json.NewEncoder(w).Encode([]map[string]any{
				{"username": "owner", "access_level": 50, "state": "active"},
				{"username": "plan", "access_level": 15, "state": "active"},
				{"username": "guest", "access_level": 10, "state": "active"},
				{"username": "gone", "access_level": 30, "state": "blocked"},
				{"username": "project_1_bot_0a1b", "access_level": 15, "state": "active"},
			})
		case r.Method == "GET" && p == "/api/v4/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "project_1_bot_0a1b"})
		case r.Method == "GET" && p == "/issues/3":
			json.NewEncoder(w).Encode(map[string]any{"iid": 3, "title": "t", "author": author("guest"), "state": "opened"})
		case r.Method == "GET" && p == "/issues/3/notes":
			notes := []map[string]any{
				{"id": 1, "body": "state <!-- workline:k -->", "author": author("owner")},
				{"id": 2, "body": "changed the description", "system": true, "author": author("plan")},
				{"id": 3, "body": "agreed", "author": author("guest")},
				{"id": 4, "body": "ok", "author": author("plan")},
				{"id": 5, "body": "done", "author": author("project_1_bot_0a1b")},
				{"id": 6, "body": "me too", "author": author("gone")},
			}
			if written {
				notes = append(notes, map[string]any{"id": 7, "body": "new <!-- workline:k -->", "author": author("project_1_bot_0a1b")})
			}
			if r.URL.Query().Get("sort") != "asc" { // GitLab's default: the newest first
				slices.Reverse(notes)
			}
			json.NewEncoder(w).Encode(notes)
		case r.Method == "PUT" && p == "/issues/3/notes/7":
			got = append(got, "PUT "+p)
			w.Write([]byte(`{"id": 7}`))
		case r.Method == "PUT" && p == "/issues/3/notes/1":
			http.Error(w, `{"message":"403 Forbidden"}`, http.StatusForbidden)
		case r.Method == "POST":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			got = append(got, r.Method+" "+p+" "+form.Get("body"))
			w.Write([]byte(`{"id": 7}`))
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}

	is, err := g.Issue(3)
	if err != nil || is.Author != "guest" || is.Insider {
		t.Fatalf("a guest's issue: %+v %v", is, err)
	}
	notes, err := g.Notes(Target{Kind: "issue", ID: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []Note{
		{Body: "state <!-- workline:k -->", Author: "owner", Insider: true, ID: "1"},
		{Body: "agreed", Author: "guest", ID: "3"},
		{Body: "ok", Author: "plan", Insider: true, ID: "4"},
		{Body: "done", Author: "project_1_bot_0a1b", Insider: true, Bot: true, ID: "5"},
		{Body: "me too", Author: "gone", ID: "6"},
	}
	if fmt.Sprint(notes) != fmt.Sprint(want) {
		t.Fatalf("notes:\n%+v\nwant\n%+v", notes, want)
	}
	if err := g.Sticky(Target{Kind: "issue", ID: 3}, "new", "<!-- workline:k -->", false); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "POST /issues/3/notes new\n\n<!-- workline:k -->" {
		t.Fatalf("a sticky note another user wrote: %q", got)
	}
	// Written anew, the bot's own is the one edited from then on.
	written, got = true, nil
	if err := g.Sticky(Target{Kind: "issue", ID: 3}, "newer", "<!-- workline:k -->", false); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "PUT /issues/3/notes/7" {
		t.Fatalf("the bot's own state note: %q", got)
	}
	// Members not readable — 403, or 401 as the job's token gets (#552):
	// an issue's author is said unknown, never every reporter taken for an
	// outsider; the comments read all the same, their bodies needing no
	// member, and with their authors each a bot by its name, none of the
	// project. The members are asked once.
	members = false
	for _, refused = range []int{http.StatusForbidden, http.StatusUnauthorized} {
		g, asked = &gitlab{repo: t.TempDir()}, 0
		bodies, err := g.Comments(Target{Kind: "issue", ID: 3})
		if err != nil || len(bodies) != 6 || bodies[0] != "state <!-- workline:k -->" || asked != 0 {
			t.Fatalf("%d, the comments: %q %v, the members asked %d times", refused, bodies, err, asked)
		}
		if _, err := g.Issue(3); !errors.Is(err, ErrAuthorsUnknown) || !strings.Contains(err.Error(), "members") {
			t.Fatalf("%d, members refused: %v", refused, err)
		}
		notes, err = g.Notes(Target{Kind: "issue", ID: 3})
		if !errors.Is(err, ErrAuthorsUnknown) || !errors.Is(err, ErrUnreachable) {
			t.Fatalf("%d, the notes: %v", refused, err)
		}
		want := []Note{
			{Body: "state <!-- workline:k -->", Author: "owner", ID: "1"},
			{Body: "agreed", Author: "guest", ID: "3"},
			{Body: "ok", Author: "plan", ID: "4"},
			{Body: "done", Author: "project_1_bot_0a1b", Bot: true, ID: "5"},
			{Body: "me too", Author: "gone", ID: "6"},
			{Body: "new <!-- workline:k -->", Author: "project_1_bot_0a1b", Bot: true, ID: "7"},
		}
		if fmt.Sprint(notes) != fmt.Sprint(want) {
			t.Fatalf("%d, the notes:\n%+v\nwant\n%+v", refused, notes, want)
		}
		if asked != 1 {
			t.Fatalf("%d: the members asked %d times, want once", refused, asked)
		}
	}
	// Members out of reach for a moment are no refusal: the notes fail.
	refused = http.StatusNotImplemented
	if _, err := (&gitlab{repo: t.TempDir()}).Notes(Target{Kind: "issue", ID: 3}); err == nil || errors.Is(err, ErrAuthorsUnknown) {
		t.Fatalf("members out of reach: %v", err)
	}
}

// A bot GitLab does not name so (#552): a service account the project
// names in `gitlab: {bots}` is a bot, the members read or not; with the
// members unread, the bot's own token's user (WORKLINE_GITLAB_TOKEN, GET
// /user) is one too, a person's token's never.
func TestGitLabBotsNamed(t *testing.T) {
	members, userAsked := true, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		author := func(u string) map[string]any { return map[string]any{"username": u} }
		switch {
		case p == "/api/v4/user":
			userAsked++
			json.NewEncoder(w).Encode(map[string]any{"id": 9, "username": "line-account", "bot": false})
		case p == "/members/all" && !members:
			http.Error(w, `{"message":"401 Unauthorized"}`, http.StatusUnauthorized)
		case p == "/members/all":
			json.NewEncoder(w).Encode([]map[string]any{{"username": "service_account_group_7_ab12", "access_level": 30, "state": "active"}})
		case p == "/merge_requests/5/notes":
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "body": "kept", "author": author("service_account_group_7_ab12")},
				{"id": 2, "body": "applied", "author": author("line-account")},
				{"id": 3, "body": "a word", "author": author("alice")},
			})
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "the-bots")
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".workline"), 0o755)
	os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("gitlab: {bots: [service_account_group_7_ab12]}\n"), 0o644)
	mr := Target{Kind: "merge-request", ID: 5}
	bots := func(notes []Note) string {
		var out []string
		for _, n := range notes {
			out = append(out, fmt.Sprintf("%s:%v", n.Author, n.Bot))
		}
		return strings.Join(out, " ")
	}

	notes, err := (&gitlab{repo: repo}).Notes(mr)
	if err != nil || bots(notes) != "service_account_group_7_ab12:true line-account:false alice:false" || !notes[0].Insider {
		t.Fatalf("members read: %+v %v", notes, err)
	}
	members = false
	notes, err = (&gitlab{repo: repo}).Notes(mr)
	if !errors.Is(err, ErrAuthorsUnknown) || bots(notes) != "service_account_group_7_ab12:true line-account:false alice:false" || userAsked != 0 {
		t.Fatalf("members unread, a person's token: %+v %v, /user asked %d times", notes, err, userAsked)
	}
	t.Setenv("WORKLINE_GITLAB_TOKEN", "the-bots")
	notes, err = (&gitlab{repo: repo}).Notes(mr)
	if !errors.Is(err, ErrAuthorsUnknown) || bots(notes) != "service_account_group_7_ab12:true line-account:true alice:false" || userAsked != 1 {
		t.Fatalf("members unread, the bot's own token: %+v %v, /user asked %d times", notes, err, userAsked)
	}
}

// A person's answer on an issue (ADR-0038): who set or took off a label,
// and when, from its label events; 👀 on a note, once — GitLab's 404 for
// an emoji already awarded is done, not a failure.
func TestGitLabAnswers(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case p == "/members/all":
			json.NewEncoder(w).Encode([]map[string]any{{"username": "owner", "access_level": 50, "state": "active"}})
		case p == "/issues/3/resource_label_events":
			json.NewEncoder(w).Encode([]map[string]any{
				{"action": "add", "created_at": "2026-10-08T08:36:31Z", "label": map[string]any{"name": "workline::proposed"}, "user": map[string]any{"username": "project_1_bot_0a1b"}},
				{"action": "remove", "created_at": "2026-10-09T07:00:00Z", "label": map[string]any{"name": "workline::proposed"}, "user": map[string]any{"username": "owner"}},
				{"action": "add", "created_at": "2026-10-09T07:01:00Z", "label": nil, "user": map[string]any{"username": "owner"}},
			})
		case r.Method == "POST" && p == "/issues/3/notes/12/award_emoji" && len(got) > 0:
			http.Error(w, `{"message":"404 Award Emoji Name has already been taken"}`, http.StatusNotFound)
		case r.Method == "POST" && p == "/issues/3/notes/12/award_emoji":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			got = append(got, "POST "+p+" "+form.Get("name"))
			w.Write([]byte(`{"name": "eyes"}`))
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}
	events, err := g.LabelEvents(3)
	want := []LabelEvent{
		{Label: "workline::proposed", Added: true, Note: Note{Author: "project_1_bot_0a1b", Bot: true, Created: "2026-10-08T08:36:31Z"}},
		{Label: "workline::proposed", Writer: true, Note: Note{Author: "owner", Insider: true, Created: "2026-10-09T07:00:00Z"}},
	}
	if err != nil || fmt.Sprint(events) != fmt.Sprint(want) {
		t.Fatalf("label events: %+v %v\nwant %+v", events, err, want)
	}
	for range 2 {
		if err := g.React(3, "12", Eyes); err != nil {
			t.Fatalf("react: %v", err)
		}
	}
	if len(got) != 1 || got[0] != "POST /issues/3/notes/12/award_emoji eyes" {
		t.Fatalf("react: %q", got)
	}
	if !ScopedLabels(g) || ScopedLabels(&github{}) {
		t.Fatal("GitLab's labels are scoped, GitHub's are not")
	}
}

// A canceled pipeline is no result (#377): the newest other pipeline on
// the merge request's head counts, a pipeline on an older head never;
// none, the head waits, pending, never red.
func TestGitLabCanceledPipeline(t *testing.T) {
	quick(t) // its 500 is tried again
	var head string
	var pipelines []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case p == "/merge_requests/7":
			json.NewEncoder(w).Encode(map[string]any{"sha": "b2", "head_pipeline": map[string]any{"status": head}})
		case p == "/merge_requests/7/pipelines" && pipelines == nil:
			http.Error(w, `{"message":"500 Internal Server Error"}`, http.StatusInternalServerError)
		case p == "/merge_requests/7/pipelines":
			json.NewEncoder(w).Encode(pipelines)
		case p == "/merge_requests/7/resource_label_events", p == "/merge_requests/7/versions", p == "/merge_requests/7/notes":
			w.Write([]byte(`[]`))
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	pipe := func(id int, sha, status string) map[string]any {
		return map[string]any{"id": id, "sha": sha, "status": status}
	}
	for _, c := range []struct {
		name, head string
		pipelines  []map[string]any
		want       string
	}{
		{"failed", "failed", nil, "failure"},
		{"an older one passed on this head", "canceled", []map[string]any{pipe(2, "b2", "success"), pipe(3, "b2", "canceled"), pipe(1, "b1", "failed")}, "success"},
		{"an older one failed on this head", "canceling", []map[string]any{pipe(3, "b2", "canceled"), pipe(2, "b2", "failed")}, "failure"},
		{"only canceled on this head", "canceled", []map[string]any{pipe(3, "b2", "canceled"), pipe(2, "b1", "success")}, "pending"},
	} {
		head, pipelines = c.head, c.pipelines
		st, err := (&gitlab{repo: t.TempDir()}).MergeState(MergeRequest{ID: 7})
		if err != nil || st.Checks != c.want || st.ChecksUnread != "" {
			t.Errorf("%s: %q (unread %q), %v; want %q", c.name, st.Checks, st.ChecksUnread, err, c.want)
		}
	}
	// The pipelines unread: said, never red.
	head, pipelines = "canceled", nil
	st, err := (&gitlab{repo: t.TempDir()}).MergeState(MergeRequest{ID: 7})
	if err != nil || st.Checks != "" || st.ChecksUnread == "" {
		t.Errorf("unread: %+v, %v", st, err)
	}
}

// GitLab's weight (ADR-0040): Premium's issues carry the key, null for
// none, Free's none, an empty project none yet; a weight is read back from
// GitLab's answer, Free answering 200 without one, as gitlab.com did on
// 2026-10-18.
func TestGitLabWeight(t *testing.T) {
	var premium bool
	var issues []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case r.Method == "GET" && p == "/issues":
			json.NewEncoder(w).Encode(issues)
		case r.Method == "PUT" && p == "/issues/9":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			answer := map[string]any{"iid": 9}
			if premium {
				answer["weight"] = json.Number(form.Get("weight"))
			}
			json.NewEncoder(w).Encode(answer)
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}
	for _, c := range []struct {
		name    string
		issues  []map[string]any
		premium bool
	}{
		{"empty", nil, false},
		{"free", []map[string]any{{"iid": 9}}, false},
		{"premium", []map[string]any{{"iid": 9, "weight": nil}}, true},
	} {
		issues, premium = c.issues, c.premium
		if got, err := g.weights(); err != nil || got != c.premium {
			t.Errorf("%s: weights %v, %v", c.name, got, err)
		}
		err := g.setWeight(9, 3)
		if c.premium && err != nil {
			t.Errorf("%s: set: %v", c.name, err)
		}
		if !c.premium && (err == nil || !strings.Contains(err.Error(), "kept no weight 3")) {
			t.Errorf("%s: set: %v; want the weight not kept said", c.name, err)
		}
	}
}

// Outside CI, the instance and the project come from the repository's remote.
func TestGitLabFromRemote(t *testing.T) {
	for _, c := range []struct{ remote, api, project string }{
		{"git@gitlab.example.com:group/sub/proj.git", "https://gitlab.example.com/api/v4", "group%2Fsub%2Fproj"},
		{"https://gitlab.com/JN0V/workline-sandbox.git", "https://gitlab.com/api/v4", "JN0V%2Fworkline-sandbox"},
		{"https://oauth2:tok@git.acme.local:8443/a/b", "https://git.acme.local:8443/api/v4", "a%2Fb"},
		{"ssh://git@gitlab.acme.io:2222/team/app.git", "https://gitlab.acme.io/api/v4", "team%2Fapp"},
	} {
		api, project, ok := fromRemote(c.remote)
		if !ok || api != c.api || project != c.project {
			t.Errorf("%s: %q %q %v; want %q %q", c.remote, api, project, ok, c.api, c.project)
		}
	}
}
