// Summary reads results.tsv and prints, per case, models, effort and judge,
// how many runs there were, the pass rate (the share of runs earning every
// point), the mean score and its range, and what a run used: one run says
// little, since a model answers differently from one run to the next, and a
// best run hides the others (ADR-0014). Fewer than five runs are marked
// `few runs`; scores earned by a model that no longer answers on this machine
// (the models-seen file) are marked `replaced`. A replay case's runs of
// different workline commits are never pooled: a before and an after stay
// apart. The replay set's cases (docs/spec/conformance/replay.md) are
// also summed by set: how many cases earned every point in each of their
// runs, and how many runs did.
//
//	go run ./tests/evaluation/summary [results.tsv]
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
)

// minRuns is the fewest runs a measure is read from (ADR-0014).
const minRuns = 5

type key struct{ kase, version, agent, models, effort, judge string }

type group struct {
	scores              []float64 // share of the points earned
	passes              int       // runs that earned every point
	tokensIn, tokensOut []float64
	seconds             []float64
	calls               []float64
}

func main() {
	path := "tests/evaluation/results.tsv"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma, r.LazyQuotes, r.FieldsPerRecord = '\t', true, -1
	head, err := r.Read()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	col := map[string]int{}
	for i, h := range head {
		col[h] = i
	}
	groups := map[key]*group{}
	lenses := map[lensKey]*lensSum{}
	outside := map[lensKey]map[string]int{}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		get := func(name string) string {
			if i, ok := col[name]; ok && i < len(row) {
				return row[i]
			}
			return ""
		}
		k := key{get("case"), get("workline"), get("agent"), get("models"), get("effort"), get("judge")}
		if !replayCase.MatchString(k.kase) {
			k.version, k.agent = "", "" // a fixture's runs pooled across commits, as ever
		}
		if k.models == "" {
			k.models = "(not recorded)"
		}
		g := groups[k]
		if g == nil {
			g = &group{}
			groups[k] = g
		}
		if got, total, ok := strings.Cut(get("score"), "/"); ok {
			if n, err1 := strconv.ParseFloat(got, 64); err1 == nil {
				if d, err2 := strconv.ParseFloat(total, 64); err2 == nil && d > 0 {
					g.scores = append(g.scores, n/d)
					if n == d {
						g.passes++
					}
				}
			}
		}
		add(&g.tokensIn, get("tokens in"))
		add(&g.tokensOut, get("tokens out"))
		add(&g.seconds, get("seconds"))
		add(&g.calls, get("agent calls"))
		addMeasure(lenses, outside, get("agent"), get("measure"))
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.kase != b.kase {
			return a.kase < b.kase
		}
		if a.version != b.version {
			return a.version < b.version
		}
		if a.models != b.models {
			return a.models < b.models
		}
		if a.effort != b.effort {
			return a.effort < b.effort
		}
		return a.judge < b.judge
	})
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	answering := currentModels(agent.Seen())
	_, _ = fmt.Fprintln(w, "case\tworkline\tmodels\teffort\tjudge\truns\tpass\tscore\trange\tcalls\ttokens in\ttokens out\tseconds\t")
	for _, k := range keys {
		g := groups[k]
		lo, hi := bounds(g.scores)
		var notes []string
		if len(g.scores) < minRuns {
			notes = append(notes, "few runs")
		}
		if answering != nil && k.models != "(not recorded)" {
			for _, m := range strings.Split(k.models, ">") {
				if !answering[m] {
					notes = append(notes, "replaced")
					break
				}
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d/%d\t%.0f%%\t%.0f–%.0f%%\t%s\t%s\t%s\t%s\t%s\n", k.kase, k.version, k.models, k.effort, k.judge, len(g.scores),
			g.passes, len(g.scores), 100*mean(g.scores), 100*lo, 100*hi, show(g.calls, "%.1f"), show(g.tokensIn, "%.0f"), show(g.tokensOut, "%.0f"), show(g.seconds, "%.0f"), strings.Join(notes, ", "))
	}
	w.Flush()
	printSets(groups)
	printLenses(lenses, outside)
}

