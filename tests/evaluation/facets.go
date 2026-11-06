package evaluation

// The reviewer's facets compared (#243): the logic that turns the figures of
// two configurations into a report and, only when the runs allow, a
// recommendation. The tests (facets_test.go) drive it.

import (
	"fmt"
	"strings"
)

// facetConfig is one configuration measured: the facets a run asks, as
// `--input lenses=` names them.
type facetConfig struct {
	Name   string
	Facets []string
}

// figures are what a configuration found over its runs.
type figures struct {
	Runs, Planted, Found, FalsePositives, JudgeRefusals, Tokens int
}

// minRuns is the runs a case a configuration needs before the evaluation
// recommends it as a default (ADR-0014: checked is earned by what was read).
const minRuns = 5

// recommendation names the configuration to set as the default for a
// moment, or says why none: nothing is recommended on fewer than minRuns runs
// a case, only the numbers. A tie, or a loss in defects found, is no win.
func recommendation(a, b facetConfig, fa, fb figures, runsPerCase int) string {
	if runsPerCase < minRuns {
		return ""
	}
	better := func(x, y figures) bool {
		return x.Found >= y.Found && x.FalsePositives <= y.FalsePositives && x.Tokens < y.Tokens
	}
	switch {
	case better(fb, fa):
		return b.Name
	case better(fa, fb):
		return a.Name
	}
	return "neither"
}

// report prints the side-by-side numbers, and a recommendation only when
// the runs allow one.
func report(a, b facetConfig, fa, fb figures, runsPerCase int) string {
	var sb strings.Builder
	for _, x := range []struct {
		c facetConfig
		f figures
	}{{a, fa}, {b, fb}} {
		fmt.Fprintf(&sb, "%s (%s): found %d/%d, false positives %d, judge refusals %d, tokens %d\n",
			x.c.Name, strings.Join(x.c.Facets, ","), x.f.Found, x.f.Planted, x.f.FalsePositives, x.f.JudgeRefusals, x.f.Tokens)
	}
	if r := recommendation(a, b, fa, fb, runsPerCase); r != "" {
		fmt.Fprintf(&sb, "recommendation: %s\n", r)
	} else {
		fmt.Fprintf(&sb, "no recommendation, %d runs a case, %d needed\n", runsPerCase, minRuns)
	}
	return sb.String()
}
