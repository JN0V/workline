package evaluation

// The judge's replay (ADR-0045, #362): the 40 questions the reviewer's
// judge was asked on workline's pull requests, measured on 2026-10-14
// (docs/research/judge-measured.md). Each is kept in judge-replay/<item>/
// as the run asked it (question.yaml) and as the judge answered it then
// (answer.yaml), both copied from the run's workline-runs artifact, with
// the truth the measure settled (item.yaml). TestJudgeReplay asks them
// again, of the judge WORKLINE_EVAL names, in one of two arms: today's
// judge, or the judge that reads (WORKLINE_EVAL_REPLAY_ROUNDS); the set is
// read, and its scoring proved, with no agent (checks_test.go).

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/builtin/reviewer"
	wjudge "github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/role"
)

const replayDir = "judge-replay"

// replayItem is one verdict measured: where the judge was asked, what it
// said then, and the truth the measure settled.
type replayItem struct {
	Name      string `yaml:"-"` // its folder
	MeasureID string `yaml:"measure-id"`
	PR        int    `yaml:"pull-request"`
	Lens      string `yaml:"lens"`
	Title     string `yaml:"title"`
	Where     string `yaml:"where"`  // the finding's cause, file:line at commit
	Commit    string `yaml:"commit"` // the head the reviewer read
	// artifact: question.yaml and answer.yaml copied from the run's
	// workline-runs artifact; rebuilt: remade from git at commit, the
	// artifact gone.
	Origin string `yaml:"origin"`
	From   struct {
		Run     int64  `yaml:"run"`     // the workflow run
		Path    string `yaml:"path"`    // the question's folder in its artifact
		Expires string `yaml:"expires"` // when the artifact was due to expire
	} `yaml:"from"`
	Said       string `yaml:"said"` // yes or no, as answer.yaml holds it
	SaidTokens struct {
		In  int `yaml:"in"`
		Out int `yaml:"out"`
	} `yaml:"said-tokens"`
	Truth     bool   `yaml:"truth"`      // the finding holds
	SettledBy string `yaml:"settled-by"` // agreement: the judge and a rater; panel: three agents, both readings
	Matters   *bool  `yaml:"matters"`    // worth raising to a person, by the panel's majority; nil when not asked
	Evidence  string `yaml:"evidence"`   // the code that settled it

	Question, Material string `yaml:"-"` // what the judge was asked (question.yaml)
	Then               struct {
		Yes   *bool  `yaml:"yes"`
		Why   string `yaml:"why"`
		Model string `yaml:"model"`
	} `yaml:"-"` // what it answered then (answer.yaml)
}

// readReplay reads the set, an item a folder, as strictly as a case: a key
// no field names fails it.
func readReplay(dir string) ([]replayItem, error) {
	files, _ := filepath.Glob(filepath.Join(dir, "*", "item.yaml"))
	sort.Strings(files)
	var items []replayItem
	for _, f := range files {
		var it replayItem
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&it); err != nil {
			return nil, fmt.Errorf("%s: %v", f, err)
		}
		it.Name = filepath.Base(filepath.Dir(f))
		var q struct{ Question, Material, Author string }
		if data, err = os.ReadFile(filepath.Join(filepath.Dir(f), "question.yaml")); err == nil {
			err = yaml.Unmarshal(data, &q)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: question: %v", it.Name, err)
		}
		it.Question, it.Material = q.Question, q.Material
		if data, err = os.ReadFile(filepath.Join(filepath.Dir(f), "answer.yaml")); err == nil {
			err = yaml.Unmarshal(data, &it.Then)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: answer: %v", it.Name, err)
		}
		items = append(items, it)
	}
	if len(items) == 0 { // a replay of nothing would pass, its tally all zeros
		return nil, fmt.Errorf("%s: no item (<item>/item.yaml)", dir)
	}
	return items, nil
}

// replayAnswer is what a judge came to on an item, as a run would show
// the finding: verified (yes), dropped (no; for a judge that reads, its
// quote found again), or not verified (unsure, a no unquoted, no answer).
type replayAnswer struct {
	Asked  bool   // the item was asked in this run
	Said   string // yes, no, unsure; "" when it gave no answer
	Shown  string // verified, dropped, or not verified: why
	Why    string
	Err    error
	Calls  []agent.Call
	Reads  []string // the reads it asked
	Served int      // rounds of reads served
	Lines  int      // lines served
	Quoted string   // a dropping no's quote: path:line
}

