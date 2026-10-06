package evaluation

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestChecks(t *testing.T) {
	if k := keptWords("ci: run the project's line in the templates", "ci: run the project's line in templates"); k < 0.85 {
		t.Errorf("cutting a word keeps the others: %.2f", k)
	}
	if k := keptWords("ci: run the project's line in the templates", "ci: refactor templates to use line routing"); k > 0.5 {
		t.Errorf("rewording keeps few of them: %.2f", k)
	}
	c := &caseFile{}
	c.Run.Message = "ci: run the line"
	if why := check("no-vague-words", true, c, &run{message: "ci: refactor the line"}); why == "" {
		t.Error("a vague word brought in is caught")
	}
	if why := check("no-vague-words", true, c, &run{message: "ci: run the line in the templates"}); why != "" {
		t.Error(why)
	}
}

func TestAnsweredBy(t *testing.T) {
	m, e, in, out, c := answeredBy([]call{{"low", "claude-haiku-4-5", 1000, 50, 0.01, "part"}, {"low", "claude-sonnet-5", 2000, 70, 0.02, "judge"}})
	if m != "claude-haiku-4-5>claude-sonnet-5" || e != "low" || in != "3000" || out != "120" || c != "0.0300" {
		t.Errorf("got %q %q %q %q %q", m, e, in, out, c)
	}
}

func TestJudge(t *testing.T) {
	c := &caseFile{}
	c.About, c.Run.Message = "a rewrite", "fix: stop crashing on an empty file"
	r := &run{repo: t.TempDir(), message: "refactor: file handling"}
	answer := func(note string) string {
		return `cmd:cat > /dev/null; echo 'model: a-judge' > "$WORKLINE_CALL"; echo '- note: "` + note + `"'`
	}
	t.Setenv("WORKLINE_EVAL", "claude")
	t.Setenv("WORKLINE_JUDGE", answer("no: the crash is gone from it"))
	if why, err := judge("Same meaning?", c, r); err != nil || why != "the crash is gone from it" || r.judgedBy != "a-judge (provider)" {
		t.Errorf("a no is a lost point, with its reason: %q %v %q", why, err, r.judgedBy)
	}
	t.Setenv("WORKLINE_JUDGE", answer("yes: same meaning"))
	if why, err := judge("Same meaning?", c, r); err != nil || why != "" {
		t.Errorf("a yes is a point: %q %v", why, err)
	}
	t.Setenv("WORKLINE_JUDGE", answer("maybe"))
	if _, err := judge("Same meaning?", c, r); err == nil {
		t.Error("neither yes nor no is no verdict")
	}
	t.Setenv("WORKLINE_JUDGE", "claude:opus")
	t.Setenv("WORKLINE_JUDGE_AT_LEAST", "provider")
	if _, err := judge("Same meaning?", c, r); err == nil {
		t.Error("a judge of the graded agent's provider is refused when another provider is asked")
	}
	t.Setenv("WORKLINE_JUDGE_AT_LEAST", "")
	t.Setenv("WORKLINE_JUDGE", "")
	c.Grade = []map[string]any{{"judge": "Same meaning?"}}
	if passed, failed, skipped := grade(c, r); passed != 0 || len(failed) != 0 || len(skipped) != 1 {
		t.Errorf("without a judge, the check is skipped: %d %v %v", passed, failed, skipped)
	}
}

// grades are the checks a case may ask, besides judge.
var grades = map[string]bool{"status": true, "agent-calls-max": true, "subject-max": true, "keeps-words": true,
	"no-vague-words": true, "file-contains": true, "file-lacks": true, "checked-is-head": true, "judged-is-head": true,
	"checked-unchanged": true, "never-confirms": true, "body-unchanged": true, "lines-max": true, "new-files-min": true,
	"finding": true, "no-finding": true, "judge": true}

