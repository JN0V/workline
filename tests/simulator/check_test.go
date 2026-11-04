package simulator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/routing"
)

// failure is the first property a run broke.
type failure struct {
	Property string
	Tick     int
	Detail   string
}

func (f *failure) String() string {
	if f == nil {
		return "none"
	}
	return fmt.Sprintf("%s at tick %d: %s", f.Property, f.Tick, f.Detail)
}

// The statuses I1 accepts, read from today's findings by their rule:
// a step started or queued (Due), a wait said (Waiting). Anything else on
// an open issue of the line, "acted, nothing to run" first, is no status.
// TestOracleRulesExist fails when the engine no longer has one of them.
var (
	dueRules     = []string{"state-wakes", "reconcile-left"}
	waitingRules = []string{"state-blocked-quiet", "state-blocked-seen", "state-waits", "state-room", "repro-waits", "state-review-missing"}
	notStatus    = []string{"state-acted"}
)

// target is what a wait must name to count (state-waits): an issue, room
// in a state, a time.
var target = regexp.MustCompile(`#\d+|\broom\b|\d{4}-\d\d-\d\d`)

// status is an open issue's status after a tick, "" for none, and what
// was said: the wait's message for a Waiting, the rules seen for none.
func status(res *line.Result, id int) (string, string) {
	where := fmt.Sprintf("#%d", id)
	if res.Woke[id] != "" {
		return "due", res.Woke[id]
	}
	var said []string
	for _, f := range res.Findings {
		if f.Where != where {
			continue
		}
		switch {
		case slices.Contains(dueRules, f.Rule):
			return "due", f.Message
		case slices.Contains(waitingRules, f.Rule) && (f.Rule != "state-waits" || target.MatchString(f.Message)):
			return "waiting", f.Message
		}
		said = append(said, f.Rule+": "+f.Message)
	}
	if len(said) == 0 {
		return "", "nothing said"
	}
	return "", strings.Join(said, "; ")
}

var recordRe = regexp.MustCompile(`workline:line-record (\{[^\n]*\}) -->`)

// record is an issue's line record: its state and hops; had, whether it
// has one.
func record(is forge.FakeItem) (state string, hops int, had bool) {
	for _, c := range is.Comments {
		if m := recordRe.FindStringSubmatch(c.Body); m != nil {
			var r struct {
				State string `json:"state"`
				Hops  int    `json:"hops"`
			}
			if json.Unmarshal([]byte(m[1]), &r) == nil {
				state, hops, had = r.State, r.Hops, true
			}
		}
	}
	return state, hops, had
}

func trail(is forge.FakeItem) string {
	st, hops, _ := record(is)
	return fmt.Sprintf("labels %v, state %s, hops %d", slices.Sorted(slices.Values(is.Labels)), st, hops)
}

func issueOf(s forge.FakeState, id int) *forge.FakeItem {
	for i := range s.Issues {
		if s.Issues[i].ID == id {
			return &s.Issues[i]
		}
	}
	return nil
}

// The properties, by name; seeded are those that hold whatever the
// agent answers, checked on every trace under seeds, pending or not.
const (
	pOneLabel   = "one-state-label"
	pI3         = "i3-machine-error-moves-nothing"
	pBudget     = "no-person-wait-with-budget"
	pI1         = "i1-status"
	pI2         = "i2-same-input"
	pTwice      = "same-tick-twice"
	pAttributed = "call-attributed"
	pQuiet      = "quiet-within-k"
	pEnd        = "expected-end"
	pOneByte    = "one-byte-due"
)

var seeded = []string{pAttributed, pOneLabel, pI3, pBudget, pI2}

// checks is what a run checks: every property (all) but those set aside
// (skip), or some (only), up to a tick.
type checks struct {
	all   bool
	skip  []string
	only  []string
	until int // the last tick checked, when not all
}

func (c checks) on(p string) bool {
	return c.all && !slices.Contains(c.skip, p) || slices.Contains(c.only, p)
}

