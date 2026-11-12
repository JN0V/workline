// Package engine runs one role once: check, prepare, propose, judge, apply
// (docs/spec/role-contract.md, "One run").
package engine

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
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
	// Command names the command run, for the message of a missing forge
	// (default: the option --open-merge-request).
	Command string
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

// Inputs the line gives the product owner it wakes on a blocked issue
// (ADR-0059, amended): what is new on it, and the state it was blocked
// from ("" unknown); with SizeInput, the role whose run was too big for
// it, the product owner asked on the issue's size (ADR-0064, step 3).
const (
	RevisitInput     = "revisit"
	RevisitFromInput = "revisit-from"
	SizeInput        = "size"
)

// SizePredicted is SizeInput's value on a bug the product owner predicted
// past its normal at refine and readied for its tests first (#1212): no
// run crossed it; its tests written, its split is asked before the
// developer takes it.
const SizePredicted = "prediction"

// RevisitUnrefinedInput, `true`, tells the product owner woken on a
// blocked issue that the line blocked it from refine on that role's own
// read, left lacking (#951, #980): its text rewritten, the cause is
// gone, and the issue goes back to refine in the same run (#1206). Never
// on a block a person or another role made, nor one on a run's size.
const RevisitUnrefinedInput = "revisit-unrefined"

// SizeTierInput is the input the line gives every run on an issue with
// a size tier (ADR-0064, §3): large or x-large. Every size target of the
// run — its token cap, its context budget, the code and change given at
// once (role.TieredSettings) — is multiplied by the tier's factor.
const SizeTierInput = "size-tier"

// FailedInput is the input the line gives a step it runs again after a
// machine error: what failed, said at the end of the agent's task
// (ADR-0063, §6).
const FailedInput = "failed"

// NeedsInput is the input the line gives a role it runs again at once
// for the code its agent said it lacked: the paths, comma-separated
// (ADR-0063, §6).
const NeedsInput = "needs"

