// Package forge applies what reaches a forge — comments, labels, issues,
// merge requests — on GitHub, GitLab, or a simulated forge for the tests. Every
// write is idempotent, so a run interrupted half-way can be resumed.
package forge

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Target is what a comment or a label goes on.
type Target struct {
	Kind string `json:"kind" yaml:"kind"` // "issue" or "merge-request"
	ID   int    `json:"id" yaml:"id"`
}

func (t Target) String() string { return fmt.Sprintf("%s #%d", t.Kind, t.ID) }

// Issue is what the line reads from a work item.
type Issue struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels"`
	Closed    bool     `json:"closed,omitempty"`
	Reason    string   `json:"reason,omitempty"`    // why it was closed, when the forge says: completed, not_planned, duplicate
	Milestone string   `json:"milestone,omitempty"` // the title of the milestone it is in, if any
	// MilestoneDue is its milestone's due date, YYYY-MM-DD, when it has
	// one: the backlog's order ranks milestones by it (Milestone).
	MilestoneDue string `json:"milestone-due,omitempty"`
	Author       string `json:"author,omitempty"`  // who opened it
	Insider      bool   `json:"insider,omitempty"` // its author is a person of the project (GitHub: owner, member, collaborator; GitLab: Planner or above); false when the forge does not say
	// BlockedBy are the issues it waits on in the forge's own relation
	// (GitHub's dependencies, GitLab's is_blocked_by), open or closed, as
	// Issues gives them; a line in its body says the rest (ADR-0028).
	BlockedBy []int `json:"blocked-by,omitempty"`
	// Children are its sub-issues in the forge's own relation (GitHub's
	// sub-issues, GitLab's tasks), open or closed, as Issues gives them; a
	// task list under "## Sub-issues" in its body says the rest (ADR-0029).
	Children []int `json:"children,omitempty"`
}

// Milestone is an open milestone: its title, and its due date when it has
// one, YYYY-MM-DD — what the backlog's order ranks milestones by first,
// the title after (docs/spec/backlog-acts.md, "Ordering").
type Milestone struct {
	Title string `json:"title"`
	Due   string `json:"due,omitempty"`
}

// Closer is what closed an issue, as the forge links it: a pull or merge
// request merged, or a commit, with its text — the evidence a parent's
// report quotes (ADR-0029).
type Closer struct {
	Kind string `json:"kind"`           // pull-request (a GitHub pull request, a GitLab merge request) or commit
	Ref  string `json:"ref"`            // how the forge names it: "#20", "!7", a short commit
	Text string `json:"text,omitempty"` // its title and description, or the commit's message
}

// Note is a comment with who wrote it: what a reply decides counts only
// from the right people (ADR-0021).
type Note struct {
	Body    string `json:"body"`
	Author  string `json:"author,omitempty"`  // who wrote it; "" when the forge does not say
	Insider bool   `json:"insider,omitempty"` // its author is a person of the project, as Issue.Insider
	Bot     bool   `json:"bot,omitempty"`     // its author is a bot: a token's user, an app
	Created string `json:"created,omitempty"` // when it was written, RFC 3339 or YYYY-MM-DD; "" when the forge does not say
	// ID is the forge's own id of the comment, to put a reaction on it
	// (React); "" when the forge does not say.
	ID string `json:"id,omitempty"`
}

// LabelEvent is a label set on an issue, or taken off, with who did it
// and when (ADR-0038): Author, Insider and Bot as a Note's; Created, when.
type LabelEvent struct {
	Label string `json:"label"`
	Added bool   `json:"added,omitempty"` // set; false: taken off
	Note
}

// Eyes is the reaction the role puts on a comment it read: 👀, as the
// forges name it.
const Eyes = "eyes"

// Trail is what a forge says of an open issue since it got a label: when
// it last got it, and the pull or merge requests and commits that name
// the issue, each with when — what started it, or nothing (ADR-0031).
type Trail struct {
	Labeled string `json:"labeled,omitempty"` // when it last got the label, RFC 3339 or YYYY-MM-DD; "" when the forge does not say
	Links   []Link `json:"links,omitempty"`
}

// Link is a pull or merge request, or a commit, naming an issue.
type Link struct {
	Kind string `json:"kind"`         // pull-request (a GitHub pull request, a GitLab merge request) or commit
	Ref  string `json:"ref"`          // how the forge names it: "#20", "!7", a short commit
	At   string `json:"at,omitempty"` // when it named the issue, RFC 3339 or YYYY-MM-DD
}

