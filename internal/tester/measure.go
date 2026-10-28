package tester

// The curator's measures (ADR-0055, step 6), no agent: the project's test
// command run on HEAD, curator-runs times in one fresh checkout; each
// test's time and result read from its JUnit report; the corpus's count,
// total time, slowest tests, red and flaky ones, those never red over a
// window of measures; the coverage the command writes, where it does; and
// whether a mutation tool exists for the project's language. Kept across
// measures in a history file the CI passes from one run to the next, as
// an artifact: no store of its own. Nothing is removed here: step 7 acts.

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/verdict"
)

// History is what the curator keeps from one measure to the next.
type History struct {
	Version int `json:"version"`
	// Count is the measures taken; each test's measures are numbered by it.
	Count    int              `json:"count"`
	Measures []MeasureTotal   `json:"measures"`
	Tests    map[string]*Kept `json:"tests"`
}

// MeasureTotal is one measure's totals, for the trend.
type MeasureTotal struct {
	N       int     `json:"n"`
	Date    string  `json:"date,omitempty"`
	Commit  string  `json:"commit,omitempty"`
	Tests   int     `json:"tests"`
	Seconds float64 `json:"seconds"`
}

// Kept is a test's record across measures.
type Kept struct {
	First   int     `json:"first"` // the measure it was first seen in
	Last    int     `json:"last"`  // the last it ran in
	Seen    int     `json:"seen"`  // the measures it ran in
	Runs    int     `json:"runs"`
	Red     int     `json:"red,omitempty"`      // the runs it was red in
	Flaky   int     `json:"flaky,omitempty"`    // the measures it was red and green in
	LastRed int     `json:"last-red,omitempty"` // the last measure it was red in
	Seconds float64 `json:"seconds"`            // its time, the mean of the last measure's runs
}

// historyVersion is the history file's form.
const historyVersion = 1

// trendKept caps the measures' totals kept: a year of weekly ones.
const trendKept = 52

// Measured is a measure's result: the verdict, and the lines a person
// reads in the job's summary.
type Measured struct {
	verdict.Verdict
	Lines []string
}

// timed is a test as one run's report holds it.
type timed struct {
	res     result
	seconds float64
}

