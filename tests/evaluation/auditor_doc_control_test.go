package evaluation

import (
	"path/filepath"
	"testing"
)

// TestAuditorTrueDocStands: Given a doc vouched for whose lines are true of
// its sources (documentalist-checked-true), When the auditor's sample reads
// it, Then the act stands: the judge saying it true lets it stand, and a
// judge saying it false, quoting nothing, does not (a doc said false with
// no passage and no source proven is no stand: ADR-0046, an auditor
// answering "stands" to everything fails). No real agent.
func TestAuditorTrueDocStands(t *testing.T) {
	c, err := readCase("cases/auditor/documentalist-checked-true.yaml")
	if err != nil {
		t.Fatal(err)
	}
	judge, _ := filepath.Abs("testdata/auditor-judge.sh")
	won := map[string]bool{}
	for _, answer := range []string{"true", "false"} {
		r, err := play(t, &c, "cmd:sh "+judge+" "+answer)
		if err != nil {
			t.Fatal(err)
		}
		if r.res.AgentCalls != 1 {
			t.Errorf("%s: %d calls, want the one doc drawn read: %+v", answer, r.res.AgentCalls, r.res.Findings)
		}
		_, failed, _ := grade(&c, r)
		won[answer] = len(failed) == 0
		t.Logf("%s: lost %v; findings %+v", answer, failed, r.res.Findings)
	}
	if !won["true"] {
		t.Error("a judge finding the true doc true does not let it stand")
	}
	if won["false"] {
		t.Error("a judge saying the doc false, with no passage and no source quoted, let it stand")
	}
}
