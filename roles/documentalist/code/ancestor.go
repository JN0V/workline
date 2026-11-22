package documentalist

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/kit/docmark"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// checkedNotAncestor refuses a `checked` a change sets to a commit its base
// lacks: a squash merge drops the commits of the branch, and the header would
// name a commit main does not hold (#2029). Only the docs whose `checked` the
// range changed are looked at, this repository's commit only.
func (r *preRun) checkedNotAncestor() error {
	if !r.atPush {
		return nil
	}
	base := gitrange.Base(rangeOf(r.runDir))
	if base == "" {
		return nil
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
		if _, err := git(r.repo, "merge-base", "--is-ancestor", now+"^{commit}", base); err != nil {
			short := now
			if len(short) > 12 {
				short = short[:12]
			}
			r.findings = append(r.findings, verdict.Finding{Rule: "checked-not-ancestor", Where: d.Path, Level: "block",
				Message: fmt.Sprintf("`checked` names %s, a commit the base %s does not hold: a squash merge drops it; name a commit of the default branch", short, strings.TrimSpace(shortRef(base)))})
		}
	}
	return nil
}

// isAncestor tells whether a is an ancestor of b: git exits 1 for "no", and
// any other failure (a base not fetched, a commit missing from a shallow
// clone: 128) is an error, never a "no".
func isAncestor(repo, a, b string) (bool, error) {
	cmd := exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", a, b)
	var errOut strings.Builder
	cmd.Stderr = &errOut
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("%s", strings.TrimSpace(errOut.String()))
}

func shortRef(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