// tickChecks are the properties checked after each tick, in order.
func (w *world) tickChecks(c checks, k int, before, after forge.FakeState, res, again *line.Result, calls []call) *failure {
	fail := func(p, format string, args ...any) *failure {
		return &failure{Property: p, Tick: k, Detail: fmt.Sprintf(format, args...)}
	}
	states := map[string]bool{}
	for _, s := range w.cfg.States {
		states[s.Label] = true
	}
	onLine := func(is forge.FakeItem) bool {
		return slices.ContainsFunc(is.Labels, func(l string) bool { return states[l] })
	}
	// Every call read back to its run's issue, else the checks below
	// would hold the wrong issue to account.
	for _, cl := range w.agent.calls {
		if cl.Tick == k && cl.Issue < 0 && c.on(pAttributed) {
			return fail(pAttributed, "the %s's call, its run's issue not read: %s", cl.Role, cl.runDir)
		}
	}
	// One state label at a time (ADR-0059 §2).
	for _, is := range after.Issues {
		var on []string
		for _, l := range is.Labels {
			if states[l] {
				on = append(on, l)
			}
		}
		if !is.Closed && len(on) > 1 && c.on(pOneLabel) {
			return fail(pOneLabel, "#%d carries %v", is.ID, on)
		}
	}
	// I3: a machine error moves nothing: no label, no state, no hop; a
	// first record, the state's entry, is not a hop: it counts one, never
	// more (due writes 1; a lost run adds none). A record catching up, at
	// the wake, with the state its labels say is no move either. Only the
	// last call on an issue in the tick counts: an error the engine
	// retried at once into progress moved the issue by that progress.
	for _, cl := range lastCalls(calls) {
		if !cl.Kind.machine() || !c.on(pI3) {
			continue
		}
		b, a := issueOf(before, cl.Issue), issueOf(after, cl.Issue)
		if b == nil || a == nil {
			continue
		}
		bs, bh, had := record(*b)
		as, ah, has := record(*a)
		if !slices.Equal(slices.Sorted(slices.Values(b.Labels)), slices.Sorted(slices.Values(a.Labels))) ||
			had && (bs != as && as != stateName(w.cfg, a.Labels) || bh != ah) || !had && has && ah != 1 {
			return fail(pI3, "#%d after the %s's %s: %s, was %s", cl.Issue, cl.Role, cl.Kind, trail(*a), trail(*b))
		}
	}
	// No Waiting(person) while a machine-error budget remains.
	for _, cl := range lastCalls(calls) {
		b, a := issueOf(before, cl.Issue), issueOf(after, cl.Issue)
		if !cl.Kind.machine() || !c.on(pBudget) || a == nil || b == nil ||
			!slices.Contains(a.Labels, "workline:blocked") || slices.Contains(b.Labels, "workline:blocked") {
			continue
		}
		errs := 0
		for _, d := range w.agent.calls {
			if d.Issue == cl.Issue && d.Role == cl.Role && d.Kind.machine() {
				errs++
			}
		}
		if errs <= budget {
			return fail(pBudget, "#%d blocked after the %s's %s %d of %d", cl.Issue, cl.Role, cl.Kind, errs, budget)
		}
	}
	// I1: every open issue on the line has a status.
	for _, is := range after.Issues {
		if is.Closed || !onLine(is) || !c.on(pI1) {
			continue
		}
		if st, said := status(res, is.ID); st == "" {
			return fail(pI1, "#%d has no status: %s", is.ID, said)
		}
	}
	// I2: never the same input twice, but after a machine error with
	// budget left: the last call on that input a machine error, the
	// errors on it within the budget. The simulator's digest is the oracle.
	for i, cl := range w.agent.calls {
		if cl.Tick != k || !c.on(pI2) {
			continue
		}
		var last *call
		errs := 0
		for j := range w.agent.calls[:i] {
			if d := w.agent.calls[j]; d.Role == cl.Role && d.Issue == cl.Issue && d.Digest == cl.Digest {
				last = &w.agent.calls[j]
				if d.Kind.machine() {
					errs++
				}
			}
		}
		if last != nil && (!last.Kind.machine() || errs > budget) {
			return fail(pI2, "the %s ran on #%d's input %s again, after a %s", cl.Role, cl.Issue, cl.Digest, last.Kind)
		}
	}
	// The same tick twice starts no second step: the second run starts
	// only the role of the state the first one moved the issue to, the
	// same role when it moved it to its own again (blocked, then refine).
	for id, role := range again.Woke {
		if !c.on(pTwice) {
			break
		}
		b, a := issueOf(before, id), issueOf(after, id)
		if b == nil || a == nil {
			return fail(pTwice, "the second run started the %s on #%d again", role, id)
		}
		if was, _ := w.cfg.StateOf(b.Labels); was.Name == stateName(w.cfg, a.Labels) {
			return fail(pTwice, "the second run started the %s on #%d again, in its state %s still", role, id, was.Name)
		}
		if now, wakes, _ := w.cfg.Wakes(a.Labels, line.IssueEvent); role != wakes {
			return fail(pTwice, "the second run started the %s on #%d, in its state %s, which wakes %q", role, id, now.Name, wakes)
		}
	}
	return nil
}

