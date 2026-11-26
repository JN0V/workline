package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestPointsCoverTheReplaySet: points.yaml states every defect the
// reviewer's replay cases plant, and nothing else, so a case added or a
// point renamed is never scored on a missing question.
func TestPointsCoverTheReplaySet(t *testing.T) {
	defs, err := loadPoints()
	if err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob("../replay/reviewer/*.yaml")
	if len(files) == 0 {
		t.Fatal("no reviewer replay case")
	}
	seen := map[string]bool{}
	for _, f := range files {
		var c struct {
			Review struct {
				Defects []struct{ Kind string }
			}
		}
		data, err := os.ReadFile(f)
		if err == nil {
			err = yaml.Unmarshal(data, &c)
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".yaml")
		seen[name] = true
		def, ok := defs[name]
		if !ok || def.Change == "" {
			t.Errorf("%s: no entry in points.yaml, or no change", name)
			continue
		}
		var planted, stated []string
		for _, d := range c.Review.Defects {
			planted = append(planted, d.Kind)
		}
		for _, p := range def.Points {
			stated = append(stated, p.Name)
			if p.Says == "" || p.Not == "" {
				t.Errorf("%s %s: says what the point is, and what does not count", name, p.Name)
			}
			if p.Class != "kept" && !slices.Contains(classes, p.Class) {
				t.Errorf("%s %s: class %q", name, p.Name, p.Class)
			}
		}
		if !slices.Equal(planted, stated) {
			t.Errorf("%s: the case plants %v, points.yaml states %v", name, planted, stated)
		}
	}
	for name := range defs {
		if !seen[name] {
			t.Errorf("points.yaml: %s is no reviewer replay case", name)
		}
	}
}

var (
	twoPoints = caseDef{Change: "Adds a drain.", Points: []point{
		{Name: "cap", Class: "out-of-diff", Says: "one poll handles 8 events", Not: "no test"},
		{Name: "untested", Class: "weak-test", Says: "the drain has no test", Not: "the cap"},
		{Name: "unsaid", Class: "in-diff", Out: "ambiguous", Says: "the change is unannounced", Not: "the cap"},
	}}
	fourFindings = []finding{
		{Place: "a.h:9", Severity: "important", Title: "Drain capped at 8", Body: "poll() stops at 8."},
		{Place: "#139", Severity: "important", Refused: true, Title: "No test of the drain"},
		{Place: "a.h:3", Severity: "nit", Title: "A name is long"},
		{Place: "b.h:1", Severity: "important", Title: "Unrelated"},
	}
)

// TestJudgeIsBlind: the judge's question holds the points and each
// finding's place and words; never the arm, its model, nor what the run
// made of a finding (its severity, the engine's judge).
func TestJudgeIsBlind(t *testing.T) {
	prompt := pointSystem + pointPrompt(twoPoints, fourFindings)
	for _, want := range []string{"## cap", "one poll handles 8 events", "Does not count: no test", "## F1, at a.h:9", "poll() stops at 8.", "## F4, at b.h:1"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the question lacks %q:\n%s", want, prompt)
		}
	}
	cr := caseRun{Arm: "free-opus", Case: "dc-260", Findings: fourFindings}
	for _, never := range []string{cr.Arm, "opus", "sonnet", "important", "refused", "nit\n", "severity"} {
		if strings.Contains(strings.ToLower(pointPrompt(twoPoints, cr.Findings)), never) {
			t.Errorf("the question holds %q", never)
		}
	}
}

// TestVerdict: a point is said by an Important shown; one the engine's
// judge dropped, or a nit, is counted apart; partly counts from an
// Important shown only.
func TestVerdict(t *testing.T) {
	for _, c := range []struct {
		said, partly []int
		want         string
	}{
		{[]int{0}, nil, "said"},
		{[]int{1}, nil, "refused"},
		{[]int{2}, nil, "nit"},
		{[]int{2, 1}, nil, "refused"},
		{[]int{1}, []int{3}, "partly"},
		{nil, []int{1, 2}, "not said"},
		{nil, nil, "not said"},
		{[]int{2, 0}, []int{3}, "said"},
	} {
		if got := verdict(fourFindings, pointAnswer{Said: c.said, Partly: c.partly}); got != c.want {
			t.Errorf("said %v, partly %v: %q, want %q", c.said, c.partly, got, c.want)
		}
	}
	if got := settle(pointRank, []string{"said", "said"}); got != "said" {
		t.Errorf("two alike: %q", got)
	}
	if got := settle(pointRank, []string{"said", "not said", "not said"}); got != "not said" {
		t.Errorf("a majority: %q", got)
	}
	if got := settle(pointRank, []string{"said", "not said", "partly"}); got != "partly" {
		t.Errorf("three apart, the middle one: %q", got)
	}
	if got := settle(cleanRank, []string{"nit", "false alarm", "false alarm"}); got != "false alarm" {
		t.Errorf("clean: %q", got)
	}
}

