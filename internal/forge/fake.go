package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
)

// Fake is a forge kept in a JSON file, shared between processes, for the
// conformance tests. It can be told to fail its N-th write, once.
type Fake struct{ Path string }

// FakeState is the file's content.
type FakeState struct {
	Issues        []FakeItem `json:"issues"`
	MergeRequests []FakeItem `json:"merge-requests"`
	Milestones    []string   `json:"milestones,omitempty"`    // open milestones, by title
	Labels        []FakeItem `json:"labels,omitempty"`        // labels defined, their name as id
	FailOnWrite   int        `json:"fail-on-write,omitempty"` // the write that fails, counting from 1
	Writes        int        `json:"writes"`
}

// FakeItem is an issue or a merge request.
type FakeItem struct {
	ID        int      `json:"id"`
	Branch    string   `json:"branch,omitempty"` // a merge request's source branch
	Base      string   `json:"base,omitempty"`
	Closed    bool     `json:"closed,omitempty"`
	Reason    string   `json:"reason,omitempty"` // why it was closed: completed or duplicate
	Milestone string   `json:"milestone,omitempty"`
	Fork      bool     `json:"fork,omitempty"` // a merge request from a fork
	Title     string   `json:"title,omitempty"`
	Body      string   `json:"body,omitempty"`
	Labels    []string `json:"labels"`
	Comments  []string `json:"comments"`
	Author    string   `json:"author,omitempty"`
	Insider   bool     `json:"insider,omitempty"`
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
			out = append(out, Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Milestone: it.Milestone, Author: it.Author, Insider: it.Insider})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *Fake) Comments(t Target) ([]string, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(t)
	if err != nil {
		return nil, err
	}
	return it.Comments, nil
}

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
			if strings.Contains(c, marker) {
				return nil
			}
		}
		it.Comments = append(it.Comments, body+"\n\n"+marker)
		return nil
	})
}

func (f *Fake) Sticky(t Target, body, marker string, create bool) error {
	return f.write(func(s *FakeState) error {
		it, err := s.item(t)
		if err != nil {
			return err
		}
		for i, c := range it.Comments {
			if strings.Contains(c, marker) {
				it.Comments[i] = body + "\n\n" + marker
				return nil
			}
		}
		if create {
			it.Comments = append(it.Comments, body+"\n\n"+marker)
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
					if strings.Contains(c, marker) {
						return nil
					}
				}
				s.Issues[i].Comments = append(s.Issues[i].Comments, body+"\n\n"+marker)
				return nil
			}
		}
		id = 1
		for _, it := range s.Issues {
			id = max(id, it.ID+1)
		}
		s.Issues = append(s.Issues, FakeItem{ID: id, Title: title, Body: body + "\n\n" + marker, Labels: []string{}, Comments: []string{}})
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
		s.MergeRequests = append(s.MergeRequests, FakeItem{ID: id, Branch: branch, Base: base, Title: title, Body: body, Labels: []string{}, Comments: []string{}})
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
	return MergeRequest{Branch: it.Branch, Base: it.Base, Here: !it.Fork}, nil
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
		s.Issues = append(s.Issues, FakeItem{ID: id, Title: title, Body: body, Labels: []string{}, Comments: []string{}})
		return nil
	})
	return id, err
}

func (f *Fake) Milestones() ([]string, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	return s.Milestones, nil
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
