package productowner

// An issue's size, the product owner's to judge (ADR-0064, step 4): at
// ready, predicted against its normal; on a run too big for its role,
// asked. Past it, the issue is split — `split` turns on where it is off
// by the role's default, never where the project set it off — or granted
// the tier large.

import (
	"fmt"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/work"
)

// noSplit says why is is offered no split for its size, "" when it is:
// a part of a split (depth 1, §4), or a project that set split off.
func noSplit(is forge.Issue, off bool) string {
	switch {
	case off:
		return "this project allows no splits (`split` off in its settings): grant it, or leave it to a person"
	case backlog.PartOf(is.Body) > 0:
		return "a part of a split is never split again"
	}
	return ""
}

// splitSaid is the split a size question offers on is, or why none.
func splitSaid(is forge.Issue, off bool) string {
	if why := noSplit(is, off); why != "" {
		return " (no `split`: " + why + ")"
	}
	return "; or split it, when its parts each pass their own " + work.Name(work.Verification) + " apart: `split` with `too-big: true`, 2 to 6 parts, then `blocked`, `person: true`: a person accepts it from what its parts deliver"
}

// predictSaid asks for is's size at ready, against its normal (§2): split
// or granted before any token is spent on it. None for a parent split
// already: its parts carry its size.
func predictSaid(b forge.Backlog, open []forge.Issue, is forge.Issue, st *backlog.State, off bool) string {
	if len(backlog.SplitInto(is, st)) > 0 {
		return ""
	}
	n, measured := line.IssueNormal(b, open, is)
	from := "the seed, until about 10 issues of its kind are done here"
	if measured {
		from = "this project's own, its 85th percentile"
	}
	var past []string
	if noSplit(is, off) == "" {
		past = append(past, "`split` it with `too-big: true`, 2 to 6 parts each passing its own "+work.Name(work.Verification)+" apart, and no `ready` on it")
	}
	if backlog.SizeTier(is.Labels) == "" {
		past = append(past, fmt.Sprintf("its need one no part proves apart, `size` with `tier: %s`, then `ready`", backlog.TierLarge))
	}
	if len(past) == 0 {
		return ""
	}
	said := past[0]
	if len(past) > 1 {
		said += "; or, " + past[1]
	}
	if why := noSplit(is, off); off && why != "" {
		said += " (" + why + ")"
	}
	return fmt.Sprintf("Its size, before `ready`: predict what the line's runs on it will spend, from its %s's files, its scenarios and its effort, against its normal here: %s fresh tokens an issue of its kind (%s). Well past it: %s. Within it, nothing more.\n",
		work.Name(work.Scope), line.Tokens(n), from, said)
}
