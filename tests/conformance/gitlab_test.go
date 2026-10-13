package conformance

import (
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
// members and their access level, as GitLab's members/all does; what the
// engine writes is written by "workline-bot", the token's user. GraphQL
// refuses all: a split's children are listed in the parent's body. Its
// issue links are GitLab Free's: is_blocked_by refused, for its license.
// A merge request's diff is the repository's, from its base (main) to its
// branch (HEAD), as GitLab computes it; a discussion on a line it does not
// show, or a kept line without its old one, is taken on no line, as
// gitlab.com does (#277).
type gitlabMock struct {
	file, repo string
	mu         sync.Mutex
}

type gitlabState struct {
	forge.FakeState
	Members []struct {
		Username string `json:"username"`
		Level    int    `json:"access_level"`
	} `json:"members,omitempty"`
	// Bots are the users the users API says are bots (`bot`), a service
	// account's among them; a merge request's author is user 1000 + its id.
	Bots []string `json:"bots,omitempty"`
}

var (
	issuePath   = regexp.MustCompile(`^/issues/(\d+)$`)
	notesPath   = regexp.MustCompile(`^/(issues|merge_requests)/(\d+)/notes(?:/(\d+))?$`)
	mrPath      = regexp.MustCompile(`^/merge_requests/(\d+)(/diffs|/discussions|/merge|/cancel_merge_when_pipeline_succeeds|/versions|/resource_label_events)?$`)
	linksPath   = regexp.MustCompile(`^/issues/(\d+)/links$`)
	closedBy    = regexp.MustCompile(`^/issues/(\d+)/closed_by$`)
	stateEvents = regexp.MustCompile(`^/issues/(\d+)/resource_state_events$`)
	commit      = regexp.MustCompile(`^/repository/commits/([0-9a-f]+)$`)
	userPath    = regexp.MustCompile(`^/api/v4/users/(\d+)$`) // outside the project's path
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
		return v
	}
	notFound := map[string]any{"message": "404 Not Found"}
	switch {
	case p == "/members/all":
		return s.Members, 200
	case p == "/merge_requests" && method == "GET":
		out := []map[string]any{}
		for _, it := range s.MergeRequests {
			if q.Get("state") == "opened" && it.Closed || q.Has("target_branch") && q.Get("target_branch") != it.Base {
				continue
			}
			source := 1 // a fork's is another project's
			if it.Fork {
				source = 2
			}
			out = append(out, map[string]any{"iid": it.ID, "source_branch": it.Branch, "target_branch": it.Base,
				"source_project_id": source, "target_project_id": 1, "title": it.Title,
				"author": map[string]any{"id": 1000 + it.ID, "username": it.Author},
				"state":  map[bool]string{true: "closed", false: "opened"}[it.Closed],
				"labels": it.Labels, "draft": it.Draft, "merge_when_pipeline_succeeds": it.AutoMerge})
		}
		return out, 200
	case p == "/api/v4/users/999", p == "/api/v4/user":
		// The token's user, who sets a merge request to merge (merge_user):
		// a bot, as a project access token's is.
		return map[string]any{"id": 999, "username": tokenUser, "bot": true}, 200
	case p == "/events" && method == "GET":
		// The project's push events, newest first: each merge request's
		// pushes, by whom the simulated forge says.
		out := []map[string]any{}
		for _, it := range s.MergeRequests {
			for _, push := range it.Pushes {
				out = append(out, map[string]any{"action_name": "pushed to", "author_username": push.By, "created_at": push.At,
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
		switch {
		case method == "GET":
			out := []map[string]any{}
			for i, c := range it.Comments {
				out = append(out, map[string]any{"id": id*1000 + i + 1, "body": c.Body, "system": false, "author": user(c.Author)})
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
			it.Comments = append(it.Comments, forge.FakeComment{Body: form.Get("body"), Author: tokenUser})
			return map[string]any{"id": id*1000 + len(it.Comments)}, 201
		case method == "PUT":
			n, _ := strconv.Atoi(sm[3])
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
		s.Labels = append(s.Labels, forge.FakeItem{ID: len(s.Labels) + 1, Title: form.Get("name"), Body: form.Get("description"), Labels: []string{}})
		return map[string]any{"name": form.Get("name")}, 201
	}
	return notFound, 404
}

// mergeRequest answers a merge request — closed by PUT state_event=close —,
// its diffs and its discussions; its notes are notesPath's.
func (m *gitlabMock) mergeRequest(s *gitlabState, it *forge.FakeItem, part, method string, form url.Values) (any, int) {
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
