package backlog

import (
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"
)

// The report issue is a page a person reads (ADR-0031): what they have to
// do first, then the boxes to tick, then what the role did and how it
// stands — the long parts folded, never left out.

// reportIntro opens the report.
const reportIntro = "The product owner keeps this backlog. This page says first what it needs from you, then what it did; each run rewrites it.\n"

// howItWorks is the report's fold on the boxes and the undoing.
const howItWorks = "- A box ticked by a person of the project is done at the next run, as written; a box ticked by anyone else, or by a bot, is said and not done.\n" +
	"- An act undone — a closing reopened, a title, a priority or a milestone put back, `" + LabelReady + "` taken off, a split's part closed as not planned, a link it set between issues taken off — puts that kind of act back to a person.\n" +
	"- When nobody answers several runs in a row, the role pauses: it asks no agent until a person ticks a box, writes here or undoes an act."

// ReportBody is the report issue's body.
func (p *Plan) ReportBody() string {
	decide, boxes := p.toDecide()
	check, checks := p.toCheck()
	did, before := p.didLines()
	var b strings.Builder
	b.WriteString(reportIntro)
	b.WriteString(p.whatToDo(boxes, checks, len(did)))
	b.WriteString(decide)
	b.WriteString(check)
	b.WriteString(p.toAccept())
	if len(p.said) > 0 {
		b.WriteString("\n## Boxes ticked\n\nWhat the role read of the boxes ticked since its last run.\n\n" + strings.Join(p.said, "\n") + "\n")
	}
	b.WriteString(p.Opening)
	b.WriteString(p.waiting())
	if len(did) > 0 {
		body := strings.Join(did, "\n")
		if len(before) > 0 {
			body += "\n\n**Before this run** — the issues it moved, as they were: to put the order back, set their priority label and milestone to these.\n\n" + strings.Join(before, "\n")
		}
		b.WriteString(fold(fmt.Sprintf("What the role did alone this run: %s, and how to undo each", plural(len(did), "act")), body))
	}
	b.WriteString(p.rechecked())
	b.WriteString(p.autonomy())
	b.WriteString(fold("How this page works", howItWorks))
	return b.String()
}

// whatToDo is the report's top: what waits on a person, how long it
// takes, and whether the role is about to pause or paused.
func (p *Plan) whatToDo(decide, checks, did int) string {
	var items []string
	r, max := p.Record, p.config.IgnoredMax
	paused := max > 0 && r.Ignored >= max
	if paused {
		items = append(items, fmt.Sprintf("**Paused**: %d runs in a row proposed something and nobody ticked a box, wrote here or undid an act. No agent is asked until a person does; tick this to resume:\n\n  - [ ] Resume %s", r.Ignored, TickMarker(KeyResume)))
	}
	accept := len(p.Record.ToAccept)
	if decide > 0 {
		items = append(items, fmt.Sprintf("**%s to decide** (*To decide*): tick a box to agree, and the role does it at its next run; leave it unticked to say no.", plural(decide, "proposal")))
	}
	if checks > 0 {
		items = append(items, fmt.Sprintf("**%s to check** (*To check*): what an issue was built on changed; read it, then tick its box.", plural(checks, "change")))
	}
	if decide+checks+accept == 0 && !paused {
		items = append(items, "**Nothing waits on you.** Below: what is next, and what the role did.")
	}
	if accept > 0 {
		items = append(items, fmt.Sprintf("**%s to accept** (*To accept*): close each once you checked what its parts delivered.", plural(accept, "need")))
	}
	switch {
	case max == 0:
		items = append(items, fmt.Sprintf("**Never paused** (ignored-runs-max: 0): the agent is asked on every run, though nobody answered the last %d.", r.Ignored))
	case !paused && r.Ignored > 0 && len(r.Proposed) > 0:
		when := "before the next run"
		if left := max - r.Ignored; left > 1 {
			when = fmt.Sprintf("within %d runs", left)
		}
		items = append(items, fmt.Sprintf("**Answer %s, or the role pauses**: nobody answered its last %s (it pauses at %d). A tick, a comment here or an act undone is an answer.", when, plural(r.Ignored, "run"), max))
	}
	if did > 0 {
		them := "them"
		if did == 1 {
			them = "it"
		}
		items = append(items, fmt.Sprintf("**%s done alone this run**: check %s under *What the role did*, and undo what you disagree with.", plural(did, "act"), them))
	}
	if level, why := p.Record.Measure.Suggest(); level != "" {
		items = append(items, fmt.Sprintf("**Suggested**: `autonomy: %s` — %s. Set it in the project's settings if you agree; the role never changes it.", level, why))
	}
	if level, why := SuggestFromRecord(p.Record); level != "" {
		items = append(items, fmt.Sprintf("**Suggested** from the acts done alone: `autonomy: %s` — %s. Set it in the project's settings if you agree; the role never changes it.", level, why))
	}
	return "\n## What to do\n\n- " + strings.Join(items, "\n- ") + "\n"
}

