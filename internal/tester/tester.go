// Package tester holds the tester's checks that need no agent (ADR-0055):
// the red-first check, which proves from the project's own test runner's
// JUnit XML report that each test a change adds fails on its base.
package tester

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/testdef"
	"github.com/JN0V/workline/internal/verdict"
)

// Settings are the tester's project settings the check reads
// (roles/tester/role.yaml).
type Settings struct {
	// Command runs the project's tests and writes a JUnit XML report;
	// {report} in it is the report's path.
	Command string `json:"test-command"`
	// Report is where the command writes the report, from the
	// repository's root: a path, or a pattern, `**` spanning folders
	// (`**/build/test-results/**/*.xml`), each file it names read. The
	// files it names are removed before a run, but those git tracks, a
	// nested repository holds, or a folder linked outside the root leads
	// to: those are read, never removed.
	Report string `json:"test-report"`
	// Paths are the test files, as patterns (`**` spans folders); unset,
	// the generic ones (testdef, ADR-0065). Homes: where a test's code
	// lives (Tests).
	Paths []string       `json:"test-paths"`
	Homes []testdef.Home `json:"test-homes"`
	// ModuleFiles name the file marking a build module's folder
	// (`build.gradle.kts`, `pom.xml`): set, `{packages}` is the modules
	// a change touches, not the folders holding tests (#1176).
	ModuleFiles []string `json:"module-files"`
	// CodeLinesMax caps the lines of code and tests given to the agent
	// when it writes an issue's tests, all files together.
	CodeLinesMax int `json:"code-lines-max"`
	// MaxTurns caps the turns of an agent that reads the project itself
	// (ADR-0063 §8), told in its task; 0, none told.
	MaxTurns int `json:"ai-max-turns"`
	// Exchanges bounds an objection's thread: the tester's answers in it
	// before the product owner settles it; objected to again after,
	// the issue is blocked (ADR-0055, ADR-0059).
	Exchanges int `json:"objection-exchanges"`
	// BuildErrors tell, by ecosystem, a build a new test broke by a name
	// the code lacks from one broken otherwise.
	BuildErrors []BuildErrors `json:"build-errors"`
	// The judge of a broken build may read the base first (ADR-0045):
	// its rounds of reads, and the lines read, all rounds together.
	JudgeReadRounds int `json:"judge-read-rounds"`
	JudgeReadMax    int `json:"judge-read-lines-max"`
	// The curator's measures (`workline test measure`): the command run,
	// when not the test command (one writing a coverage report too); the
	// runs a measure; the slowest tests named; the window of measures a
	// test never red is counted over; the corpus's time budget; the
	// coverage report the command writes, Go's profile or LCOV.
	CuratorCommand  string `json:"curator-command"`
	CuratorRuns     int    `json:"curator-runs"`
	CuratorSlowest  int    `json:"curator-slowest"`
	CuratorWindow   int    `json:"curator-window"`
	SuiteMaxSeconds int    `json:"suite-max-seconds"`
	CoverageReport  string `json:"coverage-report"`
	// The curator's removals (`workline test curate`, step 7): the test
	// files it may change; files listing tests it never removes; the
	// tests a run removes at most; the proof's command on the touched
	// folders ({packages}), and the mutation tool's on each ({package}),
	// writing MutationReport; the proof's time budget, in seconds.
	CuratorPaths    []string `json:"curator-paths"`
	CuratorKeep     []string `json:"curator-keep"`
	RemovalsMax     int      `json:"curator-removals-max"`
	ProofCommand    string   `json:"curator-proof-command"`
	MutationCommand string   `json:"curator-mutation-command"`
	MutationReport  string   `json:"curator-mutation-report"`
	ProofSeconds    int      `json:"curator-proof-seconds"`
	// DraftCommand proves a Draft's tests red on the base on the folders
	// that hold them alone ({packages}), base and tests; unset, or naming
	// no {packages}: test-command, the whole suite, twice (#979).
	DraftCommand string `json:"draft-command"`
	// deadline, when set, stops a test command past it (the proof's
	// budget); budget names the setting it comes of, when not
	// curator-proof-seconds.
	deadline time.Time
	budget   string
	// whole, when set, runs the test command past suite-max-seconds: the
	// curator's measure times the corpus, and warns over it.
	whole bool
	// tests, when set, are the change's test files: a run stopped at its
	// time says which of them no report written by then holds (#1425).
	tests []string
	// packages, when set, are the folders a run of Command takes in its
	// {packages}: those of them its tree holds.
	packages []string
	// issue, when set, is the text of the issue the tests are written
	// for: a missing name it names verbatim, the base lacking it, is red
	// at once; any other broken build is the judge's (ADR-0065).
	issue string
}

