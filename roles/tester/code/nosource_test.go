package tester

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/verdict"
)

// gradleLike runs app/test_*.sh, each a test, to a JUnit report; with
// none there it says NO-SOURCE, writes no report and exits with `quiet`,
// as Gradle skips a module with no test sources (#2091).
func gradleLike(quiet string) string {
	return `set -- app/test_*.sh; if [ ! -f "$1" ]; then echo '> Task :app:testDebugUnitTest NO-SOURCE'; exit ` + quiet + `; fi; ` +
		`f=0; { echo '<testsuite name="app">'; for t in "$@"; do n=$(basename "$t" .sh); ` +
		`if sh "$t" >/dev/null 2>&1; then echo "<testcase classname=\"app\" name=\"$n\"/>"; ` +
		`else f=1; echo "<testcase classname=\"app\" name=\"$n\"><failure/></testcase>"; fi; done; echo '</testsuite>'; } > {report}; exit $f`
}

// noSourceRepo is a project whose app module has no test on main, and a
// change adding the first test there, and mul when code is set.
func noSourceRepo(t *testing.T, code bool) string {
	t.Helper()
	repo, git := gitRepo(t, map[string]string{"lib.sh": "add() { echo $(($1 + $2)); }\n", "build.gradle.kts": "plugins {}\n"})
	if err := os.Mkdir(filepath.Join(repo, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "app", "build.gradle.kts"), []byte("plugins {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit := func(msg string) {
		git(repo, "add", ".")
		git(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", msg)
	}
	commit("build: a module")
	git(repo, "checkout", "-q", "-b", "change")
	if code {
		lib, _ := os.ReadFile(filepath.Join(repo, "lib.sh"))
		if err := os.WriteFile(filepath.Join(repo, "lib.sh"), append(lib, "mul() { echo $(($1 * $2)); }\n"...), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "app", "test_mul.sh"), []byte(". ./lib.sh\n[ \"$(mul 2 3)\" = 6 ]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit("feat: multiply")
	return repo
}

// On the base, a module with no test that writes no report and exits 0
// is an empty run: the first test is proven red there and green on the
// head. Exiting non-zero, it is still a command that did not run.
func TestRedFirstModuleWithoutTests(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	repo := noSourceRepo(t, true)
	for _, c := range []struct {
		name, quiet string
		want        string
		summary     string
	}{
		{"exit 0", "0", verdict.Pass, "1 new test, red on the base, green on the head"},
		{"exit 1", "1", verdict.Block, "the red-first check did not run"},
	} {
		s, err := SettingsFrom(map[string]any{"test-command": gradleLike(c.quiet), "test-report": "junit.xml", "test-paths": []string{"**/test_*.sh"}})
		if err != nil {
			t.Fatal(err)
		}
		p := prove(repo, "main", s)
		if p.v.Status != c.want || !strings.Contains(p.v.Summary, c.summary) {
			t.Errorf("%s: %s %q %+v, want %s %q", c.name, p.v.Status, p.v.Summary, p.v.Findings, c.want, c.summary)
		}
		if c.want == verdict.Pass && (len(p.added) != 1 || p.added[0] != "app.test_mul") {
			t.Errorf("%s: added %v, want app.test_mul", c.name, p.added)
		}
		if c.want == verdict.Block && (len(p.v.Findings) == 0 || p.v.Findings[0].Rule != "report-missing" || !strings.Contains(p.v.Findings[0].Message, "on the base")) {
			t.Errorf("%s: findings %+v, want report-missing on the base", c.name, p.v.Findings)
		}
	}
}

// A command that never writes a report, on the base nor with the tests
// (tests alone), and prints no build error is refused with the base's
// finding, never read as tests to mend.
func TestRedFirstNoReportAnywhereRefused(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	repo := noSourceRepo(t, false)
	s, err := SettingsFrom(map[string]any{"test-command": "sh app/test_mul.sh; true", "test-report": "junit.xml", "test-paths": []string{"**/test_*.sh"}})
	if err != nil {
		t.Fatal(err)
	}
	p := prove(repo, "main", s)
	if p.v.Summary != "the red-first check did not run" || len(p.v.Findings) == 0 || !strings.Contains(p.v.Findings[0].Message, "on the base") {
		t.Errorf("got %s %q %+v, want refused, report-missing on the base", p.v.Status, p.v.Summary, p.v.Findings)
	}
}
