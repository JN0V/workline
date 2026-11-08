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
		// A run too big for its role asks the product owner (ADR-0064, §2).
		{"a prompt over the budget", Block, []Finding{{Rule: "prompt-over-budget", Level: "warn", Message: "prompt too large"}, {Rule: "no-test-written"}}, "needs-input(product-owner: size)"},
		{"a prompt over the budget, passed", Pass, []Finding{{Rule: "prompt-over-budget", Level: "warn"}}, "needs-input(product-owner: size)"},
		{"blocked at its token cap", Block, []Finding{{Rule: "ai-max-tokens", Level: "warn"}, {Rule: "build-broken"}}, "needs-input(product-owner: size)"},
		{"passed at its token cap", Pass, []Finding{{Rule: "ai-max-tokens", Level: "warn"}}, "progress"},
		{"blocked, its input cut", Block, []Finding{{Rule: CutRule, Level: "warn"}, {Rule: "test-green-on-base"}}, "needs-input(product-owner: size)"},
		{"passed, its input cut", Pass, []Finding{{Rule: CutRule, Level: "warn"}}, "progress"},
		{"a machine error, its input cut", Block, []Finding{{Rule: CutRule, Level: "warn"}, {Rule: "build-broken"}}, "machine-error(build)"},
		{"the code named, its input cut", Block, []Finding{{Rule: CutRule, Level: "warn"}, {Rule: NeedsCode}}, "needs-input(engine)"},
	}
	for _, c := range cases {
		if got := OutcomeOf(c.status, c.findings).String(); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

// A size is never a machine error, so never retried: the same input, the
// same prompt (#1160); said by the role's own finding before the engine's.
func TestSizeAsksTheProductOwner(t *testing.T) {
	size := OutcomeOf(Block, []Finding{{Rule: "prompt-over-budget", Level: "warn", Message: "the engine's"}, {Rule: "prompt-over-budget", Message: "the role's"}})
	if size.Machine() || !size.Crossed() || size.Said != "the role's" {
		t.Errorf("a size: %+v; want the product owner asked, said by the role", size)
	}
	if build := OutcomeOf(Block, []Finding{{Rule: "build-broken"}}); build.Crossed() || !build.Machine() {
		t.Errorf("a build error: %+v; want a machine error", build)
	}
}
