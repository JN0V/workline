package forge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	wlclock "github.com/JN0V/workline/internal/clock"
	"go.yaml.in/yaml/v3"
)

// Local is a forge kept in the clone, never committed: for a project with no
// forge, or one whose person works alone. Each issue is a Markdown file,
// .git/workline/issues/<n>.md; each merge request is a local branch, recorded
// the same way in .git/workline/merge-requests/<n>.md. Nothing is pushed: `workline issues`
// reads them, and a person's writes go through it (open, label, comment).
// A role's writes are idempotent, as on any forge.
type Local struct{ Repo string }

// notHere is an item the clone does not hold: the forge's answer that it
// is not there (NotFound), as another forge's 404.
type notHere struct{ said string }

func (e notHere) Error() string        { return e.said }
func (e notHere) Is(target error) bool { return target == errNotFound }

// LocalItem is an issue or a merge request of the local forge.
type LocalItem struct {
	ID        int      `yaml:"-"`
	Title     string   `yaml:"title"`
	State     string   `yaml:"state"` // open or closed, as written; see Local.StateOf
	Labels    []string `yaml:"labels,flow"`
	Milestone string   `yaml:"milestone,omitempty"`
	Branch    string   `yaml:"branch,omitempty"` // a merge request's local branch
	Base      string   `yaml:"base,omitempty"`
	Draft     bool     `yaml:"draft,omitempty"` // a merge request opened as a draft
	// Events are a merge request's labels set and taken off, the head
	// each was on (local_merge.go): a person's yes covers that head.
	Events   []LocalEvent `yaml:"events,omitempty"`
	Body     string       `yaml:"-"`
	Comments []string     `yaml:"-"`
	// Dates are when each comment was written, RFC 3339, in step with
	// Comments; "" for one written before the file dated them. A comment
	// with none yet is dated when the item is saved.
	Dates []string `yaml:"-"`
}

// commentLine starts each comment in an item's file. It is not a marker
// (Marker), so no comment's own marker is taken for it.
const commentLine = "<!-- workline-comment -->"

// commentDated starts a comment's line once it is dated: the line, then
// " at=" and its time, RFC 3339.
const commentDated = "<!-- workline-comment at="

func kindDir(kind string) string {
	if kind == "merge-request" {
		return "merge-requests"
	}
	return "issues"
}

// dir is .git/workline in the repository's main clone, which its worktrees share.
func (l *Local) dir() (string, error) {
	out, err := exec.Command("git", "-C", l.Repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("the local forge needs a git repository: %v", err)
	}
	return filepath.Join(strings.TrimSpace(string(out)), "workline"), nil
}

func (l *Local) path(kind string, id int) (string, error) {
	d, err := l.dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, kindDir(kind), fmt.Sprintf("%d.md", id)), nil
}

// Item reads one issue or merge request.
func (l *Local) Item(kind string, id int) (*LocalItem, error) {
	p, err := l.path(kind, id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, notHere{fmt.Sprintf("no %s", Target{Kind: kind, ID: id})}
	}
	it, err := parseLocal(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	it.ID = id
	return it, nil
}

// Items lists the issues, or the merge requests, by number.
func (l *Local) Items(kind string) ([]*LocalItem, error) {
	d, err := l.dir()
	if err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(filepath.Join(d, kindDir(kind), "*.md"))
	var out []*LocalItem
	for _, f := range files {
		id, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(f), ".md"))
		if err != nil {
			continue
		}
		it, err := l.Item(kind, id)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// StateOf is an item's state as it stands: a merge request whose branch is
// gone is closed, one whose branch its base holds is merged — the person
// merges it with git.
func (l *Local) StateOf(it *LocalItem) string {
	if it.State != "open" || it.Branch == "" {
		return it.State
	}
	if exec.Command("git", "-C", l.Repo, "rev-parse", "-q", "--verify", "refs/heads/"+it.Branch).Run() != nil {
		return "closed"
	}
	if it.Base != "" && it.Base != it.Branch &&
		exec.Command("git", "-C", l.Repo, "merge-base", "--is-ancestor", "refs/heads/"+it.Branch, "refs/heads/"+it.Base).Run() == nil {
		return "merged"
	}
	return "open"
}

// InCI names the variable that says the run is in CI, or "" outside it.
func InCI() string {
	for _, v := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "CI"} {
		if val := os.Getenv(v); val != "" && val != "false" && val != "0" {
			return v
		}
	}
	return ""
}

