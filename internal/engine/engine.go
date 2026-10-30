// Package engine runs one role once: check, prepare, propose, judge, apply
// (docs/spec/role-contract.md, "One run").
package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/release"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/rolefs"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Options describe one run.
type Options struct {
	Repo      string            // repository the role works on
	RolesDir  string            // folder holding the roles
	Role      string            // role name
	Event     string            // event the role runs on
	AI        string            // agent spec, see agent.Parse; empty = the project's setting
	DefaultAI string            // used when neither AI nor the project says (the user's own default)
	Inputs    map[string]string // name -> value, written to in/input/<name>
	Targets   map[string]string // name -> file an intention writes back to (e.g. the hook's message file)

	Forge  string        // forge spec, see forge.Open; empty = the project's `forge` setting
	Target *forge.Target // the issue or merge request comments and labels go on
	// Branch is the one the targeted merge request comes from, when the job
	// knows it without the forge (a CI job without the forge's token); else
	// the forge is asked. Its branch tells a release tool's (ADR-0017).
	Branch string
	Scope  []string // paths the task is about; a patch outside is refused
	// NoApply stops after judging: the proposals and what apply needs are kept
	// in the run folder, for `workline apply` in another job holding the token.
	NoApply bool
	// OpenMergeRequest puts what the patches wrote on a branch of the role's,
	// and opens a merge request for it, or updates the one open (ADR-0006).
	OpenMergeRequest bool
	// DryRun: on `release`, the line's dry run (`--dry-run`, ADR-0050):
	// every step run with no agent, nothing applied, what would hold the
	// release said, none of it blocking. The line's; a role never sees it.
	DryRun bool
	// ReleaseWait: on the release, how long the line waits for the checks
	// not done on the tip it is cut from (`--release-wait`, #640), then
	// reads them again; none, it reads them once.
	ReleaseWait time.Duration

	// TamperBeforeApply changes the prepared input between propose and apply.
	// It exists only for the conformance test that proves apply notices.
	TamperBeforeApply bool

	aiWaits string // why the agent was set aside: "ready" on a draft (DraftInput)
}

// DraftInput is the input saying a merge request is a draft, `true`, or
// not, `false`: the templates pass it on every run (#359). A draft gets no
// agent (ADR-0050): every role runs as with none, told the agent waits
// for it to be marked ready (WORKLINE_AI_WAITS=ready), and the run says so
// (Result.AIWaits). A draft never merges (ADR-0042): what it skips holds
// nothing back from the run on ready.
const DraftInput = "draft"

// AIWaitsReady is what Result.AIWaits and WORKLINE_AI_WAITS hold on a
// draft: the agent waits for the merge request to be marked ready.
const AIWaitsReady = "ready"

// ReadyInput says whether a merge request's run is the one at ready,
// `true`, or a push after it, anything else; the templates pass it on every
// run. A role whose agent judges once, at ready (`merge-request-ai:
// ready`, the documentalist, ADR-0051), runs as with none on a push after
// it, told so (WORKLINE_AI_WAITS=judged-at-ready). Unsaid, a run is at
// ready: a person's own run asks the agent.
const ReadyInput = "ready"

// AIJudgedAtReady is what WORKLINE_AI_WAITS holds on a push after ready,
// for a role whose agent judged the merge request once, at ready.
const AIJudgedAtReady = "judged-at-ready"

// Result is what a run reports.
type Result struct {
	Status     string            `json:"status"`
	Summary    string            `json:"summary,omitempty"`
	Findings   []verdict.Finding `json:"findings,omitempty"`
	AgentCalls int               `json:"agent-calls"`
	Calls      []agent.Call      `json:"calls,omitempty"` // each call: what was asked, what answered
	Applied    []string          `json:"applied"`
	Refused    []string          `json:"refused"`
	Handoffs   []any             `json:"handoffs,omitempty"` // next roles asked for; routing runs them
	Notes      []string          `json:"notes,omitempty"`    // what the agent left for a person: a round's notes each
	RunDir     string            `json:"run-dir"`
	ToApply    bool              `json:"to-apply,omitempty"` // judged with NoApply: `workline apply` still has work
	// Pending: the run folders `workline apply` is given, one a round, when
	// a run judged with NoApply went round, each round a merge request of
	// its own (ADR-0013's amendment).
	Pending      []string `json:"pending,omitempty"`
	MergeRequest int      `json:"merge-request,omitempty"` // the merge request the patches went to
	// Issues and Waiting are a role keeping a backlog's links, for the
	// job's summary (ADR-0038): the address of the forge's issues, an
	// issue's number after it making its page; the saved filter of what
	// waits on a person, workline:proposed.
	Issues  string `json:"issues,omitempty"`
	Waiting string `json:"waiting,omitempty"`
	// AIWaits is why the run's agent was set aside: "ready", a merge
	// request still a draft (DraftInput); "" when it had its agent, or
	// none was set.
	AIWaits   string `json:"ai-waits,omitempty"`
	maxTokens int    // the role's ai-max-tokens: what the run may spend; 0, no cap
	// maxSeconds is the role's ai-max-seconds: how long the run may wait on
	// its agents, from the first call; 0, no cap. waitSince is that call.
	maxSeconds int
	// maxTurns is the role's ai-max-turns, told to each agent; 0, none
	// (ADR-0056).
	maxTurns    int
	waitSince   time.Time
	timeCapped  bool     // an answer was not waited for, or a call not made, the run having waited maxSeconds
	proposed    []string // the merge requests' tasks earlier rounds proposed, with NoApply, not opened yet
	then        string   // the task this round asks alone, once the others are done (in/then)
	thenAsked   bool     // the run went round once for it: never twice
	capped      bool     // a call was refused, the run having spent maxTokens
	partRefused []string // what the parts answered and may not: refused with them
}

// errTokensSpent refuses a call once the run has spent its ai-max-tokens.
var errTokensSpent = errors.New("the run spent its ai-max-tokens")

// errTimeSpent ends a call, or refuses one, once the run has waited its
// ai-max-seconds on its agents.
var errTimeSpent = errors.New("the run waited its ai-max-seconds on its agents")

// timeLeft is how long the run may still wait on its agents; capped is
// false when its role sets no ai-max-seconds.
func (res *Result) timeLeft() (left time.Duration, capped bool) {
	if res.maxSeconds <= 0 {
		return 0, false
	}
	limit := time.Duration(res.maxSeconds) * time.Second
	if res.waitSince.IsZero() {
		return limit, true
	}
	return limit - time.Since(res.waitSince), true
}

// overTime says whether the run has waited on its agents what its role allows.
func (res *Result) overTime() bool {
	left, capped := res.timeLeft()
	return capped && left <= 0
}

// bounded is a role whose one answer may take no longer than what the run
// may still wait, its own model.timeout when that is shorter; ok is false
// when the run may wait no more. The first call starts the run's wait.
func (res *Result) bounded(r *role.Role) (_ *role.Role, ok bool) {
	left, capped := res.timeLeft()
	if !capped {
		return r, true
	}
	if left < time.Millisecond { // what is left would not write as a timeout
		res.timeCapped = true
		return r, false
	}
	if res.waitSince.IsZero() {
		res.waitSince = time.Now()
	}
	if own, err := r.Model.AnswerTimeout(); err == nil && own <= left {
		return r, true
	}
	cut := *r
	cut.Model.Timeout = left.Round(time.Millisecond).String()
	return &cut, true
}

// cutShort turns an agent's failure into errTimeSpent when the run's wait
// ran out meanwhile: the answer was not waited for, past ai-max-seconds.
func (res *Result) cutShort(err error) error {
	if err == nil || !errors.Is(err, agent.ErrUnavailable) || !res.overTime() {
		return err
	}
	res.timeCapped = true
	return errTimeSpent
}

// tokensSpent adds up what the run's calls used, as the agents reported it:
// the whole input, cache included, and the output.
func (res *Result) tokensSpent() int {
	n := 0
	for _, c := range res.Calls {
		n += c.TokensIn + c.TokensOut
	}
	return n
}

// overBudget says whether the run has spent what its role allows.
func (res *Result) overBudget() bool {
	return res.maxTokens > 0 && res.tokensSpent() >= res.maxTokens
}

// Exit codes of pre and post (docs/spec/role-contract.md, "Exit codes").
const (
	exitOK       = 0
	exitBlock    = 1
	exitHuman    = 2
	exitExternal = 3
	exitNothing  = 10
)

// maxRounds caps how many times one run of a role goes round, when its pre
// leaves work for the next round (in/more: the findings it defers, one
// "<rule> <where>" a line).
const maxRounds = 5

// Run executes one run and always returns a result with a status: an error
// inside the run becomes a blocking finding, never a silent pass. When pre
// took less than there was to do and said so in in/more, and the round
// passed and applied something, the role goes round again: the next round
// sees what this one applied. Calls and applied intentions add up; a later
// round's finding replaces an earlier one of the same rule and place, and
// what a round deferred is dropped: the next round reports what is left.
//
// Judged with NoApply for merge requests of its own (gardening in CI), a
// round applies nothing the next could see; it goes round again when it
// proposed a merge request for a task no earlier round proposed: the next
// round is told that task waits, as one open on the forge, and takes up
// what it deferred — the docs judged in parts after those judged whole
// (ADR-0013). Each round is a run folder to apply, in Pending.
func Run(o Options) *Result {
	res := &Result{Applied: []string{}, Refused: []string{}}
	deferred := map[string]bool{}
	for round := 1; ; round++ {
		var earlier []verdict.Finding
		for _, f := range res.Findings {
			if !deferred[f.Rule+" "+f.Where] {
				earlier = append(earlier, f)
			}
		}
		applied := len(res.Applied)
		res.Findings, res.ToApply = nil, false
		if err := run(o, res); err != nil {
			res.Status = verdict.Block
			rule := "engine-error"
			if role.IsConfigError(err) {
				rule = "config-invalid"
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Message: err.Error()})
		}
		res.Findings = supersede(earlier, res.Findings)
		if n, err := os.ReadFile(filepath.Join(res.RunDir, "out", "note.md")); err == nil && strings.TrimSpace(string(n)) != "" {
			res.Notes = append(res.Notes, strings.TrimSpace(string(n)))
		}
		more, err := os.ReadFile(filepath.Join(res.RunDir, "in", "more"))
		deferred = map[string]bool{}
		for _, l := range strings.Split(string(more), "\n") {
			deferred[strings.TrimSpace(l)] = true
		}
		goOn := changedSomething(res.Applied[applied:])
		if o.NoApply {
			goOn = false
			if res.ToApply && !slices.Contains(res.Pending, res.RunDir) {
				res.Pending = append(res.Pending, res.RunDir)
				if task := proposedTask(res.RunDir); o.OpenMergeRequest && task != "" && !slices.Contains(res.proposed, task) {
					res.proposed = append(res.proposed, task)
					goOn = true
				}
			}
			res.ToApply = len(res.Pending) > 0 // what earlier rounds proposed is still to apply
		}
		last := err != nil || res.Status != verdict.Pass || !goOn || round == maxRounds
		// A task the role asks for once the others are done (in/then): one
		// round more, that task alone, within what the run may still spend
		// (the documentalist's pages read for their reader, ADR-0048).
		if then, _ := os.ReadFile(filepath.Join(res.RunDir, "in", "then")); last && res.Status == verdict.Pass && !res.thenAsked && strings.TrimSpace(string(then)) != "" {
			res.then, res.thenAsked, last = strings.TrimSpace(string(then)), true, false
		}
		tokens := res.capped || !last && res.overBudget()
		seconds := res.timeCapped || !last && res.overTime()
		next := "the next run"
		if tokens {
			// A cap is checked against what was spent, never estimated: the
			// call that crossed it is paid. What is left waits, said.
			res.Findings = append(res.Findings, verdict.Finding{Rule: "ai-max-tokens", Level: "warn",
				Message: fmt.Sprintf("%d tokens spent, the run allows %d: the agent was asked nothing more; what is left waits for %s", res.tokensSpent(), res.maxTokens, next)})
		}
		if seconds {
			// A wait has its budget (ADR-0050): the answer that crossed it
			// is not waited for, and nothing more is asked.
			res.Findings = append(res.Findings, verdict.Finding{Rule: "ai-max-seconds", Level: "warn",
				Message: fmt.Sprintf("the run waited %.0f s on its agents, it allows %d: the answer was not waited for, and nothing more was asked; what is left waits for a person or %s", time.Since(res.waitSince).Seconds(), res.maxSeconds, next)})
		}
		if tokens || seconds {
			return res
		}
		if last {
			if len(res.Pending) > 1 {
				res.Summary = fmt.Sprintf("judged, not applied: %d rounds, a merge request each; apply with: workline apply %s", len(res.Pending), strings.Join(res.Pending, " "))
			}
			return res
		}
	}
}

// proposedTask is the task a run judged with NoApply proposes a merge
// request for — the key the role named in out/merge-request.yaml, as its
// branch says it — or "" when it names none.
func proposedTask(runDir string) string {
	var mr struct{ Key string }
	data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml"))
	if err != nil || yaml.Unmarshal(data, &mr) != nil {
		return ""
	}
	return slugify(mr.Key)
}

// writeWaiting writes in/waiting.yaml: for each of the role's tasks
// waiting for review, the files its change touches — an open merge
// request's, from where its branch left this commit to its tip; one an
// earlier round of this run proposed, the files its patches name. A role holds back
// what they change (ADR-0013, amended). A branch that cannot be read
// leaves its task out, said: the role knows it by name only.
func writeWaiting(repo, runDir, prefix string, open []string, local bool, res *Result) error {
	waiting := map[string][]string{}
	for _, b := range open {
		files, err := branchFiles(repo, b, local)
		if err != nil {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-request-unread", Level: "warn", Where: b,
				Message: "what this merge request changes could not be read (" + err.Error() + "): the role knows it by its task only, and may judge again a doc it holds"})
			continue
		}
		waiting[strings.TrimPrefix(b, prefix)] = files
	}
	for _, dir := range res.Pending {
		task := proposedTask(dir)
		if task == "" {
			continue
		}
		intents, err := intent.Read(filepath.Join(dir, "out", "intentions.yaml"))
		if err != nil {
			return err
		}
		files := waiting[task]
		if files == nil {
			files = []string{}
		}
		for _, in := range intents {
			if in.Kind != "patch" {
				continue
			}
			for _, p := range intent.PatchFiles(in.Value) {
				if !slices.Contains(files, p) {
					files = append(files, p)
				}
			}
		}
		sort.Strings(files)
		waiting[task] = files
	}
	if len(waiting) == 0 {
		return nil
	}
	data, err := yaml.Marshal(waiting)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "in", "waiting.yaml"), data, 0o644)
}

