package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/devmark"
)

// In CI, the notice names the job's rights, not a person's account,
// and claims nothing of its token (ADR-0058); outside it, the person's.
// The pull request's line says it in the third person.
func TestRightsInCI(t *testing.T) {
	for _, v := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "CI"} {
		t.Setenv(v, "")
	}
	if got := rights(); got != "the agent runs with your account's rights; the forge's guards decide what merges" {
		t.Errorf("outside CI: %q", got)
	}
	if got := rightsLine(); got != "Its agent ran with the rights of the account that ran it; the forge's guards decide what merges." {
		t.Errorf("outside CI, the pull request: %q", got)
	}
	t.Setenv("GITLAB_CI", "true")
	if got := rights(); got != "the agent runs with the CI job's rights; the forge's guards decide what merges" {
		t.Errorf("in CI: %q", got)
	}
	if got := rightsLine(); got != "Its agent ran with the rights of the CI job; the forge's guards decide what merges." {
		t.Errorf("in CI, the pull request: %q", got)
	}
}

// The pull request says the tester's red proof was not settled, when its
// Draft's record says so (ADR-0065).
func TestBodySaysRedUnproven(t *testing.T) {
	a := devmark.Act{Issue: 3, Acceptance: []string{"T.kt"}, RedUnproven: "the judge cannot tell"}
	if b := body(a, false); !strings.Contains(b, "**Red unproven**") || !strings.Contains(b, "the judge cannot tell") {
		t.Errorf("body: %s", b)
	}
	if b := body(devmark.Act{Issue: 3}, false); strings.Contains(b, "Red unproven") {
		t.Errorf("proven, said unproven: %s", b)
	}
}

// A part left, by its agent's words, a refusal or the Scope, makes the
// pull request part of its issue, never closing it, never "proven"; an
// unproven scenario is never "proven" either (after #2216).
func TestBodyPartOfWhenAPartIsLeft(t *testing.T) {
	green := &devmark.Proof{Result: devmark.Green}
	notDone := "Its agent's closing words as they came, read as no YAML: nothing of them applied.\n\n~~~\nNot done: the CI step, only a person.\n~~~\n"
	for name, a := range map[string]devmark.Act{
		"not done": {Issue: 3, Proof: green, Unread: []string{notDone}},
		"refused":  {Issue: 3, Proof: green, Refused: []devmark.Refusal{{File: ".github/workflows/ci.yml", Rule: "gate-file"}}},
	} {
		b := body(a, true)
		if !strings.HasPrefix(b, "Part of #3\n\n## "+LeftHeading) || strings.Contains(b, "Closes #3") || strings.Contains(b, "proven on") {
			t.Errorf("%s: %s", name, b)
		}
	}
	if b := body(devmark.Act{Issue: 3, Proof: green}, true, "`.github/workflows`: in the Scope"); !strings.HasPrefix(b, "Part of #3") || !strings.Contains(b, "- `.github/workflows`: in the Scope\n") {
		t.Errorf("scope left: %s", b)
	}
	if b := body(devmark.Act{Issue: 3, Proof: green, Unproven: []string{"- **x**: by eye"}}, true); !strings.HasPrefix(b, "Closes #3") || strings.Contains(b, "proven on") {
		t.Errorf("unproven: %s", b)
	}
	if b := body(devmark.Act{Issue: 3, Proof: green, Unread: []string{"Its objections file, could not be read. As it came:\n\n~~~\nbad\n~~~\n"}}, true); !strings.HasPrefix(b, "Closes #3") || !strings.Contains(b, "## Its agent's words, unread") {
		t.Errorf("other words: %s", b)
	}
}