// toDecide is the report's boxes: each proposal, under the issue it is
// on, saying what a tick does; the kinds a person can set back to act. It
// says how many boxes it holds.
func (p *Plan) toDecide() (string, int) {
	var order []int
	byIssue := map[int][]string{}
	var opens []string
	n := 0
	for _, q := range p.Record.Proposed {
		line := q.Line
		switch {
		case q.Proposal != nil && q.Doable():
			line = offer(*q.Proposal)
		case q.Proposal != nil && q.Proposal.Do == "open", q.Key != "":
			line = strings.TrimSuffix(line, ".") + ". Opened when the import runs again, or open it yourself."
		case q.Undrafted():
			line = plainLine(line) + " Drafted when the agent next reads it."
		}
		box := "- [ ] " + line + " " + TickMarker(q.TickKey())
		if q.Agreed != "" {
			// Ticked already: nothing left to decide, only to say.
			box = fmt.Sprintf("- Ticked by %s: %s It is drafted, then done, at the next run that reads it with an agent.", q.Agreed, plainLine(q.Line))
		} else {
			n++
		}
		if q.Issue == 0 {
			opens = append(opens, box)
			continue
		}
		if _, ok := byIssue[q.Issue]; !ok {
			order = append(order, q.Issue)
		}
		byIssue[q.Issue] = append(byIssue[q.Issue], box)
	}
	var b strings.Builder
	for _, id := range order {
		head := fmt.Sprintf("**#%d %s**", id, p.title(id))
		if why := p.readFor(id); why != "" {
			head += " — " + why
		}
		b.WriteString("\n" + head + "\n\n" + strings.Join(byIssue[id], "\n") + "\n")
	}
	if len(opens) > 0 {
		b.WriteString("\n**Issues to open**\n\n" + strings.Join(opens, "\n") + "\n")
	}
	if back := p.backToAct(); back != "" {
		n += len(p.Record.Propose)
		b.WriteString(back)
	}
	if b.Len() == 0 {
		return "", 0
	}
	return "\n## To decide\n\nTick a box to agree: the engine does it at its next run, as written. Leave it unticked to say no.\n" + b.String(), n
}

// backToAct is the boxes that set a kind of act back to act, after a
// wrong closing or an act undone put it back to a person.
func (p *Plan) backToAct() string {
	if len(p.Record.Propose) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n**Let the role act alone again**\n\nBack to propose after a wrong closing or an act undone: %s.\n\n", strings.Join(p.Record.Propose, ", "))
	for _, w := range p.Record.Wrong {
		if slices.Contains(p.Record.Propose, w.Act) {
			fmt.Fprintf(&b, "- #%d, closed as %s, was reopened.\n", w.Issue, w.Act)
		}
	}
	for _, u := range p.Record.Undone {
		if slices.Contains(p.Record.Propose, u.Act) {
			fmt.Fprintf(&b, "- %s.\n", u.Evidence)
		}
	}
	for _, kind := range p.Record.Propose {
		if !strings.HasPrefix(kind, "close-") {
			standing, undone := 0, 0
			for _, d := range p.Record.Done {
				if d.Act == kind {
					standing++
				}
			}
			for _, u := range p.Record.Undone {
				if u.Act == kind {
					undone++
				}
			}
			fmt.Fprintf(&b, "- [ ] Set %s back to act: %d of its acts still standing, %d undone. %s\n", kind, standing, undone, TickMarker(KeyAct+kind))
			continue
		}
		kept, wrong := 0, 0
		for _, c := range p.Record.Closed {
			if c.Act == kind {
				kept++
			}
		}
		for _, c := range p.Record.Wrong {
			if c.Act == kind {
				wrong++
			}
		}
		fmt.Fprintf(&b, "- [ ] Set %s back to act: %d of its closings still closed, %d reopened. %s\n", kind, kept, wrong, TickMarker(KeyAct+kind))
	}
	return b.String()
}

