package tester

// The project's tests on a change not committed yet: what the developer
// proves its code with before its pull request opens (ADR-0054, step 3),
// the same test command and report as the tester's checks.

import (
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// Outcome is one run of the project's test command: the tests it reported
// passed and red (failed, an error, or skipped), and why the command
// exited non-zero, "" when it exited 0.
type Outcome struct {
	Passed []string
	Red    []string
	Exit   string
}

// Green says the run proves the change: tests reported, none red, the
// command exited 0.
func (o Outcome) Green() bool { return len(o.Passed) > 0 && len(o.Red) == 0 && o.Exit == "" }

// OnChange runs the project's test command once on rev, the patches laid on
// it, in a checkout apart, stopped past deadline when one is set. A finding
// when it did not run whole: a setting missing, a patch that does not
// apply, no report read, or the deadline passed (rule proof-over-time).
func OnChange(repo, rev string, patches []intent.Intention, s Settings, deadline time.Time) (Outcome, *verdict.Finding) {
	if v := unset(s); v != nil {
		return Outcome{}, &v.Findings[0]
	}
	tree, done, f := checkout(repo, rev)
	if f != nil {
		return Outcome{}, f
	}
	defer done()
	if v, err := layPatches(tree, patches); err != nil {
		return Outcome{}, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	} else if v != nil {
		return Outcome{}, &v.Findings[0]
	}
	s.deadline = deadline
	r, f := runIn(tree, s, "on the change")
	if f != nil {
		return Outcome{}, f
	}
	o := Outcome{Exit: r.failure}
	for id, res := range r.results {
		if res == passed {
			o.Passed = append(o.Passed, id)
		} else {
			o.Red = append(o.Red, id)
		}
	}
	slices.Sort(o.Passed)
	slices.Sort(o.Red)
	if o.Exit != "" {
		o.Exit += ": " + strings.TrimSpace(tail(r.output, 5))
	}
	return o, nil
}