func (a replayAnswer) dropped() bool  { return a.Shown == "dropped" }
func (a replayAnswer) verified() bool { return a.Shown == "verified" }

// right: a true finding verified, a false one dropped.
func (a replayAnswer) right(it replayItem) bool {
	return it.Truth && a.verified() || !it.Truth && a.dropped()
}

// asJudgedThen is what the judge said then, as a run showed it.
func asJudgedThen(it replayItem) replayAnswer {
	a := replayAnswer{Asked: true, Said: it.Said, Why: it.Then.Why, Shown: "verified"}
	if it.Said == "no" {
		a.Shown = "dropped"
	}
	return a
}

// tokenStop stops a replay once it has spent its tokens in, all calls
// together, past what earlier runs of the same replay spent.
type tokenStop struct {
	mu        sync.Mutex
	max, inTo int
	stopped   bool
}

// ask is a judge's call, unless the replay spent its tokens.
func (s *tokenStop) ask(call func() (wjudge.Answer, error)) (wjudge.Answer, error) {
	s.mu.Lock()
	if s.max > 0 && s.inTo >= s.max {
		s.stopped = true
		s.mu.Unlock()
		return wjudge.Answer{}, fmt.Errorf("%s %d tokens in, its stop at %d", stopSaid, s.inTo, s.max)
	}
	s.mu.Unlock()
	a, err := call()
	s.mu.Lock()
	s.inTo += a.Call.TokensIn
	s.mu.Unlock()
	return a, err
}

// stopSaid opens what a call refused at the stop says.
const stopSaid = "not asked: the replay spent"

func stopped(err error) bool { return err != nil && strings.HasPrefix(err.Error(), stopSaid) }

func (s *tokenStop) spent() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inTo
}

// askReplay puts an item's question to the judge spec names, as the
// engine asks it (askQuestions): the question, the material, the judge's
// facets of today; a no drops the finding.
func askReplay(spec string, judgeRole *role.Role, it replayItem, stop *tokenStop) replayAnswer {
	ans, err := stop.ask(func() (wjudge.Answer, error) { return wjudge.Ask(spec, judgeRole, it.Question, it.Material) })
	a := replayAnswer{Asked: true, Why: ans.Why, Err: err}
	if !stopped(err) { // a call refused at the stop was never made
		a.Calls = []agent.Call{ans.Call}
	}
	switch {
	case err != nil:
		a.Shown = "not verified: " + err.Error()
	case ans.Yes:
		a.Said, a.Shown = "yes", "verified"
	default:
		a.Said, a.Shown = "no", "dropped"
	}
	return a
}

// askReading puts it to the judge that reads (ADR-0045), as the reviewer
// does when judge-read-rounds is on: reads served from git at the item's
// commit, in repo, within rounds and lines.
func askReading(spec string, judgeRole *role.Role, it replayItem, repo string, rounds, lines int, stop *tokenStop) replayAnswer {
	cause, _, _ := strings.Cut(it.Where, ":")
	j := reviewer.JudgeReading(repo, it.Commit, it.Question, it.Material, cause, rounds, lines, func(q, m string) (wjudge.Answer, error) {
		return stop.ask(func() (wjudge.Answer, error) { return wjudge.AskReading(spec, judgeRole, q, m) })
	})
	a := replayAnswer{Asked: true, Said: j.Said, Shown: j.Shown, Why: j.Why, Calls: j.Calls, Reads: j.Asked, Served: j.Rounds, Lines: j.Lines, Quoted: j.Quoted}
	if j.Said == "" {
		a.Err = fmt.Errorf("%s", j.Why)
		if stopped(a.Err) && len(a.Calls) > 0 {
			a.Calls = a.Calls[:len(a.Calls)-1] // the call refused at the stop, never made
		}
	}
	return a
}

