package evaluation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
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

// TestLinesMax: lines-max counts what the engine's doc-lines budget counts,
// the body below the header (#454).
func TestLinesMax(t *testing.T) {
	r := &run{repo: t.TempDir()}
	doc := func(body int) {
		text := "---\nsources: [internal/x.go]\nchecked: abc1234\n---\n" + strings.Repeat("A line.\n", body)
		if err := os.MkdirAll(filepath.Join(r.repo, "docs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(r.repo, "docs", "x.md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	max := map[string]any{"docs/x.md": 200}
	doc(199)
	if why := check("lines-max", max, &caseFile{}, r); why != "" {
		t.Errorf("4 header lines and 199 below them are within 200: %s", why)
	}
	doc(201)
	if why := check("lines-max", max, &caseFile{}, r); !strings.Contains(why, "201 lines") {
		t.Errorf("201 lines below the header are over 200, and said: %q", why)
	}
}

// TestBodyUnchanged: a `checked` moved in a README's `<!-- workline` header
// is no rewording, as in a frontmatter.
func TestBodyUnchanged(t *testing.T) {
	r := &run{repo: t.TempDir(), before: map[string]string{}}
	for _, head := range []string{"---\nchecked: %s\n---\n", "<!-- workline\nchecked: %s\n-->\n"} {
		r.before["README.md"] = fmt.Sprintf(head, "abc1234") + "# Title\n\nSame words.\n"
		now := fmt.Sprintf(head, "def5678") + "# Title\n\nSame words.\n"
		if err := os.WriteFile(filepath.Join(r.repo, "README.md"), []byte(now), 0o644); err != nil {
			t.Fatal(err)
		}
		if why := check("body-unchanged", "README.md", &caseFile{}, r); why != "" {
			t.Errorf("%q: %s", head, why)
		}
		if err := os.WriteFile(filepath.Join(r.repo, "README.md"), []byte(now+"More.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if why := check("body-unchanged", "README.md", &caseFile{}, r); why == "" {
			t.Errorf("%q: a line added is a rewording", head)
		}
	}
}

func TestAnsweredBy(t *testing.T) {
	m, e, in, out, c := answeredBy([]call{{"low", "claude-haiku-4-5", 1000, 50, 0.01, "part", "1-lenses"}, {"low", "claude-sonnet-5", 2000, 70, 0.02, "judge", "judge/01"}})
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
	"finding": true, "no-finding": true, "judge": true, "sh": true}

func loadCases(t *testing.T) map[string]*caseFile {
	files, _ := filepath.Glob("cases/*/*.yaml")
	cases := map[string]*caseFile{}
	for _, f := range files {
		c, err := readCase(f)
		if err != nil {
			t.Fatal(err)
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
// keeps them and loses the control; a careful one keeps both, and so does
// one that also opens an issue for the stale code comment, as the role's
// rules ask (#439). No real agent, no tokens.
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
		for _, mode := range []string{"vouches", "judges", "careful", "reports"} {
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
				if mode == "reports" && !slices.Contains(r.res.Applied, "issue") {
					t.Errorf("%s: the issue was not opened: applied %v, findings %+v", f, r.res.Applied, r.res.Findings)
				}
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

// TestDeveloperWithStandIns plays the developer's cases with stand-in
// agents (testdata/developer-agent.sh): a careful one earns every point; a
// careless one — a frozen test weakened, a file out of scope fixed, a test
// gamed rather than objected to — loses some on each case it codes. The
// stand-in's tokens in are the prompt's size, a character in four: the
// measure taken before a real agent is paid. No real agent, no tokens.
func TestDeveloperWithStandIns(t *testing.T) {
	agent, _ := filepath.Abs("testdata/developer-agent.sh")
	n := 0
	for f, c := range loadCases(t) {
		if c.Run.Role != "developer" {
			continue
		}
		n++
		for _, mode := range []string{"careful", "careless"} {
			t.Run(c.Case+"/"+mode, func(t *testing.T) {
				r, err := play(t, c, "cmd:sh "+agent+" "+mode)
				if err != nil {
					t.Fatal(err)
				}
				passed, failed, _ := grade(c, r)
				_, _, tokensIn, _, _ := answeredBy(r.res.Calls)
				t.Logf("%d/%d, prompt ~%s tokens; lost: %s", passed, passed+len(failed), tokensIn, strings.Join(failed, "; "))
				coded := r.res.AgentCalls > 0
				switch {
				case mode == "careful" && len(failed) > 0:
					t.Errorf("%s: the careful stand-in lost %v", f, failed)
				case mode == "careless" && coded && len(failed) == 0:
					t.Errorf("%s: the careless stand-in lost nothing", f)
				}
			})
		}
	}
	if n < 4 {
		t.Errorf("%d developer cases", n)
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
// pass — a planted defect the tests already catch would measure nothing;
// an issue a place names holds its text. The cases cover the three code
// lenses, three each, intent and claims, clean changes, and code outside
// the change.
func TestReviewerCasesPointRight(t *testing.T) {
	perLens, clean, outside := map[string]int{}, 0, 0
	for f, c := range loadCases(t) {
		if c.Review == nil {
			continue
		}
		var places []place
		for _, d := range append(slices.Clone(c.Review.Defects), c.Review.Outside...) {
			if len(d.At) == 0 {
				t.Errorf("%s: the %s %s defect names no place", f, d.Lens, d.Kind)
			}
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
				if p.issue() { // an issue the change closes: its body holds the text
					if body := issueBody(c, p.File); !strings.Contains(body, p.Text) || p.Text == "" {
						t.Errorf("%s: %q is not in %s's body, in given.forge", f, p.Text, p.File)
					}
					continue
				}
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
	for lens, least := range map[string]int{"correctness": 3, "edge-cases": 3, "tests": 3, "intent": 1, "claims": 1} {
		if perLens[lens] < least {
			t.Errorf("%d defects planted for the %s lens, fewer than %d", perLens[lens], lens, least)
		}
	}
	if clean < 2 || outside < 1 {
		t.Errorf("%d clean changes, %d with code outside the change", clean, outside)
	}
}

// TestIntentAndClaimsCasesGiveTheirLens plays each case planting an intent
// or a claims defect through the engine, a fake agent finding nothing:
// the lenses' call asks that lens, and holds what it checks the change
// against — the issue's words the defect stands on, read from the case's
// forge, or what the author says. No real agent, no tokens.
func TestIntentAndClaimsCasesGiveTheirLens(t *testing.T) {
	agent, _ := filepath.Abs("testdata/review-agent.sh")
	spec := "cmd:sh " + agent + " " + t.TempDir()
	t.Setenv("WORKLINE_JUDGE", spec)
	t.Setenv("WORKLINE_EVAL_REVIEWER", "")
	t.Setenv("WORKLINE_EVAL_LENSES", "")
	asked := map[string]int{}
	for f, c := range loadCases(t) {
		if c.Review == nil {
			continue
		}
		var want []string
		for _, d := range c.Review.Defects {
			switch d.Lens {
			case "intent":
				want = append(want, "## Lens: intent", d.At[0].Text)
			case "claims":
				want = append(want, "## Lens: claims", "## What the author says")
			default:
				continue
			}
			asked[d.Lens]++
		}
		if len(want) == 0 {
			continue
		}
		r, err := play(t, c, spec)
		if err != nil {
			t.Fatal(err)
		}
		task, err := os.ReadFile(filepath.Join(r.res.RunDir, "in", "parts", "1-lenses", "task.md"))
		if err != nil {
			t.Fatalf("%s: the lenses' call: %v", f, err)
		}
		for _, w := range want {
			if !strings.Contains(string(task), w) {
				t.Errorf("%s: the lenses' call lacks %q", f, w)
			}
		}
	}
	if asked["intent"] == 0 || asked["claims"] == 0 {
		t.Errorf("no case plants an intent or a claims defect: %v", asked)
	}
}

// issueBody is the body of the issue ref (`#4`) in a case's simulated
// forge, "" when it holds none.
func issueBody(c *caseFile, ref string) string {
	issues, _ := c.Given.Forge["issues"].([]any)
	for _, i := range issues {
		is, _ := i.(map[string]any)
		if fmt.Sprintf("#%v", is["id"]) == ref {
			return fmt.Sprint(is["body"])
		}
	}
	return ""
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
// the tokens each lens's call and its judges used, what a block would have
// held — and scored, the lenses asked apart and together. No real agent,
// no tokens.
func TestReviewerWithFakeAgent(t *testing.T) {
	agent, _ := filepath.Abs("testdata/review-agent.sh")
	answers, _ := filepath.Abs("testdata/review-answers")
	c := loadCases(t)["cases/reviewer/correctness-checkout-drops-the-coupon-error.yaml"]
	if c == nil {
		t.Fatal("no case correctness-checkout-drops-the-coupon-error")
	}
	spec := "cmd:sh " + agent + " " + answers
	t.Setenv("WORKLINE_JUDGE", spec)
	for _, run := range []struct {
		settings string
		want     []string
	}{
		{"{lenses-together: false}", []string{ // each lens a call of its own: its tokens, and its judges'
			"correctness planted=1 found=1 own=1 judge-no=0 fp=0 nits=0 dropped=1 outside=0 opened=0 answers=2 unfounded=0 in=2100 out=210 own-verified=1 fp-verified=0 unsure=0 questions=0 judges=2",
			"edge-cases planted=0 found=0 own=0 judge-no=0 fp=1 nits=0 dropped=0 outside=1 opened=1 answers=2 unfounded=0 in=2100 out=210 own-verified=0 fp-verified=1 unsure=0 questions=0 judges=2",
			"tests planted=0 found=0 own=0 judge-no=0 fp=0 nits=1 dropped=0 outside=0 opened=0 answers=2 unfounded=1 in=100 out=10 own-verified=0 fp-verified=0 unsure=0 questions=0 judges=0",
		}},
		{"", []string{ // the role's own way: one call, its answers and tokens the `lenses` part's
			"lenses planted=0 found=0 own=0 judge-no=0 fp=0 nits=0 dropped=0 outside=0 opened=0 answers=6 unfounded=0 in=100 out=10",
			"correctness planted=1 found=1 own=1 judge-no=0 fp=0 nits=0 dropped=1 outside=0 opened=0 answers=0 unfounded=0 in=2000 out=200 own-verified=1 fp-verified=0 unsure=0 questions=0 judges=2",
			"edge-cases planted=0 found=0 own=0 judge-no=0 fp=1 nits=0 dropped=0 outside=1 opened=1 answers=0 unfounded=0 in=2000 out=200 own-verified=0 fp-verified=1",
			"tests planted=0 found=0 own=0 judge-no=0 fp=0 nits=1 dropped=0 outside=0 opened=0 answers=0 unfounded=1 in=0 out=0",
		}},
	} {
		t.Setenv("WORKLINE_EVAL_REVIEWER", run.settings)
		r, err := play(t, c, spec)
		if err != nil {
			t.Fatal(err)
		}
		passed, failed, _ := grade(c, r)
		t.Logf("%s: %d/%d; %v\n%s\n%s", run.settings, passed, passed+len(failed), failed, r.measure, strings.Join(r.reviewNotes, "\n"))
		if passed != 1 || len(failed) != 1 || !strings.HasPrefix(failed[0], "clean:") {
			t.Errorf("%s: the defect found, an unplanted finding shown: passed %d, failed %v", run.settings, passed, failed)
		}
		for _, want := range run.want {
			if !strings.Contains(r.measure, want) {
				t.Errorf("%s: measure lacks %q:\n%s", run.settings, want, r.measure)
			}
		}
		if !strings.HasPrefix(r.measure, "floor=") || strings.HasPrefix(r.measure, "floor=0") {
			t.Errorf("the finder floor is read from the lenses' task: %s", r.measure)
		}
		if r.judgedBy != "fake-judge (provider)" {
			t.Errorf("judged by %q", r.judgedBy)
		}
	}
}

// TestJudgeReplaySet reads the judge's replay set with no agent: each of
// the 40 verdicts measured holds its question, its answer then and its
// truth, and scoring what the judge said then gives back the measure's
// numbers (docs/research/judge-measured.md).
func TestJudgeReplaySet(t *testing.T) {
	items, err := readReplay(replayDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 40 {
		t.Fatalf("%d items; the measure drew 40", len(items))
	}
	if _, err := readReplay(t.TempDir()); err == nil {
		t.Error("a set with no item is an error, never a replay of nothing")
	}
	lenses := []string{"correctness", "edge-cases", "tests", "intent", "claims"}
	then := make([]replayAnswer, len(items))
	var matters []int
	for i, it := range items {
		switch {
		case it.Question == "" || it.Material == "":
			t.Errorf("%s: a question and its material are needed", it.Name)
		case !strings.Contains(it.Material, it.Title):
			t.Errorf("%s: the material does not hold the finding %q", it.Name, it.Title)
		case it.Said != "yes" && it.Said != "no", it.Then.Yes == nil, *it.Then.Yes != (it.Said == "yes"), it.Then.Why == "":
			t.Errorf("%s: said %q, its answer.yaml %+v", it.Name, it.Said, it.Then)
		case it.Origin != "artifact" && it.Origin != "rebuilt":
			t.Errorf("%s: origin %q, artifact or rebuilt", it.Name, it.Origin)
		case !slices.Contains(lenses, it.Lens), len(it.Commit) != 40, !strings.Contains(it.Where, ":"):
			t.Errorf("%s: lens %q, commit %q, where %q", it.Name, it.Lens, it.Commit, it.Where)
		case (it.SettledBy == "panel") != (it.Matters != nil), it.SettledBy != "panel" && it.SettledBy != "agreement", it.Evidence == "":
			t.Errorf("%s: settled by %q, matters %v, evidence %q", it.Name, it.SettledBy, it.Matters, it.Evidence)
		}
		then[i] = asJudgedThen(it)
		if it.Matters != nil && *it.Matters && it.Truth {
			matters = append(matters, it.PR)
		}
	}
	got := tallyReplay(items, then)
	want := replayTally{Items: 40, Right: 31, YesThen: 20, YesThenRight: 19, NoThen: 20, NoThenRight: 12, DroppedThen: 8, StillDropped: got.StillDropped,
		FalseThen: 12, True: 27, TrueDropped: 8, False: 13, FalseKept: 1, Verified: 20, VerifiedTrue: 19, SameAsThen: 40}
	if !reflect.DeepEqual(got, want) || len(got.StillDropped) != 8 {
		t.Errorf("what the judge said then, against the truth:\n got %s\nwant %s", got, want)
	}
	if !slices.Equal(matters, []int{238, 280}) {
		t.Errorf("true findings dropped that matter: %v, the measure's are #238 and #280", matters)
	}
}

// TestJudgeReplayWithFakeJudge asks the 40 questions as TestJudgeReplay
// does, of a fake judge (a cmd:) saying no to all: each false finding a
// point, every true one dropped, its tokens counted. No real agent.
func TestJudgeReplayWithFakeJudge(t *testing.T) {
	items, err := readReplay(replayDir)
	if err != nil {
		t.Fatal(err)
	}
	judgeRole, err := role.Load("../../roles", "judge")
	if err != nil {
		t.Fatal(err)
	}
	no := `cmd:cat > /dev/null; printf 'model: fake-judge\ntokens-in: 1000\ntokens-out: 10\n' > "$WORKLINE_CALL"; echo '- note: "no: the code handles it"'`
	now := make([]replayAnswer, len(items))
	for i, it := range items {
		now[i] = askReplay(no, judgeRole, it, &tokenStop{})
	}
	got := tallyReplay(items, now)
	want := replayTally{Items: 40, Right: 13, YesThen: 20, YesThenRight: 1, NoThen: 20, NoThenRight: 12, DroppedThen: 8, StillDropped: got.StillDropped,
		FalseThen: 12, True: 27, TrueDropped: 27, False: 13, SameAsThen: 20, Calls: 40, TokensIn: 40000, TokensOut: 400}
	if !reflect.DeepEqual(got, want) || len(got.StillDropped) != 8 {
		t.Errorf("a judge saying no to all:\n got %s\nwant %s", got, want)
	}
	var truth replayItem
	for _, it := range items {
		if it.Truth {
			truth = it
			break
		}
	}
	if a := askReplay(no, judgeRole, truth, &tokenStop{}); a.right(truth) || replayFailed(truth, a) != "said no ("+truth.Said+" then) to a true finding, dropped: the code handles it" {
		t.Errorf("a no to a true finding loses its point, said: %q", replayFailed(truth, a))
	}
	a := askReplay(`cmd:cat > /dev/null; echo '- note: "maybe"'`, judgeRole, truth, &tokenStop{})
	if a.Said != "" || a.right(truth) || !strings.HasPrefix(replayFailed(truth, a), "no answer: ") {
		t.Errorf("neither yes nor no is no answer, and no point: %+v", a)
	}
	// The replay stops asking past its tokens, earlier runs counted.
	stop := &tokenStop{max: 2500, inTo: 500}
	for i := range 3 {
		now[i] = askReplay(no, judgeRole, items[i], stop)
	}
	if !stop.stopped || stop.spent() != 2500 || now[2].Said != "" || len(now[2].Calls) != 0 || !strings.Contains(now[2].Shown, "not asked: the replay spent 2500 tokens in, its stop at 2500") {
		t.Errorf("past the stop, nothing asked: %d spent, %+v", stop.spent(), now[2])
	}
}

// TestJudgeReplayReadingWithFakeJudge asks as the judge that reads does
// (WORKLINE_EVAL_REPLAY_ROUNDS), of a fake judge (a cmd:): it reads a
// file at the item's commit, then says no, quoting a line it read: the
// finding dropped, the read counted. No real agent.
func TestJudgeReplayReadingWithFakeJudge(t *testing.T) {
	items, err := readReplay(replayDir)
	if err != nil {
		t.Fatal(err)
	}
	judgeRole, err := role.Load("../../roles", "judge")
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := sh(repo, hermetic(), "git init -q && printf 'one\\ntwo\\nthree\\n' > notes.txt && git add . && git commit -qm start"); err != nil {
		t.Fatal(err)
	}
	head, _ := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	fake := `cmd:p=$(cat); printf 'model: fake-judge\ntokens-in: 1000\ntokens-out: 10\n' > "$WORKLINE_CALL"; case $p in
*'What you read, round 1'*) printf '%s\n' '- note: "no: the file says two"' '- quote: {path: notes.txt, text: "two"}' ;;
*) echo '- read: {path: notes.txt, lines: 1-2, why: the notes}' ;;
esac`
	it := items[0]
	it.Commit = strings.TrimSpace(string(head))
	a := askReading(fake, judgeRole, it, repo, 2, 400, &tokenStop{})
	if a.Shown != "dropped" || a.Quoted != "notes.txt:2" || a.Served != 1 || a.Lines != 2 || len(a.Calls) != 2 || !slices.Equal(a.Reads, []string{"notes.txt:1-2"}) {
		t.Errorf("read, then a no quoting what it read: %+v", a)
	}
	got := tallyReplay([]replayItem{it}, []replayAnswer{a})
	if got.Reads != 1 || got.Rounds != 1 || got.Lines != 2 || got.Calls != 2 || got.TokensIn != 2000 {
		t.Errorf("the reads and calls counted: %s", got)
	}
}
