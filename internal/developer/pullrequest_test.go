package developer

import "testing"

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