// writable refuses a write in CI: the job's clone is thrown away after it,
// and what the local forge holds with it. Reads stay allowed, so a project
// whose config says `local` for its laptops still runs in CI what writes
// nothing.
func (l *Local) writable() error {
	if v := InCI(); v != "" {
		return fmt.Errorf("the local forge in CI (%s is set): this clone is thrown away after the job, and what it holds with it; "+
			"pass --forge github, gitlab or cmd:<command> on this job, or --forge none to refuse the writes", v)
	}
	return nil
}

func (l *Local) save(kind string, it *LocalItem) error {
	if err := l.writable(); err != nil {
		return err
	}
	p, err := l.path(kind, it.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if it.Labels == nil {
		it.Labels = []string{}
	}
	head, err := yaml.Marshal(it)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(head)
	b.WriteString("---\n\n")
	if body := strings.Trim(it.Body, "\n"); body != "" {
		b.WriteString(body + "\n")
	}
	for len(it.Dates) < len(it.Comments) {
		it.Dates = append(it.Dates, wlclock.Now().UTC().Format(time.RFC3339))
	}
	for i, c := range it.Comments {
		line := commentLine
		if d := it.Dates[i]; d != "" {
			line = commentDated + d + " -->"
		}
		b.WriteString("\n" + line + "\n" + strings.Trim(c, "\n") + "\n")
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func parseLocal(s string) (*LocalItem, error) {
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		return nil, fmt.Errorf("no front matter")
	}
	head, rest, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		head, ok = strings.CutSuffix(rest, "\n---")
		if !ok {
			return nil, fmt.Errorf("front matter not closed")
		}
		rest = ""
	}
	it := &LocalItem{}
	if err := yaml.Unmarshal([]byte(head), it); err != nil {
		return nil, err
	}
	if it.State == "" {
		it.State = "open"
	}
	parts := []string{""}
	for _, line := range strings.Split(rest, "\n") {
		if line == commentLine {
			parts = append(parts, "")
			it.Dates = append(it.Dates, "")
			continue
		}
		if at, ok := strings.CutPrefix(line, commentDated); ok && strings.HasSuffix(at, " -->") {
			parts = append(parts, "")
			it.Dates = append(it.Dates, strings.TrimSuffix(at, " -->"))
			continue
		}
		parts[len(parts)-1] += line + "\n"
	}
	it.Body = strings.Trim(parts[0], "\n")
	for _, c := range parts[1:] {
		it.Comments = append(it.Comments, strings.Trim(c, "\n"))
	}
	return it, nil
}

// change reads an item, lets change edit it, and saves it.
func (l *Local) change(t Target, change func(*LocalItem)) error {
	it, err := l.Item(t.Kind, t.ID)
	if err != nil {
		return err
	}
	change(it)
	return l.save(t.Kind, it)
}

// next is the number a new item of this kind takes.
func (l *Local) next(kind string) (int, error) {
	items, err := l.Items(kind)
	if err != nil {
		return 0, err
	}
	id := 1
	for _, it := range items {
		id = max(id, it.ID+1)
	}
	return id, nil
}

func (l *Local) Issue(id int) (*Issue, error) {
	it, err := l.Item("issue", id)
	if err != nil {
		return nil, err
	}
	return &Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Closed: it.State != "open"}, nil
}

func (l *Local) Issues() ([]Issue, error) {
	items, err := l.Items("issue")
	if err != nil {
		return nil, err
	}
	var out []Issue
	for _, it := range items {
		if it.State == "open" {
			out = append(out, Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Milestone: it.Milestone, Insider: true})
		}
	}
	return out, nil
}