// SettingsFrom reads the tester's merged settings.
func SettingsFrom(m map[string]any) (Settings, error) {
	var s Settings
	data, err := json.Marshal(m)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("the tester's settings: %w", err)
	}
	if testdef.Check(s.Homes) != nil {
		s.Homes = nil // ignored, as every role does: the generic homes; doctor says why
	}
	d := s.Tests()
	s.Paths, s.Homes = d.Paths, d.Homes
	return s, nil
}

// Tests is the project's one definition of a test file, every role's
// (ADR-0065, change 3): its test-paths and test-homes, each unset the
// generic one.
func (s Settings) Tests() testdef.Def { return testdef.From(s.Paths, s.Homes) }

// Check is the red-first check on the change from the merge base of base
// and HEAD to HEAD, as committed: the tests HEAD's report holds that the
// base's does not are the new ones; each must fail on the base with the
// change's test files laid on it, and, when the change touches more than
// tests, pass on HEAD. No report read is never a pass. A build broken
// past the fast path, with no agent to judge it, is red unproven: a
// person reads it (ADR-0065).
func Check(repo, base string, s Settings) *verdict.Verdict {
	p := prove(repo, base, s)
	if p.doubt != nil && p.v.Status == verdict.Pass {
		return p.doubt.waiting(*p.v)
	}
	return p.v
}

// proof is what the red proof found: its verdict and the new tests. When
// the build broke on the base otherwise than the fast path passes, doubt
// holds the facts the judge reads, and the verdict is the one its yes
// gives.
type proven struct {
	v     *verdict.Verdict
	added []string
	// errored: the added ids the red run wrote with no class, failed, or
	// that may be a runner's record of their suite, alone in it
	errored []string
	doubt   *doubt
	errs    []buildError // the errors of the build with the tests on the base
}

