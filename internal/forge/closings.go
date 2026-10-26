package forge

import (
	"net/url"
	"slices"
	"strings"
)

// Closings is a forge that says how the last merge request from a branch
// was closed: a role tells there a close of workline's own, its comment
// carrying a marker — `workline follow`'s carries Marker("closed") —, from
// a person's (#592).
type Closings interface {
	// ClosedWith says whether the last merge request from branch was
	// closed without merging, a comment on it carrying marker; false when
	// none is from branch, or the last is open or merged.
	ClosedWith(branch, marker string) (bool, error)
}

// GitHub, GitLab, the local and the simulated forge say it; a forge
// plugged by a command does not.
var _ = []Closings{&github{}, &gitlab{}, &Local{}, &Fake{}}

// ClosedWith reads the last pull request from branch, GitHub listing the
// newest first, and its comments when it was closed unmerged.
func (g *github) ClosedWith(branch, marker string) (bool, error) {
	out, err := g.api("repos/{owner}/{repo}/pulls?state=all&head={owner}:"+url.QueryEscape(branch)+"&per_page=1",
		"--jq", `.[] | {id: .number, state: .state, merged: (.merged_at // "")}`)
	if err != nil {
		return false, err
	}
	last, err := lines[struct {
		ID     int    `json:"id"`
		State  string `json:"state"`
		Merged string `json:"merged"`
	}](out)
	if err != nil || len(last) == 0 || last[0].State != "closed" || last[0].Merged != "" {
		return false, err
	}
	return g.hasComment(last[0].ID, marker)
}

// ClosedWith reads the last merge request from branch, the newest first,
// and its notes when it was closed: GitLab says a merged one `merged`.
func (g *gitlab) ClosedWith(branch, marker string) (bool, error) {
	out, err := g.api("projects/:id/merge_requests?source_branch=" + url.QueryEscape(branch) + "&order_by=created_at&sort=desc&per_page=1")
	if err != nil {
		return false, err
	}
	var last []struct {
		IID   int    `json:"iid"`
		State string `json:"state"`
	}
	if err := decode(out, &last); err != nil {
		return false, err
	}
	if len(last) == 0 || last[0].State != "closed" {
		return false, nil
	}
	notes, err := g.Comments(Target{Kind: "merge-request", ID: last[0].IID})
	return slices.ContainsFunc(notes, func(n string) bool { return strings.Contains(n, marker) }), err
}

// ClosedWith reads the last merge request kept from branch, by number.
func (l *Local) ClosedWith(branch, marker string) (bool, error) {
	items, err := l.Items("merge-request")
	if err != nil {
		return false, err
	}
	var last *LocalItem
	for _, it := range items {
		if it.Branch == branch && (last == nil || it.ID > last.ID) {
			last = it
		}
	}
	if last == nil || l.StateOf(last) != "closed" {
		return false, nil
	}
	return slices.ContainsFunc(last.Comments, func(c string) bool { return strings.Contains(c, marker) }), nil
}

// ClosedWith reads the last merge request from branch, by number.
func (f *Fake) ClosedWith(branch, marker string) (bool, error) {
	s, err := f.load()
	if err != nil {
		return false, err
	}
	var last *FakeItem
	for i := range s.MergeRequests {
		if m := &s.MergeRequests[i]; m.Branch == branch && (last == nil || m.ID > last.ID) {
			last = m
		}
	}
	if last == nil || !last.Closed || last.Merged {
		return false, nil
	}
	return slices.ContainsFunc(last.Comments, func(c FakeComment) bool { return strings.Contains(c.Body, marker) }), nil
}
