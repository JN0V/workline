package main

import "testing"

// A reviewer run whose findings needed no judge names no judge and fewer
// models: the same replay case, counted once in its set.
func TestPoolCasesCountsACaseOnce(t *testing.T) {
	const kase, version, agent = "reviewer/clean-dc-251-ha-command-path", "2d69e7231", "claude"
	groups := map[key]*group{
		{kase, version, agent, "claude-sonnet-5-5", "medium", "claude-sonnet-5-5 (context)"}: {scores: []float64{0, 0}},
		{kase, version, agent, "claude-sonnet-5-5", "medium", ""}:                            {scores: []float64{1}, passes: 1},
		{kase, "9cc037b76", agent, "claude-sonnet-5-5", "medium", ""}:                        {scores: []float64{1}, passes: 1},
		{"reviewer/a-fixture", "", "", "claude-sonnet-5-5", "medium", ""}:                    {scores: []float64{1}, passes: 1},
	}
	cases := poolCases(groups)
	if len(cases) != 2 {
		t.Fatalf("%d cases, want 2: one a workline commit, the fixture left out", len(cases))
	}
	g := cases[key{kase: kase, version: version, agent: agent}]
	if g == nil || len(g.scores) != 3 || g.passes != 1 {
		t.Fatalf("the case at %s: %+v, want its 3 runs pooled, 1 full", version, g)
	}
}
