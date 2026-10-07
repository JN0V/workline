package forge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// gitlab talks to GitLab's REST API itself, with nothing to install: on
// gitlab.com or an instance of one's own, in CI or on a machine. Tried on
// gitlab.com (roles/documentalist/docs/tried.md).
type gitlab struct {
	repo                  string
	base, project, header string // the API's root, the project's id, how the token goes
	token                 string
	members               map[string]int // the project's members, direct and inherited, and their access level; read once
}

// planner is GitLab's access level from which a member counts as a person
// of the project (Issue.Insider): Planner (15), Reporter, Developer,
// Maintainer, Owner — who may set a label, so accept a draft with
// workline:accepted, as GitHub's owner, member or collaborator. A Guest
// (10) or a Minimal Access member (5) is outside, as anyone not a member.
const planner = 15

// botName matches the users of project and group access tokens:
// project_<id>_bot_<random>, group_<id>_bot_<random> (GitLab's naming).
var botName = regexp.MustCompile(`^(project|group)_\d+_bot(_|$)`)

// insider says whether username is a person of the project. The members
// are read once a run: a token that may not list them fails loud rather
// than taking every reporter for an outsider.
func (g *gitlab) insider(username string) (bool, error) {
	if g.members == nil {
		out, err := g.api("--paginate", "projects/:id/members/all?per_page=100")
		if err != nil {
			return false, fmt.Errorf("reading the project's members, to tell who is of the project: %w", err)
		}
		all, err := pages[struct {
			Username string `json:"username"`
			Level    int    `json:"access_level"`
			State    string `json:"state"`
		}](out)
		if err != nil {
			return false, err
		}
		g.members = map[string]int{}
		for _, m := range all {
			if m.State == "" || m.State == "active" {
				g.members[m.Username] = max(g.members[m.Username], m.Level)
			}
		}
	}
	return username != "" && g.members[username] >= planner, nil
}

// connect finds the instance, the project and the token: what CI gives
// (CI_API_V4_URL, CI_PROJECT_ID), else GITLAB_HOST and the repository's
// remote; the token in GITLAB_TOKEN, else glab's if it is set up, else the
// job's own, which may read but not write.
func (g *gitlab) connect() error {
	if g.base != "" {
		return nil
	}
	remoteAPI, remoteProject, _ := fromRemote(gitRemote(g.repo))
	g.base = strings.TrimSuffix(os.Getenv("CI_API_V4_URL"), "/")
	if g.base == "" {
		if h := strings.TrimSuffix(os.Getenv("GITLAB_HOST"), "/"); h != "" {
			if !strings.Contains(h, "://") {
				h = "https://" + h
			}
			g.base = h + "/api/v4"
		} else {
			g.base = remoteAPI
		}
	}
	switch {
	case os.Getenv("CI_PROJECT_ID") != "":
		g.project = os.Getenv("CI_PROJECT_ID")
	case os.Getenv("CI_PROJECT_PATH") != "":
		g.project = url.PathEscape(os.Getenv("CI_PROJECT_PATH"))
	default:
		g.project = remoteProject
	}
	if g.base == "" || g.project == "" {
		return fmt.Errorf("%w: no GitLab project here: no CI_API_V4_URL and CI_PROJECT_ID, and no remote naming one", ErrUnreachable)
	}
	g.header, g.token = "PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN")
	if g.token == "" {
		if u, err := url.Parse(g.base); err == nil {
			if out, err := exec.Command("glab", "config", "get", "token", "--host", u.Host).Output(); err == nil {
				g.token = strings.TrimSpace(string(out))
			}
		}
	}
	if g.token == "" && os.Getenv("CI_JOB_TOKEN") != "" {
		g.header, g.token = "JOB-TOKEN", os.Getenv("CI_JOB_TOKEN")
	}
	if g.token == "" {
		return fmt.Errorf("%w: no GitLab token: set GITLAB_TOKEN", ErrUnreachable)
	}
	return nil
}

