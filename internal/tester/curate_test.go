package tester

import "testing"

// The mutation testing report schema (Stryker's) reads as Gremlins' JSON
// does: each mutant's status by file, place and kind.
func TestReadMutantsSchema(t *testing.T) {
	out := map[string]string{}
	data := []byte(`{"schemaVersion":"1","files":{"src/a.js":{"language":"javascript","mutants":[
		{"id":"1","mutatorName":"ArithmeticOperator","status":"Killed","location":{"start":{"line":3,"column":7},"end":{"line":3,"column":8}}},
		{"id":"2","mutatorName":"BooleanLiteral","status":"Survived","location":{"start":{"line":5,"column":1},"end":{"line":5,"column":5}}}]}}}`)
	if err := readMutants(data, "src", out); err != nil {
		t.Fatal(err)
	}
	if out["src/a.js:3:7:ArithmeticOperator"] != "Killed" || !killed(out["src/a.js:3:7:ArithmeticOperator"]) {
		t.Errorf("killed mutant: %v", out)
	}
	if st := out["src/a.js:5:1:BooleanLiteral"]; killed(st) || !counted(st) {
		t.Errorf("survived mutant: %q", st)
	}
	if err := readMutants([]byte(`{"total": 3}`), "src", out); err == nil {
		t.Error("a report of neither form read")
	}
}

// A kept name protects a file by its path's end, or a test by its id.
func TestKeeps(t *testing.T) {
	kept := []string{"tester/curate-agent-none", "example.com/m/a.TestRun/sub"}
	for name, want := range map[string]bool{
		"tests/conformance/cases/tester/curate-agent-none.yaml": true,
		"tests/conformance/cases/tester/agent-none.yaml":        false,
		"example.com/m/a.TestRun/sub":                           true,
		"example.com/m/a.TestRunAll":                            false,
	} {
		if keeps(kept, name) != want {
			t.Errorf("keeps(%q) = %v, want %v", name, !want, want)
		}
	}
}

// A note explains a removal only when it names the test whole.
func TestMentions(t *testing.T) {
	for _, c := range []struct {
		note, name string
		want       bool
	}{
		{"TestRun: TestOther checks the same", "TestRun", true},
		{"`TestRun/sub` is TestOther's", "TestRun", true},
		{"TestRunAll keeps the check", "TestRun", false},
		{"MyTestRun keeps it", "TestRun", false},
		{"test_add_again: test_add checks it", "test_add", true},
		{"test_add_again keeps it", "test_add", false},
	} {
		if got := mentions(c.note, c.name); got != c.want {
			t.Errorf("mentions(%q, %q) = %v, want %v", c.note, c.name, got, c.want)
		}
	}
}

// A folder's measures are its own tests', never a subfolder's nor a
// folder whose name only starts the same; the root's too.
func TestInFolder(t *testing.T) {
	for _, c := range []struct {
		id, dir string
		files   []string
		want    bool
	}{
		{"example.com/m/internal/scale.TestRun/sub", "internal/scale", []string{"internal/scale/a_test.go"}, true},
		{"example.com/m/internal/scale2.TestRun", "internal/scale", []string{"internal/scale/a_test.go"}, false},
		{"example.com/m/internal/scale/sub.TestRun", "internal/scale", []string{"internal/scale/a_test.go"}, false},
		{"example.com/m.TestRun", ".", []string{"a_test.go"}, true},
		{"example.com/m/a.TestRun", ".", []string{"a_test.go"}, false},
		{"tests.test_add_again", "tests", []string{"tests/test_add_again.sh"}, true},
		{"tests.unit.test_add", "tests", []string{"tests/test_add.sh"}, false},
		{"test_add.TestAdd.test_one", ".", []string{"test_add.py"}, true},
		{"tests.test_add", ".", []string{"test_add.py"}, false},
	} {
		if got := inFolder(c.id, c.dir, c.files, "example.com/m"); got != c.want {
			t.Errorf("inFolder(%q, %q) = %v, want %v", c.id, c.dir, got, c.want)
		}
	}
}
