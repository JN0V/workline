// Meaning scores reviewer replay runs by what their findings say, not by
// where they are quoted (docs/spec/conformance/replay.md, "Scored by
// meaning"). A judge, the same for every arm and blind to it, is given a
// case's points (points.yaml) and every finding one run raised on it, and
// says which findings state which point; on a clean case it reads the
// repository and says whether each Important shown is a true defect, a
// nit or a false alarm. Each judgement is asked twice, a third time when
// the two differ.
//
//	go run ./tests/evaluation/meaning -findings runs.jsonl -out scores/
//
// It writes scores.tsv (a row per point and run) and summary.md (per arm),
// and keeps each judgement under <out>/judgments, so a second run asks
// only what is new.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.yaml.in/yaml/v3"
)

//go:embed points.yaml
var pointsYAML []byte

// point is one defect a case plants, as the judge is asked it.
type point struct {
	Name  string `yaml:"name"`
	Class string `yaml:"class"`
	Says  string `yaml:"says"`
	Not   string `yaml:"not"`
	Out   string `yaml:"out"` // why it is left out of the score; "" when scored
}

type caseDef struct {
	Change string  `yaml:"change"`
	Points []point `yaml:"points"`
}

func (c caseDef) clean() bool { return len(c.Points) == 0 }

// finding is one finding of a run, as its author would read it.
type finding struct {
	Place    string `json:"place"`
	Severity string `json:"severity"` // important, nit, question
	Refused  bool   `json:"refused"`  // the engine's own judge dropped it
	Title    string `json:"title"`
	Body     string `json:"body"`
}

func (f finding) shown() bool { return f.Severity == "important" && !f.Refused }

// caseRun is one line of the findings file: a case, once, by one arm.
// Run 0 is numbered by its order among the arm's runs of the case.
type caseRun struct {
	Arm       string    `json:"arm"`
	Run       int       `json:"run"`
	Case      string    `json:"case"`
	TokensIn  int       `json:"tokens_in"`
	TokensOut int       `json:"tokens_out"`
	Findings  []finding `json:"findings"`
}

// asker asks the judge one question; dir, when set, is a tree it may read.
type asker func(system, user, dir string) (answer string, tokens int, err error)

// Verdicts, weakest first: a point's, then a clean finding's.
var (
	pointRank = []string{"not said", "nit", "refused", "partly", "said"}
	cleanRank = []string{"false alarm", "nit", "true defect"}
)

const pointSystem = `You grade the findings of a code review against defects a maintainer found in the same change. Each defect is a point. You are given every finding one review raised on the change. For each point, say which findings state it.

Rules:
- Judge by meaning, never by place or by shared words. A finding states a point when a developer who reads that finding alone learns the defect the point names. Where the finding is quoted does not matter.
- "said": the finding states the point's defect itself. It may say more, word it otherwise, or propose another fix.
- "partly": the finding states a part of the defect, or a direct consequence of it, without the defect itself.
- Credit only what a finding says. What a reader could work out from it, and it does not say, is not said. A finding on the same file, line or test that names another defect does not state the point.
- "Does not count" lists near misses: none of them states the point.
- One finding may state several points; several findings may state one.

Answer with one JSON object and nothing else:
{"points": {"<point name>": {"said": ["F2"], "partly": [], "quote": "<the words of a finding that state it, 30 words at most; empty when none>", "why": "<one short sentence>"}}}
Name every point; leave both lists empty when no finding states it.`

const cleanSystem = `You check the blocking findings a code review raised on a change that was merged and never sent back. You are in the repository at the change's head: read the code and the tests (Read, Grep, Glob) before you judge. Trust neither the finding nor the change's description.

For each finding, one verdict:
- "true defect": what it says is true of this change, and it is worth holding the merge: a wrong behaviour, or a behaviour the issue asks that the change does not deliver or that no test proves.
- "nit": what it says is true, and too small to hold a merge.
- "false alarm": what it says is not true of the code as it reads, or is no defect.

Answer with one JSON object and nothing else:
{"findings": {"F1": {"verdict": "true defect", "why": "<one or two short sentences, naming what you read>"}}}
Name every finding.`

