package developer

// A round that cannot move alone: a red check its round did not fix
// (#938), a fix held off a green head, an objection that waits for the
// tester (#987). Its issue goes Blocked for a person, its reason on it,
// or waits in Doing for the tester; sent back, the next round resumes.
// A round asked the product owner for files outside the Scope before
// ADR-0065's change 2 (#921): the Scope is a hint now, nothing is asked.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/work"
)

// scopeName is the Scope's heading, as the table names it.
var scopeName = work.Name(work.Scope)

// Widen is a round's wait, for a person or the tester, as the record
// keeps it.
type Widen struct {
	At    string `json:"at"`    // the branch's head it waited at
	Scope string `json:"scope"` // the bounds' Scope then, hashed
	// Held: no file asked, a project check red its round did not fix
	// (#938): blocked for a person, who sends it back to resume.
	Held bool `json:"held,omitempty"`
	// Back: a round ran after the issue was sent back, by a person.
	Back bool `json:"back,omitempty"`
	// Objected: the round objected to a frozen test at At, and its
	// issue waits for the tester's answer (#987): a rewrite is taken by
	// the next round, a test kept asks the round again. Kept: the
	// tester's answers that kept a test on its Draft then; one more is
	// its answer.
	Objected bool `json:"objected,omitempty"`
	Draft    int  `json:"draft,omitempty"` // the tester's Draft it objected on
	Kept     int  `json:"kept,omitempty"`
	// Review: the round held a fix of the review's findings off a green
	// head (#987): they are asked again when the round resumes. Sent
	// back, the findings its last round asked are asked again in any
	// case, from the record (#1022).
	Review bool `json:"review,omitempty"`
}

// scopeKey is the bounds' Scope, hashed: the product owner widened it
// when it changed.
func scopeKey(scope []string) string {
	s := slices.Clone(scope)
	slices.Sort(s)
	sum := sha256.Sum256([]byte(strings.Join(s, "\n")))
	return hex.EncodeToString(sum[:6])
}

// backFromAsk says whether a person sent the issue back since the
// record's wait: no more blocked, ready or taken again. One call, only
// while an ask waits.
func backFromAsk(f forge.Forge, a Act) (bool, error) {
	if a.Widen == nil || a.Widen.Back || a.Widen.Objected {
		return false, nil // an objection waits in Doing: nothing sent back
	}
	is, err := f.Issue(a.Issue)
	if err != nil {
		return false, err
	}
	if slices.Contains(is.Labels, backlog.LabelBlocked) {
		return false, nil
	}
	return slices.Contains(is.Labels, backlog.LabelReady) || slices.Contains(is.Labels, backlog.StateLabel("doing")), nil
}

// legacyNoCode says whether the record's last round wrote no fix for a
// red check before a round could ask the product owner for its Scope
// (#921), or block its issue when it asked none (#938): due once more at
// the same head, to ask or block now. Since, such a round always leaves
// an ask in the record.
func legacyNoCode(a Act) bool {
	if len(a.Rounds) == 0 || a.Widen != nil {
		return false
	}
	last := a.Rounds[len(a.Rounds)-1]
	return last.Result == "no-code" && (last.Bounds == "" || last.Bounds == "held") &&
		slices.ContainsFunc(last.Asked, func(s string) bool { return strings.HasPrefix(s, checkHolds+" ") })
}

// redChecks are the names of the project checks a round asked, red.
func redChecks(asked []Asked) []string {
	var out []string
	for _, x := range asked {
		if x.Holds == checkHolds && !slices.Contains(out, x.Where) {
			out = append(out, x.Where)
		}
	}
	return out
}

// holdCheck is what a round writes that fixed no red check (#938): the
// issue blocked for a person, the checks, its agent's notes, its
// objections' place and the checks' log on it.
func holdCheck(repo string, a Act, r round, red, notes []string, objections []intent.Intention) []intent.Intention {
	why := fmt.Sprintf("pull request %d is still red on `%s` at this head, and its round wrote no fix: the check is not asked again at this head",
		r.MR, strings.Join(red, "`, `"))
	next := "a person on the loop reads the developer's note below, then fixes the check or the test, or re-runs a flaky check; " +
		"and moves the issue back to ready: the next round runs at the same head"
	return holdFor(repo, a, r, why, next, notes, objections)
}