// stateName is the name of the state labels say, "" for none.
func stateName(cfg *routing.Config, labels []string) string {
	s, _ := cfg.StateOf(labels)
	return s.Name
}

// lastCalls are the last call on each issue among calls.
func lastCalls(calls []call) []call {
	var out []call
	for i, cl := range calls {
		if !slices.ContainsFunc(calls[i+1:], func(d call) bool { return d.Issue == cl.Issue }) {
			out = append(out, cl)
		}
	}
	return out
}

// runTrace runs a trace with an agent following script, checking c, and
// returns the first property broken and the run's log.
func runTrace(t *testing.T, tr trace, script []outcome, c checks) (*failure, string) {
	a := &simAgent{script: script}
	w := newWorld(t, tr, a)
	var log strings.Builder
	k := w.k()
	quiet := false
	for i := range k {
		if !c.all && i > c.until {
			return nil, log.String()
		}
		tick = i
		at := start.Add(time.Duration(i) * period)
		before := w.state(t)
		n := len(a.calls)
		res := w.run(at)
		mid := w.state(t)
		calls := slices.Clone(a.calls[n:])
		m := len(a.calls)
		again := w.run(at)
		fmt.Fprintf(&log, "tick %d: %s; woke %v; calls %v\n", i, res.Summary, res.Woke, calls)
		if len(again.Woke) > 0 {
			fmt.Fprintf(&log, "  again: woke %v; calls %v\n", again.Woke, a.calls[m:])
		}
		for _, mr := range w.state(t).MergeRequests {
			if !mr.Closed && (len(calls) > 0 || len(res.Woke) > 0) {
				fmt.Fprintf(&log, "  open !%d %s draft %v: %.80q\n", mr.ID, mr.Branch, mr.Draft, mr.Body)
			}
		}
		if len(calls) > 0 || len(res.Woke) > 0 {
			for _, f := range res.Findings {
				if f.Level != "info" {
					fmt.Fprintf(&log, "  %s %s %s: %.200s\n", f.Level, f.Rule, f.Where, f.Message)
				}
			}
		}
		if f := w.tickChecks(c, i, before, mid, res, again, calls); f != nil {
			return f, log.String()
		}
		// Between two ticks, the forge merges what is ready to.
		merged := w.merge(t, at.Add(period/2))
		if len(merged) > 0 {
			fmt.Fprintf(&log, "  forge: %v\n", merged)
		}
		quiet = len(merged) == 0 && len(calls) == 0 && len(res.Woke) == 0 && mid.Writes == before.Writes
	}
	if !c.all {
		return nil, log.String()
	}
	if !quiet && c.on(pQuiet) {
		return &failure{Property: pQuiet, Tick: k - 1, Detail: fmt.Sprintf("still acting after K = %d ticks", k)}, log.String()
	}
	// The trace's end: Done, or Waiting on what it names.
	at := start.Add(time.Duration(k) * period)
	tick = k
	res := w.run(at)
	for _, is := range w.state(t).Issues {
		st, said := status(res, is.ID)
		switch want, on, _ := strings.Cut(tr.end, ":"); {
		case is.Closed && want == "done":
		case !is.Closed && want == "waiting" && st == "waiting" && strings.Contains(said, on):
		case !c.on(pEnd):
		default:
			if is.Closed {
				st = "done"
			}
			return &failure{Property: pEnd, Tick: k, Detail: fmt.Sprintf("#%d %s (%s), not %s: %s", is.ID, st, trail(is), tr.end, said)}, log.String()
		}
	}
	// One byte of a body changed makes its open issue due.
	for _, is := range w.state(t).Issues {
		if is.Closed {
			continue
		}
		f := forge.Fake{Path: w.forge}
		if err := f.SetBody(is.ID, is.Body+"."); err != nil {
			t.Fatal(err)
		}
		if res := w.run(at); res.Woke[is.ID] == "" && c.on(pOneByte) {
			st, said := status(res, is.ID)
			return &failure{Property: pOneByte, Tick: k, Detail: fmt.Sprintf("#%d's body changed, not due: %s %s", is.ID, st, said)}, log.String()
		}
	}
	return nil, log.String()
}

