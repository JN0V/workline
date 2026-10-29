package developer

import "testing"

func TestReviewRows(t *testing.T) {
	comment := "| | | Where | Finding |\n|---|---|---|---|\n" +
		"| **blocks** | correctness | `lib.sh:2` | Important: mul adds — it gives 5. Fix: multiply. |\n" +
		"| warns | design | `lib.sh:1` | Important: a \\| in it; also tests: more |\n" +
		"| warns | style | `lib.sh:2` | Nit: name the operands |\n" +
		"| warns | finding-judged-no |  | dropped, from the correctness lens: \"x\" at lib.sh:3 — the judge: no |\n" +
		"| warns | gate-lint | `a.go:4` | unused variable |\n"
	asked, nits := reviewRows(comment)
	if nits != 1 || len(asked) != 2 {
		t.Fatalf("asked %v, nits %d: want 2 asked, 1 nit", asked, nits)
	}
	if asked[0].Holds != "blocks" || asked[0].Key != "correctness|lib.sh|mul adds" {
		t.Errorf("first: %+v", asked[0])
	}
	if asked[1].Holds != "warns" || asked[1].Key != "design|lib.sh|a | in it; also tests: more" {
		t.Errorf("second: %+v", asked[1])
	}
}

func TestDeveloperIssue(t *testing.T) {
	for branch, want := range map[string]int{"workline/developer/12": 12, "workline/tester/issue-12": 0, "workline/developer/x": 0} {
		if got := developerIssue(branch); got != want {
			t.Errorf("%s: %d, want %d", branch, got, want)
		}
	}
}
