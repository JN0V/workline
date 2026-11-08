package productowner

// An issue's size, the product owner's to judge (ADR-0064, step 4): at
// ready, predicted against its normal; on a run too big for its role,
// asked. Past it, the issue is split — `split` turns on, off as it is by
// default, a part never split again — or granted the tier large.

import (
	"fmt"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/work"
)

// splitSaid is the split a size question offers on is: none for a part
// of a split (depth 1, §4).
func splitSaid(is forge.Issue) string {
	if backlog.PartOf(is.Body) > 0 {
		return ""
	}
	return "; or split it, when its parts each pass their own Verification apart: `split` with `too-big: true`, 2 to 6 parts, then `blocked`, `person: true`: a person accepts it from what its parts deliver"
}

// predictSaid asks for is's size at ready, against its normal (§2): split
// or granted before any token is spent on it.
func predictSaid(b forge.Backlog, open []forge.Issue, is forge.Issue) string {
	n, measured := line.IssueNormal(b, open, is)
	from := "the seed, until about 10 issues of its kind are done here"
	if measured {
		from = "this project's own, its 85th percentile"
	}
	var past []string
	if backlog.PartOf(is.Body) == 0 {
		past = append(past, "`split` it with `too-big: true`, 2 to 6 parts each passing its own Verification apart, and no `ready` on it")
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
	return fmt.Sprintf("Its size, before `ready`: predict what the line's runs on it will spend, from its %s's files, its scenarios and its effort, against its normal here: %s fresh tokens an issue of its kind (%s). Well past it: %s. Within it, nothing more.\n",
		work.Name(work.Scope), line.Tokens(n), from, said)
}
