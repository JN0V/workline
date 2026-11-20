package tester

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A report pattern with `**` names files folders deep, as Gradle and
// Maven write one per module (#1175); `.git` is never walked; a path or
// a pattern one folder deep reads as before.
func TestReportFilesSpanFolders(t *testing.T) {
	tree := t.TempDir()
	for _, f := range []string{
		"app/build/test-results/testDebugUnitTest/TEST-a.xml",
		"core/session/build/test-results/testDebugUnitTest/TEST-b.xml",
		"core/session/build/test-results/testDebugUnitTest/binary/output.bin",
		".git/build/test-results/x/TEST-c.xml",
		"target/surefire-reports/TEST-d.xml",
		"junit.xml",
	} {
		p := filepath.Join(tree, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rel := func(pattern string) []string {
		var out []string
		for _, f := range reportFiles(tree, pattern) {
			r, _ := filepath.Rel(tree, f)
			out = append(out, filepath.ToSlash(r))
		}
		return out
	}
	for pattern, want := range map[string][]string{
		"**/build/test-results/**/*.xml": {"app/build/test-results/testDebugUnitTest/TEST-a.xml", "core/session/build/test-results/testDebugUnitTest/TEST-b.xml"},
		"core/**/TEST-*.xml":             {"core/session/build/test-results/testDebugUnitTest/TEST-b.xml"},
		"**/surefire-reports/*.xml":      {"target/surefire-reports/TEST-d.xml"},
		"target/surefire-reports/*.xml":  {"target/surefire-reports/TEST-d.xml"},
		"junit.xml":                      {"junit.xml"},
		"nowhere/**/*.xml":               nil,
	} {
		if got := rel(pattern); !slices.Equal(got, want) {
			t.Errorf("reportFiles(%q) = %v, want %v", pattern, got, want)
		}
	}
}

// reportTree is a git repository holding files, those in tracked
// committed, the others left untracked.
func reportTree(t *testing.T, tracked, untracked []string) string {
	t.Helper()
	tree := t.TempDir()
	gitIn := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", tree, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir(), "GIT_CONFIG_NOSYSTEM=1"}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	gitIn("init", "-q")
	write := func(f, body string) {
		p := filepath.Join(tree, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range tracked {
		write(f, "<project/>\n")
	}
	gitIn("add", ".")
	gitIn("commit", "-qm", "init")
	for _, f := range untracked {
		write(f, `<testsuite name="stale"><testcase classname="stale" name="TestGone" time="9"><failure/></testcase></testsuite>`)
	}
	return tree
}

// A pattern reaching outside the run's tree names nothing there and is
// refused before any command runs: a report pattern `../**/*.xml` once
// walked, and deleted, files above the worktree.
func TestReportPatternOutsideTreeRefused(t *testing.T) {
	parent := t.TempDir()
	tree := filepath.Join(parent, "tree")
	outside := filepath.Join(parent, "keep", "TEST-x.xml")
	for _, p := range []string{filepath.Join(tree, "junit.xml"), outside} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, pattern := range []string{"../**/*.xml", "../keep/*.xml", "/tmp/**/*.xml", "a/../../**/*.xml"} {
		if got := reportFiles(tree, pattern); got != nil {
			t.Errorf("reportFiles(%q) = %v, want nothing outside the tree", pattern, got)
		}
		v := unset(Settings{Command: "true", Report: pattern})
		if v == nil || len(v.Findings) != 1 || v.Findings[0].Rule != "setting-invalid" || v.Findings[0].Where != "test-report" {
			t.Errorf("unset(test-report %q) = %+v, want a setting-invalid finding on test-report", pattern, v)
		}
		if _, f := runIn(tree, Settings{Command: "true", Report: pattern}, "here"); f == nil || f.Rule != "setting-invalid" {
			t.Errorf("runIn(test-report %q) = %+v, want setting-invalid", pattern, f)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("a file outside the tree was removed: %v", err)
	}
	if v := unset(Settings{Command: "true", Report: "**/build/test-results/**/*.xml"}); v != nil {
		t.Errorf("a pattern inside the tree refused: %+v", v)
	}
}

// A module-files entry is a file's name, matched against each folder's
// files: one holding a `/` never matches, so it is refused.
func TestModuleFilesWithFolderRefused(t *testing.T) {
	v := unset(Settings{Command: "true", Report: "junit.xml", ModuleFiles: []string{"build.gradle.kts", "app/build.gradle"}})
	if v == nil || len(v.Findings) != 1 || v.Findings[0].Where != "module-files" || !strings.Contains(v.Findings[0].Message, "app/build.gradle") {
		t.Errorf("unset = %+v, want a setting-invalid finding on module-files naming app/build.gradle", v)
	}
	if v := unset(Settings{Command: "true", Report: "junit.xml", ModuleFiles: []string{"build.gradle.kts", "*.csproj"}}); v != nil {
		t.Errorf("names refused: %+v", v)
	}
}

// Before a run, the reports a past run left are removed; a file git
// tracks that the pattern names (`pom.xml` under `**/*.xml`) is kept,
// and, unchanged by the run or rewritten into no report, not read. A
// file in a nested repository (a folder holding `.git`) is kept too:
// the root's git does not list it.
func TestStaleReportsRemovedTrackedKept(t *testing.T) {
	tree := reportTree(t,
		[]string{"pom.xml", "app/src/main/AndroidManifest.xml", "app/build/test-results/old/TEST-committed.xml"},
		[]string{"app/build/test-results/old/TEST-stale.xml", "lib/.git/build/test-results/x/TEST-nested.xml", "lib/pom.xml"})
	s := Settings{Report: "**/*.xml",
		Command: `printf '<project v="2"/>' > pom.xml && mkdir -p app/build/test-results/test && printf '<testsuite name="a" time="1.5"><testcase classname="a" name="TestNew" time="1.5"/></testsuite>' > app/build/test-results/test/TEST-new.xml`}
	r, f := runIn(tree, s, "here")
	if f != nil {
		t.Fatalf("runIn: %+v", f)
	}
	for _, kept := range []string{"pom.xml", "app/src/main/AndroidManifest.xml", "app/build/test-results/old/TEST-committed.xml",
		"lib/pom.xml", "lib/.git/build/test-results/x/TEST-nested.xml"} {
		if _, err := os.Stat(filepath.Join(tree, filepath.FromSlash(kept))); err != nil {
			t.Errorf("%s removed: %v", kept, err)
		}
	}
	for _, gone := range []string{"app/build/test-results/old/TEST-stale.xml"} {
		if _, err := os.Stat(filepath.Join(tree, filepath.FromSlash(gone))); !os.IsNotExist(err) {
			t.Errorf("stale report %s kept: %v", gone, err)
		}
	}
	if len(r.results) != 1 || r.results["a.TestNew"] != passed {
		t.Errorf("results = %v, want only a.TestNew passed", r.results)
	}
}

// A folder in the tree linked outside it, committed or made by the
// build, never leads the removal out: the files out there stay.
func TestStaleReportsNotRemovedThroughLink(t *testing.T) {
	outside := t.TempDir()
	files := []string{"a.xml", "sub/x/b.xml", "sub/y/c.xml"}
	for _, f := range files {
		p := filepath.Join(outside, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("<testsuite/>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, pattern := range []string{"out/*.xml", "out/sub/**/*.xml"} {
		tree := reportTree(t, []string{"pom.xml"}, nil)
		if err := os.Symlink(outside, filepath.Join(tree, "out")); err != nil {
			t.Fatal(err)
		}
		_, _ = runIn(tree, Settings{Report: pattern, Command: "true"}, "here")
		for _, f := range files {
			if _, err := os.Stat(filepath.Join(outside, filepath.FromSlash(f))); err != nil {
				t.Errorf("test-report %q removed %s outside the tree through a link: %v", pattern, f, err)
			}
		}
	}
}

// A tracked report the command writes again is this run's: read.
func TestTrackedReportRewrittenRead(t *testing.T) {
	tree := reportTree(t, []string{"junit.xml"}, nil)
	s := Settings{Report: "junit.xml",
		Command: `printf '<testsuite name="a"><testcase classname="a" name="TestNew"/></testsuite>' > {report}`}
	r, f := runIn(tree, s, "here")
	if f != nil {
		t.Fatalf("runIn: %+v", f)
	}
	if r.results["a.TestNew"] != passed {
		t.Errorf("results = %v, want a.TestNew", r.results)
	}
}

// timedRun reads each test's time and the suites' total from this run's
// reports alone: the stale one removed, a tracked file not read.
func TestTimedRunReadsThisRunsReports(t *testing.T) {
	tree := reportTree(t, []string{"pom.xml"}, []string{"core/build/test-results/old/TEST-stale.xml"})
	s := Settings{Report: "**/*.xml",
		Command: `mkdir -p core/build/test-results/test && printf '<testsuite name="c" time="2"><testcase classname="c" name="TestA" time="1.5"/><testcase classname="c" name="TestB" time="0.5"><failure/></testcase></testsuite>' > core/build/test-results/test/TEST-c.xml`}
	tests, total, f := timedRun(tree, s, "here")
	if f != nil {
		t.Fatalf("timedRun: %+v", f)
	}
	if total != 2 || len(tests) != 2 || tests["c.TestA"].seconds != 1.5 || tests["c.TestB"].res != failed {
		t.Errorf("timedRun = %+v, total %v; want c.TestA 1.5 s, c.TestB failed, 2 s", tests, total)
	}
	if _, err := os.Stat(filepath.Join(tree, "pom.xml")); err != nil {
		t.Errorf("pom.xml removed: %v", err)
	}
	if _, _, f := timedRun(tree, Settings{Report: "**/*.xml", Command: "true"}, "here"); f == nil || f.Rule != "report-missing" {
		t.Errorf("no report written: %+v, want report-missing", f)
	}
}

// A results folder linked outside the tree, as Bazel's `bazel-testlogs`
// (#1189), is read through the link: the reports the run wrote there are
// this run's. Nothing out there is removed, and a past run's report the
// run left as it was is not read as this run's.
func TestReportsReadThroughLinkedFolder(t *testing.T) {
	for _, pattern := range []string{"out/*/test.xml", "out/**/test.xml"} {
		outside := t.TempDir()
		stale := filepath.Join(outside, "old", "test.xml")
		if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stale, []byte(`<testsuite name="o"><testcase classname="o" name="TestOld"><failure/></testcase></testsuite>`), 0o644); err != nil {
			t.Fatal(err)
		}
		tree := reportTree(t, []string{"pom.xml"}, nil)
		if err := os.Symlink(outside, filepath.Join(tree, "out")); err != nil {
			t.Fatal(err)
		}
		s := Settings{Report: pattern,
			Command: `mkdir -p out/new && printf '<testsuite name="a"><testcase classname="a" name="TestNew"/></testsuite>' > out/new/test.xml`}
		r, f := runIn(tree, s, "here")
		if f != nil {
			t.Fatalf("%s: runIn: %+v", pattern, f)
		}
		if len(r.results) != 1 || r.results["a.TestNew"] != passed {
			t.Errorf("%s: results = %v, want only a.TestNew passed", pattern, r.results)
		}
		if _, err := os.Stat(stale); err != nil {
			t.Errorf("%s: a report outside the tree was removed through the link: %v", pattern, err)
		}
	}
}
