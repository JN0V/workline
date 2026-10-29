package developer

import (
	"slices"
	"testing"
)

// A rename names no file on its +++ line alone: both sides are held to
// the bounds, so a frozen test cannot leave by a rename.
func TestTouchedHoldsBothSidesOfARename(t *testing.T) {
	diff := "diff --git a/tests/test_mul.sh b/tests/old_mul.sh\nsimilarity index 100%\nrename from tests/test_mul.sh\nrename to tests/old_mul.sh\n"
	got := touched(diff)
	for _, want := range []string{"tests/test_mul.sh", "tests/old_mul.sh"} {
		if !slices.Contains(got, want) {
			t.Errorf("touched = %v, want %s in it", got, want)
		}
	}
}

func TestRuleOrder(t *testing.T) {
	c := checker{b: Bounds{Issue: 3, Scope: []string{"src/lib.go", "AGENTS.md"}, Frozen: map[string]int{"src/lib_test.go": 3}},
		steer: func(p string) bool { return p == "AGENTS.md" }, roles: ".workline/roles"}
	for p, want := range map[string]string{
		"src/lib.go":                "",
		"src/lib_more_test.go":      "",
		"tests/test_lib.sh":         "",
		"src/lib_test.go":           "acceptance-test-edit",
		"AGENTS.md":                 "steering-file",
		"roles/developer/policy.md": "steering-file",
		"src/roles/x/persona.md":    "steering-file",
		".workline/config.yaml":     "gate-file",
		".github/workflows/ci.yml":  "gate-file",
		"README.md":                 "outside-scope",
		"../elsewhere.go":           "outside-scope",
		"../tests/test_x.sh":        "outside-scope",
		"src/other.go":              "outside-scope",
	} {
		if got := c.rule(p); got != want {
			t.Errorf("rule(%s) = %q, want %q", p, got, want)
		}
	}
}