// scripted answers each question from a list, by the call's number for
// that question, and counts the calls.
type scripted struct {
	mu      sync.Mutex
	answers map[string][]string // a text the question holds → its answers, in order
	asked   map[string]int
	dirs    []string
}

func (s *scripted) ask(system, user, dir string) (string, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirs = append(s.dirs, dir)
	for key, answers := range s.answers {
		if strings.Contains(user, key) {
			n := s.asked[key]
			s.asked[key]++
			return answers[min(n, len(answers)-1)], 100, nil
		}
	}
	return "", 0, fmt.Errorf("no answer scripted for:\n%s", user)
}

func answer(cap, untested string) string {
	return fmt.Sprintf(`Here it is: {"points": {"cap": %s, "untested": %s, "unsaid": {"said": [], "partly": []}}}`, cap, untested)
}

const none = `{"said": [], "partly": []}`

// TestScore: each case run is judged twice, a third time when the two
// differ; a run with no finding is not asked; an answer that does not
// read is asked again; a second scoring reads the judgements kept.
func TestScore(t *testing.T) {
	defs := map[string]caseDef{"dc-260": twoPoints, "kept-x": {Change: "c", Points: twoPoints.Points[:1]}}
	agree := caseRun{Arm: "lens", Run: 1, Case: "dc-260", TokensIn: 900, TokensOut: 100, Findings: fourFindings}
	differ := caseRun{Arm: "free", Run: 1, Case: "dc-260", Findings: fourFindings[:2]}
	empty := caseRun{Arm: "free", Run: 1, Case: "kept-x"}
	s := &scripted{asked: map[string]int{}, answers: map[string][]string{
		"## F4": {answer(`{"said": ["F1"], "partly": [], "quote": "poll() stops at 8."}`, `{"said": ["F2"], "partly": []}`)},
		"(2)": {
			`{"points": {"cap": {"said": ["F9"]}}}`, // names a finding the run has not, and not every point: asked again
			answer(`{"said": ["F1"], "partly": []}`, none),
			answer(none, none),
			answer(`{"said": [], "partly": ["F1"]}`, none),
		},
	}}
	j := &judge{ask: s.ask, cache: t.TempDir()}
	runs := []caseRun{agree, differ, empty}
	js, err := score(j, defs, runs, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.asked["## F4"] != 2 || s.asked["(2)"] != 4 {
		t.Errorf("asked %v: twice where the two agree; a bad answer again, then a third on a difference", s.asked)
	}
	rows := tsv(js)
	for _, want := range []string{
		"lens\t1\tdc-260\tmiss\tcap\tout-of-diff\tyes\tsaid\tsaid / said\tF1; \tpoll() stops at 8.",
		"lens\t1\tdc-260\tmiss\tuntested\tweak-test\tyes\trefused\trefused / refused\tF2; ",
		"lens\t1\tdc-260\tmiss\tunsaid\tin-diff\tambiguous\tnot said",
		"free\t1\tdc-260\tmiss\tcap\tout-of-diff\tyes\tpartly\tsaid / not said / partly\t; F1",
		"free\t1\tkept-x\tkept\tcap\tout-of-diff\tyes\tnot said\t\t",
	} {
		if !strings.Contains(rows, want) {
			t.Errorf("scores.tsv lacks %q:\n%s", want, rows)
		}
	}
	if js[0].shown != 2 || js[0].extra != 1 {
		t.Errorf("the lens run shows 2 Importants, 1 on no point: %d, %d", js[0].shown, js[0].extra)
	}
	sum := summary(runs, js, "fake")
	for _, want := range []string{
		"| lens | 1 | 2 | 1 | 1.0 | 0.0 | 0.0 | 1.0 | 2.0 | 0.0 |",
		"| free | 1 | 2 | 0 | 0.0 | 0.0 | 1.0 | 0.0 | 0.0 | 0.0 |",
		"| lens | 1/1 (100 %) | 0/1 (0 %) | 0/0 (0 %) | 0/0 (0 %) | 2.0 | 1.0 |",
		"| free | 0 of 1 | 0 | 0 | 0 | 0 | 0k |",
		"Judged twice: 6 items; the two agree on 5 (83 %); a third judgement settled 3.",
	} {
		if !strings.Contains(sum, want) {
			t.Errorf("summary.md lacks %q:\n%s", want, sum)
		}
	}
	again, err := score(j, defs, runs, 2, nil)
	if err != nil || tsv(again) != rows {
		t.Errorf("scored again from the judgements kept: %v\n%s", err, tsv(again))
	}
	if s.asked["## F4"] != 2 || s.asked["(2)"] != 4 {
		t.Errorf("a second scoring asked the judge again: %v", s.asked)
	}
	// A case points.yaml does not hold: nobody is asked, the rows are empty
	// ones the summary and the table still read.
	unknown := append(slices.Clone(runs), caseRun{Arm: "a", Case: "unknown"})
	js, err = score(&judge{ask: s.ask, cache: t.TempDir()}, defs, unknown, 1, nil)
	if err == nil || !strings.Contains(err.Error(), `no case "unknown"`) || len(js) != len(unknown) {
		t.Errorf("an unknown case: %v, %d judgements", err, len(js))
	}
	if s.asked["## F4"] != 2 || tsv(js) != "arm\trun\tcase\tset\titem\tclass\tscored\tverdict\tjudgements\tfindings\tquote\twhy\n" || summary(unknown, js, "fake") == "" {
		t.Errorf("an unknown case asked the judge, or left rows: %v\n%s", s.asked, tsv(js))
	}
}

// TestClean: on a clean case the judge is asked, in the case's tree, about
// the Importants shown and nothing else; none shown, nobody is asked.
func TestClean(t *testing.T) {
	defs := map[string]caseDef{"clean-x": {Change: "Renames a parameter."}}
	s := &scripted{asked: map[string]int{}, answers: map[string][]string{"(2)": {
		`{"findings": {"F1": {"verdict": "false alarm", "why": "the test exists"}, "F4": {"verdict": "true defect", "why": "no test"}}}`,
		`{"findings": {"F1": {"verdict": "false alarm", "why": "the test exists"}, "F4": {"verdict": "nit", "why": "small"}}}`,
		`{"findings": {"F1": {"verdict": "false alarm", "why": "the test exists"}, "F4": {"verdict": "nit", "why": "small"}}}`,
	}}}
	j := &judge{ask: s.ask, cache: t.TempDir(), tree: func(kase string) (string, string, error) {
		return "/tree/of/" + kase, "## Issue #4\n\nthe issue\n", nil
	}}
	runs := []caseRun{{Arm: "lens", Run: 1, Case: "clean-x", Findings: fourFindings}, {Arm: "free", Run: 1, Case: "clean-x", Findings: fourFindings[1:3]}}
	js, err := score(j, defs, runs, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.asked["(2)"] != 3 || !slices.Equal(s.dirs, []string{"/tree/of/clean-x", "/tree/of/clean-x", "/tree/of/clean-x"}) {
		t.Errorf("asked %v in %v: the lens run alone, three times, in the case's tree", s.asked, s.dirs)
	}
	rows := tsv(js)
	for _, want := range []string{
		"lens\t1\tclean-x\tclean\tF1\timportant\tyes\tfalse alarm\tfalse alarm / false alarm / false alarm\tF1\tDrain capped at 8 poll() stops at 8.\tthe test exists",
		"lens\t1\tclean-x\tclean\tF4\timportant\tyes\tnit\ttrue defect / nit / nit\tF4\tUnrelated\tsmall",
	} {
		if !strings.Contains(rows, want) {
			t.Errorf("scores.tsv lacks %q:\n%s", want, rows)
		}
	}
	if strings.Contains(rows, "free") {
		t.Errorf("a run with no Important shown has no row:\n%s", rows)
	}
	if sum := summary(runs, js, "fake"); !strings.Contains(sum, "| lens | 0 of 0 | 2 | 1 | 1 | 0 | 0k |") || !strings.Contains(sum, "| free | 0 of 0 | 0 | 0 | 0 | 0 | 0k |") {
		t.Errorf("summary.md:\n%s", sum)
	}
}

// TestReadFindings: a run with no number is the arm's n-th of its case.
func TestReadFindings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runs.jsonl")
	lines := `{"arm":"a","case":"reviewer/dc-1","tokens_in":5,"findings":[{"place":"x:1","severity":"important","title":"t","body":"b"}]}
{"arm":"a","case":"dc-2","findings":[]}

{"arm":"a","case":"dc-1","findings":[]}
{"arm":"b","run":7,"case":"dc-1","findings":[]}
`
	if err := os.WriteFile(file, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	runs, err := readFindings(file)
	if err != nil || len(runs) != 4 {
		t.Fatal(err, runs)
	}
	got := fmt.Sprintf("%s %d %d %v %d %d %d", runs[0].Case, runs[0].Run, runs[0].TokensIn, runs[0].Findings[0].shown(), runs[1].Run, runs[2].Run, runs[3].Run)
	if got != "dc-1 1 5 true 1 2 7" {
		t.Errorf("read %s", got)
	}
	if err := os.WriteFile(file, []byte("{}\n\n{bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readFindings(file); err == nil || !strings.Contains(err.Error(), "line 3:") {
		t.Errorf("a bad line is named by its number in the file: %v", err)
	}
	if judgeDir.ReplaceAllString("claude:opus@medium", "-") != "claude-opus-medium" {
		t.Error("the judgements kept are filed under their judge")
	}
	if _, err := askerOf("opus"); err == nil {
		t.Error("a judge that is neither claude:<model>@<effort> nor cmd: is refused")
	}
	ask, err := askerOf("cmd:cat")
	if err != nil {
		t.Fatal(err)
	}
	if out, _, err := ask("system", "user", ""); err != nil || out != "system\n\nuser" {
		t.Errorf("a command is given the question on its stdin: %q, %v", out, err)
	}
}