// Tick is a box of a task list ticked, or unticked, in an issue's body,
// with who did it: a person's yes in a report (ADR-0025).
type Tick struct {
	Item string `json:"item"`           // the item's text, as the forge gives it: its hidden markers' text in it
	Done bool   `json:"done,omitempty"` // ticked; false: unticked
	Note        // its author: Author, Insider, Bot; no author and not Insider when the forge does not say
}

// Bodies is the text of each note, in order.
func Bodies(notes []Note) []string {
	out := make([]string, 0, len(notes))
	for _, n := range notes {
		out = append(out, n.Body)
	}
	return out
}

// Forge is what the engine needs from one.
type Forge interface {
	Issue(id int) (*Issue, error)
	// Comment posts body on t unless a comment carrying marker is already there.
	Comment(t Target, body, marker string) error
	// Sticky keeps one comment carrying marker on t, edited to body on each
	// run instead of a new one each time; with create false, it only edits
	// one already there.
	Sticky(t Target, body, marker string, create bool) error
	// Label adds and removes labels; adding one already there changes nothing.
	Label(t Target, add, remove []string) error
	// OpenIssue creates an issue, or comments on an open one with the same title.
	OpenIssue(title, body, marker string) (int, error)
	// KeepIssue rewrites the body of the open issue with this title, or opens
	// it when there is none and create is true: one issue, kept in place.
	KeepIssue(title, body string, create bool) (int, error)
	// OpenMergeRequest opens a merge request from branch into base, or
	// updates the title and body of the one already open from branch.
	OpenMergeRequest(branch, base, title, body string) (int, error)
	// OpenMergeRequests lists the branches of the open merge requests whose
	// branch starts with prefix, sorted.
	OpenMergeRequests(prefix string) ([]string, error)
	// MergeRequest says where a merge request comes from and goes.
	MergeRequest(id int) (MergeRequest, error)
}

// Backlog is what a role acting on a project's issues needs from its forge
// (docs/spec/backlog-acts.md). Every forge workline speaks has it.
type Backlog interface {
	// Issues lists the open issues, merge requests left out, by number,
	// each with the issues it waits on in the forge's own relation.
	Issues() ([]Issue, error)
	// AllIssues lists the issues open and closed, merge requests left out,
	// by number: a subject a role found is looked for in both (OpenOnce).
	AllIssues() ([]Issue, error)
	// Comments lists the comments on t, oldest first.
	Comments(t Target) ([]string, error)
	// Notes lists the comments on t with their authors, oldest first, as
	// Comments does.
	Notes(t Target) ([]Note, error)
	// Close closes an issue: as a duplicate of dup when dup > 0, else as
	// completed. Closing one already closed changes nothing.
	Close(id, dup int) error
	// Milestones lists the open milestones, with their due dates.
	Milestones() ([]Milestone, error)
	// SetMilestone puts an issue in the open milestone with this title,
	// creating it when there is none.
	SetMilestone(id int, title string) error
	// SetBody rewrites an issue's body.
	SetBody(id int, body string) error
	// SetTitle renames an issue.
	SetTitle(id int, title string) error
	// AddSubIssue makes child a sub-issue of parent where the forge has
	// sub-issues, and says so; false on a forge that has none, the child
	// then listed in the parent's body by the caller. Adding one already
	// there changes nothing.
	AddSubIssue(parent, child int) (bool, error)
	// AddBlocker records that id waits on blocker in the forge's own
	// relation, and says so; false on a forge that has none — the caller
	// then writes a line in the body (ADR-0028). Adding one already there
	// changes nothing.
	AddBlocker(id, blocker int) (bool, error)
	// RemoveBlocker takes blocker off what id waits on in the forge's own
	// relation; one not there, or a forge without the relation, changes
	// nothing. Only the role's own links are taken off (ADR-0028).
	RemoveBlocker(id, blocker int) error
	// Closers lists what closed an issue, as the forge links it: the pull
	// or merge request, or the commit; nil when the forge does not say.
	Closers(id int) ([]Closer, error)
	// Ticks lists the boxes ticked and unticked in an issue's body, oldest
	// first, with who did each; nil when the forge does not say.
	Ticks(id int) ([]Tick, error)
	// Trail says when an open issue last got a label, and what pull or
	// merge requests and commits name it; an empty Labeled when the
	// forge does not say (ADR-0031).
	Trail(id int, label string) (Trail, error)
	// EnsureLabel creates a label when the project has none of that name,
	// so a person finds it in the forge's list to set.
	EnsureLabel(name, color, description string) error
	// LabelEvents lists the labels set on an issue and taken off, oldest
	// first, with who did each and when; nil when the forge does not say
	// (ADR-0038).
	LabelEvents(id int) ([]LabelEvent, error)
	// React puts a reaction (Eyes) on a comment of an issue, by the note's
	// ID; one there already changes nothing, nor does a forge without
	// reactions (ADR-0038).
	React(id int, note, emoji string) error
}

