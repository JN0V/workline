package forge

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// github talks to GitHub through the gh CLI and its REST API. Issues and pull
// requests share numbers, comments and labels there, so both use the issue
// endpoints. Tried live on 2026-09-24: comment, label added (GitHub creates a
// missing label), label removed, and every write replayed without duplicates.
// Not yet tried live: OpenIssue.
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

func (g *github) Sticky(t Target, body, marker string, create bool) error {
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments", t.ID),
		"--jq", fmt.Sprintf(".[] | select(.body | contains(%q)) | .id", marker))
	if err != nil {
		return err
	}
	if id := strings.Fields(string(out)); len(id) > 0 {
		_, err = g.api("-X", "PATCH", "repos/{owner}/{repo}/issues/comments/"+id[0], "-f", "body="+body+"\n\n"+marker)
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