// writeHeld lists, for a role opening merge requests, the files each other
// open merge request of this repository changes (in/held.yaml, by number):
// the role leaves them to it, so that neither conflicts with the other
// (ADR-0051). A fork's is not read; one whose change cannot be read is said.
func writeHeld(repo, runDir, prefix string, f forge.Forge, res *Result) error {
	mrs, err := f.ListMergeRequests()
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-requests-unread", Level: "warn",
			Message: "the open merge requests could not be listed (" + err.Error() + "): what they change is not left to them"})
		return nil
	}
	held := map[int][]string{}
	for _, mr := range mrs {
		if mr.ID == 0 || !mr.Here || mr.Branch == "" || strings.HasPrefix(mr.Branch, prefix) {
			continue
		}
		files, err := branchFiles(repo, mr.Branch, forge.KeepsBranches(f))
		if err != nil {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-request-unread", Level: "warn", Where: mr.Branch,
				Message: fmt.Sprintf("what merge request #%d changes could not be read (%v): the role may change what it changes", mr.ID, err)})
			continue
		}
		held[mr.ID] = files
	}
	if len(held) == 0 {
		return nil
	}
	data, err := yaml.Marshal(held)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "in", "held.yaml"), data, 0o644)
}

// branchFiles lists the files a branch changes from where it left the
// checked-out commit: its tip fetched from origin, or the clone's own
// branch for a forge kept there. None changed is an empty list, never nil.
func branchFiles(repo, branch string, local bool) ([]string, error) {
	tip := "refs/heads/" + branch
	if !local {
		if _, err := git(repo, nil, "fetch", "-q", "origin", "refs/heads/"+branch); err != nil {
			return nil, err
		}
		tip = "FETCH_HEAD"
	}
	tip, err := git(repo, nil, "rev-parse", "--verify", tip+"^{commit}")
	if err != nil {
		return nil, err
	}
	from, err := git(repo, nil, "merge-base", "HEAD", tip)
	if err != nil {
		return nil, err
	}
	out, err := git(repo, nil, "diff", "--name-only", "--no-renames", "-z", from, tip)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

// changedSomething says whether a round applied what the next one would
// see: a note changes nothing, and going round again would ask the same.
func changedSomething(applied []string) bool {
	for _, kind := range applied {
		if kind != "note" {
			return true
		}
	}
	return false
}

// supersede keeps the earlier findings no later one replaces, then the later ones.
func supersede(earlier, later []verdict.Finding) []verdict.Finding {
	seen := map[[2]string]bool{}
	for _, f := range later {
		seen[[2]string{f.Rule, f.Where}] = true
	}
	var out []verdict.Finding
	for _, f := range earlier {
		if !seen[[2]string{f.Rule, f.Where}] {
			out = append(out, f)
		}
	}
	return append(out, later...)
}

func run(o Options, res *Result) error {
	// 1. Check.
	r, err := role.Load(o.RolesDir, o.Role)
	if err != nil {
		return err
	}
	if !r.Accepts(o.Event) {
		return fmt.Errorf("role %q does not run on event %q", r.Name, o.Event)
	}
	for _, bin := range r.Requires {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("role %q requires %q, which is not on PATH", r.Name, bin)
		}
	}
	cfg, err := role.LoadProjectConfig(o.Repo)
	if err != nil {
		return err
	}
	if o.AI == "" {
		o.AI = cfg.AI
	}
	if o.AI == "" {
		o.AI = o.DefaultAI
	}
	if o.Inputs[DraftInput] == "true" && o.AI != "" && o.AI != "none" {
		o.AI, o.aiWaits, res.AIWaits = "none", AIWaitsReady, AIWaitsReady
	} else if v, said := o.Inputs[ReadyInput]; said && v != "true" && o.Event == "merge-request" && r.MergeRequestAI == "ready" && o.AI != "" && o.AI != "none" {
		o.AI, o.aiWaits = "none", AIJudgedAtReady
	}
	ag, err := agent.Parse(o.AI)
	if err != nil {
		return err
	}
	runDir, err := newRunDir(o.Repo, r.Name)
	if err != nil {
		return err
	}
	res.RunDir = runDir
	defer dropWorktree(o.Repo, runDir)
	settings := personalSettings(r, r.MergedSettings(cfg), res)
	if err := writeInputs(runDir, o.Inputs, settings); err != nil {
		return err
	}
	res.maxTokens = intSetting(settings, "ai-max-tokens")
	res.maxSeconds = intSetting(settings, "ai-max-seconds")
	res.maxTurns = intSetting(settings, "ai-max-turns")
	if o.Forge == "" {
		o.Forge = cfg.Forge
	}
	if slices.ContainsFunc(r.Intentions, func(k string) bool { return slices.Contains(backlog.Kinds, k) }) {
		if f, err := forge.Open(o.Forge, o.Repo); err == nil {
			links(f, res) // its summary names issues: each linked to its page
		}
	}
	env := scriptEnv(runDir, r.Name, o)
	if branch := releaseRequest(o); branch != "" {
		env = append(env, "WORKLINE_RELEASE_BRANCH="+branch)
	}
	if res.then != "" {
		env = append(env, "WORKLINE_THEN="+res.then)
	}
	if o.OpenMergeRequest {
		// The role decides what it proposes when enough of its merge requests wait.
		f, err := forge.Open(o.Forge, o.Repo)
		if err != nil {
			return err
		}
		if f == nil {
			return errors.New("--open-merge-request needs a forge: " + forge.Missing)
		}
		open, err := f.OpenMergeRequests(branchPrefix(r.Name))
		if err != nil {
			return err
		}
		// How many wait, and which tasks: the branches, the role's prefix cut;
		// those an earlier round of this run proposed wait as well.
		tasks := make([]string, len(open))
		for i, b := range open {
			tasks[i] = strings.TrimPrefix(b, branchPrefix(r.Name))
		}
		n := len(open)
		for _, t := range res.proposed {
			if !slices.Contains(tasks, t) {
				tasks, n = append(tasks, t), n+1
			}
		}
		if err := writeWaiting(o.Repo, runDir, branchPrefix(r.Name), open, forge.KeepsBranches(f), res); err != nil {
			return err
		}
		if err := writeHeld(o.Repo, runDir, branchPrefix(r.Name), f, res); err != nil {
			return err
		}
		env = append(env, fmt.Sprintf("WORKLINE_OPEN_MERGE_REQUESTS=%d", n),
			"WORKLINE_OPEN_MERGE_REQUEST_TASKS="+strings.Join(tasks, " "),
			"WORKLINE_PROPOSED_TASKS="+strings.Join(res.proposed, " "))
	}

	// 2. Prepare.
	code, err := script(r, "pre", o.Repo, env)
	if err != nil {
		return err
	}
	if stop, err := prepared(code, r, cfg, runDir, res); stop || err != nil {
		return err
	}
	// A question too large for one call is asked in parts; pre then puts
	// their answers together, and writes the one question that follows.
	if parts, _ := filepath.Glob(filepath.Join(runDir, "in", "parts", "*", "task.md")); len(parts) > 0 {
		if err := askParts(r, o, ag, runDir, parts, res); err != nil {
			return err
		}
		code, err := script(r, "pre", o.Repo, append(env[:len(env):len(env)], "WORKLINE_PARTS=answered"))
		if err != nil {
			return err
		}
		if stop, err := prepared(code, r, cfg, runDir, res); stop || err != nil {
			return err
		}
	}
	// Questions only a judge answers, one each (a reviewer's findings,
	// ADR-0020): each asked apart, at the best independence; pre then
	// settles what follows from the answers. A judge that asked to read
	// (ADR-0045) is asked again once pre served its reads: every judge
	// answers once before any is asked again.
	questions, _ := filepath.Glob(filepath.Join(runDir, "in", "judge", "*", "question.yaml"))
	for round := 1; len(questions) > 0; round++ {
		var err error
		switch {
		case round > maxJudgeRounds+1: // a role that asks again even so
			return fmt.Errorf("pre asked its judges again past %d rounds", maxJudgeRounds)
		case round > maxJudgeRounds: // not asked, said: pre settles them
			err = notAsked(questions, fmt.Sprintf("not asked: past %d rounds of the judge", maxJudgeRounds))
		default:
			err = askQuestions(o, ag, runDir, questions, res)
		}
		if err != nil {
			return err
		}
		answered := judgeAnswered(runDir)
		code, err := script(r, "pre", o.Repo, append(env[:len(env):len(env)], "WORKLINE_PARTS=answered", "WORKLINE_JUDGED=answered"))
		if err != nil {
			return err
		}
		if stop, err := prepared(code, r, cfg, runDir, res); stop || err != nil {
			return err
		}
		questions = askedAgain(runDir, answered)
	}
	digest, err := dirDigest(filepath.Join(runDir, "in"))
	if err != nil {
		return err
	}

	// 3. Propose and 4. Judge. A proposal the judge refuses is asked for again,
	// with the reasons, when the role allows it — one tier up after
	// `promote-after` refusals (docs/spec/model-grid.md).
	line, err := routing.Load(o.Repo)
	if err != nil {
		return err
	}
	_, taskErr := os.Stat(filepath.Join(runDir, "in", "task.md"))
	if _, ok := r.Model.Tasks[taskKind(runDir)]; ok {
		asked := *r // this task's needs, in place of the role's
		asked.Model = r.Model.For(taskKind(runDir))
		r = &asked
	}
	tier, failures := r.Model.Tier, 0
	var intents []intent.Intention
	var v *verdict.Verdict
	unreadAsked := false
	unasked := false // a task the agent was not asked, nor answered: what pre gave it read by no one
	for {
		a, err := attempt(r, o, ag, taskErr == nil, tier, runDir, env, line, settings, cfg, res)
		if err != nil {
			return err
		}
		unasked = taskErr == nil && (!a.askedAgent || a.external || a.unread != "")
		intents, v = a.intents, a.verdict
		if a.external && v.Status != verdict.Pass {
			res.Status, res.Summary = verdict.BlockedExternal, v.Summary
			res.Findings = append(res.Findings, v.Findings...)
			return nil
		}
		// An answer that could not be read is asked for again once, on the
		// same tier, with what the reader said: nothing of it was judged or
		// applied, and a quote left unescaped is a slip, not a refusal. The
		// documentalist's post passes with no proposal, and such a doc was
		// judged again, whole, the next night (DomoticsCore, ADR-0014 step 4).
		if a.askedAgent && a.unread != "" && v.Status != verdict.Block && !unreadAsked && r.Model.PromoteAfter > 0 {
			unreadAsked = true
			if err := os.WriteFile(filepath.Join(runDir, "out", "feedback.md"), []byte(unreadFeedback(a.unread)), 0o644); err != nil {
				return err
			}
			os.Rename(filepath.Join(runDir, "out", "agent-answer.txt"), filepath.Join(runDir, "out", "unread-answer.txt"))
			os.Remove(filepath.Join(runDir, "out", "verdict.yaml"))
			os.Remove(filepath.Join(runDir, "out", "judge.yaml"))
			continue
		}
		if !a.askedAgent || v.Status != verdict.Block || v.Final {
			break
		}
		failures++
		var retry bool
		if retry, tier = askAgain(failures, r.Model.PromoteAfter, tier); !retry || warnsOnMergeRequest(r, o) {
			break
		}
		var fb strings.Builder
		for _, f := range v.Findings {
			if f.Where != "" && !strings.Contains(f.Message, f.Where) {
				fmt.Fprintf(&fb, "- %s, %s: %s\n", f.Rule, f.Where, f.Message)
			} else {
				fmt.Fprintf(&fb, "- %s: %s\n", f.Rule, f.Message)
			}
		}
		if err := os.WriteFile(filepath.Join(runDir, "out", "feedback.md"), []byte(fb.String()), 0o644); err != nil {
			return err
		}
		// The refused answer is kept beside its refusal, to be studied, as
		// read and as it came.
		os.Rename(filepath.Join(runDir, "out", "intentions.yaml"), filepath.Join(runDir, "out", fmt.Sprintf("refused-%d.yaml", failures)))
		os.Rename(filepath.Join(runDir, "out", "agent-answer.txt"), filepath.Join(runDir, "out", fmt.Sprintf("refused-%d-answer.txt", failures)))
		os.Remove(filepath.Join(runDir, "out", "verdict.yaml"))
		os.Remove(filepath.Join(runDir, "out", "judge.yaml"))
	}
	// The agent is asked no more: the worktree its role gave it goes, its
	// diff already read by post (ADR-0056).
	dropWorktree(o.Repo, runDir)
	// Asked as often as the role allows and still refused: the docs the
	// refusals name are left out, and the rest judged again, without them.
	// A final block met after a refusal is no refusal: not the answer's,
	// nothing left out of the answer would lift it.
	if v.Status == verdict.Block && failures > 0 && !v.Final {
		fallback, _ := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
		if kept, left := leaveOut(intents, fallback, v.Findings); len(left) > 0 {
			nv, err := judgeAgain(r, o, runDir, env, cfg, kept, res)
			if err != nil {
				return err
			}
			if nv.Status == verdict.Pass {
				if kept, err = narrowedByPost(r, o, runDir, line, settings); err != nil {
					return err
				}
				intents, v = kept, nv
				v.Findings = append(left, v.Findings...)
			}
		}
	}
	// On an event whose steps all run (schedule, ADR-0037), a proposal
	// still refused once asked again is upkeep left for later: nothing of
	// it applied, each refusal said as a warning, the next run asks again.
	// The check did its job: the line is not failed for it (ADR-0037,
	// amended). Anything else that does not pass still fails it, a final
	// block met once asked again too: a person must repair it (#361).
	// A role whose agent only warns on a merge request (`merge-request-ai:
	// ready`, ADR-0051): asked once, its refused proposal is said the same
	// way, never holding the merge, and no stronger model is asked.
	if v.Status == verdict.Block && failures > 0 && !v.Final && (line != nil && !line.StopsAtFirst(o.Event) || warnsOnMergeRequest(r, o)) {
		said := make([]verdict.Finding, len(v.Findings))
		for i, f := range v.Findings {
			if f.Level == "" {
				f.Level = "warn"
			}
			said[i] = f
		}
		next := "asked again at the next run"
		if warnsOnMergeRequest(r, o) {
			next = "the weekly docs pull request judges it again"
		}
		v = &verdict.Verdict{Status: verdict.Pass, Findings: said,
			Summary: "the agent's proposal was refused, nothing of it applied; " + next + " (" + v.Summary + ")"}
		intents = nil
	}
	res.Status, res.Summary = v.Status, v.Summary
	res.Findings = append(res.Findings, v.Findings...)
	// A role keeping a backlog reads its report on every run, acts or not:
	// a closing undone, a box ticked, runs nobody answered (ADR-0025).
	keeps := slices.ContainsFunc(r.Intentions, func(k string) bool { return slices.Contains(backlog.Kinds, k) })
	// Blocked, a run applies only what its role names to tell why
	// (`on-block`): the reviewer's comment, its record in it (#226). The
	// run still blocks.
	onBlock := res.Status == verdict.Block && len(r.OnBlock) > 0
	if onBlock {
		intents = slices.DeleteFunc(intents, func(i intent.Intention) bool { return !slices.Contains(r.OnBlock, i.Kind) })
		keeps = false
	}
	if res.Status != verdict.Pass && !onBlock || (len(intents) == 0 && !keeps) {
		return nil
	}

	// 5. Apply — only what was prepared, unchanged.
	if o.TamperBeforeApply {
		if err := os.WriteFile(filepath.Join(runDir, "in", "tampered"), []byte("x"), 0o644); err != nil {
			return err
		}
	}
	now, err := dirDigest(filepath.Join(runDir, "in"))
	if err != nil {
		return err
	}
	if now != digest {
		res.Status = verdict.Block
		res.Findings = append(res.Findings, verdict.Finding{Rule: "input-changed", Message: "the prepared input changed before apply; nothing was applied"})
		return nil
	}
	// A claim says why a patch takes words out: judged, never applied. It is
	// kept apart in the run folder, so that the evidence an accepted fix gave
	// can be checked afterwards.
	var claims []intent.Intention
	for _, i := range intents {
		if i.Kind == "claim" {
			claims = append(claims, i)
		}
	}
	if len(claims) > 0 {
		if err := intent.Write(filepath.Join(runDir, "out", "claims.yaml"), claims); err != nil {
			return err
		}
	}
	// A skip says why an item of a file an import reads is not opened:
	// judged, never applied, kept apart for the import's map.
	var skips []intent.Intention
	for _, i := range intents {
		if i.Kind == "skip" {
			skips = append(skips, i)
		}
	}
	if len(skips) > 0 {
		if err := intent.Write(filepath.Join(runDir, "out", "skips.yaml"), skips); err != nil {
			return err
		}
	}
	intents = slices.DeleteFunc(intents, func(i intent.Intention) bool { return i.Kind == "claim" || i.Kind == "skip" })
	intents = splitEffort(intents)
	intent.SortForApply(intents)
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return err
	}
	if o.Forge == "" {
		o.Forge = cfg.Forge
	}
	st := runState{Role: r.Name, RolesDir: o.RolesDir, Repo: o.Repo, Forge: o.Forge, Target: o.Target, Scope: o.Scope, Digest: digest, Targets: o.Targets,
		OpenMergeRequest: o.OpenMergeRequest, Models: authors(res.Calls),
		Unasked: unasked}
	if err := st.save(runDir); err != nil {
		return err
	}
	if o.NoApply {
		switch {
		case onBlock:
			res.Summary = fmt.Sprintf("%s; what says why judged, not applied (%d proposals); apply with: workline apply %s", res.Summary, len(intents), runDir)
		default:
			res.Summary = fmt.Sprintf("judged, not applied (%d proposals); apply with: workline apply %s", len(intents), runDir)
		}
		res.ToApply = true
		for _, in := range intents {
			if in.Kind == "handoff" {
				res.Findings = append(res.Findings, deferred(r.Name, in.Value))
			}
		}
		// A role keeping a backlog decides its acts now, as the job that
		// applies them will do them: the summary says what will be done
		// alone and what proposed (ADR-0038). Reading only.
		if keeps && !onBlock {
			if err := planAhead(r, settings, st, runDir, intents, res); err != nil {
				return err
			}
			// What a person will see on the issues, not the writes it takes.
			res.Summary = fmt.Sprintf("judged, not applied (%s); apply with: workline apply %s", onIssues(res.Findings, true, st.Unasked), runDir)
		}
		return nil
	}
	return applyAll(r, settings, st, runDir, intents, res)
}

