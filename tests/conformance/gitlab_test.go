package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/JN0V/workline/internal/forge"
)

// gitlabMock answers the part of GitLab's REST API the engine's gitlab forge
// calls, over the case's simulated forge file, so a case runs `--forge
// gitlab` and is checked as any other. Its `members` give the project's
// members and their access level, as GitLab's members/all does; what the
// engine writes is written by "workline-bot", the token's user. GraphQL
// refuses all: a split's children are listed in the parent's body. Its
// issue links are GitLab Free's: is_blocked_by refused, for its license.
type gitlabMock struct {
	file string
	mu   sync.Mutex
}

type gitlabState struct {
	forge.FakeState
	Members []struct {
		Username string `json:"username"`
		Level    int    `json:"access_level"`
	} `json:"members,omitempty"`
}

var (
	issuePath = regexp.MustCompile(`^/issues/(\d+)$`)
	notesPath = regexp.MustCompile(`^/issues/(\d+)/notes(?:/(\d+))?$`)
	linksPath = regexp.MustCompile(`^/issues/(\d+)/links$`)
	closedBy  = regexp.MustCompile(`^/issues/(\d+)/closed_by$`)
	commit    = regexp.MustCompile(`^/repository/commits/([0-9a-f]+)$`)
)

const tokenUser = "workline-bot"

func (m *gitlabMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.URL.Path == "/api/graphql" {
		fmt.Fprint(w, `{"errors": [{"message": "not simulated"}]}`)
		return
	}
	data, err := os.ReadFile(m.file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var s gitlabState
	if err := json.Unmarshal(data, &s); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(body))
	p := strings.TrimPrefix(r.URL.Path, "/api/v4/projects/1")
	answer, status := m.serve(&s, r.Method, p, r.URL.Query(), form)
	if r.Method != "GET" && status < 300 {
		out, _ := json.MarshalIndent(s, "", "  ")
		os.WriteFile(m.file, out, 0o644)
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(answer)
}

func (m *gitlabMock) serve(s *gitlabState, method, p string, q, form url.Values) (any, int) {
	issue := func(id int) *forge.FakeItem {
		for i := range s.Issues {
			if s.Issues[i].ID == id {
				return &s.Issues[i]
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
		return v
	}
	notFound := map[string]any{"message": "404 Not Found"}
	switch {
	case p == "/members/all":
		return s.Members, 200
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
		}
		return asIssue(*it), 200
	case notesPath.MatchString(p):
		sm := notesPath.FindStringSubmatch(p)
		id, _ := strconv.Atoi(sm[1])
		it := issue(id)
		if it == nil {
			return notFound, 404
		}
		switch {
		case method == "GET":
			out := []map[string]any{}
			for i, c := range it.Comments {
				out = append(out, map[string]any{"id": id*1000 + i + 1, "body": c.Body, "system": false, "author": user(c.Author)})
			}
			for i, c := range it.ClosedBy { // GitLab's own note for a commit that closed it
				if c.Kind == "commit" {
					out = append(out, map[string]any{"id": id*1000 + 900 + i, "body": "closed via commit " + c.Ref, "system": true, "author": user("owner")})
				}
			}
			if q.Get("sort") != "asc" { // GitLab's default: the newest first
				slices.Reverse(out)
			}
			return out, 200
		case method == "POST":
			it.Comments = append(it.Comments, forge.FakeComment{Body: form.Get("body"), Author: tokenUser})
			return map[string]any{"id": id*1000 + len(it.Comments)}, 201
		case method == "PUT":
			n, _ := strconv.Atoi(sm[2])
			i := n - id*1000 - 1
			if i < 0 || i >= len(it.Comments) {
				return notFound, 404
			}
			if a := it.Comments[i].Author; a != "" && a != tokenUser {
				return map[string]any{"message": "403 Forbidden"}, 403 // a note is its author's
			}
			it.Comments[i] = forge.FakeComment{Body: form.Get("body"), Author: tokenUser}
			return map[string]any{"id": n}, 200
		}
	case closedBy.MatchString(p):
		// The merge requests that closed it, merged; refs "!7".
		id, _ := strconv.Atoi(closedBy.FindStringSubmatch(p)[1])
		it := issue(id)
		if it == nil {
			return notFound, 404
		}
		out := []map[string]any{}
		for _, c := range it.ClosedBy {
			if n, err := strconv.Atoi(strings.TrimPrefix(c.Ref, "!")); err == nil && c.Kind == "pull-request" {
				title, desc, _ := strings.Cut(c.Text, "\n\n")
				out = append(out, map[string]any{"iid": n, "title": title, "description": desc, "state": "merged"})
			}
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
		s.Labels = append(s.Labels, forge.FakeItem{ID: len(s.Labels) + 1, Title: form.Get("name"), Body: form.Get("description"), Labels: []string{}})
		return map[string]any{"name": form.Get("name")}, 201
	}
	return notFound, 404
}
