package tester

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Same name in another file: given a new test whose file does not
// build, and a test of the same name in another package's file the
// head's report covers, when the tester checks which new tests ran,
// then the new test counts as not run, and its build break is said.
func TestUnbuiltTestsSameNameInAnotherFile(t *testing.T) {
	repo, run := gitRepo(t, map[string]string{"go.mod": "module example.com/m\n"})
	write := func(name, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(msg string) {
		t.Helper()
		run(repo, "add", ".")
		run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", msg)
	}
	write("other/old_test.go", "package other\n\nfunc TestNew(t *testing.T) {}\n")
	commit("other")
	base := run(repo, "rev-parse", "HEAD")
	write("calc/new_test.go", "package calc\n\nfunc TestNew(t *testing.T) { check(Missing()) }\n")
	commit("new test")
	head := run(repo, "rev-parse", "HEAD")
	rules, err := compileBuildErrors([]BuildErrors{
		{Error: `^(?P<file>\S+\.go):\d+:\d+: (?P<message>.+)$`, TestName: `(?m)^func (Test\w+)\(`},
	})
	if err != nil {
		t.Fatal(err)
	}
	ch := &change{base: base, head: head, tests: []string{"calc/new_test.go"}}
	// the head's report covers the other package's test of the same name only
	headRun := ran{results: map[string]result{"example.com/m/other.TestNew": passed}}
	got, err := unbuiltTests(repo, ch, headRun, []buildError{{file: "calc/new_test.go", rules: 0}}, rules)
	if want := []string{"calc/new_test.go:TestNew"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("unbuiltTests = %q, %v; want %q", got, err, want)
	}
}
