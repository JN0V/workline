package evaluation

// The reviewer's measure (#90): a case plants defects on a change, each at a
// file and a range of lines, and says where nothing may be found. The run is
// read from its folder and scored with no agent: a planted defect is found
// when a finding shown on the change has its cause in the range, the quote
// the engine found again saying where (docs/spec/conformance.md).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/builtin/reviewer"
	"github.com/JN0V/workline/internal/intent"
)

// reviewCase is what a reviewer case plants and forbids.
type reviewCase struct {
	Defects []plant `yaml:"defects"` // on the change: each a point when found
	Outside []plant `yaml:"outside"` // before the change: measured, not scored
	Not     []place `yaml:"not"`     // nothing may be found there: a point each
}

// plant is one defect planted: the lens it is for, its kind, and where its
// cause may be quoted — any of the places.
type plant struct {
	Lens string  `yaml:"lens"`
	Kind string  `yaml:"kind"`
	At   []place `yaml:"at"`
}

// place is a file and a range of lines, as they read at the change's head;
// text is a part of one of them, which TestReviewerCasesPointRight finds
// there, so a range never drifts from what it names.
type place struct {
	File  string `yaml:"file"`
	Lines []int  `yaml:"lines"`
	Text  string `yaml:"text"`
	Why   string `yaml:"why"`
}

func (p place) holds(where string) bool {
	file, line, ok := strings.Cut(where, ":")
	n, err := strconv.Atoi(line)
	return ok && err == nil && file == p.File && len(p.Lines) == 2 && n >= p.Lines[0] && n <= p.Lines[1]
}

func (p place) String() string {
	return fmt.Sprintf("%s:%d-%d", p.File, p.Lines[0], p.Lines[1])
}

// lensFinding is one finding a lens raised, once its quotes were found
// again, and what became of it.
type lensFinding struct {
	Lens, Where, Severity, Title string
	Related                      bool   // its cause lies in the change
	Judged                       string // yes, no, or "" when not judged (a nit) or not answered
}

// shown: the author sees it — on the change, unless the judge said no.
func (f lensFinding) shown() bool { return f.Related && f.Judged != "no" }

// reviewRun is what one review did, read from its run folder.
type reviewRun struct {
	Lenses    []string
	Floor     int            // candidates each lens was asked to look for (finder-floor)
	Answers   map[string]int // findings each lens answered, before any was checked
	Unfounded map[string]int // dropped: a quote not found again
	Findings  []lensFinding
	TokensIn  map[string]int // each lens's own call, and its findings' judges
	TokensOut map[string]int
	Judge     string // the judge's model and the independence it reached
}

var (
	floorRe     = regexp.MustCompile(`Look for at least (\d+) candidates`)
	unfoundedRe = regexp.MustCompile(`^dropped, from the (\S+) lens:`)
)