// replayTally is a replay against the measure's truth, and against what
// the judge said then (docs/research/judge-measured.md).
type replayTally struct {
	Items, Right               int
	YesThen, YesThenRight      int      // the items it said yes to then; right now
	NoThen, NoThenRight        int      // the items it said no to then; right now
	DroppedThen                int      // true findings it dropped then
	StillDropped               []string // of them, dropped now
	FalseThen                  int      // false findings it dropped then
	LetThrough, LetUnchecked   int      // of them, not dropped now; shown not verified among those
	True, TrueDropped          int      // every true finding; dropped now
	False, FalseKept           int      // every false finding; not dropped now
	Verified, VerifiedTrue     int      // its yes; of them, true findings
	Unsure, Unquoted, NoAnswer int
	SameAsThen                 int // yes or no as then
	Reads, Rounds, Lines       int // reads asked; rounds and lines served
	Calls, TokensIn, TokensOut int
}

func tallyReplay(items []replayItem, now []replayAnswer) replayTally {
	var t replayTally
	for i, it := range items {
		a := now[i]
		if !a.Asked {
			continue
		}
		t.Items++
		for _, c := range a.Calls {
			t.Calls++
			t.TokensIn += c.TokensIn
			t.TokensOut += c.TokensOut
		}
		t.Reads, t.Rounds, t.Lines = t.Reads+len(a.Reads), t.Rounds+a.Served, t.Lines+a.Lines
		right := a.right(it)
		if right {
			t.Right++
		}
		if a.Said == it.Said {
			t.SameAsThen++
		}
		if it.Said == "yes" {
			t.YesThen++
			if right {
				t.YesThenRight++
			}
		} else {
			t.NoThen++
			if right {
				t.NoThenRight++
			}
		}
		switch {
		case it.Truth && it.Said == "no":
			t.DroppedThen++
			if a.dropped() {
				t.StillDropped = append(t.StillDropped, it.Name)
			}
		case !it.Truth && it.Said == "no":
			t.FalseThen++
			if !a.dropped() {
				t.LetThrough++
				if !a.verified() {
					t.LetUnchecked++
				}
			}
		}
		if it.Truth {
			t.True++
			if a.dropped() {
				t.TrueDropped++
			}
		} else {
			t.False++
			if !a.dropped() {
				t.FalseKept++
			}
		}
		if a.verified() {
			t.Verified++
			if it.Truth {
				t.VerifiedTrue++
			}
		}
		switch {
		case a.Said == "unsure":
			t.Unsure++
		case a.Said == "":
			t.NoAnswer++
		case strings.Contains(a.Shown, "judge-no-unquoted"):
			t.Unquoted++
		}
	}
	return t
}

func (t replayTally) String() string {
	perJudge := func(n int) int {
		if t.Items == 0 {
			return 0
		}
		return n / t.Items
	}
	return fmt.Sprintf("%d/%d right; said yes then: %d/%d right; said no then: %d/%d right; "+
		"true findings dropped then, dropped now: %d of %d %v; false findings dropped then, let through now: %d of %d (%d not verified); "+
		"every true finding dropped: %d of %d; every false one let through: %d of %d; its yes right: %d of %d; "+
		"unsure %d, no unquoted %d, no answer %d; same as then %d of %d; "+
		"reads asked %d, rounds served %d, lines %d; calls %d; tokens in %d, out %d (a judge: %d in, %d out)",
		t.Right, t.Items, t.YesThenRight, t.YesThen, t.NoThenRight, t.NoThen,
		len(t.StillDropped), t.DroppedThen, t.StillDropped, t.LetThrough, t.FalseThen, t.LetUnchecked,
		t.TrueDropped, t.True, t.FalseKept, t.False, t.VerifiedTrue, t.Verified,
		t.Unsure, t.Unquoted, t.NoAnswer, t.SameAsThen, t.Items,
		t.Reads, t.Rounds, t.Lines, t.Calls, t.TokensIn, t.TokensOut, perJudge(t.TokensIn), perJudge(t.TokensOut))
}

// envInt reads a number from the environment, def when unset.
func envInt(t *testing.T, name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("%s=%q: a number", name, v)
	}
	return n
}