// ScopedLabels says whether f's labels are scoped, `key::value`, as
// GitLab's: the product owner's own labels are named so there (ADR-0038).
func ScopedLabels(f any) bool {
	s, ok := f.(interface{ scopedLabels() bool })
	return ok && s.scopedLabels()
}

// LabelFilter is the address of f's open issues bearing label, a person's
// saved filter; "" when the forge has no such page.
func LabelFilter(f any, label string) string {
	if l, ok := f.(interface{ labelFilter(string) string }); ok {
		return l.labelFilter(label)
	}
	return ""
}

// MergeRequest is where a merge request comes from and where it goes.
type MergeRequest struct {
	Branch string // the branch it comes from
	Base   string // the branch it goes into; "" when the forge does not say
	Here   bool   // the branch lives in this repository, not in a fork
	Title  string // what its author calls it
	Body   string // what its author says of it: testimony for a review (ADR-0020)
}

// ErrUnreachable marks a forge that did not answer: the run is blocked by
// something outside the role.
var ErrUnreachable = errors.New("forge unreachable")

// Open returns the forge named by spec: "github", "gitlab", "local" (kept in
// the clone), "cmd:<command>" (another forge, plugged by a command),
// "fake:<file>", or "" / "none" for no forge: what needs one is refused, and
// says so.
func Open(spec, repo string) (Forge, error) {
	switch {
	case spec == "" || spec == "none":
		return nil, nil
	case spec == "github":
		return &github{repo: repo}, nil
	case spec == "gitlab":
		return &gitlab{repo: repo}, nil
	case spec == "local":
		return &Local{Repo: repo}, nil
	case strings.HasPrefix(spec, "cmd:"):
		if strings.TrimSpace(strings.TrimPrefix(spec, "cmd:")) == "" {
			return nil, fmt.Errorf("forge %q: name the command to run (cmd:<command>)", spec)
		}
		return &command{repo: repo, script: strings.TrimPrefix(spec, "cmd:")}, nil
	case strings.HasPrefix(spec, "fake:"):
		return &Fake{Path: strings.TrimPrefix(spec, "fake:")}, nil
	}
	return nil, fmt.Errorf("unknown forge %q (github, gitlab, local, cmd:<command>, fake:<file>, none)", spec)
}

// Missing says what to do when a write needs a forge and the project has none.
const Missing = "set `forge:` in .workline/config.yaml or pass --forge: github, gitlab, cmd:<command>, or `forge: local` to keep it in this clone"

// KeepsBranches says whether f's merge requests come from local branches:
// nothing is pushed to a remote, nor fetched from one.
func KeepsBranches(f Forge) bool {
	_, ok := f.(interface{ keepsBranches() })
	return ok
}

// Marker is the hidden text that makes a comment findable again.
func Marker(key string) string { return "<!-- workline:" + key + " -->" }

// Every forge workline speaks keeps a backlog.
var _ = []Backlog{&github{}, &gitlab{}, &Local{}, &command{}, &Fake{}}

// taskLine is a task list's item in a body: `- [ ] text`, `* [x] text`.
var taskLine = regexp.MustCompile(`(?m)^\s*[-*+] \[([ xX])\] (.*?)\s*$`)

// TaskItems maps each task list item of a body to whether it is ticked.
func TaskItems(body string) map[string]bool {
	out := map[string]bool{}
	for _, m := range taskLine.FindAllStringSubmatch(body, -1) {
		out[m[2]] = m[1] != " "
	}
	return out
}

// TicksBetween are the boxes a version of a body ticked or unticked from
// the one before ("" for none), each given to who wrote that version.
func TicksBetween(before, after string, who Note) []Tick {
	was := TaskItems(before)
	var out []Tick
	for _, m := range taskLine.FindAllStringSubmatch(after, -1) {
		item, done := m[2], m[1] != " "
		if prev, ok := was[item]; (ok && prev != done) || (!ok && done) {
			out = append(out, Tick{Item: item, Done: done, Note: who})
		}
	}
	return out
}
