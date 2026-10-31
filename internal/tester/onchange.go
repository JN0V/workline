package tester

// The project's tests on a change not committed yet: what the developer
// proves its code with before its pull request opens (ADR-0054, step 3),
// the same test command and report as the tester's checks.

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/verdict"
)

// Scope narrows a run to the folders a change touches (#703): Command,
// holding `{packages}`, runs on the folders of Files that hold tests,
// as the curator's proof command does. No Command naming `{packages}`,
// or no such folder: the test command runs whole.
//
// Test and Runs: a flaky-n-of-n proof (ADR-0059, decision 4), the
// command run Runs times on the one tree, `{test}` in it the test's name;
// it stops at the first run not green or not passing Test.
type Scope struct {
	Command string
	Files   []string
	Test    string
	Runs    int
}

// Outcome is one run of the project's test command: the tests it reported
// passed and red (failed, an error, or skipped), and why the command
// exited non-zero, "" when it exited 0; At, the commit its fresh tree
// checked out.
type Outcome struct {
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
func (o Outcome) Green() bool { return len(o.Passed) > 0 && len(o.Red) == 0 && o.Exit == "" }

// Passes says whether the run's report holds test, and passed: its id
// whole, or its last part after a `.` or a `/` (`TestX` for
// `pkg/x.TestX`).
func (o Outcome) Passes(test string) (reported, passed bool) {
	is := func(id string) bool {
		return id == test || strings.HasSuffix(id, "."+test) || strings.HasSuffix(id, "/"+test)
	}
	if slices.ContainsFunc(o.Red, is) {
		return true, false
	}
	ok := slices.ContainsFunc(o.Passed, is)
	return ok, ok
}

// OnChange runs the project's test command once on rev, the patches laid on
// it, in a fresh tree (fresh.go), stopped past deadline when one is set. A finding
// when it did not run whole: a setting missing, a patch that does not
// apply, no report read, or the deadline passed (rule proof-over-time).
// A scope naming `{packages}` runs on the touched folders alone.
func OnChange(repo, rev string, patches []intent.Intention, s Settings, deadline time.Time, scope Scope) (Outcome, *verdict.Finding) {
	if v := unset(s); v != nil {
		return Outcome{}, &v.Findings[0]
	}
	tree, at, done, f := freshTree(repo, rev)
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
	var pkgs []string
	if strings.Contains(scope.Command, "{packages}") {
		if pkgs = Packages(tree, scope.Files, s.Paths); len(pkgs) > 0 {
			s.Command = strings.ReplaceAll(scope.Command, "{packages}", strings.Join(pkgs, " "))
		}
	}
	s.Command = strings.ReplaceAll(s.Command, "{test}", scope.Test)
	var o Outcome
	for i := 1; i <= max(1, scope.Runs); i++ {
		r, f := runIn(tree, s, "on the change")
		if f != nil {
			return o, f
		}
		o = Outcome{Exit: r.failure, At: at, Packages: pkgs, Runs: i}
		for id, res := range r.results {
			if res == passed {
				o.Passed = append(o.Passed, id)
			} else {
				o.Red = append(o.Red, id)
			}
		}
		slices.Sort(o.Passed)
		slices.Sort(o.Red)
		if !o.Green() {
			o.Missing = MissingTools(r.output)
		}
		if o.Exit != "" {
			o.Exit += ": " + strings.TrimSpace(tail(r.output, 5))
		}
		if _, ok := o.Passes(scope.Test); !o.Green() || scope.Test != "" && !ok {
			break
		}
	}
	return o, nil
}

// Packages are the folders of files that hold tests, `./`-prefixed and
// sorted: each file's own folder, else the nearest above it holding a
// file a test pattern names (`**/*_test.go`; not `tests/**`, which names
// a folder). A file with none above it adds none: no test to run there.
func Packages(tree string, files, patterns []string) []string {
	var named []string
	for _, p := range patterns {
		if path.Base(p) != "**" {
			named = append(named, p)
		}
	}
	holds := map[string]bool{}
	tests := func(dir string) bool {
		if h, ok := holds[dir]; ok {
			return h
		}
		entries, _ := os.ReadDir(filepath.Join(tree, filepath.FromSlash(dir)))
		for _, e := range entries {
			if !e.IsDir() && pathglob.Any(named, path.Join(dir, e.Name())) {
				holds[dir] = true
				return true
			}
		}
		holds[dir] = false
		return false
	}
	var out []string
	for _, f := range files {
		for dir := path.Dir(path.Clean(f)); ; dir = path.Dir(dir) {
			if tests(dir) {
				out = append(out, pkgArg(dir))
				break
			}
			if dir == "." || dir == "/" {
				break
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// pkgArg is a folder as a test command takes it: `./internal/x`, `.`.
func pkgArg(dir string) string {
	if dir == "." {
		return "."
	}
	return "./" + dir
}