// renamedIssues are the issues workline keeps whose title changed, by the
// new title: the documentalist's tracking issue (ADR-0051).
var renamedIssues = map[string]string{"Docs due later": "Docs due at the next release"}

// prepared reads pre's exit code: stop is true when the run ends there, with
// no question for the agent or an outside service failed.
func prepared(code int, r *role.Role, cfg *role.ProjectConfig, runDir string, res *Result) (stop bool, err error) {
	switch code {
	case exitOK:
		return false, nil
	case exitNothing:
		// No question for the agent. A verdict pre wrote is final; none means pass.
		res.Status, res.Summary = verdict.Pass, "nothing to do"
		if v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml")); err == nil {
			verdict.Enforce(v, r.Enforcement(cfg))
			res.Status, res.Findings = v.Status, append(res.Findings, v.Findings...)
			if v.Summary != "" {
				res.Summary = v.Summary
			}
		}
		return true, nil
	case exitExternal:
		res.Status, res.Summary = verdict.BlockedExternal, "pre: an outside service failed"
		// What pre found before the service failed is still said.
		if v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml")); err == nil {
			res.Findings = append(res.Findings, v.Findings...)
			if v.Summary != "" {
				res.Summary = v.Summary
			}
		}
		return true, nil
	}
	return true, fmt.Errorf("pre exited with code %d", code)
}

// askParts asks each part of a question (in/parts/<name>/task.md), sorted by
// name, in a context of its own, on the tier model.tasks.part names, and puts
// each answer back as in/parts/<name>/answer.yaml. A part answers only what
// the role's part-intentions name — claims, unless it says otherwise; a
// reviewer's lens, findings — which inform and are never applied: a part that
// fails, or answers anything else, is said, and its answer never given back
// as one; why is written beside it, in/parts/<name>/unanswered, its first
// word the kind of failure (unavailable, spent, invalid, refused). Without
// an agent, no part is asked.
func askParts(r *role.Role, o Options, ag agent.Agent, runDir string, tasks []string, res *Result) error {
	if ag == nil {
		return nil
	}
	sort.Strings(tasks)
	asked := *r // a part's needs, and the intentions it may answer with
	asked.Model = r.Model.For("part")
	asked.Intentions = r.PartAnswers()
	claims := slices.Equal(asked.Intentions, []string{"claim"})
	gone := "" // the agent could not be reached: the other parts would fail the same
	goneKind := ""
	for _, task := range tasks {
		name := filepath.Base(filepath.Dir(task))
		unanswered := func(kind, why string) {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "part-unanswered", Where: name, Level: "warn",
				Message: "this part of the question got no answer that can be read (" + why + "): what it holds was judged by no one"})
			os.WriteFile(filepath.Join(filepath.Dir(task), "unanswered"), []byte(kind+": "+why+"\n"), 0o644)
		}
		if gone != "" {
			unanswered(goneKind, "not asked: "+gone)
			continue
		}
		dir := filepath.Join(runDir, "parts", name)
		for _, d := range []string{"in", "out"} {
			if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
				return err
			}
		}
		data, err := os.ReadFile(task)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "in", "task.md"), data, 0o644); err != nil {
			return err
		}
		req := agent.Request{RunDir: dir, Repo: o.Repo, Role: &asked, Tier: asked.Model.Tier}
		err = callAgent(ag, req, "part", runDir, res)
		// An answer that reads not at all is asked for again once, as the
		// main question's is (#138): a lens's text starting with a backtick
		// left a merge request not reviewed whole. The call counts as any
		// other; the second answer not reading either, the part fails.
		if why := partUnread(err, dir, claims); why != "" && asked.Model.PromoteAfter > 0 {
			if err := os.WriteFile(filepath.Join(dir, "out", "feedback.md"), []byte(unreadFeedback(why)), 0o644); err != nil {
				return err
			}
			os.Rename(filepath.Join(dir, "out", "agent-answer.txt"), filepath.Join(dir, "out", "unread-answer.txt"))
			os.Remove(filepath.Join(dir, "out", "intentions.yaml"))
			res.Findings = append(res.Findings, verdict.Finding{Rule: "part-asked-again", Where: name, Level: "warn",
				Message: "the answer to this part did not read (" + why + "): asked for again once, with what the reader said"})
			err = callAgent(ag, req, "part", runDir, res)
		}
		switch {
		case errors.Is(err, agent.ErrOverBudget):
			// Over the context budget, this part alone: the others may fit.
			unanswered("over-budget", err.Error())
			continue
		case errors.Is(err, agent.ErrUnavailable), errors.Is(err, errTokensSpent), errors.Is(err, errTimeSpent):
			gone, goneKind = err.Error(), "unavailable"
			if errors.Is(err, errTokensSpent) || errors.Is(err, errTimeSpent) {
				goneKind = "spent"
			}
			unanswered(goneKind, gone)
			continue
		case errors.Is(err, agent.ErrInvalidOutput) && claims:
			// One item written wrong spoils the whole list for a YAML
			// reader: the items are read one by one, the broken ones kept
			// as unreadable claims, which the role counts as dropped.
			raw, _ := os.ReadFile(filepath.Join(dir, "out", "agent-answer.txt"))
			read, broken, mended := claimsOneByOne(string(raw))
			if read == 0 {
				unanswered("invalid", err.Error())
				continue
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: "part-partly-read", Where: name, Level: "warn",
				Message: fmt.Sprintf("the answer to this part was not valid as a whole: %d claims read one by one, %d that could not be read counted as dropped", read, len(broken))})
			for _, m := range mended {
				res.Findings = append(res.Findings, verdict.Finding{Rule: "answer-mended", Where: name, Level: "warn", Message: m})
			}
			if err := intent.Write(filepath.Join(dir, "out", "intentions.yaml"), append(claimsRead(string(raw)), broken...)); err != nil {
				return err
			}
		case errors.Is(err, agent.ErrInvalidOutput):
			unanswered("invalid", err.Error())
			continue
		case err != nil:
			return err
		}
		answers, err := intent.Read(filepath.Join(dir, "out", "intentions.yaml"))
		if err != nil {
			unanswered("invalid", err.Error())
			continue
		}
		var other []string
		for _, c := range answers {
			if !slices.Contains(asked.Intentions, c.Kind) {
				other = append(other, c.Kind)
			}
		}
		if len(other) > 0 {
			if !claims { // a lens proposing to act: refused, as any intention a role may not emit
				res.partRefused = append(res.partRefused, other...)
				res.Findings = append(res.Findings, verdict.Finding{Rule: "intention-refused", Where: name,
					Message: strings.Join(other, ", ") + " is not allowed: a part answers with " + strings.Join(asked.Intentions, ", ") + " only"})
			}
			unanswered("refused", "it answered "+strings.Join(other, ", ")+"; a part answers with "+strings.Join(asked.Intentions, ", ")+" only")
			continue
		}
		answer := []byte("[]\n") // nothing to say of this share is an answer too
		if len(answers) > 0 {
			if answer, err = intent.Marshal(answers); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(task), "answer.yaml"), answer, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// partUnread says why a part's answer reads not at all, or "" when it reads,
// in places at least (claims are then read one by one), or when the agent
// gave no answer to read (unreachable, tokens spent).
func partUnread(err error, dir string, claims bool) string {
	if errors.Is(err, agent.ErrInvalidOutput) {
		if claims {
			raw, _ := os.ReadFile(filepath.Join(dir, "out", "agent-answer.txt"))
			if read, _, _ := claimsOneByOne(string(raw)); read > 0 {
				return ""
			}
		}
		return err.Error()
	}
	if err != nil {
		return ""
	}
	if _, rerr := intent.Read(filepath.Join(dir, "out", "intentions.yaml")); rerr != nil {
		return rerr.Error()
	}
	return ""
}

// unreadFeedback is what the agent is told when nothing of its answer read.
func unreadFeedback(why string) string {
	return "- agent-invalid-output: " + why + "\n\nNothing of that answer could be read: send it again as valid YAML. " +
		agent.BlockScalars + "\n"
}

// answerItems cuts an answer into its top-level list items: each starts
// with "- " at the start of a line and runs to the next, or to a fence.
// Every other line in between is the item's, one at the start of the line
// too (code pasted as it is), so an item never reads from text cut short;
// a line starting with "- " starts the next.
func answerItems(answer string) []string {
	var items []string
	open := false
	for _, l := range strings.Split(answer, "\n") {
		switch {
		case strings.HasPrefix(l, "- "):
			items, open = append(items, l+"\n"), true
		case strings.HasPrefix(l, "```"):
			open = false
		case open:
			items[len(items)-1] += l + "\n"
		}
	}
	return items
}

// claimsRead are the items of an answer that read as a claim each.
func claimsRead(answer string) []intent.Intention {
	var out []intent.Intention
	for _, item := range answerItems(answer) {
		item, _ = agent.Mend(item)
		var one []map[string]any
		if yaml.Unmarshal([]byte(item), &one) == nil && len(one) == 1 && len(one[0]) == 1 && one[0]["claim"] != nil {
			out = append(out, intent.Intention{Kind: "claim", Value: one[0]["claim"]})
		}
	}
	return out
}

