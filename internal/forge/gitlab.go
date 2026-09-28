package forge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// gitlab talks to GitLab through the glab CLI and its REST API. Not yet tried
// against a live instance.
type gitlab struct{ repo string }

func (g *gitlab) api(args ...string) ([]byte, error) {
	return run(g.repo, "glab", append([]string{"api"}, args...)...)
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
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	if v.Labels == nil {
		v.Labels = []string{}
	}
	return &Issue{ID: v.IID, Title: v.Title, Body: v.Description, Labels: v.Labels}, nil
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
	out, err := g.api("--paginate", path(t)+"/notes?per_page=100")
	if err != nil {
		return err
	}
	type note struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
	}
	var notes []note
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); { // one array a page
		var page []note
		if err := dec.Decode(&page); err != nil {
			return fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
		}
		notes = append(notes, page...)
	}
	for _, n := range notes {
		if strings.Contains(n.Body, marker) {
			_, err = g.api("-X", "PUT", fmt.Sprintf("%s/notes/%d", path(t), n.ID), "-f", "body="+body+"\n\n"+marker)
			return err
		}
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

func (g *gitlab) Release(tag, notes string) error {
	if _, err := g.api("projects/:id/releases/" + url.PathEscape(tag)); err == nil {
		return nil
	} else if !errors.Is(err, errNotFound) {
		return err
	}
	_, err := g.api("-X", "POST", "projects/:id/releases", "-f", "tag_name="+tag, "-f", "description="+notes)
	return err
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

func (g *gitlab) OpenMergeRequests(prefix string) (int, error) {
	open, err := g.openMergeRequests()
	if err != nil {
		return 0, err
	}
	n := 0
	for branch := range open {
		if strings.HasPrefix(branch, prefix) {
			n++
		}
	}
	return n, nil
}
