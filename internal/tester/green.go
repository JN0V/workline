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
	"os"
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
	// The base's tests by their ids alone: those a run of the same tree
	// by the same command found, when one is kept (runcache.go), else
	// one run of it.
	baseIDs, at, kept := readIDs(repo, mb, s.Command, s.Report)
	if kept {
		fmt.Fprintf(os.Stderr, "workline: tester: the base %s's %d tests as a run of the same tree found them at %s UTC, not run again\n",
			short(mb), len(baseIDs), at.Format("2006-01-02 15:04:05"))
	} else {
		baseRun, f := run(repo, mb, nil, "", s, "on the base "+short(mb))
		if f != nil {
			return *refuse("the green proof did not run", *f), nil
		}
		baseIDs = map[string]bool{}
		for id := range baseRun.results {
			baseIDs[id] = true
		}
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
	// The head's tests, kept as its first run found them: once merged up
	// to date, its tree is the next pull request's base.
	keepIDs(repo, "HEAD", s.Command, s.Report, runs[0])
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
		switch ok := baseIDs[id]; {
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
	// from its file, or by its file alone, or a record the red run wrote with no class, failed
	// (a file it could not collect), is the judge's.
	var doubt *renamed
	for _, t := range missing {
		if (named(rec, t) || fileAlone(rec, t) || slices.Contains(rec.Errored, t)) && len(added) > 0 {
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

// fileAlone says whether t is a test the red proof named by its file
// alone (the file one the record freezes): its build broke on the base,
// and its ecosystem has no test-name to name its tests by (#1293).
func fileAlone(r record, t string) bool {
	return r.Files[t] != ""
}

// ofFile is the ids of results that are a test of file, sorted: an id
// one of whose dotted parts, its name too, is the file by its name
// without extension, the words before it the folders' last ones, or
// none (`test_mul.main`, `tests.test_mul.main`, `tests/test_mul.c.main`
// or, a runner naming a test by its file, `tests.test_mul` for
// `tests/test_mul.c`). A guess: its miss is the judge's, never a block.
func (r ran) ofFile(file string) []string {
	var out []string
	for id := range r.results {
		if ofFile(id, file) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

func ofFile(id, file string) bool {
	return fileQualified(id, file)
}

// frozenIn is the head's ids the record's test t is. An id from a report
// is itself; one named by its file alone, each id of a test of that file
// (ofFile); one the red proof named from its file, its build broken on
// the base (file:name, the file one the record freezes), is each id that
// holds name as a whole part: after a dot or at the start, before the
// end or a mark that is no word's nor a dot's (a parameter's `[`, a
// subtest's `/`), qualified by its file: what comes before the name, when
// anything does, is the file's folder or the file (qualifiedBy:
// `calc.TestNew` for `calc/new_test.go`, `tests.test_x.f` for
// `tests/test_x.py`), never a nested folder's test of the same name
// (`calc/sub.TestNew`), nor a same-named folder's elsewhere. A miss is
// the judge's (renamed.go), never a block on this guess.
func (r record) frozenIn(t string, ids map[string]bool) []string {
	if ids[t] {
		return []string{t}
	}
	alone := fileAlone(r, t)
	if !alone && !named(r, t) {
		return nil
	}
	i := strings.LastIndex(t, ":")
	var out []string
	for id := range ids {
		if alone && ofFile(id, t) || !alone && namedAs(id, t[i+1:], t[:i]) {
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
// module) is file's, its words against the words of file's folders:
//   - a package that is the folder: one ends with the other (a Go
//     package `example.com/app/calc` for `calc/new_test.go`, a runner's
//     `calc`);
//   - a module or class that is the file: a word is the file's name
//     without extension, the words before it the folders' last ones, or
//     none (`tests.test_x`, `tests.test_x.TestK` or `test_x` for
//     `tests/test_x.py`; `com.example.CalcTest` for
//     `src/test/java/com/example/CalcTest.java`; `Calc.Tests.CalcTests`
//     for `tests/Calc.Tests/CalcTests.cs`);
//   - a class named after its file, its name capitalised, its package
//     apart from the folders: past the folders' last words it may end
//     with, none of its words is a folder's (Kotlin's
//     `com.example.CalcTest` for `src/test/kotlin/CalcTest.kt`, a C#
//     namespace). Another folder's class of that name cannot be told
//     from it.
//
// A folder nested in the file's (`calc/sub`), or one of the same name
// elsewhere (`example.com/app/cmd/calc` for `internal/calc/new_test.go`,
// `pkg.tests.test_x` for `tests/test_x.py`), is not it.
func qualifiedBy(qualifier, file string) bool {
	q := idWords(qualifier)
	if len(q) == 0 {
		return false
	}
	dirs := idWords(path.Dir(file))
	if len(dirs) > 0 && (endsWith(q, dirs) || endsWith(dirs, q)) {
		return true
	}
	return fileQualified(qualifier, file)
}

// fileQualified says whether a qualifier is file's: a module or class
// that is the file, or a class named after it (qualifiedBy). A stem of
// several words (`test-mul`) is one part of the qualifier, cut where a
// mark the stem does not hold is: `tests.test-mul-big` is not
// `tests/test-mul.sh`'s.
func fileQualified(qualifier, file string) bool {
	stem := path.Base(file)
	if k := strings.Index(stem, "."); k > 0 {
		stem = stem[:k]
	}
	words := idWords(stem)
	if len(words) == 0 {
		return false
	}
	marks := strings.Map(func(c rune) rune { // the stem's marks between its words
		if isWord(c) {
			return -1
		}
		return c
	}, stem)
	parts := strings.FieldsFunc(qualifier, func(c rune) bool { return !isWord(c) && !strings.ContainsRune(marks, c) })
	dirs := idWords(path.Dir(file))
	var before []string // the words of the parts before parts[k]
	for _, part := range parts {
		if slices.Equal(idWords(part), words) &&
			(endsWith(dirs, before) || unicode.IsUpper([]rune(stem)[0]) && apart(before, dirs)) {
			return true
		}
		before = append(before, idWords(part)...)
	}
	return false
}

// isWord says whether c is a word's: a letter, a digit or an underscore.
func isWord(c rune) bool {
	return c == '_' || unicode.IsLetter(c) || unicode.IsDigit(c)
}

// idWords is s's words: runs of letters, digits and underscores.
func idWords(s string) []string {
	return strings.FieldsFunc(s, func(c rune) bool { return !isWord(c) })
}

// endsWith says whether a's last words are b.
func endsWith(a, b []string) bool {
	return len(b) <= len(a) && slices.Equal(a[len(a)-len(b):], b)
}

// apart says whether a package leaves the folders aside: past the last
// words both end with, none of its words is a folder's.
func apart(pkg, dirs []string) bool {
	for len(pkg) > 0 && len(dirs) > 0 && pkg[len(pkg)-1] == dirs[len(dirs)-1] {
		pkg, dirs = pkg[:len(pkg)-1], dirs[:len(dirs)-1]
	}
	return !slices.ContainsFunc(pkg, func(w string) bool { return slices.Contains(dirs, w) })
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
