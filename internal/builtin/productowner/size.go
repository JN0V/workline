package productowner

// An issue's size, the product owner's to judge (ADR-0064, step 4): at
// ready, predicted against its normal; on a run too big for its role,
// asked. Past it, the issue is split — `split` turns on where it is off
// by the role's default, never where the project set it off — or granted
// the tier large.

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/work"
)

// noSplit says why is is offered no split for its size, "" when it is:
// a part of a split (depth 1, §4), a project that set split off, or a
// bug whose reproduction test is not written yet (first, testsFirst).
func noSplit(is forge.Issue, off bool, first string) string {
	switch {
	case off:
		return "this project allows no splits (`split` off in its settings): grant it, or leave it to a person"
	case backlog.PartOf(is.Body) > 0:
		return "a part of a split is never split again"
	case first != "":
		return first
	}
	return ""
}

// testsFirst says why a bug is offered no split yet, "" when it may be
// split (#1212): a bug goes to its tests first, the tester writing the
// reporter's symptom as a red test on its Draft, then it is split. None
// where the line runs no tester.
func testsFirst(f any, is forge.Issue, runsTester bool) string {
	if kind, _ := backlog.KindOf(is.Labels); kind != backlog.KindBug || !runsTester {
		return ""
	}
	if m, ok := f.(interface {
		OpenMergeRequests(prefix string) ([]string, error)
	}); ok {
		if drafts, err := m.OpenMergeRequests(tester.TestsBranch); err == nil && slices.Contains(drafts, tester.TestsBranch+strconv.Itoa(is.ID)) {
			return ""
		}
	}
	return "a bug is split once the tester wrote its reproduction test, on its Draft: `ready` it for its tests first; its split is asked again if a run on it crosses its size"
}

// splitSaid is the split a size question offers on is, or why none.
func splitSaid(is forge.Issue, off bool, first string) string {
	if why := noSplit(is, off, first); why != "" {
		return " (no `split`: " + why + ")"
	}
	return "; or split it, when its parts each pass their own scenarios apart: `split` with `too-big: true`, 2 to 6 parts, then `blocked`, `person: true`: a person accepts it from what its parts deliver"
}

// predictSaid asks for is's size at ready, against its normal (§2): split
// or granted before any token is spent on it. None for a parent split
// already: its parts carry its size.
func predictSaid(b forge.Backlog, open []forge.Issue, is forge.Issue, st *backlog.State, off bool, first string) string {
	if len(backlog.SplitInto(is, st)) > 0 {
		return ""
	}
	n, measured := line.IssueNormal(b, open, is)
	from := "the seed, until about 10 issues of its kind are done here"
	if measured {
		from = "this project's own, its 85th percentile"
	}
	var past []string
	if noSplit(is, off, first) == "" {
		past = append(past, "`split` it with `too-big: true`, 2 to 6 parts each passing its own scenarios apart, and no `ready` on it")
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
	if why := noSplit(is, off, first); (off || first != "") && why != "" {
		said += " (no `split`: " + why + ")"
	}
	return fmt.Sprintf("Its size, before `ready`: predict what the line's runs on it will spend, from its %s's files, its scenarios and its effort, against its normal here: %s fresh tokens an issue of its kind (%s). Well past it: %s. Within it, nothing more.\n",
		work.Name(work.Scope), line.Tokens(n), from, said)
}

// spentSaid is what is spent so far against its normal and its backstop,
// told the product owner asked on its size (line.SpentSaid).
func spentSaid(b forge.Backlog, open []forge.Issue, is forge.Issue, comments []string) string {
	return line.SpentSaid(b, open, is, comments)
}
