package verdict

import "testing"

// Only the codes that move a state have a kind; a run's error of a shape
// never seen is a machine error all the same, as engine-error
// (ADR-0063, §6).
func TestOutcomeOf(t *testing.T) {
	cases := []struct {
		name     string
		status   string
		findings []Finding
		want     string
	}{
		{"a test that does not build", Block, []Finding{{Rule: "build-broken", Message: "no matching constructor"}}, "machine-error(build)"},
		{"a patch that does not apply", Block, []Finding{{Rule: "patch-unreadable"}}, "machine-error(patch)"},
		{"an unknown failure", Block, []Finding{{Rule: "engine-error", Message: "a shape never seen"}}, "machine-error(run)"},
		{"the code named", Pass, []Finding{{Rule: NeedsCode, Message: "a.h"}}, "needs-input(engine)"},
		{"a refusal on the issue", Block, []Finding{{Rule: "test-green-on-base", Message: "passes on the base"}}, "needs-input(person)"},
		{"a warning of a machine code", Block, []Finding{{Rule: "build-broken", Level: "warn"}, {Rule: "test-green-on-base"}}, "needs-input(person)"},
		{"progress", Pass, []Finding{{Rule: "state-moved", Level: "info"}}, "progress"},
	}
	for _, c := range cases {
		if got := OutcomeOf(c.status, c.findings).String(); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}
