package doctor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/testdef"
)

// testsFix is how a project says which files are its tests.
const testsFix = "set the tester's test-paths, and test-homes where a test's code lives, in .workline/config.yaml: roles: {tester: {settings: {test-paths: [\"spec/**\"], test-homes: [{tests: \"**/src/test/**\", code: \"**/src/main/**\"}]}}}   # roles/tester/README.md"

// testFiles says which files every role reads as tests (ADR-0065, change
// 3): the tester's test-paths, or, none set, the built-in generic
// patterns, and how to set them when those miss the project's tests.
func testFiles(r *Report, o Options, cfg *role.ProjectConfig) {
	quoted := func(l []string) string { return "`" + strings.Join(l, "`, `") + "`" }
	if _, set := cfg.Roles["reviewer"].Settings["tests"]; set {
		r.add(Check{Area: "repository", Rule: "reviewer-tests-gone", Where: ".workline/config.yaml", Level: Warn,
			Message: "the reviewer's `tests` setting is gone, read by nothing: every role reads the tester's test-paths as the test files",
			Fix:     testsFix})
	}
	d, err := role.TestDef(o.RolesDir, o.Repo)
	switch {
	case errors.Is(err, testdef.ErrHomes):
		r.add(Check{Area: "repository", Rule: "test-homes-invalid", Where: ".workline/config.yaml", Level: Warn,
			Message: fmt.Sprintf("the tester's %v: every role reads the generic homes in their place", err), Fix: testsFix})
	case err != nil:
		r.add(Check{Area: "repository", Rule: "test-paths-unread", Where: o.Repo, Level: Warn,
			Message: fmt.Sprintf("the tester's settings could not be read whole (%v): every role reads as the test files %s", err, quoted(d.Patterns()))})
		return
	}
	if d.PathsSet {
		r.add(Check{Area: "repository", Rule: "test-paths-set", Where: o.Repo, Level: OK,
			Message: fmt.Sprintf("every role reads the tester's test-paths as the test files: %s", quoted(d.Paths))})
		return
	}
	r.add(Check{Area: "repository", Rule: "test-paths-generic", Where: o.Repo, Level: Note,
		Message: fmt.Sprintf("no test-paths set: every role reads the built-in generic patterns as the test files (%s); a test they miss is code to the developer, asked of the product owner", quoted(d.Paths)),
		Fix:     testsFix})
}
