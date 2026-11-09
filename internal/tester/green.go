package tester

// The green proof (ADR-0055, step 5): on the pull request that closes the
// issue — the tester's branch, marked ready — each test the tester froze
// passes on the head three runs out of three: the project's test command,
// three times one after the other, in one fresh checkout of the head. A
// test red in some runs and green in others is flaky: blocking, named,
// never run again until green (TestGen-LLM's flakiness filter, cheaper:
// three runs, not five). The tests required are the ids the tester's
// record keeps, never the head's tests the base lacks: a change to the
// build that stops one running blocks, named (ADR-0065, change 2). A test
// the change adds beyond them is required green too.

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/JN0V/workline/internal/verdict"
)

// greenRuns is how many times the head runs: each new test passes in all.
const greenRuns = 3

// green runs the green proof on HEAD, against the merge base of the
// record's base and HEAD.
func green(repo string, rec record, s Settings) verdict.Verdict {
	if v := unset(s); v != nil {
		v.Summary = "the green proof did not run"
		return *v
	}
	mb, err := mergeBase(repo, rec.Base)
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
	v := verdict.Verdict{Status: verdict.Pass}
	// The tester's tests, by the ids its record keeps: each found in the
	// head's reports, or a finding naming it.
	frozen := map[string]bool{}
	for _, t := range rec.Tests {
		found := rec.frozenIn(t, ids)
		if len(found) == 0 {
			v.Findings = append(v.Findings, verdict.Finding{Rule: "frozen-test-missing", Where: t,
				Message: fmt.Sprintf("a test the tester froze is not in the head's reports, in any of the %d runs: the change stops it running "+
					"(a build file, a filter, a skip) or renames it. Run it again as the tester wrote it; a test that does not fit, "+
					"object on its line, the comment opened by `workline:objection` (ADR-0055)", len(runs))})
		}
		for _, id := range found {
			frozen[id] = true
		}
	}
	if len(rec.Tests) == 0 {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "frozen-ids-unknown", Level: "warn", Where: "tester-record",
			Message: "the tester's record names its files, not its tests (written before it kept them): the proof takes the head's tests " +
				"the base lacks, so a test the change stops running is not seen; the tester's next write or rewrite records them"})
	}
	var added, old []string
	for id := range ids {
		switch _, ok := baseRun.results[id]; {
		case frozen[id]:
		case ok:
			old = append(old, id)
		default:
			added = append(added, id)
		}
	}
	slices.Sort(added)
	slices.Sort(old)
	var required []string
	for id := range frozen {
		required = append(required, id)
	}
	slices.Sort(required)
	required = append(required, added...)
	if len(rec.Tests) == 0 && len(added) == 0 {
		return *refuse("the green proof found no new test", verdict.Finding{Rule: "no-new-test", Where: s.Report,
			Message: fmt.Sprintf("the head's reports name no test the base's (%s) does not: the tester's tests did not run on the head", short(mb))})
	}
	for _, id := range required {
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
	what := fmt.Sprintf("the tester's %d test%s", len(rec.Tests), plural(len(rec.Tests)))
	if len(added) > 0 {
		what += fmt.Sprintf(" and %d more new test%s", len(added), plural(len(added)))
	}
	if len(rec.Tests) == 0 {
		what = fmt.Sprintf("%d new test%s", len(added), plural(len(added)))
	}
	v.Summary = what + " green on the head, three runs out of three"
	for _, f := range v.Findings {
		if f.Level == "" {
			v.Status = verdict.Block
			v.Summary = what + ", not each found green on the head three runs out of three"
			break
		}
	}
	return v
}

// frozenIn is the head's ids the record's test t is. An id from a report
// is itself; one the red proof named from its file, its build broken on
// the base (file:name, the file one the record freezes), is each id that
// holds name as a whole part: after a dot or at the start, before the
// end or a mark that is no word's nor a dot's (a parameter's `[`, a
// subtest's `/`). A loose match may only find a test, never lose one.
func (r record) frozenIn(t string, ids map[string]bool) []string {
	if ids[t] {
		return []string{t}
	}
	i := strings.LastIndex(t, ":")
	if i < 0 || r.Files[t[:i]] == "" || t[i+1:] == "" {
		return nil
	}
	var out []string
	for id := range ids {
		if namedAs(id, t[i+1:]) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// namedAs says whether id holds name as a whole part.
func namedAs(id, name string) bool {
	for i := 0; i <= len(id)-len(name); {
		j := strings.Index(id[i:], name)
		if j < 0 {
			return false
		}
		j += i
		end := j + len(name)
		before := j == 0 || id[j-1] == '.'
		after := end == len(id)
		if !after {
			c := []rune(id[end:])[0]
			after = c != '.' && c != '_' && !unicode.IsLetter(c) && !unicode.IsDigit(c)
		}
		if before && after {
			return true
		}
		i = j + 1
	}
	return false
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