// claimsOneByOne counts the items that read as claims, and returns the
// others as claims with nothing to check, which the role drops, and what
// was mended of those read (agent.Mend).
func claimsOneByOne(answer string) (int, []intent.Intention, []string) {
	read := len(claimsRead(answer))
	var broken []intent.Intention
	var mended []string
	for _, item := range answerItems(answer) {
		item, said := agent.Mend(item)
		var one []map[string]any
		if yaml.Unmarshal([]byte(item), &one) != nil || len(one) != 1 || len(one[0]) != 1 || one[0]["claim"] == nil {
			broken = append(broken, intent.Intention{Kind: "claim", Value: map[string]any{"unreadable": strings.TrimSpace(item)}})
			continue
		}
		mended = append(mended, said...)
	}
	return read, broken, mendedTogether(mended)
}

// mendedTogether sums what was mended item by item into one line per kind
// of mending, as for a whole answer: "2 block(s) …" and "1 block(s) …"
// make "3 block(s) …".
func mendedTogether(said []string) []string {
	var kinds []string
	count := map[string]int{}
	for _, s := range said {
		var n int
		kind := strings.TrimLeft(s, "0123456789")
		if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
			n, kind = -1, s // no count to sum: said as it is
		}
		if _, ok := count[kind]; !ok {
			kinds = append(kinds, kind)
		}
		count[kind] += n
	}
	var out []string
	for _, k := range kinds {
		if count[k] < 0 {
			out = append(out, k)
			continue
		}
		out = append(out, fmt.Sprint(count[k])+k)
	}
	return out
}

// callAgent asks the agent once and records the call: in the result, and
// with the run (out/calls.jsonl), what it cost, to be read afterwards.
func callAgent(ag agent.Agent, req agent.Request, task, runDir string, res *Result) error {
	if res.overBudget() {
		res.capped = true
		return errTokensSpent
	}
	bounded, ok := res.bounded(req.Role)
	if !ok {
		return errTimeSpent
	}
	req.Role, req.MaxTurns = bounded, res.maxTurns
	start := time.Now()
	call, err := ag.Propose(req)
	if errors.Is(err, agent.ErrOverBudget) {
		return err // refused before the call: nothing asked, nothing spent
	}
	err = res.cutShort(err)
	call.TimeCut = errors.Is(err, errTimeSpent)
	res.AgentCalls++
	call.Seconds = math.Round(time.Since(start).Seconds()*10) / 10
	call.Task = task
	if task == "part" {
		call.For = filepath.Base(req.RunDir)
	}
	res.Calls = append(res.Calls, call)
	if data, err := json.Marshal(call); err == nil {
		if f, err := os.OpenFile(filepath.Join(runDir, "out", "calls.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			f.Write(append(data, '\n'))
			f.Close()
		}
	}
	// What the engine mended of the answer before reading it is said:
	// the answer as it came stays in out/agent-answer.txt.
	for _, m := range call.Mended {
		where := ""
		if task == "part" {
			where = filepath.Base(req.RunDir)
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "answer-mended", Where: where, Level: "warn", Message: m})
	}
	if n := agent.Notice(agent.Seen(), call); n != "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "model-changed", Level: "warn", Message: n})
	}
	return err
}

// runState is what a later `workline apply` needs to resume a run.
type runState struct {
	Role     string            `yaml:"role"`
	RolesDir string            `yaml:"roles-dir"`
	Repo     string            `yaml:"repo"`
	Forge    string            `yaml:"forge,omitempty"`
	Target   *forge.Target     `yaml:"target,omitempty"`
	Scope    []string          `yaml:"scope,omitempty"`
	Digest   string            `yaml:"digest"`
	Targets  map[string]string `yaml:"targets,omitempty"`
	Applied  []int             `yaml:"applied"`
	// OpenMergeRequest: the patches go to a merge request; Written are the
	// files they wrote, and MergeRequest the one opened, once it is.
	OpenMergeRequest bool     `yaml:"open-merge-request,omitempty"`
	Written          []string `yaml:"written,omitempty"`
	MergeRequest     int      `yaml:"merge-request,omitempty"`
	// Pushed: what the patches wrote went in the targeted merge request's
	// comment.
	Pushed bool `yaml:"pushed,omitempty"`
	// Models: the agents and models that wrote the proposals, as
	// <agent>:<model>, named in the commit (ModelTrailer).
	Models []string `yaml:"models,omitempty"`
	// Unasked: the run had a task and no answer of its agent's was read —
	// over the role's context budget, its tokens spent: what pre gave it
	// was read by no one, and no issue is said read (#427).
	Unasked bool `yaml:"unasked,omitempty"`
}

// authors are the agents and models whose answers a run applies, as
// <agent>:<model>, in the order they first answered; a judge is not one.
func authors(calls []agent.Call) []string {
	var out []string
	for _, c := range calls {
		if c.Task == "judge" || c.Model == "" {
			continue
		}
		if a := c.Agent + ":" + c.Model; !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	return out
}

// trailers name, in an engine's commit, the role, and the agents and models
// that wrote what it commits: a later reader of the doc stands apart from
// them (ADR-0005; the weekly sample, ADR-0014 step 4).
func (s *runState) trailers() string {
	t := OwnTrailer + ": " + s.Role
	for _, m := range s.Models {
		t += "\n" + ModelTrailer + ": " + m
	}
	return t
}

func (s *runState) save(runDir string) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "out", "run.yaml"), data, 0o644)
}

// applyAll applies what is not applied yet, recording each success, so an
// interrupted run can be resumed where it stopped.
func applyAll(r *role.Role, settings map[string]any, st runState, runDir string, intents []intent.Intention, res *Result) error {
	f, err := forge.Open(st.Forge, st.Repo)
	if err != nil {
		return err
	}
	ap := applier{repo: st.Repo, runDir: runDir, targets: st.Targets, writes: r.Writes(settings),
		forge: f, target: st.Target, runID: filepath.Base(runDir), role: r.Name, settings: settings, own: st.trailers()}
	for _, c := range res.Calls {
		if c.Task != "judge" && c.Model != "" {
			ap.model = c.Model
		}
	}
	plan, err := planActs(f, r, settings, st, runDir, intents)
	if errors.Is(err, forge.ErrUnreachable) {
		res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while reading the backlog; resume with: workline apply %s", runDir)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
		return nil
	}
	if err != nil {
		return err
	}
	ap.plan = plan
	done := map[int]bool{}
	for _, i := range st.Applied {
		done[i] = true
	}
	// applyOne applies intention i; stopped when the forge is out of reach.
	applyOne := func(i int) (stopped bool, err error) {
		in := intents[i]
		acted := !slices.Contains(backlog.Kinds, in.Kind) || plan.Acts(i)
		if done[i] {
			if acted {
				res.Applied = append(res.Applied, in.Kind)
			}
			return false, nil
		}
		ap.index = i
		if err := ap.apply(in); err != nil {
			if errors.Is(err, forge.ErrUnreachable) {
				res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while applying %s; resume with: workline apply %s", in.Kind, runDir)
				res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
				return true, nil
			}
			return false, fmt.Errorf("apply %s: %w", in.Kind, err)
		}
		st.Applied = append(st.Applied, i)
		for _, w := range ap.written { // a resumed run adds to what the first attempt wrote
			if !slices.Contains(st.Written, w) {
				st.Written = append(st.Written, w)
			}
		}
		if err := st.save(runDir); err != nil {
			return false, err
		}
		if acted {
			res.Applied = append(res.Applied, in.Kind)
		}
		return false, nil
	}
	// The role's own branch, asked by the role (the tester's rewrite on an
	// objection, ADR-0055): its patches first, committed on top and
	// pushed, never forced; its comments only once the commit landed, so
	// a record never names what is not on the branch.
	ownMessage := ownPush(runDir)
	ownBranch := ownMessage != "" && f != nil && st.Target != nil && st.Target.Kind == "merge-request" && !st.OpenMergeRequest
	var order, after []int
	for i, in := range intents {
		if ownBranch && in.Kind != "patch" {
			after = append(after, i)
		} else {
			order = append(order, i)
		}
	}
	for _, i := range order {
		if stopped, err := applyOne(i); stopped || err != nil {
			return err
		}
	}
	if ownBranch && len(st.Written) > 0 && !st.Pushed {
		if err := commitOnOwnBranch(f, st, ownMessage); err != nil {
			if errors.Is(err, forge.ErrUnreachable) {
				res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while pushing to the role's branch; resume with: workline apply %s", runDir)
				res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
				return nil
			}
			res.Status, res.Summary = verdict.Block, "not pushed to the role's branch: nothing else is said"
			res.Findings = append(res.Findings, verdict.Finding{Rule: "not-pushed", Message: err.Error()})
			return nil
		}
		st.Pushed = true
		if err := st.save(runDir); err != nil {
			return err
		}
	}
	for _, i := range after {
		if stopped, err := applyOne(i); stopped || err != nil {
			return err
		}
	}
	res.Findings = append(res.Findings, ap.findings...)
	if ap.capped > 0 {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "issues-capped", Level: "warn",
			Message: fmt.Sprintf("%d more issues not opened (issues-max %d): found again, they are opened at a later run", ap.capped, ap.opens.Max)})
	}
	if plan != nil {
		links(f, res)
		// Written back first: what it labels is named in the summary
		// (#428).
		err := report(f, r.Name, plan)
		res.Findings = append(res.Findings, plan.Findings...)
		if res.Status == verdict.Pass {
			res.Summary = onIssues(res.Findings, false, st.Unasked)
		}
		if errors.Is(err, forge.ErrUnreachable) {
			res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while writing back what the run left on the issues; resume with: workline apply %s", runDir)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
			return nil
		} else if err != nil {
			return fmt.Errorf("writing back the issues: %w", err)
		}
	}
	res.Handoffs = ap.handoffs
	if st.OpenMergeRequest && len(st.Written) > 0 && st.MergeRequest == 0 {
		id, err := openMergeRequest(f, st, runDir)
		if errors.Is(err, forge.ErrUnreachable) {
			res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while opening the merge request; resume with: workline apply %s", runDir)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
			return nil
		}
		switch {
		case errors.Is(err, ErrLeftToAPerson):
			// Theirs now: this run's change stays in the working tree, said.
			res.Findings = append(res.Findings, verdict.Finding{Rule: "not-proposed", Level: "warn", Where: branchPrefix(st.Role) + slugify(mergeRequestKey(st, runDir)),
				Message: strings.TrimPrefix(err.Error(), ErrLeftToAPerson.Error()+": ") + ": this run's change is not pushed over it; the branch and its merge request are left to them"})
		case err != nil:
			return fmt.Errorf("open a merge request: %w", err)
		default:
			st.MergeRequest = id
			if err := st.save(runDir); err != nil {
				return err
			}
			// The role's record on the merge request it opened, in
			// workline's own comment (ADR-0051): the tester's tests and
			// their hashes, which only its own runs move (ADR-0055).
			if key, body := openedRecord(runDir); key != "" {
				t := forge.Target{Kind: "merge-request", ID: id}
				if err := forge.KeepOwn(f, t, backlog.Inert(body), forge.Marker("sticky="+st.Role+"/"+key), true); err != nil {
					return fmt.Errorf("the record on the merge request opened: %w", err)
				}
			}
		}
	}
	res.MergeRequest = st.MergeRequest
	if ownBranch {
		return nil // its commit is pushed above, never a comment
	}
	// On a merge request, what the patches wrote goes in a comment, never
	// on its branch (ADR-0051).
	if f != nil && st.Target != nil && st.Target.Kind == "merge-request" && !st.OpenMergeRequest && len(st.Written) > 0 && !st.Pushed {
		finding, err := pushToMergeRequest(f, st, runDir)
		if errors.Is(err, forge.ErrUnreachable) {
			res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while pushing to the merge request; resume with: workline apply %s", runDir)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
			return nil
		}
		if err != nil {
			return fmt.Errorf("push to the merge request: %w", err)
		}
		st.Pushed = true
		if err := st.save(runDir); err != nil {
			return err
		}
		res.Findings = append(res.Findings, finding)
	}
	return nil
}

// openedRecord is the record a role keeps on the merge request it opens,
// out/merge-request.yaml's `record: {sticky: key, body}`; "" for none.
func openedRecord(runDir string) (key, body string) {
	var mr struct {
		Record struct{ Sticky, Body string }
	}
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err != nil || yaml.Unmarshal(data, &mr) != nil {
		return "", ""
	}
	return mr.Record.Sticky, mr.Record.Body
}

// ownPush is the commit message a role asks its patches pushed with, on
// the merge request it targets, out/merge-request.yaml's `push:`; "" for
// none: then they go in a comment.
func ownPush(runDir string) string {
	var mr struct{ Push string }
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err != nil || yaml.Unmarshal(data, &mr) != nil {
		return ""
	}
	return strings.TrimSpace(mr.Push)
}