// prove is Check, with the new tests it found, sorted, and its doubt.
func prove(repo, base string, s Settings) proven {
	refused := func(v *verdict.Verdict) proven { return proven{v: v} }
	if v := unset(s); v != nil {
		return refused(v)
	}
	ch, err := changeOf(repo, base, s.Paths)
	if err != nil {
		return refused(refuse("the red-first check did not run", verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}))
	}
	s.tests = ch.tests
	baseRun, f := run(repo, ch.base, nil, "", s, "on the base "+short(ch.base))
	if f != nil {
		return refused(refuse("the red-first check did not run", *f))
	}
	// A run that wrote no report is read by its compiler's lines when it
	// is the one where the change's tests were laid on the base: a build
	// that stops at the tests' compiling writes none (Gradle's), and that
	// is no proof yet either way (#1192).
	onHead := len(ch.other) > 0
	headRun, f := run(repo, ch.head, nil, "", s, "on the head "+short(ch.head))
	var noReport *verdict.Finding
	switch {
	case f != nil && !onHead && f.Rule == "report-missing":
		noReport = f
	case f != nil:
		return refused(refuse("the red-first check did not run", *f))
	}
	// Tests alone changed: the head is the base with its tests, and red
	// there is all a change holding no code yet can prove.
	redRun := headRun
	if onHead {
		if redRun, f = run(repo, ch.base, ch.tests, ch.head, s, "on the base "+short(ch.base)+" with the change's tests"); f != nil {
			if f.Rule != "report-missing" {
				return refused(refuse("the red-first check did not run", *f))
			}
			noReport = f
		}
	}
	// The build where the change's tests were laid on the base: each error
	// a missing name the base lacks and the issue names is red, proven
	// (the fast path); any other error is the judge's to read, its facts
	// raw (ADR-0065, change 1).
	rules, _ := compileBuildErrors(s.BuildErrors) // unset said a bad one
	errs := buildErrorsIn(redRun.output, ch.tests, rules, redRun.roots...)
	// On the head, a test file of the change that does not build, but for
	// a name the code lacks, holds no test that ran: the build is said
	// broken, its lines quoted, no side blamed (the test, the code or the
	// command), never a red test read as the change's fault (#1283:
	// pytest that cannot import the project).
	if onHead {
		if broken := brokenTests(buildErrorsIn(headRun.output, ch.tests, rules, headRun.roots...)); len(broken) > 0 {
			var lines []string
			for _, e := range broken {
				lines = append(lines, e.line)
			}
			return refused(&verdict.Verdict{Status: verdict.Block, Summary: "the change's tests do not build on the head",
				Findings: []verdict.Finding{{Rule: "build-broken", Where: broken[0].file,
					Message: fmt.Sprintf("the build broke on the head in the change's tests (%s): no test there ran, so none is proven green; "+
						"the tests, the change's code or the test command is to mend, as the lines tell",
						clipRunes(forge.Mask(sample(lines)), quoteMax))}}})
		}
	}
	if noReport != nil && len(errs) == 0 {
		// No report and no error a row reads: the tests' own to mend,
		// asked again with what the command said, never final.
		return refused(&verdict.Verdict{Status: verdict.Block, Summary: "the test command wrote no report with the change's tests, and no error a build-errors row reads",
			Findings: []verdict.Finding{*noReport}})
	}
	reader := &redFirst{repo: repo, base: ch.base, issue: s.issue,
		code: onceString(func() string {
			if !onHead {
				return ""
			}
			// The code's diff alone: a name the tests write is not one
			// the change's code touches.
			out, _ := git(repo, append([]string{"--literal-pathspecs", "diff", "--no-color", "-U0", ch.base, ch.head, "--"}, ch.other...)...)
			return changedLines(out)
		}),
		test: testFiles(repo, ch.head), tests: ch.tests}
	var dbt *doubt
	if len(errs) > 0 && !reader.fast(errs) {
		dbt = &doubt{Facts: reader.facts(errs, unrooted(redRun.output, redRun.roots), s.BuildErrors), Base: ch.base, errs: errs}
		for _, e := range errs {
			dbt.Lines = append(dbt.Lines, e.line)
			if dbt.Where == "" && e.inTest {
				dbt.Where = e.file
			}
		}
		dbt.Where = cmp.Or(dbt.Where, errs[0].file)
	}
	byName := len(errs) > 0
	// A base test the red run lacks was not built there. Beside code, the
	// red run holds every base test file (deleted ones are not laid on
	// it), so each one it lacks is unbuilt; one the red run alone names is
	// the runner's own record of the build that failed (gotestsum's
	// TestMain), never in the head's report. Tests alone, the red run is
	// the head, which lacks the tests deleted too: unbuilt are the tests
	// of a suite (a Go package) none of whose base tests ran where a new
	// red id is the build's record, not a test proven red — only when it
	// has the record's shape: the suite's one case in the head's report,
	// with no classname (a collection error) or named TestMain
	// (gotestsum's). A suite whose old tests were all deleted, and new ones
	// added, keeps its new tests counted; a new suite's record is one when
	// the build broke by a missing name.
	var added, unbuilt, records []string
	lost := map[string]bool{} // suites none of whose base tests ran
	if !onHead {
		for id := range baseRun.results {
			lost[baseRun.suite[id]] = true
		}
		for id := range redRun.results {
			if _, ok := baseRun.results[id]; ok {
				delete(lost, redRun.suite[id])
			}
		}
	}
	for id := range headRun.results {
		if _, ok := baseRun.results[id]; ok {
			continue
		}
		if !onHead && headRun.results[id] == failed && (lost[headRun.suite[id]] || byName) && headRun.buildRecord(id) {
			records = append(records, id)
			continue
		}
		added = append(added, id)
	}
	broke := map[string]bool{} // tests alone: a lost suite with no build record lost its tests to a deletion
	for _, id := range records {
		broke[headRun.suite[id]] = true
	}
	for id := range baseRun.results {
		if !onHead && !broke[baseRun.suite[id]] {
			continue
		}
		if _, ok := redRun.results[id]; !ok {
			unbuilt = append(unbuilt, id)
		}
	}
	// Tests alone: a runner's record of a suite, beside what names the
	// suite's tests, is no test (#1325).
	var suiteRecords []string
	held := heldBy(ch.tests, reader.test)
	if !onHead {
		added, suiteRecords = headRun.suiteRecords(added, byName, held)
	}
	// Tests alone, broken by a missing name: no report names the tests
	// that did not build, so their files do, as file:name.
	if !onHead && byName {
		names, err := unbuiltTests(repo, ch, headRun, errs, rules)
		if err != nil {
			return refused(refuse("the red-first check did not run", verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}))
		}
		added = append(added, names...)
	}
	slices.Sort(added)
	slices.Sort(unbuilt)
	slices.Sort(records)
	slices.Sort(suiteRecords)
	v := &verdict.Verdict{Status: verdict.Pass}
	if len(suiteRecords) > 0 {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "suite-record", Level: "info", Where: s.Report,
			Message: fmt.Sprintf("%s: red by an error, its name in none of the change's test files, beside its suite's tests: "+
				"the runner's record of the suite, not a test; not frozen", sample(suiteRecords))})
	}
	if len(unbuilt) > 0 && !byName {
		msg := fmt.Sprintf("%d of the base's tests are not in the report on the base with the change's tests (%s): their build broke there, "+
			"so a new test is red by a missing name, not by its own assertion", len(unbuilt), sample(unbuilt))
		if len(records) > 0 {
			msg += fmt.Sprintf("; %s, the runner's record of that build, is not counted as a new test", sample(records))
		}
		v.Findings = append(v.Findings, verdict.Finding{Rule: "tests-not-built", Level: "warn", Where: s.Report, Message: msg})
	}
	if len(added) == 0 {
		v.Summary = "no new test in the test report: nothing to prove red"
		if len(ch.tests) > 0 {
			v.Findings = append(v.Findings, verdict.Finding{Rule: "no-new-test", Level: "warn", Where: s.Report,
				Message: fmt.Sprintf("the change touches %d test file(s) (%s) but its report names no test the base's does not: a build that fails names none", len(ch.tests), sample(ch.tests))})
		}
		return proven{v: v, doubt: dbt, errs: errs}
	}
	missing := 0
	for _, id := range added {
		switch r, ok := redRun.results[id]; {
		case !ok:
			missing++ // not built on the base: a missing name, red
		case r == passed:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-green-on-base", Where: id,
				Message: greenOnBase})
		case r == skipped:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-green-on-base", Where: id,
				Message: skippedOnBase})
		}
		if !onHead {
			continue
		}
		switch r, ok := headRun.results[id]; {
		case !ok:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-red-on-head", Where: id, Message: "is not in the head's report: it did not run there"})
		case r == failed:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-red-on-head", Where: id, Message: "fails on the head: the change does not do what it checks"})
		case r == skipped:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-red-on-head", Where: id, Message: "is skipped on the head: a skipped test is not green"})
		}
	}
	v.Summary = fmt.Sprintf("%d new test%s, red on the base", len(added), plural(len(added)))
	if onHead {
		v.Summary += ", green on the head"
	}
	if missing > 0 {
		// Said: the fast path passed it, without the judge (ADR-0065).
		why := "a missing name the issue names"
		switch {
		case dbt != nil:
			why = "a broken build"
		case s.issue == "":
			why = "a missing name the change's code adds"
		}
		v.Summary += fmt.Sprintf(" (%d not built there, %s)", missing, why)
	}
	// The tests alone skip none of the base's: a test that ran on the
	// base and is skipped with them was changed by them, whatever lines
	// they kept (#1259: a skip inserted in a test of a file shared with
	// the base). An inserted line that leaves it running is the
	// reviewer's to judge.
	if !onHead {
		var skippedNow []string
		for id, r := range baseRun.results {
			if now, ok := redRun.results[id]; ok && r != skipped && now == skipped {
				skippedNow = append(skippedNow, id)
			}
		}
		slices.Sort(skippedNow)
		for _, id := range skippedNow {
			v.Findings = append(v.Findings, verdict.Finding{Rule: "base-test-skipped", Where: id,
				Message: "ran on the base, and is skipped with the change's tests: a test already there is never skipped nor weakened; add the new tests beside it"})
		}
	}
	for _, f := range v.Findings {
		if f.Level == "" {
			v.Status = verdict.Block
			v.Summary = fmt.Sprintf("%d new test%s, not each red on the base then green on the head", len(added), plural(len(added)))
			break
		}
	}
	// A new id the red run wrote with no class, failed, may be the
	// runner's record of a file it could not collect, not a test case: the
	// green proof hands it to the judge when it is gone (renamed.go). So
	// is a suite record (suiteRecord) kept, alone in its suite.
	var errored []string
	for _, id := range added {
		if redRun.results[id] == failed && (redRun.classless[id] || !onHead && headRun.suiteRecord(id, held)) {
			errored = append(errored, id)
		}
	}
	return proven{v: v, added: added, errored: errored, doubt: dbt, errs: errs}
}