func loadCases(t *testing.T) map[string]*caseFile {
	files, _ := filepath.Glob("cases/*/*.yaml")
	cases := map[string]*caseFile{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var c caseFile
		if err := yaml.Unmarshal(data, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		cases[f] = &c
	}
	return cases
}

// TestCasesLoad: every case reads, and asks only checks that exist.
func TestCasesLoad(t *testing.T) {
	for f, c := range loadCases(t) {
		if c.Case == "" || c.Run.Role == "" {
			t.Errorf("%s: no case or role", f)
		}
		for _, g := range c.Grade {
			for kind := range g {
				if !grades[kind] {
					t.Errorf("%s: unknown check %q", f, kind)
				}
			}
		}
	}
}

// TestDriftedCasesGradeBoth: a documentalist case on the `drifted` fixture
// grades both sides of ADR-0014 — never `checked` over a planted falsehood,
// and `checked` on the clean control.
func TestDriftedCasesGradeBoth(t *testing.T) {
	for f, c := range loadCases(t) {
		if c.Given.Repo != "drifted" {
			continue
		}
		falsehood, control := false, false
		for _, g := range c.Grade {
			_, falsehood1 := g["never-confirms"]
			falsehood = falsehood || falsehood1
			control = control || g["checked-is-head"] == "docs/clock/events.md"
		}
		if !falsehood || !control {
			t.Errorf("%s: grades a falsehood never confirmed (%v), the control checked (%v)", f, falsehood, control)
		}
	}
}

// TestDriftedWithFakeAgents plays the `drifted` cases with fake agents that
// change headers only (testdata/drifted-agent.sh): one vouching for every
// doc must lose the falsehood checks; one recording `judged` on every doc
// keeps them and loses the control; a careful one keeps both. No real
// agent, no tokens.
func TestDriftedWithFakeAgents(t *testing.T) {
	agent, _ := filepath.Abs("testdata/drifted-agent.sh")
	lost := func(failed []string, kind string) bool {
		for _, f := range failed {
			if strings.HasPrefix(f, kind+":") {
				return true
			}
		}
		return false
	}
	n := 0
	for f, c := range loadCases(t) {
		if c.Given.Repo != "drifted" {
			continue
		}
		n++
		for _, mode := range []string{"vouches", "judges", "careful"} {
			t.Run(c.Case+"/"+mode, func(t *testing.T) {
				r, err := play(t, c, "cmd:sh "+agent+" "+mode)
				if err != nil {
					t.Fatal(err)
				}
				if len(r.res.Applied) == 0 {
					t.Fatalf("%s: the fake agent's patch was not applied: %+v", f, r.res.Findings)
				}
				passed, failed, _ := grade(c, r)
				t.Logf("%d/%d; lost: %s", passed, passed+len(failed), strings.Join(failed, "; "))
				for kind, want := range map[string]bool{
					"status":            false,
					"never-confirms":    mode == "vouches",
					"checked-unchanged": mode == "vouches",
					"judged-is-head":    mode == "vouches",
					"checked-is-head":   mode == "judges",
					"no-finding":        false,
					"finding":           false,
				} {
					if lost(failed, kind) != want {
						t.Errorf("%s, %s: %s lost: %v, want %v (failed: %v)", f, mode, kind, !want, want, failed)
					}
				}
			})
		}
	}
	if n < 2 {
		t.Errorf("%d cases on the drifted fixture", n)
	}
}

func TestNewChecks(t *testing.T) {
	dir := t.TempDir()
	write := func(p, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755)
		os.WriteFile(filepath.Join(dir, p), []byte(s), 0o644)
	}
	write("a.md", "---\nsources: [x]\nchecked: 1234567\njudged: abcdef1\n---\n# A\n")
	write("b.md", "<!-- workline\nsources: [x]\nchecked: 1234567\n-->\n# B\n")
	r := &run{repo: dir, headBefore: "abcdef1234", before: map[string]string{"a.md": "---\nchecked: 1234567\n---\n", "b.md": "---\nchecked: 7654321\n---\n"}}
	r.res.Findings = append(r.res.Findings, struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Message string `json:"message"`
	}{"count-off", "docs/x.md", "`Clock.h` has 569 lines, not 524 (line 26, Clock/include/Clock.h)"})
	for _, tc := range []struct {
		kind string
		v    any
		pass bool
	}{
		{"judged-is-head", "a.md", true},
		{"checked-is-head", "a.md", false},
		{"judged-is-head", "b.md", false},
		{"checked-unchanged", "a.md", true},
		{"checked-unchanged", "b.md", false},
		{"finding", map[string]any{"rule": "count-off", "where": "docs/x.md", "message": "Clock.h"}, true},
		{"finding", map[string]any{"rule": "count-off", "message": "ClockWebUI.h"}, false},
		{"no-finding", map[string]any{"rule": "count-off", "message": "800"}, true},
		{"no-finding", map[string]any{"rule": "count-off"}, false},
	} {
		if why := check(tc.kind, tc.v, &caseFile{}, r); (why == "") != tc.pass {
			t.Errorf("%s %v: passed %v, want %v (%s)", tc.kind, tc.v, why == "", tc.pass, why)
		}
	}
	if headerField("---\nchecked: 1234567\n---\nchecked: 9999999\n", "checked") != "1234567" || headerField("# no header\nchecked: 1\n", "checked") != "" {
		t.Error("headerField reads the header only")
	}
}

