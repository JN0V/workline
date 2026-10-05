package forge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// github talks to GitHub through the gh CLI and its REST API. Issues and pull
// requests share numbers, comments and labels there, so both use the issue
// endpoints. Tried live on 2026-09-24: comment, label added (GitHub creates a
// missing label), label removed, and every write replayed without duplicates.
// Not yet tried live: OpenIssue.
type github struct {
	repo    string
	writers map[string]bool  // who may write to the repository, read once each
	unread  map[string]error // whose rights the token could not read, asked once each
}

func (g *github) api(args ...string) ([]byte, error) {
	return run(g.repo, "gh", append([]string{"api"}, args...)...)
}

func (g *github) Issue(id int) (*Issue, error) {
	out, err := g.api(fmt.Sprintf("repos/{owner}/{repo}/issues/%d", id))
	if err != nil {
		return nil, err
	}
	var v struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Association string `json:"author_association"`
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	is := &Issue{ID: v.Number, Title: v.Title, Body: v.Body, Labels: []string{}, Closed: v.State == "closed",
		Author: v.User.Login, Insider: insider(v.Association)}
	for _, l := range v.Labels {
		is.Labels = append(is.Labels, l.Name)
	}
	return is, nil
}

// lines decodes the JSON values gh prints one after another, as --jq
// prints them for every page.
func lines[T any](out []byte) ([]T, error) {
	var all []T
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); {
		var v T
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
		}
		all = append(all, v)
	}
	return all, nil
}

func (g *github) Issues() ([]Issue, error) { return g.issues("open") }

func (g *github) AllIssues() ([]Issue, error) { return g.issues("all") }

func (g *github) issues(state string) ([]Issue, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/issues?state="+state+"&per_page=100",
		"--jq", ".[] | select(.pull_request == null) | {id: .number, title, body: (.body // \"\"), labels: [.labels[].name], milestone: (.milestone.title // \"\"), author: .user.login, association: .author_association, closed: (.state == \"closed\"), reason: (.state_reason // \"\"), blocked: (.issue_dependencies_summary.total_blocked_by // 0), subs: (.sub_issues_summary.total // 0), repo: .repository_url}")
	if err != nil {
		return nil, err
	}
	found, err := lines[struct {
		Issue
		Association string `json:"association"`
		Blocked     int    `json:"blocked"`
		Subs        int    `json:"subs"`
		Repo        string `json:"repo"`
	}](out)
	if err != nil {
		return nil, err
	}
	var all []Issue
	for _, f := range found {
		f.Issue.Insider = insider(f.Association)
		if f.Blocked > 0 && state == "open" {
			// The listing says how many it waits on; only those are asked
			// which (ADR-0028).
			if f.Issue.BlockedBy, err = g.blockedBy(f.Issue.ID); err != nil {
				return nil, err
			}
		}
		if f.Subs > 0 && state == "open" {
			// The listing says how many sub-issues it has; only those
			// parents are asked which (ADR-0029).
			if f.Issue.Children, err = g.subIssues(f.Issue.ID, f.Repo); err != nil {
				return nil, err
			}
		}
		all = append(all, f.Issue)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all, err
}

// subIssues lists a parent's sub-issues, open or closed, by number; with
// repo (its repository_url), only those of the same repository — GitHub
// allows a sub-issue from another, whose number means another issue here.
func (g *github) subIssues(parent int, repo string) ([]int, error) {
	filter := ".[].number"
	if repo != "" {
		filter = fmt.Sprintf(".[] | select(.repository_url == %q) | .number", repo)
	}
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/sub_issues?per_page=100", parent), "--jq", filter)
	if err != nil {
		return nil, err
	}
	var ids []int
	for _, f := range strings.Fields(string(out)) {
		if n, err := strconv.Atoi(f); err == nil {
			ids = append(ids, n)
		}
	}
	return ids, nil
}

