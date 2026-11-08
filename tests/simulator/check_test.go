package simulator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"maps"
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
	"github.com/JN0V/workline/internal/verdict"
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
// An issue the line blocks for a person, its reason written on it — a
// claim that wrote no code, code said lacking twice, no test possible
// left unsettled, machine errors past the budget — waits on them.
// TestOracleRulesExist fails when the engine no longer has one of them.
var (
	dueRules     = []string{"state-wakes", "reconcile-left"}
	waitingRules = []string{"state-blocked-quiet", "state-blocked-seen", "state-waits", "state-room", "repro-waits", "state-review-missing",
		"no-code-spent", "needs-spent", "no-test-unsettled", "machine-error-spent", "backstop-crossed", "backstop-waits"}
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

// atBackstop says whether the line blocked is at its backstop, its
// record saying so (ADR-0064, §5).
func atBackstop(is forge.FakeItem) bool {
	for _, c := range is.Comments {
		if m := recordRe.FindStringSubmatch(c.Body); m != nil {
			var r struct {
				State    string `json:"state"`
				Backstop int    `json:"backstop"`
			}
			if json.Unmarshal([]byte(m[1]), &r) == nil {
				return r.State == "blocked" && r.Backstop > 0
			}
		}
	}
	return false
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
	pNeeds      = "needs-input-on-its-source"
	pRequests   = "requests-a-tick"
)

var seeded = []string{pAttributed, pOneLabel, pI3, pBudget, pI2, pNeeds, pRequests}

// maxRequests bounds the forge's requests in one reconcile of a trace's
// one issue, reads and writes, the role scripts' with the engine's; and
// perCallMore each agent call past the first: a role run again in its
// step, for the code its agent lacked (ADR-0063, §5), reads and writes
// again, less than a whole step.
// Measured on 2026-11-06, every trace and seeds 1 to 100, a comment
// read counted one request (it was its logged line's bytes, 11 or so,
// before): a tick with no agent call 3 to 9, as ADR-0063 §11 estimates
// (3 to 5 an issue); a tick of one call 11 to 31, the step's own reads
// and writes; of two calls, a step run again in place, 16 to 49 (38
// more than the fewest a step of one reads). Each bound is the next 50
// above its highest, as the first measure set them (121 to 150, 81 to
// 100), and is raised only with the measure that needs it.
const (
	maxRequests = 50
	perCallMore = 50
)

// checks is what a run checks: every property (all) but those set aside
// (skip), or some (only), up to a tick.
type checks struct {
	all   bool
	skip  []string
	only  []string
	until int // the last tick checked, when not all
	// known: a needs-input an open issue says breaks its property is not
	// held to it (knownNeeds), under seeds.
	known bool
}

// knownNeeds: the needs-input answers an open issue says end with no
// status: the product owner's from the engine. Its question, a person's,
// is held: it waits on its reporter, said (#1082). The tester's is
// held, and the developer's, its file outside the Scope asked of the
// product owner (#1084, ADR-0063 §9).
func knownNeeds(cl call) bool {
	return cl.Role == "product-owner" && cl.Kind == needsEngine
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
		// The claim's labels say no one state while #1081 stands: its
		// record is held to its hops only.
		claim := cl.Role == "developer" && claimed(b.Labels, a.Labels)
		// A record catching up counts the hop its labels made before,
		// by another role's hand-back: one, never more.
		catchUp := bs != as && as == stateName(w.cfg, a.Labels)
		// The claim's record stays, catches up with the labels before the
		// claim, or says developing: nothing else.
		claimRecord := claim && (as == stateName(w.cfg, b.Labels) || as == stateName(w.cfg, []string{"workline:developing"}))
		if !sameLabels(b.Labels, a.Labels) && !claim ||
			had && (bs != as && !catchUp && !claimRecord || bh != ah && (!catchUp || ah != bh+1)) || !had && has && ah != 1 {
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
	// A needs-input ends Due or Waiting on its source, as the line says
	// it right after (the second run): the engine's, Due; another role's,
	// Due on that role, or Waiting on it; a person's, Waiting.
	for _, cl := range lastCalls(calls) {
		a := issueOf(after, cl.Issue)
		if !cl.Kind.needs() || !c.on(pNeeds) || c.known && knownNeeds(cl) || a == nil || a.Closed {
			continue
		}
		st, said := status(again, cl.Issue)
		_, next, _ := w.cfg.Wakes(a.Labels, line.IssueEvent)
		ok := false
		switch cl.Kind {
		case needsEngine:
			// Due, given what it lacked; said lacking again once given it,
			// in the same step, a person (ADR-0063, §5): Waiting.
			ok = st == "due" || st == "waiting" && givenBefore(calls, cl)
		case needsRole:
			ok = st == "due" && (again.Woke[cl.Issue] == "product-owner" || next == "product-owner") ||
				st == "waiting" && strings.Contains(said, "product owner")
		case needsPerson:
			ok = st == "waiting"
		}
		if !ok {
			if st == "" {
				st = "no status"
			}
			return fail(pNeeds, "#%d after the %s's %s: %s (%s), %s", cl.Issue, cl.Role, cl.Kind, st, trail(*a), said)
		}
	}
	// The same tick twice starts no second step: the second run starts
	// only the role of the state the first one moved the issue to, the
	// same role when it moved it to its own again (blocked, then refine).
	// A wake with no role is a write with no agent call (a block the
	// engine writes itself), not a step started.
	for id, role := range again.Woke {
		if !c.on(pTwice) {
			break
		}
		if role == "" {
			continue
		}
		b, a := issueOf(before, id), issueOf(after, id)
		if b == nil || a == nil {
			return fail(pTwice, "the second run started the %s on #%d again", role, id)
		}
		if was, _ := w.cfg.StateOf(b.Labels); was.Name == stateName(w.cfg, a.Labels) {
			var inputs []string
			for _, d := range w.agent.calls {
				if d.Tick == k && d.Issue == id && d.Role == role {
					inputs = append(inputs, d.Digest)
				}
			}
			return fail(pTwice, "the second run started the %s on #%d again, in its state %s still; its inputs this tick %v", role, id, was.Name, inputs)
		}
		if now, wakes, _ := w.cfg.Wakes(a.Labels, line.IssueEvent); role != wakes {
			return fail(pTwice, "the second run started the %s on #%d, in its state %s, which wakes %q", role, id, now.Name, wakes)
		}
	}
	return nil
}

func sameLabels(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}

// claimed: after is before with the developer's claim, made at its wake
// before its agent runs (ADR-0059 §2): workline:developing on, in place
// of workline:ready or, while #1081 stands, beside it. The claim is the
// step's start, not its agent's outcome: I3 holds from it on.
func claimed(before, after []string) bool {
	if !slices.Contains(before, "workline:ready") || slices.Contains(before, "workline:developing") {
		return false
	}
	beside := append(slices.Clone(before), "workline:developing")
	instead := append(slices.DeleteFunc(slices.Clone(before), func(l string) bool { return l == "workline:ready" }), "workline:developing")
	return sameLabels(after, beside) || sameLabels(after, instead)
}

// stateName is the name of the state labels say, "" for none.
func stateName(cfg *routing.Config, labels []string) string {
	s, _ := cfg.StateOf(labels)
	return s.Name
}

// givenBefore says whether the call just before cl among calls, on the
// same issue and role, was a needs-input of the engine: cl is its run
// again, given what it lacked.
func givenBefore(calls []call, cl call) bool {
	j := slices.Index(calls, cl)
	for i := j - 1; i >= 0; i-- {
		if d := calls[i]; d.Issue == cl.Issue && d.Role == cl.Role {
			return d.Kind == needsEngine
		}
	}
	return false
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
	f, log, _ := measured(t, tr, script, c)
	return f, log
}

// measure is what a run measured, tick by tick: the forge's requests and
// the agent's calls of each tick's reconcile; the states its issue went
// through, run by run, each once in a row.
type measure struct {
	requests, calls []int
	path            []string
	unused          map[string]int // the trial's answers left unused, by role
	// end: the forge at the trace's end, before a body is changed.
	end forge.FakeState
}

func (m measure) String() string {
	top := func(xs []int) (int, float64) {
		hi, sum := 0, 0
		for _, x := range xs {
			hi, sum = max(hi, x), sum+x
		}
		if len(xs) == 0 {
			return 0, 0
		}
		return hi, float64(sum) / float64(len(xs))
	}
	rh, rm := top(m.requests)
	ch, cm := top(m.calls)
	return fmt.Sprintf("forge requests a tick: highest %d, mean %.1f; agent calls a tick: highest %d, mean %.1f, over %d ticks", rh, rm, ch, cm, len(m.requests))
}

// step adds the state the trace's issues are in to the path, when it is
// another than the last.
func (m *measure) step(cfg *routing.Config, s forge.FakeState) {
	for _, is := range s.Issues {
		st := stateName(cfg, is.Labels)
		if is.Closed {
			st = "done"
		}
		if st != "" && (len(m.path) == 0 || m.path[len(m.path)-1] != st) {
			m.path = append(m.path, st)
		}
	}
}

// weeklyRevisit says whether a tick's run read a blocked issue again
// for its week gone by, no news (ADR-0059): a wake the clock makes.
func weeklyRevisit(res *line.Result) bool {
	return slices.ContainsFunc(res.Findings, func(f verdict.Finding) bool {
		return f.Rule == "state-revisit" && strings.Contains(f.Message, "(routing.revisit)")
	})
}

// quietWatch follows whether a run's line is quiet, and the weekly
// revisit that woke it last, if it has not been quiet since.
type quietWatch struct {
	quiet bool
	// weekly: the tick a weekly revisit woke the quiet line, -1 for
	// none, or for one the line was quiet again after.
	weekly int
}

func newQuietWatch() quietWatch { return quietWatch{weekly: -1} }

// ran takes tick i's run: a weekly revisit counts only on a quiet line.
func (q *quietWatch) ran(i int, revisit bool) {
	if q.quiet && revisit {
		q.weekly = i
	}
}

// ended takes whether tick i ended quiet: quiet again, a revisit before
// it no longer excuses what follows.
func (q *quietWatch) ended(quiet bool) {
	q.quiet = quiet
	if quiet {
		q.weekly = -1
	}
}

// excused says whether a line still acting at the run's end is so since
// a weekly revisit woke it, never quiet after: the revisit's own K ticks
// end past the run's last.
func (q quietWatch) excused() bool { return !q.quiet && q.weekly >= 0 }

// measured is runTrace, with what it measured.
func measured(t *testing.T, tr trace, script []outcome, c checks) (_ *failure, _ string, ms measure) {
	a := &simAgent{script: script}
	w := newWorld(t, tr, a)
	var log strings.Builder
	defer func() { ms.unused = a.unused() }()
	ms.step(w.cfg, w.state(t))
	k := w.k()
	qw := newQuietWatch()
	for i := range k {
		if !c.all && i > c.until {
			return nil, log.String(), ms
		}
		tick = i
		at := start.Add(time.Duration(i) * period)
		before := w.state(t)
		n := len(a.calls)
		r := w.requests(t)
		res := w.run(at)
		qw.ran(i, weeklyRevisit(res))
		ms.requests = append(ms.requests, w.requests(t)-r)
		mid := w.state(t)
		ms.step(w.cfg, mid)
		calls := slices.Clone(a.calls[n:])
		ms.calls = append(ms.calls, len(calls))
		m := len(a.calls)
		again := w.run(at)
		ms.step(w.cfg, w.state(t))
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
				if f.Level != "info" || os.Getenv("SIM_INFO") != "" {
					fmt.Fprintf(&log, "  %s %s %s: %.200s\n", f.Level, f.Rule, f.Where, f.Message)
				}
			}
		}
		if f := w.tickChecks(c, i, before, mid, res, again, calls); f != nil {
			return f, log.String(), ms
		}
		if got, bound := ms.requests[i], maxRequests+perCallMore*max(0, ms.calls[i]-1); got > bound && c.on(pRequests) {
			return &failure{Property: pRequests, Tick: i, Detail: fmt.Sprintf("%d requests to the forge in one reconcile of %d agent calls, more than %d", got, ms.calls[i], bound)}, log.String(), ms
		}
		// Between two ticks, the forge merges what is ready to.
		merged := w.merge(t, at.Add(period/2))
		if len(merged) > 0 {
			fmt.Fprintf(&log, "  forge: %v\n", merged)
			ms.step(w.cfg, w.state(t))
		}
		qw.ended(len(merged) == 0 && len(calls) == 0 && len(res.Woke) == 0 && mid.Writes == before.Writes)
	}
	if !c.all {
		return nil, log.String(), ms
	}
	// A weekly revisit wakes a quiet line by design (ADR-0059), a new
	// input: its own K ticks start there and end past the run's last,
	// so still acting since, never quiet in between, is pending, not
	// loud. A line never quiet before it, quiet again after it, or
	// woken by anything else, is held to K.
	quiet := qw.quiet
	if qw.excused() && c.on(pQuiet) {
		fmt.Fprintf(&log, "  quiet: a weekly revisit at tick %d, its K ticks not over\n", qw.weekly)
		quiet = true
	}
	if !quiet && c.on(pQuiet) {
		return &failure{Property: pQuiet, Tick: k - 1, Detail: fmt.Sprintf("still acting after K = %d ticks", k)}, log.String(), ms
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
			return &failure{Property: pEnd, Tick: k, Detail: fmt.Sprintf("#%d %s (%s), not %s: %s", is.ID, st, trail(is), tr.end, said)}, log.String(), ms
		}
	}
	ms.end = w.state(t)
	// One byte of a body changed makes its open issue due; but one
	// blocked at its backstop, which waits for a person, no product
	// owner reading it again (ADR-0064, §5).
	for _, is := range w.state(t).Issues {
		if is.Closed || atBackstop(is) {
			continue
		}
		f := forge.Fake{Path: w.forge}
		if err := f.SetBody(is.ID, is.Body+"."); err != nil {
			t.Fatal(err)
		}
		if res := w.run(at); res.Woke[is.ID] == "" && c.on(pOneByte) {
			st, said := status(res, is.ID)
			return &failure{Property: pOneByte, Tick: k, Detail: fmt.Sprintf("#%d's body changed, not due: %s %s", is.ID, st, said)}, log.String(), ms
		}
	}
	return nil, log.String(), ms
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
				f1, log1, m1 := measured(t, tr, nil, c)
				if j == 0 {
					f2, log2, m2 := measured(t, tr, nil, c)
					if log1 != log2 || f1.String() != f2.String() || m1.String() != m2.String() {
						t.Fatalf("deterministic: the same trace ran twice differs\n--- first\n%s%s\n--- second\n%s%s", log1, f1, log2, f2)
					}
				}
				if j == len(props) {
					t.Log(m1)
					if tr.replay {
						checkReplay(t, tr, m1)
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

// trialPath is the states the trace's issue went through in the trial,
// each once in a row, from its label events: from the person's
// workline:to-refine on, as a fresh trace starts.
func trialPath(t *testing.T, tr trace) []string {
	data, err := os.ReadFile(filepath.Join("traces", tr.file+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var s forge.FakeState
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	cfg, err := routing.Load(repoTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var m measure
	for _, is := range s.Issues {
		var labels []string
		on := false
		for _, e := range is.LabelEvents {
			on = on || e.Label == "workline:to-refine" && e.Added
			if e.Added {
				labels = append(labels, e.Label)
			} else {
				labels = slices.DeleteFunc(labels, func(l string) bool { return l == e.Label })
			}
			if on {
				m.step(cfg, forge.FakeState{Issues: []forge.FakeItem{{Labels: labels}}})
			}
		}
		if is.Closed && is.Reason != "not_planned" {
			m.step(cfg, forge.FakeState{Issues: []forge.FakeItem{{Closed: true}}})
		}
	}
	return m.path
}

// parting is where the simulator's path parts from the trial's, "none"
// when it follows it to its end.
func parting(trial, sim []string) string {
	for i, st := range trial {
		switch {
		case i >= len(sim):
			return fmt.Sprintf("after %s, the trial went on to %s, the simulator stopped", sim[i-1], st)
		case sim[i] != st && i == 0:
			return fmt.Sprintf("at the start: the trial in %s, the simulator in %s", st, sim[i])
		case sim[i] != st:
			return fmt.Sprintf("after %s: the trial went to %s, the simulator to %s", sim[i-1], st, sim[i])
		}
	}
	return "none"
}

// checkReplay checks a replayed trace takes the trial's path, or parts from
// it where traces/answers/<file>/parts says on its first line, what
// parting says, and the trial's answers no call took, why on the lines
// below. A line that no longer holds fails, the one to write instead
// given.
func checkReplay(t *testing.T, tr trace, m measure) {
	trial := trialPath(t, tr)
	got := parting(trial, m.path)
	if len(m.unused) > 0 {
		var left []string
		for _, role := range slices.Sorted(maps.Keys(m.unused)) {
			left = append(left, fmt.Sprintf("%d %s", m.unused[role], role))
		}
		got += "; unused: " + strings.Join(left, ", ")
	} else {
		got += "; every answer used"
	}
	t.Logf("the trial's path %v; the simulator's %v; parts: %s", trial, m.path, got)
	file := filepath.Join("traces", "answers", tr.file, "parts")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Errorf("%s: %v; it parts: %s", file, err, got)
		return
	}
	if want, _, _ := strings.Cut(string(data), "\n"); want != got {
		t.Errorf("%s says it parts %q; it parts %q: write that on its first line, why below it", file, want, got)
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
		// A rule is named in a Finding, or given its blocked helper (outcome.go).
		if !regexp.MustCompile(`(Rule:\s*|blocked\()"` + regexp.QuoteMeta(rule) + `"`).MatchString(src.String()) {
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
		switch r := rng.Intn(16); {
		case r < 6:
			s[i] = progress
		case r < 8:
			s[i] = waiting
		case r < 10:
			s[i] = machineError
		default:
			s[i] = []outcome{malformed, hunk, broken, needsEngine, needsRole, needsPerson}[r-10]
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
// The seeds draw over the plain traces: a case's point is its own answer,
// a replay's the trial's, both run by TestTraces. A needs-input on its
// source is held under seeds but for the role and kind an open issue says
// break it (knownNeeds); the named needs traces hold those too.
func TestSeeds(t *testing.T) {
	want := pending(t)
	base, n := seeds()
	var runs []trace
	limits := map[string]checks{}
	for _, tr := range traces {
		v := os.Getenv("SIM_TRACE")
		if v != "" && v != tr.name || v == "" && (tr.by != nil || tr.replay || tr.tokens > 0) {
			continue
		}
		runs = append(runs, tr)
		limits[tr.name] = checks{all: true, skip: []string{pEnd}, known: true}
		if len(want[tr.name]) > 0 {
			f, _ := runTrace(t, tr, nil, checks{all: true})
			if f == nil {
				t.Fatalf("%s is pending, and breaks nothing", tr.name)
			}
			limits[tr.name] = checks{only: slices.DeleteFunc(slices.Clone(seeded), func(p string) bool { return slices.Contains(want[tr.name], p) }), until: f.Tick, known: true}
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

// TestQuietWatch holds the weekly revisit's excuse to a line still
// acting since the revisit, never quiet in between.
func TestQuietWatch(t *testing.T) {
	// A tick: L loud, Q quiet, R a weekly revisit, loud.
	for _, tc := range []struct {
		name, ticks string
		excused     bool
	}{
		{"revisit at the end", "LQQQQQQQQQQQQQQQQQQQQRL", true},
		{"quiet again, then another wake", "LQRLQQQQQQLLLLLLLLLLLLL", false},
		{"another issue acting every other tick", "LQLQLQLQLRQLQLQLQLQLQLL", false},
		{"revisit on a loud line", "LLLLLLLLLLLLLLLLLLLLLRL", false},
		{"quiet at the end", "LQQQQQQQQQQQQQQQQQQQQRQ", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := newQuietWatch()
			for i, c := range tc.ticks {
				q.ran(i, c == 'R')
				q.ended(c == 'Q')
			}
			if q.excused() != tc.excused {
				t.Errorf("%s: excused %v, want %v", tc.ticks, q.excused(), tc.excused)
			}
		})
	}
}
