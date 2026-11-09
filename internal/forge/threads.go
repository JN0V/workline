package forge

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Thread is a conversation on a line of a merge request: its first
// comment and the replies under it, oldest first, as a person reads it on
// the forge's changes page (GitHub's review thread, GitLab's discussion
// on the diff). ID is what Reply answers it by; URL, where a person opens
// it.
type Thread struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Resolved bool   `json:"resolved,omitempty"`
	URL      string `json:"url,omitempty"`
	Comments []Note `json:"comments"`
	node     string // GitHub's thread id, which resolving it takes
}

// Threaded is a forge that lists a merge request's conversations on its
// lines and replies in one: the tester answers an objection there
// (ADR-0055). A forge without it says so, never a thread read as none.
type Threaded interface {
	Threads(id int) ([]Thread, error)
	Reply(id int, thread, body string) error
}

// Resolver is a forge that marks conversations on a merge request's lines
// resolved, each by its Thread.ID: the reviewer's own, once the code
// answers them (#1039). A person's is never resolved by the line.
type Resolver interface {
	Resolve(id int, threads []string) error
}

// LineKeys lists the keys of the reviewer's line comments a text holds.
func LineKeys(body string) []string { return keysIn(body) }

// Threads keeps each comment on a line as a thread, its replies under it;
// a thread given no id is named by its place, counting from 1.
func (f *Fake) Threads(id int) ([]Thread, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(Target{Kind: "merge-request", ID: id})
	if err != nil {
		return nil, err
	}
	var out []Thread
	for i, t := range it.Threads {
		tid := t.ID
		if tid == "" {
			tid = strconv.Itoa(i + 1)
		}
		th := Thread{ID: tid, Path: t.Path, Line: t.Line, Resolved: t.Resolved,
			URL:      fmt.Sprintf("https://forge.example/merge-requests/%d#thread-%s", id, tid),
			Comments: []Note{{Body: t.Body, Author: t.Author}}}
		for _, r := range t.Replies {
			th.Comments = append(th.Comments, Note(r))
		}
		out = append(out, th)
	}
	return out, nil
}

// Reply adds a reply to the thread, as the token's user: its author left
// unsaid, as the simulated forge's own comments are.
func (f *Fake) Reply(id int, thread, body string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "merge-request", ID: id})
		if err != nil {
			return err
		}
		for i := range it.Threads {
			if it.Threads[i].ID == thread || it.Threads[i].ID == "" && strconv.Itoa(i+1) == thread {
				it.Threads[i].Replies = append(it.Threads[i].Replies, FakeComment{Body: body})
				return nil
			}
		}
		return fmt.Errorf("merge request %d holds no thread %s", id, thread)
	})
}

// Resolve marks the threads resolved; those it does not hold, the rest
// resolved still, are said in the error.
func (f *Fake) Resolve(id int, threads []string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "merge-request", ID: id})
		if err != nil {
			return err
		}
		var missing []string
	next:
		for _, thread := range threads {
			for i := range it.Threads {
				if it.Threads[i].ID == thread || it.Threads[i].ID == "" && strconv.Itoa(i+1) == thread {
					it.Threads[i].Resolved = true
					continue next
				}
			}
			missing = append(missing, thread)
		}
		return noThreads(id, missing)
	})
}

// noThreads says the threads a merge request no longer holds, if any.
func noThreads(id int, missing []string) error {
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("merge request %d holds no thread %s: the others resolved", id, strings.Join(missing, ", "))
}

// threadsFullQuery lists a pull request's review threads with their
// comments, a hundred of each, who wrote each: GitHub says whether a
// thread is resolved in GraphQL only.
const threadsFullQuery = `query($owner: String!, $name: String!, $number: Int!, $endCursor: String) { repository(owner: $owner, name: $name) { pullRequest(number: $number) { reviewThreads(first: 100, after: $endCursor) { pageInfo { hasNextPage endCursor } nodes { id isResolved path line comments(first: 100) { pageInfo { hasNextPage } nodes { databaseId body url createdAt authorAssociation author { __typename login } } } } } } } }`

// threadCommentsQuery lists the comments of one review thread, by its
// GraphQL id, a hundred a page: a thread holds more than a page of them.
const threadCommentsQuery = `query($id: ID!, $endCursor: String) { node(id: $id) { ... on PullRequestReviewThread { comments(first: 100, after: $endCursor) { pageInfo { hasNextPage endCursor } nodes { databaseId body url createdAt authorAssociation author { __typename login } } } } } }`

