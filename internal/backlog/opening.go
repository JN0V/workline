package backlog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// LabelTriage marks an issue a role opened on a finding of its own, until a
// person or the product owner takes it (ADR-0018, "Opening issues, for every
// role").
const LabelTriage = "needs-triage"

// Keeper is the role whose state comment an opened issue is given: the
// product owner reads it from its next run, as any other issue.
const Keeper = "product-owner"

// Outcomes of an opening.
const (
	Opened     = "opened"      // a new issue
	StillOpen  = "open"        // an open issue holds the subject: nothing written
	Settled    = "settled"     // closed as not planned or as a duplicate: a person's no, nothing written
	FoundAgain = "found-again" // closed otherwise, the subject found again: said once on it, not reopened
	Capped     = "capped"      // new, past the run's cap: not opened, found again at a later run
)

// Opening is a subject a role opens an issue for.
type Opening struct {
	Role    string   // the role that found it, named in the issue
	Key     string   // the marker's key: issue=<subject>, or import=<a file's text>
	Also    []string // older keys the same subject was opened under
	Title   string
	Body    string
	From    string   // where it was opened from, said after "Opened": " from `ROADMAP.md`"
	Sources []string // the code it is about, for the keeper's state
	Commit  string   // the commit it was seen at
	Triage  bool     // a role's finding: labelled needs-triage, capped, said once when found after it was closed
}

// Openings is the one way every role opens an issue (ADR-0018): a stable
// key per subject, hidden in the issue's body; the issues open and closed
// looked in, once a run; no AI. A subject an issue holds is never opened
// again: open, it is left as it is; closed as not planned or as a
// duplicate, it is a person's no, left as it is; closed otherwise, it is
// said once on it that it was found again, and it stays closed.
type Openings struct {
	Max    int // a run's findings opened as issues, at most; the rest Capped
	opened int
	f      forge.Forge
	b      forge.Backlog
	issues []forge.Issue // read on the first opening, kept up to date by the run's own
}

// NewOpenings prepares a run's openings on f, at most max findings opened.
func NewOpenings(f forge.Forge, max int) (*Openings, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, errors.New("this forge cannot list issues, to open one only once")
	}
	return &Openings{f: f, b: b, Max: max}, nil
}

// holder is the issue holding one of keys: an open one first.
func (o *Openings) holder(keys []string) (*forge.Issue, error) {
	if o.issues == nil {
		all, err := o.b.AllIssues()
		if err != nil {
			return nil, err
		}
		o.issues = append([]forge.Issue{}, all...)
	}
	var closed *forge.Issue
	for i, is := range o.issues {
		if !slices.ContainsFunc(keys, func(k string) bool { return strings.Contains(is.Body, forge.Marker(k)) }) {
			continue
		}
		if !is.Closed {
			return &o.issues[i], nil
		}
		if closed == nil {
			closed = &o.issues[i]
		}
	}
	return closed, nil
}

// Open opens the issue for op once, and says what became of it, with the
// issue holding it.
func (o *Openings) Open(op Opening) (string, int, error) {
	if op.Key == "" || op.Role == "" {
		return "", 0, errors.New("an issue a role opens needs its key and its role")
	}
	is, err := o.holder(append([]string{op.Key}, op.Also...))
	if err != nil {
		return "", 0, err
	}
	switch {
	case is != nil && !is.Closed:
		return StillOpen, is.ID, nil
	case is != nil && (is.Reason == "not_planned" || is.Reason == "duplicate" || !op.Triage):
		return Settled, is.ID, nil
	case is != nil:
		// Closed as done, or for a reason the forge does not say: the
		// subject is back, a regression perhaps. Said once there, for a
		// person to reopen; a role never reopens one.
		t := forge.Target{Kind: "issue", ID: is.ID}
		say := fmt.Sprintf("Found again by the %s role at %s. Left closed: reopen it if it is back; this is said once.", op.Role, op.Commit)
		return FoundAgain, is.ID, o.f.Comment(t, say, forge.Marker("found-again"))
	case op.Triage && o.opened >= o.Max:
		return Capped, 0, nil
	}
	if op.Triage {
		o.opened++
	}
	body := fmt.Sprintf("%s\n\nOpened%s by the %s role.\n\n%s", strings.TrimRight(op.Body, "\n"), op.From, op.Role, forge.Marker("opened-by="+op.Role))
	id, err := o.f.OpenIssue(strings.TrimSpace(op.Title), body, forge.Marker(op.Key))
	if err != nil {
		return "", 0, err
	}
	o.issues = append(o.issues, forge.Issue{ID: id, Title: op.Title, Body: body + "\n\n" + forge.Marker(op.Key)})
	t := forge.Target{Kind: "issue", ID: id}
	if op.Triage {
		if err := o.b.EnsureLabel(LabelTriage, "ededed", "Opened by a workline role: a person or the product owner takes it from here"); err != nil {
			return "", 0, err
		}
		if err := o.f.Label(t, []string{LabelTriage}, nil); err != nil {
			return "", 0, err
		}
	}
	st := State{Sources: op.Sources, Confirmed: op.Commit}
	return Opened, id, o.f.Sticky(t, FormatState(st), StateMarker(Keeper), true)
}

// OpenedBy is the role that opened an issue through Openings, or "".
func OpenedBy(body string) string {
	_, rest, ok := strings.Cut(body, "<!-- workline:opened-by=")
	if !ok {
		return ""
	}
	role, _, _ := strings.Cut(rest, " -->")
	return role
}

// CodeKey is the subject of a finding on a line of code: its file and the
// line as it reads, spaces aside. Two roles finding the same line get the
// same key; the line changed, it is another subject.
func CodeKey(path, line string) string {
	sum := sha256.Sum256([]byte(path + "\n" + squeeze(line)))
	return "issue=" + path + "#" + hex.EncodeToString(sum[:])[:8]
}

// TitleKey is the subject of a finding that names no code: its title,
// spaces and case aside.
func TitleKey(title string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(squeeze(title))))
	return "issue=title#" + hex.EncodeToString(sum[:])[:8]
}
