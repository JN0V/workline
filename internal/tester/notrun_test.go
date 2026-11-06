package tester

import (
	"strings"
	"testing"
)

// A new test no report names is said not run, the filters above it on
// the base named, each by its own key, in the same order every run; with
// none, the command alone (#1149).
func TestNotRunNamesTheFilters(t *testing.T) {
	repo, _ := gitRepo(t, map[string]string{
		"platformio.ini": "[env:native]\ntest_filter = test_a, test_b\ntest_filter_extra = x\n",
		"package.json":   "{\n  \"jest\": {\n    \"testMatch\": [\"**/*.spec.js\"],\n    \"rootsDir\": \"x\"\n  }\n}\n",
	})
	files := []string{"web/test/test_new/test_new.cpp"}
	got := notRun(repo, "HEAD", files, "sh run.sh {report}")
	want := "the tests were written (web/test/test_new/test_new.cpp), but the report names no test of them: " +
		"the project's test command (`sh run.sh {report}`) did not run them, so nothing is proven red; " +
		"it may leave them out by `package.json`'s `testMatch` (`\"testMatch\": [\"**/*.spec.js\"],`), " +
		"`platformio.ini`'s `test_filter` (`test_filter = test_a, test_b`). Write them where that command runs them"
	if !strings.HasPrefix(got, want) {
		t.Errorf("notRun =\n%s\nwant it to open\n%s", got, want)
	}
	if again := notRun(repo, "HEAD", files, "sh run.sh {report}"); again != got {
		t.Errorf("not the same twice:\n%s\n%s", got, again)
	}
	bare, _ := gitRepo(t, map[string]string{"run.sh": "true\n"})
	if got := notRun(bare, "HEAD", files, "sh run.sh {report}"); strings.Contains(got, "leave them out") {
		t.Errorf("a filter named where none is set:\n%s", got)
	}
}
