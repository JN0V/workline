// Package tester holds the tester's checks that need no agent (ADR-0055):
// the red-first check, which proves from the project's own test runner's
// JUnit XML report that each test a change adds fails on its base.
package tester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/verdict"
)

// Settings are the tester's project settings the check reads
// (roles/tester/role.yaml).
type Settings struct {
	// Command runs the project's tests and writes a JUnit XML report;
	// {report} in it is the report's path.
	Command string `json:"test-command"`
	// Report is where the command writes the report, from the
	// repository's root: a path, or a pattern one folder deep (*.xml).
	Report string `json:"test-report"`
	// Paths are the test files, as patterns (`**` spans folders).
	Paths []string `json:"test-paths"`
	// CodeLinesMax caps the lines of code and tests given to the agent
	// when it writes an issue's tests, all files together.
	CodeLinesMax int `json:"code-lines-max"`
	// Exchanges bounds an objection's thread: the tester's answers in it
	// before a person decides (ADR-0055); Person, who is then asked: a
	// label put on the merge request, or an @login mentioned.
	Exchanges int    `json:"objection-exchanges"`
	Person    string `json:"objection-person"`
	// BuildErrors tell, by ecosystem, a build a new test broke by a name
	// the code lacks from one broken otherwise.
	BuildErrors []BuildErrors `json:"build-errors"`
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
	return s, nil
}

// Check is the red-first check on the change from the merge base of base
// and HEAD to HEAD, as committed: the tests HEAD's report holds that the
// base's does not are the new ones; each must fail on the base with the
// change's test files laid on it, and, when the change touches more than
// tests, pass on HEAD. No report read is never a pass.
func Check(repo, base string, s Settings) *verdict.Verdict {
	v, _ := Prove(repo, base, s)
	return v
}

// Prove is Check, with the new tests it found, sorted.
func Prove(repo, base string, s Settings) (*verdict.Verdict, []string) {
	if v := unset(s); v != nil {
		return v, nil
	}
	ch, err := changeOf(repo, base, s.Paths)
	if err != nil {
		return refuse("the red-first check did not run", verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}), nil
	}
	baseRun, f := run(repo, ch.base, nil, "", s, "on the base "+short(ch.base))
	if f != nil {
		return refuse("the red-first check did not run", *f), nil
	}
	headRun, f := run(repo, ch.head, nil, "", s, "on the head "+short(ch.head))
	if f != nil {
		return refuse("the red-first check did not run", *f), nil
	}
	// Tests alone changed: the head is the base with its tests, and red
	// there is all a change holding no code yet can prove.
	redRun, onHead := headRun, len(ch.other) > 0
	if onHead {
		if redRun, f = run(repo, ch.base, ch.tests, ch.head, s, "on the base "+short(ch.base)+" with the change's tests"); f != nil {
			return refuse("the red-first check did not run", *f), nil
		}
	}
	// The build where the change's tests were laid on the base: an error
	// it printed in a test of the change, naming what the base lacks, is
	// a missing name, red for the right reason; any other error is a
	// build broken otherwise, never a proof (build-errors).
	rules, _ := compileBuildErrors(s.BuildErrors) // unset said a bad one
	errs := buildErrorsIn(redRun.output, ch.tests, rules)
	var broken []buildError
	for _, e := range errs {
		if !e.missing {
			broken = append(broken, e)
		}
	}
	if len(broken) > 0 {
		lines := make([]string, len(broken))
		for i, e := range broken {
			lines[i] = e.line
		}
		return &verdict.Verdict{Status: verdict.Block, Summary: "the build broke on the base with the change's tests, not by a missing name",
			Findings: []verdict.Finding{{Rule: "build-broken", Where: broken[0].file,
				Message: fmt.Sprintf("%d build error%s not a name the base lacks in a test of the change (%s): a test must fail by its assertion or a missing name, "+
					"never a syntax error, a wrong import or code that does not build", len(broken), plural(len(broken)), sample(lines))}}}, nil
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
	// Tests alone, broken by a missing name: no report names the tests
	// that did not build, so their files do, as file:name.
	if !onHead && byName {
		names, err := unbuiltTests(repo, ch, headRun, errs, rules)
		if err != nil {
			return refuse("the red-first check did not run", verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}), nil
		}
		added = append(added, names...)
	}
	slices.Sort(added)
	slices.Sort(unbuilt)
	slices.Sort(records)
	v := &verdict.Verdict{Status: verdict.Pass}
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
		return v, nil
	}
	missing := 0
	for _, id := range added {
		switch r, ok := redRun.results[id]; {
		case !ok:
			missing++ // not built on the base: a missing name, red
		case r == passed:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-green-on-base", Where: id,
				Message: "passes on the base: it proves nothing the base does not already do; a new test must fail there first"})
		case r == skipped:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-green-on-base", Where: id,
				Message: "is skipped on the base: a skipped test is not red; a new test must fail there first"})
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
		v.Summary += fmt.Sprintf(" (%d not built there, a missing name)", missing)
	}
	for _, f := range v.Findings {
		if f.Level == "" {
			v.Status = verdict.Block
			v.Summary = fmt.Sprintf("%d new test%s, not each red on the base then green on the head", len(added), plural(len(added)))
			break
		}
	}
	return v, added
}

