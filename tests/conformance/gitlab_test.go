package conformance

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// gitlabMock answers the part of GitLab's REST API the engine's gitlab forge
// calls, over the case's simulated forge file, so a case runs `--forge
// gitlab` and is checked as any other. Its `members` give the project's
// members and their access level, as GitLab's members/all does, or
// `members-refused` its refusal; its `variables`, the project's CI/CD
// variables and their environment scope, or `variables-refused`; what the engine writes is written by the
// token's user, "workline-bot" or its `token-user`. GraphQL
// refuses all: a split's children are listed in the parent's body. Its
// issue links are GitLab Free's: is_blocked_by refused, for its license.
// With `weights`, its issues carry Premium's `weight` key, written by a
// PUT; without, Free's: no key, a weight sent ignored.
// A merge request's diff is the repository's, from its base (main) to its
// branch (HEAD), as GitLab computes it; a discussion on a line it does not
// show, or a kept line without its old one, is taken on no line, as
// gitlab.com does (#277).
// Its `faults` fail calls for a moment (gitlabFault); a fault no call
// met fails the case, which would prove nothing (unmet).
type gitlabMock struct {
	file, repo string
	mu         sync.Mutex
	met        map[int]*faultMet // by the fault's place in faults
}

// gitlabFault fails the calls of a method on a path for a moment: they
// get no answer until the engine gives up (`stall`), or a status (`status`:
// 500, 502, 503, 429), with `retry-after` seconds. `skip` calls are
// answered first; then `times` fail (0: all), then they are answered. A
// call sooner than the retry-after asked gets the status again, uncounted.
// A write stalled is taken all the same, answered too late.
type gitlabFault struct {
	Method     string `json:"method,omitempty"` // GET when not given
	Path       string `json:"path"`             // the project's, as /issues/9/notes
	Stall      bool   `json:"stall,omitempty"`
	Status     int    `json:"status,omitempty"`
	RetryAfter int    `json:"retry-after,omitempty"`
	Skip       int    `json:"skip,omitempty"`
	Times      int    `json:"times,omitempty"`
}

// faultMet is what a fault met: the calls it saw, those it failed, when
// it last failed one.
type faultMet struct {
	calls, failed int
	at            time.Time
}

type gitlabState struct {
	forge.FakeState
	Members []struct {
		Username string `json:"username"`
		Level    int    `json:"access_level"`
	} `json:"members,omitempty"`
	// Bots are the users the users API says are bots (`bot`), a service
	// account's among them; a merge request's author is user 1000 + its id.
	Bots   []string      `json:"bots,omitempty"`
	Faults []gitlabFault `json:"faults,omitempty"`
	// MembersRefused is the status members/all answers the token, when it
	// may not list them: 401, as the job's own token gets (#552); asked or
	// not, unlike a fault.
	MembersRefused int `json:"members-refused,omitempty"`
	// Variables are the project's CI/CD variables, as GitLab lists them
	// to a Maintainer's token, values included; VariablesRefused, the
	// status it answers a token that may not list them (403, a
	// Developer's).
	Variables []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Scope string `json:"environment_scope"`
	} `json:"variables,omitempty"`
	VariablesRefused int `json:"variables-refused,omitempty"`
	// TokenUser is the token's user, who writes what the engine writes:
	// workline-bot when not given; project_<id>_bot_… for a project
	// access token's, a bot by its name.
	TokenUser string `json:"token-user,omitempty"`
}

func (s *gitlabState) tokenUser() string { return cmp.Or(s.TokenUser, defaultTokenUser) }

var (
	issuePath      = regexp.MustCompile(`^/issues/(\d+)$`)
	notesPath      = regexp.MustCompile(`^/(issues|merge_requests)/(\d+)/notes(?:/(\d+))?$`)
	mrPath         = regexp.MustCompile(`^/merge_requests/(\d+)(/diffs|/discussions|/merge|/cancel_merge_when_pipeline_succeeds|/versions|/resource_label_events)?$`)
	linksPath      = regexp.MustCompile(`^/issues/(\d+)/links$`)
	closedBy       = regexp.MustCompile(`^/issues/(\d+)/closed_by$`)
	stateEvents    = regexp.MustCompile(`^/issues/(\d+)/resource_state_events$`)
	commit         = regexp.MustCompile(`^/repository/commits/([0-9a-f]+)$`)
	commitStatuses = regexp.MustCompile(`^/repository/commits/([0-9a-f]+)/statuses$`)
	userPath       = regexp.MustCompile(`^/api/v4/users/(\d+)$`) // outside the project's path
)