// api calls the REST API as `glab api` would, the subset this forge uses:
// [--paginate] [-X METHOD] path [-f field=value]…, `:id` in the path the
// project. Paginated, the pages' arrays follow one another.
func (g *gitlab) api(args ...string) ([]byte, error) {
	if err := g.connect(); err != nil {
		return nil, err
	}
	method, path, paginate := "GET", "", false
	form := url.Values{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paginate":
			paginate = true
		case "-X":
			i++
			method = args[i]
		case "-f":
			i++
			k, v, _ := strings.Cut(args[i], "=")
			form.Add(k, v)
		default:
			path = args[i]
		}
	}
	next := g.base + "/" + strings.Replace(path, ":id", g.project, 1)
	var all []byte
	for next != "" {
		var body io.Reader
		if len(form) > 0 {
			body = strings.NewReader(form.Encode())
		}
		req, err := http.NewRequest(method, next, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set(g.header, g.token)
		if body != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
		}
		switch {
		case resp.StatusCode == http.StatusNotFound:
			return nil, fmt.Errorf("%w: %s %s", errNotFound, method, path)
		case resp.StatusCode == http.StatusForbidden:
			return nil, fmt.Errorf("%w: GitLab %s %s: %s %s (the token's role or scope does not allow it)", errForbidden, method, path, resp.Status, strings.TrimSpace(string(data)))
		case resp.StatusCode >= 300:
			return nil, fmt.Errorf("%w: GitLab %s %s: %s %s", ErrUnreachable, method, path, resp.Status, strings.TrimSpace(string(data)))
		}
		all = append(all, data...)
		next = ""
		if page := resp.Header.Get("X-Next-Page"); paginate && page != "" {
			u, _ := url.Parse(req.URL.String())
			q := u.Query()
			q.Set("page", page)
			u.RawQuery = q.Encode()
			next = u.String()
		}
	}
	return all, nil
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// gitRemote is the URL of the repository's origin, or "".
func gitRemote(repo string) string {
	out, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// fromRemote reads a remote's URL — https, ssh:// or scp-like — as the API
// of its instance, over https, and the project's path, escaped. An ssh port
// is not the API's.
func fromRemote(remote string) (api, project string, ok bool) {
	var host, p string
	switch {
	case strings.Contains(remote, "://"):
		u, err := url.Parse(remote)
		if err != nil {
			return "", "", false
		}
		host, p = u.Host, u.Path
		if u.Scheme == "ssh" {
			host = u.Hostname()
		}
	case strings.Contains(remote, ":"):
		at := strings.LastIndex(remote, "@")
		host, p, _ = strings.Cut(remote[at+1:], ":")
	default:
		return "", "", false
	}
	p = strings.TrimSuffix(strings.Trim(p, "/"), ".git")
	if host == "" || p == "" {
		return "", "", false
	}
	return "https://" + host + "/api/v4", url.PathEscape(p), true
}

func path(t Target) string {
	if t.Kind == "merge-request" {
		return fmt.Sprintf("projects/:id/merge_requests/%d", t.ID)
	}
	return fmt.Sprintf("projects/:id/issues/%d", t.ID)
}

func (g *gitlab) Issue(id int) (*Issue, error) {
	out, err := g.api(path(Target{Kind: "issue", ID: id}))
	if err != nil {
		return nil, err
	}
	var v struct {
		IID         int      `json:"iid"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Labels      []string `json:"labels"`
		State       string   `json:"state"`
		Author      struct {
			Username string `json:"username"`
		} `json:"author"`
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	if v.Labels == nil {
		v.Labels = []string{}
	}
	in, err := g.insider(v.Author.Username)
	return &Issue{ID: v.IID, Title: v.Title, Body: v.Description, Labels: v.Labels, Closed: v.State == "closed",
		Author: v.Author.Username, Insider: in}, err
}

// pages decodes the arrays a paginated call prints, one a page.
func pages[T any](out []byte) ([]T, error) {
	var all []T
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); {
		var page []T
		if err := dec.Decode(&page); err != nil {
			return nil, fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
		}
		all = append(all, page...)
	}
	return all, nil
}

func (g *gitlab) Issues() ([]Issue, error) { return g.issues("opened") }

// AllIssues: GitLab keeps no close reason; a closed issue's is left empty.
func (g *gitlab) AllIssues() ([]Issue, error) { return g.issues("all") }

func (g *gitlab) issues(state string) ([]Issue, error) {
	out, err := g.api("--paginate", "projects/:id/issues?state="+state+"&per_page=100")
	if err != nil {
		return nil, err
	}
	found, err := pages[struct {
		IID         int      `json:"iid"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Labels      []string `json:"labels"`
		Milestone   *struct {
			Title string `json:"title"`
			Due   string `json:"due_date"`
		} `json:"milestone"`
		Author struct {
			Username string `json:"username"`
		} `json:"author"`
		State string `json:"state"`
	}](out)
	var all []Issue
	for _, f := range found {
		if f.Labels == nil {
			f.Labels = []string{}
		}
		// Who is of the project is not in the issue: its author's access
		// level, from the project's members.
		in, err := g.insider(f.Author.Username)
		if err != nil {
			return nil, err
		}
		is := Issue{ID: f.IID, Title: f.Title, Body: f.Description, Labels: f.Labels, Author: f.Author.Username, Insider: in, Closed: f.State == "closed"}
		if f.Milestone != nil {
			is.Milestone, is.MilestoneDue = f.Milestone.Title, f.Milestone.Due
		}
		all = append(all, is)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	if err != nil || state != "opened" {
		return all, err
	}
	blocked, err := g.blockedBy()
	if err != nil {
		return all, err
	}
	children, err := g.children()
	for i := range all {
		all[i].BlockedBy = blocked[all[i].ID]
		all[i].Children = children[all[i].ID]
	}
	return all, err
}

// children reads each open issue's tasks, open or closed, in the work
// items' hierarchy, in one GraphQL query a page of a hundred (ADR-0029).
// A GitLab that answers it with errors — an instance without work items —
// has none: a parent's body lists its children.
func (g *gitlab) children() (map[int][]int, error) {
	out := map[int][]int{}
	after := ""
	for {
		var v struct {
			Project struct {
				WorkItems struct {
					Nodes []struct {
						IID     string `json:"iid"`
						Widgets []struct {
							Children *struct {
								Nodes []struct {
									IID string `json:"iid"`
								} `json:"nodes"`
								PageInfo struct {
									HasNextPage bool `json:"hasNextPage"`
								} `json:"pageInfo"`
							} `json:"children"`
						} `json:"widgets"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"workItems"`
			} `json:"project"`
		}
		vars := map[string]any{"p": g.projectPath()}
		if after != "" {
			vars["after"] = after
		}
		ok, err := g.graphql(`query($p: ID!, $after: String) { project(fullPath: $p) { workItems(state: opened, first: 100, after: $after) { nodes { iid widgets { ... on WorkItemWidgetHierarchy { children(first: 100) { nodes { iid } pageInfo { hasNextPage } } } } } pageInfo { hasNextPage endCursor } } } }`, vars, &v)
		if err != nil || !ok {
			return out, err
		}
		for _, n := range v.Project.WorkItems.Nodes {
			id, _ := strconv.Atoi(n.IID)
			for _, w := range n.Widgets {
				if w.Children == nil {
					continue
				}
				if w.Children.PageInfo.HasNextPage {
					return out, fmt.Errorf("#%d has more than 100 tasks: GitLab's children are read a hundred at most", id)
				}
				for _, c := range w.Children.Nodes {
					if cid, err := strconv.Atoi(c.IID); err == nil {
						out[id] = append(out[id], cid)
					}
				}
			}
		}
		if !v.Project.WorkItems.PageInfo.HasNextPage {
			return out, nil
		}
		after = v.Project.WorkItems.PageInfo.EndCursor
	}
}

// Closers reads what closed the issue last, from its state events: a
// commit pushed with a closing pattern (`source_commit`, its message read)
// or a merge request merged (`source_merge_request_id`, found among the
// merge requests closed_by lists, with its title and description). An
// issue closed by hand has none. GitLab writes no note for either: tried
// on gitlab.com, 2026-10-05.
func (g *gitlab) Closers(id int) ([]Closer, error) {
	t := Target{Kind: "issue", ID: id}
	out, err := g.api("--paginate", path(t)+"/resource_state_events?per_page=100")
	if err != nil {
		return nil, err
	}
	events, err := pages[struct {
		State  string `json:"state"`
		Commit string `json:"source_commit"`
		MR     int    `json:"source_merge_request_id"`
	}](out)
	if err != nil {
		return nil, err
	}
	commit, mr := "", 0 // the last closing's
	for _, e := range events {
		if e.State == "closed" {
			commit, mr = e.Commit, e.MR
		}
	}
	switch {
	case commit != "":
		c := Closer{Kind: "commit", Ref: commit[:min(8, len(commit))]}
		if out, err := g.api("projects/:id/repository/commits/" + commit); err == nil {
			var v struct {
				Message string `json:"message"`
			}
			if decode(out, &v) == nil {
				c.Text = v.Message
			}
		}
		return []Closer{c}, nil
	case mr != 0:
		out, err := g.api(path(t) + "/closed_by")
		if err != nil {
			return nil, err
		}
		var mrs []struct {
			ID          int    `json:"id"`
			IID         int    `json:"iid"`
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := decode(out, &mrs); err != nil {
			return nil, err
		}
		for _, m := range mrs {
			if m.ID == mr {
				return []Closer{{Kind: "pull-request", Ref: fmt.Sprintf("!%d", m.IID), Text: strings.TrimSpace(m.Title + "\n\n" + m.Description)}}, nil
			}
		}
	}
	return nil, nil
}

// blockedBy reads what each open issue waits on in GitLab's is_blocked_by
// links, in one GraphQL query a page of a hundred: empty on Free, where the
// links are not available (ADR-0028). A GitLab that answers the query
// with errors — an instance without the field — has none: the bodies say
// what an issue waits on.
func (g *gitlab) blockedBy() (map[int][]int, error) {
	out := map[int][]int{}
	after := ""
	for {
		var v struct {
			Project struct {
				Issues struct {
					Nodes []struct {
						IID             string `json:"iid"`
						BlockedByIssues struct {
							Nodes []struct {
								IID string `json:"iid"`
							} `json:"nodes"`
							PageInfo struct {
								HasNextPage bool `json:"hasNextPage"`
							} `json:"pageInfo"`
						} `json:"blockedByIssues"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"issues"`
			} `json:"project"`
		}
		vars := map[string]any{"p": g.projectPath()}
		if after != "" {
			vars["after"] = after
		}
		ok, err := g.graphql(`query($p: ID!, $after: String) { project(fullPath: $p) { issues(state: opened, first: 100, after: $after) { nodes { iid blockedByIssues(first: 100) { nodes { iid } pageInfo { hasNextPage } } } pageInfo { hasNextPage endCursor } } } }`, vars, &v)
		if err != nil || !ok {
			return out, err
		}
		for _, n := range v.Project.Issues.Nodes {
			id, _ := strconv.Atoi(n.IID)
			if n.BlockedByIssues.PageInfo.HasNextPage {
				// Never an issue offered first for blockers left unread.
				return out, fmt.Errorf("#%d waits on more than 100 issues: GitLab's blockers are read a hundred at most", id)
			}
			for _, b := range n.BlockedByIssues.Nodes {
				if bid, err := strconv.Atoi(b.IID); err == nil {
					out[id] = append(out[id], bid)
				}
			}
		}
		if !v.Project.Issues.PageInfo.HasNextPage {
			return out, nil
		}
		after = v.Project.Issues.PageInfo.EndCursor
	}
}

// AddBlocker links id to blocker as is_blocked_by, one already there left.
// GitLab Free refuses the link type (403, "not available for current
// license"), as an instance without links does (404): false, and the body
// says it instead (ADR-0028).
func (g *gitlab) AddBlocker(id, blocker int) (bool, error) {
	out, err := g.api(path(Target{Kind: "issue", ID: id}) + "/links")
	if errors.Is(err, errNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var links []struct {
		IID  int    `json:"iid"`
		Type string `json:"link_type"`
	}
	if err := decode(out, &links); err != nil {
		return false, err
	}
	for _, l := range links {
		if l.IID == blocker && l.Type == "is_blocked_by" {
			return true, nil
		}
	}
	_, err = g.api("-X", "POST", path(Target{Kind: "issue", ID: id})+"/links", "-f", "target_project_id="+g.projectPath(),
		"-f", "target_issue_iid="+strconv.Itoa(blocker), "-f", "link_type=is_blocked_by")
	switch {
	case errors.Is(err, errNotFound), errors.Is(err, errForbidden) && strings.Contains(err.Error(), "license"):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

func (g *gitlab) Comments(t Target) ([]string, error) {
	notes, err := g.Notes(t)
	return Bodies(notes), err
}

// Notes reads each note's author, of the project by their access level; a
// project or group access token's user is a bot.
func (g *gitlab) Notes(t Target) ([]Note, error) {
	out, err := g.api("--paginate", path(t)+"/notes?sort=asc&order_by=created_at&per_page=100")
	if err != nil {
		return nil, err
	}
	found, err := pages[struct {
		Body    string `json:"body"`
		System  bool   `json:"system"`
		Created string `json:"created_at"`
		Author  struct {
			Username string `json:"username"`
		} `json:"author"`
	}](out)
	if err != nil {
		return nil, err
	}
	var all []Note
	for _, n := range found {
		if n.System { // GitLab's own notes: "changed the description", …
			continue
		}
		in, err := g.insider(n.Author.Username)
		if err != nil {
			return nil, err
		}
		all = append(all, Note{Body: n.Body, Author: n.Author.Username, Insider: in, Bot: botName.MatchString(n.Author.Username), Created: n.Created})
	}
	return all, nil
}

// mentioned reads GitLab's system notes naming an issue from elsewhere:
// "mentioned in commit 1a2b3c4d", "mentioned in merge request !7", with
// another project's path before the reference when it is another's —
// "group/other@1a2b3c4d", "group/other!7" —, kept in the reference so it
// is never taken for this project's.
var mentioned = regexp.MustCompile(`^mentioned in (?:(commit) (\S*?[0-9a-f]{7,40})|(merge request) (\S*?![0-9]+))$`)

// Trail reads when the issue last got the label, from its label events,
// and what names it, from its system notes: two listings (ADR-0031).
func (g *gitlab) Trail(id int, label string) (Trail, error) {
	t := Target{Kind: "issue", ID: id}
	out, err := g.api("--paginate", path(t)+"/resource_label_events?per_page=100")
	if err != nil {
		return Trail{}, err
	}
	events, err := pages[struct {
		Action  string `json:"action"`
		Created string `json:"created_at"`
		Label   *struct {
			Name string `json:"name"`
		} `json:"label"`
	}](out)
	if err != nil {
		return Trail{}, err
	}
	var tr Trail
	for _, e := range events {
		if e.Action == "add" && e.Label != nil && e.Label.Name == label {
			tr.Labeled = e.Created
		}
	}
	out, err = g.api("--paginate", path(t)+"/notes?sort=asc&order_by=created_at&per_page=100")
	if err != nil {
		return Trail{}, err
	}
	notes, err := pages[struct {
		Body    string `json:"body"`
		System  bool   `json:"system"`
		Created string `json:"created_at"`
	}](out)
	if err != nil {
		return Trail{}, err
	}
	for _, n := range notes {
		m := mentioned.FindStringSubmatch(strings.TrimSpace(n.Body))
		if !n.System || m == nil {
			continue
		}
		l := Link{Kind: "commit", Ref: m[2], At: n.Created}
		if m[3] != "" {
			l.Kind, l.Ref = "pull-request", m[4]
		}
		tr.Links = append(tr.Links, l)
	}
	return tr, nil
}

// Close closes an issue; a duplicate through GitLab's own quick action,
// which closes it and links the original.
func (g *gitlab) Close(id, dup int) error {
	t := Target{Kind: "issue", ID: id}
	if dup > 0 {
		_, err := g.api("-X", "POST", path(t)+"/notes", "-f", fmt.Sprintf("body=/duplicate #%d", dup))
		return err
	}
	_, err := g.api("-X", "PUT", path(t), "-f", "state_event=close")
	return err
}

func (g *gitlab) Comment(t Target, body, marker string) error {
	out, err := g.api("--paginate", path(t)+"/notes?per_page=100")
	if err != nil {
		return err
	}
	if strings.Contains(string(out), marker) {
		return nil
	}
	_, err = g.api("-X", "POST", path(t)+"/notes", "-f", "body="+body+"\n\n"+marker)
	return err
}

func (g *gitlab) Sticky(t Target, body, marker string, create bool) error {
	// Oldest first: GitLab lists the newest first unless asked.
	out, err := g.api("--paginate", path(t)+"/notes?sort=asc&order_by=created_at&per_page=100")
	if err != nil {
		return err
	}
	notes, err := pages[struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
	}](out)
	if err != nil {
		return err
	}
	// The last carrying the marker, the one read (backlog's readBlock).
	// GitLab lets only its author, or a Maintainer, edit a note: one
	// another token wrote — a person's, before the project's bot took over
	// — is left, and a new one, this token's, carries the marker after it.
	for i := len(notes) - 1; i >= 0; i-- {
		if !strings.Contains(notes[i].Body, marker) {
			continue
		}
		if notes[i].Body == body+"\n\n"+marker {
			return nil // as it is already: not edited again
		}
		_, err = g.api("-X", "PUT", fmt.Sprintf("%s/notes/%d", path(t), notes[i].ID), "-f", "body="+body+"\n\n"+marker)
		if !errors.Is(err, errForbidden) {
			return err
		}
		create = true
		break
	}
	if !create {
		return nil
	}
	_, err = g.api("-X", "POST", path(t)+"/notes", "-f", "body="+body+"\n\n"+marker)
	return err
}

func (g *gitlab) Label(t Target, add, remove []string) error {
	args := []string{"-X", "PUT", path(t)}
	if len(add) > 0 {
		args = append(args, "-f", "add_labels="+strings.Join(add, ","))
	}
	if len(remove) > 0 {
		args = append(args, "-f", "remove_labels="+strings.Join(remove, ","))
	}
	_, err := g.api(args...)
	return err
}

func (g *gitlab) OpenIssue(title, body, marker string) (int, error) {
	out, err := g.api("projects/:id/issues?state=opened&in=title&per_page=100&search=" + url.QueryEscape(title))
	if err != nil {
		return 0, err
	}
	var found []struct {
		IID   int    `json:"iid"`
		Title string `json:"title"`
	}
	if err := decode(out, &found); err != nil {
		return 0, err
	}
	for _, f := range found {
		if f.Title == title {
			return f.IID, g.Comment(Target{Kind: "issue", ID: f.IID}, body, marker)
		}
	}
	out, err = g.api("-X", "POST", "projects/:id/issues", "-f", "title="+title, "-f", "description="+body+"\n\n"+marker)
	if err != nil {
		return 0, err
	}
	var created struct {
		IID int `json:"iid"`
	}
	return created.IID, decode(out, &created)
}

// openMergeRequests lists the open merge requests as iid by source branch.
func (g *gitlab) openMergeRequests() (map[string]int, error) {
	out, err := g.api("--paginate", "projects/:id/merge_requests?state=opened&per_page=100")
	if err != nil {
		return nil, err
	}
	type mr struct {
		IID    int    `json:"iid"`
		Source string `json:"source_branch"`
	}
	open := map[string]int{}
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); { // one array a page
		var page []mr
		if err := dec.Decode(&page); err != nil {
			return nil, fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
		}
		for _, m := range page {
			open[m.Source] = m.IID
		}
	}
	return open, nil
}

func (g *gitlab) OpenMergeRequest(branch, base, title, body string) (int, error) {
	open, err := g.openMergeRequests()
	if err != nil {
		return 0, err
	}
	if id, ok := open[branch]; ok {
		_, err := g.api("-X", "PUT", fmt.Sprintf("projects/:id/merge_requests/%d", id), "-f", "title="+title, "-f", "description="+body)
		return id, err
	}
	out, err := g.api("-X", "POST", "projects/:id/merge_requests", "-f", "source_branch="+branch, "-f", "target_branch="+base,
		"-f", "title="+title, "-f", "description="+body, "-f", "remove_source_branch=true")
	if err != nil {
		return 0, err
	}
	var created struct {
		IID int `json:"iid"`
	}
	if err := decode(out, &created); err != nil {
		return 0, err
	}
	return created.IID, nil
}

func (g *gitlab) OpenMergeRequests(prefix string) ([]string, error) {
	open, err := g.openMergeRequests()
	if err != nil {
		return nil, err
	}
	var out []string
	for branch := range open {
		if strings.HasPrefix(branch, prefix) {
			out = append(out, branch)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (g *gitlab) MergeRequest(id int) (MergeRequest, error) {
	out, err := g.api(path(Target{Kind: "merge-request", ID: id}))
	if err != nil {
		return MergeRequest{}, err
	}
	var mr struct {
		Source        string `json:"source_branch"`
		Target        string `json:"target_branch"`
		SourceProject int    `json:"source_project_id"`
		TargetProject int    `json:"target_project_id"`
		Title         string `json:"title"`
		Description   string `json:"description"`
	}
	if err := decode(out, &mr); err != nil {
		return MergeRequest{}, err
	}
	return MergeRequest{Branch: mr.Source, Base: mr.Target, Here: mr.SourceProject == mr.TargetProject, Title: mr.Title, Body: mr.Description}, nil
}

func (g *gitlab) KeepIssue(title, body string, create bool) (int, error) {
	out, err := g.api("projects/:id/issues?state=opened&in=title&per_page=100&search=" + url.QueryEscape(title))
	if err != nil {
		return 0, err
	}
	var found []struct {
		IID   int    `json:"iid"`
		Title string `json:"title"`
	}
	if err := decode(out, &found); err != nil {
		return 0, err
	}
	for _, f := range found {
		if f.Title == title {
			_, err := g.api("-X", "PUT", fmt.Sprintf("projects/:id/issues/%d", f.IID), "-f", "description="+body)
			return f.IID, err
		}
	}
	if !create {
		return 0, nil
	}
	out, err = g.api("-X", "POST", "projects/:id/issues", "-f", "title="+title, "-f", "description="+body)
	if err != nil {
		return 0, err
	}
	var created struct {
		IID int `json:"iid"`
	}
	return created.IID, decode(out, &created)
}

// milestoneIDs maps the active milestones' titles to their ids.
func (g *gitlab) milestoneIDs() (map[string]int, error) {
	all, err := g.milestones()
	m := map[string]int{}
	for _, x := range all {
		m[x.Title] = x.ID
	}
	return m, err
}

type gitlabMilestone struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Due   string `json:"due_date"`
}

// milestones lists the active milestones, each due on a day or not.
func (g *gitlab) milestones() ([]gitlabMilestone, error) {
	out, err := g.api("--paginate", "projects/:id/milestones?state=active&per_page=100")
	if err != nil {
		return nil, err
	}
	return pages[gitlabMilestone](out)
}

func (g *gitlab) Milestones() ([]Milestone, error) {
	all, err := g.milestones()
	var out []Milestone
	for _, x := range all {
		out = append(out, Milestone{Title: x.Title, Due: x.Due})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, err
}

func (g *gitlab) EnsureLabel(name, color, description string) error {
	_, err := g.api("projects/:id/labels/" + url.PathEscape(name))
	if !errors.Is(err, errNotFound) {
		return err
	}
	_, err = g.api("-X", "POST", "projects/:id/labels", "-f", "name="+name, "-f", "color=#"+color, "-f", "description="+description)
	return err
}

func (g *gitlab) SetBody(id int, body string) error {
	_, err := g.api("-X", "PUT", path(Target{Kind: "issue", ID: id}), "-f", "description="+body)
	return err
}

func (g *gitlab) SetTitle(id int, title string) error {
	_, err := g.api("-X", "PUT", path(Target{Kind: "issue", ID: id}), "-f", "title="+title)
	return err
}

// AddSubIssue makes child a task of parent: an issue's children on GitLab
// are tasks, a work item type the REST API still lists, labels and
// comments as an issue (ADR-0022). The hierarchy is the work items'
// GraphQL API's only: the child is converted to a task, then given its
// parent. A GitLab that refuses either — an instance without work items —
// answers false, and the parent's body lists the child instead.
func (g *gitlab) AddSubIssue(parent, child int) (bool, error) {
	ids := map[int]string{}
	for _, iid := range []int{parent, child} {
		out, err := g.api(path(Target{Kind: "issue", ID: iid}))
		if err != nil {
			return false, err
		}
		var v struct {
			ID   int    `json:"id"`
			Type string `json:"issue_type"`
		}
		if err := decode(out, &v); err != nil {
			return false, err
		}
		ids[iid] = fmt.Sprintf("gid://gitlab/WorkItem/%d", v.ID)
		if iid == child && v.Type != "task" {
			// The project's Task type, then the conversion.
			var types struct {
				Project struct {
					WorkItemTypes struct {
						Nodes []struct{ ID, Name string }
					}
				}
			}
			ok, err := g.graphql(`query($p: ID!) { project(fullPath: $p) { workItemTypes(name: TASK) { nodes { id name } } } }`,
				map[string]any{"p": g.projectPath()}, &types)
			if err != nil || !ok || len(types.Project.WorkItemTypes.Nodes) == 0 {
				return false, err
			}
			var conv struct {
				WorkItemConvert struct{ Errors []string }
			}
			ok, err = g.graphql(`mutation($id: WorkItemID!, $t: WorkItemsTypeID!) { workItemConvert(input: {id: $id, workItemTypeId: $t}) { errors } }`,
				map[string]any{"id": ids[iid], "t": types.Project.WorkItemTypes.Nodes[0].ID}, &conv)
			if err != nil || !ok || len(conv.WorkItemConvert.Errors) > 0 {
				return false, err
			}
		}
	}
	var up struct {
		WorkItemUpdate struct{ Errors []string }
	}
	ok, err := g.graphql(`mutation($id: WorkItemID!, $p: WorkItemID!) { workItemUpdate(input: {id: $id, hierarchyWidget: {parentId: $p}}) { errors } }`,
		map[string]any{"id": ids[child], "p": ids[parent]}, &up)
	return err == nil && ok && len(up.WorkItemUpdate.Errors) == 0, err
}

// projectPath is the project's full path, for GraphQL, which takes no
// numeric id.
func (g *gitlab) projectPath() string {
	if p := os.Getenv("CI_PROJECT_PATH"); p != "" {
		return p
	}
	p, _ := url.PathUnescape(g.project)
	return p
}

// graphql runs one query on GitLab's GraphQL API and decodes its data into
// v; ok is false when GitLab answered with errors — a refusal, said by the
// caller's fallback — and err when it could not be reached.
func (g *gitlab) graphql(query string, vars map[string]any, v any) (bool, error) {
	if err := g.connect(); err != nil {
		return false, err
	}
	payload, _ := json.Marshal(map[string]any{"query": query, "variables": vars})
	req, err := http.NewRequest("POST", strings.TrimSuffix(g.base, "/v4")+"/graphql", bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	req.Header.Set(g.header, g.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("%w: GitLab GraphQL: %s %s", ErrUnreachable, resp.Status, strings.TrimSpace(string(data)))
	}
	var answer struct {
		Data   json.RawMessage `json:"data"`
		Errors []any           `json:"errors"`
	}
	if err := decode(data, &answer); err != nil {
		return false, err
	}
	if len(answer.Errors) > 0 || len(answer.Data) == 0 || string(answer.Data) == "null" {
		return false, nil
	}
	return true, decode(answer.Data, v)
}

func (g *gitlab) SetMilestone(id int, title string) error {
	m, err := g.milestoneIDs()
	if err != nil {
		return err
	}
	n, ok := m[title]
	if !ok {
		out, err := g.api("-X", "POST", "projects/:id/milestones", "-f", "title="+title)
		if err != nil {
			return err
		}
		var created struct {
			ID int `json:"id"`
		}
		if err := decode(out, &created); err != nil {
			return err
		}
		n = created.ID
	}
	_, err = g.api("-X", "PUT", path(Target{Kind: "issue", ID: id}), "-f", fmt.Sprintf("milestone_id=%d", n))
	return err
}

// taskNote is GitLab's system note for a box ticked or unticked in a
// description — "marked the checklist item **…** as completed" (formerly
// "the task"), written whether the box was ticked on the page or the
// description edited through the API; the item's markdown escaped, its
// hidden comments' text kept.
var taskNote = regexp.MustCompile(`(?s)^marked the (?:checklist item|task) \*\*(.*)\*\* as (completed|incomplete)$`)

// escaped is a character GitLab's note escaped: `\#`, `\=`, `\-`.
var escaped = regexp.MustCompile(`\\(.)`)

// Ticks reads the boxes ticked from the issue's system notes, each with
// its author, of the project by their access level.
func (g *gitlab) Ticks(id int) ([]Tick, error) {
	out, err := g.api("--paginate", path(Target{Kind: "issue", ID: id})+"/notes?sort=asc&order_by=created_at&per_page=100")
	if err != nil {
		return nil, err
	}
	found, err := pages[struct {
		Body   string `json:"body"`
		System bool   `json:"system"`
		Author struct {
			Username string `json:"username"`
		} `json:"author"`
	}](out)
	if err != nil {
		return nil, err
	}
	var ticks []Tick
	for _, n := range found {
		m := taskNote.FindStringSubmatch(strings.TrimSpace(n.Body))
		if !n.System || m == nil {
			continue
		}
		in, err := g.insider(n.Author.Username)
		if err != nil {
			return nil, err
		}
		ticks = append(ticks, Tick{Item: escaped.ReplaceAllString(m[1], "$1"), Done: m[2] == "completed",
			Note: Note{Author: n.Author.Username, Insider: in, Bot: botName.MatchString(n.Author.Username)}})
	}
	return ticks, nil
}