// commitOnOwnBranch commits what the patches wrote on the merge request's
// branch, the role's own (workline/<role>/…, in this repository), on top
// of what others pushed, and pushes it without force: the working tree
// must be at that branch's tip, else nothing is committed. Another's
// branch is never written to (ADR-0051).
func commitOnOwnBranch(f forge.Forge, st runState, message string) error {
	mr, err := f.MergeRequest(st.Target.ID)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(mr.Branch, branchPrefix(st.Role)) || !mr.Here {
		return fmt.Errorf("merge request %d comes from %s, not a branch of the %s's own: nothing is pushed there", st.Target.ID, mr.Branch, st.Role)
	}
	head, err := git(st.Repo, nil, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	tipRef := mr.Branch // the clone keeps it; else its tip on origin, fetched
	if !forge.KeepsBranches(f) {
		if _, err := git(st.Repo, nil, "fetch", "-q", "origin", mr.Branch); err != nil {
			return fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
		}
		tipRef = "FETCH_HEAD"
	}
	tip, err := git(st.Repo, nil, "rev-parse", tipRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tip) != strings.TrimSpace(head) {
		return fmt.Errorf("the working tree is at %s, not at %s's tip %s: check the branch out first; nothing is committed", short(head), mr.Branch, short(tip))
	}
	steps := [][]string{
		append([]string{"add", "--"}, st.Written...),
		{"commit", "-q", "-m", message, "-m", st.trailers()},
	}
	for _, s := range steps {
		if _, err := git(st.Repo, nil, s...); err != nil {
			_, _ = git(st.Repo, nil, append([]string{"reset", "-q", "--"}, st.Written...)...)
			return err
		}
	}
	if forge.KeepsBranches(f) {
		if on, _ := git(st.Repo, nil, "symbolic-ref", "-q", "--short", "HEAD"); strings.TrimSpace(on) == mr.Branch {
			return nil
		}
		_, err := git(st.Repo, nil, "branch", "-f", mr.Branch, "HEAD")
		return err
	}
	if _, err := git(st.Repo, nil, "push", "-q", "origin", "HEAD:refs/heads/"+mr.Branch); err != nil {
		return fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return nil
}

// pushToMergeRequest puts the diff of what the run's patches wrote in one
// comment on the merge request it targets, for its author to apply: workline
// pushes nothing to a merge request's branch (ADR-0051), not even a move of
// `checked`.
func pushToMergeRequest(f forge.Forge, st runState, runDir string) (verdict.Finding, error) {
	diff, err := writtenDiff(st)
	if err != nil {
		return verdict.Finding{}, err
	}
	body := fmt.Sprintf("**Proposed by workline's %s**, not pushed: workline pushes nothing to a merge request's branch. Apply it with `git apply`:\n\n%s", st.Role, diffBlock(diff))
	if err := f.Sticky(*st.Target, body, forge.Marker("sticky="+st.Role+"/patch"), true); err != nil {
		return verdict.Finding{}, err
	}
	return verdict.Finding{Rule: "patch-as-comment", Level: "warn", Message: "the patches went in a comment: workline pushes nothing to a merge request's branch"}, nil
}

// writtenDiff is the diff of the files the run wrote, as git prints it:
// never trimmed, since a hunk's last line may be a blank line of context,
// and without it the diff no longer applies. A file the run created — a
// split's or a condense's new doc — is not tracked yet, so `git diff`
// leaves it out: it is diffed against nothing, a new file, the index left
// as it was.
func writtenDiff(st runState) (string, error) {
	if len(st.Written) == 0 {
		return "", nil // no path: git would diff, and list, the whole tree
	}
	diff, err := gitRaw(st.Repo, append([]string{"diff", "--"}, st.Written...)...)
	if err != nil {
		return "", err
	}
	created, err := gitRaw(st.Repo, append([]string{"ls-files", "-z", "--others", "--"}, st.Written...)...)
	if err != nil {
		return "", err
	}
	for _, p := range strings.Split(created, "\x00") {
		if p == "" {
			continue
		}
		d, err := gitRaw(st.Repo, "diff", "--no-index", "--", os.DevNull, p)
		if err != nil {
			return "", err
		}
		diff += d
	}
	return diff, nil
}

// gitRaw runs git in repo and returns what it printed, untrimmed. `diff
// --no-index` exits 1 when the files differ: that is its answer, not a
// failure.
func gitRaw(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.quotePath=off"}, args...)...)
	var errOut strings.Builder
	cmd.Stderr = &errOut
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 && slices.Contains(args, "--no-index") {
		err = nil
	}
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(errOut.String()))
	}
	return string(out), nil
}

// diffBlock is a diff as a comment shows it, for a person to copy whole and
// apply with `git apply`: its fence longer than any run of backticks in it,
// so a code block of a doc it changes does not end the block early (#145).
func diffBlock(diff string) string {
	if !strings.HasSuffix(diff, "\n") {
		diff += "\n"
	}
	fence := backlog.FenceFor(diff)
	return fence + "diff\n" + diff + fence
}

// rolesHere is where the roles a run was judged with are on this machine. A
// run judged on another — CI's judge job, applied by its apply job — names
// the cache the engine's built-in roles were extracted to there; here they
// are extracted again from this engine.
func rolesHere(dir string) string {
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	if strings.HasPrefix(filepath.Base(dir), "roles-") {
		if here, err := rolefs.Dir(); err == nil {
			return here
		}
	}
	return dir
}

// OwnTrailer marks the commits the engine makes itself, naming the role, so
// that they do not wake the line again (docs/spec/routing.md).
const OwnTrailer = "Workline-Role"

// ModelTrailer names, in the engine's own commits, each agent and model whose
// answer they hold, as <agent>:<model> (claude:claude-sonnet-5).
const ModelTrailer = "Workline-Model"

// Rewriter is the role whose trailer marks a person's commit whose
// message it rewrote (ADR-0046): never a commit workline made.
const Rewriter = "committer"

// madeBy are the roles a commit's Workline-Role trailers say made it, a
// comma between two: the committer's left out, as a rewrite's.
func madeBy(roles string) string {
	var out []string
	for _, r := range strings.Split(roles, ",") {
		if r = strings.TrimSpace(r); r != "" && !strings.EqualFold(r, Rewriter) {
			out = append(out, r)
		}
	}
	return strings.Join(out, ",")
}

// withTrailers is a rewritten message with the run's own trailers — its
// role, the agents and models that wrote it — in its trailer block, as
// git places them; never the words it replaced (ADR-0046).
func withTrailers(repo, msg, trailers string) (string, error) {
	args := []string{"interpret-trailers", "--if-exists", "addIfDifferent"}
	for _, t := range strings.Split(trailers, "\n") {
		args = append(args, "--trailer", t)
	}
	return git(repo, strings.NewReader(msg+"\n"), args...)
}

// branchPrefix is where a role's merge requests come from (ADR-0006).
func branchPrefix(roleName string) string { return "workline/" + roleName + "/" }

// DefaultTitle is a role's merge request's title when it names none, or
// one too long for a squash's " (#N)".
func DefaultTitle(roleName string) string {
	return "chore(" + roleName + "): what the " + roleName + " proposes"
}

// openMergeRequest commits what the run's patches wrote on the role's branch
// for this task, force-pushes it, opens its merge request or updates the one
// open, and puts the working tree back on the branch it was on. The role
// names the task's key and the title in out/merge-request.yaml.
func openMergeRequest(f forge.Forge, st runState, runDir string) (int, error) {
	var mr struct {
		Key, Title, Body string
		Draft            bool
		Proposal         `yaml:",inline"`
	}
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &mr); err != nil {
			return 0, fmt.Errorf("out/merge-request.yaml: %w", err)
		}
	}
	mr.Key = mergeRequestKey(st, runDir)
	limit := forge.TitleMax // a squash adds " (#N)" (ADR-0051)
	if mr.Draft {
		limit -= len("Draft: ") // GitLab's draft is a title prefix
	}
	if mr.Title == "" || len([]rune(mr.Title)) > limit {
		mr.Title = DefaultTitle(st.Role)
	}
	return ProposeBranchFrom(f, st.Repo, st.Role, mr.Key, mr.Title, mr.Body, mr.Draft, st.trailers(), st.Written, mr.Proposal)
}

// Proposal is how a role's branch is built, beyond the files written
// (out/merge-request.yaml): the commit it starts from, and the files in
// commits apart.
type Proposal struct {
	// From is the commit the branch starts from, "" the base's tip: the
	// developer's on the tester's Draft (ADR-0054). Its commits past the
	// base are not a person's.
	From string `yaml:"from,omitempty"`
	// Commits are the files written, each group its own commit, in order;
	// those no group names go in the last, the title its message (the
	// developer's code and tests apart, ADR-0054).
	Commits []Commit `yaml:"commits,omitempty"`
}

// Commit is one commit of a branch: its message, the files it holds.
type Commit struct {
	Message string   `yaml:"message"`
	Files   []string `yaml:"files"`
}

// commits are the commits the written files go in: the groups named, each
// holding the files written it names, then the rest under the title.
func (p Proposal) commits(title string, written []string) []Commit {
	var out []Commit
	left := slices.Clone(written)
	for _, c := range p.Commits {
		var files []string
		for _, f := range c.Files {
			if i := slices.Index(left, f); i >= 0 {
				files = append(files, f)
				left = slices.Delete(left, i, i+1)
			}
		}
		if len(files) > 0 && strings.TrimSpace(c.Message) != "" {
			out = append(out, Commit{Message: c.Message, Files: files})
		}
	}
	if len(left) > 0 || len(out) == 0 {
		out = append(out, Commit{Message: title, Files: left})
	}
	return out
}

// mergeRequestKey is the task's key the role named in
// out/merge-request.yaml, its branch's name; the role's name without one.
func mergeRequestKey(st runState, runDir string) string {
	var mr struct{ Key string }
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		yaml.Unmarshal(data, &mr)
	}
	if mr.Key == "" {
		return st.Role
	}
	return mr.Key
}

// ErrLeftToAPerson says a role's branch holds a commit a person added:
// nothing is pushed over it, the branch and its merge request are theirs
// (ADR-0034, amended).
var ErrLeftToAPerson = errors.New("left to a person")

// ProposeBranch commits the files written in the working tree on the role's
// branch for a task (workline/<role>/<key>), force-pushes it — unless the
// forge keeps its branches in the clone — opens its merge request or
// updates the one open, and puts the working tree back on the branch it was
// on: a draft, when draft is true. trailers end the commit's message. A
// branch a person committed to is never pushed over: ErrLeftToAPerson, the
// working tree as it was.
func ProposeBranch(f forge.Forge, repo, roleName, key, title, body string, draft bool, trailers string, written []string) (int, error) {
	return ProposeBranchFrom(f, repo, roleName, key, title, body, draft, trailers, written, Proposal{})
}

// ProposeBranchFrom is ProposeBranch, the branch built as p says: from
// p.From, the files written in p's commits.
func ProposeBranchFrom(f forge.Forge, repo, roleName, key, title, body string, draft bool, trailers string, written []string, p Proposal) (int, error) {
	base, err := git(repo, nil, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return 0, errors.New("a merge request is opened from a branch; this run is on a detached HEAD")
	}
	branch := branchPrefix(roleName) + slugify(key)
	baseTip, was, err := tips(repo, base, branch, forge.KeepsBranches(f))
	if err != nil {
		return 0, err
	}
	if was != "" {
		// The commits it started from are not a person's; on another role's
		// commit, nor that role's older ones, rewritten since.
		since := []string{baseTip}
		if p.From != "" {
			since = append(since, p.From)
		}
		if who := personsCommitPast(repo, since, was, roleName, p.From != ""); who != "" {
			return 0, fmt.Errorf("%w: %s", ErrLeftToAPerson, who)
		}
	}
	start := []string{"checkout", "-q", "-B", branch}
	if p.From != "" {
		start = append(start, p.From) // the files written carried over
	}
	steps := [][]string{start}
	for _, c := range p.commits(title, written) {
		steps = append(steps, append([]string{"add", "--"}, c.Files...), []string{"commit", "-q", "-m", c.Message, "-m", trailers})
	}
	if !forge.KeepsBranches(f) {
		steps = append(steps, []string{"push", "-q", "--force", "origin", branch})
	}
	for _, s := range steps {
		if _, err := git(repo, nil, s...); err != nil {
			// Back where it was: the patches in the working tree, unstaged.
			git(repo, nil, append([]string{"reset", "-q", "--"}, written...)...)
			git(repo, nil, "checkout", "-q", base)
			if s[0] == "push" {
				return 0, fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
			}
			return 0, err
		}
	}
	if _, err := git(repo, nil, "checkout", "-q", base); err != nil {
		return 0, err
	}
	return f.OpenMergeRequest(branch, base, title, body, draft)
}

// slugify keeps a branch name to lowercase letters, digits and dashes.
func slugify(s string) string { return intent.Slug(s) }

// Resume applies what an interrupted run had not applied yet, from what that
// run recorded. The agent is not called again, and nothing is applied if the
// prepared input changed since.
func Resume(runDir string) *Result {
	res := &Result{Applied: []string{}, Refused: []string{}, RunDir: runDir}
	err := func() error {
		data, err := os.ReadFile(filepath.Join(runDir, "out", "run.yaml"))
		if err != nil {
			return fmt.Errorf("not a run that reached apply: %w", err)
		}
		var st runState
		if err := yaml.Unmarshal(data, &st); err != nil {
			return err
		}
		now, err := dirDigest(filepath.Join(runDir, "in"))
		if err != nil {
			return err
		}
		if now != st.Digest {
			res.Status = verdict.Block
			res.Findings = append(res.Findings, verdict.Finding{Rule: "input-changed", Message: "the prepared input changed since the run; nothing was applied"})
			return nil
		}
		r, err := role.Load(rolesHere(st.RolesDir), st.Role)
		if err != nil {
			return err
		}
		cfg, err := role.LoadProjectConfig(st.Repo)
		if err != nil {
			return err
		}
		intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
		if err != nil {
			return err
		}
		res.Status = verdict.Pass
		if err := applyAll(r, r.MergedSettings(cfg), st, runDir, intents, res); err != nil {
			return err
		}
		for _, h := range res.Handoffs {
			res.Findings = append(res.Findings, deferred(r.Name, h))
		}
		return nil
	}()
	if err != nil {
		res.Status = verdict.Block
		res.Findings = append(res.Findings, verdict.Finding{Rule: "engine-error", Message: err.Error()})
	}
	return res
}

// deferred says that a handoff was recorded and its role not run: applying
// runs no role, since the job that applies holds no AI key (--no-apply). It is
// not a failure, but it must not pass silently.
func deferred(from string, h any) verdict.Finding {
	m, _ := h.(map[string]any)
	to, _ := m["role"].(string)
	reason, _ := m["reason"].(string)
	return verdict.Finding{Rule: "handoff-deferred", Where: from,
		Message: fmt.Sprintf("%s asked for %s (%s), which applying does not run; run it where an agent may judge: workline run-role %s --event handoff --input handoff-from=%s", from, to, reason, to, from)}
}

// outOfBounds lists the files patches would touch outside the role's duties
// or the task's scope.
func outOfBounds(repo string, in []intent.Intention, writes, scope []string) []string {
	var bad []string
	for _, i := range in {
		if i.Kind != "patch" {
			continue
		}
		for _, f := range patchFiles(repo, i.Value) {
			if !pathglob.Any(writes, f) || (len(scope) > 0 && !pathglob.Any(scope, f)) {
				bad = append(bad, f)
			}
		}
	}
	return bad
}

// patchFiles lists the files a patch touches; an unreadable diff lists nothing
// here and fails at apply.
func patchFiles(repo string, v any) []string {
	if m, ok := v.(map[string]any); ok {
		if f, ok := m["file"].(string); ok {
			return []string{filepath.ToSlash(filepath.Clean(f))}
		}
		return nil
	}
	diff, _ := v.(string)
	out, err := git(repo, strings.NewReader(intent.NormalizeDiff(diff)), "apply", "--recount", "--unidiff-zero", "--numstat", "-")
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) == 3 {
			files = append(files, f[2])
		}
	}
	return files
}