// closersQuery asks what closed an issue last: GitHub's ClosedEvent names
// its closer, the pull request merged or the commit pushed.
const closersQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { issue(number: $number) { timelineItems(itemTypes: [CLOSED_EVENT], last: 1) { nodes { ... on ClosedEvent { closer { __typename ... on PullRequest { number title body } ... on Commit { abbreviatedOid message } } } } } } } }`

// Closers reads the closer of the issue's last closing: one GraphQL call.
// An issue closed by hand has none.
func (g *github) Closers(id int) ([]Closer, error) {
	out, err := g.api("graphql", "-f", "query="+closersQuery, "-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("number=%d", id))
	if err != nil {
		return nil, err
	}
	var v struct {
		Data struct {
			Repository struct {
				Issue struct {
					Timeline struct {
						Nodes []struct {
							Closer *struct {
								Type    string `json:"__typename"`
								Number  int    `json:"number"`
								Title   string `json:"title"`
								Body    string `json:"body"`
								Oid     string `json:"abbreviatedOid"`
								Message string `json:"message"`
							} `json:"closer"`
						} `json:"nodes"`
					} `json:"timelineItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	var all []Closer
	for _, n := range v.Data.Repository.Issue.Timeline.Nodes {
		switch c := n.Closer; {
		case c == nil:
		case c.Type == "PullRequest":
			all = append(all, Closer{Kind: "pull-request", Ref: fmt.Sprintf("#%d", c.Number), Text: strings.TrimSpace(c.Title + "\n\n" + c.Body)})
		case c.Type == "Commit":
			all = append(all, Closer{Kind: "commit", Ref: c.Oid, Text: c.Message})
		}
	}
	return all, nil
}

// trailQuery asks an issue's timeline for its labels set and what names
// it: a pull request linked by hand (ConnectedEvent) or naming it
// (CrossReferencedEvent), a commit naming it (ReferencedEvent), each with
// when (docs/research/product-owner.md, "What is next, and what is stuck").
const trailQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { issue(number: $number) { timelineItems(itemTypes: [LABELED_EVENT, CONNECTED_EVENT, CROSS_REFERENCED_EVENT, REFERENCED_EVENT], last: 100) { nodes { __typename ... on LabeledEvent { createdAt label { name } } ... on ConnectedEvent { createdAt subject { __typename ... on PullRequest { number } } } ... on CrossReferencedEvent { createdAt isCrossRepository source { __typename ... on PullRequest { number repository { nameWithOwner } } } } ... on ReferencedEvent { createdAt commit { abbreviatedOid } } } } } } }`

// Trail reads an issue's last hundred timeline events of those kinds: one
// GraphQL call (ADR-0031).
func (g *github) Trail(id int, label string) (Trail, error) {
	out, err := g.api("graphql", "-f", "query="+trailQuery, "-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("number=%d", id))
	if err != nil {
		return Trail{}, err
	}
	type pr struct {
		Type       string `json:"__typename"`
		Number     int    `json:"number"`
		Repository struct {
			Name string `json:"nameWithOwner"`
		} `json:"repository"`
	}
	var v struct {
		Data struct {
			Repository struct {
				Issue struct {
					Timeline struct {
						Nodes []struct {
							Type      string `json:"__typename"`
							CreatedAt string `json:"createdAt"`
							Cross     bool   `json:"isCrossRepository"`
							Label     *struct {
								Name string `json:"name"`
							} `json:"label"`
							Subject *pr `json:"subject"`
							Source  *pr `json:"source"`
							Commit  *struct {
								Oid string `json:"abbreviatedOid"`
							} `json:"commit"`
						} `json:"nodes"`
					} `json:"timelineItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return Trail{}, err
	}
	var t Trail
	for _, n := range v.Data.Repository.Issue.Timeline.Nodes {
		switch {
		case n.Type == "LabeledEvent" && n.Label != nil && n.Label.Name == label:
			t.Labeled = n.CreatedAt
		case n.Subject != nil && n.Subject.Type == "PullRequest":
			t.Links = append(t.Links, Link{Kind: "pull-request", Ref: fmt.Sprintf("#%d", n.Subject.Number), At: n.CreatedAt})
		case n.Source != nil && n.Source.Type == "PullRequest":
			ref := fmt.Sprintf("#%d", n.Source.Number)
			if n.Cross {
				ref = n.Source.Repository.Name + ref // another repository's, never taken for this one's
			}
			t.Links = append(t.Links, Link{Kind: "pull-request", Ref: ref, At: n.CreatedAt})
		case n.Commit != nil:
			t.Links = append(t.Links, Link{Kind: "commit", Ref: n.Commit.Oid, At: n.CreatedAt})
		}
	}
	return t, nil
}

// insider says whether GitHub's author_association gives write access:
// the owner, a member of the organisation, a collaborator.
func insider(association string) bool {
	return association == "OWNER" || association == "MEMBER" || association == "COLLABORATOR"
}