// Measure runs the curator's measures on HEAD. history, when not empty,
// is the file read before and written after: a missing one is the first
// measure.
func Measure(repo, history string, s Settings) Measured {
	start := time.Now()
	if v := unset(s); v != nil {
		v.Summary = "the measure did not run"
		return Measured{Verdict: *v}
	}
	command := s.Command
	if strings.TrimSpace(s.CuratorCommand) != "" {
		command = s.CuratorCommand
	}
	runs := max(s.CuratorRuns, 1)
	h, first, err := readHistory(history)
	if err != nil {
		return Measured{Verdict: *refuse("the measure did not run", verdict.Finding{Rule: "history-unreadable", Where: history, Message: err.Error()})}
	}
	tree, done, f := checkout(repo, "HEAD")
	if f != nil {
		return Measured{Verdict: *refuse("the measure did not run", *f)}
	}
	defer done()
	head, _ := git(tree, "rev-parse", "HEAD")
	var reports []map[string]timed
	var suiteSeconds float64
	for i := 1; i <= runs; i++ {
		rs := s
		rs.Command = command
		rep, total, f := timedRun(tree, rs, fmt.Sprintf("on HEAD, run %d of %d", i, runs))
		if f != nil {
			return Measured{Verdict: *refuse("the measure did not run", *f)}
		}
		reports = append(reports, rep)
		suiteSeconds += total
	}
	suiteSeconds /= float64(runs)

	ids := map[string]bool{}
	for _, r := range reports {
		for id := range r {
			ids[id] = true
		}
	}
	all := slices.Sorted(func(yield func(string) bool) {
		for id := range ids {
			if !yield(id) {
				return
			}
		}
	})
	parents := parentsOf(all)
	h.Count++
	n := h.Count
	v := verdict.Verdict{Status: verdict.Pass}
	type slow struct {
		id      string
		seconds float64
	}
	var times []slow
	for _, id := range all {
		k := h.Tests[id]
		if k == nil {
			k = &Kept{First: n}
			h.Tests[id] = k
		}
		red, ran, sum := 0, 0, 0.0
		for _, r := range reports {
			t, ok := r[id]
			if !ok {
				continue
			}
			ran++
			sum += t.seconds
			if t.res == failed {
				red++
			}
		}
		k.Last, k.Seen, k.Runs, k.Red = n, k.Seen+1, k.Runs+ran, k.Red+red
		k.Seconds = sum / float64(max(ran, 1))
		if red > 0 {
			k.LastRed = n
		}
		switch {
		case red > 0 && red < ran:
			k.Flaky++
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-flaky", Level: "warn", Where: id,
				Message: fmt.Sprintf("red in %d of %d runs on %s: flaky; an issue naming it and the ecosystem's skip, never deleted (ADR-0055)", red, ran, short(head))})
		case red > 0:
			v.Findings = append(v.Findings, verdict.Finding{Rule: "test-red", Level: "warn", Where: id,
				Message: fmt.Sprintf("red in each of the %d runs on %s", ran, short(head))})
		}
		if !parents[id] {
			times = append(times, slow{id, k.Seconds})
		}
	}
	// A test gone longer than the window is forgotten.
	window := max(s.CuratorWindow, 1)
	for id, k := range h.Tests {
		if k.Last <= n-window {
			delete(h.Tests, id)
		}
	}
	h.Measures = append(h.Measures, MeasureTotal{N: n, Date: time.Now().UTC().Format("2006-01-02"), Commit: short(head), Tests: len(all), Seconds: round(suiteSeconds)})
	if len(h.Measures) > trendKept {
		h.Measures = h.Measures[len(h.Measures)-trendKept:]
	}

	var lines []string
	say := func(format string, a ...any) { lines = append(lines, fmt.Sprintf(format, a...)) }
	say("- **corpus**: %d test%s, %s s, on %s; the measure took %s s, %d run%s", len(all), plural(len(all)), num(suiteSeconds),
		short(head), num(time.Since(start).Seconds()), runs, plural(runs))
	switch {
	case history == "":
		say("- **history**: none asked (`--history`): no history kept, flakiness over time unmeasured")
	case first:
		say("- **history**: the first measure: no history before it")
	default:
		if len(h.Measures) >= 2 {
			p := h.Measures[len(h.Measures)-2]
			since := fmt.Sprint(p.N)
			if p.Date != "" {
				since += ", " + p.Date
			}
			say("- **trend**: %+d test%s, %+.1f s since measure %s", len(all)-p.Tests, plural(abs(len(all)-p.Tests)), suiteSeconds-p.Seconds, since)
		}
	}
	if s.SuiteMaxSeconds > 0 && suiteSeconds > float64(s.SuiteMaxSeconds) {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "corpus-slow", Level: "warn", Where: "suite-max-seconds",
			Message: fmt.Sprintf("the corpus takes %s s, over suite-max-seconds (%d s)", num(suiteSeconds), s.SuiteMaxSeconds)})
	}
	sort.SliceStable(times, func(i, j int) bool { return times[i].seconds > times[j].seconds })
	if top := min(max(s.CuratorSlowest, 0), len(times)); top > 0 {
		say("- **slowest %d**:", top)
		for _, t := range times[:top] {
			say("  - `%s` — %s s", t.id, num(t.seconds))
		}
	}
	var reds []string
	for _, id := range all {
		k := h.Tests[id]
		if k.Red > 0 {
			reds = append(reds, fmt.Sprintf("  - `%s` — red in %d run%s of %d, flaky in %d measure%s of %d, last red in measure %d",
				id, k.Red, plural(k.Red), k.Runs, k.Flaky, plural(k.Flaky), k.Seen, k.LastRed))
		}
	}
	if len(reds) > 0 {
		say("- **red or flaky, over the measures kept**: %d", len(reds))
		lines = append(lines, reds...)
	}
	if history != "" && n >= window {
		quiet := 0
		for _, id := range all {
			if k := h.Tests[id]; k.First <= n-window+1 && k.LastRed <= n-window {
				quiet++
			}
		}
		say("- **never red in the last %d measures**: %d of %d tests (candidates step 7 weighs, never removed on this alone)", window, quiet, len(all))
	}
	lines = append(lines, coverageLines(tree, s.CoverageReport)...)
	lines = append(lines, mutationLine(tree))

	if history != "" {
		if err := writeHistory(history, h); err != nil {
			return Measured{Verdict: *refuse("the measure was not kept", verdict.Finding{Rule: "history-unwritten", Where: history, Message: err.Error()})}
		}
	}
	v.Summary = fmt.Sprintf("%d test%s, %s s on %s", len(all), plural(len(all)), num(suiteSeconds), short(head))
	if history == "" {
		v.Summary += "; no history kept"
	}
	return Measured{Verdict: v, Lines: lines}
}