// TestJudgeReplay asks the 40 questions again of the judge WORKLINE_EVAL
// names, today's judge, or with WORKLINE_EVAL_REPLAY_ROUNDS the judge that
// reads, served from git at each item's commit. A line of results.tsv an
// item, its tally logged. Past WORKLINE_EVAL_TOKENS_MAX tokens in, earlier
// runs counted (WORKLINE_EVAL_TOKENS_BEFORE), no call starts; those in
// flight finish, -parallel at most; the test fails, said
// (docs/spec/conformance/evaluation.md).
func TestJudgeReplay(t *testing.T) {
	spec := os.Getenv("WORKLINE_EVAL")
	if spec == "" {
		t.Skip("set WORKLINE_EVAL=claude:opus to replay the judge's 40 measured questions: it calls a real agent and costs tokens")
	}
	items, err := readReplay(replayDir)
	if err != nil {
		t.Fatal(err)
	}
	judgeRole, err := role.Load("../../roles", "judge")
	if err != nil {
		t.Fatal(err)
	}
	rounds, lines := envInt(t, "WORKLINE_EVAL_REPLAY_ROUNDS", 0), envInt(t, "WORKLINE_EVAL_REPLAY_LINES", 400)
	stop := &tokenStop{max: envInt(t, "WORKLINE_EVAL_TOKENS_MAX", 2100000), inTo: envInt(t, "WORKLINE_EVAL_TOKENS_BEFORE", 0)}
	repo, _ := filepath.Abs("../..")
	arm := "judge"
	if rounds > 0 {
		arm = "judge-reads"
		for _, it := range items { // each item's commit, to read at
			if err := exec.Command("git", "-C", repo, "cat-file", "-e", it.Commit+"^{commit}").Run(); err != nil {
				t.Fatalf("%s: commit %s is not here: git fetch origin pull/%d/head", it.Name, it.Commit, it.PR)
			}
		}
	}
	version := worklineVersion()
	now := make([]replayAnswer, len(items))
	t.Run("items", func(t *testing.T) {
		for i, it := range items {
			t.Run(it.Name, func(t *testing.T) {
				t.Parallel()
				start := time.Now()
				var a replayAnswer
				if rounds > 0 {
					a = askReading(spec, judgeRole, it, repo, rounds, lines, stop)
				} else {
					a = askReplay(spec, judgeRole, it, stop)
				}
				now[i] = a
				score, failed := "1/1", ""
				if !a.right(it) {
					score, failed = "0/1", replayFailed(it, a)
				}
				t.Logf("%s — %s; then %s, now %s (%s%s): %.300s; tokens in so far: %d (stop at %d)", it.Name, score, it.Said, orDash(a.Said),
					a.Shown, readsLine(a), a.Why, stop.spent(), stop.max)
				var calls []call
				for _, c := range a.Calls {
					calls = append(calls, call{c.Effort, c.Model, c.TokensIn, c.TokensOut, c.CostUSD, "judge"})
				}
				models, efforts, tokensIn, tokensOut, cost := answeredBy(calls)
				line := strings.Join([]string{time.Now().UTC().Format(time.RFC3339), version, spec, models, efforts,
					arm + "/" + it.Name, score, fmt.Sprint(len(a.Calls)), tokensIn, tokensOut, cost, fmt.Sprintf("%.0f", time.Since(start).Seconds()),
					failed, "", ""}, "\t")
				if err := appendResult(line); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
	t.Logf("judge replay, %s, %s: %s; the replay's tokens in, earlier runs counted: %d", spec, arm, tallyReplay(items, now), stop.spent())
	if stop.stopped {
		t.Errorf("the replay stopped at %d tokens in (WORKLINE_EVAL_TOKENS_MAX %d): the items after it were not asked", stop.spent(), stop.max)
	}
}

// readsLine says what a judge read, when it read.
func readsLine(a replayAnswer) string {
	if len(a.Reads) == 0 {
		return ""
	}
	return fmt.Sprintf("; read %s, %d rounds, %d lines", strings.Join(slices.Compact(a.Reads), ", "), a.Served, a.Lines)
}

// replayFailed says what an item's answer lost its point on, one line.
func replayFailed(it replayItem, a replayAnswer) string {
	if a.Said == "" {
		return fmt.Sprintf("no answer: %.300s", oneLine(fmt.Sprint(a.Err)))
	}
	kind := map[bool]string{true: "true", false: "false"}[it.Truth]
	return fmt.Sprintf("said %s (%s then) to a %s finding, %s: %.300s", a.Said, it.Said, kind, oneLine(a.Shown), oneLine(a.Why))
}

var spaces = regexp.MustCompile(`\s+`)

func oneLine(s string) string { return spaces.ReplaceAllString(strings.TrimSpace(s), " ") }