func (g *github) EnsureLabel(name, color, description string) error {
	_, err := g.api("repos/{owner}/{repo}/labels/" + url.PathEscape(name))
	if !errors.Is(err, errNotFound) {
		return err
	}
	_, err = g.api("-X", "POST", "repos/{owner}/{repo}/labels", "-f", "name="+name, "-f", "color="+color, "-f", "description="+description)
	return err
}

func (g *github) SetBody(id int, body string) error {
	_, err := g.api("-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/issues/%d", id), "-f", "body="+body)
	return err
}

func (g *github) SetTitle(id int, title string) error {
	_, err := g.api("-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/issues/%d", id), "-f", "title="+title)
	return err
}

// AddSubIssue links child under parent with GitHub's sub-issues, which
// take the child's id, not its number; one already linked is left.
func (g *github) AddSubIssue(parent, child int) (bool, error) {
	have, err := g.subIssues(parent, "")
	if err != nil {
		return false, err
	}
	if slices.Contains(have, child) {
		return true, nil
	}
	out, err := g.api(fmt.Sprintf("repos/{owner}/{repo}/issues/%d", child), "--jq", ".id")
	if err != nil {
		return false, err
	}
	_, err = g.api("-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/sub_issues", parent), "-F", "sub_issue_id="+strings.TrimSpace(string(out)))
	return err == nil, err
}

// blockedBy lists the issues id waits on, open or closed, in GitHub's
// issue dependencies.
func (g *github) blockedBy(id int) ([]int, error) {
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/dependencies/blocked_by?per_page=100", id), "--jq", ".[].number")
	if err != nil {
		return nil, err
	}
	var ids []int
	for _, f := range strings.Fields(string(out)) {
		if n, err := strconv.Atoi(f); err == nil {
			ids = append(ids, n)
		}
	}
	return ids, nil
}

// AddBlocker records that id is blocked by blocker in GitHub's issue
// dependencies, which take the blocker's id, not its number; one already
// there is left. A GitHub without them (an older server) answers 404: the
// body says it instead (ADR-0028).
func (g *github) AddBlocker(id, blocker int) (bool, error) {
	have, err := g.blockedBy(id)
	if errors.Is(err, errNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if slices.Contains(have, blocker) {
		return true, nil
	}
	out, err := g.api(fmt.Sprintf("repos/{owner}/{repo}/issues/%d", blocker), "--jq", ".id")
	if err != nil {
		return false, err
	}
	_, err = g.api("-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/dependencies/blocked_by", id), "-F", "issue_id="+strings.TrimSpace(string(out)))
	return err == nil, err
}

func (g *github) Comments(t Target) ([]string, error) {
	notes, err := g.Notes(t)
	return Bodies(notes), err
}

// Notes reads each comment's author and their author_association, as an
// issue's; an app's comment (a user of type Bot) is a bot's.
func (g *github) Notes(t Target) ([]Note, error) {
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments?per_page=100", t.ID),
		"--jq", ".[] | {body: (.body // \"\"), author: .user.login, association: .author_association, bot: (.user.type == \"Bot\"), created: .created_at}")
	if err != nil {
		return nil, err
	}
	found, err := lines[struct {
		Note
		Association string `json:"association"`
	}](out)
	notes := make([]Note, 0, len(found))
	for _, f := range found {
		f.Note.Insider = insider(f.Association)
		notes = append(notes, f.Note)
	}
	return notes, err
}

// Close closes with GitHub's own reason; a duplicate's original is named by
// the engine's comment, "Duplicate of #n", which GitHub links.
func (g *github) Close(id, dup int) error {
	reason := "completed"
	if dup > 0 {
		reason = "duplicate"
	}
	_, err := g.api("-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/issues/%d", id), "-f", "state=closed", "-f", "state_reason="+reason)
	return err
}

func (g *github) hasComment(id int, marker string) (bool, error) {
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments", id), "--jq", ".[].body")
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), marker), nil
}

func (g *github) Comment(t Target, body, marker string) error {
	if found, err := g.hasComment(t.ID, marker); err != nil || found {
		return err
	}
	_, err := g.api("-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments", t.ID), "-f", "body="+body+"\n\n"+marker)
	return err
}

func (g *github) Sticky(t Target, body, marker string, create bool) error {
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments", t.ID),
		"--jq", fmt.Sprintf(".[] | select(.body | contains(%q)) | {id, body}", marker))
	if err != nil {
		return err
	}
	found, err := lines[struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
	}](out)
	if err != nil {
		return err
	}
	if len(found) > 0 {
		if found[0].Body == body+"\n\n"+marker {
			return nil // as it is already: not edited again
		}
		_, err = g.api("-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/issues/comments/%d", found[0].ID), "-f", "body="+body+"\n\n"+marker)
		return err
	}
	if !create {
		return nil
	}
	_, err = g.api("-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments", t.ID), "-f", "body="+body+"\n\n"+marker)
	return err
}

