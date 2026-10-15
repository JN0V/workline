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
// there, so a range never drifts from what it names. An issue the change
// closes is a place too, `#4` and no lines: an intent finding quoting it
// is there (#126); text is then a part of the issue's body.
type place struct {
	File  string `yaml:"file"`
	Lines []int  `yaml:"lines"`
	Text  string `yaml:"text"`
	Why   string `yaml:"why"`
}

func (p place) issue() bool { return strings.HasPrefix(p.File, "#") }

func (p place) holds(where string) bool {
	if p.issue() {
		return where == p.File
	}
	file, line, ok := strings.Cut(where, ":")
	n, err := strconv.Atoi(line)
	return ok && err == nil && file == p.File && len(p.Lines) == 2 && n >= p.Lines[0] && n <= p.Lines[1]
}

func (p place) String() string {
	if p.issue() {
		return p.File
	}
	return fmt.Sprintf("%s:%d-%d", p.File, p.Lines[0], p.Lines[1])
}

// lensFinding is one finding a lens raised, once its quotes were found
// again, and what became of it.
type lensFinding struct {
	Lens, Where, Severity, Title string
	Related                      bool   // its cause lies in the change
	Judged                       string // yes (verified), no (dropped), unsure (shown not verified), or "" when not judged (a nit)
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
	Judges    map[string]int // judge calls on each lens's findings, a round of reads each
	Questions map[string]int // decisions each lens put to a person (#126)
	Judge     string         // the judge's model and the independence it reached
}

var (
	floorRe     = regexp.MustCompile(`Look for at least (\d+) candidates`)
	unfoundedRe = regexp.MustCompile(`^dropped, from the (\S+) lens:`)
)

// readReview reads a reviewer run's folder; res gives its calls, in order,
// and the findings it logged.
func readReview(runDir string, res result) (*reviewRun, error) {
	rr := &reviewRun{Answers: map[string]int{}, Unfounded: map[string]int{}, TokensIn: map[string]int{}, TokensOut: map[string]int{},
		Judges: map[string]int{}, Questions: map[string]int{}}
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
		Related, Outside, Questions []reviewer.Finding
		Asked                       []string
	}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "candidates.json"))
	if err == nil {
		err = json.Unmarshal(data, &c)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("candidates: %v", err)
	}
	for _, q := range c.Questions {
		rr.Questions[q.Lens]++
	}
	// What the run kept once judged (in/review.json): each finding judged
	// is verified, shown not verified — the judge unsure, its no quoting
	// nothing found again, its reads or the run's tokens spent (ADR-0045)
	// — or, gone from it, dropped.
	var rv struct{ Change, Outside []reviewer.Finding }
	kept := map[string]string{}
	data, err = os.ReadFile(filepath.Join(runDir, "in", "review.json"))
	reviewed := err == nil && json.Unmarshal(data, &rv) == nil
	for _, f := range flat(rv.Change, rv.Outside) {
		kept[reviewer.JudgeKey(f)] = f.Verified
	}
	keys := map[string]string{} // a finding's line, lens and title → the judge's question
	lensAt := map[string]string{}
	for _, a := range c.Asked {
		i := strings.LastIndex(a, "\x00")
		keys[a[:i]] = a[i+1:]
	}
	var judgeCalls []string
	answered := map[string]string{}
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
			answered[key] = "yes"
		case a.Yes != nil:
			answered[key] = "no"
		}
		if a.Model != "" && rr.Judge == "" {
			rr.Judge = a.Model + " (" + a.Level + ")"
		}
		if !strings.HasPrefix(a.Error, "not asked") {
			judgeCalls = append(judgeCalls, key)
		}
	}
	for _, f := range flat(c.Related, c.Outside) {
		key, judged := keys[reviewer.JudgeKey(f)]
		lensAt[key] = f.Lens
		said := ""
		switch verified, shown := kept[reviewer.JudgeKey(f)]; {
		case !judged: // a nit, never judged
		case !reviewed: // a run that wrote no review: the judge's answer
			said = answered[key]
		case !shown:
			said = "no"
		case strings.HasPrefix(verified, "verified"):
			said = "yes"
		default:
			said = "unsure"
		}
		rr.Findings = append(rr.Findings, lensFinding{f.Lens, f.Where, f.Severity, f.Title, f.Related, said})
	}
	// Each call by what it answered (`for`: its part, or its judge's
	// question, asked again for each round of reads); in order when the
	// agent's calls do not say it.
	p, j := 0, 0
	for _, call := range res.Calls {
		lens := "other"
		switch {
		case call.Task == "part" && call.For != "":
			_, lens, _ = strings.Cut(call.For, "-")
		case call.Task == "judge" && strings.HasPrefix(call.For, "judge/"):
			if l, ok := lensAt[strings.TrimPrefix(call.For, "judge/")]; ok {
				lens = l
			}
		case call.Task == "part" && p < len(partCalls):
			lens, p = partCalls[p], p+1
		case call.Task == "judge" && j < len(judgeCalls):
			lens, j = lensAt[judgeCalls[j]], j+1
		}
		if call.Task == "judge" {
			rr.Judges[lens]++
		}
		rr.TokensIn[lens] += call.TokensIn
		rr.TokensOut[lens] += call.TokensOut
	}
	return rr, nil
}

