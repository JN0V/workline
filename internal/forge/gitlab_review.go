package forge

import (
	"errors"
	"fmt"
	"strings"
)

// Review posts each comment as a discussion on its line of the merge
// request's diff, GitLab's own, which its author answers and resolves
// there. GitLab takes a discussion only on a line its diff shows, a kept
// line given with its old line too: both are read from the merge
// request's diffs first. A discussion refused is said, the others still
// posted; a token that may not comment (403, 401: the job's own) is said
// for every one left.
func (g *gitlab) Review(id int, head, body string, comments []LineComment) (Reviewed, error) {
	return review(g, id, head, body, comments)
}

type gitlabDiff struct {
	Old  string `json:"old_path"`
	New  string `json:"new_path"`
	Diff string `json:"diff"`
}

// shownLines reads the merge request's diffs, page by page; a GitLab
// older than 15.7 has none, and gives its changes instead.
func (g *gitlab) shownLines(id int) (shown, error) {
	mr := path(Target{Kind: "merge-request", ID: id})
	var diffs []gitlabDiff
	out, err := g.api("--paginate", mr+"/diffs?per_page=100")
	if errors.Is(err, errNotFound) {
		out, err = g.api(mr + "/changes")
		if err != nil {
			return shown{}, err
		}
		var v struct {
			Changes []gitlabDiff `json:"changes"`
		}
		if err := decode(out, &v); err != nil {
			return shown{}, err
		}
		diffs = v.Changes
	} else if err != nil {
		return shown{}, err
	} else if diffs, err = pages[gitlabDiff](out); err != nil {
		return shown{}, err
	}
	s := newShown()
	for _, d := range diffs {
		s.add(d.New, d.Old, d.Diff)
	}
	return s, nil
}

// openKeys reads the keys the merge request's discussions not resolved
// hold, in their first note.
func (g *gitlab) openKeys(id int) (map[string]bool, error) {
	out, err := g.api("--paginate", path(Target{Kind: "merge-request", ID: id})+"/discussions?per_page=100")
	if err != nil {
		return nil, err
	}
	discussions, err := pages[struct {
		Notes []struct {
			Body     string `json:"body"`
			Resolved bool   `json:"resolved"`
		} `json:"notes"`
	}](out)
	if err != nil {
		return nil, err
	}
	open := map[string]bool{}
	for _, d := range discussions {
		if len(d.Notes) == 0 || d.Notes[0].Resolved {
			continue
		}
		for _, k := range keysIn(d.Notes[0].Body) {
			open[k] = true
		}
	}
	return open, nil
}

// mergeRequestPage reads the merge request's diff refs, which a position
// names, and its page.
func (g *gitlab) mergeRequestPage(id int) (refs struct{ Base, Start, Head string }, web string, err error) {
	out, err := g.api(path(Target{Kind: "merge-request", ID: id}))
	if err != nil {
		return refs, "", err
	}
	var v struct {
		Web  string `json:"web_url"`
		Refs struct {
			Base  string `json:"base_sha"`
			Start string `json:"start_sha"`
			Head  string `json:"head_sha"`
		} `json:"diff_refs"`
	}
	if err := decode(out, &v); err != nil {
		return refs, "", err
	}
	refs.Base, refs.Start, refs.Head = v.Refs.Base, v.Refs.Start, v.Refs.Head
	return refs, v.Web, nil
}

// postLines posts one discussion a comment, positioned at the merge
// request's latest diff: a line added by its new line, a kept one by both.
// GitLab has no review body: the summary comment is its own.
func (g *gitlab) postLines(id int, _, _ string, at shown, comments []LineComment) (string, []string, error) {
	refs, web, err := g.mergeRequestPage(id)
	if err != nil {
		return "", nil, err
	}
	whys := make([]string, len(comments))
	stop := ""
	for i, c := range comments {
		if stop != "" {
			whys[i] = stop
			continue
		}
		args := []string{"-X", "POST", path(Target{Kind: "merge-request", ID: id}) + "/discussions",
			"-f", "body=" + c.withKey(), "-f", "position[position_type]=text",
			"-f", "position[base_sha]=" + refs.Base, "-f", "position[start_sha]=" + refs.Start, "-f", "position[head_sha]=" + refs.Head,
			"-f", "position[old_path]=" + at.oldPath(c.Path), "-f", "position[new_path]=" + c.Path,
			"-f", fmt.Sprintf("position[new_line]=%d", c.Line)}
		if old, _ := at.at(c.Path, c.Line); old > 0 {
			args = append(args, "-f", fmt.Sprintf("position[old_line]=%d", old))
		}
		_, err := g.api(args...)
		switch {
		case errors.Is(err, errForbidden):
			stop = "the forge refused the discussion: " + said(err)
			whys[i] = stop
		case errors.Is(err, errRefused):
			whys[i] = "the forge refused the discussion: " + said(err)
		case err != nil:
			return "", nil, err
		}
	}
	return changes(web), whys, nil
}

// changesPage is the merge request's changes, where its discussions show.
func (g *gitlab) changesPage(id int) string {
	_, web, err := g.mergeRequestPage(id)
	if err != nil {
		return ""
	}
	return changes(web)
}

func changes(web string) string {
	if web == "" {
		return ""
	}
	return strings.TrimSuffix(web, "/") + "/diffs"
}
