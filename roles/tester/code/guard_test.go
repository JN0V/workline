package tester

import (
	"strings"
	"testing"
)

// Guard written: Given a scenario kept as today that no existing test
// checks, When the tester writes the issue's tests, Then it is no longer
// said that a test of it "is not written": its Draft holds a guard test
// of it, green on the base. A scenario kept as today that an existing
// test checks keeps its reason, naming that test.
func TestKeptScenarioWithNoGuardIsGivenOne(t *testing.T) {
	repo, run := gitRepo(t, map[string]string{
		"add.go":      "package calc\n\nfunc Add(a, b int) int { return a + b }\n",
		"add_test.go": "package calc\n\nimport \"testing\"\n\nfunc TestAddTwoNumbers(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"sum\")\n\t}\n}\n",
	})
	base := run(repo, "rev-parse", "HEAD")
	s := Settings{Paths: []string{"**/*_test.go"}}
	said := map[string]string{
		"Late fee":  "kept as today: no existing test checks it",
		"Add twice": "kept as today: guarded by `TestAddTwoNumbers`",
	}
	guardedSaid(repo, base, s, said)
	for _, bad := range []string{"is not written", "nothing guards it"} {
		if strings.Contains(said["Late fee"], bad) {
			t.Errorf("a scenario kept as today with no guard is said %q, when the tester writes its guard test: %q", bad, said["Late fee"])
		}
	}
	if got := said["Add twice"]; got != "kept as today: guarded by `TestAddTwoNumbers`" {
		t.Errorf("a scenario guarded by an existing test keeps its reason, naming it: %q", got)
	}
}
