package forge

import (
	"errors"
	"fmt"
	"strings"
)

// github talks to GitHub through the gh CLI and its REST API. Issues and pull
// requests share numbers, comments and labels there, so both use the issue
// endpoints. Tried live on 2026-09-24: comment, label added (GitHub creates a
// missing label), label removed, and every write replayed without duplicates.
// Not yet tried live: OpenIssue and Release.
type github struct{ repo string }

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
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	is := &Issue{ID: v.Number, Title: v.Title, Body: v.Body, Labels: []string{}}
	for _, l := range v.Labels {
		is.Labels = append(is.Labels, l.Name)
	}
	return is, nil
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

func (g *github) Release(tag, notes string) error {
	if _, err := g.api("repos/{owner}/{repo}/releases/tags/" + tag); err == nil {
		return nil // already published
	} else if !errors.Is(err, errNotFound) {
		return err
	}
	_, err := g.api("-X", "POST", "repos/{owner}/{repo}/releases", "-f", "tag_name="+tag, "-f", "name="+tag, "-f", "body="+notes)
	return err
}
