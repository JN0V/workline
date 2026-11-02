package backlog

// An issue's kind is the forge's (ADR-0039): bug, feature or task, read
// from its labels, never from an agent's say. The product owner sets one
// where none is (the `kind` act, ADR-0039 (c)); a person's correction
// holds. The kind decides who validates (ADR-0060, decision 4): a bug's
// fix merges by the policy once a test red on the base proves the bug, a
// feature waits for a person's yes at `normal`.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// The kinds of an issue.
const (
	KindBug     = "bug"
	KindFeature = "feature"
	KindTask    = "task"
)

// IssueKinds are the kinds, in the order the act names them.
var IssueKinds = []string{KindBug, KindFeature, KindTask}

// LabelNeedsRepro marks a bug the tester could not reproduce: its test
// passed on the base. Set and cleared by the engine (ADR-0039); past the
// product owner's delay with no answer, the issue is Blocked as "not
// reproduced" (ADR-0060).
const LabelNeedsRepro = "workline:needs-repro"

// NotReproduced opens the reason the line blocks such a bug with.
const NotReproduced = "not reproduced"

// KindLabels are the labels that tell each kind, matched whole, case
// aside; the first of each is the one the product owner sets when the
// project has none of them (`kinds` names the project's own).
var KindLabels = map[string][]string{
	KindBug:     {"bug", "fix", "type::bug", "kind/bug"},
	KindFeature: {"enhancement", "feature", "type::feature", "kind/feature"},
	KindTask:    {"task", "chore", "type::maintenance", "kind/task"},
}

// KindOf is the kind an issue's labels give it, and the label that told
// it: a feature over a bug over a task, on the safe side; "" when none
// says it.
func KindOf(labels []string) (kind, label string) {
	for _, k := range []string{KindFeature, KindBug, KindTask} {
		for _, l := range labels {
			if slices.ContainsFunc(KindLabels[k], func(w string) bool { return strings.EqualFold(w, strings.TrimSpace(l)) }) {
				return k, l
			}
		}
	}
	return "", ""
}

// KindLabel is the label the act sets for kind: the first of the
// project's names for it (`kinds`, else KindLabels) the project has, else
// the first name, which the forge creates as it labels the issue.
func KindLabel(kind string, names map[string][]string, have []string) string {
	list := names[kind]
	if len(list) == 0 {
		list = KindLabels[kind]
	}
	for _, n := range list {
		if slices.ContainsFunc(have, func(h string) bool { return strings.EqualFold(h, n) }) {
			return n
		}
	}
	if len(list) == 0 {
		return ""
	}
	return list[0]
}

// checkKind checks a kind act: before the issue's state is read (st nil),
// a kind it knows, on an open issue no kind label tells already — never
// a swap —, and the label it sets; once read, that the role never set a
// kind on it before: a person's change or removal since stands.
func (p *Plan) checkKind(f forge.Backlog, c *Proposal, st *State) (rule, why string) {
	is, ok := p.issues[c.Issue]
	switch {
	case !ok || c.Issue == p.Report:
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	case st != nil && st.Kind != "":
		return "kind-once", fmt.Sprintf("the role set its kind once already (`%s`): never again; a person's change or removal stands", st.Kind)
	case st != nil:
		return "", ""
	}
	c.IssueKind = strings.ToLower(strings.TrimSpace(c.IssueKind))
	if !slices.Contains(IssueKinds, c.IssueKind) {
		return "kind-unknown", fmt.Sprintf("`%s` is no kind: %s", c.IssueKind, strings.Join(IssueKinds, ", "))
	}
	if _, label := KindOf(is.Labels); label != "" {
		return "kind-kept", fmt.Sprintf("its label `%s` says its kind already: a kind is never swapped", label)
	}
	if p.labels == nil {
		var open []forge.Issue
		for _, o := range p.issues {
			open = append(open, o)
		}
		labels, err := forge.ProjectLabels(f, open)
		if err != nil {
			// Never a default label in place of the project's own, unseen:
			// a duplicate kind label would follow (principle 12).
			return "kind-labels-unread", "the project's labels could not be listed (" + err.Error() + "): which one tells the kind is not known"
		}
		p.labels = labels
		if p.labels == nil {
			p.labels = []string{}
		}
	}
	c.KindLabel = KindLabel(c.IssueKind, p.config.Kinds, p.labels)
	return "", ""
}

// KindNames reads the `kinds` setting: each kind's labels, the first that
// exists set; none for a kind, KindLabels'.
func KindNames(settings map[string]any) map[string][]string {
	out := map[string][]string{}
	m, _ := settings["kinds"].(map[string]any)
	for _, k := range IssueKinds {
		l, _ := m[k].([]any)
		for _, v := range l {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				out[k] = append(out[k], strings.TrimSpace(s))
			}
		}
	}
	return out
}