// withSection ends the task at file, when pre wrote one, with a section.
func withSection(file, title, text string) error {
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(file, []byte(strings.TrimRight(string(data), "\n")+"\n\n## "+title+"\n\n"+text+"\n"), 0o644)
}

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
	AIWaits string `json:"ai-waits,omitempty"`
	// NoVerdict: the run ended with no verdict read from its post
	// (ErrNoVerdict): nothing it did was applied, its work lost (#953).
	NoVerdict bool `json:"no-verdict,omitempty"`
	maxTokens int  // the role's ai-max-tokens: what the run may spend; 0, no cap
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
	askedAgain  bool     // the attempt asks again, with what the first answer needs mended
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
// the fresh tokens, cache reads apart (agent.Call.Fresh).
func (res *Result) tokensSpent() int {
	n := 0
	for _, c := range res.Calls {
		n += c.Fresh()
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

// ErrNoVerdict is a run whose post left no verdict to read: it died, or
// its run folder went (#953). Nothing the run did is applied.
var ErrNoVerdict = errors.New("post wrote no readable verdict")

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
			res.NoVerdict = res.NoVerdict || errors.Is(err, ErrNoVerdict)
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
	if tier := o.Inputs[SizeTierInput]; backlog.TierFactor(tier) > 1 {
		// Its issue's tier (ADR-0064, §3): every size target multiplied,
		// said, never silent.
		n := backlog.TierFactor(tier)
		settings, r.Context.Budget = role.Tiered(settings, n), r.Context.Budget*n
		res.Findings = append(res.Findings, verdict.Finding{Rule: "size-tier", Level: "info",
			Message: fmt.Sprintf("its issue's size tier is %s (`%s`): every size target of this run ×%d — context.budget, %s", tier, backlog.TierLabel(tier), n, strings.Join(role.TieredSettings, ", "))})
	}
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
			return errors.New(cmp.Or(o.Command, "--open-merge-request") + " needs a forge: " + forge.Missing)
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
	// A run again after a machine error carries what failed (ADR-0063,
	// §6): the agent reads it, and its input is not the one that failed.
	if failed := strings.TrimSpace(o.Inputs[FailedInput]); failed != "" {
		if err := withSection(filepath.Join(runDir, "in", "task.md"), "Your last run on this failed", failed+
			"\n\nThe engine could not use that run: answer again, whole, with this fixed."); err != nil {
			return err
		}
	}
	// Run again for the code its agent said it lacked: told what it is
	// given now (ADR-0063, §6).
	if needs := strings.TrimSpace(o.Inputs[NeedsInput]); needs != "" {
		if err := withSection(filepath.Join(runDir, "in", "task.md"), "The code you said you lacked", "`"+strings.ReplaceAll(needs, ", ", "`, `")+
			"`: given in this task, as far as it fits. Answer with it; name a file only if it is still missing."); err != nil {
			return err
		}
	}
	// What pre cut of the task past a cap is said to the person, not to
	// the agent alone (#1159).
	res.Findings = append(res.Findings, verdict.CutFindings(runDir, res.Findings)...)
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
	// This task's needs, instruction and answers in place of the role's.
	r = r.ForTask(taskKind(runDir))
	tier, failures := r.Model.Tier, 0
	var intents, said []intent.Intention
	var v *verdict.Verdict
	askedOnce := false // asked again once already, an answer unread or one post said to mend
	unasked := false   // a task the agent was not asked, nor answered: what pre gave it read by no one
	for {
		// A size counts on the first call alone (#1160): one asked again
		// carries what its first answer needs mended, an unread answer's
		// or a refused one's (`promote-after`).
		res.askedAgain = askedOnce || failures > 0
		a, err := attempt(r, o, ag, taskErr == nil, tier, runDir, env, line, settings, cfg, res)
		if err != nil {
			return err
		}
		unasked = taskErr == nil && (!a.askedAgent || a.external || a.unread != "")
		intents, v, said = a.intents, a.verdict, a.said
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
		// A role that never asks a refused answer again (`promote-after`
		// 0) still asks an unread one again, whatever post said: its block
		// comes of nothing read, as the tester's no test written (#761).
		if a.askedAgent && a.unread != "" && !askedOnce && !v.Final && (v.Status != verdict.Block || r.Model.PromoteAfter <= 0) {
			askedOnce = true
			if err := os.WriteFile(filepath.Join(runDir, "out", "feedback.md"), []byte(unreadFeedback(a.unread)), 0o644); err != nil {
				return err
			}
			os.Rename(filepath.Join(runDir, "out", "agent-answer.txt"), filepath.Join(runDir, "out", "unread-answer.txt"))
			os.Remove(filepath.Join(runDir, "out", "verdict.yaml"))
			os.Remove(filepath.Join(runDir, "out", "judge.yaml"))
			continue
		}
		// A block post says the answer can mend (`again`: the tester's
		// tests that do not lint, #964) is asked again the same way, once,
		// with its findings, by a role that asks no refused answer again:
		// the one more call an unread answer gets, not a second one.
		if a.askedAgent && v.Status == verdict.Block && v.Again && !askedOnce && !v.Final && r.Model.PromoteAfter <= 0 {
			askedOnce = true
			fb := feedback(v.Findings) + "\nYour answer was read, and refused for these findings alone: send it again whole, each of them fixed.\n"
			if err := os.WriteFile(filepath.Join(runDir, "out", "feedback.md"), []byte(fb), 0o644); err != nil {
				return err
			}
			// Kept to be studied when it can be; what post and the judge
			// said of it goes, written anew by the next answer's.
			_ = os.Rename(filepath.Join(runDir, "out", "intentions.yaml"), filepath.Join(runDir, "out", "refused-1.yaml"))
			_ = os.Rename(filepath.Join(runDir, "out", "agent-answer.txt"), filepath.Join(runDir, "out", "refused-1-answer.txt"))
			_ = os.Remove(filepath.Join(runDir, "out", "verdict.yaml"))
			_ = os.Remove(filepath.Join(runDir, "out", "judge.yaml"))
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
		if err := os.WriteFile(filepath.Join(runDir, "out", "feedback.md"), []byte(feedback(v.Findings)), 0o644); err != nil {
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
		warned := make([]verdict.Finding, len(v.Findings))
		for i, f := range v.Findings {
			if f.Level == "" {
				f.Level = "warn"
			}
			warned[i] = f
		}
		next := "asked again at the next run"
		if warnsOnMergeRequest(r, o) {
			next = "the weekly docs pull request judges it again"
		}
		v = &verdict.Verdict{Status: verdict.Pass, Findings: warned,
			Summary: "the agent's proposal was refused, nothing of it applied; " + next + " (" + v.Summary + ")"}
		intents = said // what post said of its refusal, on its target still
	}
	res.Status, res.Summary = v.Status, v.Summary
	res.Findings = append(res.Findings, v.Findings...)
	// A role keeping a backlog reads its report on every run, acts or not:
	// a closing undone, a box ticked, runs nobody answered (ADR-0025).
	keeps := slices.ContainsFunc(r.Intentions, func(k string) bool { return slices.Contains(backlog.Kinds, k) })
	// Blocked, a run applies only what its role names to tell why
	// (`on-block`): the reviewer's comment, its record in it (#226). The
	// run still blocks. A proposal pre made itself marked `on-block: true`
	// tells why too: a refusal written on its issue (ADR-0059, decision 1);
	// so does what post left in out/on-block.yaml, blocking.
	fallback, _ := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	tells := func(i intent.Intention) bool {
		m, _ := i.Value.(map[string]any)
		b, _ := m["on-block"].(bool)
		return b && isFallbackOf(i, fallback)
	}
	onBlock := res.Status == verdict.Block && (len(r.OnBlock) > 0 || len(said) > 0 || slices.ContainsFunc(intents, tells))
	if onBlock {
		intents = slices.DeleteFunc(intents, func(i intent.Intention) bool { return !slices.Contains(r.OnBlock, i.Kind) && !tells(i) })
		intents = append(intents, said...)
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
	// A covers entry tells post which test covers which scenario: read
	// there, never applied (ADR-0065).
	intents = slices.DeleteFunc(intents, func(i intent.Intention) bool { return i.Kind == "claim" || i.Kind == "skip" || i.Kind == "covers" })
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

// feedback is what the agent is told of a refusal: each finding, a line,
// where it is when its message does not say it.
func feedback(fs []verdict.Finding) string {
	var fb strings.Builder
	for _, f := range fs {
		if f.Where != "" && !strings.Contains(f.Message, f.Where) {
			fmt.Fprintf(&fb, "- %s, %s: %s\n", f.Rule, f.Where, f.Message)
		} else {
			fmt.Fprintf(&fb, "- %s: %s\n", f.Rule, f.Message)
		}
	}
	return fb.String()
}

// unreadFeedback is what the agent is told when nothing of its answer read.
func unreadFeedback(why string) string {
	return "- agent-invalid-output: " + why + "\n\nNothing of that answer could be read: send it again as valid YAML. " +
		agent.BlockScalars + "\n"
}

// itemStart is a line that starts a real item (see answerItems): "- " and
// one of the keys an item starts with, bare or quoted, or a flow mapping
// opening with one. Any other key ("- name: x") is the claim's own text.
var itemStart = regexp.MustCompile(`^-[ \t]+(\{[ \t]*)?("(claim|agent-invalid-output)"|'(claim|agent-invalid-output)'|(claim|agent-invalid-output))[ \t]*:([ \t\r]|$)`)

// answerItems cuts an answer into its top-level list items: each starts
// with "- " and the key "claim" or "agent-invalid-output" ("- claim:",
// "-  claim :", "- 'claim':") or a flow mapping ("- {claim: x}") at the start
// of a line, and runs to the next, or to a fence. Every other line in
// between is the item's, one at the start of the line too (code pasted as
// it is, a list line "- foo" or "- name: x" included), so an item never reads from text
// cut short.
func answerItems(answer string) []string {
	var items []string
	open := false
	lines := strings.Split(answer, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1] // the answer's final newline, not a line
	}
	for _, l := range lines {
		switch {
		case itemStart.MatchString(l):
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
	limit, _ := req.Role.Model.AnswerTimeout()
	phase(req.Role.Name, "asks its agent, "+limit.String()+" at most")
	start := time.Now()
	call, err := ag.Propose(req)
	if errors.Is(err, agent.ErrOverBudget) {
		return err // refused before the call: nothing asked, nothing spent
	}
	err = res.cutShort(err)
	call.TimeCut = errors.Is(err, errTimeSpent)
	// Every failed call is recorded, for the audit; a call asked again
	// that answers supersedes it, as the developer reads only the last.
	if err != nil && !call.TimeCut {
		call.Failed = err.Error()
	}
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
	// Committed: the tip of the role's branch this run committed, its push
	// failed; a resume pushes it as it is, nothing committed again (#739).
	Committed string `yaml:"committed,omitempty"`
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
	// A role handing its work on an issue over in a merge request, with
	// none to be opened: its patches would land on the person's branch,
	// and what it says on the issue would name a branch that is not there
	// (#1280). Nothing applied, said.
	if task := proposedTask(runDir); f != nil && st.Target != nil && st.Target.Kind == "issue" && !st.OpenMergeRequest && task != "" {
		res.Status, res.Summary = verdict.Block, fmt.Sprintf("nothing applied: %s hands its work over in a merge request, and this run opens none", r.Name)
		res.Findings = unmoved(res.Findings)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-request-not-opened", Where: st.Target.String(),
			Message: fmt.Sprintf("the %s proposes %s%s; run again with --open-merge-request: the issue keeps its state, the working tree is left as it was", r.Name, branchPrefix(r.Name), task)})
		return nil
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
	commenting := false // the patches, written apart, go in a comment
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
			if commenting && in.Kind == "patch" && errors.Is(err, errNotApplied) {
				// Made on a checkout with changes not committed, it does
				// not apply on its head: said, the rest still applied.
				res.Findings = append(res.Findings, verdict.Finding{Rule: "patch-not-applied", Level: "warn",
					Message: fmt.Sprintf("a patch to comment does not apply on the head judged, apart from the checkout's uncommitted changes: %v", err)})
				return false, nil
			}
			return false, fmt.Errorf("apply %s: %w", in.Kind, err)
		}
		if !slices.Contains(st.Applied, i) {
			st.Applied = append(st.Applied, i)
		}
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
	onTarget := f != nil && st.Target != nil && st.Target.Kind == "merge-request" && !st.OpenMergeRequest
	ownBranch := ownMessage != "" && onTarget
	// Its own Draft marked ready, asked by the role (the developer's,
	// green, #969): after its push, before its comments say it.
	ready := ownReady(runDir) && onTarget
	// Patches that go in a comment (ADR-0051) are written in a worktree
	// of the head apart, never in the checkout, which a person may be
	// working in (#1328); a resumed run writes them there again.
	commentRepo := st.Repo
	if onTarget && !ownBranch && !st.Pushed && slices.ContainsFunc(intents, func(in intent.Intention) bool { return in.Kind == "patch" }) {
		tree, err := os.MkdirTemp("", "workline-patches-")
		if err == nil {
			_ = os.Remove(tree) // git makes it
			err = tester.Worktree(st.Repo, "HEAD", tree)
		}
		if err != nil {
			return fmt.Errorf("a worktree for the patches to comment: %w", err)
		}
		defer tester.RemoveWorktree(st.Repo, tree)
		ap.tree, commentRepo, commenting = tree, tree, true
		for i, in := range intents {
			if in.Kind == "patch" {
				delete(done, i)
			}
		}
	}
	// So too the files a role proposes in the merge request it opens:
	// written, then committed on its branch, in a worktree of HEAD apart;
	// the checkout never leaves its branch, nor holds them (#1372). A
	// resumed run whose commits are its branch's tip still only pushes
	// them: nothing written again on a HEAD that may have moved since.
	pushOnly := false
	if st.Committed != "" {
		tip, _ := git(st.Repo, nil, "rev-parse", "--verify", "-q", "refs/heads/"+branchPrefix(st.Role)+slugify(mergeRequestKey(st, runDir)))
		pushOnly = tip == st.Committed
	}
	if st.OpenMergeRequest && st.MergeRequest == 0 && !pushOnly && slices.ContainsFunc(intents, func(in intent.Intention) bool { return in.Kind == "patch" }) {
		tree, err := os.MkdirTemp("", "workline-propose-")
		if err == nil {
			_ = os.Remove(tree) // git makes it
			// HEAD as it is, never a commit its agent put in its place
			// in the git store its worktree shares (ADR-0056).
			if _, err = git(st.Repo, nil, "--no-replace-objects", "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", tree, "HEAD"); err != nil {
				tester.RemoveWorktree(st.Repo, tree)
			}
		}
		if err != nil {
			return fmt.Errorf("a worktree for the files to propose: %w", err)
		}
		defer tester.RemoveWorktree(st.Repo, tree)
		ap.tree = tree
		for i, in := range intents {
			if in.Kind == "patch" {
				delete(done, i)
			}
		}
	}
	var order, after []int
	for i, in := range intents {
		if (ownBranch || ready) && in.Kind != "patch" {
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
	merge := ownMerge(runDir)
	if ownBranch && (len(st.Written) > 0 || merge != nil) && !st.Pushed {
		if err := commitOnOwnBranch(f, st, ownMessage, merge); err != nil {
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
	if ready {
		if err := markOwnReady(f, st); err != nil {
			if errors.Is(err, forge.ErrUnreachable) {
				res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while marking the role's merge request ready; resume with: workline apply %s", runDir)
				res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
				return nil
			}
			res.Status, res.Summary = verdict.Block, "the role's merge request not marked ready: nothing else is said"
			res.Findings = append(res.Findings, verdict.Finding{Rule: "not-ready", Message: err.Error()})
			return nil
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
		id, tip, err := openMergeRequest(f, st, runDir, ap.tree)
		if tip != "" && tip == st.Committed {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "already-committed", Level: "info", Where: branchPrefix(st.Role) + slugify(mergeRequestKey(st, runDir)),
				Message: "nothing new to commit: the run before committed this code here, then stopped; pushed as it is"})
		}
		if err != nil && tip != "" && tip != st.Committed {
			// Committed, whatever failed after: its resume pushes these
			// commits, the code no longer in the working tree.
			st.Committed = tip
			if err := st.save(runDir); err != nil {
				return err
			}
		}
		if errors.Is(err, forge.ErrUnreachable) {
			res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while opening the merge request; resume with: workline apply %s", runDir)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
			return nil
		}
		switch {
		case errors.Is(err, ErrLeftToAPerson):
			// Theirs now: this run's change, written apart, stays in its
			// run's answer, said.
			res.Findings = append(res.Findings, verdict.Finding{Rule: "not-proposed", Level: "warn", Where: branchPrefix(st.Role) + slugify(mergeRequestKey(st, runDir)),
				Message: strings.TrimPrefix(err.Error(), ErrLeftToAPerson.Error()+": ") + ": this run's change is not pushed over it, kept in " + filepath.Join(runDir, "out", "intentions.yaml") + "; the branch and its merge request are left to them"})
		case err != nil:
			// What the role asked done when it does not open (#856): the
			// developer's claim given back, the reason on its issue.
			undo, uerr := unopened(runDir)
			if uerr != nil {
				return fmt.Errorf("open a merge request: %w; and %w", err, uerr)
			}
			if len(undo) == 0 {
				return fmt.Errorf("open a merge request: %w", err)
			}
			for _, in := range undo {
				if aerr := ap.apply(sayWhy(in, err)); aerr != nil {
					return fmt.Errorf("open a merge request: %w; then, %s: %w", err, in.Kind, aerr)
				}
			}
			res.Status, res.Summary = verdict.Block, "its merge request did not open: said where the role asked"
			res.Findings = unmoved(res.Findings)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "not-opened", Where: branchPrefix(st.Role) + slugify(mergeRequestKey(st, runDir)),
				Message: err.Error() + "; then applied what the role asks for that case: " + strings.Join(intent.Kinds(undo), ", ")})
			return nil
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
		at := st
		at.Repo = commentRepo // the worktree the patches were written in
		finding, err := pushToMergeRequest(f, at, runDir)
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

// unopened are the intentions a role asks applied when the merge request
// it opens does not open, for a reason other than a forge out of reach or
// a branch left to a person: out/merge-request.yaml's `unopened:`, labels
// and comments alone (the developer's claim given back, #856).
func unopened(runDir string) ([]intent.Intention, error) {
	var mr struct {
		Unopened []map[string]any `yaml:"unopened"`
	}
	data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml"))
	if err != nil {
		return nil, nil
	}
	if err := yaml.Unmarshal(data, &mr); err != nil {
		return nil, fmt.Errorf("out/merge-request.yaml: %w", err)
	}
	var out []intent.Intention
	for i, m := range mr.Unopened {
		if len(m) != 1 {
			return nil, fmt.Errorf("out/merge-request.yaml: unopened entry %d must have exactly one kind", i+1)
		}
		for k, v := range m {
			if k != "label" && k != "comment" {
				return nil, fmt.Errorf("out/merge-request.yaml: unopened entry %d is a %s: a label or a comment only", i+1, k)
			}
			out = append(out, intent.Intention{Kind: k, Value: v})
		}
	}
	return out, nil
}

// unmoved drops the role's word that its issue moved state: with its
// merge request not opened, the move is not made, or undone (#1280).
func unmoved(findings []verdict.Finding) []verdict.Finding {
	return slices.DeleteFunc(findings, func(f verdict.Finding) bool { return f.Rule == "state-moved" })
}

// sayWhy puts what failed in a comment's body, in place of intent.WhyMark,
// inert: a git error's text names no one and opens no section.
func sayWhy(in intent.Intention, failed error) intent.Intention {
	m, ok := in.Value.(map[string]any)
	if in.Kind != "comment" || !ok {
		return in
	}
	body, _ := m["body"].(string)
	why := strings.Join(strings.Fields(failed.Error()), " ")
	out := maps.Clone(m)
	out["body"] = strings.ReplaceAll(body, intent.WhyMark, "`"+strings.ReplaceAll(why, "`", "'")+"`")
	return intent.Intention{Kind: in.Kind, Value: out}
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

// baseMerge is a merge of the base the role asks committed on its own
// branch (ADR-0067): the base's commit merged in, and the tree the merge
// settled, its conflicts' markers gone.
type baseMerge struct{ Base, Tree string }

// ownMerge is out/merge-request.yaml's `merge:` and `tree:`, nil none.
func ownMerge(runDir string) *baseMerge {
	var mr struct{ Merge, Tree string }
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err != nil || yaml.Unmarshal(data, &mr) != nil || mr.Merge == "" || mr.Tree == "" {
		return nil
	}
	return &baseMerge{Base: mr.Merge, Tree: mr.Tree}
}

// ownReady says the role asks its own Draft, the merge request it
// targets, marked ready for review: out/merge-request.yaml's `ready:
// true` (the developer's, #969).
func ownReady(runDir string) bool {
	var mr struct{ Ready bool }
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err != nil || yaml.Unmarshal(data, &mr) != nil {
		return false
	}
	return mr.Ready
}

// markOwnReady marks the merge request ready for review: from the role's
// own branch only, never another's; one ready already is left, so a
// resumed apply marks nothing twice.
func markOwnReady(f forge.Forge, st runState) error {
	mr, err := f.MergeRequest(st.Target.ID)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(mr.Branch, branchPrefix(st.Role)) || !mr.Here {
		return fmt.Errorf("merge request %d comes from %s, not a branch of the %s's own: not marked ready", st.Target.ID, mr.Branch, st.Role)
	}
	if !mr.Draft {
		return nil
	}
	r, ok := f.(forge.Readied)
	if !ok {
		return fmt.Errorf("this forge marks no draft ready: a person marks merge request %d ready", st.Target.ID)
	}
	return r.MarkReady(st.Target.ID)
}

// commitOnOwnBranch commits what the patches wrote on the merge request's
// branch, the role's own (workline/<role>/…, in this repository), on top
// of what others pushed, and pushes it without force: the working tree
// must be at that branch's tip, else nothing is committed. Another's
// branch is never written to (ADR-0051). A merge of the base (ADR-0067)
// is committed whole, its tree the role's, its parents the tip and the
// base's commit, never a rewrite; the working tree moved to it.
func commitOnOwnBranch(f forge.Forge, st runState, message string, merge *baseMerge) error {
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
	if merge != nil {
		if err := commitMerge(f, st, mr, message, *merge); err != nil {
			return err
		}
	} else if err := commitWritten(st, message); err != nil {
		return err
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

// commitMerge commits the role's merge of the base on HEAD: the base's
// commit in its base branch's history, never another branch's, nothing
// written by a patch beside it. The working tree moves to it as a
// fast-forward would: git refuses it whole, or makes it whole.
func commitMerge(f forge.Forge, st runState, mr forge.MergeRequest, message string, m baseMerge) error {
	if len(st.Written) > 0 {
		return errors.New("a merge of the base is committed alone: patches beside it are not")
	}
	baseRef := "refs/heads/" + mr.Base
	if !forge.KeepsBranches(f) {
		if _, err := git(st.Repo, nil, "fetch", "-q", "origin", mr.Base); err != nil {
			return fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
		}
		baseRef = "FETCH_HEAD"
	}
	if !isAncestor(st.Repo, m.Base, baseRef) {
		return fmt.Errorf("%s is not in %s's history: only the base is merged in", short(m.Base), mr.Base)
	}
	if kind, err := git(st.Repo, nil, "cat-file", "-t", m.Tree); err != nil || strings.TrimSpace(kind) != "tree" {
		return fmt.Errorf("the merge's tree %s is not in this clone", short(m.Tree))
	}
	args := []string{"commit-tree", m.Tree, "-p", "HEAD", "-p", m.Base, "-F", "-"}
	if signed, _ := git(st.Repo, nil, "config", "--bool", "--get", "commit.gpgsign"); strings.TrimSpace(signed) == "true" {
		args = append(args, "-S") // as `git commit` signs
	}
	commit, err := git(st.Repo, strings.NewReader(message+"\n\n"+st.trailers()+"\n"), args...)
	if err != nil {
		return err
	}
	_, err = git(st.Repo, nil, "merge", "--ff-only", "-q", strings.TrimSpace(commit))
	return err
}

// commitWritten commits what the patches wrote on HEAD.
func commitWritten(st runState, message string) error {
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
// open. It is built in tree, the worktree apart the patches were written
// in, the checkout never moved (#1372). The role names the task's key and
// the title in out/merge-request.yaml.
//
// A run whose push failed has its commits on the branch still: the tip
// is returned, kept (runState.Committed), and its resume pushes them as
// they are, nothing committed again (#739).
func openMergeRequest(f forge.Forge, st runState, runDir, tree string) (int, string, error) {
	var mr struct {
		Key, Title, Body string
		Draft            bool
		Proposal         `yaml:",inline"`
	}
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &mr); err != nil {
			return 0, "", fmt.Errorf("out/merge-request.yaml: %w", err)
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
	return proposeBranch(f, st.Repo, tree, st.Role, mr.Key, mr.Title, mr.Body, mr.Draft, st.trailers(), st.Written, mr.Proposal, st.Committed)
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
	// those no group names go in the last, Message its message (the
	// developer's code and tests apart, ADR-0054).
	Commits []Commit `yaml:"commits,omitempty"`
	// Message is the last commit's message, "" the title: the developer's
	// subject, as the project's commit check reads it (#736).
	Message string `yaml:"message,omitempty"`
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
		msg := title
		if strings.TrimSpace(p.Message) != "" {
			msg = p.Message
		}
		out = append(out, Commit{Message: msg, Files: left})
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
	id, _, err := proposeBranch(f, repo, "", roleName, key, title, body, draft, trailers, written, p, "")
	return id, err
}

// proposeBranch is ProposeBranchFrom, also returning the branch's tip once
// committed. committed, when the branch's tip still, is a failed push's
// commits: pushed as they are, nothing committed again (#739). tree, when
// not "", is a worktree of repo's HEAD, detached, holding the files
// written: the branch is built there, repo's checkout never moved (#1372).
func proposeBranch(f forge.Forge, repo, tree, roleName, key, title, body string, draft bool, trailers string, written []string, p Proposal, committed string) (int, string, error) {
	base, err := git(repo, nil, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return 0, "", errors.New("a merge request is opened from a branch; this run is on a detached HEAD")
	}
	// at: where the branch is built; home: the checkout that puts it back.
	at, home := repo, []string{base}
	if tree != "" {
		head, err := git(tree, nil, "rev-parse", "HEAD")
		if err != nil {
			return 0, "", err
		}
		at, home = tree, []string{"--detach", head}
	}
	branch := branchPrefix(roleName) + slugify(key)
	baseTip, was, err := tips(repo, base, branch, forge.KeepsBranches(f))
	if err != nil {
		return 0, "", err
	}
	if was != "" {
		// The commits it started from are not a person's; on another role's
		// commit, nor that role's older ones, rewritten since.
		since := []string{baseTip}
		if p.From != "" {
			since = append(since, p.From)
		}
		if who := personsCommitPast(repo, since, was, roleName, p.From != ""); who != "" {
			return 0, "", fmt.Errorf("%w: %s", ErrLeftToAPerson, who)
		}
	}
	var steps [][]string
	tip := ""
	if now, _ := git(repo, nil, "rev-parse", "--verify", "-q", "refs/heads/"+branch); committed == "" || now != committed {
		start := []string{"checkout", "-q", "-B", branch}
		if p.From != "" {
			// The files written carried over onto p.From, merged: the
			// working tree is at the base's tip, which may have changed
			// them since p.From left it, and a plain checkout refuses
			// then (#856). Their change from the base's tip lands on
			// p.From, the base's own changes to them not carried in.
			start = []string{"checkout", "-q", "-m", "-B", branch, p.From}
		}
		steps = append(steps, start)
		for _, c := range p.commits(title, written) {
			steps = append(steps, append([]string{"add", "--"}, c.Files...), []string{"commit", "-q", "-m", c.Message, "-m", trailers})
		}
	} else {
		tip = committed // on the branch already: nothing new to commit
	}
	if !forge.KeepsBranches(f) {
		steps = append(steps, []string{"push", "-q", "--force", "origin", branch})
	}
	// What the files written change from the base's tip, kept to put
	// them back should their merge onto p.From conflict.
	var saved string
	if p.From != "" && len(steps) > 0 && steps[0][0] == "checkout" {
		if saved, err = gitRaw(at, append([]string{"diff", "--binary", "HEAD", "--"}, written...)...); err != nil {
			return 0, "", err
		}
	}
	for i, s := range steps {
		if s[0] == "push" && tip == "" {
			tip, _ = git(repo, nil, "rev-parse", "refs/heads/"+branch)
		}
		if i == 0 && saved != "" {
			if err := mergeOnto(at, home, s, saved, p.From); err != nil {
				return 0, "", err
			}
			continue
		}
		dir := at
		if s[0] == "push" {
			dir = repo // a remote's relative path reads from the clone, never the worktree
		}
		if _, err := git(dir, nil, s...); err != nil {
			// Back where it was: the patches in the working tree, unstaged.
			_, _ = git(at, nil, append([]string{"reset", "-q", "--"}, written...)...)
			_, _ = git(at, nil, append([]string{"checkout", "-q"}, home...)...)
			if s[0] == "push" {
				// Committed, kept on the branch: a resume pushes it.
				return 0, tip, fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
			}
			return 0, "", err
		}
	}
	if len(steps) > 0 && steps[0][0] == "checkout" { // else never moved
		if _, err := git(at, nil, append([]string{"checkout", "-q"}, home...)...); err != nil {
			return 0, tip, err
		}
	}
	id, err := f.OpenMergeRequest(branch, base, title, body, draft)
	return id, tip, err
}

// mergeOnto runs start, the checkout that carries the files written onto
// from, merged. A conflict puts the working tree back by the checkout of
// home, the files written as they were (saved, their diff from base's
// tip), and names the files it conflicts on.
func mergeOnto(repo string, home, start []string, saved, from string) error {
	if _, err := git(repo, nil, start...); err != nil {
		return err // refused: nothing moved
	}
	unmerged, err := git(repo, nil, "diff", "--name-only", "--diff-filter=U")
	if err != nil || unmerged == "" {
		return err
	}
	back := [][]string{{"reset", "-q"}, append([]string{"checkout", "-q", "-f"}, home...)}
	for _, s := range back {
		if _, err := git(repo, nil, s...); err != nil {
			return fmt.Errorf("the code conflicts with %s on %s, and the working tree is not put back: %w", short(from), strings.ReplaceAll(unmerged, "\n", ", "), err)
		}
	}
	if _, err := git(repo, strings.NewReader(saved), "apply", "--binary", "-"); err != nil {
		return fmt.Errorf("the code conflicts with %s on %s, and its change is not put back in the working tree: %w", short(from), strings.ReplaceAll(unmerged, "\n", ", "), err)
	}
	return fmt.Errorf("the code, written on %s, conflicts with what the base changed since in %s: nothing committed, the working tree as it was", short(from), strings.ReplaceAll(unmerged, "\n", ", "))
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
	intents       []intent.Intention
	verdict       *verdict.Verdict
	findings      []verdict.Finding // why proposals were refused before judging
	askedAgent    bool
	external      bool
	unread        string             // why the agent's answer could not be read, if it could not
	refused       string             // why the engine refused the answer's proposals whole, if it did
	said          []intent.Intention // what post said of its block (out/on-block.yaml)
	overBudget    bool               // the task was not put before the agent: over the context budget
	overBudgetWhy string             // the refusal of the prompt, said to post
}

// attempt asks the agent (when there is a question and an agent), merges the
// fallback proposals, refuses what cannot be applied, and runs the judge.
func attempt(r *role.Role, o Options, ag agent.Agent, hasTask bool, tier, runDir string, env []string,
	line *routing.Config, settings map[string]any, cfg *role.ProjectConfig, res *Result) (*attemptResult, error) {
	a := &attemptResult{}
	// A kind of task may give its own instruction and fewer intentions
	// (model.tasks, #878): the agent reads those, and answers with those.
	asked := r.Asked()
	if hasTask && ag != nil {
		a.askedAgent = true
		err := callAgent(ag, agent.Request{RunDir: runDir, Repo: o.Repo, Role: asked, Tier: tier}, taskKind(runDir), runDir, res)
		switch {
		case err == nil:
		case errors.Is(err, errTokensSpent), errors.Is(err, errTimeSpent):
			a.askedAgent = false // not asked, or not waited for: no answer to refuse, none to ask again
		case errors.Is(err, agent.ErrOverBudget) && res.askedAgain:
			// Over only with what the first answer added (its feedback):
			// not the input's size, which fitted. A machine error a run
			// again may mend, never one of size (#1160).
			a.askedAgent = false
			a.findings = append(a.findings, verdict.Finding{Rule: "prompt-over-budget-asked-again", Level: "warn",
				Message: err.Error() + ": the task with what its first answer needs mended; not asked again"})
		case errors.Is(err, agent.ErrOverBudget):
			// A task too large for the role's context is the role's to
			// shrink, not a block of the line: the run goes on as without
			// an agent, and what the task held waits for a person.
			a.askedAgent, a.overBudget, a.overBudgetWhy = false, true, err.Error()
			a.findings = append(a.findings, verdict.Finding{Rule: "prompt-over-budget", Level: "warn",
				Message: err.Error() + " (context.budget in its role.yaml): the agent was not asked; what this task held is judged by no one this run, a person reads it (in/task.md of the run)"})
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
	if asked != r {
		// What the agent answered outside its task's intentions is
		// dropped, said; the rest of its answer, and what pre falls back
		// on, stand.
		var dropped []string
		intents = slices.DeleteFunc(intents, func(in intent.Intention) bool {
			if asked.Allows(in.Kind) {
				return false
			}
			dropped = append(dropped, in.Kind)
			return true
		})
		if len(dropped) > 0 {
			a.findings = append(a.findings, verdict.Finding{Rule: "intention-refused", Level: "warn",
				Message: fmt.Sprintf("%s dropped: a %s task answers with %s only", strings.Join(slices.Compact(slices.Sorted(slices.Values(dropped))), ", "), taskKind(runDir), strings.Join(asked.Intentions, ", "))})
		}
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
		a.refused = why
		intents = nil
		os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	}
	if bad := outOfBounds(o.Repo, intents, r.Writes(settings), o.Scope); len(bad) > 0 {
		res.Refused = append(append([]string{}, res.partRefused...), "patch")
		a.refused = "a patch reaches outside the task or the role's duties: " + strings.Join(bad, ", ") + "; anything found there belongs in an issue"
		a.findings = append(a.findings, verdict.Finding{Rule: "intention-refused", Message: a.refused})
		intents = nil
		os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	}
	a.intents = intents

	// post is told why the answer did not read, or was refused whole, to
	// say it; what it leaves in out/on-block.yaml is its own, never an
	// agent's: removed first.
	postEnv := env
	if a.unread != "" {
		postEnv = append(env[:len(env):len(env)], "WORKLINE_AGENT_UNREAD="+a.unread)
	} else if a.refused != "" {
		postEnv = append(env[:len(env):len(env)], "WORKLINE_AGENT_REFUSED="+a.refused)
	} else if a.overBudget {
		// Not asked: post says the budget, not an empty answer (#1017).
		postEnv = append(env[:len(env):len(env)], "WORKLINE_AGENT_OVER_BUDGET="+a.overBudgetWhy)
	} else if a.askedAgent && !a.external {
		// Its answer read and kept: what it holds, or lacks, is the
		// agent's word (#794).
		postEnv = append(env[:len(env):len(env)], "WORKLINE_AGENT_ANSWERED=true")
	}
	onBlockFile := filepath.Join(runDir, "out", "on-block.yaml")
	if err := os.Remove(onBlockFile); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	code, err := postAsking(r, o, ag, runDir, postEnv, res)
	if err != nil {
		return nil, err
	}
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoVerdict, err)
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
	// An agent that edited a worktree may answer nothing: its diff is its
	// proposal, post's to write (ADR-0056). One that proposed nothing may
	// leave post moving its issue all the same: the tester's "no test
	// possible", its label and comment, are applied (#1194).
	if v.Status == verdict.Pass {
		if a.intents, err = narrowedByPost(r, o, runDir, line, settings); err != nil {
			return nil, err
		}
	}
	if v.Status == verdict.Pass && a.askedAgent {
		if err := askJudge(o, res, runDir, v, a); err != nil {
			return nil, err
		}
	}
	// Blocked, post may tell why where its target is read, as pre's
	// on-block proposals do (ADR-0059, decision 1): held to the role's
	// catalogue and bounds, applied though the run blocks.
	if v.Status == verdict.Block {
		said, err := intent.Read(onBlockFile)
		if err != nil {
			return nil, fmt.Errorf("post's out/on-block.yaml: %w", err)
		}
		if refused, why := invalid(r, said, line); len(refused) > 0 {
			return nil, fmt.Errorf("post's out/on-block.yaml: %s", why)
		}
		if bad := outOfBounds(o.Repo, said, r.Writes(settings), o.Scope); len(bad) > 0 {
			return nil, fmt.Errorf("post's out/on-block.yaml reaches outside the role's duties: %s", strings.Join(bad, ", "))
		}
		a.said = said
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
	code, err := postAsking(r, o, nil, runDir, env, res)
	if err != nil {
		return nil, err
	}
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoVerdict, err)
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
	// tree: where the patches are written, when not in repo: a worktree
	// apart, for patches that go in a comment (#1328).
	tree        string
	targets     map[string]string
	writes      []string    // paths the role may write
	written     []string    // files changed by this run's patches
	handoffs    []any       // next roles asked for, recorded for routing
	forge       forge.Forge // nil when the project has no forge
	target      *forge.Target
	runID, role string
	index       int           // position of the intention being applied, for its marker
	plan        *backlog.Plan // what becomes of the acts on the backlog, nil without any
	settings    map[string]any
	opens       *backlog.Openings // the run's issues opened, made on the first
	capped      int               // findings not opened as issues, past issues-max
	findings    []verdict.Finding // what the applying says
	model       string            // the model whose answer is applied, as an announcement records it
	own         string            // the run's own trailers: its role, the agents and models that answered
	scheme      *scale.Scheme     // where the project keeps each axis, detected at the first act writing one (ADR-0040)
	ensured     map[string]bool   // the labels this run made sure the project has
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
			if ids := strs(m["resolve"]); len(ids) > 0 {
				// The role's own conversations on lines the code answers
				// now (#1039).
				if err := a.resolveThreads(*target, ids); err != nil {
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
	case "verification":
		// The tester's test plan, written in the issue it names (ADR-0066):
		// added, or its own rewritten; another's kept as it reads.
		m, _ := in.Value.(map[string]any)
		n, _ := m["issue"].(int)
		text, _ := m["text"].(string)
		if n <= 0 || strings.TrimSpace(text) == "" {
			return errors.New("a verification needs {issue, text}")
		}
		if err := a.needForge("verification"); err != nil {
			return err
		}
		b, ok := a.forge.(forge.Backlog)
		if !ok {
			return errors.New("a verification is written in an issue, on a forge that keeps a backlog")
		}
		is, err := a.forge.Issue(n)
		if err != nil {
			return err
		}
		t := forge.Target{Kind: "issue", ID: n}
		comments, err := b.Comments(t)
		if err != nil {
			return err
		}
		st, found, err := backlog.ReadState(comments, backlog.Keeper)
		if err != nil {
			st = nil // one that does not read tells no Verification of its own
		}
		body, changed := backlog.WithVerification(is.Body, text, st)
		if !changed {
			return nil
		}
		if err := b.SetBody(n, body); err != nil {
			return err
		}
		// The tester's test plan is no person's change: the product
		// owner, which read the body as it was, keeps it as it is now,
		// and is not woken to read it again.
		if st == nil || !found || st.Body != backlog.BodyDigest(is.Body) {
			return nil // none, or one that does not read: left as it is
		}
		st.Keep(body)
		return a.forge.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(backlog.Keeper), false)
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
		add, remove := strs(m["add"]), strs(m["remove"])
		var card *board.Move
		if slot, _ := m["card"].(string); slot != "" && target.Kind == "issue" {
			// The state's card on the board, read now (ADR-0059,
			// decision 2): GitLab's column label in the same call.
			card = board.Along(a.forge, a.repo, slot)
			add, remove = card.Fold(add, remove)
		}
		if err := a.forge.Label(*target, add, remove); err != nil {
			return err
		}
		if card.Writes() {
			a.moveCard(target.ID, card.Project, card.Column)
		}
		if st, ok := m["status"].(map[string]any); ok && target.Kind == "issue" {
			project, _ := st["project"].(int)
			option, _ := st["option"].(string)
			a.moveCard(target.ID, project, option)
		}
		return nil
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
		return a.refining(d.Act)
	case "kind":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.kind(d.Act)
	case "size":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.size(d.Act)
	case "rename":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return rename(a.forge, a.role, d.Act)
	case "unready":
		// Back to refine, alone or a person having accepted it (ADR-0059,
		// amended): its labels moved, a developer's claim with them (one state
		// at a time); why, and how to undo it, the role's
		// one comment on the issue says (Persist).
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		t := forge.Target{Kind: "issue", ID: d.Act.Issue}
		card := board.Along(a.forge, a.repo, "refine") // its card back to Refine, when the board has it
		add, remove := card.Fold([]string{backlog.LabelToRefine}, []string{backlog.LabelReady, backlog.LabelNeedsTests, "workline:developing"})
		if err := a.forge.Label(t, add, remove); err != nil {
			return err
		}
		if card.Writes() {
			a.moveCard(t.ID, card.Project, card.Column)
		}
		return nil
	case "unblock":
		// Out of Blocked, alone (ADR-0059, amended): to the state it came
		// from or to refine, every other state's label off, its card
		// moved; why, and how to undo it, the role's one comment says.
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		line, err := routing.Load(a.repo)
		if err != nil {
			return err
		}
		add, remove, err := line.Move(d.Act.To)
		if err != nil {
			return err
		}
		t := forge.Target{Kind: "issue", ID: d.Act.Issue}
		card := board.Along(a.forge, a.repo, d.Act.To)
		add, remove = card.Fold(add, remove)
		if err := a.forge.Label(t, add, remove); err != nil {
			return err
		}
		if card.Writes() {
			a.moveCard(t.ID, card.Project, card.Column)
		}
		return nil
	case "blocked":
		return nil // kept blocked: said in the role's one comment, nothing moved
	case "assume":
		// A question unanswered past the delay (ADR-0060): the assumption
		// under the issue's Assumptions, said in the role's one comment
		// (Persist); the person's sections left as they are.
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return a.assume(d.Act)
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

// errNotApplied: a diff, read and allowed, that git does not apply on the
// files as they are.
var errNotApplied = errors.New("the diff does not apply")

// patch applies a unified diff, or replaces one file with {file, content}.
func (a *applier) patch(v any) error {
	root := cmp.Or(a.tree, a.repo) // a worktree apart, when the patches go in a comment
	if m, ok := v.(map[string]any); ok {
		file, _ := m["file"].(string)
		content, ok := m["content"].(string)
		if file == "" || !ok {
			return errors.New("expected {file, content}")
		}
		if !a.allowed(file) {
			return fmt.Errorf("%s is outside what this role may write", file)
		}
		if link := throughLink(root, file); link != "" {
			return fmt.Errorf("%s is a link: nothing is written through it", link)
		}
		// A folder the base lacks is made, as git apply makes it for a
		// diff (#989): no link on the way, checked above, and within
		// the repository, as allowed reads the path.
		full := filepath.Join(root, file)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
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
	files, err := git(root, strings.NewReader(diff), "apply", "--recount", "--unidiff-zero", "--numstat", "-z", "-")
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
		if link := throughLink(root, f[2]); link != "" {
			return fmt.Errorf("%s is a link: nothing is written through it", link)
		}
		touched = append(touched, f[2])
	}
	// --recount: a diff is applied as its lines read. Agents often get the
	// counts of a hunk header wrong, and git would drop the lines past them.
	// --unidiff-zero: a role's own diff may hold no context (intent.Merge).
	if _, err := git(root, strings.NewReader(diff), "apply", "--recount", "--unidiff-zero", "-"); err != nil {
		return fmt.Errorf("%w: %w", errNotApplied, err)
	}
	a.written = append(a.written, touched...)
	return nil
}

// notText says why a diff does more than change text files: a rename or a
// copy — whose source git's count does not name —, a mode changed, a link
// or a submodule touched, an executable made or deleted, a binary patch;
// "" when it does not: an executable's text edited is text.
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
			// The mode an index line gives is the file's, unchanged: an
			// executable script's text is text to edit (#1330). What is
			// committed keeps the index's mode, whatever the line says;
			// a link, a submodule is not text.
			if f := strings.Fields(l); len(f) > 2 && f[2] != "100644" && f[2] != "100755" {
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

// keepRuns is how many past runs are kept for inspection; what an older
// one's agent calls spent stays in the folder's tally (TallyFile).
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
	if entries, err := os.ReadDir(base); err == nil {
		var old []os.DirEntry
		for _, e := range entries {
			if e.IsDir() { // the tally is no run
				old = append(old, e)
			}
		}
		for i := 0; i < len(old)-keepRuns; i++ { // names start with a timestamp, so the oldest come first
			past := filepath.Join(base, old[i].Name())
			if keepTally(base, past) != nil {
				continue // what it spent could not be kept: the folder stays, to be tried again
			}
			dropWorktree(repo, past) // an interrupted run's tree may lie outside it
			_ = os.RemoveAll(past)
		}
	}
	return dir, nil
}

// dropWorktree removes the worktree, or the read tree, a role's pre made
// in the run for its agent, kept through every attempt (ADR-0056,
// ADR-0063 §8); none, nothing.
func dropWorktree(repo, runDir string) {
	for name, tree := range map[string]string{agent.WorktreeDir: agent.Worktree(runDir), agent.ReadTreeDir: agent.ReadTree(runDir)} {
		if tree != "" {
			tester.RemoveRunTree(repo, runDir, name, tree)
		}
	}
}

// RunsDir is where repo's runs are kept: WORKLINE_RUNS_DIR when set — in
// CI, somewhere an artifact can carry them to the job that applies —, else
// its git folder's workline/runs: the main clone's, which its worktrees
// share, so a run made in one outlives it (#1324).
func RunsDir(repo string) string {
	if d := os.Getenv(forge.RunsDirEnv); d != "" {
		return d
	}
	if out, err := exec.Command("git", "-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output(); err == nil {
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

// Self is the engine's binary a role's scripts run (WORKLINE_BIN): this
// one, or the one a test built when the line runs in its process (the
// line simulator, ADR-0063).
var Self = func() string { self, _ := os.Executable(); return self }

func scriptEnv(runDir, roleName string, o Options) []string {
	ai := o.AI
	if ai == "" {
		ai = "none"
	}
	self := Self()
	env := append(os.Environ(),
		"WORKLINE_RUN_DIR="+runDir,
		"WORKLINE_EVENT="+o.Event,
		"WORKLINE_AI="+ai,
		"WORKLINE_ROLE="+roleName,
		"WORKLINE_BIN="+self,
		"WORKLINE_ROLES_DIR="+o.RolesDir,
	)
	if n := backlog.TierFactor(o.Inputs[SizeTierInput]); n > 1 { // its issue's size tier (ADR-0064)
		env = append(env, fmt.Sprintf("%s=%d", role.SizeFactorEnv, n))
	}
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

// phase says in CI's job log which step of a role's run starts: a run
// that hangs says where it was (#1197). Outside CI, nothing.
func phase(role, step string) {
	if forge.InCI() != "" {
		fmt.Fprintf(os.Stderr, "workline: %s: %s (%s UTC)\n", role, step, time.Now().UTC().Format("15:04:05"))
	}
}

// script runs roles/<name>/<step> in the repository and returns its exit code.
func script(r *role.Role, step, repo string, env []string) (int, error) {
	path := filepath.Join(r.Dir, step)
	if _, err := os.Stat(path); err != nil {
		return 0, fmt.Errorf("role %q has no %s script", r.Name, step)
	}
	phase(r.Name, step)
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

// postAsking runs post, then the questions it leaves for a judge
// (out/judge/<key>/question.yaml, as pre's): each asked as pre's are, and
// post run again with their answers (WORKLINE_JUDGED=answered), until it
// asks none (the tester's red proof of a broken build, ADR-0065). With
// no agent, or past the rounds, a question is answered as not asked.
func postAsking(r *role.Role, o Options, ag agent.Agent, runDir string, env []string, res *Result) (int, error) {
	code, err := script(r, "post", o.Repo, env)
	for round := 1; err == nil; round++ {
		questions := unanswered(filepath.Join(runDir, "out", "judge"))
		if len(questions) == 0 {
			break
		}
		switch {
		case round > maxJudgeRounds+1:
			return 0, fmt.Errorf("post asked its judges again past %d rounds", maxJudgeRounds)
		case ag == nil:
			err = notAsked(questions, "not asked: no agent to judge with")
		case round > maxJudgeRounds:
			err = notAsked(questions, fmt.Sprintf("not asked: past %d rounds of the judge", maxJudgeRounds))
		default:
			err = askQuestions(o, ag, runDir, questions, res)
		}
		if err != nil {
			return 0, err
		}
		code, err = script(r, "post", o.Repo, append(env[:len(env):len(env)], "WORKLINE_JUDGED=answered"))
	}
	return code, err
}

// unanswered are the questions under dir holding no answer yet.
func unanswered(dir string) []string {
	questions, _ := filepath.Glob(filepath.Join(dir, "*", "question.yaml"))
	return slices.DeleteFunc(questions, func(q string) bool {
		_, err := os.Stat(filepath.Join(filepath.Dir(q), "answer.yaml"))
		return err == nil
	})
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