// holdGreen is what a round writes whose fix of the review turned
// frozen tests red on a green head (#987): not pushed, and it objected
// to no test. Nothing else
// would move: blocked for a person, the tests and the findings on it.
func holdGreen(repo string, a Act, r round, red, notes []string, objections []intent.Intention) []intent.Intention {
	why := fmt.Sprintf("its fix of the review turns the frozen tests red on pull request %d's green head (`%s`): not pushed, the head kept; it objected to no test, or objected already at this head",
		r.MR, strings.Join(red, "`, `"))
	next := "a person on the loop reads the findings and the developer's note below, then answers the finding or fixes the test; " +
		"and moves the issue back to ready: the next round asks the findings again at the same head"
	return holdFor(repo, a, r, why, next, notes, objections)
}

// holdFor blocks the issue for a person, why and next said, then what
// the round was asked, its agent's notes and where its objections went.
func holdFor(repo string, a Act, r round, why, next string, notes []string, objections []intent.Intention) []intent.Intention {
	var b strings.Builder
	b.WriteString(backlog.Said(who, "blocks this issue", why, next))
	b.WriteString("\nWhat it was asked:\n\n" + askedList(r.Asked))
	if len(notes) > 0 {
		b.WriteString("\nIts note:\n\n")
		for _, n := range notes {
			fmt.Fprintf(&b, "- %s\n", inertText(strings.ReplaceAll(strings.TrimSpace(n), "\n", " ")))
		}
	}
	for _, o := range objections {
		if m, ok := o.Value.(map[string]any); ok && m["sticky"] == "objection" {
			fmt.Fprintf(&b, "\nIts objections to the frozen tests are on the tester's Draft !%v, on their lines: the tester keeps or rewrites each.\n", m["merge-request"])
		}
	}
	b.WriteString(checkLogsSaid(r.Asked)) // a log's end is bounded already (logEnd)
	return []intent.Intention{
		moveTo(repo, "blocked", a.Issue, []string{backlog.LabelBlocked}, []string{backlog.LabelReady, backlog.StateLabel("doing")}),
		sayOn(a.Issue, b.String()),
	}
}

// objectedOn says whether a round's comments post an objection on the
// tester's Draft: the bounds kept it, its line known.
func objectedOn(comments []intent.Intention) bool {
	return slices.ContainsFunc(comments, func(o intent.Intention) bool {
		m, ok := o.Value.(map[string]any)
		return ok && m["sticky"] == "objection"
	})
}

// checkLogsSaid is the end of each red check's failed steps' log, as
// an issue's comment quotes it: its marks broken (#926).
func checkLogsSaid(asked []Asked) string {
	var b strings.Builder
	for _, x := range asked {
		if x.Holds == checkHolds && x.Detail != "" {
			fmt.Fprintf(&b, "\nThe end of `%s`'s failed steps' log:\n\n~~~~\n%s\n~~~~\n", x.Where, forge.Unmarked(strings.ReplaceAll(x.Detail, "~~~~", "~~~ ~")))
		}
	}
	return b.String()
}

// moveTo is the label that moves the issue to the line's state named,
// as the project's routing says it; fallback when it cannot be read.
func moveTo(repo, state string, issue int, add, remove []string) intent.Intention {
	if line, err := routing.Load(repo); err == nil {
		if ad, rm, err := line.Move(state); err == nil {
			add, remove = ad, rm
		}
	}
	return intent.Intention{Kind: "label", Value: map[string]any{"issue": issue, "add": add, "remove": remove, "card": state}}
}

// besideSaid is the issue's comment listing the new test files the
// developer's rounds kept beside its Scope, with the mark the product
// owner reads to fold them in (ADR-0063, §9): one comment, edited.
func besideSaid(a Act) intent.Intention {
	body := fmt.Sprintf("**The developer** wrote new test files beside this issue's %s: `%s`. The product owner adds them to the %[1]s at its next run.\n\n%[3]s\n",
		scopeName, strings.Join(a.Beside, "`, `"), backlog.ScopeKeptMark(a.Beside))
	return intent.Intention{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "beside", "body": body}}
}