// brokenTests is the errors in the change's test files that are no name
// the code lacks.
func brokenTests(errs []buildError) []buildError {
	return slices.DeleteFunc(errs, func(e buildError) bool { return !e.inTest || e.missing })
}

// unbuiltTests is the tests the change's test files declare, by the
// test-name of the ecosystems whose errors broke the build, that the
// head's report does not name: they did not build, as file:name. A test
// the base's own file declares is the base's, in a file the change adds
// to, never a new one (#1259): told by its file, never by a name another
// file's test may share. An ecosystem with no test-name (the generic
// pack's) names no test: each test file the change adds that holds one
// of that ecosystem's errors, no row naming a test of it nor the head's
// report, is one test, named by the file alone, which proves only that
// the file is there (#1293). A file the build did not break on (a
// helper, a fixture) is no test.
func unbuiltTests(repo string, ch *change, head ran, errs []buildError, rules []buildRules) ([]string, error) {
	used := map[int]bool{}
	broken := map[string]bool{} // the files holding an error of a row with no test-name
	for _, e := range errs {
		used[e.rules] = true
		if e.rules < len(rules) && rules[e.rules].testName == nil {
			broken[e.file] = true
		}
	}
	ran := func(name string) bool {
		for id := range head.results {
			if id == name || strings.HasSuffix(id, "."+name) {
				return true
			}
		}
		return false
	}
	var out []string
	for _, file := range ch.tests {
		content, err := git(repo, "show", ch.head+":"+file)
		if err != nil {
			return nil, err
		}
		old, err := git(repo, "show", ch.base+":"+file) // an error: a new file
		isNew := err != nil
		declares := false // a used row names a test of the file
		for i, r := range rules {
			if !used[i] || r.testName == nil {
				continue
			}
			base := testNames(old, r)
			for _, name := range testNames(content, r) {
				declares = true
				if !ran(name) && !slices.Contains(base, name) {
					out = append(out, file+":"+name)
				}
			}
		}
		if broken[file] && !declares && isNew && len(head.ofFile(file)) == 0 {
			out = append(out, file)
		}
	}
	return out, nil
}

