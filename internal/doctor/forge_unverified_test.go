package doctor

import "testing"

func TestCommandKnowingNoReadOperationIsUnverified(t *testing.T) {
	repo := realCallStub(t, `echo '{"error":"unknown operation"}'`)
	cs := forgeChecks(t, repo, "cmd:./forgejo.sh")
	if len(cs) != 1 || cs[0].Level != Warn || cs[0].Rule != "forge-command-unverified" || cs[0].Fix == "" {
		t.Fatalf("want one unverified warning, got %+v", cs)
	}
}

func TestUnknownMilestonesFallsBackToIssues(t *testing.T) {
	repo := realCallStub(t, `case "$WORKLINE_FORGE_OPERATION" in milestones) echo '{"error":"unknown operation milestones"}' ;; *) echo '{}' ;; esac`)
	cs := forgeChecks(t, repo, "cmd:./forgejo.sh")
	if len(cs) != 1 || cs[0].Level != OK || cs[0].Rule != "forge-command" {
		t.Fatalf("want one reachable check, got %+v", cs)
	}
}

func TestIssuesFailingAfterUnknownMilestonesIsReported(t *testing.T) {
	repo := realCallStub(t, `case "$WORKLINE_FORGE_OPERATION" in milestones) echo '{"error":"unknown operation milestones"}' ;; *) echo '401 bad token' >&2; exit 1 ;; esac`)
	cs := forgeChecks(t, repo, "cmd:./forgejo.sh")
	if len(cs) != 1 || cs[0].Rule != "forge-command-failing" {
		t.Fatalf("want a failing warning, got %+v", cs)
	}
}