// findingsText lists findings for the judge: an id, the place, the words.
// Neither the arm, its model, the lens nor the severity is given.
func findingsText(fs []finding, ids []int) string {
	var b strings.Builder
	for _, i := range ids {
		f := fs[i]
		fmt.Fprintf(&b, "## F%d, at %s\n\n%s\n\n", i+1, f.Place, strings.TrimSpace(f.Title+"\n"+f.Body))
	}
	return b.String()
}

func pointPrompt(def caseDef, fs []finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# The change\n\n%s\n\n# Points\n\n", def.Change)
	for _, p := range def.Points {
		fmt.Fprintf(&b, "## %s\n\n- States it: %s\n- Does not count: %s\n\n", p.Name, p.Says, p.Not)
	}
	ids := make([]int, len(fs))
	for i := range fs {
		ids[i] = i
	}
	fmt.Fprintf(&b, "# The review's findings (%d)\n\n%s", len(fs), findingsText(fs, ids))
	return b.String()
}

var idRe = regexp.MustCompile(`^F(\d+)$`)

// ids reads finding ids ("F3") as indexes; one that names no finding is an error.
func ids(names []string, n int) ([]int, error) {
	var out []int
	for _, name := range names {
		m := idRe.FindStringSubmatch(strings.TrimSpace(name))
		if m == nil {
			return nil, fmt.Errorf("%q is no finding id", name)
		}
		i, _ := strconv.Atoi(m[1])
		if i < 1 || i > n {
			return nil, fmt.Errorf("finding F%d: the run has %d findings", i, n)
		}
		out = append(out, i-1)
	}
	return out, nil
}

// jsonIn is the JSON object an answer holds, whatever is written around it.
func jsonIn(answer string) string {
	i, j := strings.Index(answer, "{"), strings.LastIndex(answer, "}")
	if i < 0 || j < i {
		return ""
	}
	return answer[i : j+1]
}

// pointAnswer is what the judge said of one point.
type pointAnswer struct {
	Said, Partly []int
	Quote, Why   string
}

func readPoints(answer string, def caseDef, n int) (map[string]pointAnswer, error) {
	var a struct {
		Points map[string]struct {
			Said, Partly []string
			Quote, Why   string
		}
	}
	if err := json.Unmarshal([]byte(jsonIn(answer)), &a); err != nil {
		return nil, fmt.Errorf("answer: %v", err)
	}
	out := map[string]pointAnswer{}
	for _, p := range def.Points {
		got, ok := a.Points[p.Name]
		if !ok {
			return nil, fmt.Errorf("answer: no point %q", p.Name)
		}
		said, err := ids(got.Said, n)
		if err != nil {
			return nil, err
		}
		partly, err := ids(got.Partly, n)
		if err != nil {
			return nil, err
		}
		out[p.Name] = pointAnswer{said, partly, got.Quote, got.Why}
	}
	return out, nil
}

// verdict is what a run earns on a point: said or partly by an Important
// shown; else refused, when only an Important the engine's judge dropped
// states it; else nit, when only a nit or a question does.
func verdict(fs []finding, a pointAnswer) string {
	best := 0
	raise := func(v string) { best = max(best, slices.Index(pointRank, v)) }
	for _, i := range a.Said {
		switch f := fs[i]; {
		case f.shown():
			raise("said")
		case f.Severity == "important":
			raise("refused")
		default:
			raise("nit")
		}
	}
	for _, i := range a.Partly {
		if fs[i].shown() {
			raise("partly")
		}
	}
	return pointRank[best]
}

// settle is the verdict of two or three judgements: the majority, the
// middle one when all three differ.
func settle(rank []string, vs []string) string {
	sorted := slices.Clone(vs)
	sort.Slice(sorted, func(i, j int) bool { return slices.Index(rank, sorted[i]) < slices.Index(rank, sorted[j]) })
	return sorted[len(sorted)/2]
}

