package developer

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// The merge of the developer's pull request (ADR-0054, step 7): at
// `normal`, a fix or a task merges by the policy once every gate passes,
// a fix once its bug was reproduced too, by the tester's test red on the
// base (ADR-0060); a feature waits for a person's yes. The kind is its issue's, as the
// forge's labels say it (ADR-0039), never the agent's say.

// The kinds of the issue a developer's pull request closes.
const (
	KindFix     = "fix"
	KindTask    = "task"
	KindFeature = "feature"
)

// Unreproduced marks the merge policy's word on a fix whose bug no test
// red on the base showed: it waits for a person's yes (ADR-0060).
const Unreproduced = "unreproduced"

// IssueKind is the kind its labels give an issue, and the label that
// told it (backlog.KindOf): a feature over a fix over a task, on the safe
// side; "" when none says it.
func IssueKind(labels []string) (kind, label string) {
	k, l := backlog.KindOf(labels)
	switch k {
	case backlog.KindBug:
		return KindFix, l
	case backlog.KindFeature:
		return KindFeature, l
	case backlog.KindTask:
		return KindTask, l
	}
	return "", ""
}

// Merge is what the merge policy reads of a developer's pull request.
type Merge struct {
	Issue int    // the issue its record names
	Kind  string // the issue's kind; "" when no label says it
	Label string // the label that told the kind
	// Open are the reviewer's findings on it that block, or warn and are
	// Important, as its comment says them now: a nit holds nothing.
	Open []string
	// Why it is not read as the developer's, or not whole: "" when it is.
	Why string
	// Reproduced: its code starts from the tester's Draft, whose
	// acceptance tests the tester proved red on the base before opening it
	// (ADR-0055): a bug shown before it is fixed (ADR-0060). The record
	// names the Draft's commit and its tests; the branch's commits say
	// the rest (the merge policy checks one is the tester's).
	Reproduced bool
	// Milestone is the issue's milestone, "" for none: a feature merging
	// at `enterprising` goes into one, its release candidate (ADR-0060).
	Milestone string
}

// Says says the pull request for a person: whose code, for which issue,
// of which kind and why.
func (m Merge) Says() string {
	if m.Kind == "" {
		return fmt.Sprintf("the developer's code for #%d, of no kind its labels say", m.Issue)
	}
	return fmt.Sprintf("the developer's code for #%d, a %s: its issue labelled `%s`", m.Issue, m.Kind, m.Label)
}

// ReadMerge reads a developer's pull request for the merge: its record,
// workline's own comment on it, the issue it names and that issue's kind,
// and the reviewer's findings still open on it.
func ReadMerge(f forge.Forge, mr forge.MergeRequest) (Merge, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: mr.ID})
	if err != nil {
		return Merge{}, err
	}
	body, whole := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	a, read := ReadAct(body)
	switch {
	case !whole || !read || a.Act != "code":
		return Merge{Why: "the developer's record on it, workline's own comment, cannot be read"}, nil
	case a.Issue == 0 || a.Issue != developerIssue(mr.Branch):
		return Merge{Why: fmt.Sprintf("the developer's record on it names #%d, not its branch's issue", a.Issue)}, nil
	case a.Handed:
		return Merge{Issue: a.Issue, Why: "the developer handed it to a person past its last review round"}, nil
	}
	m := Merge{Issue: a.Issue, Reproduced: a.From != "" && len(a.Acceptance) > 0}
	is, err := f.Issue(a.Issue)
	if err != nil {
		// One pull request's issue gone, or unreadable, holds that one
		// only: the follow run goes on with the others.
		m.Why = fmt.Sprintf("#%d, the issue it closes, cannot be read: %v", a.Issue, err)
		return m, nil
	}
	m.Kind, m.Label = IssueKind(is.Labels)
	m.Milestone = is.Milestone
	for _, marker := range reviewerMarkers {
		c, whole := kept.Last(marker)
		switch {
		case c == "":
			continue
		case !whole:
			m.Why = "the reviewer's comment on it was edited by another: its findings cannot be read"
			return m, nil
		}
		asked, _ := reviewRows(c)
		for _, x := range asked {
			m.Open = append(m.Open, fmt.Sprintf("%s %s `%s`: %s", x.Holds, x.Lens, x.Where, x.Text))
		}
	}
	return m, nil
}

// MergeSaid is how the merge policy decided on a developer's pull request,
// from workline's own record on it, for the auditor (ADR-0046): merged
// with no person's yes, its kind and why, or held for one; "" when the
// policy left no record: a person's yes then.
func MergeSaid(f forge.Forge, id int) (string, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		return "", err
	}
	for _, marker := range []string{"merge-alone=code", "merge-waits=" + KindFeature, "merge-waits=" + Unreproduced} {
		if body, whole := kept.Last(forge.Marker(marker)); body != "" && whole {
			said, _, _ := strings.Cut(body, "\n")
			return strings.TrimSpace(said), nil
		}
	}
	return "", nil
}