// didLines are the acts this run did alone, each with how to undo it, and
// the issues it moved as they were.
func (p *Plan) didLines() (did, before []string) {
	listed := map[int]bool{}
	for _, d := range p.Decisions {
		if d.Mode != Act {
			continue
		}
		undo := " Reopen it to undo."
		switch d.Act.Do {
		case "sources", "ask", "undepend", "keep":
			undo = ""
		case "milestone", "order":
			undo = " Put it back as it was before this run (below) to undo."
		case "refine":
			undo = " Edit its body to undo."
			if d.Act.ToReporter {
				undo = ""
			}
		case "ready":
			undo = " Remove the label workline:ready to undo."
		case "unready":
			undo = " Put the label workline:ready back to undo."
		case "rename":
			undo = " Rename it back to undo."
		case "split":
			undo = fmt.Sprintf(" Close the issues opened from #%d to undo: its own text was left as it was.", d.Act.Issue)
		case "depend":
			undo = " Remove the link, or the line in its body, to undo."
		case "close":
			if d.Act.Announce {
				undo = fmt.Sprintf(" To keep it open, write on it or take the label %s off.", LabelObsolete)
			}
		}
		if d.Act.Ticked != "" {
			undo += " Ticked by " + d.Act.Ticked + "."
		}
		did = append(did, "- "+describe(d.Act, "Closed")+undo)
		if slices.Contains(moves, d.Act.Do) && !listed[d.Act.Issue] {
			listed[d.Act.Issue] = true
			before = append(before, fmt.Sprintf("- #%d: %s", d.Act.Issue, d.Act.Before))
		}
	}
	return did, before
}

// offer says, in plain words, what ticking a proposal does to the issue
// it sits under, and why; describe's line where there is nothing plainer.
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
		return fmt.Sprintf("Name the code it is about: %s. %s %s", strings.Join(c.Sources, ", "), cite(*c.Quote), why)
	case "close":
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
	var drafts []string
	for _, n := range added {
		if slices.Contains(drafted, n) {
			drafts = append(drafts, n)
		}
	}
	what := fmt.Sprintf("Add %s to it", and(added))
	if len(drafts) > 0 {
		what += fmt.Sprintf(" (%s as drafts for you to correct)", and(drafts))
	}
	return what + "."
}

// oldRefine is a refine's line as an earlier engine recorded it, with no
// act to do: "Refine #83: Need (draft), Validation (draft). Why".
var oldRefine = regexp.MustCompile(`^Refine #\d+: ([^.]+)\.\s*(.*)$`)

// plainLine says a proposal recorded with its line alone in the words of
// offer, where its line is an earlier engine's refine.
func plainLine(line string) string {
	m := oldRefine.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	var added []string
	for _, n := range strings.Split(m[1], ", ") {
		added = append(added, strings.TrimSuffix(n, " (draft)"))
	}
	return addSections(added) // its reason, recorded with it, said the same again
}