func (g *github) Label(t Target, add, remove []string) error {
	if len(add) > 0 {
		args := []string{"-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/labels", t.ID)}
		for _, l := range add {
			args = append(args, "-f", "labels[]="+l)
		}
		if _, err := g.api(args...); err != nil {
			return err
		}
	}
	for _, l := range remove {
		if _, err := g.api("-X", "DELETE", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/labels/%s", t.ID, l)); err != nil && !errors.Is(err, errNotFound) {
			return err
		}
	}
	return nil
}

func (g *github) OpenIssue(title, body, marker string) (int, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/issues?state=open&per_page=100", "--jq", ".[] | select(.pull_request == null) | [.number, .title] | @tsv")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		num, t, ok := strings.Cut(line, "\t")
		if ok && t == title {
			var id int
			fmt.Sscan(num, &id)
			return id, g.Comment(Target{Kind: "issue", ID: id}, body, marker)
		}
	}
	out, err = g.api("-X", "POST", "repos/{owner}/{repo}/issues", "-f", "title="+title, "-f", "body="+body+"\n\n"+marker, "--jq", ".number")
	if err != nil {
		return 0, err
	}
	var id int
	fmt.Sscan(strings.TrimSpace(string(out)), &id)
	return id, nil
}

// openPulls lists the open pull requests as number and head branch.
func (g *github) openPulls() (map[string]int, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/pulls?state=open&per_page=100", "--jq", ".[] | [.number, .head.ref] | @tsv")
	if err != nil {
		return nil, err
	}
	pulls := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if num, branch, ok := strings.Cut(line, "\t"); ok {
			var id int
			fmt.Sscan(num, &id)
			pulls[branch] = id
		}
	}
	return pulls, nil
}

func (g *github) OpenMergeRequest(branch, base, title, body string) (int, error) {
	pulls, err := g.openPulls()
	if err != nil {
		return 0, err
	}
	if id, ok := pulls[branch]; ok {
		_, err := g.api("-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d", id), "-f", "title="+title, "-f", "body="+body)
		return id, err
	}
	out, err := g.api("-X", "POST", "repos/{owner}/{repo}/pulls", "-f", "title="+title, "-f", "head="+branch, "-f", "base="+base, "-f", "body="+body, "--jq", ".number")
	if err != nil {
		return 0, err
	}
	var id int
	fmt.Sscan(strings.TrimSpace(string(out)), &id)
	return id, nil
}

func (g *github) OpenMergeRequests(prefix string) ([]string, error) {
	pulls, err := g.openPulls()
	if err != nil {
		return nil, err
	}
	var out []string
	for branch := range pulls {
		if strings.HasPrefix(branch, prefix) {
			out = append(out, branch)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (g *github) MergeRequest(id int) (MergeRequest, error) {
	out, err := g.api(fmt.Sprintf("repos/{owner}/{repo}/pulls/%d", id), "--jq", "[.head.ref, .base.ref, (.head.repo.full_name == .base.repo.full_name)] | @tsv")
	if err != nil {
		return MergeRequest{}, err
	}
	f := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(f) != 3 {
		return MergeRequest{}, fmt.Errorf("pull request %d: unexpected answer %q", id, out)
	}
	return MergeRequest{Branch: f[0], Base: f[1], Here: f[2] == "true"}, nil
}

func (g *github) KeepIssue(title, body string, create bool) (int, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/issues?state=open&per_page=100", "--jq", ".[] | select(.pull_request == null) | [.number, .title] | @tsv")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if num, t, ok := strings.Cut(line, "\t"); ok && t == title {
			var id int
			fmt.Sscan(num, &id)
			_, err := g.api("-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/issues/%d", id), "-f", "body="+body)
			return id, err
		}
	}
	if !create {
		return 0, nil
	}
	out, err = g.api("-X", "POST", "repos/{owner}/{repo}/issues", "-f", "title="+title, "-f", "body="+body, "--jq", ".number")
	if err != nil {
		return 0, err
	}
	var id int
	fmt.Sscan(strings.TrimSpace(string(out)), &id)
	return id, nil
}

