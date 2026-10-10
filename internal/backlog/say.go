package backlog

import (
	"fmt"
	"html"
	"slices"
	"strings"
)

// What the role says on an issue, in its one comment there (ADR-0038):
// each act it proposes in plain words, a long part folded.

// offer says, in plain words, what accepting a proposal does to the issue
// it sits on, and why; describe's line where there is nothing plainer.
func offer(c Proposal) string {
	why := strings.TrimSpace(c.Why)
	say := func(what string) string {
		if why == "" {
			return what
		}
		return what + " Why: " + why
	}
	switch c.Do {
	case "milestone":
		if c.Milestone != "" {
			return say(fmt.Sprintf("Put it in the milestone %q (now: %s).", c.Milestone, c.Before))
		}
	case "order":
		return say(fmt.Sprintf("Set its priority to %d (now: %s).", c.Priority, c.Before))
	case "rename":
		return say(fmt.Sprintf("Rename it to %q.", strings.TrimSpace(c.Title)))
	case "split":
		var titles []string
		for _, ch := range c.Into {
			titles = append(titles, fmt.Sprintf("%q", strings.TrimSpace(ch.Title)))
		}
		return say(fmt.Sprintf("Split it into %d issues, each linked to it: %s.", len(c.Into), strings.Join(titles, ", ")))
	case "refine":
		if c.ToReporter {
			return say(fmt.Sprintf("Propose to its reporter (an outsider), in a comment, the sections %s: nothing is written in the issue until they or a maintainer agree.", and(c.Added)))
		}
		return say(addSections(c.Added))
	case "ready":
		return say(fmt.Sprintf("Label it `%s`: offered to whoever builds next.", LabelReady))
	case "unready":
		return say(fmt.Sprintf("Move it back to refine: `%s` off, `%s` on.", LabelReady, LabelToRefine))
	case "depend":
		return say(fmt.Sprintf("Mark it as waiting on %s: ordered after, never offered first.", issueList(c.BlockedBy)))
	case "undepend":
		return say(fmt.Sprintf("Take off the link the role set to %s.", issueList(c.BlockedBy)))
	case "ask":
		again := ""
		if c.Round > 1 {
			again = fmt.Sprintf(" again, after their answer (round %d)", c.Round)
		}
		return fmt.Sprintf("Ask its reporter%s, in a comment: %s", again, strings.TrimSpace(c.Questions))
	case "sources":
		if c.Quote == nil {
			return fmt.Sprintf("Name the code it is about: %s. %s", strings.Join(c.Sources, ", "), why)
		}
		return fmt.Sprintf("Name the code it is about: %s. %s %s", strings.Join(c.Sources, ", "), cite(*c.Quote), why)
	case "close":
		if c.Quote == nil {
			break
		}
		if c.Advice {
			what := "what it asks looks done already"
			if c.Reason == "duplicate" {
				what = fmt.Sprintf("it looks like a duplicate of #%d", c.DuplicateOf)
			}
			return fmt.Sprintf("Close it yourself if you agree: %s. %s %s The role does not close it (`close-%s` is off).", what, cite(*c.Quote), why, c.Reason)
		}
		if c.Announce {
			return fmt.Sprintf("Announce it obsolete, on the issue: closed at a later run if nobody writes on it and a second judge agrees. %s %s", cite(*c.Quote), why)
		}
		return strings.Replace(describe(c, "Close"), fmt.Sprintf("Close #%d", c.Issue), "Close it", 1)
	}
	return describe(c, "Close")
}

// addSections says what a refine adds: the sections, the drafts among
// them.
func addSections(added []string) string {
	var drafts, sections []string
	rewrite := false
	for _, n := range added {
		switch {
		case n == DescriptionName:
			rewrite = true
			continue
		case slices.Contains(drafted, n):
			drafts = append(drafts, n)
		}
		sections = append(sections, n)
	}
	var what []string
	if rewrite {
		what = append(what, "Rewrite its description in plain words")
	}
	if len(sections) > 0 || !rewrite {
		add := fmt.Sprintf("add %s to it", and(sections))
		if len(drafts) > 0 {
			add += fmt.Sprintf(" (%s as drafts for you to correct)", and(drafts))
		}
		what = append(what, add)
	}
	s := strings.Join(what, ", and ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

// and joins names as a sentence says them: "Need", "Need and Scope",
// "Need, Scope and Validation".
func and(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// fold is a long part folded: its summary in one line, its body shown on
// a click.
func fold(summary, body string) string {
	return "<details><summary>" + html.EscapeString(summary) + "</summary>\n\n" + body + "\n\n</details>"
}
