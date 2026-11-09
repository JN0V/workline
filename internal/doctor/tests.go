package doctor

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/role"
)

// testFiles says which files every role reads as tests (ADR-0065, change
// 3): the tester's test-paths, or, none set, the built-in generic
// patterns, and how to set them when those miss the project's tests.
func testFiles(r *Report, o Options) {
	d, err := role.TestDef(o.RolesDir, o.Repo)
	if err != nil {
		r.add(Check{Area: "repository", Rule: "test-paths-unread", Where: o.Repo, Level: Warn,
			Message: "the tester's test-paths could not be read, so every role reads the generic test files: " + err.Error()})
		return
	}
	quoted := func(l []string) string { return "`" + strings.Join(l, "`, `") + "`" }
	if d.PathsSet {
		r.add(Check{Area: "repository", Rule: "test-paths-set", Where: o.Repo, Level: OK,
			Message: fmt.Sprintf("every role reads the tester's test-paths as the test files: %s", quoted(d.Paths))})
		return
	}
	r.add(Check{Area: "repository", Rule: "test-paths-generic", Where: o.Repo, Level: Note,
		Message: fmt.Sprintf("no test-paths set: every role reads the built-in generic patterns as the test files (%s); a test they miss is code to the developer, asked of the product owner", quoted(d.Paths)),
		Fix:     "set the tester's test-paths, and test-homes where a test's code lives, in .workline/config.yaml: roles: {tester: {settings: {test-paths: [\"spec/**\"], test-homes: [{tests: \"**/src/test/**\", code: \"**/src/main/**\"}]}}}   # roles/tester/README.md"})
}