// milestoneNumbers maps the open milestones' titles to their numbers.
func (g *github) milestoneNumbers() (map[string]int, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/milestones?state=open&per_page=100", "--jq", ".[] | {number, title}")
	if err != nil {
		return nil, err
	}
	all, err := lines[struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}](out)
	m := map[string]int{}
	for _, x := range all {
		m[x.Title] = x.Number
	}
	return m, err
}

func (g *github) Milestones() ([]string, error) {
	m, err := g.milestoneNumbers()
	var out []string
	for t := range m {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, err
}

func (g *github) SetMilestone(id int, title string) error {
	m, err := g.milestoneNumbers()
	if err != nil {
		return err
	}
	n, ok := m[title]
	if !ok {
		out, err := g.api("-X", "POST", "repos/{owner}/{repo}/milestones", "-f", "title="+title, "--jq", ".number")
		if err != nil {
			return err
		}
		fmt.Sscan(strings.TrimSpace(string(out)), &n)
	}
	_, err = g.api("-X", "PATCH", fmt.Sprintf("repos/{owner}/{repo}/issues/%d", id), "-F", fmt.Sprintf("milestone=%d", n))
	return err
}

// ticksQuery reads an issue body's versions, each with who wrote it: GitHub
// keeps no event for a box ticked, its edit history does (userContentEdits,
// each node the whole body as that edit left it, newest first: `first`
// is the newest, `last` the oldest).
const ticksQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) { issue(number: $number) {
    userContentEdits(first: 100) { totalCount nodes { editedAt diff editor { login __typename } } } } } }`

// Ticks reads the boxes ticked from the body's edit history: each version
// against the one before, given to its editor. Only who has write access
// edits a body they did not write (GitHub's roles); the editor is a person
// of the project when GitHub gives them write, maintain or admin.
func (g *github) Ticks(id int) ([]Tick, error) {
	out, err := g.api("graphql", "-f", "query="+ticksQuery, "-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("number=%d", id))
	if err != nil {
		return nil, err
	}
	var v struct {
		Data struct {
			Repository struct {
				Issue struct {
					Edits struct {
						Total int `json:"totalCount"`
						Nodes []struct {
							EditedAt string  `json:"editedAt"`
							Diff     *string `json:"diff"`
							Editor   *struct {
								Login string `json:"login"`
								Type  string `json:"__typename"`
							} `json:"editor"`
						} `json:"nodes"`
					} `json:"userContentEdits"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	edits := v.Data.Repository.Issue.Edits
	nodes := edits.Nodes
	// Oldest first: GitHub gives them newest first, and two edits in the
	// same second keep that order reversed.
	slices.Reverse(nodes)
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].EditedAt < nodes[j].EditedAt })
	var ticks []Tick
	// Older versions not read, or one deleted from the history: the next
	// version is only what the later ones are read against, its boxes
	// nobody's — never a tick credited to whoever edited after the gap.
	before, gap := "", edits.Total > len(nodes)
	for _, n := range nodes {
		if n.Diff == nil {
			gap = true
			continue
		}
		if gap {
			before, gap = *n.Diff, false
			continue
		}
		who := Note{}
		if n.Editor != nil {
			who.Author, who.Bot = n.Editor.Login, n.Editor.Type == "Bot"
			if !who.Bot {
				if who.Insider, err = g.writer(who.Author); err != nil {
					// A token that may not read permissions: who ticked is
					// not known, and the tick is no yes — said, not failed.
					who = Note{}
				}
			}
		}
		ticks = append(ticks, TicksBetween(before, *n.Diff, who)...)
		before = *n.Diff
	}
	return ticks, nil
}

// writer says whether login may write to the repository: a person of the
// project, as an author association of owner, member or collaborator.
func (g *github) writer(login string) (bool, error) {
	if g.writers == nil {
		g.writers = map[string]bool{}
	}
	if w, ok := g.writers[login]; ok {
		return w, nil
	}
	if err := g.unread[login]; err != nil {
		return false, err
	}
	out, err := g.api("repos/{owner}/{repo}/collaborators/"+url.PathEscape(login)+"/permission", "--jq", ".permission")
	if errors.Is(err, errNotFound) {
		out, err = nil, nil
	}
	if err != nil {
		if g.unread == nil {
			g.unread = map[string]error{}
		}
		g.unread[login] = err
		return false, err
	}
	p := strings.TrimSpace(string(out))
	g.writers[login] = p == "admin" || p == "maintain" || p == "write"
	return g.writers[login], nil
}
