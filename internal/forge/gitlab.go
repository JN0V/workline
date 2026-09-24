package forge

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// gitlab talks to GitLab through the glab CLI and its REST API. Not yet tried
// against a live instance: the first use at work is its first real test.
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