// readReview reads a reviewer run's folder; res gives its calls, in order,
// and the findings it logged.
func readReview(runDir string, res result) (*reviewRun, error) {
	rr := &reviewRun{Answers: map[string]int{}, Unfounded: map[string]int{}, TokensIn: map[string]int{}, TokensOut: map[string]int{}}
	parts, _ := filepath.Glob(filepath.Join(runDir, "in", "parts", "*", "task.md"))
	sort.Strings(parts)
	askedAgain := map[string]bool{}
	for _, f := range res.Findings {
		if f.Rule == "part-asked-again" {
			askedAgain[f.Where] = true
		}
		if m := unfoundedRe.FindStringSubmatch(f.Message); f.Rule == "finding-unfounded" && m != nil {
			rr.Unfounded[m[1]]++
		}
	}
	// Part calls come in the order of the parts, one each, two when asked
	// again, none when not asked (the agent gone, the tokens spent).
	var partCalls []string
	for _, task := range parts {
		dir := filepath.Dir(task)
		name := filepath.Base(dir)
		_, lens, _ := strings.Cut(name, "-")
		rr.Lenses = append(rr.Lenses, lens)
		data, _ := os.ReadFile(task)
		if m := floorRe.FindSubmatch(data); m != nil {
			rr.Floor, _ = strconv.Atoi(string(m[1]))
		}
		if answers, err := intent.Read(filepath.Join(dir, "answer.yaml")); err == nil {
			rr.Answers[lens] = len(answers)
		}
		if why, err := os.ReadFile(filepath.Join(dir, "unanswered")); err == nil && strings.Contains(string(why), "not asked") {
			continue
		}
		partCalls = append(partCalls, lens)
		if askedAgain[name] {
			partCalls = append(partCalls, lens)
		}
	}
	var c struct {
		Related, Outside []reviewer.Finding
		Asked            []string
	}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "candidates.json"))
	if err == nil {
		err = json.Unmarshal(data, &c)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("candidates: %v", err)
	}
	keys := map[string]string{} // a finding's line, lens and title → the judge's question
	lensAt := map[string]string{}
	for _, a := range c.Asked {
		i := strings.LastIndex(a, "\x00")
		keys[a[:i]] = a[i+1:]
	}
	var judgeCalls []string
	verdicts := map[string]string{}
	var questions []string
	for _, k := range keys {
		questions = append(questions, k)
	}
	sort.Strings(questions)
	for _, key := range questions {
		var a struct {
			Yes                 *bool  `yaml:"yes"`
			Model, Level, Error string `yaml:",omitempty"`
		}
		data, err := os.ReadFile(filepath.Join(runDir, "in", "judge", key, "answer.yaml"))
		if err == nil {
			yaml.Unmarshal(data, &a)
		}
		switch {
		case a.Yes != nil && *a.Yes:
			verdicts[key] = "yes"
		case a.Yes != nil:
			verdicts[key] = "no"
		}
		if a.Model != "" && rr.Judge == "" {
			rr.Judge = a.Model + " (" + a.Level + ")"
		}
		if !strings.HasPrefix(a.Error, "not asked") {
			judgeCalls = append(judgeCalls, key)
		}
	}
	for _, list := range [][]reviewer.Finding{c.Related, c.Outside} {
		for _, g := range list {
			lead := g
			lead.Also = nil
			for _, f := range append([]reviewer.Finding{lead}, g.Also...) { // grouped on its line, each judged apart (#229)
				key := keys[reviewer.JudgeKey(f)]
				lensAt[key] = f.Lens
				rr.Findings = append(rr.Findings, lensFinding{f.Lens, f.Where, f.Severity, f.Title, f.Related, verdicts[key]})
			}
		}
	}
	p, j := 0, 0
	for _, call := range res.Calls {
		lens := "other"
		switch {
		case call.Task == "part" && p < len(partCalls):
			lens, p = partCalls[p], p+1
		case call.Task == "judge" && j < len(judgeCalls):
			lens, j = lensAt[judgeCalls[j]], j+1
		}
		rr.TokensIn[lens] += call.TokensIn
		rr.TokensOut[lens] += call.TokensOut
	}
	return rr, nil
}

// lensTally is one lens's share of a review: the defects planted for it and
// what became of them, and what it raised that nothing planted.
type lensTally struct {
	Planted   int // defects of this lens's kind planted on the change
	Found     int // of them, shown on the change, whichever lens raised it
	Own       int // of them, raised by this lens
	JudgeNo   int // of them, refused by the judge at least once: its false negatives
	FP        int // important findings it raised on the change that nothing planted, shown
	Nits      int // nits it raised on the change that nothing planted, shown
	Dropped   int // findings it raised that nothing planted, refused by the judge
	Outside   int // important findings it raised outside the change
	Opened    int // of them, verified: an issue each on a forge
	Answers   int // findings it answered, before any check
	Unfounded int // dropped, their quote not found again
	In, Out   int // tokens: its call and its findings' judges
}

var tallyFields = []string{"planted", "found", "own", "judge-no", "fp", "nits", "dropped", "outside", "opened", "answers", "unfounded", "in", "out"}

func (t *lensTally) values() []*int {
	return []*int{&t.Planted, &t.Found, &t.Own, &t.JudgeNo, &t.FP, &t.Nits, &t.Dropped, &t.Outside, &t.Opened, &t.Answers, &t.Unfounded, &t.In, &t.Out}
}