// timedRun runs the command in tree and reads each test's result and time,
// and the suites' total time.
func timedRun(tree string, s Settings, where string) (map[string]timed, float64, *verdict.Finding) {
	if _, f := runIn(tree, s, where); f != nil {
		return nil, 0, f
	}
	found, _ := filepath.Glob(filepath.Join(tree, filepath.FromSlash(s.Report)))
	out := map[string]timed{}
	var total float64
	for _, file := range found {
		t, err := readTimes(file, out)
		if err != nil {
			return nil, 0, &verdict.Finding{Rule: "report-missing", Where: s.Report, Message: err.Error()}
		}
		total += t
	}
	return out, total, nil
}

// readTimes adds each test of a JUnit report to out, its result and time,
// and returns the report's total time: the suites' that hold tests, or
// their tests' when a suite gives none.
func readTimes(file string, out map[string]timed) (float64, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return 0, err
	}
	var root suite
	if err := xml.Unmarshal(data, &root); err != nil {
		return 0, err
	}
	var total float64
	var walk func(s suite)
	walk = func(s suite) {
		cases := 0.0
		for _, c := range s.Cases {
			id := c.Name
			if c.Class != "" {
				id = c.Class + "." + c.Name
			}
			r := passed
			switch {
			case c.Failure != nil || c.Error != nil:
				r = failed
			case c.Skipped != nil:
				r = skipped
			}
			sec, _ := strconv.ParseFloat(c.Time, 64)
			cases += sec
			if prev, ok := out[id]; !ok || r > prev.res {
				out[id] = timed{r, prev.seconds + sec}
			} else {
				out[id] = timed{prev.res, prev.seconds + sec}
			}
		}
		if len(s.Cases) > 0 {
			if sec, err := strconv.ParseFloat(s.Time, 64); err == nil && sec > 0 {
				total += sec
			} else {
				total += cases
			}
		}
		for _, sub := range s.Suites {
			walk(sub)
		}
	}
	walk(root)
	return total, nil
}

// parentsOf is the tests holding subtests (Go's id/sub): their time is
// their subtests', left out of the slowest.
func parentsOf(ids []string) map[string]bool {
	parents := map[string]bool{}
	for _, id := range ids {
		for i := strings.IndexByte(id, '/'); i >= 0; i = nextSlash(id, i) {
			parents[id[:i]] = true
		}
	}
	return parents
}

func nextSlash(id string, i int) int {
	if j := strings.IndexByte(id[i+1:], '/'); j >= 0 {
		return i + 1 + j
	}
	return -1
}

func readHistory(file string) (History, bool, error) {
	h := History{Version: historyVersion, Tests: map[string]*Kept{}}
	if file == "" {
		return h, true, nil
	}
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return h, true, nil
	}
	if err != nil {
		return h, false, err
	}
	if err := json.Unmarshal(data, &h); err != nil {
		return h, false, fmt.Errorf("not the curator's history: %v", err)
	}
	if h.Version != historyVersion {
		return h, false, fmt.Errorf("history version %d, this engine reads %d", h.Version, historyVersion)
	}
	if h.Tests == nil {
		h.Tests = map[string]*Kept{}
	}
	return h, false, nil
}

func writeHistory(file string, h History) error {
	data, err := json.MarshalIndent(h, "", " ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(file); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(file, append(data, '\n'), 0o644)
}

// num is seconds for a person: two decimals at most.
func num(f float64) string { return strconv.FormatFloat(round(f), 'f', -1, 64) }

func round(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
