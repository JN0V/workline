package backlog

// A blocked issue read again (ADR-0059, amended): the line wakes the role
// on it when there is news, and the role sends it back — to the state it
// came from, or to refine — or keeps it blocked, its reason said afresh.
// Both done alone: a label put back undoes the first, the second only
// says.

import (
	"fmt"
	"slices"
)

// BlockedElsewhere opens the line's reason for a block it did not make:
// the reason is in the comment of whoever blocked it.
const BlockedElsewhere = "a role or a person blocked it"

// stateLabels are the labels of the line's states as shipped
// (routing.default.yaml), by the state's name.
var stateLabels = map[string]string{"refine": LabelToRefine, "tests": LabelNeedsTests, "ready": LabelReady, "doing": "workline:developing", "blocked": LabelBlocked}

// StateLabel is the label of the line's state name, as shipped; the name
// itself for a state it does not know.
func StateLabel(name string) string {
	if l, ok := stateLabels[name]; ok {
		return l
	}
	return name
}

// UnblockTo is where an unblock may send an issue blocked from state
// from ("" unknown): refine, and the state it came from — Ready for one
// a developer had claimed, the claim gone with the block.
func UnblockTo(from string) []string {
	out := []string{"refine"}
	switch from {
	case "", "refine", "blocked":
	case "doing":
		out = append(out, "ready")
	default:
		out = append(out, from)
	}
	return out
}

// checkBlocked says why an act on a blocked issue, or an unblock or a
// blocked on one that is not, cannot be done; nothing when it can.
func (p *Plan) checkBlocked(c *Proposal) (rule, why string) {
	is, ok := p.issues[c.Issue]
	blocked := ok && slices.Contains(is.Labels, LabelBlocked)
	switch {
	case (c.Do == "unblock" || c.Do == "blocked") && !ok:
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	case (c.Do == "unblock" || c.Do == "blocked") && !blocked:
		return "not-blocked", fmt.Sprintf("#%d is not %s: nothing to unblock nor keep blocked", c.Issue, LabelBlocked)
	case c.Do == "unblock" && !slices.Contains(UnblockTo(p.config.RevisitFrom), c.To):
		return "unblock-to", fmt.Sprintf("#%d goes back to %s only, not %q", c.Issue, orList(UnblockTo(p.config.RevisitFrom)), c.To)
	case (c.Do == "unblock" || c.Do == "blocked") && p.revisited[c.Issue] != "":
		return "once-a-run", fmt.Sprintf("#%d: one decision on a blocked issue a run; the %s before it is kept", c.Issue, p.revisited[c.Issue])
	case c.Do == "ready" && blocked:
		return "blocked-first", fmt.Sprintf("#%d is %s: `unblock` it, to the state it came from or to refine, rather than ready", c.Issue, LabelBlocked)
	}
	if c.Do == "unblock" || c.Do == "blocked" {
		if p.revisited == nil {
			p.revisited = map[int]string{}
		}
		p.revisited[c.Issue] = c.Do
	}
	return "", ""
}

// orList says names as choices: refine or tests.
func orList(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += " or "
		}
		out += n
	}
	return out
}