// AllIssues: the local forge keeps no close reason.
func (l *Local) AllIssues() ([]Issue, error) {
	items, err := l.Items("issue")
	if err != nil {
		return nil, err
	}
	var out []Issue
	for _, it := range items {
		out = append(out, Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels, Milestone: it.Milestone, Insider: true, Closed: it.State != "open"})
	}
	return out, nil
}

func (l *Local) Comments(t Target) ([]string, error) {
	it, err := l.Item(t.Kind, t.ID)
	if err != nil {
		return nil, err
	}
	return it.Comments, nil
}

// Notes: the local forge keeps no author; whoever writes in the clone is
// of the project, as its issues' authors are. workline's own carry its
// mark, as each comment a role writes does; the person's carry none.
func (l *Local) Notes(t Target) ([]Note, error) {
	it, err := l.Item(t.Kind, t.ID)
	if err != nil {
		return nil, err
	}
	notes := make([]Note, 0, len(it.Comments))
	for i, c := range it.Comments {
		n := Note{Body: c, Insider: true, ID: strconv.Itoa(i + 1), Own: strings.Contains(c, "<!-- workline:")}
		if i < len(it.Dates) {
			n.Created = it.Dates[i]
		}
		notes = append(notes, n)
	}
	return notes, nil
}

// LabelEvents: the local forge keeps no history: it does not say who set
// a label.
func (l *Local) LabelEvents(id int) ([]LabelEvent, error) { return nil, nil }

// React: the local forge has no reactions; nothing to do.
func (l *Local) React(id int, note, emoji string) error { return nil }

// Close closes an issue; the local forge keeps no reason, the engine's
// comment says it.
func (l *Local) Close(id, dup int) error {
	return l.change(Target{Kind: "issue", ID: id}, func(it *LocalItem) { it.State = "closed" })
}

func (l *Local) Comment(t Target, body, marker string) error {
	return l.change(t, func(it *LocalItem) {
		for _, c := range it.Comments {
			if strings.Contains(c, marker) {
				return
			}
		}
		it.Comments = append(it.Comments, body+"\n\n"+marker)
	})
}

func (l *Local) Sticky(t Target, body, marker string, create bool) error {
	return l.change(t, func(it *LocalItem) {
		for i, c := range it.Comments {
			if strings.Contains(c, marker) {
				it.Comments[i] = body + "\n\n" + marker
				return
			}
		}
		if create {
			it.Comments = append(it.Comments, body+"\n\n"+marker)
		}
	})
}

func (l *Local) Label(t Target, add, remove []string) error {
	return l.change(t, func(it *LocalItem) {
		if t.Kind == "merge-request" {
			l.labelled(it, add, remove)
		}
		for _, a := range add {
			if !slices.Contains(it.Labels, a) {
				it.Labels = append(it.Labels, a)
			}
		}
		it.Labels = slices.DeleteFunc(it.Labels, func(x string) bool { return slices.Contains(remove, x) })
	})
}

// openTitled is the open issue with this title, if any.
func (l *Local) openTitled(title string) (*LocalItem, error) {
	items, err := l.Items("issue")
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.Title == title && it.State == "open" {
			return it, nil
		}
	}
	return nil, nil
}

func (l *Local) OpenIssue(title, body, marker string) (int, error) {
	it, err := l.openTitled(title)
	if err != nil {
		return 0, err
	}
	if it != nil {
		return it.ID, l.Comment(Target{Kind: "issue", ID: it.ID}, body, marker)
	}
	id, err := l.next("issue")
	if err != nil {
		return 0, err
	}
	return id, l.save("issue", &LocalItem{ID: id, Title: title, State: "open", Body: body + "\n\n" + marker})
}

