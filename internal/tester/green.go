package tester

// The green proof (ADR-0055, step 5): on the pull request that closes the
// issue — the tester's branch, marked ready — each test the change adds
// passes on the head three runs out of three: the project's test command,
// three times one after the other, in one fresh checkout of the head. A
// test red in some runs and green in others is flaky: blocking, named,
// never run again until green (TestGen-LLM's flakiness filter, cheaper:
// three runs, not five). The new tests are those the head's reports hold
// and the base's does not.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// greenRuns is how many times the head runs: each new test passes in all.
const greenRuns = 3

// green runs the green proof on HEAD, against the merge base of base and
// HEAD.
func green(repo, base string, s Settings) verdict.Verdict {
	if v := unset(s); v != nil {
		v.Summary = "the green proof did not run"
		return *v
	}
	mb, err := mergeBase(repo, base)
	if err != nil {
		return *refuse("the green proof did not run", verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()})
	}
	baseRun, f := run(repo, mb, nil, "", s, "on the base "+short(mb))
	if f != nil {
		return *refuse("the green proof did not run", *f)
	}
	// The head's runs share one fresh checkout: a second build there is
	// the runner's cache, not a new compile (Go: 1 s, not 4.5 s, on
	// workline's tester package). A test that a run before it breaks is
	// flaky too: it fails on a person's second run.
	tree, done, f := checkout(repo, "HEAD")
	if f != nil {
		return *refuse("the green proof did not run", *f)
	}
	defer done()
	var runs []ran
	for i := 1; i <= greenRuns; i++ {
		r, f := runIn(tree, s, fmt.Sprintf("on the head, run %d of %d", i, greenRuns))
		if f != nil {
			return *refuse("the green proof did not run", *f)
		}
		runs = append(runs, r)
	}
	ids := map[string]bool{}
	for _, r := range runs {
		for id := range r.results {
			ids[id] = true
		}
	}
	var added, old []string
	for id := range ids {
		if _, ok := baseRun.results[id]; ok {
			old = append(old, id)
		} else {
			added = append(added, id)
		}
	}
	slices.Sort(added)
	slices.Sort(old)
	if len(added) == 0 {
		return *refuse("the green proof found no new test", verdict.Finding{Rule: "no-new-test", Where: s.Report,
			Message: fmt.Sprintf("the head's reports name no test the base's (%s) does not: the tester's tests did not run on the head", short(mb))})
	}
	v := verdict.Verdict{Status: verdict.Pass}
	for _, id := range added {
		if f := judge(id, runs); f != nil {
			v.Findings = append(v.Findings, *f)
		}
	}
	// A test the base already had, flaky here, is not the change's: said,
	// not blocking (principle 8).
	for _, id := range old {
		if f := judge(id, runs); f != nil && f.Rule == "test-flaky" {
			f.Level = "warn"
			f.Message += "; it is the base's, not the change's: an issue naming it, the ecosystem's skip, never deleted"
			v.Findings = append(v.Findings, *f)
		}
	}
	v.Summary = fmt.Sprintf("%d new test%s green on the head, three runs out of three", len(added), plural(len(added)))
	for _, f := range v.Findings {
		if f.Level == "" {
			v.Status = verdict.Block
			v.Summary = fmt.Sprintf("%d new test%s, not each green on the head three runs out of three", len(added), plural(len(added)))
			break
		}
	}
	return v
}

// judge is id's finding from the runs: nil when it passed in each.
func judge(id string, runs []ran) *verdict.Finding {
	var red []string
	for i, r := range runs {
		if res, ok := r.results[id]; !ok || res != passed {
			red = append(red, fmt.Sprint(i+1))
		}
	}
	switch len(red) {
	case 0:
		return nil
	case len(runs):
		return &verdict.Finding{Rule: "test-red-on-head", Where: id,
			Message: fmt.Sprintf("red in each of the %d runs on the head (failed, skipped or not run): the change does not do what it checks", len(runs))}
	}
	return &verdict.Finding{Rule: "test-flaky", Where: id,
		Message: fmt.Sprintf("passed %d of %d runs, red in run %s: a flaky test, not run again until green. "+
			"Find the cause (shared state, order, time, the network) in the code or the test: a test of the tester's, object on its line (ADR-0055)",
			len(runs)-len(red), len(runs), strings.Join(red, ", "))}
}

// mergeBase is the merge base of base and HEAD; base read as a branch of
// origin when it is not one here, as in a CI checkout.
func mergeBase(repo, base string) (string, error) {
	var errs []string
	for _, ref := range []string{base, "origin/" + base} {
		out, err := git(repo, "merge-base", ref, "HEAD")
		if err == nil {
			return strings.TrimSpace(out), nil
		}
		errs = append(errs, err.Error())
	}
	return "", fmt.Errorf("the merge base of %s and HEAD is not known (not fetched, or a shallow clone): %s", base, strings.Join(errs, "; "))
}