// unset is the check's refusal when test-command or test-report is not
// set, never a pass; nil when both are.
func unset(s Settings) *verdict.Verdict {
	switch {
	case strings.TrimSpace(s.Command) == "":
		return refuse("the red-first check did not run", verdict.Finding{Rule: "setting-missing", Where: "test-command",
			Message: "no test command: set roles.tester.settings.test-command in .workline/config.yaml, the command running the project's tests " +
				"and writing a JUnit XML report to {report}, as `pytest --junitxml={report}` or `gotestsum --junitfile {report} ./...`"})
	case strings.TrimSpace(s.Report) == "":
		return refuse("the red-first check did not run", verdict.Finding{Rule: "setting-missing", Where: "test-report",
			Message: "no test report: set roles.tester.settings.test-report, where the test command writes its JUnit XML report"})
	}
	if why := reportInTree(s.Report); why != "" {
		return refuse("the red-first check did not run", verdict.Finding{Rule: "setting-invalid", Where: "test-report", Message: why})
	}
	for _, m := range s.ModuleFiles {
		if strings.ContainsAny(m, `/\`) {
			return refuse("the red-first check did not run", verdict.Finding{Rule: "setting-invalid", Where: "module-files",
				Message: fmt.Sprintf("module-files %q holds a folder: give a file's name alone, as `build.gradle.kts` or `*.csproj`, matched in each folder", m)})
		}
	}
	if _, err := compileBuildErrors(s.BuildErrors); err != nil {
		return refuse("the red-first check did not run", verdict.Finding{Rule: "setting-invalid", Where: "build-errors", Message: err.Error()})
	}
	return nil
}

func refuse(summary string, f verdict.Finding) *verdict.Verdict {
	return &verdict.Verdict{Status: verdict.Block, Summary: summary, Findings: []verdict.Finding{f}, Final: true}
}

// change is the commits from the merge base to HEAD: the test files they
// add or change (a deleted one is the base's, never laid on it), and the
// other files they touch.
type change struct {
	base, head string
	tests      []string
	other      []string
}

func changeOf(repo, base string, paths []string) (*change, error) {
	if _, err := git(repo, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		return nil, fmt.Errorf("%s is not a commit here (misspelled, or not fetched): the tests the change adds cannot be told", base)
	}
	head, err := git(repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	mb, err := git(repo, "merge-base", base, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("the merge base of %s and HEAD is not known (a shallow clone?): %v", base, err)
	}
	c := &change{base: strings.TrimSpace(mb), head: strings.TrimSpace(head)}
	out, err := git(repo, "-c", "core.quotePath=false", "diff", "--no-renames", "--name-status", "-z", "--no-relative", c.base, c.head, "--")
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		status, name := fields[i], fields[i+1]
		switch {
		case !pathglob.Any(paths, name):
			c.other = append(c.other, name)
		case status != "D":
			c.tests = append(c.tests, name)
		}
	}
	return c, nil
}

// ran is a run's tests: each one's result, and its suite, by id; the ids
// written with no classname.
type ran struct {
	output    string // the command's, stdout and stderr
	failure   string // why the command exited non-zero, "" when it exited 0
	results   map[string]result
	suite     map[string]string
	classless map[string]bool
	erred     map[string]string // the name of each case red by an error alone (readJUnit)
	roots     []string // the folder it ran in, its links resolved too: an error's absolute path below it is from the root
	reports   []string // the report files read, this run's
}

// buildRecord says whether id has the shape of a runner's record of a
// build that failed: its suite's only case, with no classname or named
// TestMain (gotestsum's).
func (r ran) buildRecord(id string) bool {
	for other, s := range r.suite {
		if other != id && s == r.suite[id] {
			return false
		}
	}
	return r.classless[id] || id == "TestMain" || strings.HasSuffix(id, ".TestMain")
}

// suiteRecord says whether id may be a runner's record of its suite, not
// a test: red by an error alone, its name, past a parameter's mark, in
// none of the change's test files (PlatformIO's `native:<suite>`, a
// collection error's module).
func (r ran) suiteRecord(id string, held func(string) bool) bool {
	name, ok := r.erred[id]
	return ok && r.results[id] == failed && !held(name)
}

// suiteRecords splits added, the new ids of the tests alone, into tests
// and a runner's records of a suite: a suiteRecord beside what names that
// suite's tests (another new case of its suite that is none, or, the
// build broken by a missing name, the tests named from their files).
// Alone, it is the only word of its tests: kept, marked errored.
func (r ran) suiteRecords(added []string, byName bool, held func(string) bool) (tests, records []string) {
	named := map[string]bool{} // suites a new case that is no record names
	for _, id := range added {
		if !r.suiteRecord(id, held) {
			named[r.suite[id]] = true
		}
	}
	for _, id := range added {
		if r.suiteRecord(id, held) && (byName || named[r.suite[id]]) {
			records = append(records, id)
			continue
		}
		tests = append(tests, id)
	}
	return tests, records
}

// heldBy says whether a test's name, cut at a parameter's mark, is in one
// of files, read by read.
func heldBy(files []string, read func(string) string) func(string) bool {
	return func(name string) bool {
		if i := strings.IndexAny(name, "[("); i > 0 {
			name = name[:i]
		}
		name = strings.TrimSpace(name)
		for _, f := range files {
			if strings.Contains(read(f), name) {
				return true
			}
		}
		return name == ""
	}
}

// run checks rev out apart, the files given laid on it as from holds them,
// runs the test command there and reads its report.
func run(repo, rev string, files []string, from string, s Settings, where string) (ran, *verdict.Finding) {
	tree, done, f := checkout(repo, rev)
	if f != nil {
		return ran{}, f
	}
	defer done()
	for _, file := range files {
		if _, err := git(tree, "-c", "core.hooksPath=/dev/null", "--literal-pathspecs", "checkout", from, "--", file); err != nil {
			return ran{}, &verdict.Finding{Rule: "check-error", Where: file, Message: fmt.Sprintf("%s could not be laid on %s: %v", file, short(rev), err)}
		}
	}
	if len(s.packages) > 0 {
		// The folders this tree holds: one the base lacks holds none of
		// its tests, and a runner named a folder that is not there fails.
		var here []string
		for _, p := range s.packages {
			if fi, err := os.Stat(filepath.Join(tree, filepath.FromSlash(p))); err == nil && fi.IsDir() {
				here = append(here, p)
			}
		}
		if len(here) == 0 {
			return ran{results: map[string]result{}, suite: map[string]string{}, classless: map[string]bool{}}, nil
		}
		s.Command = strings.ReplaceAll(s.Command, "{packages}", strings.Join(here, " "))
	}
	r, f := runIn(tree, s, where)
	r.roots = rootsOf(tree)
	if f == nil && len(files) == 0 {
		keepIDs(repo, rev, s.Command, s.Report, r)
	}
	return r, f
}

// checkout checks rev out apart; done removes the checkout.
func checkout(repo, rev string) (tree string, done func(), f *verdict.Finding) {
	tree, err := os.MkdirTemp("", "workline-test-")
	if err != nil {
		return "", nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	}
	if err := Worktree(repo, rev, tree); err != nil {
		return "", nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	}
	return tree, func() { RemoveWorktree(repo, tree) }, nil
}

// Worktree checks rev out in tree, a git worktree of repo apart, detached,
// its hooks off: where a change is laid and tried, and where the
// developer's agent edits (ADR-0056). Removed when it cannot be made.
func Worktree(repo, rev, tree string) error {
	if _, err := git(repo, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", tree, rev); err != nil {
		RemoveWorktree(repo, tree)
		return fmt.Errorf("%s could not be checked out: %v", short(rev), err)
	}
	return nil
}

// RunTree checks rev out for a run's agent, at runDir/name (an
// agent.WorktreeDir or agent.ReadTreeDir). A run folder under a .git
// folder, the default (no WORKLINE_RUNS_DIR), has its tree in a folder of
// its own outside it, its path in the run's agent.TreeAt file: Claude
// Code edits no file under .git, taking it for a sensitive one (#1304).
func RunTree(repo, rev, runDir, name string) error {
	if !underGitDir(runDir) {
		return Worktree(repo, rev, filepath.Join(runDir, name))
	}
	tree, err := os.MkdirTemp("", "workline-"+name+"-")
	if err != nil {
		return err
	}
	if err := Worktree(repo, rev, tree); err != nil {
		return err
	}
	if err := os.WriteFile(agent.TreeAt(runDir, name), []byte(tree+"\n"), 0o644); err != nil {
		RemoveWorktree(repo, tree)
		return err
	}
	return nil
}

// RemoveRunTree removes a tree RunTree made, and the file naming it.
func RemoveRunTree(repo, runDir, name, tree string) {
	RemoveWorktree(repo, tree)
	_ = os.Remove(agent.TreeAt(runDir, name))
}

// underGitDir says dir lies under a folder named .git.
func underGitDir(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	return slices.Contains(strings.Split(filepath.ToSlash(abs), "/"), ".git")
}

// RemoveWorktree removes a worktree Worktree made, whatever it holds.
func RemoveWorktree(repo, tree string) {
	if _, err := git(repo, "worktree", "remove", "--force", tree); err != nil {
		_ = os.RemoveAll(tree)
		_, _ = git(repo, "worktree", "prune")
	}
}

// exitedZero is a run's failure when it wrote no report and exited 0.
const exitedZero = "it exited 0"

// runIn runs the test command in tree and reads its report.
func runIn(tree string, s Settings, where string) (ran, *verdict.Finding) {
	if why := reportInTree(s.Report); why != "" {
		return ran{}, &verdict.Finding{Rule: "setting-invalid", Where: "test-report", Message: why}
	}
	report := filepath.Join(tree, filepath.FromSlash(s.Report))
	kept, err := clearReports(tree, s.Report) // a past run's report is not this run's
	if err != nil {
		return ran{}, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	}
	// One run on one tree stops at suite-max-seconds, whatever the
	// proof's other budgets: a test that hangs costs one suite's time,
	// not the whole proof's (#1425).
	deadline, budget := s.deadline, s.budget
	if s.SuiteMaxSeconds > 0 && !s.whole {
		if stop := time.Now().Add(time.Duration(s.SuiteMaxSeconds) * time.Second); deadline.IsZero() || stop.Before(deadline) {
			deadline, budget = stop, "suite-max-seconds"
		}
	}
	if forge.InCI() != "" {
		// A phase line in the job's log: a run that hangs says where (#1197).
		stops := ""
		if !deadline.IsZero() {
			stops = ", stopped at " + deadline.UTC().Format("15:04:05") + " UTC"
		}
		fmt.Fprintf(os.Stderr, "workline: tester: running the tests %s (%s UTC%s)\n", where, time.Now().UTC().Format("15:04:05"), stops)
	}
	out, late, runErr := shell(tree, strings.ReplaceAll(s.Command, "{report}", report), deadline)
	if late {
		f := overTime(where, budget)
		f.Message += stoppedAt(tree, s, kept, out.String())
		return ran{}, f
	}
	found := freshReports(tree, s.Report, kept)
	if len(found) == 0 {
		said := exitedZero
		if runErr != nil {
			said = runErr.Error()
		}
		// Its output kept: the compiler's lines may say why (#1192).
		return ran{output: out.String(), failure: said, results: map[string]result{}, suite: map[string]string{}, classless: map[string]bool{}},
			&verdict.Finding{Rule: "report-missing", Where: s.Report,
				Message: fmt.Sprintf("the test command wrote no report %s (%s); %s", where, said, errorLines(out.String(), s.BuildErrors, 10))}
	}
	rep := ran{output: out.String(), results: map[string]result{}, suite: map[string]string{}, classless: map[string]bool{}, erred: map[string]string{}, reports: found}
	if runErr != nil {
		rep.failure = runErr.Error()
	}
	for _, f := range found {
		if err := readJUnit(f, rep.results, rep.suite, rep.classless, rep.erred); err != nil {
			rel, _ := filepath.Rel(tree, f)
			return ran{}, &verdict.Finding{Rule: "report-missing", Where: s.Report,
				Message: fmt.Sprintf("the report %s %s is not JUnit XML: %v", rel, where, err)}
		}
	}
	return rep, nil
}

// shell runs a command line in dir, Go's result cache off, stopped past
// deadline when one is set: late says it was. Its output comes back
// masked (forge.Mask).
func shell(dir, line string, deadline time.Time) (out bytes.Buffer, late bool, err error) {
	ctx := context.Background()
	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", line)
	if !deadline.IsZero() {
		agent.InGroup(cmd) // stopped with what it started; a detached daemon lives on (#1197)
	}
	cmd.Dir = dir
	cmd.Env = uncached(forge.ProjectEnv(os.Environ()))
	cmd.Stdout, cmd.Stderr = &out, &out
	cmd.WaitDelay = 5 * time.Second
	err = cmd.Run() // a test that fails exits non-zero: the report says which
	// Its secrets masked before a reason quotes or cuts it (#1215).
	masked := bytes.NewBufferString(forge.Mask(out.String()))
	return *masked, ctx.Err() != nil, err
}

// overTime is the finding of a command stopped past the proof's budget.
func overTime(where, budget string) *verdict.Finding {
	if budget == "" {
		budget = "curator-proof-seconds"
	}
	return &verdict.Finding{Rule: "proof-over-time", Where: budget,
		Message: "the proof ran past its time budget " + where + ": stopped; no proof, nothing proposed"}
}

// stoppedAt says where a run stopped at its time was, for the agent run
// again to mend its own test (#1425): the change's test files no report
// written by then holds, when the runner had written some (one per class,
// as Gradle's and Maven's), and the command's last lines.
func stoppedAt(tree string, s Settings, kept map[string]keptFile, out string) string {
	said := ""
	if found := freshReports(tree, s.Report, kept); len(found) > 0 && len(s.tests) > 0 {
		var ids []string
		for _, f := range found {
			results, suites := map[string]result{}, map[string]string{}
			_ = readJUnit(f, results, suites, nil, nil) // one the stop cut holds nothing
			for id := range results {
				ids = append(ids, id, suites[id])
			}
		}
		var open []string
		for _, file := range s.tests {
			stem := strings.TrimSuffix(path.Base(file), path.Ext(file))
			if !slices.ContainsFunc(ids, func(id string) bool { return strings.Contains(id, stem) }) {
				open = append(open, "`"+file+"`")
			}
		}
		if len(open) > 0 {
			said = fmt.Sprintf("; of the change's tests, no report written by then holds %s: a test there did not end", strings.Join(open, ", "))
		}
	}
	last := []rune(tail(out, 15))
	if len(last) > quoteMax { // its end kept: the last line names where it hung
		last = append([]rune("…"), last[len(last)-quoteMax+1:]...)
	}
	return said + "; its last lines: " + string(last)
}

// uncached is env with Go told to run each test, never to replay a
// cached result (-count=1): a run after another in one checkout would
// pass on the first one's word. Other commands ignore GOFLAGS, and go
// commands not taking -count too; a -count the project set is kept.
func uncached(env []string) []string {
	flags := ""
	out := slices.DeleteFunc(slices.Clone(env), func(e string) bool {
		v, ok := strings.CutPrefix(e, "GOFLAGS=")
		if ok {
			flags = v
		}
		return ok
	})
	if !strings.Contains(flags, "-count") {
		flags = strings.TrimSpace(flags + " -count=1")
	}
	return append(out, "GOFLAGS="+flags)
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func sample(l []string) string {
	if len(l) <= 3 {
		return strings.Join(l, ", ")
	}
	return strings.Join(l[:3], ", ") + fmt.Sprintf(", and %d more", len(l)-3)
}

// errorLine is a line a build tool or compiler says an error on.
var errorLine = regexp.MustCompile(`(?i)^(?:e: |error\b|failure:|\S+:\d+(?::\d+)?:? (?:error|e)\b)`)

// errorLines is what a command's output says went wrong, as a reason
// quotes it: the error lines the build-errors rows read, else the lines
// saying an error, then its last n lines, a line quoted above left out
// (#1192, #1209: Gradle's last lines say only BUILD FAILED, the
// compiler's above them say why). With no error line, it says so, its
// last lines kept. Each line is cut at quoteLine runes, the whole at
// quoteMax: a forge's comment holding it has a limit.
func errorLines(out string, list []BuildErrors, n int) string {
	rules, _ := compileBuildErrors(list)
	var read, said []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		for _, r := range rules {
			if r.error.MatchString(line) {
				read = append(read, line)
				break
			}
		}
		if errorLine.MatchString(line) {
			said = append(said, line)
		}
	}
	quoted := map[string]bool{}
	quote := func(l []string) string {
		more := ""
		if len(l) > 5 {
			l, more = l[:5], fmt.Sprintf(" / and %d more", len(l)-5)
		}
		cut := make([]string, len(l))
		for i, x := range l {
			quoted[x], cut[i] = true, clipRunes(x, quoteLine)
		}
		return strings.Join(cut, " / ") + more
	}
	var says string
	switch {
	case len(read) > 0:
		says = "its errors: " + quote(read)
	case len(said) > 0:
		says = "its error lines: " + quote(said)
	default:
		says = "no error line in its output"
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	lines = lines[max(0, len(lines)-n):]
	var last []string
	for _, line := range lines {
		if !quoted[strings.TrimSpace(line)] {
			last = append(last, clipRunes(line, quoteLine))
		}
	}
	switch {
	case strings.TrimSpace(out) == "":
		says += "; its output ends: (nothing)"
	case len(last) > 0:
		says += "; its output ends: " + strings.Join(last, " / ")
	}
	return clipRunes(says, quoteMax)
}

// The most runes a reason quotes of one line of output, and in all.
const (
	quoteLine = 300
	quoteMax  = 2000
)

// clipRunes is s cut at max runes, an ellipsis saying so.
func clipRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}

// changedLines is the lines a diff adds or removes, never its headers
// (`diff --git a/x b/x`, `@@ … @@ context`): the words its code touches.
func changedLines(diff string) string {
	var out []string
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, "+++") || strings.HasPrefix(l, "---") {
			continue
		}
		if strings.HasPrefix(l, "+") || strings.HasPrefix(l, "-") {
			out = append(out, l[1:])
		}
	}
	return strings.Join(out, "\n")
}

// testFiles reads a test file of the change as head holds it, each once.
func testFiles(repo, head string) func(string) string {
	read := map[string]string{}
	return func(file string) string {
		if c, ok := read[file]; ok {
			return c
		}
		c, _ := git(repo, "show", head+":"+file)
		read[file] = c
		return c
	}
}

// onceString is f, run the first time alone.
func onceString(f func() string) func() string {
	var done bool
	var v string
	return func() string {
		if !done {
			v, done = f(), true
		}
		return v
	}
}

// tail is the last n lines of s, on one line.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	if t := strings.Join(lines, " / "); t != "" {
		return t
	}
	return "(nothing)"
}
