package developer

// A gate red on the developer's pull request is a round, like the
// reviewer's verdict: the gates' summary comment names what the change
// adds to their tools, at the head they ran on (ADR-0049, step 3); the
// inspector's comment says why each matters and how to fix it. A round
// asks them once a report: its record keeps the head the gates were
// read at.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/builtin/inspector"
	"github.com/JN0V/workline/internal/forge"
)

// gateHolds is how a round says a gate's finding: neither the reviewer's
// nor a person's.
const gateHolds = "a gate"

var (
	gatesHead  = regexp.MustCompile("(?m)^On `([0-9a-f]{7,40})`, only what this change adds")
	gateName   = regexp.MustCompile("^\\*\\*`?([^`*]+)`?\\*\\*")
	gatePlaces = regexp.MustCompile(", \\d+ new: `([^`]+)`")
)

// gateRows reads the gates' summary comment: the head they ran on, and
// each rule's new findings, at the first place it names; a check that
// did not pass, or whose findings a command fixes, too. A check that did
// not run asks nothing: no code of the change fixes a tool not run.
func gateRows(comment string) (head string, asked []Asked) {
	m := gatesHead.FindStringSubmatch(comment)
	if m == nil {
		return "", nil
	}
	for _, l := range strings.Split(comment, "\n") {
		text, ok := strings.CutPrefix(strings.TrimSpace(l), "- ")
		if !ok {
			continue
		}
		n := gateName.FindStringSubmatch(text)
		if n == nil || strings.Contains(text, "** did not run") {
			continue // a line counting the rules not listed, or a tool not run
		}
		where, file := "", ""
		if p := gatePlaces.FindStringSubmatch(text); p != nil {
			where = p[1]
			file, _, _ = strings.Cut(where, ":")
		}
		asked = append(asked, Asked{Key: "gates|" + file + "|" + n[1], Where: where, Text: text, Holds: gateHolds})
	}
	return m[1], asked
}

// triageDetails gives each gate's finding what the inspector said of it,
// triaged at the gates' head: why it matters, its fix, or why it is
// likely a false positive. An inspector's comment that does not read
// adds nothing: the gates' own words stand.
func triageDetails(asked []Asked, comment, head string) {
	rec, err := inspector.ReadTriage(comment)
	if err != nil {
		return
	}
	for i, a := range asked {
		if a.Holds != gateHolds || a.Where == "" {
			continue
		}
		_, rule, _ := strings.Cut(strings.TrimPrefix(a.Key, "gates|"), "|")
		for _, t := range rec.Triaged {
			if t.Where != a.Where || t.Rule != rule && t.Check != rule || !sameCommit(t.At, head) {
				continue
			}
			d := t.Matters
			switch {
			case t.FalsePositive:
				d += " The inspector holds it likely a false positive: " + t.Why
			case t.Fix != "":
				d += " Fix: " + t.Fix
			}
			asked[i].Detail = strings.TrimSpace(d)
		}
	}
}

// sameCommit says two ids name one commit, one maybe abbreviated.
func sameCommit(a, b string) bool {
	if len(a) < 7 || len(b) < 7 {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

// gatesAsked reads the gates' new findings a round asks: none when the
// gates reported on another head than its branch's (head; "" when not
// known: the round reads it), or on the one its last round read
// (a.GatesAt). It returns the head they were read at.
func gatesAsked(kept forge.Kept, a Act, head string) (string, []Asked) {
	c, ok := kept.Last(forge.GatesMarker)
	if !ok {
		return "", nil
	}
	at, asked := gateRows(c)
	if at == "" || len(asked) == 0 || sameCommit(at, a.GatesAt) || head != "" && !sameCommit(at, head) {
		return "", nil
	}
	if t, ok := kept.Last(inspector.Marker); ok {
		triageDetails(asked, t, at)
	}
	return at, asked
}

// gateLine is a gate's finding as a round's question and comment say it.
func gateLine(a Asked) string {
	if a.Where == "" {
		return fmt.Sprintf("%s's new finding: %s", gateHolds, a.Text)
	}
	return fmt.Sprintf("%s's new finding `%s`: %s", gateHolds, a.Where, a.Text)
}
