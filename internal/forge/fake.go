package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Fake is a forge kept in a JSON file, shared between processes, for the
// conformance tests. It can be told to fail its N-th write, once. Repo is
// the repository it serves: a merge request's diff is computed there.
type Fake struct{ Path, Repo string }

// FakeState is the file's content.
type FakeState struct {
	Issues        []FakeItem `json:"issues"`
	MergeRequests []FakeItem `json:"merge-requests"`
	Milestones    []string   `json:"milestones,omitempty"` // open milestones, by title
	// MilestoneDue gives a milestone its due date, YYYY-MM-DD, by title.
	MilestoneDue map[string]string `json:"milestone-due,omitempty"`
	Labels       []FakeItem        `json:"labels,omitempty"`       // labels defined, their name as id
	SubIssues    bool              `json:"sub-issues,omitempty"`   // the forge has sub-issues, as GitHub
	Dependencies bool              `json:"dependencies,omitempty"` // the forge has a blocked-by relation, as GitHub
	// Scoped: its labels are scoped, `key::value`, as GitLab's (ADR-0038).
	Scoped bool `json:"scoped-labels,omitempty"`
	// IssueFields are its single-select issue fields, as a GitHub
	// organisation's (ADR-0040); Public, the repository is public; Blind,
	// why the token reads no value visible to members only, "" when it
	// does.
	IssueFields []Field `json:"issue-fields,omitempty"`
	Public      bool    `json:"public,omitempty"`
	Blind       string  `json:"fields-blind,omitempty"`
	// Weights: its issues carry a weight, as GitLab Premium's (ADR-0040).
	Weights     bool `json:"weights,omitempty"`
	FailOnWrite int  `json:"fail-on-write,omitempty"` // the write that fails, counting from 1
	Writes      int  `json:"writes"`
	// ReviewRefused: the forge refuses every comment on a line, as a
	// token that may not review does, with this reason (HTTP 403).
	ReviewRefused string `json:"review-refused,omitempty"`
	Reviews       int    `json:"reviews,omitempty"` // the reviews posted, each holding comments on lines
	// JobToken: the token is a CI job's own, whose merges start no
	// pipeline, as GitHub's GITHUB_TOKEN (MergeToken, ADR-0042).
	JobToken bool `json:"job-token,omitempty"`
	// Unguarded: why a push after a yes could merge before follow takes
	// it back — a base not requiring workline/accepted (Guarded).
	Unguarded string `json:"unguarded,omitempty"`
	// Statuses are the statuses set on commits, as GitHub keeps them: the
	// commit a revision of the repository, who set it (`workline-bot`, the
	// token's user, or another).
	Statuses []FakeStatus `json:"statuses,omitempty"`
	// Checks are the checks written on commits, each with its annotations
	// (Annotate), in the order written.
	Checks []FakeCheck `json:"checks,omitempty"`
	// ChecksRefused: the forge refuses every check, as a token without
	// checks: write does (HTTP 403), with this reason.
	ChecksRefused string `json:"checks-refused,omitempty"`
	// ConversationsRequired are the branches requiring every conversation
	// of a merge request resolved before it merges; ConversationsUnread,
	// what of the branches' rules the token cannot read (Conversations).
	ConversationsRequired []string `json:"conversations-required,omitempty"`
	ConversationsUnread   string   `json:"conversations-unread,omitempty"`
	// Tips are the runs on each branch's tip, by branch, as Tip reads
	// them; TipsUnread, why the forge does not answer them.
	Tips       map[string][]CheckRun `json:"tips,omitempty"`
	TipsUnread string                `json:"tips-unread,omitempty"`
	// TipsThen are the runs a branch's tip carries once Tip has read it:
	// its checks ending while the line waits for them.
	TipsThen map[string][]CheckRun `json:"tips-then,omitempty"`
	// TipsUnreadThen: why the forge does not answer the read after the
	// first, once, a hiccup while the line waits; TipsUnreadNext carries
	// it from the first read to the next.
	TipsUnreadThen string `json:"tips-unread-then,omitempty"`
	TipsUnreadNext string `json:"tips-unread-next,omitempty"`
	// Squash is how its merge requests land when squashed, the same for
	// each but Suffix, where "{n}" is the merge request's number (Squasher).
	Squash *Squash `json:"squash,omitempty"`
}