// TestReviewerCasesPointRight builds each reviewer case: each place it names
// holds its text within its lines, the code builds, and the fixture's tests
// pass — a planted defect the tests already catch would measure nothing.
// The cases cover the three lenses, three each, clean changes, and code
// outside the change.
func TestReviewerCasesPointRight(t *testing.T) {
	perLens, clean, outside := map[string]int{}, 0, 0
	for f, c := range loadCases(t) {
		if c.Review == nil {
			continue
		}
		var places []place
		for _, d := range append(slices.Clone(c.Review.Defects), c.Review.Outside...) {
			places = append(places, d.At...)
		}
		places = append(places, c.Review.Not...)
		for _, d := range c.Review.Defects {
			perLens[d.Lens]++
		}
		if len(c.Review.Defects) == 0 {
			clean++
		}
		if len(c.Review.Outside) > 0 {
			outside++
		}
		t.Run(c.Case, func(t *testing.T) {
			t.Parallel()
			_, repo, env, err := build(t, c)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range places {
				data, err := os.ReadFile(filepath.Join(repo, p.File))
				lines := strings.Split(string(data), "\n")
				if err != nil || len(p.Lines) != 2 || p.Lines[0] < 1 || p.Lines[1] < p.Lines[0] || p.Lines[1] > len(lines) {
					t.Fatalf("%s: %v is not a range of %s (%v)", f, p.Lines, p.File, err)
				}
				if !strings.Contains(strings.Join(lines[p.Lines[0]-1:p.Lines[1]], "\n"), p.Text) {
					t.Errorf("%s: %q is not within %s", f, p.Text, p)
				}
			}
			for _, cmd := range []string{"go vet ./...", "go test -count=1 ./..."} {
				if err := sh(repo, append(env, "GOFLAGS=-mod=mod", "GOTOOLCHAIN=local"), cmd); err != nil {
					t.Errorf("%s: %v", f, err)
				}
			}
		})
	}
	for _, lens := range []string{"correctness", "edge-cases", "tests"} {
		if perLens[lens] < 3 {
			t.Errorf("%d defects planted for the %s lens, fewer than three", perLens[lens], lens)
		}
	}
	if clean < 2 || outside < 1 {
		t.Errorf("%d clean changes, %d with code outside the change", clean, outside)
	}
}