// row is one line of scores.tsv.
type row struct {
	Arm                 string
	Run                 int
	Case, Set           string // miss, kept, clean
	Item, Class, Scored string
	Verdict             string
	Judges              []string
	Findings            string // the findings that state it: said; partly
	Quote, Why          string
}

// judgement is one case run judged: its rows, and what the judge cost.
type judgement struct {
	rows          []row
	calls, tokens int
	extra         int // Importants shown on a miss that state no point
	shown         int // Importants shown
}

// judge holds what a scoring run needs.
type judge struct {
	ask   asker
	cache string
	tree  func(kase string) (dir, material string, err error) // a clean case's repository and change
	pause time.Duration                                       // before asking again an answer that failed or did not read
	mu    sync.Mutex
}

// asked asks the n-th judgement of a prompt, or reads it from the cache.
// read checks the answer; one that does not read is asked again, twice.
func (j *judge) asked(system, user, dir string, n int, read func(string) error) (string, bool, int, error) {
	sum := sha256.Sum256([]byte(system + "\x00" + user))
	file := filepath.Join(j.cache, fmt.Sprintf("%x-%d.json", sum[:8], n))
	var kept struct {
		Answer string `json:"answer"`
		Tokens int    `json:"tokens"`
	}
	if data, err := os.ReadFile(file); err == nil && json.Unmarshal(data, &kept) == nil && read(kept.Answer) == nil {
		return kept.Answer, false, 0, nil
	}
	var last error
	tokens := 0
	for try := 0; try < 3; try++ {
		answer, used, err := j.ask(system, user, dir)
		tokens += used
		if err == nil {
			err = read(answer)
		}
		if err != nil {
			last = err
			time.Sleep(time.Duration(try+1) * j.pause)
			continue
		}
		kept.Answer, kept.Tokens = answer, tokens
		data, _ := json.Marshal(kept)
		if err := errors.Join(os.MkdirAll(j.cache, 0o755), os.WriteFile(file, data, 0o644)); err != nil {
			return "", true, tokens, err
		}
		return answer, true, tokens, nil
	}
	return "", true, tokens, last
}

func setOf(kase string) string {
	switch {
	case strings.HasPrefix(kase, "kept-"):
		return "kept"
	case strings.HasPrefix(kase, "clean-"):
		return "clean"
	}
	return "miss"
}

// points judges a miss or a kept catch: each point, twice, a third time
// for a run whose two judgements differ on any point.
func (j *judge) points(def caseDef, cr caseRun) (judgement, error) {
	var out judgement
	var answers []map[string]pointAnswer
	if len(cr.Findings) > 0 {
		user := pointPrompt(def, cr.Findings)
		for n := 1; n <= 3; n++ {
			var got map[string]pointAnswer
			_, fresh, tokens, err := j.asked(pointSystem, user, "", n, func(answer string) (err error) {
				got, err = readPoints(answer, def, len(cr.Findings))
				return err
			})
			if fresh {
				out.calls++
			}
			out.tokens += tokens
			if err != nil {
				return out, err
			}
			answers = append(answers, got)
			if n == 2 && !slices.ContainsFunc(def.Points, func(p point) bool {
				return verdict(cr.Findings, answers[0][p.Name]) != verdict(cr.Findings, answers[1][p.Name])
			}) {
				break
			}
		}
	}
	stated := map[int]bool{}
	for _, p := range def.Points {
		r := row{Arm: cr.Arm, Run: cr.Run, Case: cr.Case, Set: setOf(cr.Case), Item: p.Name, Class: p.Class, Scored: "yes", Verdict: "not said"}
		if p.Out != "" {
			r.Scored = p.Out
		}
		var best pointAnswer
		for _, a := range answers {
			v := verdict(cr.Findings, a[p.Name])
			r.Judges = append(r.Judges, v)
			for _, i := range append(slices.Clone(a[p.Name].Said), a[p.Name].Partly...) {
				stated[i] = true
			}
		}
		if len(answers) > 0 {
			r.Verdict = settle(pointRank, r.Judges)
			best = answers[slices.Index(r.Judges, r.Verdict)][p.Name]
			r.Findings = idList(best.Said) + "; " + idList(best.Partly)
			r.Quote, r.Why = best.Quote, best.Why
		}
		out.rows = append(out.rows, r)
	}
	for i, f := range cr.Findings {
		if f.shown() {
			out.shown++
			if !stated[i] {
				out.extra++
			}
		}
	}
	return out, nil
}