// taskKind is the kind of question pre named in in/task-kind, if any.
func taskKind(runDir string) string {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "task-kind"))
	return strings.TrimSpace(string(data))
}

// askAgain says whether a refused proposal is asked for again, and on which
// tier: `promote-after` refusals on the role's tier, then one last attempt a
// tier up (docs/spec/model-grid.md). 0 never asks again. failures counts the
// refusals so far, this one included.
func askAgain(failures, promoteAfter int, tier string) (bool, string) {
	if promoteAfter <= 0 || failures > promoteAfter {
		return false, tier
	}
	if failures == promoteAfter {
		return true, stepUp(tier)
	}
	return true, tier
}

var tiers = []string{"light", "standard", "frontier"}

func stepUp(tier string) string {
	for i, t := range tiers {
		if t == tier && i+1 < len(tiers) {
			return tiers[i+1]
		}
	}
	return tier
}

type attemptResult struct {
	intents    []intent.Intention
	verdict    *verdict.Verdict
	findings   []verdict.Finding // why proposals were refused before judging
	askedAgent bool
	external   bool
	unread     string // why the agent's answer could not be read, if it could not
	overBudget bool   // the task was not put before the agent: over the context budget
}

// attempt asks the agent (when there is a question and an agent), merges the
// fallback proposals, refuses what cannot be applied, and runs the judge.
func attempt(r *role.Role, o Options, ag agent.Agent, hasTask bool, tier, runDir string, env []string,
	line *routing.Config, settings map[string]any, cfg *role.ProjectConfig, res *Result) (*attemptResult, error) {
	a := &attemptResult{}
	if hasTask && ag != nil {
		a.askedAgent = true
		err := callAgent(ag, agent.Request{RunDir: runDir, Repo: o.Repo, Role: r, Tier: tier}, taskKind(runDir), runDir, res)
		switch {
		case err == nil:
		case errors.Is(err, errTokensSpent), errors.Is(err, errTimeSpent):
			a.askedAgent = false // not asked, or not waited for: no answer to refuse, none to ask again
		case errors.Is(err, agent.ErrOverBudget):
			// A task too large for the role's context is the role's to
			// shrink, not a block of the line: the run goes on as without
			// an agent, and what the task held waits for a person.
			a.askedAgent, a.overBudget = false, true
			a.findings = append(a.findings, verdict.Finding{Rule: "prompt-over-budget", Level: "warn",
				Message: err.Error() + ": the agent was not asked; what this task held is judged by no one this run, a person reads it (in/task.md of the run)"})
		case errors.Is(err, agent.ErrUnavailable):
			a.external = true
			a.findings = append(a.findings, verdict.Finding{Rule: "agent-unavailable", Message: err.Error()})
		case errors.Is(err, agent.ErrInvalidOutput):
			a.unread = err.Error()
			a.findings = append(a.findings, verdict.Finding{Rule: "agent-invalid-output", Message: err.Error()})
		default:
			return nil, err
		}
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		a.unread = err.Error()
		a.findings = append(a.findings, verdict.Finding{Rule: "agent-invalid-output", Message: err.Error()})
		intents = nil
	}
	fallback, err := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	if err != nil {
		return nil, err
	}
	if !a.askedAgent || a.external || a.unread != "" {
		// A fallback marked if-answered records that the agent read what it
		// was given: without an answer that reads, it is not written.
		fallback = slices.DeleteFunc(fallback, func(f intent.Intention) bool {
			m, _ := f.Value.(map[string]any)
			only, _ := m["if-answered"].(bool)
			return only
		})
	}
	intents = intent.Merge(fallback, intents)
	os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return nil, err
	}
	res.Refused = append([]string{}, res.partRefused...)
	if refused, why := invalid(r, intents, line); len(refused) > 0 {
		res.Refused = append(res.Refused, refused...) // the set is refused whole; these are the kinds that caused it
		a.findings = append(a.findings, verdict.Finding{Rule: "intention-refused", Message: why})
		intents = nil
		os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	}
	if bad := outOfBounds(o.Repo, intents, r.Writes(settings), o.Scope); len(bad) > 0 {
		res.Refused = append(append([]string{}, res.partRefused...), "patch")
		a.findings = append(a.findings, verdict.Finding{Rule: "intention-refused",
			Message: "a patch reaches outside the task or the role's duties: " + strings.Join(bad, ", ") + "; anything found there belongs in an issue"})
		intents = nil
		os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	}
	a.intents = intents

	code, err := script(r, "post", o.Repo, env)
	if err != nil {
		return nil, err
	}
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		return nil, fmt.Errorf("post wrote no readable verdict: %w", err)
	}
	if want := statusForExit(code); want == "" {
		return nil, fmt.Errorf("post exited with code %d", code)
	} else if want != v.Status {
		return nil, fmt.Errorf("post exited %d but its verdict says %q", code, v.Status)
	}
	verdict.Enforce(v, r.Enforcement(cfg))
	// post may narrow the proposals it passes: a place it refuses taken out,
	// the rest kept, rather than the whole task asked again for it (the
	// documentalist, ADR-0014 step 4). What it leaves in out/intentions.yaml
	// is what is applied, held to the same catalogue and bounds. A release
	// held for what the run did not settle keeps them as a pass does (#543).
	if v.Status == verdict.Pass && len(intents) > 0 {
		if a.intents, err = narrowedByPost(r, o, runDir, line, settings); err != nil {
			return nil, err
		}
	}
	if v.Status == verdict.Pass && a.askedAgent {
		if err := askJudge(o, res, runDir, v, a); err != nil {
			return nil, err
		}
	}
	v.Findings = append(a.findings, v.Findings...)
	// On an event that fails closed — a gate: commit-msg, pre-push, a merge
	// request, the release — a task nobody judged never passes: a person
	// decides. On one whose steps all run (schedule, ADR-0037), it is upkeep
	// left for later: reported, and the line goes on.
	if a.overBudget && v.Status == verdict.Pass && (line == nil || line.StopsAtFirst(o.Event)) {
		v.Status = verdict.Human
		v.Summary = overBudgetSummary(v.Summary)
	}
	a.verdict = v
	return a, nil
}

// warnsOnMergeRequest says whether the role's agent only warns on this
// run: a merge request, for a role whose agent judges it once, at ready.
func warnsOnMergeRequest(r *role.Role, o Options) bool {
	return o.Event == "merge-request" && r.MergeRequestAI == "ready"
}

// leaveOut takes out of the patches the files the refusals name, and says
// why each was left: a finding a person reads. Nothing is left out when the
// refusals name no file a patch touches, or when no fix of the agent's is
// left to apply: then the refusal stands.
func leaveOut(intents, fallback []intent.Intention, refusals []verdict.Finding) ([]intent.Intention, []verdict.Finding) {
	why := map[string]string{}
	for _, f := range refusals {
		if f.Where != "" && !strings.ContainsAny(f.Where, " \t") && f.Where != "patch" && f.Where != "proposal" {
			if why[f.Where] == "" {
				why[f.Where] = f.Rule + ": " + f.Message
			}
		}
	}
	var kept []intent.Intention
	var left []verdict.Finding
	seen := map[string]bool{}
	agentKept := 0
	for _, in := range intents {
		diff, ok := in.Value.(string)
		if in.Kind != "patch" || !ok || isFallbackOf(in, fallback) {
			kept = append(kept, in)
			continue
		}
		var keep []string
		for _, s := range splitDiff(diff) {
			if reason := why[s.path]; reason != "" {
				if !seen[s.path] {
					seen[s.path] = true
					left = append(left, verdict.Finding{Rule: "left-out", Where: s.path, Level: "warn",
						Message: "its fix was refused, after asking again (" + reason + "): left out, the other docs' fixes applied; it stays as it was"})
				}
				continue
			}
			keep = append(keep, s.text)
		}
		if len(keep) > 0 {
			in.Value = strings.Join(keep, "")
			kept = append(kept, in)
			agentKept += len(keep)
		}
	}
	if len(left) == 0 || agentKept == 0 {
		return intents, nil
	}
	return kept, left
}

// isFallbackOf says whether a proposal is one the role made itself, not the
// agent: a derived block regenerated, never a fix to keep or leave out.
func isFallbackOf(in intent.Intention, fallback []intent.Intention) bool {
	for _, f := range fallback {
		if f.Kind == in.Kind && fmt.Sprint(f.Value) == fmt.Sprint(in.Value) {
			return true
		}
	}
	return false
}

// fileDiff is the part of a unified diff about one file.
type fileDiff struct{ path, text string }

// splitDiff cuts a unified diff into its files: each starts at `diff --git`,
// or at a `--- ` line followed by `+++ `.
func splitDiff(diff string) []fileDiff {
	lines := strings.SplitAfter(diff, "\n")
	var out []fileDiff
	for i, l := range lines {
		start := strings.HasPrefix(l, "diff --git ") ||
			strings.HasPrefix(l, "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ") && (i == 0 || !strings.HasPrefix(lines[i-1], "diff --git "))
		if start || len(out) == 0 {
			out = append(out, fileDiff{})
		}
		cur := &out[len(out)-1]
		cur.text += l
		if p, ok := strings.CutPrefix(l, "+++ "); ok && cur.path == "" {
			cur.path = strings.TrimPrefix(strings.TrimSpace(p), "b/")
		}
	}
	return out
}

// judgeAgain runs post on the proposals kept, without asking the agent.
func judgeAgain(r *role.Role, o Options, runDir string, env []string, cfg *role.ProjectConfig, kept []intent.Intention, res *Result) (*verdict.Verdict, error) {
	for _, f := range []string{"intentions.yaml", "verdict.yaml", "judge.yaml"} {
		os.Remove(filepath.Join(runDir, "out", f))
	}
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), kept); err != nil {
		return nil, err
	}
	code, err := script(r, "post", o.Repo, env)
	if err != nil {
		return nil, err
	}
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		return nil, fmt.Errorf("post wrote no readable verdict: %w", err)
	}
	if want := statusForExit(code); want != v.Status {
		return nil, fmt.Errorf("post exited %d but its verdict says %q", code, v.Status)
	}
	verdict.Enforce(v, r.Enforcement(cfg))
	if v.Status == verdict.Pass {
		if err := askJudge(o, res, runDir, v, &attemptResult{askedAgent: true}); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// narrowedByPost reads the proposals post passed, as it left them in
// out/intentions.yaml: post may take out a place it refuses and keep the
// rest (the documentalist, ADR-0014 step 4). They are held to the same
// catalogue and bounds as the agent's.
func narrowedByPost(r *role.Role, o Options, runDir string, line *routing.Config, settings map[string]any) ([]intent.Intention, error) {
	narrowed, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return nil, fmt.Errorf("post left out/intentions.yaml unreadable: %w", err)
	}
	if refused, why := invalid(r, narrowed, line); len(refused) > 0 {
		return nil, fmt.Errorf("post narrowed the proposals to an invalid set: %s", why)
	}
	if bad := outOfBounds(o.Repo, narrowed, r.Writes(settings), o.Scope); len(bad) > 0 {
		return nil, fmt.Errorf("post narrowed the proposals outside the task: %s", strings.Join(bad, ", "))
	}
	return narrowed, nil
}

func statusForExit(code int) string {
	switch code {
	case exitOK:
		return verdict.Pass
	case exitBlock:
		return verdict.Block
	case exitHuman:
		return verdict.Human
	case exitExternal:
		return verdict.BlockedExternal
	}
	return ""
}

// invalid returns the kinds that cannot be applied, and why: not in the
// catalogue, not allowed for the role, or a handoff routing does not declare.
func invalid(r *role.Role, in []intent.Intention, line *routing.Config) ([]string, string) {
	var bad, why []string
	patches := false
	for _, i := range in {
		patches = patches || i.Kind == "patch"
	}
	for _, i := range in {
		switch {
		case i.Kind == "claim" && (!r.Allows("claim") || !patches):
			// Beside a patch, a claim cites why it takes words out (ADR-0014).
			bad, why = append(bad, i.Kind), append(why, "a claim answers a part of a question (in/parts), or cites why a patch takes words out, beside it; never the question itself")
		case !intent.Catalogue[i.Kind] || !r.Allows(i.Kind):
			bad, why = append(bad, i.Kind), append(why, fmt.Sprintf("%s is not allowed for this role", i.Kind))
		case i.Kind == "handoff":
			m, _ := i.Value.(map[string]any)
			to, _ := m["role"].(string)
			if !line.Allowed(r.Name, to) {
				bad, why = append(bad, i.Kind), append(why, fmt.Sprintf("routing declares no handoff from %s to %q", r.Name, to))
			}
		}
	}
	return bad, strings.Join(why, "; ")
}

// applier carries out intentions. This version knows the local ones.
type applier struct {
	repo, runDir string
	targets      map[string]string
	writes       []string    // paths the role may write
	written      []string    // files changed by this run's patches
	handoffs     []any       // next roles asked for, recorded for routing
	forge        forge.Forge // nil when the project has no forge
	target       *forge.Target
	runID, role  string
	index        int           // position of the intention being applied, for its marker
	plan         *backlog.Plan // what becomes of the acts on the backlog, nil without any
	settings     map[string]any
	opens        *backlog.Openings // the run's issues opened, made on the first
	capped       int               // findings not opened as issues, past issues-max
	findings     []verdict.Finding // what the applying says
	model        string            // the model whose answer is applied, as an announcement records it
	own          string            // the run's own trailers: its role, the agents and models that answered
	scheme       *scale.Scheme     // where the project keeps each axis, detected at the first act writing one (ADR-0040)
	ensured      map[string]bool   // the labels this run made sure the project has
}

func (a *applier) marker() string { return forge.Marker(fmt.Sprintf("run=%s/%d", a.runID, a.index)) }

