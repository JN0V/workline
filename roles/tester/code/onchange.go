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

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/pathglob"
)

// OnChange runs the project's test command once on rev, the patches laid on
// it, in a fresh tree (fresh.go), stopped past deadline when one is set. A finding
// when it did not run whole: a setting missing, a patch that does not
// apply, no report read, or the deadline passed (rule proof-over-time).
// A scope naming `{packages}` runs on the touched folders alone.
func OnChange(repo, rev string, patches []intent.Intention, s Settings, deadline time.Time, scope contract.TestScope) (contract.TestOutcome, *verdict.Finding) {
	if v := unset(s); v != nil {
		return contract.TestOutcome{}, &v.Findings[0]
	}
	tree, at, done, f := freshTree(repo, rev)
	if f != nil {
		return contract.TestOutcome{}, f
	}
	defer done()
	if v, err := layPatches(tree, patches); err != nil {
		return contract.TestOutcome{}, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	} else if v != nil {
		return contract.TestOutcome{}, &v.Findings[0]
	}
	s.deadline = deadline
	var pkgs []string
	if strings.Contains(scope.Command, "{packages}") {
		if pkgs = Packages(tree, scope.Files, s.Paths, s.ModuleFiles, s.Packages); len(pkgs) > 0 {
			args := strings.Join(pkgs, " ")
			if run := caseGroups(scope.Files, pkgs); run != "" {
				args += " -run '" + run + "'"
			}
			s.Command = strings.ReplaceAll(scope.Command, "{packages}", args)
		}
	}
	s.Command = strings.ReplaceAll(s.Command, "{test}", scope.Test)
	var o contract.TestOutcome
	for i := 1; i <= max(1, scope.Runs); i++ {
		r, f := runIn(tree, s, "on the change")
		if f != nil {
			return o, f
		}
		o = contract.TestOutcome{Exit: r.failure, At: at, Packages: pkgs, Runs: i}
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
			// The compiler's lines first, then the tail (#1209).
			o.Exit += ": " + errorLines(r.output, s.BuildErrors, 5)
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
//
// mapping, the tester's packages setting, a glob to a package: a file the
// glob names is in that package, whatever tests lie above it (a role's
// conformance cases, `roles/*/cases/**` to `./tests/conformance`); the
// others are found as above.
//
// modules, the tester's module-files, set: the build modules instead
// (#1176), as Gradle or Maven take them: the nearest folder above each
// file holding a file one of them names (`build.gradle.kts`, `pom.xml`,
// `*.csproj`). A file of the root module (`build.gradle.kts`,
// `gradle/libs.versions.toml`) reaches every module: none, the whole
// suite runs; with a build file at the root, so does a file outside any
// other module (`docs/x.md`). With none there, such a file adds none.
func Packages(tree string, files, patterns, modules []string, mapping map[string]string) []string {
	if len(modules) > 0 {
		return buildModules(tree, files, modules)
	}
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
	globs := make([]string, 0, len(mapping))
	for g := range mapping {
		globs = append(globs, g)
	}
	slices.Sort(globs)
	var out []string
	for _, f := range files {
		if i := slices.IndexFunc(globs, func(g string) bool { return pathglob.Match(g, path.Clean(f)) }); i >= 0 {
			out = append(out, mapping[globs[i]])
			continue
		}
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

// buildModules are the folders below the root nearest each file that
// hold a file names matches by its base name, `./`-prefixed and sorted;
// none when a file's nearest is the root.
func buildModules(tree string, files, names []string) []string {
	holds := map[string]bool{}
	module := func(dir string) bool {
		if h, ok := holds[dir]; ok {
			return h
		}
		entries, _ := os.ReadDir(filepath.Join(tree, filepath.FromSlash(dir)))
		holds[dir] = slices.ContainsFunc(entries, func(e os.DirEntry) bool {
			return !e.IsDir() && slices.ContainsFunc(names, func(n string) bool {
				ok, _ := path.Match(n, e.Name())
				return ok
			})
		})
		return holds[dir]
	}
	var out []string
	for _, f := range files {
		for dir := path.Dir(path.Clean(f)); ; dir = path.Dir(dir) {
			if dir == "." || dir == "/" {
				if module(".") {
					return nil // the root build reaches every module: the whole suite
				}
				break
			}
			if module(dir) {
				out = append(out, pkgArg(dir))
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

// conformancePkg and conformanceTest: the folder holding the conformance
// cases (`cases/<group>/<case>.yaml`) and the test that runs each as a
// subtest named `<group>/<case>`.
const (
	conformancePkg  = "tests/conformance"
	conformanceTest = "TestConformance"
)

// caseGroups is the `-run` pattern for the conformance groups the files
// touch, "" unless every file is a case of the one package run: a change
// to anything else runs the package whole (the test job runs all cases).
func caseGroups(files, pkgs []string) string {
	if len(pkgs) != 1 || pkgs[0] != "./"+conformancePkg || len(files) == 0 {
		return ""
	}
	var groups []string
	for _, f := range files {
		rest, ok := strings.CutPrefix(path.Clean(f), conformancePkg+"/cases/")
		group, name, found := strings.Cut(rest, "/")
		if !ok || !found || name == "" || group == "" || !validGroup(group) {
			return ""
		}
		groups = append(groups, group)
	}
	slices.Sort(groups)
	groups = slices.Compact(groups)
	return "^" + conformanceTest + "$/^(" + strings.Join(groups, "|") + ")$"
}

// validGroup: a folder name safe in a regexp and a shell quote.
func validGroup(g string) bool {
	for _, r := range g {
		if !groupRune(r) {
			return false
		}
	}
	return true
}

// groupRune: a letter, a digit, '-' or '_'.
func groupRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_'
}