func idList(is []int) string {
	var s []string
	for _, i := range is {
		s = append(s, fmt.Sprintf("F%d", i+1))
	}
	return strings.Join(s, " ")
}

// clean judges a clean case's Importants shown, in its repository.
func (j *judge) clean(def caseDef, cr caseRun) (judgement, error) {
	var out judgement
	var shown []int
	for i, f := range cr.Findings {
		if f.shown() {
			shown = append(shown, i)
		}
	}
	out.shown = len(shown)
	if len(shown) == 0 {
		return out, nil
	}
	j.mu.Lock() // one tree a case, made once
	dir, material, err := j.tree(cr.Case)
	j.mu.Unlock()
	if err != nil {
		return out, err
	}
	user := fmt.Sprintf("# The change\n\n%s\n\n%s\n# The findings (%d)\n\n%s", def.Change, material, len(shown), findingsText(cr.Findings, shown))
	type said struct{ Verdict, Why string }
	var answers []map[string]said
	read := func(answer string) (map[string]said, error) {
		var a struct{ Findings map[string]said }
		if err := json.Unmarshal([]byte(jsonIn(answer)), &a); err != nil {
			return nil, fmt.Errorf("answer: %v", err)
		}
		for _, i := range shown {
			if v := a.Findings[fmt.Sprintf("F%d", i+1)].Verdict; !slices.Contains(cleanRank, v) {
				return nil, fmt.Errorf("answer: F%d: verdict %q", i+1, v)
			}
		}
		return a.Findings, nil
	}
	for n := 1; n <= 3; n++ {
		var got map[string]said
		_, fresh, tokens, err := j.asked(cleanSystem, user, dir, n, func(answer string) (err error) {
			got, err = read(answer)
			return err
		})
		if fresh {
			out.calls++
		}
		out.tokens += tokens
		if err != nil {
			return out, err
		}
		answers = append(answers, got)
		if n == 2 && !slices.ContainsFunc(shown, func(i int) bool {
			id := fmt.Sprintf("F%d", i+1)
			return answers[0][id].Verdict != answers[1][id].Verdict
		}) {
			break
		}
	}
	for _, i := range shown {
		id := fmt.Sprintf("F%d", i+1)
		f := cr.Findings[i]
		r := row{Arm: cr.Arm, Run: cr.Run, Case: cr.Case, Set: "clean", Item: id, Class: "important", Scored: "yes", Findings: id,
			Quote: firstWords(strings.TrimSpace(f.Title+" "+f.Body), 30)}
		for _, a := range answers {
			r.Judges = append(r.Judges, a[id].Verdict)
		}
		r.Verdict = settle(cleanRank, r.Judges)
		r.Why = answers[slices.Index(r.Judges, r.Verdict)][id].Why
		out.rows = append(out.rows, r)
	}
	return out, nil
}

func firstWords(s string, n int) string {
	w := strings.Fields(s)
	if len(w) > n {
		return strings.Join(w[:n], " ") + " …"
	}
	return strings.Join(w, " ")
}

// readFindings reads the findings file, numbering the runs that have none.
func readFindings(path string) ([]caseRun, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []caseRun
	seen := map[string]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 16<<20)
	for sc.Scan() {
		if len(bytes.TrimSpace(sc.Bytes())) == 0 {
			continue
		}
		var cr caseRun
		if err := json.Unmarshal(sc.Bytes(), &cr); err != nil {
			return nil, fmt.Errorf("%s, line %d: %v", path, len(out)+1, err)
		}
		cr.Case = strings.TrimPrefix(cr.Case, "reviewer/")
		seen[cr.Arm+"\x00"+cr.Case]++
		if cr.Run == 0 {
			cr.Run = seen[cr.Arm+"\x00"+cr.Case]
		}
		out = append(out, cr)
	}
	return out, sc.Err()
}

