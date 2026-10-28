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