// flat is findings as each was judged: those grouped on a line under
// another (#229) taken apart.
func flat(lists ...[]reviewer.Finding) []reviewer.Finding {
	var out []reviewer.Finding
	for _, list := range lists {
		for _, g := range list {
			lead := g
			lead.Also = nil
			out = append(append(out, lead), g.Also...)
		}
	}
	return out
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
	// What `ai-findings: block` on this lens would have held a merge on
	// (#360): only a finding the judge verified blocks.
	OwnVerified int // of the defects planted for it, raised by this lens and verified
	FPVerified  int // of fp, verified: a block on nothing planted
	Unsure      int // important findings it raised on the change, shown not verified
	Questions   int // decisions it put to a person, never judged (#126)
	Judges      int // judge calls on its findings, a round of reads each
}

var tallyFields = []string{"planted", "found", "own", "judge-no", "fp", "nits", "dropped", "outside", "opened", "answers", "unfounded", "in", "out",
	"own-verified", "fp-verified", "unsure", "questions", "judges"}

func (t *lensTally) values() []*int {
	return []*int{&t.Planted, &t.Found, &t.Own, &t.JudgeNo, &t.FP, &t.Nits, &t.Dropped, &t.Outside, &t.Opened, &t.Answers, &t.Unfounded, &t.In, &t.Out,
		&t.OwnVerified, &t.FPVerified, &t.Unsure, &t.Questions, &t.Judges}
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
	// Asked together, each lens's findings, judges and questions are its
	// own; the call's answers and tokens are the `lenses` part's.
	for _, m := range []map[string]int{rr.TokensIn, rr.TokensOut, rr.Judges, rr.Questions, rr.Unfounded} {
		for l := range m {
			tally(l)
		}
	}
	for l, t := range tallies {
		t.In, t.Out, t.Judges, t.Questions, t.Unfounded = rr.TokensIn[l], rr.TokensOut[l], rr.Judges[l], rr.Questions[l], rr.Unfounded[l]
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
		found, own, refused, blocks := false, false, false, false
		for _, f := range rr.Findings {
			if !f.Related || at(rc.Defects[i:i+1], f) < 0 {
				continue
			}
			if f.shown() {
				found, own = true, own || f.Lens == d.Lens
				blocks = blocks || (f.Lens == d.Lens && f.Severity == "important" && f.Judged == "yes")
			}
			refused = refused || f.Judged == "no"
		}
		if blocks {
			t.OwnVerified++
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
		if f.Related && f.Severity == "important" && f.Judged == "unsure" {
			t.Unsure++
		}
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
			if f.Judged == "yes" {
				t.FPVerified++
			}
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

// lensRank keeps the role's order of the lenses in a measure; `lenses`, the
// call asking them together (its answers and tokens; each finding's under
// its lens), first.
func lensRank(l string) int {
	if i := slices.Index([]string{"lenses", "correctness", "edge-cases", "tests", "intent", "claims", "diff-alone"}, l); i >= 0 {
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
