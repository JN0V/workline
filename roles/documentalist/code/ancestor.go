package documentalist

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/kit/docmark"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// checkedNotAncestor refuses a `checked` a change sets to a commit its base
// lacks: a squash merge drops the commits of the branch, and the header would
// name a commit main does not hold (#2029). Only the docs whose `checked` the
// range changed are looked at, this repository's commit only. A doc whose last change in the range is the
// documentalist's own (its trailer) is let through: its fix vouches at the
// branch's HEAD.
func (r *preRun) checkedNotAncestor() {
	if !r.atPush {
		return
	}
	base := gitrange.Base(rangeOf(r.runDir))
	if base == "" {
		return
	}
	for _, d := range r.docs {
		now := d.Checked[""]
		if now == "" {
			continue
		}
		if old, err := git(r.repo, "show", base+":"+d.Path); err == nil {
			if od, err := docmark.ParseDoc(d.Path, []byte(old)); err == nil && od != nil && od.Checked[""] == now {
				continue
			}
		}
		// The role's own fix vouches at the branch's HEAD, which it cannot
		// replace by a base commit: it is let through, not refused by its
		// own check.
		if own, err := git(r.repo, "log", "-1", "--format=%(trailers:key=Workline-Role,valueonly)", base+"..HEAD", "--", d.Path); err == nil && strings.TrimSpace(own) == "documentalist" {
			continue
		}
		if _, err := git(r.repo, "merge-base", "--is-ancestor", now+"^{commit}", base); err != nil {
			short := now
			if len(short) > 12 {
				short = short[:12]
			}
			r.findings = append(r.findings, verdict.Finding{Rule: "checked-not-ancestor", Where: d.Path, Level: "block",
				Message: fmt.Sprintf("`checked` names %s, a commit the base %s does not hold: a squash merge drops it; name a commit of the default branch", short, strings.TrimSpace(shortRef(base)))})
		}
	}
}

func shortRef(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
