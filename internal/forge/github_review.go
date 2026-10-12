package forge

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Review posts the comments in one pull request review, event COMMENT:
// it never approves nor requests changes (ADR-0020). GitHub takes a
// review whole or not at all: a line its diff does not show is left out
// first, from the pull request's files; a review refused (HTTP 403, a
// token without `pull-requests: write`; 422) is said, each comment's why.
func (g *github) Review(id int, head, body string, comments []LineComment) (Reviewed, error) {
	return review(g, id, head, body, comments)
}

// shownLines reads the pull request's files, each with its patch; a file
// GitHub gives none of (binary, too large) shows no line.
func (g *github) shownLines(id int) (shown, error) {
	out, err := g.api("--paginate", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/files?per_page=100", id),
		"--jq", `.[] | {filename, previous: (.previous_filename // ""), patch: (.patch // "")}`)
	if err != nil {
		return shown{}, err
	}
	files, err := lines[struct {
		Filename string `json:"filename"`
		Previous string `json:"previous"`
		Patch    string `json:"patch"`
	}](out)
	if err != nil {
		return shown{}, err
	}
	s := newShown()
	for _, f := range files {
		s.add(f.Filename, f.Previous, f.Patch)
	}
	return s, nil
}

// threadsQuery lists a pull request's review threads, a hundred a page:
// whether each is resolved, and its first comment.
const threadsQuery = `query($owner: String!, $name: String!, $number: Int!, $endCursor: String) { repository(owner: $owner, name: $name) { pullRequest(number: $number) { reviewThreads(first: 100, after: $endCursor) { pageInfo { hasNextPage endCursor } nodes { isResolved comments(first: 1) { nodes { body } } } } } } }`

// openKeys reads the keys the pull request's threads not resolved hold:
// GitHub says whether a thread is resolved in GraphQL only.
func (g *github) openKeys(id int) (map[string]bool, error) {
	out, err := g.api("graphql", "--paginate", "-f", "query="+threadsQuery, "-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("number=%d", id),
		"--jq", `.data.repository.pullRequest.reviewThreads.nodes[] | select(.isResolved | not) | {body: (.comments.nodes[0].body // "")}`)
	if err != nil {
		return nil, err
	}
	threads, err := lines[struct {
		Body string `json:"body"`
	}](out)
	if err != nil {
		return nil, err
	}
	open := map[string]bool{}
	for _, t := range threads {
		for _, k := range keysIn(t.Body) {
			open[k] = true
		}
	}
	return open, nil
}

// postLines posts one review holding every comment, each on its line on
// the right side, at head when given.
func (g *github) postLines(id int, head, body string, _ shown, comments []LineComment) (string, []string, error) {
	type comment struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Side string `json:"side"`
		Body string `json:"body"`
	}
	req := struct {
		Commit   string    `json:"commit_id,omitempty"`
		Body     string    `json:"body"`
		Event    string    `json:"event"`
		Comments []comment `json:"comments"`
	}{Commit: head, Body: body, Event: "COMMENT"}
	for _, c := range comments {
		req.Comments = append(req.Comments, comment{Path: c.Path, Line: c.Line, Side: "RIGHT", Body: c.withKey()})
	}
	input, _ := json.Marshal(req)
	out, err := runInput(g.repo, input, "gh", "api", "-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/reviews", id), "--input", "-")
	if err != nil {
		if refusal(err) && strings.Contains(err.Error(), "HTTP 403") {
			err = fmt.Errorf("%w (the token needs pull-requests: write)", err)
		}
		return "", nil, err
	}
	var posted struct {
		URL string `json:"html_url"`
	}
	if err := decode(out, &posted); err != nil {
		return "", nil, err
	}
	return posted.URL, nil, nil
}

// changesPage is the pull request's files, where its threads show.
func (g *github) changesPage(id int) string {
	if p := g.issuePages(); p != "" {
		return fmt.Sprintf("%spull/%d/files", strings.TrimSuffix(p, "issues/"), id)
	}
	return ""
}