// noteText is a note as a person reads it: its text, after the docs it
// names when it names some — `{doc, text}`, a documentalist's note on what
// it could not confirm in a doc (#449).
func noteText(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	t, ok := m["text"]
	if !ok {
		return strings.TrimSpace(fmt.Sprint(v)) // not a {doc, text} note
	}
	text := "" // `text:` left empty
	if t != nil {
		text = fmt.Sprint(t) // a text YAML read as a number or a list, as it reads
	}
	var docs []string
	switch d := m["doc"].(type) {
	case string:
		docs = append(docs, d)
	case []any:
		for _, x := range d {
			if s, ok := x.(string); ok {
				docs = append(docs, s)
			}
		}
	}
	if len(docs) == 0 {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(strings.Join(docs, ", ") + ": " + strings.TrimSpace(text))
}

func (a *applier) needForge(kind string) error {
	if a.forge == nil {
		return fmt.Errorf("a %s needs a forge; %s", kind, forge.Missing)
	}
	return nil
}

func (a *applier) apply(in intent.Intention) error {
	switch in.Kind {
	case "commit-message":
		msg, ok := in.Value.(string)
		if !ok {
			return errors.New("value must be text")
		}
		// A rewrite says who wrote it, so the weekly sample draws it.
		msg, err := withTrailers(a.repo, msg, a.own)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(a.runDir, "out", "commit-message"), []byte(msg), 0o644); err != nil {
			return err
		}
		if t := a.targets["message"]; t != "" {
			return os.WriteFile(t, []byte(msg+"\n"), 0o644)
		}
		return nil
	case "note":
		// Several notes of one answer are all kept, one after the other.
		f, err := os.OpenFile(filepath.Join(a.runDir, "out", "note.md"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprintf(f, "%s\n\n", noteText(in.Value))
		return err
	case "patch":
		return a.patch(in.Value)
	case "comment":
		// A sticky comment, {body, sticky: key}, is one comment edited on
		// each run; with update-only, it is never created.
		body, _ := in.Value.(string)
		m, sticky := in.Value.(map[string]any)
		if sticky {
			body, _ = m["body"].(string)
		}
		if err := a.needForge("comment"); err != nil {
			return err
		}
		target := a.target
		if n, ok := m["issue"].(int); ok { // a role keeping the backlog comments on the issue it names
			target = &forge.Target{Kind: "issue", ID: n}
		} else if n, ok := m["merge-request"].(int); ok { // the developer's objection, on the tester's Draft (ADR-0054)
			target = &forge.Target{Kind: "merge-request", ID: n}
		}
		if target == nil {
			return errors.New("a comment needs a target issue or merge request (--target)")
		}
		if thread, ok := m["thread"].(string); ok && thread != "" {
			// A reply in a conversation on a line of the merge request:
			// the tester's answer to an objection (ADR-0055).
			th, ok := a.forge.(forge.Threaded)
			if !ok || target.Kind != "merge-request" {
				return errors.New("a reply in a thread needs a merge request, on a forge that lists its threads")
			}
			return th.Reply(target.ID, thread, backlog.Inert(body))
		}
		if sticky {
			key, _ := m["sticky"].(string)
			if key == "" {
				return errors.New("a sticky comment needs a key: {body, sticky: key}")
			}
			only, _ := m["update-only"].(bool)
			if r, ok := m["review"]; ok {
				// Its comments on the merge request's lines first: the
				// sticky comment says what became of them (#277).
				var err error
				if body, err = a.lineReview(*target, r, body); err != nil {
					return err
				}
			}
			if read, ok := m["read"].(string); ok {
				// The record the run read, merged with the one the
				// comment holds now: never a run's commits nor judged
				// findings lost to a run applied since (#375).
				var err error
				if body, err = a.mergedRecord(*target, key, body, read); err != nil {
					return err
				}
			}
			marker := forge.Marker("sticky=" + a.role + "/" + key)
			if target.Kind == "merge-request" {
				// workline's own, the comment its record is read back
				// from (ADR-0051); on an issue, the role's by its
				// marker, whatever account wrote it (ADR-0038).
				return forge.KeepOwn(a.forge, *target, backlog.Inert(body), marker, !only)
			}
			return a.forge.Sticky(*target, backlog.Inert(body), marker, !only)
		}
		return a.forge.Comment(*target, backlog.Inert(body), a.marker())
	case "label":
		m, _ := in.Value.(map[string]any)
		if err := a.needForge("label"); err != nil {
			return err
		}
		target := a.target
		if n, ok := m["issue"].(int); ok { // the issue a merge request's role names: the tester's, sending a need back
			target = &forge.Target{Kind: "issue", ID: n}
		}
		if target == nil {
			return errors.New("a label needs a target issue or merge request (--target)")
		}
		return a.forge.Label(*target, strs(m["add"]), strs(m["remove"]))
	case "issue":
		m, _ := in.Value.(map[string]any)
		title, _ := m["title"].(string)
		body, _ := m["body"].(string)
		if title == "" {
			return errors.New("an issue needs a title")
		}
		if sticky, _ := m["sticky"].(bool); sticky {
			// One issue kept in place, its body rewritten on each run; with
			// update-only, never opened to say nothing.
			if err := a.needForge("issue"); err != nil {
				return err
			}
			only, _ := m["update-only"].(bool)
			if _, err := a.forge.KeepIssue(title, body+fmt.Sprintf("\n\nKept up to date by the %s role.", a.role), !only); err != nil {
				return err
			}
			// The one issue workline renamed (ADR-0051): kept under its new
			// title, the old one, still open, says so and is closed. Only
			// this title, known here, never one an answer names.
			if was := renamedIssues[title]; was != "" {
				id, err := a.forge.KeepIssue(was, fmt.Sprintf("Renamed: the %s role keeps this list in the issue %q now.", a.role, title), false)
				if b, ok := a.forge.(forge.Backlog); err == nil && ok && id != 0 {
					err = b.Close(id, 0)
				}
				return err
			}
			return nil
		}
		if err := a.needForge("issue"); err != nil {
			return err
		}
		return a.openIssue(title, body, m)
	case "close":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil // proposed on the issue, or dropped with a finding
		}
		is := forge.Target{Kind: "issue", ID: d.Act.Issue}
		switch {
		case d.Act.Announce:
			return a.announce(d.Act)
		case d.Act.Announced != "":
			// Closed after its announcement (ADR-0024): the label goes, so a
			// reopening reads as the announcement kept open.
			if err := a.forge.Comment(is, backlog.Inert(backlog.ClosingComment(d.Act, a.role)), a.marker()); err != nil {
				return err
			}
			if err := a.forge.(forge.Backlog).Close(d.Act.Issue, 0); err != nil {
				return err
			}
			return a.forge.Label(is, nil, []string{backlog.LabelObsolete})
		}
		if err := a.forge.Comment(is, backlog.Inert(backlog.Comment(d.Act, a.role)), a.marker()); err != nil {
			return err
		}
		return a.forge.(forge.Backlog).Close(d.Act.Issue, d.Act.DuplicateOf)
	case "keep":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return keepOpen(a.forge, a.role, d.Act)
	case "open":
		// An issue opened from a file's text, quoted as the file has it, and
		// given its state: confirmed at this commit, no sources yet.
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		q := *d.Act.Quote
		body, where := strings.TrimSpace(q.Text), "`"+q.Path+"`"
		if from, to, original, ok := backlog.Locate(a.repo, q.Path, q.Text); ok {
			body, where = strings.TrimSpace(original), fmt.Sprintf("`%s`, lines %d to %d", q.Path, from, to)
		}
		o, err := a.openings()
		if err != nil {
			return err
		}
		head, err := git(a.repo, nil, "rev-parse", "--short", "HEAD")
		if err != nil {
			return err
		}
		// Through the one way: a text closed as an issue is not opened again.
		outcome, id, err := o.Open(backlog.Opening{Role: a.role, Key: backlog.ImportKey(q), Title: d.Act.Title, Body: body, From: " from " + where, Commit: head, Quiet: true})
		if err != nil {
			return err
		}
		if outcome == backlog.Settled {
			a.findings = append(a.findings, verdict.Finding{Rule: "issue-closed", Level: "warn", Where: fmt.Sprintf("#%d", id),
				Message: fmt.Sprintf("%q: #%d, closed, holds this text; not opened again", d.Act.Title, id)})
		}
		// The import's map reads which issue holds it from here: a forge's
		// list may not show an issue opened a moment ago.
		return backlog.RecordOpening(a.runDir, a.index, outcome, id)
	case "milestone":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.forge.(forge.Backlog).SetMilestone(d.Act.Issue, strings.TrimSpace(d.Act.Milestone))
	case "order", "effort":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.level(d.Act)
	case "sources":
		// The code the issue is about, named: its state gets them, and the
		// issue is read again, with them, at the next run.
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		is := forge.Target{Kind: "issue", ID: d.Act.Issue}
		comments, err := a.forge.(forge.Backlog).Comments(is)
		if err != nil {
			return err
		}
		st, _, err := backlog.ReadState(comments, a.role)
		if err != nil {
			return fmt.Errorf("#%d: its state comment: %w", is.ID, err)
		}
		st.Sources, st.Judged = d.Act.Sources, ""
		return a.forge.Sticky(is, backlog.FormatState(*st), backlog.StateMarker(a.role), false)
	case "refine", "ready", "ask":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return refining(a.forge, a.role, d.Act)
	case "rename":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return rename(a.forge, a.role, d.Act)
	case "unready":
		// Back to refine, a person having accepted it (ADR-0032): the issue
		// told why, its labels moved.
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		t := forge.Target{Kind: "issue", ID: d.Act.Issue}
		say := fmt.Sprintf("Moved back to refine: %s\n\nBy the %s role, %s having accepted it. Put the label %s back to undo.",
			strings.TrimSpace(d.Act.Why), a.role, d.Act.Ticked, backlog.LabelReady)
		if err := a.forge.Comment(t, backlog.Inert(say), a.marker()); err != nil {
			return err
		}
		return a.forge.Label(t, []string{backlog.LabelToRefine}, []string{backlog.LabelReady})
	case "split":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.split(d.Act)
	case "depend":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.depend(d.Act.Issue, d.Act.BlockedBy, true)
	case "undepend":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.undepend(d.Act)
	case "handoff":
		a.handoffs = append(a.handoffs, in.Value)
		return intent.Write(filepath.Join(a.runDir, "out", "handoffs.yaml"), []intent.Intention{in})
	}
	return fmt.Errorf("not implemented yet in this engine")
}

// allowed reports whether a repository path is within the role's duties.writes.
func (a *applier) allowed(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	if strings.HasPrefix(clean, "../") || filepath.IsAbs(path) {
		return false
	}
	return pathglob.Any(a.writes, clean)
}

// patch applies a unified diff, or replaces one file with {file, content}.
func (a *applier) patch(v any) error {
	if m, ok := v.(map[string]any); ok {
		file, _ := m["file"].(string)
		content, ok := m["content"].(string)
		if file == "" || !ok {
			return errors.New("expected {file, content}")
		}
		if !a.allowed(file) {
			return fmt.Errorf("%s is outside what this role may write", file)
		}
		if link := throughLink(a.repo, file); link != "" {
			return fmt.Errorf("%s is a link: nothing is written through it", link)
		}
		if err := os.WriteFile(filepath.Join(a.repo, file), []byte(content), 0o644); err != nil {
			return err
		}
		a.written = append(a.written, file)
		return nil
	}
	diff, ok := v.(string)
	if !ok {
		return errors.New("expected a unified diff or {file, content}")
	}
	diff = intent.NormalizeDiff(diff)
	if why := notText(diff); why != "" {
		return errors.New(why)
	}
	// -z: each path whole, as git reads it, a space or a quote in it too.
	files, err := git(a.repo, strings.NewReader(diff), "apply", "--recount", "--unidiff-zero", "--numstat", "-z", "-")
	if err != nil {
		return fmt.Errorf("unreadable diff: %w", err)
	}
	var touched []string
	for _, rec := range strings.Split(files, "\x00") {
		if rec == "" {
			continue
		}
		f := strings.SplitN(rec, "\t", 3)
		if len(f) != 3 || f[2] == "" {
			return fmt.Errorf("unreadable diff: git names no file in %q", rec)
		}
		if !a.allowed(f[2]) {
			return fmt.Errorf("%s is outside what this role may write", f[2])
		}
		if link := throughLink(a.repo, f[2]); link != "" {
			return fmt.Errorf("%s is a link: nothing is written through it", link)
		}
		touched = append(touched, f[2])
	}
	// --recount: a diff is applied as its lines read. Agents often get the
	// counts of a hunk header wrong, and git would drop the lines past them.
	// --unidiff-zero: a role's own diff may hold no context (intent.Merge).
	if _, err := git(a.repo, strings.NewReader(diff), "apply", "--recount", "--unidiff-zero", "-"); err != nil {
		return err
	}
	a.written = append(a.written, touched...)
	return nil
}

// notText says why a diff does more than change text files: a rename or a
// copy — whose source git's count does not name —, a mode changed, a link,
// a submodule or an executable made, a binary patch; "" when it does not.
// Read from git's own extended headers, which a hunk's lines never start.
func notText(diff string) string {
	for _, l := range strings.Split(diff, "\n") {
		if l == "" || strings.ContainsRune(" +-@\\", rune(l[0])) {
			continue
		}
		switch {
		case strings.HasPrefix(l, "rename ") || strings.HasPrefix(l, "copy ") || strings.HasPrefix(l, "similarity index ") ||
			strings.HasPrefix(l, "dissimilarity index "):
			return "the patch renames or copies a file (" + l + "): a role changes text, in place"
		case strings.HasPrefix(l, "old mode ") || strings.HasPrefix(l, "new mode "):
			return "the patch changes a file's mode (" + l + "): a role changes text"
		case strings.HasPrefix(l, "new file mode ") || strings.HasPrefix(l, "deleted file mode "):
			if mode := l[strings.LastIndex(l, " ")+1:]; mode != "100644" {
				return "the patch makes or deletes a file of mode " + mode + ": a role writes text files"
			}
		case strings.HasPrefix(l, "index "):
			if f := strings.Fields(l); len(f) > 2 && f[2] != "100644" {
				return "the patch names a file of mode " + f[2] + ": a role writes text files"
			}
		case strings.HasPrefix(l, "GIT binary patch") || strings.HasPrefix(l, "Binary files "):
			return "the patch is binary: a role writes text files"
		}
	}
	return ""
}

func strs(v any) []string {
	list, _ := v.([]any)
	var out []string
	for _, x := range list {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func git(repo string, stdin io.Reader, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.quotePath=off"}, args...)...)
	cmd.Stdin = stdin
	var out, errOut strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		said := strings.TrimSpace(errOut.String())
		if said == "" { // git commit says "nothing to commit" on its output
			said = strings.TrimSpace(out.String())
		}
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, said)
	}
	return strings.TrimSpace(out.String()), nil
}

// keepRuns is how many past runs are kept for inspection.
const keepRuns = 50

// newRunDir creates the run's folder inside the repository's git directory,
// where it is never tracked and never shows up as a change.
func newRunDir(repo, roleName string) (string, error) {
	base := RunsDir(repo)
	id := fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102T150405.000000000"), roleName)
	dir := filepath.Join(base, id)
	for _, d := range []string{"in/input", "out"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return "", err
		}
	}
	if old, err := os.ReadDir(base); err == nil && len(old) > keepRuns {
		for _, e := range old[:len(old)-keepRuns] { // names start with a timestamp, so the oldest come first
			os.RemoveAll(filepath.Join(base, e.Name()))
		}
	}
	return dir, nil
}

