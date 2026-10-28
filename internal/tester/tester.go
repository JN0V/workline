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
	// The new tests are the head's: one the red run alone names is the
	// runner's own record of a build that failed there (gotestsum's
	// TestMain), not a test.
	var added, unbuilt []string
	for id := range headRun {
		if _, ok := baseRun[id]; !ok {
			added = append(added, id)
		}
	}
	for id := range baseRun {
		if _, ok := redRun[id]; !ok {
			unbuilt = append(unbuilt, id)
		}
	}
	slices.Sort(added)
	slices.Sort(unbuilt)
	v := &verdict.Verdict{Status: verdict.Pass}
	if len(unbuilt) > 0 {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "tests-not-built", Level: "warn", Where: s.Report,
			Message: fmt.Sprintf("%d of the base's tests are not in the report on the base with the change's tests (%s): their build broke there, "+
				"so a new test is red by a missing name, not by its own assertion", len(unbuilt), sample(unbuilt))})
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
		switch r, ok := redRun[id]; {
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
		switch r, ok := headRun[id]; {
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
// change, deleted ones too, and whether they change anything else.
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
	names, err := git(repo, "-c", "core.quotePath=false", "diff", "--no-renames", "--name-only", "-z", "--no-relative", c.base, c.head, "--")
	if err != nil {
		return nil, err
	}
	for _, name := range strings.Split(strings.TrimSuffix(names, "\x00"), "\x00") {
		switch {
		case name == "":
		case pathglob.Any(paths, name):
			c.tests = append(c.tests, name)
		default:
			c.other = append(c.other, name)
		}
	}
	return c, nil
}

// run checks rev out apart, the files given laid on it as from holds them
// (gone when it holds none), runs the test command there and reads its
// report: each test's result by its id.
func run(repo, rev string, files []string, from string, s Settings, where string) (map[string]result, *verdict.Finding) {
	tree, err := os.MkdirTemp("", "workline-test-")
	if err != nil {
		return nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	}
	defer func() {
		if _, err := git(repo, "worktree", "remove", "--force", tree); err != nil {
			_ = os.RemoveAll(tree)
			_, _ = git(repo, "worktree", "prune")
		}
	}()
	if _, err := git(repo, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", tree, rev); err != nil {
		return nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: fmt.Sprintf("%s could not be checked out: %v", short(rev), err)}
	}
	for _, file := range files {
		if _, err := git(tree, "-c", "core.hooksPath=/dev/null", "--literal-pathspecs", "checkout", from, "--", file); err != nil {
			_ = os.Remove(filepath.Join(tree, filepath.FromSlash(file))) // gone on the head
		}
	}
	report := filepath.Join(tree, filepath.FromSlash(s.Report))
	old, _ := filepath.Glob(report)
	for _, f := range old { // a report committed is not this run's
		if err := os.Remove(f); err != nil {
			return nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
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
		return nil, &verdict.Finding{Rule: "report-missing", Where: s.Report,
			Message: fmt.Sprintf("the test command wrote no report %s (%s); its output ends: %s", where, said, tail(out.String(), 10))}
	}
	results := map[string]result{}
	for _, f := range found {
		if err := readJUnit(f, results); err != nil {
			rel, _ := filepath.Rel(tree, f)
			return nil, &verdict.Finding{Rule: "report-missing", Where: s.Report,
				Message: fmt.Sprintf("the report %s %s is not JUnit XML: %v", rel, where, err)}
		}
	}
	return results, nil
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