// pending reads pending.txt: per trace, the properties it breaks today,
// in order: the first, then the next with the first set aside, and so on.
func pending(t *testing.T) map[string][]string {
	out := map[string][]string{}
	f, err := os.Open("pending.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		out[fields[0]] = fields[1:]
	}
	return out
}

// TestTraces runs each trace twice with an agent that always progresses:
// the same run each time, and the first property broken the one
// pending.txt names first, or none. Each next one pending.txt names is
// the first broken with those before it set aside: a trace still runs to
// its end past a defect it shows, checked there.
func TestTraces(t *testing.T) {
	want := pending(t)
	for name := range want {
		if !slices.ContainsFunc(traces, func(tr trace) bool { return tr.name == name }) {
			t.Errorf("pending.txt names %s, no such trace", name)
		}
	}
	for _, tr := range traces {
		t.Run(tr.name, func(t *testing.T) {
			props := want[tr.name]
			for j := 0; j <= len(props); j++ {
				c := checks{all: true, skip: props[:j]}
				f1, log1 := runTrace(t, tr, nil, c)
				if j == 0 {
					f2, log2 := runTrace(t, tr, nil, c)
					if log1 != log2 || f1.String() != f2.String() {
						t.Fatalf("deterministic: the same trace ran twice differs\n--- first\n%s%s\n--- second\n%s%s", log1, f1, log2, f2)
					}
				}
				if os.Getenv("SIM_LOG") != "" {
					t.Logf("set aside %v:\n%s", props[:j], log1)
				}
				got, next := "", ""
				if f1 != nil {
					got = f1.Property
				}
				if j < len(props) {
					next = props[j]
				}
				switch {
				case got == next && got != "":
					t.Logf("broken, %v set aside: %s", props[:j], f1)
					continue
				case got == next:
				case next == "":
					t.Errorf("%v set aside: %s\n%s", props[:j], f1, log1)
				case got == "":
					t.Errorf("passes now with %v set aside: take %s off pending.txt for %s", props[:j], next, tr.name)
				default:
					t.Errorf("pending.txt says %s breaks %s, %v set aside; it breaks %s\n%s", tr.name, next, props[:j], f1, log1)
				}
				return
			}
		})
	}
}

// TestOracleRulesExist: the rules the I1 oracle reads are the engine's
// still. A rule renamed or gone would turn a status into none, or hide
// one, without a word.
func TestOracleRulesExist(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "internal", "line", "*.go"))
	var src strings.Builder
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			src.Write(data)
		}
	}
	for _, rule := range slices.Concat(dueRules, waitingRules, notStatus) {
		if !regexp.MustCompile(`Rule:\s*"` + regexp.QuoteMeta(rule) + `"`).MatchString(src.String()) {
			t.Errorf("the I1 oracle reads the rule %s, which internal/line no longer has: read its statuses again", rule)
		}
	}
}