const defaultTokenUser = "workline-bot"

func (m *gitlabMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	answer, status, f := m.answer(r)
	if f != nil && f.Stall {
		// No answer until the engine gives up: it closes the connection.
		select {
		case <-r.Context().Done():
		case <-time.After(time.Minute):
		}
		return
	}
	if f != nil && f.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(f.RetryAfter))
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(answer)
}

// answer answers a call over the forge file, or the fault it meets: a
// status fault's status, or a stall's answer, never sent, its write taken.
func (m *gitlabMock) answer(r *http.Request) (any, int, *gitlabFault) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := os.ReadFile(m.file)
	if err != nil {
		return map[string]any{"message": err.Error()}, http.StatusInternalServerError, nil
	}
	var s gitlabState
	if err := json.Unmarshal(data, &s); err != nil {
		return map[string]any{"message": err.Error()}, http.StatusInternalServerError, nil
	}
	body, _ := io.ReadAll(r.Body)
	if r.URL.Path == "/api/graphql" {
		return editorsQuery(&s, body), 200, nil
	}
	form, _ := url.ParseQuery(string(body))
	p := strings.TrimPrefix(r.URL.Path, "/api/v4/projects/1")
	f := m.fault(s.Faults, r.Method, p)
	if f != nil && !f.Stall {
		return map[string]any{"message": http.StatusText(f.Status)}, f.Status, f
	}
	answer, status := m.serve(&s, r.Method, p, r.URL.Query(), form)
	if r.Method != "GET" && status < 300 {
		out, _ := json.MarshalIndent(s, "", "  ")
		os.WriteFile(m.file, out, 0o644)
	}
	return answer, status, f
}