// replayCase names a replay case's set: its role, then kept (a catch to
// keep), clean (a change nobody sent back) or miss; "" for a fixture.
var replayCase = regexp.MustCompile(`^([a-z-]+)/(kept-|clean-)?[a-z]+-[0-9]+-`)

// printSets sums the replay cases by set, workline commit and agent: the
// cases that earned every point in each of their runs (k of n runs, the
// worst counted), the runs that did, and the mean of each case's worst
// run.
func printSets(groups map[key]*group) {
	type setKey struct{ set, version, agent string }
	type setSum struct {
		cases, allPass, runs, passes int
		worst                        []float64
	}
	sets := map[setKey]*setSum{}
	for k, g := range poolCases(groups) {
		m := replayCase.FindStringSubmatch(k.kase)
		kind := map[string]string{"": "miss", "kept-": "kept", "clean-": "clean"}[m[2]]
		sk := setKey{m[1] + "/" + kind, k.version, k.agent}
		s := sets[sk]
		if s == nil {
			s = &setSum{}
			sets[sk] = s
		}
		lo, _ := bounds(g.scores)
		s.cases, s.runs, s.passes, s.worst = s.cases+1, s.runs+len(g.scores), s.passes+g.passes, append(s.worst, lo)
		if g.passes == len(g.scores) {
			s.allPass++
		}
	}
	if len(sets) == 0 {
		return
	}
	keys := make([]setKey, 0, len(sets))
	for k := range sets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "replay set\tworkline\tagent\tcases\tevery run full\truns full\tworst run, mean\t")
	for _, k := range keys {
		s := sets[k]
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d/%d\t%d/%d\t%.0f%%\t\n", k.set, k.version, k.agent, s.cases, s.allPass, s.cases, s.passes, s.runs, 100*mean(s.worst))
	}
	_ = w.Flush()
}

// poolCases is each replay case's runs at one workline commit and agent,
// whatever models and judge a run named: a reviewer run whose findings
// needed no judge names neither, and is the same case, counted once.
func poolCases(groups map[key]*group) map[key]*group {
	cases := map[key]*group{}
	for k, g := range groups {
		if !replayCase.MatchString(k.kase) || len(g.scores) == 0 {
			continue
		}
		ck := key{kase: k.kase, version: k.version, agent: k.agent}
		if cases[ck] == nil {
			cases[ck] = &group{}
		}
		cases[ck].scores, cases[ck].passes = append(cases[ck].scores, g.scores...), cases[ck].passes+g.passes
	}
	return cases
}

// The reviewer's measure (#90): its cases write, a run, each lens's tallies
// in the `measure` column (tests/evaluation/review_test.go); they are summed
// here by lens, the agent asked (the `agent` column: a run with no judge
// call names fewer models, and is the same measure) and how the lenses
// were asked — together, apart, or the diff-alone facet alone (#360).
type lensKey struct{ lens, agent, asked string }

type lensSum struct {
	runs   int
	n      map[string]int // the tallies, by name, summed
	floors []float64
}

// addMeasure reads one row's measure: "floor=2; correctness planted=1
// found=1 …; edge-cases …; outside-planted tests:untested=opened by tests".
func addMeasure(lenses map[lensKey]*lensSum, outside map[lensKey]map[string]int, agent, measure string) {
	if measure == "" {
		return
	}
	asked := askedIn(measure)
	floor := ""
	for _, part := range strings.Split(measure, "; ") {
		if v, ok := strings.CutPrefix(part, "floor="); ok {
			floor = v
			continue
		}
		if v, ok := strings.CutPrefix(part, "outside-planted "); ok {
			ok := lensKey{"", agent, asked}
			if outside[ok] == nil {
				outside[ok] = map[string]int{}
			}
			for _, o := range strings.Split(v, ", ") {
				outside[ok][o]++
			}
			continue
		}
		fields := strings.Fields(part)
		if len(fields) == 0 || strings.Contains(fields[0], "=") { // a replay's extra=N: no lens
			continue
		}
		lk := lensKey{fields[0], agent, asked}
		s := lenses[lk]
		if s == nil {
			s = &lensSum{n: map[string]int{}}
			lenses[lk] = s
		}
		s.runs++
		add(&s.floors, floor)
		for _, f := range fields[1:] {
			name, v, _ := strings.Cut(f, "=")
			n, _ := strconv.Atoi(v)
			s.n[name] += n
		}
	}
}

