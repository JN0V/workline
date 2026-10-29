package evaluation

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAuditorCasesWithBlindJudge plays the auditor's cases with a judge
// answering `true`, then `false`, to every act, quoting nothing
// (testdata/auditor-judge.sh): no planted case is won that way, since a
// false act needs the material's words found again; each control is won
// by one of the two, which proves its act is drawn and read. ADR-0046:
// an auditor answering "stands" to everything fails here. No real agent.
func TestAuditorCasesWithBlindJudge(t *testing.T) {
	judge, _ := filepath.Abs("testdata/auditor-judge.sh")
	n := 0
	for f, c := range loadCases(t) {
		if c.Run.Role != "auditor" {
			continue
		}
		n++
		planted := false
		for _, g := range c.Grade {
			if w, ok := g["finding"].(map[string]any); ok && w["rule"] == "act-false" {
				planted = true
			}
		}
		won := 0
		for _, answer := range []string{"true", "false"} {
			r, err := play(t, c, "cmd:sh "+judge+" "+answer)
			if err != nil {
				t.Fatal(err)
			}
			if r.res.AgentCalls != 1 {
				t.Errorf("%s, %s: %d calls, want the one act drawn read: %+v", f, answer, r.res.AgentCalls, r.res.Findings)
			}
			_, failed, _ := grade(c, r)
			if len(failed) == 0 {
				won++
			}
			if planted && len(failed) == 0 {
				t.Errorf("%s: won by a judge answering %s blind", f, answer)
			}
			t.Logf("%s, %s: lost %s", c.Case, answer, strings.Join(failed, "; "))
		}
		if !planted && won != 1 {
			t.Errorf("%s: a control won by %d of the two blind answers, want 1", f, won)
		}
	}
	if n < 6 {
		t.Errorf("%d auditor cases, want 6 at least", n)
	}
}
