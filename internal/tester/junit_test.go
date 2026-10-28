package tester

import (
	"os"
	"path/filepath"
	"testing"
)

// Each runner nests its report its own way: pytest one testsuite under
// testsuites, gotestsum one a package, a bare testsuite for others.
func TestReadJUnitAsRunnersWriteIt(t *testing.T) {
	reports := map[string]string{
		"pytest.xml": `<?xml version="1.0"?><testsuites><testsuite name="pytest"><testcase classname="tests.test_x" name="test_ok"/>` +
			`<testcase classname="tests.test_x" name="test_bad"><failure message="assert"/></testcase>` +
			`<testcase classname="tests.test_x" name="test_off"><skipped/></testcase></testsuite></testsuites>`,
		"go.xml": `<testsuites><testsuite name="example.com/m"><testcase classname="example.com/m" name="TestA"/>` +
			`<testcase classname="example.com/m" name="TestB"><error/></testcase></testsuite></testsuites>`,
		"bare.xml": `<testsuite><testcase name="alone"/><testcase name="twice"/><testcase name="twice"><failure/></testcase></testsuite>`,
	}
	want := map[string]result{"tests.test_x.test_ok": passed, "tests.test_x.test_bad": failed, "tests.test_x.test_off": skipped,
		"example.com/m.TestA": passed, "example.com/m.TestB": failed, "alone": passed, "twice": failed}
	dir := t.TempDir()
	got := map[string]result{}
	for name, body := range reports {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := readJUnit(file, got, nil); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for id, r := range want {
		if got[id] != r {
			t.Errorf("%s: %v, want %v", id, got[id], r)
		}
	}
	if len(got) != len(want) {
		t.Errorf("%d tests read, want %d: %v", len(got), len(want), got)
	}
}
