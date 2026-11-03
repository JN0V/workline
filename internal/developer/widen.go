package developer

// A fix outside the Scope (#921): a round whose fix needs files the
// issue's Scope does not hold writes none, and asks the product owner,
// whose Scope it is (ADR-0060), to widen it. Its issue goes Blocked, the
// files and why on it; the line wakes the product owner at once. It
// widens the Scope and sends the issue back to ready, and the next round
// resumes on the same pull request with the new bounds; or it keeps the
// issue blocked, its reason written. One ask a head and a Scope: asked
// again on the same, the issue is blocked for a person.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
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

// Widen is a round's ask to the product owner for files outside the
// issue's Scope, as the record keeps it.
type Widen struct {
	At    string   `json:"at"`    // the branch's head it was asked at
	Scope string   `json:"scope"` // the bounds' Scope then, hashed
	Files []string `json:"files"`
	// Back: a round ran after the issue was sent back, by the product
	// owner or a person.
	Back bool `json:"back,omitempty"`
	// Again: asked again on the same head and Scope, blocked for a
	// person; one who sends it back resumes the round too.
	Again bool `json:"again,omitempty"`
}

// outsideScope are the repository's files the round's patches reached
// outside the issue's Scope: what the product owner may widen it with.
// A gate's, a steering file, a frozen test are never its to give.
func outsideScope(refused []Refusal) []string {
	var out []string
	for _, r := range refused {
		f := r.File
		if r.Rule != "outside-scope" || f == ".." || strings.HasPrefix(f, "../") || path.IsAbs(f) || slices.Contains(out, f) {
			continue
		}
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}

// noteNeeds are the files a note's `needs` names: those its fix needs
// outside the bounds, given as a list or one path.
func noteNeeds(v any) []string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	switch n := m["needs"].(type) {
	case string:
		return []string{strings.TrimSpace(n)}
	case []any:
		var out []string
		for _, x := range n {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	}
	return nil
}

// needed holds the files the agent's notes say its fix needs to the
// bounds, as a patch's: each the bounds would refuse, its rule said; one
// within them asks nothing.
func needed(repo string, b Bounds, files []string) []Refusal {
	c := newChecker(repo, b)
	var out []Refusal
	for _, f := range files {
		f = path.Clean(filepath.ToSlash(strings.Trim(f, "`")))
		if r := c.rule(f); r != "" {
			out = append(out, Refusal{File: f, Rule: r})
		}
	}
	return out
}

// scopeKey is the bounds' Scope, hashed: the product owner widened it
// when it changed.
func scopeKey(scope []string) string {
	s := slices.Clone(scope)
	slices.Sort(s)
	sum := sha256.Sum256([]byte(strings.Join(s, "\n")))
	return hex.EncodeToString(sum[:6])
}

// backFromAsk says whether the product owner sent the issue back since
// the record's ask: no more blocked, ready or taken again. One call, only
// while an ask waits.
func backFromAsk(f forge.Forge, a Act) (bool, error) {
	if a.Widen == nil || a.Widen.Back {
		return false, nil
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
// (#921): due once more at the same head, to ask now.
func legacyNoCode(a Act) bool {
	if len(a.Rounds) == 0 || a.Widen != nil {
		return false
	}
	last := a.Rounds[len(a.Rounds)-1]
	return last.Result == "no-code" && last.Bounds == "" && slices.ContainsFunc(last.Asked, func(s string) bool { return strings.HasPrefix(s, checkHolds+" ") })
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

// askScope is what a round that needs files outside the Scope writes: the
// issue blocked, the ask on it with what the round saw, for the product
// owner; asked already on this head and Scope, blocked for a person.
func askScope(repo string, a Act, r round, files []string, again bool, notes []string) []intent.Intention {
	list := "`" + strings.Join(files, "`, `") + "`"
	var why, next string
	if again {
		why = fmt.Sprintf("the fix of pull request %d still needs %s, outside its %s, asked already at this head: the %[3]s did not change", r.MR, list, scopeName)
		next = "a person on the loop widens the " + scopeName + ", or fixes or closes the pull request, and moves the issue back by its label"
	} else {
		why = fmt.Sprintf("the fix of pull request %d needs %s, outside its %s: it asks the product owner to widen it", r.MR, list, scopeName)
		next = fmt.Sprintf("the product owner widens the %s and moves the issue back to ready, the developer resumes on pull request %d with the new bounds; or keeps it blocked, why not said", scopeName, r.MR)
	}
	var b strings.Builder
	b.WriteString(backlog.Said(who, "blocks this issue", why, next))
	b.WriteString("\nWhat it was asked:\n\n" + askedList(r.Asked))
	if len(notes) > 0 {
		b.WriteString("\nWhy it needs them:\n\n")
		for _, n := range notes {
			fmt.Fprintf(&b, "- %s\n", inertText(strings.ReplaceAll(strings.TrimSpace(n), "\n", " ")))
		}
	}
	for _, x := range r.Asked {
		if x.Holds == checkHolds && x.Detail != "" {
			fmt.Fprintf(&b, "\nThe end of `%s`'s failed steps' log:\n\n~~~~\n%s\n~~~~\n", x.Where, forge.Unmarked(strings.ReplaceAll(x.Detail, "~~~~", "~~~ ~")))
		}
	}
	body := b.String() // a log's end is bounded already (logEnd)
	if !again {
		body += "\n" + backlog.ScopeAskMark(r.At, files) + "\n"
	}
	return []intent.Intention{
		moveTo(repo, "blocked", a.Issue, []string{backlog.LabelBlocked}, []string{backlog.LabelReady, backlog.StateLabel("doing")}),
		sayOn(a.Issue, body),
	}
}
