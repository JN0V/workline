package backlog

// A blocked issue read again (ADR-0059, amended): the line wakes the role
// on it when there is news, and the role sends it back — to the state it
// came from, or to refine — or keeps it blocked, its reason said afresh.
// Both done alone: a label put back undoes the first, the second only
// says.

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
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

// scopeAsk is the developer's mark on the comment that blocks its issue
// to ask the product owner for files outside its Scope (#921): the head
// of its pull request it asked at, and the files.
var scopeAsk = regexp.MustCompile(`<!-- workline:scope-ask at=(\S+) files=(\S+) -->`)

// ScopeAskMark is the mark of a developer's ask for files outside its
// Scope, its pull request's head at, on the comment that blocks the issue.
func ScopeAskMark(at string, files []string) string {
	return fmt.Sprintf("<!-- workline:scope-ask at=%s files=%s -->", at, strings.Join(files, ","))
}

// ScopeAskRefine is what the developer's first step says when it sends
// its issue back to refine to ask the product owner for files outside
// its Scope (ADR-0063, §9): the product owner's state, not Blocked.
const ScopeAskRefine = "sends this issue back to refine"

// ScopeAsked reads the files a developer asks to add to the issue's
// Scope from the comment blocking it, or sending it back to refine, when
// that comment is its ask; none otherwise. Only the last such comment is
// read, the line's pointing to it aside: a block after the ask is
// another.
func ScopeAsked(comments []string) []string {
	for _, c := range slices.Backward(comments) {
		if !strings.Contains(c, "blocks this issue") && !strings.Contains(c, ScopeAskRefine) || strings.Contains(c, BlockedElsewhere) {
			continue
		}
		if m := scopeAsk.FindStringSubmatch(c); m != nil {
			return strings.Split(m[2], ",")
		}
		return nil
	}
	return nil
}

// scopeKept is the developer's mark listing the new test files its code
// wrote beside the issue's Scope, kept by the write fence (ADR-0063, §9).
var scopeKept = regexp.MustCompile(`<!-- workline:scope-kept files=(\S+) -->`)

// ScopeKeptMark is the mark of the new test files the developer kept
// beside the issue's Scope.
func ScopeKeptMark(files []string) string {
	return fmt.Sprintf("<!-- workline:scope-kept files=%s -->", strings.Join(files, ","))
}

// ScopeKept are the new test files the developer kept beside the issue's
// Scope, every mark read: the product owner adds them to the Scope.
func ScopeKept(comments []string) []string {
	var out []string
	for _, c := range comments {
		for _, m := range scopeKept.FindAllStringSubmatch(c, -1) {
			for _, f := range strings.Split(m[1], ",") {
				if f != "" && !slices.Contains(out, f) {
					out = append(out, f)
				}
			}
		}
	}
	return out
}

// ScopeAsks are every file a developer asked to add to the issue's
// Scope, in any of its asks, whatever the head (ADR-0063, §9): the same
// set asked again waits for a person.
func ScopeAsks(comments []string) []string {
	var out []string
	for _, c := range comments {
		for _, m := range scopeAsk.FindAllStringSubmatch(c, -1) {
			for _, f := range strings.Split(m[2], ",") {
				if f != "" && !slices.Contains(out, f) {
					out = append(out, f)
				}
			}
		}
	}
	return out
}

// scopeKeptOf folds into a refine's Scope the new test files the
// developer kept beside it (ScopeKept, ADR-0063 §9): each it does not
// name is added to it, and its why says so. All the files kept, for the
// check that a Scope's files are in the repository: they are on the pull
// request's branch only, until it merges.
func (p *Plan) scopeKeptOf(f forge.Backlog, is forge.Issue, c *Proposal) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	comments, err := f.Comments(forge.Target{Kind: "issue", ID: is.ID})
	if err != nil {
		return nil, err
	}
	kept := ScopeKept(comments)
	var added []string
	for _, k := range kept {
		if !strings.Contains(c.Scope, "`"+k+"`") {
			c.Scope = strings.TrimSpace(c.Scope + "\n- `" + k + "`: a new test file the developer kept beside the " + work.Name(work.Scope) + ".")
			added = append(added, k)
		}
	}
	if len(added) > 0 {
		said := fmt.Sprintf("its %s names `%s` now: new test files the developer kept beside it", work.Name(work.Scope), strings.Join(added, "`, `"))
		c.Why = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(c.Why), ".") + "; " + said + ".")
		p.Findings = append(p.Findings, verdict.Finding{Rule: "scope-kept", Level: "info", Where: c.where(), Message: said})
	}
	return kept, nil
}

// scopeAskOf reads, on a blocked issue or one in refine, the files the
// developer asks its Scope widened with (ScopeAsked); none for another,
// or with no forge.
func (p *Plan) scopeAskOf(f forge.Backlog, is forge.Issue) ([]string, error) {
	if f == nil || !slices.Contains(is.Labels, LabelBlocked) && !slices.Contains(is.Labels, LabelToRefine) {
		return nil, nil
	}
	comments, err := f.Comments(forge.Target{Kind: "issue", ID: is.ID})
	if err != nil {
		return nil, err
	}
	return ScopeAsked(comments), nil
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
	case c.Do == "unblock" && len(p.widenDropped[c.Issue]) > 0:
		return "scope-not-widened", fmt.Sprintf("not moved out of Blocked: the developer asked to widen its %s with %s, and its %[1]s did not change", work.Name(work.Scope), strings.Join(p.widenDropped[c.Issue], ", "))
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
