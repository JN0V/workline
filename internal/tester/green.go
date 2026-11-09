package tester

// The green proof (ADR-0055, step 5): on the pull request that closes the
// issue, marked ready — the developer's, stacked on the tester's Draft, or
// the Draft itself — each test the tester froze passes on the head three
// runs out of three: the project's test command, three times one after
// the other, in one fresh checkout of the head. A
// test red in some runs and green in others is flaky: blocking, named,
// never run again until green (TestGen-LLM's flakiness filter, cheaper:
// three runs, not five). The tests required are the ids the tester's
// record keeps, never the head's tests the base lacks: a change to the
// build that stops one running blocks, named (ADR-0065, change 2). A test
// the change adds beyond them is required green too.

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/JN0V/workline/internal/verdict"
)

// greenRuns is how many times the head runs: each new test passes in all.
const greenRuns = 3

// green runs the green proof on HEAD, against the merge base of the
// record's base and HEAD. A frozen test named from its file that no head
// id holds, while the head holds new tests it may run as, is the judge's
// to tell (renamed): never a block on a guess (ADR-0065, rule A).
func green(repo string, rec record, s Settings) (verdict.Verdict, *renamed) {
	if v := unset(s); v != nil {
		v.Summary = "the green proof did not run"
		return *v, nil
	}
	mb, err := mergeBase(repo, rec.Base)
	if err != nil {
		return *refuse("the green proof did not run", verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}), nil
	}
	baseRun, f := run(repo, mb, nil, "", s, "on the base "+short(mb))
	if f != nil {
		return *refuse("the green proof did not run", *f), nil
	}
	// The head's runs share one fresh checkout: a second build there is
	// the runner's cache, not a new compile (Go: 1 s, not 4.5 s, on
	// workline's tester package). A test that a run before it breaks is
	// flaky too: it fails on a person's second run.
	tree, done, f := checkout(repo, "HEAD")
	if f != nil {
		return *refuse("the green proof did not run", *f), nil
	}
	defer done()
	var runs []ran
	for i := 1; i <= greenRuns; i++ {
		r, f := runIn(tree, s, fmt.Sprintf("on the head, run %d of %d", i, greenRuns))
		if f != nil {
			return *refuse("the green proof did not run", *f), nil
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
	var missing []string
	for _, t := range rec.Tests {
		found := rec.frozenIn(t, ids)
		if len(found) == 0 {
			missing = append(missing, t)
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
	// A frozen id missing: blocking when it is a report's test case (it
	// converges) or when no new test of the head could be it; one named
	// from its file, or a record the red run wrote with no class, failed
	// (a file it could not collect), is the judge's.
	var doubt *renamed
	for _, t := range missing {
		if (named(rec, t) || slices.Contains(rec.Errored, t)) && len(added) > 0 {
			if doubt == nil {
				doubt = &renamed{Candidates: added, Runs: len(runs)}
			}
			doubt.Missing = append(doubt.Missing, t)
			continue
		}
		v.Findings = append(v.Findings, missingFinding(t, len(runs), ""))
	}
	if doubt != nil {
		doubt.Facts = renameFacts(repo, mb, rec, doubt)
	}
	var required []string
	for id := range frozen {
		required = append(required, id)
	}
	slices.Sort(required)
	required = append(required, added...)
	if len(rec.Tests) == 0 && len(added) == 0 {
		return *refuse("the green proof found no new test", verdict.Finding{Rule: "no-new-test", Where: s.Report,
			Message: fmt.Sprintf("the head's reports name no test the base's (%s) does not: the tester's tests did not run on the head", short(mb))}), nil
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
	return settled(v), doubt
}

// greenSaid and redSaid end a green proof's summary, passed or not.
const (
	greenSaid = " green on the head, three runs out of three"
	redSaid   = ", not each found green on the head three runs out of three"
)

// settled is v blocking, its summary said so, when a finding blocks.
func settled(v verdict.Verdict) verdict.Verdict {
	for _, f := range v.Findings {
		if f.Level == "" && v.Status != verdict.Block {
			v.Status = verdict.Block
			v.Summary = strings.Replace(v.Summary, greenSaid, redSaid, 1)
			break
		}
	}
	return v
}

// missingFinding is a frozen test in none of the runs, blocking; why is
// the judge's word when it told so.
func missingFinding(t string, runs int, why string) verdict.Finding {
	msg := fmt.Sprintf("a test the tester froze is not in the head's reports, in any of the %d runs: the change stops it running "+
		"(a build file, a filter, a skip) or renames it", runs)
	if why != "" {
		msg = "a test the tester froze is not in the head's reports, and the judge says it was dropped, not run under another name: " + why
	}
	return verdict.Finding{Rule: "frozen-test-missing", Where: t,
		Message: msg + ". Run it again as the tester wrote it; a test that does not fit, object on its line, the comment opened by `workline:objection` (ADR-0055)"}
}

// named says whether t is a test the red proof named from its file
// (file:name, the file one the record freezes): no report named it yet.
func named(r record, t string) bool {
	i := strings.LastIndex(t, ":")
	return i > 0 && r.Files[t[:i]] != "" && t[i+1:] != ""
}

// frozenIn is the head's ids the record's test t is. An id from a report
// is itself; one the red proof named from its file, its build broken on
// the base (file:name, the file one the record freezes), is each id that
// holds name as a whole part: after a dot or at the start, before the
// end or a mark that is no word's nor a dot's (a parameter's `[`, a
// subtest's `/`), qualified by its file: what comes before the name, when
// anything does, holds the file's folder or its name without extension
// as a word (`calc.TestNew` for `calc/new_test.go`, `tests.test_x.f` for
// `tests/test_x.py`), never another folder's test of the same name. A
// miss is the judge's (renamed.go), never a block on this guess.
func (r record) frozenIn(t string, ids map[string]bool) []string {
	if ids[t] {
		return []string{t}
	}
	if !named(r, t) {
		return nil
	}
	i := strings.LastIndex(t, ":")
	var out []string
	for id := range ids {
		if namedAs(id, t[i+1:], t[:i]) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// namedAs says whether id holds name as a whole part, qualified by file.
func namedAs(id, name, file string) bool {
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
		if before && after && (j == 0 || qualifiedBy(id[:j-1], file)) {
			return true
		}
		i = j + 1
	}
	return false
}

// qualifiedBy says whether a test id's qualifier (its package, class or
// module) holds, as a word, file's folder or its name without extension.
func qualifiedBy(qualifier, file string) bool {
	words := strings.FieldsFunc(qualifier, func(c rune) bool { return c != '_' && !unicode.IsLetter(c) && !unicode.IsDigit(c) })
	stem := path.Base(file)
	if k := strings.Index(stem, "."); k > 0 {
		stem = stem[:k]
	}
	dir := path.Base(path.Dir(file))
	return slices.ContainsFunc(words, func(w string) bool { return w == stem || (dir != "." && dir != "/" && w == dir) })
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