// Open opens an issue as a person does on a forge: a new number each
// time, whatever the titles of the others, its labels set, each once
// as on a forge (#1284).
func (l *Local) Open(title, body string, labels []string) (int, error) {
	id, err := l.next("issue")
	if err != nil {
		return 0, err
	}
	var set []string
	for _, a := range labels {
		if !slices.Contains(set, a) {
			set = append(set, a)
		}
	}
	return id, l.save("issue", &LocalItem{ID: id, Title: title, State: "open", Labels: set, Body: body})
}

// Answer adds a person's comment after the others, as written: no
// marker, so two alike are two comments, as on a forge (#1284).
func (l *Local) Answer(t Target, body string) error {
	return l.change(t, func(it *LocalItem) {
		it.Comments = append(it.Comments, body)
		it.Dates = append(it.Dates[:len(it.Comments)-1], "") // a person's comment is left undated, as written
	})
}

func (l *Local) KeepIssue(title, body string, create bool) (int, error) {
	it, err := l.openTitled(title)
	if err != nil {
		return 0, err
	}
	if it != nil {
		it.Body = body
		return it.ID, l.save("issue", it)
	}
	if !create {
		return 0, nil
	}
	id, err := l.next("issue")
	if err != nil {
		return 0, err
	}
	return id, l.save("issue", &LocalItem{ID: id, Title: title, State: "open", Body: body})
}

func (l *Local) OpenMergeRequest(branch, base, title, body string, draft bool) (int, error) {
	items, err := l.Items("merge-request")
	if err != nil {
		return 0, err
	}
	for _, it := range items {
		if it.Branch == branch && l.StateOf(it) == "open" {
			it.Title, it.Body = title, body
			return it.ID, l.save("merge-request", it)
		}
	}
	id, err := l.next("merge-request")
	if err != nil {
		return 0, err
	}
	return id, l.save("merge-request", &LocalItem{ID: id, Title: title, State: "open", Branch: branch, Base: base, Body: body, Draft: draft})
}