// seeds is how many seeded runs there are, spread over the traces: 100
// on a pull request, under two minutes with g++'s builds; the night sets
// SIM_SEEDS higher and SIM_SEED_BASE new.
func seeds() (base, n int64) {
	n, base = 100, 1
	if v, err := strconv.ParseInt(os.Getenv("SIM_SEEDS"), 10, 64); err == nil {
		n = v
	}
	if v, err := strconv.ParseInt(os.Getenv("SIM_SEED_BASE"), 10, 64); err == nil {
		base = v
	}
	return base, n
}

// scriptOf draws a run's outcomes from seed.
func scriptOf(seed int64) []outcome {
	rng := rand.New(rand.NewSource(seed))
	s := make([]outcome, 48)
	for i := range s {
		switch r := rng.Intn(10); {
		case r < 5:
			s[i] = progress
		case r < 7:
			s[i] = waiting
		case r < 9:
			s[i] = machineError
		default:
			s[i] = malformed
		}
	}
	return s
}

// shrink turns outcomes into progress one at a time while the run still
// breaks the same property: the smallest script that shows it.
func shrink(t *testing.T, tr trace, script []outcome, prop string, c checks) []outcome {
	s := slices.Clone(script)
	for i := range s {
		if s[i] == progress {
			continue
		}
		try := slices.Clone(s)
		try[i] = progress
		if f, _ := runTrace(t, tr, try, c); f != nil && f.Property == prop {
			s = try
		}
	}
	for len(s) > 0 && s[len(s)-1] == progress {
		s = s[:len(s)-1]
	}
	return s
}

func parseScript(v string) []outcome {
	var s []outcome
	for _, o := range strings.Split(v, ",") {
		s = append(s, outcome(strings.TrimSpace(o)))
	}
	return s
}

// TestSeeds runs every trace under seeded outcomes: progress, waiting,
// machine errors, malformed answers. A trace pending.txt does not name
// is held to every property but its end, which only an agent that always
// progresses must reach (a week-long wait outlasts K); a pending one to
// the seeded ones that hold whatever the agent answers, but its own, up
// to the tick its first pending property breaks.
// A seed's failure is never pending: it prints the seed and its shrunk
// script (SIM_SEED, SIM_SCRIPT; SIM_TRACE takes one trace).
func TestSeeds(t *testing.T) {
	want := pending(t)
	base, n := seeds()
	var runs []trace
	limits := map[string]checks{}
	for _, tr := range traces {
		if v := os.Getenv("SIM_TRACE"); v != "" && v != tr.name {
			continue
		}
		runs = append(runs, tr)
		limits[tr.name] = checks{all: true, skip: []string{pEnd}}
		if len(want[tr.name]) > 0 {
			f, _ := runTrace(t, tr, nil, checks{all: true})
			if f == nil {
				t.Fatalf("%s is pending, and breaks nothing", tr.name)
			}
			limits[tr.name] = checks{only: slices.DeleteFunc(slices.Clone(seeded), func(p string) bool { return slices.Contains(want[tr.name], p) }), until: f.Tick}
		}
	}
	if len(runs) == 0 {
		t.Fatalf("no trace named %q", os.Getenv("SIM_TRACE"))
	}
	if v := os.Getenv("SIM_SCRIPT"); v != "" {
		for _, tr := range runs {
			if f, log := runTrace(t, tr, parseScript(v), limits[tr.name]); f != nil {
				t.Errorf("%s: %s\n%s", tr.name, f, log)
			}
		}
		return
	}
	if v, err := strconv.ParseInt(os.Getenv("SIM_SEED"), 10, 64); err == nil {
		base, n = v, 1
	}
	for i := range n {
		seed := base + i
		tr := runs[int(seed)%len(runs)]
		c := limits[tr.name]
		script := scriptOf(seed)
		if f, _ := runTrace(t, tr, script, c); f != nil {
			small := shrink(t, tr, script, f.Property, c)
			_, log := runTrace(t, tr, small, c)
			var words []string
			for _, o := range small {
				words = append(words, string(o))
			}
			t.Errorf("%s, seed %d: %s\nreplay: SIM_SEED=%d, or shrunk: SIM_TRACE=%s SIM_SCRIPT=%q\n%s", tr.name, seed, f, seed, tr.name, strings.Join(words, ","), log)
		}
	}
}