// dropWorktree removes the worktree a role's pre made in the run for its
// agent, kept through every attempt (ADR-0056); none, nothing.
func dropWorktree(repo, runDir string) {
	if tree := agent.Worktree(runDir); tree != "" {
		tester.RemoveWorktree(repo, tree)
	}
}

// RunsDir is where repo's runs are kept: WORKLINE_RUNS_DIR when set — in
// CI, somewhere an artifact can carry them to the job that applies —, else
// its git folder's workline/runs.
func RunsDir(repo string) string {
	if d := os.Getenv("WORKLINE_RUNS_DIR"); d != "" {
		return d
	}
	if out, err := exec.Command("git", "-C", repo, "rev-parse", "--absolute-git-dir").Output(); err == nil {
		return filepath.Join(strings.TrimSpace(string(out)), "workline", "runs")
	}
	return filepath.Join(repo, ".workline", "runs")
}

// personalSettings are the role's settings with what the person's own
// config sets for it laid over, as far as the role lets a person set it
// (role.Role.WithPersonal); what it may not set, and a config that cannot
// be read, are said, and change nothing.
func personalSettings(r *role.Role, merged map[string]any, res *Result) map[string]any {
	personal, err := role.PersonalSettings(r.Name)
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "personal-config-invalid", Level: "warn", Where: role.PersonalConfig(),
			Message: "your own config could not be read (" + err.Error() + "): the project's settings stand"})
		return merged
	}
	out, refused := r.WithPersonal(merged, personal)
	for _, x := range refused {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "personal-setting-ignored", Level: "warn", Where: role.PersonalConfig(),
			Message: fmt.Sprintf("your own config sets roles.%s.settings.%s, left out: %s; the project's stands", r.Name, x.Key, x.Why)})
	}
	return out
}

// intSetting reads a whole number from the role's settings; 0 when unset.
func intSetting(settings map[string]any, key string) int {
	switch v := settings[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func writeInputs(runDir string, inputs map[string]string, settings map[string]any) error {
	for k, v := range inputs {
		if err := os.WriteFile(filepath.Join(runDir, "in", "input", k), []byte(v), 0o644); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "in", "settings.json"), data, 0o644)
}

// releaseRequest tells whether the merge request a run targets is a release
// tool's, by its branch (ADR-0017). The role is told the branch, which the
// reviewer reads to leave the versions the tool bumps (#384). It returns
// the branch, "" when it is not a release's or cannot be learnt.
func releaseRequest(o Options) string {
	if o.Event != "merge-request" {
		return ""
	}
	rs, err := release.Load(o.Repo)
	if err != nil {
		return ""
	}
	branch := o.Branch
	if branch == "" && o.Target != nil && o.Target.Kind == "merge-request" {
		f, err := forge.Open(o.Forge, o.Repo)
		if err != nil || f == nil {
			return ""
		}
		mr, err := f.MergeRequest(o.Target.ID)
		if err != nil {
			return ""
		}
		branch = mr.Branch
	}
	if !rs.IsBranch(branch) {
		return ""
	}
	return branch
}

func scriptEnv(runDir, roleName string, o Options) []string {
	ai := o.AI
	if ai == "" {
		ai = "none"
	}
	self, _ := os.Executable()
	env := append(os.Environ(),
		"WORKLINE_RUN_DIR="+runDir,
		"WORKLINE_EVENT="+o.Event,
		"WORKLINE_AI="+ai,
		"WORKLINE_ROLE="+roleName,
		"WORKLINE_BIN="+self,
		"WORKLINE_ROLES_DIR="+o.RolesDir,
	)
	if o.aiWaits != "" { // the agent set aside, a draft (DraftInput)
		env = append(env, "WORKLINE_AI_WAITS="+o.aiWaits)
	}
	if o.Forge != "" && o.Forge != "none" {
		env = append(env, "WORKLINE_FORGE="+o.Forge) // what the role may write to
	}
	if o.Target != nil && o.Forge != "" && o.Forge != "none" { // where a comment would go
		env = append(env, fmt.Sprintf("WORKLINE_TARGET=%s:%d", o.Target.Kind, o.Target.ID))
	}
	return env
}

// script runs roles/<name>/<step> in the repository and returns its exit code.
func script(r *role.Role, step, repo string, env []string) (int, error) {
	path := filepath.Join(r.Dir, step)
	if _, err := os.Stat(path); err != nil {
		return 0, fmt.Errorf("role %q has no %s script", r.Name, step)
	}
	cmd := exec.Command(path)
	cmd.Dir, cmd.Env = repo, env
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return 0, fmt.Errorf("%s: %w", step, err)
	}
	return 0, nil
}

// dirDigest hashes every file under dir, names and contents, in a stable order.
func dirDigest(dir string) (string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return err
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		rel, _ := filepath.Rel(dir, f)
		fmt.Fprintf(h, "%s\x00%d\x00", rel, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// askJudge puts the question a role's judge step could not answer
// (out/judge.yaml: {question, material}) to a judge, at the best independence
// available from the agent that answered (ADR-0005): WORKLINE_JUDGE when set,
// else another Claude model, else the same agent in a context of its own. A
// "no" refuses the proposal with the judge's reason, so the agent is asked
// again; a judge that cannot answer blocks, since nothing unjudged is applied.
func askJudge(o Options, res *Result, runDir string, v *verdict.Verdict, a *attemptResult) error {
	data, err := os.ReadFile(filepath.Join(runDir, "out", "judge.yaml"))
	if err != nil {
		return nil // nothing the checks could not answer
	}
	var q struct{ Question, Material string }
	if err := yaml.Unmarshal(data, &q); err != nil || q.Question == "" {
		return fmt.Errorf("out/judge.yaml: a question is needed: %v", err)
	}
	var authorModels []string
	for _, c := range res.Calls {
		if c.Task != "judge" { // an earlier round's judge is not an author
			authorModels = append(authorModels, c.Model)
		}
	}
	last := ""
	if len(authorModels) > 0 {
		last = authorModels[len(authorModels)-1]
	}
	spec := os.Getenv("WORKLINE_JUDGE")
	if spec == "" {
		spec = judge.Pick(o.AI, last)
	}
	judgeRole, err := role.Load(o.RolesDir, "judge")
	if err != nil {
		return err
	}
	bounded, ok := res.bounded(judgeRole)
	// Anything but a yes blocks for the answer.
	if !ok {
		a.external = true
		v.Final = false
		v.Status, v.Summary = verdict.Block, "the judge was not asked"
		v.Findings = append(v.Findings, verdict.Finding{Rule: "judge-unavailable", Message: "not asked: " + errTimeSpent.Error()})
		return nil
	}
	ans, err := judge.Ask(spec, bounded, q.Question, q.Material)
	err = res.cutShort(err)
	res.Calls = append(res.Calls, ans.Call) // its tokens count too; it is not the role's agent
	level := judge.Independence(spec, ans.Model, o.AI, authorModels)
	who := fmt.Sprintf("independence: %s (%s → %s)", level, last, ans.Model)
	switch {
	case err != nil:
		a.external = true
		v.Final = false
		v.Status, v.Summary = verdict.Block, "the judge could not answer"
		v.Findings = append(v.Findings, verdict.Finding{Rule: "judge-unavailable", Message: err.Error() + "; " + who})
	case !ans.Yes:
		v.Final = false
		v.Status, v.Summary = verdict.Block, "the judge refused the proposal"
		v.Findings = append(v.Findings, verdict.Finding{Rule: "judged-no", Message: ans.Why + " (" + who + ")"})
	default:
		v.Findings = append(v.Findings, verdict.Finding{Rule: "judged", Level: "warn", Message: "yes: " + ans.Why + " (" + who + ")"})
	}
	return nil
}

// maxJudgeRounds bounds how often pre may ask its judges again: a role
// bounds its own rounds (the reviewer's judge-read-rounds); past it, a
// question pre asks again is answered as not asked, and pre settles it.
const maxJudgeRounds = 10

// notAsked answers questions with why they were not asked.
func notAsked(questions []string, why string) error {
	out, _ := yaml.Marshal(map[string]string{"error": why})
	for _, q := range questions {
		if err := os.WriteFile(filepath.Join(filepath.Dir(q), "answer.yaml"), out, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// judgeAnswered are the judge's questions holding an answer.
func judgeAnswered(runDir string) []string {
	answers, _ := filepath.Glob(filepath.Join(runDir, "in", "judge", "*", "answer.yaml"))
	for i, a := range answers {
		answers[i] = filepath.Join(filepath.Dir(a), "question.yaml")
	}
	return answers
}

// askedAgain are the questions pre asks again (ADR-0045): answered before
// it ran, it took their answer away, and wrote what the judge read.
func askedAgain(runDir string, answered []string) []string {
	var again []string
	for _, q := range answered {
		if _, err := os.Stat(filepath.Join(filepath.Dir(q), "answer.yaml")); os.IsNotExist(err) {
			if _, err := os.Stat(q); err == nil {
				again = append(again, q)
			}
		}
	}
	return again
}

// askQuestions puts each question pre wrote (in/judge/<key>/question.yaml:
// {question, material}) to a judge, in a context of its own, at the best
// independence available from the agents that answered the run (ADR-0005),
// and writes its answer beside it (answer.yaml: {yes, why, model, judge,
// level}, or {error} when it could not answer). A question saying `reads:
// true` lets its judge ask to read instead (ADR-0045): its answer then holds
// its `reads`, or `unsure`, and a no its `quote`. Without an agent, or once
// the run spent its ai-max-tokens, none is asked; the agent unreachable, the
// rest are not asked either. Unlike a post's question, a no refuses nothing:
// pre reads each answer, a finding at a time.
func askQuestions(o Options, ag agent.Agent, runDir string, questions []string, res *Result) error {
	if ag == nil {
		return nil
	}
	sort.Strings(questions)
	var authorModels []string
	for _, c := range res.Calls {
		if c.Task != "judge" && c.Model != "" {
			authorModels = append(authorModels, c.Model)
		}
	}
	last := ""
	if len(authorModels) > 0 {
		last = authorModels[len(authorModels)-1]
	}
	spec := os.Getenv("WORKLINE_JUDGE")
	if spec == "" {
		spec = judge.Pick(o.AI, last)
	}
	judgeRole, err := role.Load(o.RolesDir, "judge")
	if err != nil {
		return err
	}
	gone := ""
	for _, file := range questions {
		answer := map[string]any{}
		var q struct {
			Question, Material, Author string
			Reads                      bool // its judge may ask to read first (ADR-0045)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, &q); err != nil || q.Question == "" {
			return fmt.Errorf("%s: a question is needed: %v", file, err)
		}
		// What an earlier run's agent proposed names its model (author):
		// the judge stands apart from that one (an announced issue, ADR-0024).
		spec, last, authorModels := spec, last, authorModels
		if q.Author != "" {
			last, authorModels = q.Author, []string{q.Author}
			if os.Getenv("WORKLINE_JUDGE") == "" {
				spec = judge.Pick(o.AI, last)
			}
		}
		switch {
		case gone != "":
			answer["error"] = "not asked: " + gone
		case res.overBudget():
			res.capped = true
			answer["error"] = "not asked: " + errTokensSpent.Error()
		default:
			bounded, ok := res.bounded(judgeRole)
			if !ok {
				answer["error"] = "not asked: " + errTimeSpent.Error()
				break
			}
			ask := judge.Ask
			if q.Reads {
				ask = judge.AskReading
			}
			ans, err := ask(spec, bounded, q.Question, q.Material)
			err = res.cutShort(err)
			ans.Call.For = "judge/" + filepath.Base(filepath.Dir(file))
			res.Calls = append(res.Calls, ans.Call) // its tokens count too; it is not the role's agent
			res.AgentCalls++
			if data, err := json.Marshal(ans.Call); err == nil { // kept with the run's calls, what it cost
				if f, err := os.OpenFile(filepath.Join(runDir, "out", "calls.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
					f.Write(append(data, '\n'))
					f.Close()
				}
			}
			if err != nil {
				answer["error"] = err.Error()
				if errors.Is(err, agent.ErrUnavailable) {
					gone = err.Error()
				}
				break
			}
			answer = map[string]any{"why": ans.Why, "model": ans.Model, "judge": spec,
				"author": last, "level": judge.Independence(spec, ans.Model, o.AI, authorModels)}
			switch { // a judge that may read: its reads, or unsure, in place of yes or no
			case len(ans.Reads) > 0:
				answer["reads"] = ans.Reads
			case ans.Unsure:
				answer["unsure"] = true
			default:
				answer["yes"] = ans.Yes
			}
			if ans.Quote != nil {
				answer["quote"] = ans.Quote
			}
		}
		out, err := yaml.Marshal(answer)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(file), "answer.yaml"), out, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// overBudgetSummary is the summary of a gate's run whose task was over the
// role's context budget: what post said follows, when it said anything.
func overBudgetSummary(said string) string {
	s := "the task was over the role's context budget and nobody judged it: a person does"
	if said != "" {
		s += " (" + said + ")"
	}
	return s
}