// scoreReview scores a review against its case: a point for each defect
// shown on the change, for each place kept clean, and one when nothing
// important was shown that nothing planted. measure is the run's tallies,
// a lens each, for results.tsv; notes say what each finding was taken for.
func scoreReview(rc *reviewCase, rr *reviewRun) (passed int, failed []string, measure string, notes []string) {
	tallies := map[string]*lensTally{}
	tally := func(lens string) *lensTally {
		if tallies[lens] == nil {
			tallies[lens] = &lensTally{}
		}
		return tallies[lens]
	}
	for _, l := range rr.Lenses {
		t := tally(l)
		t.Answers, t.Unfounded, t.In, t.Out = rr.Answers[l], rr.Unfounded[l], rr.TokensIn[l], rr.TokensOut[l]
	}
	at := func(ps []plant, f lensFinding) int {
		for i, p := range ps {
			for _, pl := range p.At {
				if pl.holds(f.Where) {
					return i
				}
			}
		}
		return -1
	}
	for i, d := range rc.Defects {
		t := tally(d.Lens)
		t.Planted++
		found, own, refused := false, false, false
		for _, f := range rr.Findings {
			if !f.Related || at(rc.Defects[i:i+1], f) < 0 {
				continue
			}
			if f.shown() {
				found, own = true, own || f.Lens == d.Lens
			}
			refused = refused || f.Judged == "no"
		}
		if found {
			passed++
			t.Found++
		} else {
			failed = append(failed, fmt.Sprintf("defect: %s %s at %s not found", d.Lens, d.Kind, d.At[0]))
		}
		if own {
			t.Own++
		}
		if refused {
			t.JudgeNo++
		}
	}
	unplanted := 0
	for _, f := range rr.Findings {
		t := tally(f.Lens)
		what := "planted"
		switch {
		case !f.Related:
			if f.Severity == "important" {
				t.Outside++
				if f.Judged == "yes" {
					t.Opened++
				}
			}
			what = "outside, unplanted"
			if i := at(rc.Outside, f); i >= 0 {
				what = "outside, planted " + rc.Outside[i].Kind
			}
		case at(rc.Defects, f) >= 0:
		case f.Judged == "no":
			t.Dropped++
			what = "unplanted, refused by the judge"
		case f.Severity == "important":
			t.FP++
			unplanted++
			what = "unplanted, shown"
		default:
			t.Nits++
			what = "unplanted nit, shown"
		}
		notes = append(notes, fmt.Sprintf("%s %s %s (judge: %s) — %s: %s", f.Lens, f.Severity, f.Where, orDash(f.Judged), what, f.Title))
	}
	for _, n := range rc.Not {
		hit := ""
		for _, f := range rr.Findings {
			if f.shown() && n.holds(f.Where) {
				hit = f.Lens + ": " + f.Title
			}
		}
		if hit != "" {
			failed = append(failed, fmt.Sprintf("not: %s found at %s, where %s", hit, n, n.Why))
		} else {
			passed++
		}
	}
	if unplanted == 0 {
		passed++
	} else {
		failed = append(failed, fmt.Sprintf("clean: %d important findings shown that nothing planted", unplanted))
	}
	var outside []string
	for _, o := range rc.Outside {
		got := "not raised"
		for _, f := range rr.Findings {
			if !f.Related && at([]plant{o}, f) == 0 {
				got = "raised by " + f.Lens + ", judge " + orDash(f.Judged)
				if f.Judged == "yes" {
					got = "opened by " + f.Lens
				}
			}
		}
		outside = append(outside, fmt.Sprintf("%s:%s=%s", o.Lens, o.Kind, got))
	}
	lenses := slices.Clone(rr.Lenses)
	for l := range tallies {
		if !slices.Contains(lenses, l) {
			lenses = append(lenses, l)
		}
	}
	sort.SliceStable(lenses, func(i, j int) bool { return lensRank(lenses[i]) < lensRank(lenses[j]) })
	parts := []string{fmt.Sprintf("floor=%d", rr.Floor)}
	for _, l := range lenses {
		fields := []string{l}
		for i, v := range tallies[l].values() {
			fields = append(fields, fmt.Sprintf("%s=%d", tallyFields[i], *v))
		}
		parts = append(parts, strings.Join(fields, " "))
	}
	if len(outside) > 0 {
		parts = append(parts, "outside-planted "+strings.Join(outside, ", "))
	}
	return passed, failed, strings.Join(parts, "; "), notes
}

// lensRank keeps the role's order of the lenses in a measure.
func lensRank(l string) int {
	if i := slices.Index([]string{"correctness", "edge-cases", "tests"}, l); i >= 0 {
		return i
	}
	return 99
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
