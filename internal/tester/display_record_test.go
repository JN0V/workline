package tester

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Given a new test reported under a display name not written as such in
// its file, When it is red by an error on the base, Then it is frozen as
// a test, not dropped as a suite record.
func TestDisplayNameRedByErrorIsFrozenNotDropped(t *testing.T) {
	file := filepath.Join(t.TempDir(), "r.xml")
	body := `<testsuite name="io.x.FooTest">` +
		`<testcase classname="io.x.FooTest" name="plain"><failure/></testcase>` +
		`<testcase classname="io.x.FooTest" name="starts in IDLE state()"><error/></testcase></testsuite>`
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := ran{results: map[string]result{}, suite: map[string]string{}, classless: map[string]bool{}, erred: map[string]string{}}
	if err := readJUnit(file, r.results, r.suite, r.classless, r.erred); err != nil {
		t.Fatal(err)
	}
	// The test file writes the method `plain`, and the display name only
	// in an annotation, not as the report spells it.
	held := heldBy([]string{"FooTest.java"}, func(string) string { return "void plain() {}" })
	added := []string{"io.x.FooTest.plain", "io.x.FooTest.starts in IDLE state"}
	tests, records := r.suiteRecords(added, false, held)
	if !slices.Contains(tests, "io.x.FooTest.starts in IDLE state") {
		t.Errorf("tests %v, records %v: the display-named test is dropped as a suite record", tests, records)
	}
	if len(records) != 0 {
		t.Errorf("records %v, want none", records)
	}
}
