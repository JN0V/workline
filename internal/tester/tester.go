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
	switch {
	case strings.TrimSpace(s.Command) == "":
		return refuse("the red-first check did not run", verdict.Finding{Rule: "setting-missing", Where: "test-command",
			Message: "no test command: set roles.tester.settings.test-command in .workline/config.yaml, the command running the project's tests " +
				"and writing a JUnit XML report to {report}, as `pytest --junitxml={report}` or `gotestsum --junitfile {report} ./...`"})
	case strings.TrimSpace(s.Report) == "":
		return refuse("the red-first check did not run", verdict.Finding{Rule: "setting-missing", Where: "test-report",
			Message: "no test report: set roles.tester.settings.test-report, where the test command writes its JUnit XML report"})
	}
	ch, err := changeOf(repo, base, s.Paths)
	if err != nil {
		return refuse("the red-first check did not run", verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()})
	}
	baseRun, f := run(repo, ch.base, nil, "", s, "on the base "+short(ch.base))
	if f != nil {
		return refuse("the red-first check did not run", *f)
	}
	headRun, f := run(repo, ch.head, nil, "", s, "on the head "+short(ch.head))
	if f != nil {
		return refuse("the red-first check did not run", *f)
	}
	// Tests alone changed: the head is the base with its tests, and red
	// there is all a change holding no code yet can prove.
	redRun, onHead := headRun, len(ch.other) > 0
	if onHead {
		if redRun, f = run(repo, ch.base, ch.tests, ch.head, s, "on the base "+short(ch.base)+" with the change's tests"); f != nil {
			return refuse("the red-first check did not run", *f)
		}
	}
	// A base test the red run lacks was not built there. Beside code, the
	// red run holds every base test file (deleted ones are not laid on
	// it), so each one it lacks is unbuilt; one the red run alone names is
	// the runner's own record of the build that failed (gotestsum's
	// TestMain), never in the head's report. Tests alone, the red run is
	// the head, which lacks the tests deleted too: unbuilt are the tests
	// of a class (a package, a module) none of whose base tests ran where
	// a new red id is the build's record, not a test proven red.
	var added, unbuilt, records []string
	lost := map[string]bool{} // classes none of whose base tests ran
	if !onHead {
		for id := range baseRun.results {
			lost[baseRun.class[id]] = true
		}
		for id := range redRun.results {
			if _, ok := baseRun.results[id]; ok {
				delete(lost, redRun.class[id])
			}
		}
	}
	for id := range headRun.results {
		if _, ok := baseRun.results[id]; ok {
			continue
		}
		if !onHead && headRun.results[id] == failed && lost[headRun.class[id]] {
			records = append(records, id)
			continue
		}
		added = append(added, id)
	}
	broke := map[string]bool{} // tests alone: a lost class with no build record lost its tests to a deletion
	for _, id := range records {
		broke[headRun.class[id]] = true
	}
	for id := range baseRun.results {
		if !onHead && !broke[baseRun.class[id]] {
			continue
		}
		if _, ok := redRun.results[id]; !ok {
			unbuilt = append(unbuilt, id)
		}
	}
	slices.Sort(added)
	slices.Sort(unbuilt)
	slices.Sort(records)
	v := &verdict.Verdict{Status: verdict.Pass}
	if len(unbuilt) > 0 {
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
		return v
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
	return v
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

// ran is a run's tests: each one's result, and its classname, by id.
type ran struct {
	results map[string]result
	class   map[string]string
}

// run checks rev out apart, the files given laid on it as from holds them,
// runs the test command there and reads its report.
func run(repo, rev string, files []string, from string, s Settings, where string) (ran, *verdict.Finding) {
	tree, err := os.MkdirTemp("", "workline-test-")
	if err != nil {
		return ran{}, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	}
	defer func() {
		if _, err := git(repo, "worktree", "remove", "--force", tree); err != nil {
			_ = os.RemoveAll(tree)
			_, _ = git(repo, "worktree", "prune")
		}
	}()
	if _, err := git(repo, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", tree, rev); err != nil {
		return ran{}, &verdict.Finding{Rule: "check-error", Where: "test", Message: fmt.Sprintf("%s could not be checked out: %v", short(rev), err)}
	}
	for _, file := range files {
		if _, err := git(tree, "-c", "core.hooksPath=/dev/null", "--literal-pathspecs", "checkout", from, "--", file); err != nil {
			return ran{}, &verdict.Finding{Rule: "check-error", Where: file, Message: fmt.Sprintf("%s could not be laid on %s: %v", file, short(rev), err)}
		}
	}
	report := filepath.Join(tree, filepath.FromSlash(s.Report))
	old, _ := filepath.Glob(report)
	for _, f := range old { // a report committed is not this run's
		if err := os.Remove(f); err != nil {
			return ran{}, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
		}
	}
	cmd := exec.Command("sh", "-c", strings.ReplaceAll(s.Command, "{report}", report))
	cmd.Dir = tree
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
	rep := ran{results: map[string]result{}, class: map[string]string{}}
	for _, f := range found {
		if err := readJUnit(f, rep.results, rep.class); err != nil {
			rel, _ := filepath.Rel(tree, f)
			return ran{}, &verdict.Finding{Rule: "report-missing", Where: s.Report,
				Message: fmt.Sprintf("the report %s %s is not JUnit XML: %v", rel, where, err)}
		}
	}
	return rep, nil
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