// editorsQuery answers the one GraphQL query the simulated GitLab knows:
// who last edited a merge request's or an issue's notes (lastEditedBy), a
// comment's editor; any other is not simulated. Its notes' ids are the
// REST listing's.
func editorsQuery(s *gitlabState, body []byte) any {
	var req struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	_ = json.Unmarshal(body, &req)
	if !strings.Contains(req.Query, "lastEditedBy") {
		return map[string]any{"errors": []map[string]any{{"message": "not simulated"}}}
	}
	id, _ := strconv.Atoi(fmt.Sprint(req.Variables["iid"]))
	items := s.Issues
	if strings.Contains(req.Query, "mergeRequest(") {
		items = s.MergeRequests
	}
	nodes := []map[string]any{}
	for _, it := range items {
		if it.ID != id {
			continue
		}
		for i, c := range it.Comments {
			n := id*1000 + i + 1
			if v, err := strconv.Atoi(c.ID); err == nil {
				n = v
			}
			var editor any
			if c.Editor != "" {
				editor = map[string]any{"username": c.Editor}
			}
			nodes = append(nodes, map[string]any{"id": fmt.Sprintf("gid://gitlab/Note/%d", n), "author": map[string]any{"username": c.Author}, "lastEditedBy": editor})
		}
	}
	return map[string]any{"data": map[string]any{"project": map[string]any{"item": map[string]any{"notes": map[string]any{"nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false}}}}}}
}

// fault is the fault a call meets, nil for none; it counts what each
// fault met.
func (m *gitlabMock) fault(faults []gitlabFault, method, p string) *gitlabFault {
	if m.met == nil {
		m.met = map[int]*faultMet{}
	}
	for i, f := range faults {
		if f.Path != p || cmp.Or(f.Method, "GET") != method {
			continue
		}
		if m.met[i] == nil {
			m.met[i] = &faultMet{}
		}
		met := m.met[i]
		if met.failed > 0 && f.RetryAfter > 0 && time.Since(met.at) < time.Duration(f.RetryAfter)*time.Second {
			return &f // sooner than asked: refused again, uncounted
		}
		met.calls++
		if met.calls <= f.Skip || f.Times > 0 && met.failed >= f.Times {
			return nil
		}
		met.failed++
		met.at = time.Now()
		return &f
	}
	return nil
}

// unmet names the faults no call met: a case giving one proves nothing
// of it.
func (m *gitlabMock) unmet() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var s gitlabState
	data, _ := os.ReadFile(m.file)
	if err := json.Unmarshal(data, &s); err != nil {
		return []string{"the simulated GitLab's file: " + err.Error()}
	}
	var out []string
	for i, f := range s.Faults {
		if m.met[i] == nil || m.met[i].failed == 0 {
			out = append(out, fmt.Sprintf("faults[%d] (%s %s) met no call: the case proves nothing of it", i, cmp.Or(f.Method, "GET"), f.Path))
		}
	}
	return out
}

func (m *gitlabMock) serve(s *gitlabState, method, p string, q, form url.Values) (any, int) {
	tokenUser := s.tokenUser()
	issue := func(id int) *forge.FakeItem {
		for i := range s.Issues {
			if s.Issues[i].ID == id {
				return &s.Issues[i]
			}
		}
		return nil
	}
	mergeRequest := func(id int) *forge.FakeItem {
		for i := range s.MergeRequests {
			if s.MergeRequests[i].ID == id {
				return &s.MergeRequests[i]
			}
		}
		return nil
	}
	user := func(name string) map[string]any { return map[string]any{"username": name} }
	asIssue := func(it forge.FakeItem) map[string]any {
		v := map[string]any{"id": it.ID + 1000, "iid": it.ID, "title": it.Title, "description": it.Body,
			"labels": it.Labels, "state": map[bool]string{true: "closed", false: "opened"}[it.Closed],
			"author": user(it.Author), "issue_type": "issue"}
		if it.Milestone != "" {
			v["milestone"] = map[string]any{"title": it.Milestone}
		}
		if s.Weights {
			v["weight"] = it.Weight // Premium's key, null for none; Free's payload has none
		}
		return v
	}
	notFound := map[string]any{"message": "404 Not Found"}
	switch {
	case p == "/members/all" && s.MembersRefused != 0:
		return map[string]any{"message": fmt.Sprintf("%d %s", s.MembersRefused, http.StatusText(s.MembersRefused))}, s.MembersRefused
	case p == "/members/all":
		return s.Members, 200
	case p == "/variables" && s.VariablesRefused != 0:
		return map[string]any{"message": fmt.Sprintf("%d %s", s.VariablesRefused, http.StatusText(s.VariablesRefused))}, s.VariablesRefused
	case p == "/variables" && method == "GET":
		out := []map[string]any{}
		for _, v := range s.Variables {
			out = append(out, map[string]any{"variable_type": "env_var", "key": v.Key, "value": v.Value, "protected": false,
				"masked": true, "hidden": false, "raw": false, "environment_scope": v.Scope, "description": nil})
		}
		return out, 200
	case p == "/merge_requests" && method == "GET":
		out := []map[string]any{}
		for _, it := range s.MergeRequests {
			if q.Get("state") == "opened" && it.Closed || q.Get("state") == "merged" && it.MergedAt == "" ||
				q.Has("target_branch") && q.Get("target_branch") != it.Base {
				continue
			}
			source := 1 // a fork's is another project's
			if it.Fork {
				source = 2
			}
			mr := map[string]any{"iid": it.ID, "source_branch": it.Branch, "target_branch": it.Base,
				"source_project_id": source, "target_project_id": 1, "title": it.Title,
				"author": map[string]any{"id": 1000 + it.ID, "username": it.Author},
				"state":  map[bool]string{true: "closed", false: "opened"}[it.Closed],
				"labels": it.Labels, "draft": it.Draft, "merge_when_pipeline_succeeds": it.AutoMerge}
			if it.MergedAt != "" {
				// Merged (`merged-at`, `merged-as`): when, and the commit it
				// became, a revision of the repository, as its squash.
				at := it.MergedAt
				if len(at) == len(time.DateOnly) {
					at += "T00:00:00Z"
				}
				sha, _ := exec.Command("git", "-C", m.repo, "rev-parse", "--verify", "-q", it.MergedAs+"^{commit}").Output()
				mr["state"], mr["merged_at"], mr["squash_commit_sha"] = "merged", at, strings.TrimSpace(string(sha))
			}
			out = append(out, mr)
		}
		return out, 200
	case p == "/merge_requests" && method == "POST":
		// Opened by the token's user; a draft is a title starting "Draft:".
		id := 1
		for _, it := range s.MergeRequests {
			id = max(id, it.ID+1)
		}
		title, draft := strings.CutPrefix(form.Get("title"), "Draft: ")
		s.MergeRequests = append(s.MergeRequests, forge.FakeItem{ID: id, Title: title, Body: form.Get("description"),
			Branch: form.Get("source_branch"), Base: form.Get("target_branch"), Draft: draft,
			Labels: []string{}, Comments: []forge.FakeComment{}, Author: tokenUser})
		return map[string]any{"iid": id}, 201
	case p == "/api/v4/users/999", p == "/api/v4/user":
		// The token's user, who sets a merge request to merge (merge_user):
		// a bot, as a project access token's is.
		return map[string]any{"id": 999, "username": tokenUser, "bot": true}, 200
	case strings.HasPrefix(p, "/statuses/") && method == "POST":
		// A commit status, set by the token's user (the merge policy's mark).
		s.Statuses = append(s.Statuses, forge.FakeStatus{Commit: strings.TrimPrefix(p, "/statuses/"), Name: form.Get("name"),
			State: form.Get("state"), Description: form.Get("description"), By: tokenUser})
		return map[string]any{"id": len(s.Statuses)}, 201
	case commitStatuses.MatchString(p) && method == "GET":
		// A commit's statuses, the newest first, each with its author; the
		// simulated forge's commits are revisions of the repository.
		sha := commitStatuses.FindStringSubmatch(p)[1]
		out := []map[string]any{}
		for i, st := range s.Statuses {
			c, err := exec.Command("git", "-C", m.repo, "rev-parse", "--verify", "-q", st.Commit+"^{commit}").Output()
			if err != nil || strings.TrimSpace(string(c)) != sha || q.Has("name") && q.Get("name") != st.Name {
				continue
			}
			out = append(out, map[string]any{"name": st.Name, "status": st.State, "description": st.Description,
				"created_at": fmt.Sprintf("2026-01-04T10:%02d:00Z", i), "author": user(st.By)})
		}
		slices.Reverse(out)
		return out, 200
	case p == "/events" && method == "GET":
		// The project's push events, newest first: each merge request's
		// pushes, by whom the simulated forge says.
		out := []map[string]any{}
		for _, it := range s.MergeRequests {
			for i, push := range it.Pushes {
				action := map[bool]string{true: "pushed new", false: "pushed to"}[i == 0] // the first creates the branch
				out = append(out, map[string]any{"action_name": action, "author_username": push.By, "created_at": push.At,
					"push_data": map[string]any{"ref": it.Branch, "ref_type": "branch"}})
			}
		}
		slices.Reverse(out)
		return out, 200
	case userPath.MatchString(p):
		id, _ := strconv.Atoi(userPath.FindStringSubmatch(p)[1])
		it := mergeRequest(id - 1000)
		if it == nil {
			return notFound, 404
		}
		return map[string]any{"id": id, "username": it.Author, "bot": slices.Contains(s.Bots, it.Author)}, 200
	case p == "/issues" && method == "GET":
		out := []map[string]any{}
		for _, it := range s.Issues {
			if (q.Get("state") == "opened" && it.Closed) || (q.Get("search") != "" && !strings.Contains(it.Title, q.Get("search"))) {
				continue
			}
			out = append(out, asIssue(it))
		}
		return out, 200
	case p == "/issues" && method == "POST":
		id := 1
		for _, it := range s.Issues {
			id = max(id, it.ID+1)
		}
		s.Issues = append(s.Issues, forge.FakeItem{ID: id, Title: form.Get("title"), Body: form.Get("description"),
			Labels: []string{}, Comments: []forge.FakeComment{}, Author: tokenUser})
		return asIssue(s.Issues[len(s.Issues)-1]), 201
	case issuePath.MatchString(p):
		id, _ := strconv.Atoi(issuePath.FindStringSubmatch(p)[1])
		it := issue(id)
		if it == nil {
			return notFound, 404
		}
		if method == "PUT" {
			if v, ok := form["description"]; ok {
				it.Body = v[0]
			}
			if v, ok := form["title"]; ok {
				it.Title = v[0]
			}
			for _, l := range strings.Split(form.Get("add_labels"), ",") {
				if l != "" && !slices.Contains(it.Labels, l) {
					it.Labels = append(it.Labels, l)
				}
			}
			remove := strings.Split(form.Get("remove_labels"), ",")
			it.Labels = slices.DeleteFunc(it.Labels, func(l string) bool { return slices.Contains(remove, l) })
			if form.Get("state_event") == "close" {
				it.Closed, it.Reason = true, ""
			}
			if v := form.Get("milestone_id"); v != "" {
				n, _ := strconv.Atoi(v)
				if n >= 1 && n <= len(s.Milestones) {
					it.Milestone = s.Milestones[n-1]
				}
			}
			if v, ok := form["weight"]; ok && s.Weights {
				// Premium's; Free ignores the key, its answer holding none.
				n, _ := strconv.Atoi(v[0])
				it.Weight = &n
			}
		}
		return asIssue(*it), 200
	case notesPath.MatchString(p):
		sm := notesPath.FindStringSubmatch(p)
		id, _ := strconv.Atoi(sm[2])
		it := issue(id)
		if sm[1] == "merge_requests" {
			it = mergeRequest(id)
		}
		if it == nil {
			return notFound, 404
		}
		// A note's id is its place, kept once a note before it is deleted.
		noteID := func(i int, c forge.FakeComment) int {
			if n, err := strconv.Atoi(c.ID); err == nil {
				return n
			}
			return id*1000 + i + 1
		}
		at := func(n int) int {
			for i, c := range it.Comments {
				if noteID(i, c) == n {
					return i
				}
			}
			return -1
		}
		switch {
		case method == "GET":
			out := []map[string]any{}
			for i, c := range it.Comments {
				out = append(out, map[string]any{"id": noteID(i, c), "body": c.Body, "system": false, "author": user(c.Author)})
			}
			for i, t := range it.Threads { // a discussion's note is a note too, as GitLab lists it
				out = append(out, map[string]any{"id": id*1000 + 500 + i + 1, "body": t.Body, "system": false, "author": user(tokenUser), "type": "DiffNote"})
			}
			if it.AutoMergeAt != "" { // GitLab's own note when a merge request is set to merge
				out = append(out, map[string]any{"id": id*1000 + 900, "body": "enabled an automatic merge when the pipeline for 1a2b3c4d succeeds",
					"system": true, "created_at": it.AutoMergeAt, "author": user(it.AutoMergeBy)})
			}
			if q.Get("sort") != "asc" { // GitLab's default: the newest first
				slices.Reverse(out)
			}
			return out, 200
		case method == "POST":
			c := forge.FakeComment{Body: form.Get("body"), Author: tokenUser}
			n := id*1000 + len(it.Comments) + 1
			if k := len(it.Comments); k > 0 && it.Comments[k-1].ID != "" {
				n = noteID(k-1, it.Comments[k-1]) + 1
				c.ID = strconv.Itoa(n)
			}
			it.Comments = append(it.Comments, c)
			it.Posted++ // each mails the issue's followers
			return map[string]any{"id": n}, 201
		case method == "PUT":
			n, _ := strconv.Atoi(sm[3])
			i := at(n)
			if i < 0 {
				return notFound, 404
			}
			if a := it.Comments[i].Author; a != "" && a != tokenUser {
				return map[string]any{"message": "403 Forbidden"}, 403 // a note is its author's
			}
			it.Comments[i] = forge.FakeComment{Body: form.Get("body"), Author: tokenUser, ID: it.Comments[i].ID}
			return map[string]any{"id": n}, 200
		case method == "DELETE":
			n, _ := strconv.Atoi(sm[3])
			i := at(n)
			if i < 0 {
				return notFound, 404
			}
			if a := it.Comments[i].Author; a != "" && a != tokenUser {
				return map[string]any{"message": "403 Forbidden"}, 403 // a note is its author's, or a Maintainer's to delete
			}
			for k := range it.Comments {
				it.Comments[k].ID = strconv.Itoa(noteID(k, it.Comments[k]))
			}
			it.Comments = slices.Delete(it.Comments, i, i+1)
			return nil, 204
		}
	case mrPath.MatchString(p):
		sm := mrPath.FindStringSubmatch(p)
		id, _ := strconv.Atoi(sm[1])
		it := mergeRequest(id)
		if it == nil {
			return notFound, 404
		}
		return m.mergeRequest(s, it, sm[2], method, form)
	case closedBy.MatchString(p):
		// The merge requests that closed it, merged; refs "!7", id 5000+7.
		id, _ := strconv.Atoi(closedBy.FindStringSubmatch(p)[1])
		it := issue(id)
		if it == nil {
			return notFound, 404
		}
		out := []map[string]any{}
		for _, c := range it.ClosedBy {
			if n, err := strconv.Atoi(strings.TrimPrefix(c.Ref, "!")); err == nil && c.Kind == "pull-request" {
				title, desc, _ := strings.Cut(c.Text, "\n\n")
				out = append(out, map[string]any{"id": 5000 + n, "iid": n, "title": title, "description": desc, "state": "merged"})
			}
		}
		return out, 200
	case stateEvents.MatchString(p):
		// Its closings, the last naming what closed it, as GitLab's state
		// events do; a closed issue with no closer was closed by hand.
		id, _ := strconv.Atoi(stateEvents.FindStringSubmatch(p)[1])
		it := issue(id)
		if it == nil {
			return notFound, 404
		}
		out := []map[string]any{}
		if it.Closed {
			e := map[string]any{"state": "closed", "source_commit": nil, "source_merge_request_id": nil}
			if it.Closing != nil {
				e["user"] = user(it.Closing.Author)
			}
			for _, c := range it.ClosedBy {
				if n, err := strconv.Atoi(strings.TrimPrefix(c.Ref, "!")); err == nil && c.Kind == "pull-request" {
					e["source_merge_request_id"] = 5000 + n
				} else if c.Kind == "commit" {
					e["source_commit"] = c.Ref
				}
			}
			out = append(out, e)
		}
		return out, 200
	case commit.MatchString(p):
		sha := commit.FindStringSubmatch(p)[1]
		for _, it := range s.Issues {
			for _, c := range it.ClosedBy {
				if c.Kind == "commit" && c.Ref == sha {
					return map[string]any{"id": sha, "message": c.Text}, 200
				}
			}
		}
		return notFound, 404
	case linksPath.MatchString(p) && method == "GET":
		return []any{}, 200
	case linksPath.MatchString(p) && method == "POST":
		// GitLab Free: no blocks nor is_blocked_by (ADR-0028).
		return map[string]any{"message": "Blocked issues not available for current license"}, 403
	case p == "/milestones" && method == "GET":
		out := []map[string]any{}
		for i, t := range s.Milestones {
			out = append(out, map[string]any{"id": i + 1, "title": t})
		}
		return out, 200
	case p == "/milestones" && method == "POST":
		s.Milestones = append(s.Milestones, form.Get("title"))
		return map[string]any{"id": len(s.Milestones)}, 201
	case strings.HasPrefix(p, "/labels/") && method == "GET":
		name, _ := url.PathUnescape(strings.TrimPrefix(p, "/labels/"))
		for _, l := range s.Labels {
			if l.Title == name {
				return map[string]any{"name": name}, 200
			}
		}
		return notFound, 404
	case p == "/labels" && method == "POST":
		rank, _ := strconv.Atoi(form.Get("priority")) // a prioritized label's rank
		s.Labels = append(s.Labels, forge.FakeItem{ID: len(s.Labels) + 1, Title: form.Get("name"), Body: form.Get("description"), Labels: []string{}, Rank: rank})
		return map[string]any{"name": form.Get("name")}, 201
	case p == "/labels" && method == "GET":
		// The project's labels: those defined, and those set on an issue,
		// as GitLab creates one set by name.
		var names []string
		for _, l := range s.Labels {
			names = append(names, l.Title)
		}
		for _, it := range s.Issues {
			names = append(names, it.Labels...)
		}
		out := []map[string]any{}
		for i, n := range names {
			if !slices.Contains(names[:i], n) {
				out = append(out, map[string]any{"name": n})
			}
		}
		return out, 200
	}
	return notFound, 404
}

// mergeRequest answers a merge request — closed by PUT state_event=close —,
// its diffs and its discussions; its notes are notesPath's.
func (m *gitlabMock) mergeRequest(s *gitlabState, it *forge.FakeItem, part, method string, form url.Values) (any, int) {
	tokenUser := s.tokenUser()
	base, head := it.Base, it.Branch
	if base == "" {
		base = "main"
	}
	if head == "" {
		head = "HEAD"
	}
	git := func(args ...string) string {
		out, _ := exec.Command("git", append([]string{"-C", m.repo}, args...)...).Output()
		return strings.TrimSpace(string(out))
	}
	web := fmt.Sprintf("https://gitlab.example/group/proj/-/merge_requests/%d", it.ID)
	// The head GitLab names: the branch's tip on origin, as a push left it.
	tip := git("ls-remote", "origin", "refs/heads/"+it.Branch)
	tip, _, _ = strings.Cut(tip, "\t")
	if tip == "" {
		tip = git("rev-parse", head)
	}
	asMR := func() map[string]any {
		source := 1
		if it.Fork {
			source = 2
		}
		state := map[bool]string{true: "closed", false: "opened"}[it.Closed]
		if it.Merged {
			state = "merged"
		}
		v := map[string]any{"iid": it.ID, "title": it.Title, "description": it.Body, "source_branch": it.Branch, "target_branch": it.Base,
			"source_project_id": source, "target_project_id": 1, "web_url": web, "state": state, "labels": it.Labels,
			"sha": tip, "draft": it.Draft, "merge_when_pipeline_succeeds": it.AutoMerge,
			"diff_refs": map[string]any{"base_sha": git("merge-base", base, head), "start_sha": git("rev-parse", base), "head_sha": git("rev-parse", head)}}
		if it.AutoMerge {
			id := 0 // a person's, named by the author's id when it is the author
			if it.AutoMergeBy == tokenUser {
				id = 999
			}
			v["merge_user"] = map[string]any{"id": id, "username": it.AutoMergeBy}
		}
		if it.Checks != "" {
			v["head_pipeline"] = map[string]any{"status": map[string]string{"success": "success", "pending": "running", "failure": "failed"}[it.Checks]}
		}
		return v
	}
	switch {
	case part == "" && (method == "GET" || method == "PUT"):
		if method == "PUT" && form.Get("state_event") == "close" {
			it.Closed = true
		}
		for _, l := range strings.Split(form.Get("add_labels"), ",") {
			if l != "" && !slices.Contains(it.Labels, l) {
				it.Labels = append(it.Labels, l)
			}
		}
		if remove := strings.Split(form.Get("remove_labels"), ","); form.Has("remove_labels") {
			it.Labels = slices.DeleteFunc(it.Labels, func(l string) bool { return slices.Contains(remove, l) })
			if it.Labels == nil {
				it.Labels = []string{}
			}
		}
		return asMR(), 200
	case part == "/merge" && method == "PUT":
		// Set to merge when its pipeline succeeds, at the head given; one
		// GitLab could merge now is merged at once.
		if form.Get("sha") != tip {
			return map[string]any{"message": "SHA does not match HEAD of source branch"}, 409
		}
		if it.Clean {
			it.Closed, it.Merged = true, true
		} else if form.Get("auto_merge") == "true" {
			it.AutoMerge, it.AutoMergeBy, it.AutoMergeAt = true, tokenUser, time.Now().UTC().Format(time.RFC3339)
		}
		return asMR(), 200
	case part == "/cancel_merge_when_pipeline_succeeds" && method == "POST":
		it.AutoMerge, it.AutoMergeBy = false, ""
		return asMR(), 201
	case part == "/versions" && method == "GET":
		out := []map[string]any{} // newest first, as GitLab lists them
		for i := len(it.Pushes) - 1; i >= 0; i-- {
			out = append(out, map[string]any{"id": i + 1, "head_commit_sha": git("rev-parse", it.Pushes[i].Head), "created_at": it.Pushes[i].At})
		}
		return out, 200
	case part == "/resource_label_events" && method == "GET":
		out := []map[string]any{}
		for _, e := range it.LabelEvents {
			out = append(out, map[string]any{"action": map[bool]string{true: "add", false: "remove"}[e.Added], "created_at": e.Created,
				"label": map[string]any{"name": e.Label}, "user": map[string]any{"username": e.Author}})
		}
		return out, 200
	case part == "/diffs" && method == "GET":
		return diffsOf(git("diff", "--no-color", "--no-ext-diff", "-M", "-U3", base+"..."+head)), 200
	case part == "/discussions" && method == "GET":
		out := []map[string]any{}
		for i, t := range it.Threads {
			out = append(out, map[string]any{"id": fmt.Sprintf("d%d", i+1), "notes": []map[string]any{{"id": it.ID*1000 + 500 + i + 1, "body": t.Body,
				"resolvable": true, "resolved": t.Resolved, "position": map[string]any{"new_path": t.Path, "new_line": t.Line}}}})
		}
		return out, 200
	case part == "/discussions" && method == "POST":
		if s.ReviewRefused != "" {
			return map[string]any{"message": "403 Forbidden"}, 403
		}
		path, line, old := form.Get("position[new_path]"), form.Get("position[new_line]"), form.Get("position[old_line]")
		n, _ := strconv.Atoi(line)
		shown := false
		for _, d := range diffsOf(git("diff", "--no-color", "--no-ext-diff", "-M", "-U3", base+"..."+head)) {
			if d["new_path"] == path {
				kept, ok := newLines(fmt.Sprint(d["diff"]))[n]
				shown = ok && (kept == 0 && old == "" || kept > 0 && old == strconv.Itoa(kept))
			}
		}
		if form.Get("position[head_sha]") != git("rev-parse", head) || !shown {
			path, n = "", 0 // tried on gitlab.com: a discussion on no line, said by nothing
		}
		it.Threads = append(it.Threads, forge.FakeThread{Path: path, Line: n, Body: form.Get("body")})
		return map[string]any{"id": fmt.Sprintf("d%d", len(it.Threads))}, 201
	}
	return map[string]any{"message": "404 Not Found"}, 404
}

// diffsOf splits a git diff by file, as GitLab's diffs give it: each
// file's paths and its hunks.
func diffsOf(out string) []map[string]any {
	diffs := []map[string]any{}
	var cur map[string]any
	var hunks []string
	header := false
	flush := func() {
		if cur != nil {
			cur["diff"] = strings.Join(hunks, "\n") + "\n"
			diffs = append(diffs, cur)
		}
		cur, hunks = nil, nil
	}
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			flush()
			cur, header = map[string]any{}, true
		case header && strings.HasPrefix(l, "--- "):
			cur["old_path"] = strings.TrimPrefix(strings.TrimPrefix(l, "--- "), "a/")
		case header && strings.HasPrefix(l, "+++ "):
			cur["new_path"] = strings.TrimPrefix(strings.TrimPrefix(l, "+++ "), "b/")
		case strings.HasPrefix(l, "@@"):
			header = false
			hunks = append(hunks, l)
		case !header && cur != nil:
			hunks = append(hunks, l)
		}
	}
	flush()
	return diffs
}

// newLines are the new lines a file's hunks show, each with its old line
// when kept, 0 when added.
func newLines(diff string) map[int]int {
	lines := map[int]int{}
	o, n := 0, 0
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, "@@") {
			fmt.Sscanf(l, "@@ -%d", &o)
			if i := strings.Index(l, "+"); i > 0 {
				fmt.Sscanf(l[i:], "+%d", &n)
			}
			continue
		}
		switch {
		case strings.HasPrefix(l, "+"):
			lines[n] = 0
			n++
		case strings.HasPrefix(l, "-"):
			o++
		case strings.HasPrefix(l, " "):
			lines[n] = o
			n, o = n+1, o+1
		}
	}
	return lines
}
