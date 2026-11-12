package doctor

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/packs"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/testdef"
	"github.com/JN0V/workline/internal/tester"
)

// testCommandRuns runs the test command set on HEAD, apart, a minute at
// most: one that cannot run here, a tool or an SDK missing, is said with
// its error lines, before a tester's agent spends tokens to find it out
// (#1333). One still running at the end is taken to run.
func testCommandRuns(r *Report, o Options, merged map[string]any) {
	s, err := tester.SettingsFrom(merged)
	if err != nil {
		return // the tester's own load says it, where it runs
	}
	head, err := exec.Command("git", "-C", o.Repo, "rev-parse", "--verify", "--quiet", "HEAD^{commit}").Output()
	if err != nil {
		return // no commit yet: nothing to run it on
	}
	f := tester.Runs(o.Repo, strings.TrimSpace(string(head)), s, tester.RunsWithin)
	if f == nil {
		r.add(Check{Area: "repository", Rule: "test-command-runs", Where: "tester", Level: OK,
			Message: "the test command runs on HEAD, a minute at most watched: `" + s.Command + "`"})
		return
	}
	r.add(Check{Area: "repository", Rule: f.Rule, Where: "tester", Level: Warn,
		Message: fmt.Sprintf("the test command `%s` does not run on HEAD: the tester stops before its agent, the developer's code stays an unproven Draft; %s", s.Command, f.Message),
		Fix:     "run it in a fresh clone, mend what it lacks (a tool, an SDK, a variable a gitignored file held) or the test-command setting in .workline/config.yaml   # roles/tester/README.md#settings"})
}

// testsFix is how a project says which files are its tests.
const testsFix = "set the tester's test-paths, and test-homes where a test's code lives, in .workline/config.yaml: roles: {tester: {settings: {test-paths: [\"spec/**\"], test-homes: [{tests: \"**/src/test/**\", code: \"**/src/main/**\"}]}}}   # roles/tester/README.md"

// testFiles says which files every role reads as tests (ADR-0065, change
// 3): the tester's test-paths, or, none set, its listed packs' then the
// built-in generic patterns, and how to set them when those miss the
// project's tests.
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
	if _, own := cfg.Roles["tester"].Settings[testdef.PathsSetting]; !own && d.PathsSet {
		r.add(Check{Area: "repository", Rule: "test-paths-packs", Where: o.Repo, Level: OK,
			Message: fmt.Sprintf("every role reads the test paths of the packs listed (%s), then the generic pack's, as the test files: %s; the tester's test-paths, set, would win", strings.Join(cfg.Packs, ", "), quoted(d.Paths))})
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

// testCommand says a tester or developer routed with no test-command: the
// tester's check refuses each run, the developer's code stays a Draft
// unproven; said before the first run, with the setting (#1284), the
// listed packs' suggested test-command where they give one (ADR-0065).
func testCommand(r *Report, o Options, cfg *role.ProjectConfig, line *routing.Config) {
	var routed []string
	for _, steps := range line.Events {
		for _, name := range []string{"tester", "developer"} {
			if slices.Contains(steps, name) && !slices.Contains(routed, name) {
				routed = append(routed, name)
			}
		}
	}
	if len(routed) == 0 {
		return
	}
	rl, err := role.Load(o.RolesDir, "tester")
	if err != nil {
		return // the role's own load says it, where it runs
	}
	merged := rl.MergedSettings(cfg)
	if c, _ := merged["test-command"].(string); strings.TrimSpace(c) != "" {
		testCommandRuns(r, o, merged)
		return
	}
	slices.Sort(routed)
	examples := "as `pytest --junitxml={report}` or `gotestsum --junitfile {report} ./...`"
	listed, _ := packs.Listed(o.Repo, cfg.Packs) // read whole by LoadProjectConfig already
	var suggested []string
	for _, p := range listed {
		if p.TestCommand != "" && !slices.Contains(suggested, p.TestCommand) {
			suggested = append(suggested, p.TestCommand)
		}
	}
	if len(suggested) > 0 {
		examples = "as the packs listed suggest: `" + strings.Join(suggested, "` or `") + "`"
	}
	r.add(Check{Area: "repository", Rule: "test-command-missing", Where: "tester", Level: Warn,
		Message: fmt.Sprintf("no test-command is set (the tester's, which the developer's proof runs too), and %s routed: the tester refuses each run, the developer's code stays an unproven Draft",
			map[bool]string{true: "the " + routed[0] + " is", false: "the developer and the tester are"}[len(routed) == 1]),
		Fix: "set the command running the project's tests in its own language and writing a JUnit XML report to {report}, in .workline/config.yaml, " +
			examples + ": " +
			"roles: {tester: {settings: {test-command: \"<command>\"}}}   # roles/tester/README.md#settings"})
}