// unbuiltTests is the tests the change's test files declare, by the
// test-name of the ecosystems whose errors broke the build, that the
// head's report does not name: they did not build, as file:name.
func unbuiltTests(repo string, ch *change, head ran, errs []buildError, rules []buildRules) ([]string, error) {
	used := map[int]bool{}
	for _, e := range errs {
		used[e.rules] = true
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
		for i, r := range rules {
			if !used[i] {
				continue
			}
			for _, name := range testNames(content, r) {
				if !ran(name) {
					out = append(out, file+":"+name)
				}
			}
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
	results   map[string]result
	suite     map[string]string
	classless map[string]bool
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
	return runIn(tree, s, where)
}

// checkout checks rev out apart; done removes the checkout.
func checkout(repo, rev string) (tree string, done func(), f *verdict.Finding) {
	tree, err := os.MkdirTemp("", "workline-test-")
	if err != nil {
		return "", nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	}
	done = func() {
		if _, err := git(repo, "worktree", "remove", "--force", tree); err != nil {
			_ = os.RemoveAll(tree)
			_, _ = git(repo, "worktree", "prune")
		}
	}
	if _, err := git(repo, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", tree, rev); err != nil {
		done()
		return "", nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: fmt.Sprintf("%s could not be checked out: %v", short(rev), err)}
	}
	return tree, done, nil
}

// runIn runs the test command in tree and reads its report.
func runIn(tree string, s Settings, where string) (ran, *verdict.Finding) {
	report := filepath.Join(tree, filepath.FromSlash(s.Report))
	old, _ := filepath.Glob(report)
	for _, f := range old { // a report committed is not this run's
		if err := os.Remove(f); err != nil {
			return ran{}, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
		}
	}
	cmd := exec.Command("sh", "-c", strings.ReplaceAll(s.Command, "{report}", report))
	cmd.Dir = tree
	cmd.Env = uncached(os.Environ())
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	runErr := cmd.Run() // a test that fails exits non-zero: the report says which
	found, _ := filepath.Glob(report)
	if len(found) == 0 {
		said := "it exited 0"
		if runErr != nil {
			said = runErr.Error()
		}
		return ran{}, &verdict.Finding{Rule: "report-missing", Where: s.Report,
			Message: fmt.Sprintf("the test command wrote no report %s (%s); its output ends: %s", where, said, tail(out.String(), 10))}
	}
	rep := ran{output: out.String(), results: map[string]result{}, suite: map[string]string{}, classless: map[string]bool{}}
	for _, f := range found {
		if err := readJUnit(f, rep.results, rep.suite, rep.classless); err != nil {
			rel, _ := filepath.Rel(tree, f)
			return ran{}, &verdict.Finding{Rule: "report-missing", Where: s.Report,
				Message: fmt.Sprintf("the report %s %s is not JUnit XML: %v", rel, where, err)}
		}
	}
	return rep, nil
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
