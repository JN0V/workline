package evaluation

// The reviewer's facets compared (#243): the single lens a push asks and the
// diff alone, each run over the same cases, their figures side by side. The
// figures are read from scoreReview's measure, so a stub run (no agent) and a
// real one are summed the same way.

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

// measured sums the figures of one scored review, read from its measure.
func (f *figures) add(rc *reviewCase, rr *reviewRun) {
	_, _, measure, _ := scoreReview(rc, rr)
	f.Runs++
	for _, part := range strings.Split(measure, "; ") {
		fields := strings.Fields(part)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "floor=") || strings.HasPrefix(fields[0], "outside-planted") {
			continue
		}
		for _, kv := range fields[1:] {
			k, v, _ := strings.Cut(kv, "=")
			n, _ := strconv.Atoi(v)
			switch k {
			case "planted":
				f.Planted += n
			case "found":
				f.Found += n
			case "fp":
				f.FalsePositives += n
			case "judge-no":
				f.JudgeRefusals += n
			case "in", "out":
				f.Tokens += n
			}
		}
	}
}

// compare runs both configurations over the cases, runs times each.
func compare(a, b facetConfig, cases []*reviewCase, run func(facetConfig, int, int) *reviewRun, runs int) (fa, fb figures) {
	for i, rc := range cases {
		for n := 0; n < runs; n++ {
			fa.add(rc, run(a, i, n))
			fb.add(rc, run(b, i, n))
		}
	}
	return fa, fb
}

func TestFacetConfigurationsOnPlantedDefects(t *testing.T) {
	rc := &reviewCase{Defects: []plant{
		{Lens: "correctness", Kind: "k1", At: []place{{File: "a.go", Lines: []int{1, 5}}}},
		{Lens: "edge-cases", Kind: "k2", At: []place{{File: "b.go", Lines: []int{1, 5}}}},
	}}
	lens := facetConfig{"single-lens", []string{"correctness"}}
	alone := facetConfig{"diff-alone", []string{"diff-alone"}}
	asked := map[string][]string{}
	run := func(c facetConfig, _, _ int) *reviewRun {
		asked[c.Name] = c.Facets
		rr := &reviewRun{Lenses: c.Facets, TokensIn: map[string]int{}, TokensOut: map[string]int{}}
		if c.Name == "single-lens" { // finds one, 100 tokens in, 10 out
			rr.Findings = []lensFinding{{Lens: "correctness", Where: "a.go:3", Severity: "important", Related: true, Judged: "yes"}}
			rr.TokensIn["correctness"], rr.TokensOut["correctness"] = 100, 10
			return rr
		}
		// the diff alone: both defects, one false positive, one refusal, 60 in
		rr.Findings = []lensFinding{
			{Lens: "diff-alone", Where: "a.go:3", Severity: "important", Related: true, Judged: "yes"},
			{Lens: "diff-alone", Where: "b.go:2", Severity: "important", Related: true, Judged: "yes"},
			{Lens: "diff-alone", Where: "b.go:3", Severity: "important", Related: true, Judged: "no"},
			{Lens: "diff-alone", Where: "c.go:9", Severity: "important", Related: true, Judged: "yes"},
		}
		rr.TokensIn["diff-alone"], rr.TokensOut["diff-alone"] = 60, 5
		return rr
	}
	fa, fb := compare(lens, alone, []*reviewCase{rc}, run, 1)
	if want := (figures{Runs: 1, Planted: 2, Found: 1, FalsePositives: 0, JudgeRefusals: 0, Tokens: 110}); fa != want {
		t.Errorf("single lens: %+v, want %+v", fa, want)
	}
	if want := (figures{Runs: 1, Planted: 2, Found: 2, FalsePositives: 1, JudgeRefusals: 1, Tokens: 65}); fb != want {
		t.Errorf("diff alone: %+v, want %+v", fb, want)
	}
	if slices.Equal(asked["single-lens"], asked["diff-alone"]) {
		t.Error("both configurations asked the same facets: one run twice")
	}
	out := report(lens, alone, fa, fb, 1)
	if strings.Contains(out, "recommendation:") || !strings.Contains(out, "no recommendation") {
		t.Errorf("one run recommended:\n%s", out)
	}
}

func TestRecommendationNeedsFiveRuns(t *testing.T) {
	a, b := facetConfig{Name: "single-lens"}, facetConfig{Name: "diff-alone"}
	fa := figures{Found: 5, Tokens: 200}
	fb := figures{Found: 9, Tokens: 100}
	for runs := 1; runs < minRuns; runs++ {
		if r := recommendation(a, b, fa, fb, runs); r != "" {
			t.Errorf("%d runs a case: recommended %q", runs, r)
		}
	}
	if r := recommendation(a, b, fa, fb, minRuns); r != "diff-alone" {
		t.Errorf("five runs: %q, want diff-alone", r)
	}
	if !strings.Contains(report(a, b, fa, fb, 5), "recommendation: diff-alone") {
		t.Error("report at five runs lacks the recommendation")
	}
}
