package contract

import (
	"slices"
	"strings"
)

// TestScope narrows a run to the folders a change touches (#703): Command,
// holding `{packages}`, runs on the folders of Files that hold tests,
// as a Draft's proof does. No Command naming `{packages}`,
// or no such folder: the test command runs whole.
//
// Test and Runs: a flaky-n-of-n proof (ADR-0059, decision 4), the
// command run Runs times on the one tree, `{test}` in it the test's name;
// it stops at the first run not green or not passing Test.
type TestScope struct {
	Command string
	Files   []string
	Test    string
	Runs    int
}

// TestOutcome is one run of the project's test command: the tests it reported
// passed and red (failed, an error, or skipped), and why the command
// exited non-zero, "" when it exited 0; At, the commit its fresh tree
// checked out.
type TestOutcome struct {
	Passed []string
	Red    []string
	Exit   string
	At     string
	// Packages: the folders the scoped command ran on, `./`-prefixed;
	// none when the test command ran whole.
	Packages []string
	// Runs: the runs made, the last one's tests above; 1 but for a
	// Scope's Runs.
	Runs int
	// Missing: the tools its output says were missing, when not green
	// (#760).
	Missing []string
}

// Green says the run proves the change: tests reported, none red, the
// command exited 0.
func (o TestOutcome) Green() bool {
	return len(o.Passed) > 0 && len(o.Red) == 0 && o.Exit == ""
}

// Passes says whether the run's report holds test, and passed: its id
// whole, or its last part after a `.` or a `/` (`TestX` for
// `pkg/x.TestX`).
func (o TestOutcome) Passes(test string) (reported, passed bool) {
	is := func(id string) bool {
		return id == test || strings.HasSuffix(id, "."+test) || strings.HasSuffix(id, "/"+test)
	}
	if slices.ContainsFunc(o.Red, is) {
		return true, false
	}
	ok := slices.ContainsFunc(o.Passed, is)
	return ok, ok
}
