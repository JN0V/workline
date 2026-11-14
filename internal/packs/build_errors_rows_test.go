package packs

import (
	"fmt"
	"strings"
	"testing"
)

// Each pack with build-errors rows gives them as the tester's default,
// and a pack listed twice, or two packs sharing a row, give it once.
func TestTesterDefaultsBuildErrorsOtherPacks(t *testing.T) {
	b := Builtin()
	for _, name := range []string{"go", "java-maven", "python-pytest"} {
		p := b[name]
		if len(p.BuildErrors) == 0 {
			t.Errorf("%s: no build-errors rows", name)
			continue
		}
		d := TesterDefaults([]Pack{p})
		rows, _ := d["build-errors"].([]any)
		if len(rows) < len(p.BuildErrors) {
			t.Errorf("%s: %d rows in the default, the pack has %d", name, len(rows), len(p.BuildErrors))
		}
		if !strings.Contains(fmt.Sprint(rows), p.BuildErrors[0].Error) {
			t.Errorf("%s: its first error pattern is not in the default: %v", name, rows)
		}
	}
}

func TestTesterDefaultsBuildErrorsDedup(t *testing.T) {
	k := Builtin()["kotlin-gradle"]
	once := TesterDefaults([]Pack{k})["build-errors"].([]any)
	twice := TesterDefaults([]Pack{k, k})["build-errors"].([]any)
	if len(once) != len(twice) {
		t.Errorf("a pack listed twice doubled its rows: %d then %d", len(once), len(twice))
	}
	// Same error pattern, different fields: both rows stay.
	a := Pack{Name: "a", BuildErrors: []BuildError{{Error: "x", TestName: "t1"}}}
	c := Pack{Name: "c", BuildErrors: []BuildError{{Error: "x", TestName: "t2"}}}
	rows := TesterDefaults([]Pack{a, c})["build-errors"].([]any)
	if len(rows) < 2 {
		t.Errorf("rows with one pattern and different fields were merged: %v", rows)
	}
}
