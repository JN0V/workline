package forge

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
)

// Woken is a forge the wake script reads after a merge (ADR-0068 §8):
// the issues each merge request closes, as the forge links them, and the
// open issues and merge requests bearing a label. GitHub, GitLab and the
// simulated forge.
type Woken interface {
	// Closes lists, by merge request, the issues the forge links it to
	// close, each with its labels and whether it is closed now.
	Closes(mrs []int) (map[int][]Issue, error)
	// Labelled lists the open issues and merge requests bearing label.
	Labelled(label string) ([]Target, error)
}

// AppStatuses is a forge that says whether an app set a status on a
// commit: workline's App alone sets workline/accepted (GitHub, ADR-0068
// §6). GitLab keeps none: its protection holds the merge.
type AppStatuses interface {
	AppStatus(head, name string) (bool, error)
}

var (
	_ = []Woken{&github{}, &gitlab{}, &Fake{}}
	_ = []AppStatuses{&github{}, &Fake{}}
)

// closesChunk is how many pull requests one GraphQL call asks of.
const closesChunk = 50

// Closes asks GitHub each pull request's closing references, 50 a call:
// the issues it closes as GitHub links them, from its description's
// keywords or by hand; another repository's left out.
func (g *github) Closes(mrs []int) (map[int][]Issue, error) {
	out := map[int][]Issue{}
	for start := 0; start < len(mrs); start += closesChunk {
		chunk := mrs[start:min(start+closesChunk, len(mrs))]
		var q strings.Builder
		q.WriteString(`query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { nameWithOwner `)
		for _, id := range chunk {
			fmt.Fprintf(&q, `p%d: pullRequest(number: %d) { closingIssuesReferences(first: 20) { nodes { number state repository { nameWithOwner } labels(first: 30) { nodes { name } } } } } `, id, id)
		}
		q.WriteString(`} }`)
		data, err := g.api("graphql", "-f", "query="+q.String(), "-F", "owner={owner}", "-F", "name={repo}")
		if err != nil {
			return out, err
		}
		type node struct {
			Number     int    `json:"number"`
			State      string `json:"state"`
			Repository struct {
				Name string `json:"nameWithOwner"`
			} `json:"repository"`
			Labels struct {
				Nodes []struct {
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"labels"`
		}
		var v struct {
			Data struct {
				Repository map[string]json.RawMessage `json:"repository"`
			} `json:"data"`
		}
		if err := decode(data, &v); err != nil {
			return out, err
		}
		var here string
		_ = json.Unmarshal(v.Data.Repository["nameWithOwner"], &here)
		for _, id := range chunk {
			raw := v.Data.Repository[fmt.Sprintf("p%d", id)]
			var pr struct {
				Refs struct {
					Nodes []node `json:"nodes"`
				} `json:"closingIssuesReferences"`
			}
			if raw == nil || string(raw) == "null" || decode(raw, &pr) != nil {
				// A merge unread is never taken for one closing nothing.
				return out, fmt.Errorf("%w: GitHub gave no closing links for #%d", ErrUnreachable, id)
			}
			for _, n := range pr.Refs.Nodes {
				if here != "" && n.Repository.Name != "" && n.Repository.Name != here {
					continue
				}
				is := Issue{ID: n.Number, Closed: n.State == "CLOSED", Labels: []string{}}
				for _, l := range n.Labels.Nodes {
					is.Labels = append(is.Labels, l.Name)
				}
				out[id] = append(out[id], is)
			}
		}
	}
	return out, nil
}

// Labelled lists GitHub's open issues and pull requests bearing label:
// one listing, the issues' endpoint holding both.
func (g *github) Labelled(label string) ([]Target, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/issues?state=open&per_page=100&labels="+url.QueryEscape(label),
		"--jq", `.[] | {id: .number, pr: (.pull_request != null)}`)
	if err != nil {
		return nil, err
	}
	found, err := lines[struct {
		ID int  `json:"id"`
		PR bool `json:"pr"`
	}](out)
	if err != nil {
		return nil, err
	}
	var all []Target
	for _, f := range found {
		kind := "issue"
		if f.PR {
			kind = "merge-request"
		}
		all = append(all, Target{Kind: kind, ID: f.ID})
	}
	return all, nil
}

// AppStatus says whether workline's App set the status name on head,
// successful: GitHub names the one who set it, and only the App the
// workflow names (WORKLINE_APP) counts; another app's is no yes. With no
// App named, it is not read: the wake's reader is not the one who set it.
func (g *github) AppStatus(head, name string) (bool, error) {
	app := strings.TrimSpace(os.Getenv("WORKLINE_APP"))
	if app == "" {
		return false, fmt.Errorf("%w: no App named: give the job workline's App slug in WORKLINE_APP", ErrUnreachable)
	}
	out, err := g.api("--paginate", "repos/{owner}/{repo}/commits/"+head+"/statuses?per_page=100",
		"--jq", fmt.Sprintf(`.[] | select(.context == %q and .state == "success" and .creator.login == %q) | .id`, name, app+"[bot]"))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// Closes asks GitLab each merge request's closes_issues: one call each.
func (g *gitlab) Closes(mrs []int) (map[int][]Issue, error) {
	out := map[int][]Issue{}
	for _, id := range mrs {
		data, err := g.api("--paginate", path(Target{Kind: "merge-request", ID: id})+"/closes_issues?per_page=100")
		if err != nil {
			return out, err
		}
		found, err := pages[struct {
			IID    int      `json:"iid"`
			State  string   `json:"state"`
			Labels []string `json:"labels"`
		}](data)
		if err != nil {
			return out, err
		}
		for _, f := range found {
			out[id] = append(out[id], Issue{ID: f.IID, Closed: f.State == "closed", Labels: append([]string{}, f.Labels...)})
		}
	}
	return out, nil
}

// Labelled lists GitLab's open issues, then its open merge requests,
// bearing label.
func (g *gitlab) Labelled(label string) ([]Target, error) {
	var all []Target
	for _, kind := range []string{"issue", "merge-request"} {
		list := "issues"
		if kind == "merge-request" {
			list = "merge_requests"
		}
		data, err := g.api("--paginate", "projects/:id/"+list+"?state=opened&per_page=100&labels="+url.QueryEscape(label))
		if err != nil {
			return all, err
		}
		found, err := pages[struct {
			IID int `json:"iid"`
		}](data)
		if err != nil {
			return all, err
		}
		for _, f := range found {
			all = append(all, Target{Kind: kind, ID: f.IID})
		}
	}
	return all, nil
}

// Closes reads each merge request's `closes` in the file, each issue as
// the file holds it.
func (f *Fake) Closes(mrs []int) (map[int][]Issue, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	out := map[int][]Issue{}
	for _, m := range s.MergeRequests {
		if !slices.Contains(mrs, m.ID) {
			continue
		}
		for _, id := range m.Closes {
			is := Issue{ID: id, Labels: []string{}}
			if it, err := s.item(Target{Kind: "issue", ID: id}); err == nil {
				is.Closed, is.Labels = it.Closed, append(is.Labels, it.Labels...)
			}
			out[m.ID] = append(out[m.ID], is)
		}
	}
	return out, nil
}

// Labelled lists the file's open issues, then its open merge requests,
// bearing label.
func (f *Fake) Labelled(label string) ([]Target, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var all []Target
	for _, it := range s.Issues {
		if !it.Closed && slices.Contains(it.Labels, label) {
			all = append(all, Target{Kind: "issue", ID: it.ID})
		}
	}
	for _, it := range s.MergeRequests {
		if !it.Closed && !it.Merged && slices.Contains(it.Labels, label) {
			all = append(all, Target{Kind: "merge-request", ID: it.ID})
		}
	}
	return all, nil
}

// AppStatus: a status of that name on head, successful, set by an app
// (`workline-bot`, or a name ending in [bot], as GitHub names an app).
func (f *Fake) AppStatus(head, name string) (bool, error) {
	s, err := f.load()
	if err != nil {
		return false, err
	}
	for _, st := range s.Statuses {
		if st.Name == name && st.State == "success" && (st.By == fakeBot || strings.HasSuffix(st.By, "[bot]")) && f.resolve(st.Commit) == head {
			return true, nil
		}
	}
	return false, nil
}