// score judges every case run, `parallel` at a time, and returns the rows
// in the file's order with each run's tallies.
func score(j *judge, defs map[string]caseDef, runs []caseRun, parallel int, progress func(done, total int)) ([]judgement, error) {
	out := make([]judgement, len(runs))
	errs := make([]error, len(runs))
	slots := make(chan struct{}, max(parallel, 1))
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	for i, cr := range runs {
		def, ok := defs[cr.Case]
		if !ok {
			return nil, fmt.Errorf("points.yaml has no case %q", cr.Case)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			if def.clean() {
				out[i], errs[i] = j.clean(def, cr)
			} else {
				out[i], errs[i] = j.points(def, cr)
			}
			if errs[i] != nil {
				errs[i] = fmt.Errorf("%s run %d, %s: %w", cr.Arm, cr.Run, cr.Case, errs[i])
			}
			mu.Lock()
			done++
			if progress != nil {
				progress(done, len(runs))
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out, errors.Join(errs...)
}

func tsv(js []judgement) string {
	var b strings.Builder
	b.WriteString("arm\trun\tcase\tset\titem\tclass\tscored\tverdict\tjudgements\tfindings\tquote\twhy\n")
	clean := strings.NewReplacer("\t", " ", "\n", " ")
	for _, j := range js {
		for _, r := range j.rows {
			fmt.Fprintf(&b, "%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.Arm, r.Run, r.Case, r.Set, r.Item, r.Class, r.Scored, r.Verdict,
				strings.Join(r.Judges, " / "), r.Findings, clean.Replace(r.Quote), clean.Replace(r.Why))
		}
	}
	return b.String()
}

func meanSD(xs []float64) (mean, sd float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	if len(xs) < 2 {
		return mean, 0
	}
	for _, x := range xs {
		sd += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(sd / float64(len(xs)-1))
}

func list(xs []float64) string {
	var s []string
	for _, x := range xs {
		s = append(s, strconv.FormatFloat(x, 'f', -1, 64))
	}
	return strings.Join(s, ", ")
}

var classes = []string{"out-of-diff", "weak-test", "not-delivered", "in-diff"}

// summary is the per-arm report: the points said a run, their kinds,
// what was said beside the points, the clean cases, the tokens.
func summary(runs []caseRun, js []judgement, judgeName string) string {
	type tally struct {
		said, half, refused, nit, kept, extra, shown, in, out float64
		cleanShown, falseAlarm, cleanNit, trueDefect          float64
		points, keptPoints                                    int
		class, classOf                                        map[string]float64
	}
	arms := []string{}
	per := map[string]map[int]*tally{}
	agree, twice, thirds, calls, tokens := 0, 0, 0, 0, 0
	for i, cr := range runs {
		if per[cr.Arm] == nil {
			per[cr.Arm] = map[int]*tally{}
			arms = append(arms, cr.Arm)
		}
		t := per[cr.Arm][cr.Run]
		if t == nil {
			t = &tally{class: map[string]float64{}, classOf: map[string]float64{}}
			per[cr.Arm][cr.Run] = t
		}
		j := js[i]
		calls, tokens = calls+j.calls, tokens+j.tokens
		t.in, t.out = t.in+float64(cr.TokensIn), t.out+float64(cr.TokensOut)
		switch setOf(cr.Case) {
		case "miss":
			t.extra, t.shown = t.extra+float64(j.extra), t.shown+float64(j.shown)
		case "clean":
			t.cleanShown += float64(j.shown)
		}
		for _, r := range j.rows {
			if len(r.Judges) >= 2 {
				twice++
				if r.Judges[0] == r.Judges[1] {
					agree++
				}
				if len(r.Judges) > 2 {
					thirds++
				}
			}
			switch {
			case r.Set == "clean":
				switch r.Verdict {
				case "false alarm":
					t.falseAlarm++
				case "nit":
					t.cleanNit++
				default:
					t.trueDefect++
				}
			case r.Set == "kept":
				t.keptPoints++
				if r.Verdict == "said" {
					t.kept++
				}
			case r.Scored == "yes":
				t.points++
				t.classOf[r.Class]++
				switch r.Verdict {
				case "said":
					t.said++
					t.class[r.Class]++
				case "partly":
					t.half++
				case "refused":
					t.refused++
				case "nit":
					t.nit++
				}
			}
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Reviewer replay, scored by meaning\n\nJudge: %s, blind to the arm, each judgement twice, a third on a difference.\n\n", judgeName)
	b.WriteString("## Misses: scored points said by an Important shown\n\n")
	b.WriteString("| Arm | Runs | Points | Said a run | Mean | SD | Partly | Refused by the engine's judge | Said as a nit |\n|---|---|---|---|---|---|---|---|---|\n")
	col := func(arm string, get func(*tally) float64) []float64 {
		var ns []int
		for n := range per[arm] {
			ns = append(ns, n)
		}
		sort.Ints(ns)
		var xs []float64
		for _, n := range ns {
			xs = append(xs, get(per[arm][n]))
		}
		return xs
	}
	sum := func(xs []float64) (s float64) {
		for _, x := range xs {
			s += x
		}
		return s
	}
	mean := func(xs []float64) float64 { m, _ := meanSD(xs); return m }
	for _, arm := range arms {
		said := col(arm, func(t *tally) float64 { return t.said })
		m, sd := meanSD(said)
		fmt.Fprintf(&b, "| %s | %d | %.0f | %s | %.1f | %.1f | %.1f | %.1f | %.1f |\n", arm, len(said),
			mean(col(arm, func(t *tally) float64 { return float64(t.points) })), list(said), m, sd,
			mean(col(arm, func(t *tally) float64 { return t.half })), mean(col(arm, func(t *tally) float64 { return t.refused })),
			mean(col(arm, func(t *tally) float64 { return t.nit })))
	}
	b.WriteString("\n## Misses by kind of point, and what else was said\n\n")
	b.WriteString("| Arm | Out of the diff | Weak or missing test | Issue not delivered | In-diff code | Importants shown a run | Of them on no point |\n|---|---|---|---|---|---|---|\n")
	for _, arm := range arms {
		fmt.Fprintf(&b, "| %s |", arm)
		for _, c := range classes {
			got, of := sum(col(arm, func(t *tally) float64 { return t.class[c] })), sum(col(arm, func(t *tally) float64 { return t.classOf[c] }))
			fmt.Fprintf(&b, " %.0f/%.0f (%.0f %%) |", got, of, 100*got/math.Max(of, 1))
		}
		fmt.Fprintf(&b, " %.1f | %.1f |\n", mean(col(arm, func(t *tally) float64 { return t.shown })), mean(col(arm, func(t *tally) float64 { return t.extra })))
	}
	b.WriteString("\n## Kept catches, clean changes, tokens\n\n")
	b.WriteString("| Arm | Kept said a run | Clean: Importants shown a run | False alarms a run | Nits a run | True defects a run | Tokens a run (in + out) |\n|---|---|---|---|---|---|---|\n")
	for _, arm := range arms {
		fmt.Fprintf(&b, "| %s | %s of %.0f | %s | %s | %s | %s | %.0fk |\n", arm,
			list(col(arm, func(t *tally) float64 { return t.kept })), mean(col(arm, func(t *tally) float64 { return float64(t.keptPoints) })),
			list(col(arm, func(t *tally) float64 { return t.cleanShown })), list(col(arm, func(t *tally) float64 { return t.falseAlarm })),
			list(col(arm, func(t *tally) float64 { return t.cleanNit })), list(col(arm, func(t *tally) float64 { return t.trueDefect })),
			mean(col(arm, func(t *tally) float64 { return t.in + t.out }))/1000)
	}
	fmt.Fprintf(&b, "\n## The judge\n\n- Judged twice: %d items; the two agree on %d (%.0f %%); a third judgement settled %d.\n", twice, agree, 100*float64(agree)/math.Max(float64(twice), 1), thirds)
	fmt.Fprintf(&b, "- This run asked the judge %d times, %dk tokens; what the cache held was not asked again.\n", calls, tokens/1000)
	return b.String()
}

// claude asks `claude -p`, with no tool, or the read tools in dir.
func claude(model, effort string) asker {
	return func(system, user, dir string) (string, int, error) {
		sys, err := os.CreateTemp("", "meaning-system-*.md")
		if err != nil {
			return "", 0, err
		}
		defer func() { _ = os.Remove(sys.Name()) }()
		if _, err := sys.WriteString(system); err != nil {
			return "", 0, err
		}
		_ = sys.Close()
		args := []string{"-p", "--output-format", "json", "--strict-mcp-config", "--no-session-persistence", "--system-prompt-file", sys.Name(), "--model", model, "--effort", effort}
		if dir == "" {
			args = append(args, "--tools", "")
			dir = os.TempDir() // no project instructions of the caller's folder
		} else {
			args = append(args, "--tools", "Read,Glob,Grep", "--permission-mode", "dontAsk", "--max-turns", "40")
		}
		cmd := exec.Command("claude", args...)
		cmd.Dir, cmd.Stdin = dir, strings.NewReader(user)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		var res struct {
			Result  string `json:"result"`
			IsError bool   `json:"is_error"`
			Usage   struct {
				In     int `json:"input_tokens"`
				Read   int `json:"cache_read_input_tokens"`
				Create int `json:"cache_creation_input_tokens"`
				Out    int `json:"output_tokens"`
			} `json:"usage"`
		}
		if jerr := json.Unmarshal(stdout, &res); jerr != nil {
			return "", 0, fmt.Errorf("claude: %v: %.200s %.200s", errors.Join(err, jerr), stdout, stderr.String())
		}
		tokens := res.Usage.In + res.Usage.Read + res.Usage.Create + res.Usage.Out
		if err != nil || res.IsError {
			return "", tokens, fmt.Errorf("claude: %v: %.300s", err, res.Result)
		}
		return res.Result, tokens, nil
	}
}

// command asks any command: the question on its stdin, its answer on stdout.
func command(line string) asker {
	return func(system, user, dir string) (string, int, error) {
		cmd := exec.Command("sh", "-c", line)
		cmd.Dir, cmd.Stdin = dir, strings.NewReader(system+"\n\n"+user)
		out, err := cmd.Output()
		return string(out), 0, err
	}
}

// askerOf reads the judge's name: claude:<model>@<effort>, or cmd:<command>.
func askerOf(spec string) (asker, error) {
	if line, ok := strings.CutPrefix(spec, "cmd:"); ok {
		return command(line), nil
	}
	rest, ok := strings.CutPrefix(spec, "claude:")
	model, effort, at := strings.Cut(rest, "@")
	if !ok || !at || model == "" || effort == "" {
		return nil, fmt.Errorf("judge %q: claude:<model>@<effort> or cmd:<command>", spec)
	}
	return claude(model, effort), nil
}

// trees gives a clean case its repository at the change's head, from the
// replay mirrors (~/.cache/workline/eval-mirrors), and the issue, the
// pull request and the diff as the judge's material.
func trees(cases, mirrors, work string) func(string) (string, string, error) {
	type made struct {
		dir, material string
		err           error
	}
	seen := map[string]made{}
	return func(kase string) (string, string, error) {
		if m, ok := seen[kase]; ok {
			return m.dir, m.material, m.err
		}
		dir, material, err := tree(filepath.Join(cases, kase+".yaml"), mirrors, filepath.Join(work, kase))
		seen[kase] = made{dir, material, err}
		return dir, material, err
	}
}

func tree(file, mirrors, dir string) (string, string, error) {
	var c struct {
		Given struct {
			Clone struct{ Repo, Base, Head string }
			Forge struct {
				Issues []struct {
					ID          int `yaml:"id"`
					Title, Body string
				}
				MergeRequests []struct {
					ID          int `yaml:"id"`
					Title, Body string
				} `yaml:"merge-requests"`
			}
		}
	}
	data, err := os.ReadFile(file)
	if err == nil {
		err = yaml.Unmarshal(data, &c)
	}
	if err != nil {
		return "", "", err
	}
	mirror := filepath.Join(mirrors, strings.ReplaceAll(c.Given.Clone.Repo, "/", "_")+".git")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	archive := exec.Command("sh", "-c", `git --git-dir="$1" archive "$2" | tar -x -C "$3"`, "sh", mirror, c.Given.Clone.Head, dir)
	if out, err := archive.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("no tree of %s at %.9s (a replay run fetches the mirror): %v: %s", c.Given.Clone.Repo, c.Given.Clone.Head, err, out)
	}
	diff, err := exec.Command("git", "--git-dir", mirror, "diff", c.Given.Clone.Base+"..."+c.Given.Clone.Head).Output()
	if err != nil {
		return "", "", fmt.Errorf("diff: %v", err)
	}
	if len(diff) > 60000 {
		diff = append(diff[:60000], "\n[cut: read the files]\n"...)
	}
	var b strings.Builder
	for _, i := range c.Given.Forge.Issues {
		fmt.Fprintf(&b, "## Issue #%d: %s\n\n%s\n\n", i.ID, i.Title, i.Body)
	}
	for _, m := range c.Given.Forge.MergeRequests {
		fmt.Fprintf(&b, "## Pull request #%d: %s\n\n%s\n\n", m.ID, m.Title, m.Body)
	}
	fmt.Fprintf(&b, "## The diff\n\n```diff\n%s```\n", diff)
	return dir, b.String(), nil
}

func main() {
	findings := flag.String("findings", "", "the runs to score: a JSON line a case run, {arm, run, case, tokens_in, tokens_out, findings: [{place, severity, refused, title, body}]}")
	out := flag.String("out", "meaning-scores", "the folder scores.tsv, summary.md and the judgements kept go to")
	spec := flag.String("judge", "claude:opus@medium", "the judge, the same for every arm: claude:<model>@<effort>, or cmd:<command> answering on stdout the question on its stdin")
	parallel := flag.Int("parallel", 4, "judge calls at a time")
	cases := flag.String("cases", "tests/evaluation/replay/reviewer", "the replay cases, for a clean case's issue and commits")
	flag.Parse()
	if *findings == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*findings, *out, *spec, *cases, *parallel); err != nil {
		fmt.Fprintln(os.Stderr, "meaning:", err)
		os.Exit(1)
	}
}

// run scores the findings file into out. What was judged is written even
// when a judgement failed: the error names what was not.
func run(findings, out, spec, cases string, parallel int) error {
	defs, err := loadPoints()
	if err != nil {
		return err
	}
	runs, err := readFindings(findings)
	if err != nil {
		return err
	}
	ask, err := askerOf(spec)
	if err != nil {
		return err
	}
	cache, _ := os.UserCacheDir()
	work, err := os.MkdirTemp("", "meaning-trees-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()
	j := &judge{ask: ask, cache: filepath.Join(out, "judgments"), pause: 5 * time.Second, tree: trees(cases, filepath.Join(cache, "workline", "eval-mirrors"), work)}
	js, err := score(j, defs, runs, parallel, func(done, total int) {
		if done%20 == 0 || done == total {
			fmt.Fprintf(os.Stderr, "%d/%d case runs judged\n", done, total)
		}
	})
	return errors.Join(err, os.MkdirAll(out, 0o755),
		os.WriteFile(filepath.Join(out, "scores.tsv"), []byte(tsv(js)), 0o644),
		os.WriteFile(filepath.Join(out, "summary.md"), []byte(summary(runs, js, spec)), 0o644))
}

// loadPoints reads points.yaml strictly.
func loadPoints() (map[string]caseDef, error) {
	var defs map[string]caseDef
	dec := yaml.NewDecoder(bytes.NewReader(pointsYAML))
	dec.KnownFields(true)
	return defs, dec.Decode(&defs)
}