// Threads reads the review threads; a thread's id is its first comment's,
// the one a reply answers (REST). A bot's login is given as REST gives
// it, with `[bot]`, as workline's own account is named.
func (g *github) Threads(id int) ([]Thread, error) {
	out, err := g.api("graphql", "--paginate", "-f", "query="+threadsFullQuery, "-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("number=%d", id),
		"--jq", `.data.repository.pullRequest.reviewThreads.nodes[]`)
	if err != nil {
		return nil, err
	}
	type comment struct {
		ID          int    `json:"databaseId"`
		Body        string `json:"body"`
		URL         string `json:"url"`
		Created     string `json:"createdAt"`
		Association string `json:"authorAssociation"`
		Author      *struct {
			Type  string `json:"__typename"`
			Login string `json:"login"`
		} `json:"author"`
	}
	found, err := lines[struct {
		Node     string `json:"id"`
		Resolved bool   `json:"isResolved"`
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Comments struct {
			Nodes []comment `json:"nodes"`
		} `json:"comments"`
	}](out)
	if err != nil {
		return nil, err
	}
	var all []Thread
	for _, t := range found {
		if len(t.Comments.Nodes) == 0 {
			continue
		}
		th := Thread{ID: strconv.Itoa(t.Comments.Nodes[0].ID), Path: t.Path, Line: t.Line, Resolved: t.Resolved, URL: t.Comments.Nodes[0].URL, node: t.Node}
		for _, c := range t.Comments.Nodes {
			n := Note{Body: c.Body, Created: c.Created, ID: strconv.Itoa(c.ID), Insider: insider(c.Association)}
			if c.Author != nil {
				n.Author, n.Bot = c.Author.Login, c.Author.Type == "Bot"
				if n.Bot && !strings.HasSuffix(n.Author, "[bot]") {
					n.Author += "[bot]"
				}
			}
			th.Comments = append(th.Comments, n)
		}
		all = append(all, th)
	}
	return all, nil
}

// Reply answers the thread whose first comment is thread.
func (g *github) Reply(id int, thread, body string) error {
	_, err := g.api("-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/comments/%s/replies", id, thread), "-f", "body="+body)
	return err
}

// resolveThread marks a review thread resolved, by its GraphQL id.
const resolveThread = `mutation($id: ID!) { resolveReviewThread(input: {threadId: $id}) { thread { isResolved } } }`

// Resolve marks the review threads whose first comment is each of threads
// resolved: GitHub resolves a thread in GraphQL only, by its own id.
func (g *github) Resolve(id int, threads []string) error {
	all, err := g.Threads(id)
	if err != nil {
		return err
	}
	var missing []string // gone since the run read them: the rest resolved still
	for _, thread := range threads {
		i := slices.IndexFunc(all, func(t Thread) bool { return t.ID == thread })
		switch {
		case i < 0:
			missing = append(missing, thread)
			continue
		case all[i].Resolved:
			continue
		}
		if _, err := g.api("graphql", "-f", "query="+resolveThread, "-f", "id="+all[i].node); err != nil {
			return err
		}
	}
	return noThreads(id, missing)
}

// Threads reads the merge request's discussions on its diff, those with a
// position; a discussion's id is GitLab's own.
func (g *gitlab) Threads(id int) ([]Thread, error) {
	mr := path(Target{Kind: "merge-request", ID: id})
	_, web, err := g.mergeRequestPage(id)
	if err != nil {
		return nil, err
	}
	out, err := g.api("--paginate", mr+"/discussions?per_page=100")
	if err != nil {
		return nil, err
	}
	discussions, err := pages[struct {
		ID    string `json:"id"`
		Notes []struct {
			ID       int    `json:"id"`
			Body     string `json:"body"`
			Resolved bool   `json:"resolved"`
			Created  string `json:"created_at"`
			Author   struct {
				Username string `json:"username"`
				Bot      bool   `json:"bot"`
			} `json:"author"`
			Position *struct {
				New  string `json:"new_path"`
				Line int    `json:"new_line"`
			} `json:"position"`
		} `json:"notes"`
	}](out)
	if err != nil {
		return nil, err
	}
	var all []Thread
	for _, d := range discussions {
		if len(d.Notes) == 0 || d.Notes[0].Position == nil {
			continue
		}
		first := d.Notes[0]
		th := Thread{ID: d.ID, Path: first.Position.New, Line: first.Position.Line, Resolved: first.Resolved,
			URL: fmt.Sprintf("%s#note_%d", web, first.ID)}
		for _, n := range d.Notes {
			th.Comments = append(th.Comments, Note{Body: n.Body, Author: n.Author.Username, Bot: n.Author.Bot, Created: n.Created, ID: strconv.Itoa(n.ID)})
		}
		all = append(all, th)
	}
	return all, nil
}

// Reply adds a note to the discussion.
func (g *gitlab) Reply(id int, thread, body string) error {
	_, err := g.api("-X", "POST", path(Target{Kind: "merge-request", ID: id})+"/discussions/"+thread+"/notes", "-f", "body="+body)
	return err
}

// Resolve marks each discussion resolved; one refused leaves the others
// resolved still, each refused said with why. A forge out of reach stops.
func (g *gitlab) Resolve(id int, threads []string) error {
	var refused []string
	for _, thread := range threads {
		_, err := g.api("-X", "PUT", path(Target{Kind: "merge-request", ID: id})+"/discussions/"+thread, "-f", "resolved=true")
		switch {
		case err == nil:
		case refusal(err):
			refused = append(refused, thread+" ("+said(err)+")")
		default:
			return err
		}
	}
	if len(refused) == 0 {
		return nil
	}
	return fmt.Errorf("%w: discussions refused, the others resolved: %s", errRefused, strings.Join(refused, "; "))
}
