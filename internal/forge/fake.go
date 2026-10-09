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
// conformance tests. It can be told to fail its N-th write, once.
type Fake struct{ Path string }

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
	Scoped      bool `json:"scoped-labels,omitempty"`
	FailOnWrite int  `json:"fail-on-write,omitempty"` // the write that fails, counting from 1
	Writes      int  `json:"writes"`
}

// FakeItem is an issue or a merge request.
type FakeItem struct {
	ID        int           `json:"id"`
	Branch    string        `json:"branch,omitempty"` // a merge request's source branch
	Base      string        `json:"base,omitempty"`
	Closed    bool          `json:"closed,omitempty"`
	Reason    string        `json:"reason,omitempty"` // why it was closed: completed, not_planned or duplicate
	Milestone string        `json:"milestone,omitempty"`
	Fork      bool          `json:"fork,omitempty"` // a merge request from a fork
	Title     string        `json:"title,omitempty"`
	Body      string        `json:"body,omitempty"`
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
}

// FakeComment is a comment: in the file, its body alone, or with its
// author as {body, author, insider, bot, created}.
type FakeComment Note

func (c FakeComment) MarshalJSON() ([]byte, error) {
	if c.Author == "" && !c.Insider && !c.Bot && c.Created == "" && c.ID == "" {
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
	return &Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Closed: it.Closed, Author: it.Author, Insider: it.Insider}, nil
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
			out = append(out, Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Milestone: it.Milestone, MilestoneDue: s.MilestoneDue[it.Milestone], Author: it.Author, Insider: it.Insider, BlockedBy: it.BlockedBy, Children: children})
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
		out = append(out, Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Milestone: it.Milestone, MilestoneDue: s.MilestoneDue[it.Milestone], Author: it.Author, Insider: it.Insider, Closed: it.Closed, Reason: it.Reason})
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
		it.Comments = append(it.Comments, FakeComment{Body: body + "\n\n" + marker})
		return nil
	})
}

func (f *Fake) Sticky(t Target, body, marker string, create bool) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(t)
		if err != nil {
			return err
		}
		for i := len(it.Comments) - 1; i >= 0; i-- { // the last, as a forge's
			if strings.Contains(it.Comments[i].Body, marker) {
				if it.Comments[i].Body != body+"\n\n"+marker {
					it.Comments[i] = FakeComment{Body: body + "\n\n" + marker}
				}
				return nil
			}
		}
		if create {
			it.Comments = append(it.Comments, FakeComment{Body: body + "\n\n" + marker})
		}
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
			if s.Issues[i].Title == title {
				id = s.Issues[i].ID
				for _, c := range s.Issues[i].Comments {
					if strings.Contains(c.Body, marker) {
						return nil
					}
				}
				s.Issues[i].Comments = append(s.Issues[i].Comments, FakeComment{Body: body + "\n\n" + marker})
				return nil
			}
		}
		id = 1
		for _, it := range s.Issues {
			id = max(id, it.ID+1)
		}
		s.Issues = append(s.Issues, FakeItem{ID: id, Title: title, Body: body + "\n\n" + marker, Labels: []string{}, Comments: []FakeComment{}})
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

func (f *Fake) MergeRequest(id int) (MergeRequest, error) {
	s, err := f.load()
	if err != nil {
		return MergeRequest{}, err
	}
	it, err := s.item(Target{Kind: "merge-request", ID: id})
	if err != nil {
		return MergeRequest{}, err
	}
	return MergeRequest{Branch: it.Branch, Base: it.Base, Here: !it.Fork, Title: it.Title, Body: it.Body}, nil
}

func (f *Fake) KeepIssue(title, body string, create bool) (int, error) {
	id := 0
	err := f.write(func(s *FakeState) error {
		for i := range s.Issues {
			if s.Issues[i].Title == title && !s.Issues[i].Closed {
				s.Issues[i].Body, id = body, s.Issues[i].ID
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
	return f.write(func(s *FakeState) error {
		for _, l := range s.Labels {
			if fmt.Sprint(l.Title) == name {
				return nil
			}
		}
		s.Labels = append(s.Labels, FakeItem{ID: len(s.Labels) + 1, Title: name, Body: description, Labels: []string{}})
		return nil
	})
}

func (f *Fake) SetBody(id int, body string) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "issue", ID: id})
		if err != nil {
			return err
		}
		it.Body = body
		return nil
	})
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