func (l *Local) OpenMergeRequests(prefix string) ([]string, error) {
	items, err := l.Items("merge-request")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, it := range items {
		if strings.HasPrefix(it.Branch, prefix) && l.StateOf(it) == "open" {
			out = append(out, it.Branch)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (l *Local) CloseMergeRequest(branch, body, marker string) error {
	items, err := l.Items("merge-request")
	if err != nil {
		return err
	}
	for _, it := range items {
		if it.Branch == branch && l.StateOf(it) == "open" {
			// One write, its comment and its state: never said closed and left open.
			if !slices.ContainsFunc(it.Comments, func(c string) bool { return strings.Contains(c, marker) }) {
				it.Comments = append(it.Comments, body+"\n\n"+marker)
			}
			it.State = "closed"
			return l.save("merge-request", it)
		}
	}
	return nil
}

func (l *Local) MergeRequest(id int) (MergeRequest, error) {
	it, err := l.Item("merge-request", id)
	if err != nil {
		return MergeRequest{}, err
	}
	return MergeRequest{ID: id, Branch: it.Branch, Base: it.Base, Here: true, Title: it.Title, Body: it.Body, Draft: it.Draft, Author: localAuthor(it.Branch), Labels: it.Labels}, nil
}

// MarkReady marks a draft merge request ready; one ready already is left.
func (l *Local) MarkReady(id int) error {
	it, err := l.Item("merge-request", id)
	if err != nil || !it.Draft {
		return err
	}
	it.Draft = false
	return l.save("merge-request", it)
}

// ListMergeRequests lists the open merge requests: each from a branch of
// this clone, by whoever wrote it.
func (l *Local) ListMergeRequests() ([]MergeRequest, error) {
	items, err := l.Items("merge-request")
	if err != nil {
		return nil, err
	}
	var out []MergeRequest
	for _, it := range items {
		if it.Branch != "" && l.StateOf(it) == "open" {
			out = append(out, MergeRequest{ID: it.ID, Branch: it.Branch, Base: it.Base, Here: true, Title: it.Title, Body: it.Body, Labels: it.Labels, Draft: it.Draft, Author: localAuthor(it.Branch)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (l *Local) keepsBranches() {}

// Milestones are the ones its open issues are in: the local forge keeps no
// milestone apart from them.
func (l *Local) Milestones() ([]Milestone, error) {
	open, err := l.Issues()
	if err != nil {
		return nil, err
	}
	var titles []string
	for _, is := range open {
		if is.Milestone != "" && !slices.Contains(titles, is.Milestone) {
			titles = append(titles, is.Milestone)
		}
	}
	sort.Strings(titles)
	var out []Milestone
	for _, t := range titles {
		out = append(out, Milestone{Title: t}) // no due date: the local forge keeps none
	}
	return out, nil
}

// EnsureLabel: the local forge keeps no list of labels apart from its items.
func (l *Local) EnsureLabel(name, color, description string) error { return nil }

func (l *Local) SetBody(id int, body string) error {
	return l.change(Target{Kind: "issue", ID: id}, func(it *LocalItem) { it.Body = body })
}

func (l *Local) SetTitle(id int, title string) error {
	return l.change(Target{Kind: "issue", ID: id}, func(it *LocalItem) { it.Title = title })
}

// AddSubIssue: the local forge has no sub-issues; the parent's body lists
// its children.
func (l *Local) AddSubIssue(parent, child int) (bool, error) { return false, nil }

// AddBlocker: the local forge has no relation between issues; the body
// says what an issue waits on (ADR-0028).
func (l *Local) AddBlocker(id, blocker int) (bool, error) { return false, nil }

// RemoveBlocker: no relation, nothing to take off; the body's line is the
// caller's.
func (l *Local) RemoveBlocker(id, blocker int) error { return nil }

func (l *Local) SetMilestone(id int, title string) error {
	return l.change(Target{Kind: "issue", ID: id}, func(it *LocalItem) { it.Milestone = title })
}

// Closers: the local forge does not link what closed an issue.
func (l *Local) Closers(id int) ([]Closer, error) { return nil, nil }

// Trail: the local forge keeps no history, nor links: it does not say
// when an issue got a label (ADR-0031).
func (l *Local) Trail(id int, label string) (Trail, error) { return Trail{}, nil }

// Settled is a merge request merged outside the forge, and the issues its
// merge closed.
type Settled struct {
	MergeRequest int
	Base         string
	Issues       []int
}

// Settler is a forge whose merge requests are merged outside it, with git:
// what a forge does on a merge, closing the issues the request says it
// closes, is done when it is asked.
type Settler interface {
	// SettleMerges closes the issues each merge request merged since says
	// it closes (`Closes #n`), once: the request is written merged.
	SettleMerges() ([]Settled, error)
}

var _ Settler = &Local{}

// SettleMerges closes, once each, the open issues a merge request merged
// with git says it closes in its body, as a forge does on its merge
// (#1282). The request is then written merged: an issue a person opens
// again stays open.
func (l *Local) SettleMerges() ([]Settled, error) {
	items, err := l.Items("merge-request")
	if err != nil {
		return nil, err
	}
	var out []Settled
	for _, it := range items {
		if it.State != "open" || l.StateOf(it) != "merged" {
			continue
		}
		s := Settled{MergeRequest: it.ID, Base: it.Base}
		for _, n := range Closes(it.Body) {
			is, err := l.Item("issue", n)
			if err != nil || is.State != "open" {
				continue // none here, or closed already
			}
			if err := l.Comment(Target{Kind: "issue", ID: n}, fmt.Sprintf("Closed by !%d, merged into %s.", it.ID, it.Base), Marker(fmt.Sprintf("closed-by=%d", it.ID))); err != nil {
				return out, err
			}
			if err := l.Close(n, 0); err != nil {
				return out, err
			}
			s.Issues = append(s.Issues, n)
		}
		it.State = "merged"
		if err := l.save("merge-request", it); err != nil {
			return out, err
		}
		out = append(out, s)
	}
	return out, nil
}