// TestScoreReview scores canned reviews: a defect found, by another lens; a
// true finding the judge refused; findings nothing planted, shown or
// refused; a place that must stay clean; code outside the change.
func TestScoreReview(t *testing.T) {
	rc := &reviewCase{
		Defects: []plant{
			{Lens: "correctness", Kind: "off-by-one", At: []place{{File: "a.go", Lines: []int{10, 12}}}},
			{Lens: "tests", Kind: "untested", At: []place{{File: "a.go", Lines: []int{20, 30}}, {File: "a_test.go", Lines: []int{5, 9}}}},
		},
		Outside: []plant{{Lens: "tests", Kind: "untested", At: []place{{File: "b.go", Lines: []int{1, 9}}}}},
		Not:     []place{{File: "a.go", Lines: []int{40, 40}, Why: "it is documented"}},
	}
	rr := &reviewRun{
		Lenses: []string{"correctness", "edge-cases", "tests"}, Floor: 2,
		Answers: map[string]int{"correctness": 2, "edge-cases": 2, "tests": 1}, Unfounded: map[string]int{"tests": 1},
		TokensIn: map[string]int{"correctness": 5000, "edge-cases": 4000, "tests": 3000}, TokensOut: map[string]int{"correctness": 500},
		Findings: []lensFinding{
			{"edge-cases", "a.go:11", "important", "off by one", true, "yes"},        // the correctness defect, found by another lens
			{"correctness", "a.go:11", "important", "off by one", true, "yes"},       // merged on its line
			{"tests", "a_test.go:7", "important", "asserts nothing", true, "no"},     // the tests defect, refused: a judge false negative
			{"correctness", "a.go:40", "nit", "reads oddly", true, ""},               // a nit where nothing may be found
			{"edge-cases", "a.go:50", "important", "a guess", true, "no"},            // unplanted, refused
			{"edge-cases", "b.go:3", "important", "Snapshot untested", false, "yes"}, // outside, planted, opened
		},
	}
	passed, failed, measure, notes := scoreReview(rc, rr)
	if passed != 2 || len(failed) != 2 {
		t.Errorf("passed %d, failed %v", passed, failed)
	}
	for _, want := range []string{
		"floor=2",
		"correctness planted=1 found=1 own=1 judge-no=0 fp=0 nits=1 dropped=0 outside=0 opened=0 answers=2 unfounded=0 in=5000 out=500",
		"edge-cases planted=0 found=0 own=0 judge-no=0 fp=0 nits=0 dropped=1 outside=1 opened=1 answers=2",
		"tests planted=1 found=0 own=0 judge-no=1 fp=0 nits=0 dropped=0 outside=0 opened=0 answers=1 unfounded=1 in=3000",
		"outside-planted tests:untested=opened by edge-cases",
	} {
		if !strings.Contains(measure, want) {
			t.Errorf("measure lacks %q:\n%s", want, measure)
		}
	}
	if len(notes) != len(rr.Findings) {
		t.Errorf("a note a finding: %v", notes)
	}
	// Nothing shown that nothing planted is the clean point; an important
	// one shown loses it.
	rr.Findings = append(rr.Findings, lensFinding{"tests", "a.go:60", "important", "invented", true, "yes"})
	if _, failed, measure, _ := scoreReview(rc, rr); !slices.ContainsFunc(failed, func(f string) bool { return strings.HasPrefix(f, "clean:") }) || !strings.Contains(measure, "tests planted=1 found=0 own=0 judge-no=1 fp=1") {
		t.Errorf("an invented finding shown: %v\n%s", failed, measure)
	}
}

// TestReviewerWithFakeAgent plays a reviewer case through the engine with a
// fake agent (testdata/review-agent.sh): the run's folder is read back —
// each lens's answers, the quotes not found again, the judge's verdicts,
// the tokens each lens's call and its judges used — and scored. No real
// agent, no tokens.
func TestReviewerWithFakeAgent(t *testing.T) {
	agent, _ := filepath.Abs("testdata/review-agent.sh")
	answers, _ := filepath.Abs("testdata/review-answers")
	c := loadCases(t)["cases/reviewer/correctness-checkout-drops-the-coupon-error.yaml"]
	if c == nil {
		t.Fatal("no case correctness-checkout-drops-the-coupon-error")
	}
	spec := "cmd:sh " + agent + " " + answers
	t.Setenv("WORKLINE_JUDGE", spec)
	r, err := play(t, c, spec)
	if err != nil {
		t.Fatal(err)
	}
	passed, failed, _ := grade(c, r)
	t.Logf("%d/%d; %v\n%s\n%s", passed, passed+len(failed), failed, r.measure, strings.Join(r.reviewNotes, "\n"))
	if passed != 1 || len(failed) != 1 || !strings.HasPrefix(failed[0], "clean:") {
		t.Errorf("the defect found, an unplanted finding shown: passed %d, failed %v", passed, failed)
	}
	for _, want := range []string{
		"correctness planted=1 found=1 own=1 judge-no=0 fp=0 nits=0 dropped=1 outside=0 opened=0 answers=2 unfounded=0 in=2100 out=210",
		"edge-cases planted=0 found=0 own=0 judge-no=0 fp=1 nits=0 dropped=0 outside=1 opened=1 answers=2 unfounded=0 in=2100 out=210",
		"tests planted=0 found=0 own=0 judge-no=0 fp=0 nits=1 dropped=0 outside=0 opened=0 answers=2 unfounded=1 in=100 out=10",
	} {
		if !strings.Contains(r.measure, want) {
			t.Errorf("measure lacks %q:\n%s", want, r.measure)
		}
	}
	if !strings.HasPrefix(r.measure, "floor=") || strings.HasPrefix(r.measure, "floor=0") {
		t.Errorf("the finder floor is read from the lenses' task: %s", r.measure)
	}
	if r.judgedBy != "fake-judge (provider)" {
		t.Errorf("judged by %q", r.judgedBy)
	}
}