// askedIn is how a row's lenses were asked: together (the `lenses` part,
// their one call), the diff-alone facet alone (its part, and no
// `lenses`), one lens (one part only spent tokens, as a push to a merge
// request asks), or apart.
func askedIn(measure string) string {
	asked, spent := "apart", 0
	for _, part := range strings.Split(measure, "; ") {
		switch {
		case strings.HasPrefix(part, "lenses "):
			return "together"
		case strings.HasPrefix(part, "diff-alone "):
			asked = "alone"
		case strings.Contains(part, " in=") && !strings.Contains(part, " in=0 "):
			spent++
		}
	}
	if asked == "apart" && spent == 1 {
		return "one lens"
	}
	return asked
}

// printLenses prints, a lens each, the defects planted for it and how many
// were found (by any lens), those its own finding verified (what a block on
// it holds), the judge's refusals of true findings, what it raised that
// nothing planted — shown, verified, nits, or refused — its findings shown
// not verified, what it raised outside the change, its decisions for a
// person, its judges' calls, the finder floor, and its tokens.
func printLenses(lenses map[lensKey]*lensSum, outside map[lensKey]map[string]int) {
	if len(lenses) == 0 {
		return
	}
	keys := make([]lensKey, 0, len(lenses))
	for k := range lenses {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.agent+a.asked != b.agent+b.asked {
			return a.agent+a.asked < b.agent+b.asked
		}
		return lensOrder(a.lens) < lensOrder(b.lens)
	})
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "reviewer lens\tagent\tasked\truns\tfound/planted\town\town verified\tjudge refused true\tfalse shown\tfalse verified\tnot verified\tnits\tfalse refused\toutside (opened)\tanswers\tdecisions\tjudge calls\tfloor\ttokens in\ttokens out\t")
	for _, k := range keys {
		s := lenses[k]
		n := s.n
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d/%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d (%d)\t%d\t%d\t%d\t%s\t%d\t%d\t\n", k.lens, k.agent, k.asked, s.runs,
			n["found"], n["planted"], n["own"], n["own-verified"], n["judge-no"], n["fp"], n["fp-verified"], n["unsure"], n["nits"], n["dropped"],
			n["outside"], n["opened"], n["answers"], n["questions"], n["judges"], show(s.floors, "%.1f"), n["in"], n["out"])
	}
	w.Flush()
	for k, o := range outside {
		var lines []string
		for what, n := range o {
			lines = append(lines, fmt.Sprintf("%s ×%d", what, n))
		}
		sort.Strings(lines)
		fmt.Printf("outside the change, planted (%s, %s): %s\n", k.agent, k.asked, strings.Join(lines, "; "))
	}
}

func lensOrder(l string) int {
	for i, x := range []string{"lenses", "correctness", "edge-cases", "tests", "intent", "claims", "diff-alone"} {
		if l == x {
			return i
		}
	}
	return 99
}

// currentModels returns the models answering now, from the models-seen file;
// nil when there is none, so nothing is marked.
func currentModels(file string) map[string]bool {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var seen map[string]struct {
		Model string `yaml:"model"`
	}
	if yaml.Unmarshal(data, &seen) != nil {
		return nil
	}
	now := map[string]bool{}
	for _, s := range seen {
		now[s.Model] = true
	}
	return now
}

// add keeps a number, when the column holds one: older runs did not record
// every column.
func add(to *[]float64, s string) {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		*to = append(*to, v)
	}
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	t := 0.0
	for _, x := range v {
		t += x
	}
	return t / float64(len(v))
}

func bounds(v []float64) (lo, hi float64) {
	for i, x := range v {
		if i == 0 || x < lo {
			lo = x
		}
		if i == 0 || x > hi {
			hi = x
		}
	}
	return lo, hi
}

// show gives a mean, or - when no run recorded it.
func show(v []float64, format string) string {
	if len(v) == 0 {
		return "-"
	}
	return fmt.Sprintf(format, mean(v))
}
