package tester

import (
	"slices"
	"testing"
)

func TestParseObjection(t *testing.T) {
	o, missing := parseObjection("workline:objection\n- **test:** `TestTwoByThree`\n- Scenario: Two by three\n- kind: Wrong interface\n- why: mul takes a list,\nnot two numbers.")
	if len(missing) > 0 || o.Test != "TestTwoByThree" || o.Scenario != "Two by three" || kindOf(o.Kind) != "interface" || o.Why != "mul takes a list,\nnot two numbers." {
		t.Fatalf("%+v, missing %v", o, missing)
	}
	_, missing = parseObjection("workline:objection\ntest: x\nwhy: too strict")
	if !slices.Equal(missing, []string{"scenario", "kind (one of: " + kindNames() + ")"}) {
		t.Fatalf("missing %v", missing)
	}
	if o, _ := parseObjection("workline:objection\nwhy: use the `validate`"); o.Why != "use the `validate`" {
		t.Fatalf("why %q", o.Why)
	}
	if o, _ := parseObjection("workline:objection\nscenario: **Two** by **three**"); o.Scenario != "**Two** by **three**" {
		t.Fatalf("scenario %q", o.Scenario)
	}
	if k := kindOf("the need itself"); k != "need" {
		t.Fatalf("kind %q", k)
	}
}