// FakeCheck is a check written on a commit.
type FakeCheck struct {
	Commit      string       `json:"commit"`
	Name        string       `json:"name"`
	Conclusion  string       `json:"conclusion"`
	Title       string       `json:"title,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Annotations []Annotation `json:"annotations"`
}

// FakeStatus is a status set on a commit.
type FakeStatus struct {
	Commit      string `json:"commit"`
	Name        string `json:"name"`
	State       string `json:"state,omitempty"`
	Description string `json:"description,omitempty"`
	By          string `json:"by"`
}

// FakeItem is an issue or a merge request.
type FakeItem struct {
	ID        int    `json:"id"`
	Branch    string `json:"branch,omitempty"` // a merge request's source branch
	Base      string `json:"base,omitempty"`
	Closed    bool   `json:"closed,omitempty"`
	Reason    string `json:"reason,omitempty"` // why it was closed: completed, not_planned or duplicate
	Milestone string `json:"milestone,omitempty"`
	Fork      bool   `json:"fork,omitempty"` // a merge request from a fork
	Bot       bool   `json:"bot,omitempty"`  // a merge request a bot opened: an app, a token's user
	Title     string `json:"title,omitempty"`
	Body      string `json:"body,omitempty"`
	// Edits are an issue's earlier bodies, newest first, as GitHub's edit
	// history keeps them: SetBody adds the one it replaces (Versions).
	Edits     []string      `json:"edits,omitempty"`
	Labels    []string      `json:"labels"`
	Comments  []FakeComment `json:"comments"`
	Author    string        `json:"author,omitempty"`
	Insider   bool          `json:"insider,omitempty"`
	Parent    int           `json:"parent,omitempty"`     // the issue it is a sub-issue of
	BlockedBy []int         `json:"blocked-by,omitempty"` // the issues it waits on, in the forge's own relation
	ClosedBy  []Closer      `json:"closed-by,omitempty"`  // what closed it: a pull request, a commit
	// Labeled says when it last got each label; Links, what names it
	// (Trail, ADR-0031).
	Labeled map[string]string `json:"labeled,omitempty"`
	Links   []Link            `json:"links,omitempty"`
	// LabelEvents are its labels set and taken off, with who; Reactions,
	// the reactions on its comments, by the comment's id (ADR-0038).
	LabelEvents []LabelEvent        `json:"label-events,omitempty"`
	Reactions   map[string][]string `json:"reactions,omitempty"`
	// ReactionsBy are the reactions people put on its comments, by the
	// comment's id, with who: what Reactions reads, the role's own besides.
	ReactionsBy map[string][]Reaction `json:"reactions-by,omitempty"`
	// Posted counts the comments posted on it, each mailing whoever
	// follows the issue on a real forge; an edit posts none (Renewer).
	Posted int `json:"posted,omitempty"`
	// Threads are a merge request's comments on lines, as a review leaves
	// them (LineReviewer).
	Threads []FakeThread `json:"threads,omitempty"`
	// A merge request's state for the merge policy (MergeState, ADR-0042):
	// a draft; its auto-merge on, when it was last turned on and by whom;
	// its head's checks, summed up, or each run of them, summed up as
	// GitHub's are; its branch's pushes, each head a revision of the
	// repository. Clean: the forge would merge it at once, its rules met,
	// so turning auto-merge on merges it (Merged).
	Draft       bool       `json:"draft,omitempty"`
	AutoMerge   bool       `json:"auto-merge,omitempty"`
	AutoMergeAt string     `json:"auto-merge-at,omitempty"`
	AutoMergeBy string     `json:"auto-merge-by,omitempty"`
	Checks      string     `json:"checks,omitempty"`
	CheckRuns   []CheckRun `json:"check-runs,omitempty"`
	Pushes      []FakePush `json:"pushes,omitempty"`
	Clean       bool       `json:"clean,omitempty"`
	Merged      bool       `json:"merged,omitempty"`
	// MergedAt: when a merge request was merged, RFC 3339 or YYYY-MM-DD;
	// MergedAs, the commit it became, a revision of the repository (main,
	// main~1) — what MergedSince lists.
	MergedAt string `json:"merged-at,omitempty"`
	MergedAs string `json:"merged-as,omitempty"`
	// Closing: who closed an issue, as Closing reads it.
	Closing *LabelEvent `json:"closing,omitempty"`
	// Rank: a label's, where the forge ranks them (GitLab's prioritized
	// labels); 0 for none.
	Rank int `json:"rank,omitempty"`
	// Fields: an issue's values in the forge's issue fields, by name
	// (GitHub's, ADR-0040).
	Fields map[string]string `json:"fields,omitempty"`
	// Weight: an issue's GitLab weight, with weights; none when absent.
	Weight *int `json:"weight,omitempty"`
}

// FakePush is a push to a merge request's branch: the head, a revision
// of the repository (a branch, branch~1), when, and who pushed it ("" the
// forge does not say).
type FakePush struct {
	Head string `json:"head"`
	At   string `json:"at"`
	By   string `json:"by,omitempty"`
}

// FakeThread is a comment on a line of a merge request: its file and line
// in the file at the head of the change, its text, and whether a person
// resolved it.
type FakeThread struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Body     string `json:"body"`
	Resolved bool   `json:"resolved,omitempty"`
}

// FakeComment is a comment: in the file, its body alone, or with its
// author as {body, author, insider, bot, created}, and who last edited
// it when not its author (editor).
type FakeComment Note

func (c FakeComment) MarshalJSON() ([]byte, error) {
	if c.Author == "" && !c.Insider && !c.Bot && c.Created == "" && c.ID == "" && c.App == "" && !c.Own && c.Editor == "" {
		return json.Marshal(c.Body)
	}
	return json.Marshal(Note(c))
}

func (c *FakeComment) UnmarshalJSON(data []byte) error {
	if json.Unmarshal(data, &c.Body) == nil {
		return nil
	}
	return json.Unmarshal(data, (*Note)(c))
}

func (f *Fake) load() (*FakeState, error) {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	var s FakeState
	return &s, json.Unmarshal(data, &s)
}

func (f *Fake) save(s *FakeState) error {
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(f.Path, data, 0o644)
}

// write runs change as one write, failing it if it is the one told to fail.
func (f *Fake) write(change func(*FakeState) error) error {
	s, err := f.load()
	if err != nil {
		return err
	}
	s.Writes++
	if s.FailOnWrite > 0 && s.Writes == s.FailOnWrite {
		s.FailOnWrite = 0 // fails once, then the forge is back
		f.save(s)
		return fmt.Errorf("%w: the simulated forge failed write %d", ErrUnreachable, s.Writes)
	}
	if err := change(s); err != nil {
		return err
	}
	return f.save(s)
}

func (s *FakeState) item(t Target) (*FakeItem, error) {
	list := s.Issues
	if t.Kind == "merge-request" {
		list = s.MergeRequests
	}
	for i := range list {
		if list[i].ID == t.ID {
			return &list[i], nil
		}
	}
	return nil, fmt.Errorf("no %s", t)
}

func (f *Fake) Issue(id int) (*Issue, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(Target{Kind: "issue", ID: id})
	if err != nil {
		return nil, err
	}
	return &Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Closed: it.Closed, Author: it.Author, Insider: it.Insider, Fields: it.Fields, Weight: it.Weight}, nil
}

func (f *Fake) Issues() ([]Issue, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []Issue
	for _, it := range s.Issues {
		if !it.Closed {
			var children []int
			for _, c := range s.Issues {
				if c.Parent == it.ID {
					children = append(children, c.ID)
				}
			}
			out = append(out, Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Milestone: it.Milestone, MilestoneDue: s.MilestoneDue[it.Milestone], Author: it.Author, Insider: it.Insider, BlockedBy: it.BlockedBy, Children: children, Fields: it.Fields, Weight: it.Weight})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *Fake) AllIssues() ([]Issue, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []Issue
	for _, it := range s.Issues {
		out = append(out, Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Milestone: it.Milestone, MilestoneDue: s.MilestoneDue[it.Milestone], Author: it.Author, Insider: it.Insider, Closed: it.Closed, Reason: it.Reason, Fields: it.Fields, Weight: it.Weight})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *Fake) Comments(t Target) ([]string, error) {
	notes, err := f.Notes(t)
	return Bodies(notes), err
}

func (f *Fake) Notes(t Target) ([]Note, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(t)
	if err != nil {
		return nil, err
	}
	notes := make([]Note, 0, len(it.Comments))
	for i, c := range it.Comments {
		n := Note(c)
		if n.ID == "" {
			n.ID = strconv.Itoa(i + 1) // its place: the fake's comments are never deleted
		}
		if n.Editor == n.Author {
			n.Editor = ""
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func (f *Fake) LabelEvents(id int) ([]LabelEvent, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(Target{Kind: "issue", ID: id})
	if err != nil {
		return nil, err
	}
	return it.LabelEvents, nil
}

// Reactions are the people's reactions on a comment, and the role's own,
// its token's user's.
func (f *Fake) Reactions(issue int, note string) ([]Reaction, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(Target{Kind: "issue", ID: issue})
	if err != nil {
		return nil, err
	}
	out := slices.Clone(it.ReactionsBy[note])
	for _, e := range it.Reactions[note] {
		out = append(out, Reaction{Emoji: e, Author: fakeBot, Bot: true})
	}
	return out, nil
}

func (f *Fake) React(id int, note, emoji string) error {
	if note == "" {
		return nil
	}
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil {
			return err
		}
		if it.Reactions == nil {
			it.Reactions = map[string][]string{}
		}
		if !slices.Contains(it.Reactions[note], emoji) {
			it.Reactions[note] = append(it.Reactions[note], emoji)
		}
		return nil
	})
}

// scopedLabels: as the file says, GitLab's or not.
func (f *Fake) scopedLabels() bool {
	s, err := f.load()
	return err == nil && s.Scoped
}

// labelFilter: an address the simulated forge would give.
func (f *Fake) labelFilter(label string) string {
	return "https://forge.example/issues?label=" + label
}

// issuePages: the address of the simulated forge's issues.
func (f *Fake) issuePages() string { return "https://forge.example/issues/" }

// commentAnchor: a comment's place on the simulated forge's issue page.
func (f *Fake) commentAnchor(id string) string { return "#comment-" + id }

// filePages: the address of the simulated forge's files.
func (f *Fake) filePages() string { return "https://forge.example/blob/HEAD/" }

func (f *Fake) Close(id, dup int) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil || it.Closed {
			return err
		}
		it.Closed, it.Reason = true, "completed"
		if dup > 0 {
			it.Reason = "duplicate"
		}
		return nil
	})
}

func (f *Fake) Comment(t Target, body, marker string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(t)
		if err != nil {
			return err
		}
		for _, c := range it.Comments {
			if strings.Contains(c.Body, marker) {
				return nil
			}
		}
		it.Comments = append(it.Comments, FakeComment{Body: body + "\n\n" + marker, Author: fakeBot, Bot: true})
		it.Posted++
		return nil
	})
}

func (f *Fake) Sticky(t Target, body, marker string, create bool) error {
	return f.sticky(t, body, marker, create, false)
}

// stickyOwn edits the last comment carrying marker its token's user
// wrote (KeepOwn): with only others' carrying it, its own is posted
// after them, as a forge's.
func (f *Fake) stickyOwn(t Target, body, marker string, create bool) error {
	return f.sticky(t, body, marker, create, true)
}

func (f *Fake) sticky(t Target, body, marker string, create, mine bool) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(t)
		if err != nil {
			return err
		}
		own, _ := f.own()
		for i := len(it.Comments) - 1; i >= 0; i-- { // the last, as a forge's
			switch {
			case !strings.Contains(it.Comments[i].Body, marker):
			case mine && !own.Wrote(Note(it.Comments[i])):
				create = true
			default:
				// Edited, it stays its author's, as on a forge, and the
				// token's user is its last editor.
				it.Comments[i].Body = body + "\n\n" + marker
				it.Comments[i].Editor = ""
				if a := it.Comments[i].Author; a != "" && a != fakeBot {
					it.Comments[i].Editor = fakeBot
				}
				return nil
			}
		}
		if create {
			it.Comments = append(it.Comments, FakeComment{Body: body + "\n\n" + marker, Author: fakeBot, Bot: true})
			it.Posted++
		}
		return nil
	})
}

// Renew posts the comment anew, the token's, and deletes the ones carrying
// marker before it; the comments left keep their ids (Renewer).
func (f *Fake) Renew(t Target, body, marker string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(t)
		if err != nil {
			return err
		}
		next := len(it.Comments) + 1
		var kept []FakeComment
		for i, c := range it.Comments {
			if c.ID == "" {
				c.ID = strconv.Itoa(i + 1) // its place, kept once another goes
			}
			if n, err := strconv.Atoi(c.ID); err == nil && n >= next {
				next = n + 1
			}
			if !strings.Contains(c.Body, marker) {
				kept = append(kept, c)
			}
		}
		it.Comments = append(kept, FakeComment{Body: body + "\n\n" + marker, Author: fakeBot, Bot: true, ID: strconv.Itoa(next)})
		it.Posted++
		return nil
	})
}

func (f *Fake) Label(t Target, add, remove []string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(t)
		if err != nil {
			return err
		}
		for _, l := range add {
			if !slices.Contains(it.Labels, l) {
				it.Labels = append(it.Labels, l)
			}
		}
		it.Labels = slices.DeleteFunc(it.Labels, func(l string) bool { return slices.Contains(remove, l) })
		if it.Labels == nil {
			it.Labels = []string{}
		}
		return nil
	})
}

func (f *Fake) OpenIssue(title, body, marker string) (int, error) {
	id := 0
	err := f.write(func(s *FakeState) error {
		for i := range s.Issues {
			if s.Issues[i].Title == title && !s.Issues[i].Closed { // an open one, as GitHub reuses it
				id = s.Issues[i].ID
				for _, c := range s.Issues[i].Comments {
					if strings.Contains(c.Body, marker) {
						return nil
					}
				}
				s.Issues[i].Comments = append(s.Issues[i].Comments, FakeComment{Body: body + "\n\n" + marker, Author: fakeBot, Bot: true})
				return nil
			}
		}
		id = 1
		for _, it := range s.Issues {
			id = max(id, it.ID+1)
		}
		s.Issues = append(s.Issues, FakeItem{ID: id, Title: title, Body: body + "\n\n" + marker, Labels: []string{}, Comments: []FakeComment{}, Author: fakeBot})
		return nil
	})
	return id, err
}

func (f *Fake) OpenMergeRequest(branch, base, title, body string) (int, error) {
	id := 0
	err := f.write(func(s *FakeState) error {
		for i := range s.MergeRequests {
			if m := &s.MergeRequests[i]; m.Branch == branch && !m.Closed {
				m.Title, m.Body, id = title, body, m.ID
				return nil
			}
		}
		for _, m := range s.MergeRequests {
			id = max(id, m.ID)
		}
		id++
		s.MergeRequests = append(s.MergeRequests, FakeItem{ID: id, Branch: branch, Base: base, Title: title, Body: body, Labels: []string{}, Comments: []FakeComment{}})
		return nil
	})
	return id, err
}

func (f *Fake) OpenMergeRequests(prefix string) ([]string, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, m := range s.MergeRequests {
		if !m.Closed && m.Branch != "" && strings.HasPrefix(m.Branch, prefix) {
			out = append(out, m.Branch)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (f *Fake) CloseMergeRequest(branch, body, marker string) error {
	return f.write(func(s *FakeState) error {
		for i := range s.MergeRequests {
			m := &s.MergeRequests[i]
			if m.Branch != branch || m.Closed {
				continue
			}
			said := slices.ContainsFunc(m.Comments, func(c FakeComment) bool { return strings.Contains(c.Body, marker) })
			if !said {
				m.Comments = append(m.Comments, FakeComment{Body: body + "\n\n" + marker, Author: fakeBot, Bot: true})
			}
			m.Closed = true
			return nil
		}
		return nil
	})
}

func (f *Fake) MergeRequest(id int) (MergeRequest, error) {
	s, err := f.load()
	if err != nil {
		return MergeRequest{}, err
	}
	it, err := s.item(Target{Kind: "merge-request", ID: id})
	if err != nil {
		return MergeRequest{}, err
	}
	return MergeRequest{ID: id, Branch: it.Branch, Base: it.Base, Here: !it.Fork, Title: it.Title, Body: it.Body}, nil
}

// Squash is the state's, a merge request's number in its suffix; none set,
// the forge never squashes.
func (f *Fake) Squash(id int) (Squash, error) {
	s, err := f.load()
	if err != nil || s.Squash == nil {
		return Squash{}, err
	}
	sq := *s.Squash
	sq.Suffix = strings.ReplaceAll(sq.Suffix, "{n}", strconv.Itoa(id))
	return sq, nil
}

// ListMergeRequests lists the open merge requests, each head kept under
// refs/pull/<id>/head, as GitHub keeps them.
func (f *Fake) ListMergeRequests() ([]MergeRequest, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []MergeRequest
	for _, m := range s.MergeRequests {
		if !m.Closed && m.Branch != "" {
			out = append(out, MergeRequest{ID: m.ID, Branch: m.Branch, Base: m.Base, Here: !m.Fork, Title: m.Title, Body: m.Body,
				Author: m.Author, Bot: m.Bot, Head: fmt.Sprintf("refs/pull/%d/head", m.ID),
				Labels: m.Labels, Draft: m.Draft, AutoMerge: m.AutoMerge})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *Fake) KeepIssue(title, body string, create bool) (int, error) {
	id := 0
	err := f.write(func(s *FakeState) error {
		for i := range s.Issues {
			if it := &s.Issues[i]; it.Title == title && !it.Closed {
				if it.Body != body { // its edit history, as SetBody keeps it
					it.Edits = append([]string{it.Body}, it.Edits...)
				}
				it.Body, id = body, it.ID
				return nil
			}
		}
		if !create {
			return nil
		}
		id = 1
		for _, it := range s.Issues {
			id = max(id, it.ID+1)
		}
		s.Issues = append(s.Issues, FakeItem{ID: id, Title: title, Body: body, Labels: []string{}, Comments: []FakeComment{}})
		return nil
	})
	return id, err
}

func (f *Fake) Milestones() ([]Milestone, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []Milestone
	for _, t := range s.Milestones {
		out = append(out, Milestone{Title: t, Due: s.MilestoneDue[t]})
	}
	return out, nil
}

func (f *Fake) EnsureLabel(name, color, description string) error {
	return f.ensureRanked(name, color, description, 0)
}

// ensureRanked: a label defined with its rank, as GitLab's prioritized
// labels, when the file says its labels are scoped; one there is left.
func (f *Fake) ensureRanked(name, color, description string, rank int) error {
	return f.write(func(s *FakeState) error {
		for _, l := range s.Labels {
			if fmt.Sprint(l.Title) == name {
				return nil
			}
		}
		if !s.Scoped {
			rank = 0
		}
		s.Labels = append(s.Labels, FakeItem{ID: len(s.Labels) + 1, Title: name, Body: description, Labels: []string{}, Rank: rank})
		return nil
	})
}

// issueFields: the fields the file holds, as GitHub lists an
// organisation's for a repository.
func (f *Fake) issueFields() (Fields, error) {
	s, err := f.load()
	if err != nil {
		return Fields{}, err
	}
	return Fields{List: s.IssueFields, Public: s.Public, Blind: s.Blind}, nil
}

// setField: an issue's value in a field the file holds, by an option's
// name; another is refused, as GitHub refuses it.
func (f *Fake) setField(issue int, field Field, option string) error {
	return f.write(func(s *FakeState) error {
		i := slices.IndexFunc(s.IssueFields, func(x Field) bool { return x.ID == field.ID })
		if i < 0 || !slices.Contains(s.IssueFields[i].Options, option) {
			return fmt.Errorf("no field %d with an option %q", field.ID, option)
		}
		it, err := s.item(Target{Kind: "issue", ID: issue})
		if err != nil {
			return err
		}
		if it.Fields == nil {
			it.Fields = map[string]string{}
		}
		it.Fields[s.IssueFields[i].Name] = option
		return nil
	})
}

// weights: whether the file's issues carry a weight, as GitLab Premium's.
func (f *Fake) weights() (bool, error) {
	s, err := f.load()
	return err == nil && s.Weights, err
}

// setWeight: an issue's weight, where the file keeps weights; refused
// elsewhere, as a forge that keeps none.
func (f *Fake) setWeight(issue, weight int) error {
	return f.write(func(s *FakeState) error {
		if !s.Weights {
			return fmt.Errorf("this forge keeps no weight: none set on #%d", issue)
		}
		it, err := s.item(Target{Kind: "issue", ID: issue})
		if err != nil {
			return err
		}
		it.Weight = &weight
		return nil
	})
}

// projectLabels: the labels defined, and those on any issue, as a forge
// creates one set on an issue.
func (f *Fake) projectLabels() ([]string, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range s.Labels {
		if !slices.Contains(out, l.Title) {
			out = append(out, l.Title)
		}
	}
	for _, it := range s.Issues {
		for _, l := range it.Labels {
			if !slices.Contains(out, l) {
				out = append(out, l)
			}
		}
	}
	return out, nil
}

func (f *Fake) SetBody(id int, body string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil {
			return err
		}
		if it.Body != body && it.Body != "" {
			it.Edits = append([]string{it.Body}, it.Edits...)
		}
		it.Body = body
		return nil
	})
}

// Versions lists an issue's bodies, the one it has now first, then its
// earlier ones, as GitHub's edit history does once it was edited.
func (f *Fake) Versions(id int) ([]string, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(Target{Kind: "issue", ID: id})
	if err != nil || len(it.Edits) == 0 {
		return nil, err
	}
	return append([]string{it.Body}, it.Edits...), nil
}

func (f *Fake) SetMilestone(id int, title string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil {
			return err
		}
		if !slices.Contains(s.Milestones, title) {
			s.Milestones = append(s.Milestones, title)
		}
		it.Milestone = title
		return nil
	})
}

func (f *Fake) SetTitle(id int, title string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil {
			return err
		}
		it.Title = title
		return nil
	})
}

func (f *Fake) AddSubIssue(parent, child int) (bool, error) {
	s, err := f.load()
	if err != nil || !s.SubIssues {
		return false, err
	}
	return true, f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: child})
		if err != nil {
			return err
		}
		it.Parent = parent
		return nil
	})
}

func (f *Fake) AddBlocker(id, blocker int) (bool, error) {
	s, err := f.load()
	if err != nil || !s.Dependencies {
		return false, err
	}
	if it, err := s.item(Target{Kind: "issue", ID: id}); err == nil && slices.Contains(it.BlockedBy, blocker) {
		return true, nil
	}
	return true, f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil {
			return err
		}
		it.BlockedBy = append(it.BlockedBy, blocker)
		return nil
	})
}

func (f *Fake) RemoveBlocker(id, blocker int) error {
	s, err := f.load()
	if err != nil {
		return err
	}
	if it, err := s.item(Target{Kind: "issue", ID: id}); err != nil || !slices.Contains(it.BlockedBy, blocker) {
		return err
	}
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil {
			return err
		}
		it.BlockedBy = slices.DeleteFunc(it.BlockedBy, func(n int) bool { return n == blocker })
		return nil
	})
}

func (f *Fake) Closers(id int) ([]Closer, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(Target{Kind: "issue", ID: id})
	if err != nil {
		return nil, err
	}
	return it.ClosedBy, nil
}

func (f *Fake) Trail(id int, label string) (Trail, error) {
	s, err := f.load()
	if err != nil {
		return Trail{}, err
	}
	it, err := s.item(Target{Kind: "issue", ID: id})
	if err != nil {
		return Trail{}, err
	}
	return Trail{Labeled: it.Labeled[label], Links: it.Links}, nil
}