// and joins names as a sentence says them: "Need", "Need and Scope",
// "Need, Scope and Validation".
func and(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// autonomy is the report's fold on how far the role goes, kind by kind.
func (p *Plan) autonomy() string {
	var lines []string
	for _, m := range p.config.Modes(p.Record.Propose) {
		lines = append(lines, "- "+m.Kind+": "+m.Say())
	}
	return fold(fmt.Sprintf("Autonomy: %s — what the role does alone, and what it proposes", p.config.Level),
		"Each kind of act is done alone up to a number a run (act), proposed to you (propose), or not done (off) — as the level sets it (level), as the project's settings do (setting), or proposed since a person undid one (demoted). Change it in the project's settings.\n\n"+strings.Join(lines, "\n"))
}

// fold is a part of the report folded: its summary in one line, its body
// shown on a click.
func fold(summary, body string) string {
	return "\n<details><summary>" + html.EscapeString(summary) + "</summary>\n\n" + body + "\n\n</details>\n"
}

// title is an issue's title, as the run read it; "" when unknown.
func (p *Plan) title(id int) string {
	return p.issues[id].Title
}

// readFor says why an issue was read again, for its proposals: each change
// it was read with; "" when none.
func (p *Plan) readFor(id int) string {
	var why []string
	for _, c := range p.Record.Changes {
		for _, t := range c.Touch {
			if t.Issue != id || !t.Read {
				continue
			}
			if c.Path != "" {
				why = append(why, fmt.Sprintf("the lines of `%s` it was opened from changed (now: lines %s; found %s)", c.Path, c.Lines, c.Since))
				continue
			}
			how := map[string]string{TouchPart: "which it is a part of", TouchWaits: "which it waits on", TouchSources: "which names the same code"}[t.How]
			why = append(why, fmt.Sprintf("#%d %s, %s, had its %s rewritten by a person (found %s)", c.Issue, p.title(c.Issue), how, strings.Join(c.What, " and "), c.Since))
		}
	}
	if len(why) == 0 {
		return ""
	}
	return "read again because " + strings.Join(why, "; ")
}

// toCheck is the report's part on the changes a person checks — an issue
// they touch the role could not read again — and how many boxes it holds.
// A change whose every issue was read again is said with their proposals,
// or settled with no box (settleChanges).
func (p *Plan) toCheck() (string, int) {
	var lines []string
	n := 0
	for _, c := range p.Record.Changes {
		if !slices.ContainsFunc(c.Touch, func(t Touch) bool { return p.open[t.Issue] && !t.Read }) {
			continue
		}
		n++
		if c.Path != "" {
			lines = append(lines, fmt.Sprintf("- [ ] `%s`, lines %s, which #%d %s was opened from, changed (found %s): check #%d against them, then tick. %s",
				c.Path, c.Lines, c.Issue, p.title(c.Issue), c.Since, c.Issue, TickMarker(c.Key())))
			continue
		}
		lines = append(lines, fmt.Sprintf("- [ ] #%d %s: a person rewrote its %s (found %s). Check the issues built on it, then tick. %s",
			c.Issue, p.title(c.Issue), strings.Join(c.What, " and "), c.Since, TickMarker(c.Key())))
		for _, t := range c.Touch {
			if !p.open[t.Issue] {
				continue
			}
			how := map[string]string{TouchPart: "a part of it", TouchWaits: "waits on it", TouchSources: "names the same code, " + t.Files, TouchImport: "opened from those lines"}[t.How]
			what := "not read by the role: check it yourself"
			if t.Read {
				what = "read again by the role: nothing to change"
				if len(p.proposedOn(t.Issue)) > 0 {
					what = "read again by the role: its proposals are under To decide"
				}
			}
			lines = append(lines, fmt.Sprintf("  - #%d %s — %s; %s.", t.Issue, p.title(t.Issue), how, what))
		}
	}
	if n == 0 {
		return "", 0
	}
	return "\n## To check\n\nWhat these issues were built on changed, and the role could not read them all again. Check each against the change, then tick its box: it leaves this page.\n\n" + strings.Join(lines, "\n") + "\n", n
}

// rechecked is the report's line on the changes this run settled with no
// person, folded: each issue read again, and the change it was read with.
func (p *Plan) rechecked() string {
	var lines []string
	for _, c := range p.Rechecked {
		for _, t := range c.Touch {
			if !p.open[t.Issue] {
				continue
			}
			what := fmt.Sprintf("#%d %s had its %s rewritten", c.Issue, p.title(c.Issue), strings.Join(c.What, " and "))
			if c.Path != "" {
				what = fmt.Sprintf("`%s`, lines %s, which it was opened from, changed", c.Path, c.Lines)
			}
			lines = append(lines, fmt.Sprintf("- #%d %s — %s.", t.Issue, p.title(t.Issue), what))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	were := "were"
	if len(lines) == 1 {
		were = "was"
	}
	return fold(fmt.Sprintf("%s whose source changed %s read again: nothing to change", plural(len(lines), "issue"), were), strings.Join(lines, "\n"))
}
