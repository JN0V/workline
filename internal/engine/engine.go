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
	"strconv"
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
	// PushToMergeRequest commits what the patches wrote to the branch of the
	// merge request the run targets; from a fork, the diff goes in a comment.
	PushToMergeRequest bool

	// TamperBeforeApply changes the prepared input between propose and apply.
	// It exists only for the conformance test that proves apply notices.
	TamperBeforeApply bool
}

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
	maxTokens    int      // the role's ai-max-tokens: what the run may spend; 0, no cap
	proposed     []string // the merge requests' tasks earlier rounds proposed, with NoApply, not opened yet
	capped       bool     // a call was refused, the run having spent maxTokens
	partRefused  []string // what the parts answered and may not: refused with them
}

// errTokensSpent refuses a call once the run has spent its ai-max-tokens.
var errTokensSpent = errors.New("the run spent its ai-max-tokens")

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
		if res.capped || !last && res.overBudget() {
			// A cap is checked against what was spent, never estimated: the
			// call that crossed it is paid. What is left waits, said.
			res.Findings = append(res.Findings, verdict.Finding{Rule: "ai-max-tokens", Level: "warn",
				Message: fmt.Sprintf("%d tokens spent, the run allows %d: the agent was asked nothing more; what is left waits for the next run", res.tokensSpent(), res.maxTokens)})
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
	ag, err := agent.Parse(o.AI)
	if err != nil {
		return err
	}
	runDir, err := newRunDir(o.Repo, r.Name)
	if err != nil {
		return err
	}
	res.RunDir = runDir
	if err := writeInputs(runDir, o.Inputs, r.MergedSettings(cfg)); err != nil {
		return err
	}
	res.maxTokens = intSetting(r.MergedSettings(cfg), "ai-max-tokens")
	if o.Forge == "" {
		o.Forge = cfg.Forge
	}
	releaseFrom := releaseRequest(r, &o, res)
	env := scriptEnv(runDir, r.Name, o)
	if releaseFrom != "" {
		env = append(env, "WORKLINE_RELEASE_BRANCH="+releaseFrom)
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
	// settles what follows from the answers.
	if questions, _ := filepath.Glob(filepath.Join(runDir, "in", "judge", "*", "question.yaml")); len(questions) > 0 {
		if err := askQuestions(o, ag, runDir, questions, res); err != nil {
			return err
		}
		code, err := script(r, "pre", o.Repo, append(env[:len(env):len(env)], "WORKLINE_PARTS=answered", "WORKLINE_JUDGED=answered"))
		if err != nil {
			return err
		}
		if stop, err := prepared(code, r, cfg, runDir, res); stop || err != nil {
			return err
		}
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
	settings := r.MergedSettings(cfg)
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
	for {
		a, err := attempt(r, o, ag, taskErr == nil, tier, runDir, env, line, settings, cfg, res)
		if err != nil {
			return err
		}
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
		if retry, tier = askAgain(failures, r.Model.PromoteAfter, tier); !retry {
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
	// Asked as often as the role allows and still refused: the docs the
	// refusals name are left out, and the rest judged again, without them.
	if v.Status == verdict.Block && failures > 0 {
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
	intent.SortForApply(intents)
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return err
	}
	if o.Forge == "" {
		o.Forge = cfg.Forge
	}
	st := runState{Role: r.Name, RolesDir: o.RolesDir, Repo: o.Repo, Forge: o.Forge, Target: o.Target, Scope: o.Scope, Digest: digest, Targets: o.Targets,
		OpenMergeRequest: o.OpenMergeRequest, PushToMergeRequest: o.PushToMergeRequest, Release: releaseFrom, Models: authors(res.Calls)}
	if err := st.save(runDir); err != nil {
		return err
	}
	if o.NoApply {
		if onBlock {
			res.Summary = fmt.Sprintf("%s; what says why judged, not applied (%d proposals); apply with: workline apply %s", res.Summary, len(intents), runDir)
		} else {
			res.Summary = fmt.Sprintf("judged, not applied (%d proposals); apply with: workline apply %s", len(intents), runDir)
		}
		res.ToApply = true
		for _, in := range intents {
			if in.Kind == "handoff" {
				res.Findings = append(res.Findings, deferred(r.Name, in.Value))
			}
		}
		// Judged here, the fix is not on the release pull request: it waits.
		if releaseFrom != "" && o.PushToMergeRequest && slices.ContainsFunc(intents, func(i intent.Intention) bool { return i.Kind == "patch" }) {
			res.Status = verdict.Block
			res.Findings = append(res.Findings, verdict.Finding{Rule: "fixed-elsewhere", Level: "block", Where: releaseFrom,
				Message: "once applied, the fix goes to a merge request of its own into the release's base, not onto this branch, which the release tool rewrites: the release waits until it is merged"})
		}
		return nil
	}
	return applyAll(r, settings, st, runDir, intents, res)
}

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
		case errors.Is(err, agent.ErrUnavailable), errors.Is(err, errTokensSpent):
			gone, goneKind = err.Error(), "unavailable"
			if errors.Is(err, errTokensSpent) {
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
	res.AgentCalls++
	start := time.Now()
	call, err := ag.Propose(req)
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
	// PushToMergeRequest: the patches go to the targeted merge request's
	// branch; Pushed once they did, as a commit or as a comment.
	PushToMergeRequest bool `yaml:"push-to-merge-request,omitempty"`
	Pushed             bool `yaml:"pushed,omitempty"`
	// Release: the targeted merge request is a release tool's, from this
	// branch, which the tool rewrites: the patches go to a merge request of
	// their own on its base instead (ADR-0017).
	Release string `yaml:"release,omitempty"`
	// Models: the agents and models that wrote the proposals, as
	// <agent>:<model>, named in the commit (ModelTrailer).
	Models []string `yaml:"models,omitempty"`
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
		forge: f, target: st.Target, runID: filepath.Base(runDir), role: r.Name, settings: settings}
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
	for i, in := range intents {
		acted := !slices.Contains(backlog.Kinds, in.Kind) || plan.Acts(i)
		if done[i] {
			if acted {
				res.Applied = append(res.Applied, in.Kind)
			}
			continue
		}
		ap.index = i
		if err := ap.apply(in); err != nil {
			if errors.Is(err, forge.ErrUnreachable) {
				res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while applying %s; resume with: workline apply %s", in.Kind, runDir)
				res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
				return nil
			}
			return fmt.Errorf("apply %s: %w", in.Kind, err)
		}
		st.Applied = append(st.Applied, i)
		for _, w := range ap.written { // a resumed run adds to what the first attempt wrote
			if !slices.Contains(st.Written, w) {
				st.Written = append(st.Written, w)
			}
		}
		if err := st.save(runDir); err != nil {
			return err
		}
		if acted {
			res.Applied = append(res.Applied, in.Kind)
		}
	}
	res.Findings = append(res.Findings, ap.findings...)
	if ap.capped > 0 {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "issues-capped", Level: "warn",
			Message: fmt.Sprintf("%d more issues not opened (issues-max %d): found again, they are opened at a later run", ap.capped, ap.opens.Max)})
	}
	if plan != nil {
		res.Findings = append(res.Findings, plan.Findings...)
		if err := report(f, r.Name, plan); errors.Is(err, forge.ErrUnreachable) {
			res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while writing the backlog report; resume with: workline apply %s", runDir)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
			return nil
		} else if err != nil {
			return fmt.Errorf("the backlog report: %w", err)
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
		if err != nil {
			return fmt.Errorf("open a merge request: %w", err)
		}
		st.MergeRequest = id
		if err := st.save(runDir); err != nil {
			return err
		}
	}
	if st.PushToMergeRequest && st.Release != "" && len(st.Written) > 0 {
		if !st.Pushed {
			finding, id, err := fixElsewhere(f, st, runDir)
			if errors.Is(err, forge.ErrUnreachable) {
				res.Status, res.Summary = verdict.BlockedExternal, fmt.Sprintf("stopped while proposing the fix; resume with: workline apply %s", runDir)
				res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
				return nil
			}
			if err != nil {
				return fmt.Errorf("propose the fix: %w", err)
			}
			st.Pushed, st.MergeRequest = true, id
			if err := st.save(runDir); err != nil {
				return err
			}
			if id == 0 { // in a comment, for a person
				res.Findings = append(res.Findings, finding)
			}
		}
		// Until the fix is merged, the release waits for it.
		res.Status = verdict.Block
		for _, w := range st.Written {
			if st.MergeRequest != 0 {
				res.Findings = append(res.Findings, verdict.Finding{Rule: "fixed-elsewhere", Level: "block", Where: w,
					Message: fmt.Sprintf("fixed in merge request #%d, not on %s, which the release tool rewrites: the release waits until #%d is merged", st.MergeRequest, st.Release, st.MergeRequest)})
			}
		}
	}
	res.MergeRequest = st.MergeRequest
	if st.PushToMergeRequest && len(st.Written) > 0 && !st.Pushed {
		if f == nil || st.Target == nil || st.Target.Kind != "merge-request" {
			return errors.New("--push-to-merge-request needs a forge and --target merge-request:<n>")
		}
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

// fixElsewhere puts what the run's patches wrote, on a release tool's pull
// request, on a merge request of the role's own into the release's base
// (ADR-0017): the tool rewrites its branch from the base, so a commit pushed
// there would be lost, and once the fix is merged the tool brings it in. It
// returns the merge request opened; 0 when the patches could not go there and
// went in a comment, said by the finding. The working tree goes back to
// where it was.
func fixElsewhere(f forge.Forge, st runState, runDir string) (verdict.Finding, int, error) {
	if f == nil || st.Target == nil || st.Target.Kind != "merge-request" {
		return verdict.Finding{}, 0, errors.New("--push-to-merge-request needs a forge and --target merge-request:<n>")
	}
	var mr struct{ Title, Body string }
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		yaml.Unmarshal(data, &mr)
	}
	if mr.Title == "" {
		mr.Title = "docs: what the release waits for"
	}
	diff, err := git(st.Repo, nil, append([]string{"diff", "--"}, st.Written...)...)
	if err != nil {
		return verdict.Finding{}, 0, err
	}
	to, err := f.MergeRequest(st.Target.ID)
	if err != nil {
		return verdict.Finding{}, 0, err
	}
	why := "the forge does not say which branch the release goes into"
	if to.Base != "" {
		why, err = commitOntoBase(st, to.Base, mr.Title, forge.KeepsBranches(f))
		if err != nil {
			return verdict.Finding{}, 0, err
		}
	}
	if why == "" {
		body := fmt.Sprintf("**Proposed by workline's %s** for the release pull request #%d (%s), which waits for it: "+
			"fixed here, not on its branch, which the release tool rewrites from %s. Once this is merged, the tool "+
			"brings it in, and the release is no longer held.\n\n%s", st.Role, st.Target.ID, st.Release, to.Base, mr.Body)
		id, err := f.OpenMergeRequest(branchPrefix(st.Role)+"release", to.Base, mr.Title, body)
		return verdict.Finding{}, id, err
	}
	body := fmt.Sprintf("**Proposed by workline's %s**, not opened as a merge request: %s. Apply it on %s with `git apply`; the release waits for it:\n\n```diff\n%s\n```", st.Role, why, to.Base, diff)
	if err := f.Sticky(*st.Target, body, forge.Marker("sticky="+st.Role+"/patch"), true); err != nil {
		return verdict.Finding{}, 0, err
	}
	return verdict.Finding{Rule: "fix-in-comment", Level: "block", Where: st.Release,
		Message: "the fix went in a comment, for a person to apply on the release's base: " + why}, 0, nil
}

// commitOntoBase commits what the run wrote onto the tip of base, on the
// role's release branch, and pushes it — force: the branch is the role's,
// and holds only the last fix. It says why not when the patches collide
// with the base. The working tree goes back to where it was.
func commitOntoBase(st runState, base, title string, local bool) (why string, err error) {
	orig, err := git(st.Repo, nil, "symbolic-ref", "-q", "--short", "HEAD")
	if err != nil {
		if orig, err = git(st.Repo, nil, "rev-parse", "HEAD"); err != nil {
			return "", err
		}
	}
	back := func() {
		git(st.Repo, nil, append([]string{"reset", "-q", "--"}, st.Written...)...)
		git(st.Repo, nil, "checkout", "-q", orig)
	}
	tip := "FETCH_HEAD"
	if local {
		if tip, err = git(st.Repo, nil, "rev-parse", "--verify", "refs/heads/"+base); err != nil {
			return "", fmt.Errorf("the release's base %s is not in this clone", base)
		}
	} else if _, err := git(st.Repo, nil, "fetch", "-q", "origin", base); err != nil {
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	if tip, err = git(st.Repo, nil, "rev-parse", tip); err != nil {
		return "", err
	}
	branch := branchPrefix(st.Role) + "release"
	// A person's commit on the branch is never pushed over (ADR-0034).
	baseTip, was, err := tips(st.Repo, base, branch, local)
	if err != nil {
		return "", err
	}
	if was != "" {
		if who := personsCommit(st.Repo, baseTip, was, st.Role); who != "" {
			return fmt.Sprintf("%s holds the last fix, and %s: not pushed over it", branch, who), nil
		}
	}
	if _, err := git(st.Repo, nil, "checkout", "-q", "--detach", tip); err != nil {
		back()
		return fmt.Sprintf("the docs it fixes differ on %s", base), nil
	}
	for _, s := range [][]string{append([]string{"add", "--"}, st.Written...), {"commit", "-q", "-m", title, "-m", st.trailers()}} {
		if _, err := git(st.Repo, nil, s...); err != nil {
			back()
			return "", err
		}
	}
	move := []string{"push", "-q", "--force-with-lease=refs/heads/" + branch + ":" + was, "origin", "HEAD:refs/heads/" + branch}
	if local {
		move = []string{"branch", "-q", "-f", branch, "HEAD"}
	}
	if _, err := git(st.Repo, nil, move...); err != nil {
		back()
		if local {
			return "", err
		}
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	_, err = git(st.Repo, nil, "checkout", "-q", orig)
	return "", err
}

// pushToMergeRequest commits what the run's patches wrote to the branch of
// the merge request it targets, as pre-commit.ci and autofix.ci do: a
// suggestion can only sit on lines the merge request changes, and a doc made
// suspect by a change of code usually has none. From a fork, or when the
// branch moved on, the diff goes in one comment instead, for the author to
// apply. The working tree goes back to where it was.
func pushToMergeRequest(f forge.Forge, st runState, runDir string) (verdict.Finding, error) {
	var mr struct{ Title string }
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		yaml.Unmarshal(data, &mr)
	}
	if mr.Title == "" {
		mr.Title = "chore(" + st.Role + "): what the " + st.Role + " proposes"
	}
	diff, err := git(st.Repo, nil, append([]string{"diff", "--"}, st.Written...)...)
	if err != nil {
		return verdict.Finding{}, err
	}
	from, err := f.MergeRequest(st.Target.ID)
	if err != nil {
		return verdict.Finding{}, err
	}
	branch, here := from.Branch, from.Here
	why := "it comes from a fork, where this job cannot push"
	if here {
		why, err = commitOnto(st, branch, mr.Title, forge.KeepsBranches(f))
		if err != nil {
			return verdict.Finding{}, err
		}
		if why == "" {
			return verdict.Finding{Rule: "pushed", Level: "warn", Where: branch,
				Message: fmt.Sprintf("the patches were committed to %s (%q)", branch, mr.Title)}, nil
		}
	}
	body := fmt.Sprintf("**Proposed by workline's %s**, not pushed: %s. Apply it with `git apply`:\n\n```diff\n%s\n```", st.Role, why, diff)
	if err := f.Sticky(*st.Target, body, forge.Marker("sticky="+st.Role+"/patch"), true); err != nil {
		return verdict.Finding{}, err
	}
	return verdict.Finding{Rule: "patch-as-comment", Level: "warn", Message: "the patches went in a comment: " + why}, nil
}

// commitOnto commits the written files on top of branch, fetched from
// origin, and pushes it, without force; on a forge kept in the clone, on top
// of the local branch, moved only if it did not move meanwhile. It says why
// when it could not: the patches do not apply there, or the branch moved on.
func commitOnto(st runState, branch, title string, local bool) (why string, err error) {
	orig, err := git(st.Repo, nil, "symbolic-ref", "-q", "--short", "HEAD") // back on the branch it was on
	if err != nil {
		if orig, err = git(st.Repo, nil, "rev-parse", "HEAD"); err != nil { // or where CI left it
			return "", err
		}
	}
	back := func() {
		git(st.Repo, nil, append([]string{"reset", "-q", "--"}, st.Written...)...)
		git(st.Repo, nil, "checkout", "-q", orig)
	}
	tip := "FETCH_HEAD"
	if local {
		if tip, err = git(st.Repo, nil, "rev-parse", "--verify", "refs/heads/"+branch); err != nil {
			return "", fmt.Errorf("the merge request's branch %s is not in this clone", branch)
		}
	} else if _, err := git(st.Repo, nil, "fetch", "-q", "origin", branch); err != nil {
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	// The patches ride along to the branch's tip; git refuses if they collide there.
	if _, err := git(st.Repo, nil, "checkout", "-q", "--detach", tip); err != nil {
		back()
		return "the docs changed on the branch since this run read them", nil
	}
	for _, s := range [][]string{append([]string{"add", "--"}, st.Written...), {"commit", "-q", "-m", title, "-m", st.trailers()}} {
		if _, err := git(st.Repo, nil, s...); err != nil {
			back()
			return "", err
		}
	}
	move := []string{"push", "-q", "origin", "HEAD:refs/heads/" + branch}
	if local {
		move = []string{"update-ref", "refs/heads/" + branch, "HEAD", tip}
	}
	if _, err := git(st.Repo, nil, move...); err != nil {
		back()
		return "the branch moved on while this run worked", nil
	}
	_, err = git(st.Repo, nil, "checkout", "-q", orig)
	return "", err
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

// branchPrefix is where a role's merge requests come from (ADR-0006).
func branchPrefix(roleName string) string { return "workline/" + roleName + "/" }

// openMergeRequest commits what the run's patches wrote on the role's branch
// for this task, force-pushes it, opens its merge request or updates the one
// open, and puts the working tree back on the branch it was on. The role
// names the task's key and the title in out/merge-request.yaml.
func openMergeRequest(f forge.Forge, st runState, runDir string) (int, error) {
	var mr struct{ Key, Title, Body string }
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &mr); err != nil {
			return 0, fmt.Errorf("out/merge-request.yaml: %w", err)
		}
	}
	if mr.Key == "" {
		mr.Key = st.Role
	}
	if mr.Title == "" {
		mr.Title = "chore(" + st.Role + "): what the " + st.Role + " proposes"
	}
	return ProposeBranch(f, st.Repo, st.Role, mr.Key, mr.Title, mr.Body, st.trailers(), st.Written)
}

// ProposeBranch commits the files written in the working tree on the role's
// branch for a task (workline/<role>/<key>), force-pushes it — unless the
// forge keeps its branches in the clone — opens its merge request or
// updates the one open, and puts the working tree back on the branch it was
// on. trailers end the commit's message.
func ProposeBranch(f forge.Forge, repo, roleName, key, title, body, trailers string, written []string) (int, error) {
	base, err := git(repo, nil, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return 0, errors.New("a merge request is opened from a branch; this run is on a detached HEAD")
	}
	branch := branchPrefix(roleName) + slugify(key)
	steps := [][]string{
		{"checkout", "-q", "-B", branch},
		append([]string{"add", "--"}, written...),
		{"commit", "-q", "-m", title, "-m", trailers},
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
	return f.OpenMergeRequest(branch, base, title, body)
}

// slugify keeps a branch name to lowercase letters, digits and dashes.
func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

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
		case errors.Is(err, errTokensSpent):
			a.askedAgent = false // not asked: no answer to refuse, none to ask again
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
	// is what is applied, held to the same catalogue and bounds.
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
	a.verdict = v
	return a, nil
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
}

func (a *applier) marker() string { return forge.Marker(fmt.Sprintf("run=%s/%d", a.runID, a.index)) }

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
		_, err = fmt.Fprintf(f, "%s\n\n", strings.TrimSpace(fmt.Sprint(in.Value)))
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
		}
		if target == nil {
			return errors.New("a comment needs a target issue or merge request (--target)")
		}
		if sticky {
			key, _ := m["sticky"].(string)
			if key == "" {
				return errors.New("a sticky comment needs a key: {body, sticky: key}")
			}
			only, _ := m["update-only"].(bool)
			return a.forge.Sticky(*target, body, forge.Marker("sticky="+a.role+"/"+key), !only)
		}
		return a.forge.Comment(*target, body, a.marker())
	case "label":
		m, _ := in.Value.(map[string]any)
		if err := a.needForge("label"); err != nil {
			return err
		}
		if a.target == nil {
			return errors.New("a label needs a target issue or merge request (--target)")
		}
		return a.forge.Label(*a.target, strs(m["add"]), strs(m["remove"]))
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
			_, err := a.forge.KeepIssue(title, body+fmt.Sprintf("\n\nKept up to date by the %s role.", a.role), !only)
			return err
		}
		if err := a.needForge("issue"); err != nil {
			return err
		}
		return a.openIssue(title, body, m)
	case "close":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil // proposed in the report, or dropped with a finding
		}
		is := forge.Target{Kind: "issue", ID: d.Act.Issue}
		switch {
		case d.Act.Announce:
			return a.announce(d.Act)
		case d.Act.Announced != "":
			// Closed after its announcement (ADR-0024): the label goes, so a
			// reopening reads as the announcement kept open.
			if err := a.forge.Comment(is, backlog.ClosingComment(d.Act, a.role), a.marker()); err != nil {
				return err
			}
			if err := a.forge.(forge.Backlog).Close(d.Act.Issue, 0); err != nil {
				return err
			}
			return a.forge.Label(is, nil, []string{backlog.LabelObsolete})
		}
		if err := a.forge.Comment(is, backlog.Comment(d.Act, a.role), a.marker()); err != nil {
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
		outcome, id, err := o.Open(backlog.Opening{Role: a.role, Key: backlog.ImportKey(q), Title: d.Act.Title, Body: body, From: " from " + where, Commit: head})
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
	case "order":
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		return order(a.forge, a.role, d.Act)
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
		// Back to refine, a person having ticked it (ADR-0032): the issue
		// told why, its labels moved.
		d := a.plan.Decision(a.index)
		if d == nil || d.Mode != backlog.Act {
			return nil
		}
		t := forge.Target{Kind: "issue", ID: d.Act.Issue}
		say := fmt.Sprintf("Moved back to refine: %s\n\nBy the %s role, %s having ticked it in the report. Put the label %s back to undo.",
			strings.TrimSpace(d.Act.Why), a.role, d.Act.Ticked, backlog.LabelReady)
		if err := a.forge.Comment(t, say, a.marker()); err != nil {
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
	case "handoff":
		a.handoffs = append(a.handoffs, in.Value)
		return intent.Write(filepath.Join(a.runDir, "out", "handoffs.yaml"), []intent.Intention{in})
	}
	return fmt.Errorf("not implemented yet in this engine")
}

// announce tells an issue it looks obsolete (ADR-0024): the label first —
// a comment without it reads as kept open, never the other way round —,
// then the comment to its reporter, with the block read back at the next
// runs.
func (a *applier) announce(c backlog.Proposal) error {
	b := a.forge.(forge.Backlog)
	is, err := a.forge.Issue(c.Issue)
	if err != nil {
		return err
	}
	head, err := git(a.repo, nil, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	if err := b.EnsureLabel(backlog.LabelObsolete, "cfd3d7", "Announced obsolete: closed after a delay unless someone writes or takes this label off"); err != nil {
		return err
	}
	if err := a.forge.Label(t, []string{backlog.LabelObsolete}, nil); err != nil {
		return err
	}
	days := backlog.Settings(a.settings)["close-obsolete"].Delay()
	ann := backlog.Announcement{Quote: *c.Quote, Why: c.Why, Commit: head,
		Announced: time.Now().Format("2006-01-02"), By: a.model}
	return a.forge.Comment(t, backlog.AnnouncementComment(is.Author, c, a.role, ann, days), backlog.AnnounceMarker(a.role))
}

// keepOpen settles an issue's announcement as obsolete: kept open, its
// label taken off, its evidence recorded in the state so it is not
// announced again for it, and the issue told why when no person kept it.
func keepOpen(f forge.Forge, role string, c backlog.Proposal) error {
	b := f.(forge.Backlog)
	t := forge.Target{Kind: "issue", ID: c.Issue}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	key := backlog.ObsoleteKey(*c.Quote)
	if !slices.Contains(st.Kept, key) {
		st.Kept = append(st.Kept, key)
	}
	if err := f.Label(t, nil, []string{backlog.LabelObsolete}); err != nil {
		return err
	}
	if err := f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(role), false); err != nil {
		return err
	}
	if !c.Say {
		return nil
	}
	return f.Comment(t, backlog.KeptComment(c.Why), backlog.KeptMarker(role))
}

// openings is the run's one way to open an issue (backlog.Openings), made
// on the first one; the role's issues-max caps its findings opened.
func (a *applier) openings() (*backlog.Openings, error) {
	if a.opens == nil {
		max := 3
		switch n := a.settings["issues-max"].(type) {
		case int:
			max = n
		case float64:
			max = int(n)
		}
		o, err := backlog.NewOpenings(a.forge, max)
		if err != nil {
			return nil, err
		}
		a.opens = o
	}
	return a.opens, nil
}

// openIssue opens an issue a role found outside its task, once, through
// the one way every role opens one (ADR-0018). Its subject is keyed by the
// engine, never by the agent: the line of code it quotes (`at`), found
// again here, or else its title.
func (a *applier) openIssue(title, body string, m map[string]any) error {
	o, err := a.openings()
	if err != nil {
		return err
	}
	head, err := git(a.repo, nil, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	op := backlog.Opening{Role: a.role, Title: title, Body: body, Key: backlog.TitleKey(title), Sources: strs(m["sources"]), Commit: head, Triage: true}
	if at, ok := m["at"].(map[string]any); ok {
		path, _ := at["path"].(string)
		text, _ := at["text"].(string)
		if _, _, original, found := backlog.Locate(a.repo, path, text); found {
			line, _, _ := strings.Cut(original, "\n")
			op.Key = backlog.CodeKey(path, line)
			if !slices.Contains(op.Sources, path) {
				op.Sources = append(op.Sources, path)
			}
		} else {
			// Keyed by its title instead: another role finding the same
			// line would not meet it, so it is said.
			a.findings = append(a.findings, verdict.Finding{Rule: "issue-quote-not-found", Level: "warn", Where: path,
				Message: fmt.Sprintf("%q: the line it quotes is not found in %s; keyed by its title, a duplicate on that line not seen", title, path)})
		}
	}
	// The form a role's key had before every role shared one: its name first.
	op.Also = []string{"issue=" + a.role + "/" + strings.TrimPrefix(op.Key, "issue=")}
	outcome, id, err := o.Open(op)
	if err != nil {
		return err
	}
	switch outcome {
	case backlog.Capped:
		a.capped++
	case backlog.Settled, backlog.FoundAgain:
		a.findings = append(a.findings, verdict.Finding{Rule: "issue-closed", Level: "warn", Where: fmt.Sprintf("#%d", id),
			Message: fmt.Sprintf("%q: #%d, closed, holds this subject; not opened again (%s)", title, id, map[string]string{
				backlog.Settled: "closed as not planned or as a duplicate: nothing written", backlog.FoundAgain: "said once on it that it was found again"}[outcome])})
	}
	return nil
}

// order sets an issue's one priority label, the others taken off, and
// records it in the issue's state: a priority other than the one recorded
// is a person's (docs/spec/backlog-acts.md, "Ordering").
func order(f forge.Forge, role string, c backlog.Proposal) error {
	b := f.(forge.Backlog)
	label := backlog.PriorityLabel(c.Priority)
	if err := b.EnsureLabel(label, priorityColors[c.Priority], fmt.Sprintf("Priority %d of %d, 1 the most pressing: set by the %s or a person", c.Priority, backlog.Levels, strings.ReplaceAll(role, "-", " "))); err != nil {
		return err
	}
	var others []string
	for n := 1; n <= backlog.Levels; n++ {
		if n != c.Priority {
			others = append(others, backlog.PriorityLabel(n))
		}
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	if err := f.Label(t, []string{label}, others); err != nil {
		return err
	}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	st.Priority = c.Priority
	return f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(role), false)
}

// rename sets an issue's title and records it in its state: a title other
// than the one recorded is a person's (ADR-0022).
func rename(f forge.Forge, role string, c backlog.Proposal) error {
	b := f.(forge.Backlog)
	title := strings.TrimSpace(c.Title)
	if err := b.SetTitle(c.Issue, title); err != nil {
		return err
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	st.Title = title
	return f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(role), false)
}

// split opens a need's children through the one way, each with its four
// sections, links them to it — a sub-issue where the forge has them, a task
// list in its body elsewhere — and records them in its state, last: a run
// stopped half-way, resumed, finds the children it opened (ADR-0022).
func (a *applier) split(c backlog.Proposal) error {
	b := a.forge.(forge.Backlog)
	o, err := a.openings()
	if err != nil {
		return err
	}
	head, err := git(a.repo, nil, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	if err := b.EnsureLabel(backlog.LabelAccepted, "0e8a16", "A person accepted the product owner's drafts: the next run moves the issue to ready"); err != nil {
		return err
	}
	var ids, listed []int
	closed := map[int]bool{} // children closed already: left as they are
	for _, ch := range c.Into {
		outcome, id, err := o.Open(backlog.Opening{Role: a.role, Key: backlog.SplitKey(c.Issue, ch.Title), Title: strings.TrimSpace(ch.Title),
			Body: backlog.ChildBody(c.Issue, ch, a.role), From: fmt.Sprintf(" from #%d", c.Issue), Sources: ch.Sources, Commit: head})
		if err != nil {
			return err
		}
		if outcome == backlog.Settled {
			a.findings = append(a.findings, verdict.Finding{Rule: "issue-closed", Level: "warn", Where: fmt.Sprintf("#%d", id),
				Message: fmt.Sprintf("%q, a part of #%d: #%d, closed, holds it; not opened again", ch.Title, c.Issue, id)})
			ids = append(ids, id) // still one of its children: the issue is not split again
			closed[id] = true
			continue
		}
		ids = append(ids, id)
		t := forge.Target{Kind: "issue", ID: id}
		if err := a.forge.Label(t, []string{backlog.LabelToRefine, backlog.LabelDraft}, nil); err != nil {
			return err
		}
		native, err := b.AddSubIssue(c.Issue, id)
		if err != nil {
			return err
		}
		if !native {
			listed = append(listed, id)
		}
	}
	// What a child waits on among its siblings (ADR-0028): opened, never
	// read yet, so its state's digest is left.
	for i, ch := range c.Into {
		var blockers []int
		for _, k := range ch.After {
			blockers = append(blockers, ids[k-1])
		}
		if len(blockers) > 0 && !closed[ids[i]] {
			if err := a.depend(ids[i], blockers, false); err != nil {
				return err
			}
		}
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	body := ""
	if len(listed) > 0 {
		is, err := a.forge.Issue(c.Issue)
		if err != nil {
			return err
		}
		if body = backlog.ListChildren(is.Body, listed); body != is.Body {
			if err := b.SetBody(c.Issue, body); err != nil {
				return err
			}
		}
	}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, a.role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	st.Split = ids
	if body != "" {
		st.Keep(body) // the engine's own change: not read again for it
	}
	return a.forge.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(a.role), false)
}

// depend records what an issue waits on (ADR-0028): the forge's own
// relation where it has one, else the engine's line in its body, rewritten
// with them; with digest, the issue's state gets the body's digest, so the
// engine's own change is not read as a person's.
func (a *applier) depend(id int, blockers []int, digest bool) error {
	b := a.forge.(forge.Backlog)
	var inBody []int
	for _, bl := range blockers {
		native, err := b.AddBlocker(id, bl)
		if err != nil {
			return err
		}
		if !native {
			inBody = append(inBody, bl)
		}
	}
	if len(inBody) == 0 {
		return nil
	}
	is, err := a.forge.Issue(id)
	if err != nil {
		return err
	}
	body := backlog.WithBlockers(is.Body, inBody)
	if body == is.Body {
		return nil
	}
	if err := b.SetBody(id, body); err != nil || !digest {
		return err
	}
	t := forge.Target{Kind: "issue", ID: id}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, a.role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", id, err)
	}
	st.Keep(body)
	return a.forge.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(a.role), false)
}

// priorityColors are the priority labels' colours, the most pressing the
// warmest.
var priorityColors = map[int]string{1: "b60205", 2: "d93f0b", 3: "fbca04", 4: "c5def5"}

// refining applies a refine, a ready or an ask, against the issue as it is
// now (docs/spec/backlog-acts.md, "Refining to ready").
func refining(f forge.Forge, role string, c backlog.Proposal) error {
	b := f.(forge.Backlog)
	is, err := f.Issue(c.Issue)
	if err != nil {
		return err
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	switch c.Do {
	case "ask":
		return f.Comment(t, backlog.Ask(is.Author, c.Questions, c.Round), backlog.AskMarker(role, c.Round))
	case "refine":
		if !c.ToReporter {
			break
		}
		// An outsider's issue: the text proposed to its reporter, nothing
		// written in the body until they or a person of the project agree.
		if err := b.EnsureLabel(backlog.LabelAccepted, "0e8a16", "A person accepted the product owner's drafts: the next run moves the issue to ready"); err != nil {
			return err
		}
		if err := f.Label(t, []string{backlog.LabelToRefine}, nil); err != nil {
			return err
		}
		return f.Comment(t, backlog.ProposalComment(is.Author, c, role), backlog.ProposalMarker(role, c.Round))
	case "ready":
		accepted := backlog.Accepted(*is)
		if missing := backlog.NotReady(is.Body, accepted); len(missing) > 0 {
			return nil // changed since the plan: it stays to refine
		}
		if accepted && strings.Contains(is.Body, backlog.DraftMarker) {
			// The drafts are the person's: their lines go.
			body := backlog.StripDrafts(is.Body)
			if err := b.SetBody(c.Issue, body); err != nil {
				return err
			}
			if err := keepBody(f, role, t, body, nil); err != nil {
				return err
			}
		}
		return f.Label(t, []string{backlog.LabelReady}, []string{backlog.LabelToRefine, backlog.LabelDraft, backlog.LabelAccepted})
	}
	body, added, _ := backlog.Refine(is.Body, c, role)
	if len(added) > 0 {
		if err := b.SetBody(c.Issue, body); err != nil {
			return err
		}
	}
	labels := []string{backlog.LabelToRefine}
	if slices.Contains(added, "Need") || slices.Contains(added, "Validation") {
		labels = append(labels, backlog.LabelDraft)
		// The label a person accepts the drafts with, there to be picked
		// from the forge's list.
		if err := b.EnsureLabel(backlog.LabelAccepted, "0e8a16", "A person accepted the product owner's drafts: the next run moves the issue to ready"); err != nil {
			return err
		}
	}
	if err := f.Label(t, labels, nil); err != nil {
		return err
	}
	return keepBody(f, role, t, body, c.Sources)
}

// keepBody records in an issue's state the body the engine left, so only a
// person's change has it read again, and the files its scope names when it
// had none.
func keepBody(f forge.Forge, role string, t forge.Target, body string, sources []string) error {
	b := f.(forge.Backlog)
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	if len(st.Sources) == 0 {
		st.Sources = sources
	}
	st.Keep(body)
	return f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(role), false)
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
	files, err := git(a.repo, strings.NewReader(diff), "apply", "--recount", "--unidiff-zero", "--numstat", "-")
	if err != nil {
		return fmt.Errorf("unreadable diff: %w", err)
	}
	var touched []string
	for _, line := range strings.Split(strings.TrimSpace(files), "\n") {
		if f := strings.Fields(line); len(f) == 3 {
			if !a.allowed(f[2]) {
				return fmt.Errorf("%s is outside what this role may write", f[2])
			}
			touched = append(touched, f[2])
		}
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
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(errOut.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// keepRuns is how many past runs are kept for inspection.
const keepRuns = 50

// newRunDir creates the run's folder inside the repository's git directory,
// where it is never tracked and never shows up as a change.
func newRunDir(repo, roleName string) (string, error) {
	base := filepath.Join(repo, ".workline", "runs")
	if out, err := exec.Command("git", "-C", repo, "rev-parse", "--absolute-git-dir").Output(); err == nil {
		base = filepath.Join(strings.TrimSpace(string(out)), "workline", "runs")
	}
	if d := os.Getenv("WORKLINE_RUNS_DIR"); d != "" {
		base = d // in CI, somewhere an artifact can carry it to the job that applies
	}
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

// releaseRequest runs a role that has release duties on `release` when the
// merge request a run targets is a release tool's, told by its branch
// (ADR-0017): that pull request is the release, merged before anything is
// tagged. It returns the branch, "" when it is not a release's. When the
// branch cannot be learnt, it says so and the run stays a merge request's.
func releaseRequest(r *role.Role, o *Options, res *Result) string {
	if o.Event != "merge-request" || !r.Accepts("release") {
		return ""
	}
	rs, err := release.Load(o.Repo)
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "release-unknown", Level: "warn", Message: err.Error()})
		return ""
	}
	branch := o.Branch
	if branch == "" && o.Target != nil && o.Target.Kind == "merge-request" {
		f, err := forge.Open(o.Forge, o.Repo)
		if err == nil && f != nil {
			var mr forge.MergeRequest
			mr, err = f.MergeRequest(o.Target.ID)
			branch = mr.Branch
		}
		if err != nil {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "release-unknown", Level: "warn",
				Message: fmt.Sprintf("whether %s comes from a release tool's branch is unknown, so it is not held as the release: %v; give its branch with --branch", o.Target, err)})
			return ""
		}
	}
	if !rs.IsBranch(branch) {
		return ""
	}
	o.Event = "release"
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
	ans, err := judge.Ask(spec, judgeRole, q.Question, q.Material)
	res.Calls = append(res.Calls, ans.Call) // its tokens count too; it is not the role's agent
	level := judge.Independence(spec, ans.Model, o.AI, authorModels)
	who := fmt.Sprintf("independence: %s (%s → %s)", level, last, ans.Model)
	switch {
	case err != nil:
		a.external = true
		v.Status, v.Summary = verdict.Block, "the judge could not answer"
		v.Findings = append(v.Findings, verdict.Finding{Rule: "judge-unavailable", Message: err.Error() + "; " + who})
	case !ans.Yes:
		v.Status, v.Summary = verdict.Block, "the judge refused the proposal"
		v.Findings = append(v.Findings, verdict.Finding{Rule: "judged-no", Message: ans.Why + " (" + who + ")"})
	default:
		v.Findings = append(v.Findings, verdict.Finding{Rule: "judged", Level: "warn", Message: "yes: " + ans.Why + " (" + who + ")"})
	}
	return nil
}

// askQuestions puts each question pre wrote (in/judge/<key>/question.yaml:
// {question, material}) to a judge, in a context of its own, at the best
// independence available from the agents that answered the run (ADR-0005),
// and writes its answer beside it (answer.yaml: {yes, why, model, judge,
// level}, or {error} when it could not answer). Without an agent, or once
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
		var q struct{ Question, Material, Author string }
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
			ans, err := judge.Ask(spec, judgeRole, q.Question, q.Material)
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
			answer = map[string]any{"yes": ans.Yes, "why": ans.Why, "model": ans.Model, "judge": spec,
				"author": last, "level": judge.Independence(spec, ans.Model, o.AI, authorModels)}
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

// planActs decides what becomes of the run's acts on the backlog, once: a
// resumed run reads the plan the first attempt wrote, as the forge it reads
// has changed since. Nil when the run proposes none.
func planActs(f forge.Forge, r *role.Role, settings map[string]any, st runState, runDir string, intents []intent.Intention) (*backlog.Plan, error) {
	role := r.Name
	closes := map[int]backlog.Proposal{}
	for i, in := range intents {
		if !slices.Contains(backlog.Kinds, in.Kind) {
			continue
		}
		var c backlog.Proposal
		data, _ := yaml.Marshal(in.Value)
		if err := yaml.Unmarshal(data, &c); err != nil {
			c = backlog.Proposal{Reason: "unreadable: " + err.Error()}
		}
		c.Do = in.Kind
		closes[i] = c
	}
	// A role keeping a backlog reads its record on every run, acts or not:
	// a closing a person undid is found at the next run, whatever it does.
	if len(closes) == 0 && !slices.ContainsFunc(r.Intentions, func(k string) bool { return slices.Contains(backlog.Kinds, k) }) {
		return nil, nil
	}
	if len(closes) == 0 && f == nil {
		return nil, nil
	}
	file := filepath.Join(runDir, "out", "acts.yaml")
	if data, err := os.ReadFile(file); err == nil {
		var p backlog.Plan
		return &p, yaml.Unmarshal(data, &p)
	}
	if f == nil {
		return nil, fmt.Errorf("an act on an issue needs a forge; %s", forge.Missing)
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, errors.New("this forge cannot list or close issues")
	}
	var read []int
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "issues-read")); err == nil {
		for _, f := range strings.Fields(string(data)) {
			if n, err := strconv.Atoi(f); err == nil {
				read = append(read, n)
			}
		}
	}
	// The second judge's answers on the issues announced obsolete, as the
	// engine wrote them beside pre's questions (ADR-0024).
	judged := map[int]backlog.Judged{}
	answers, _ := filepath.Glob(filepath.Join(runDir, "in", "judge", backlog.JudgeKeyPrefix+"*", "answer.yaml"))
	for _, file := range answers {
		id, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(filepath.Dir(file)), backlog.JudgeKeyPrefix))
		if err != nil {
			continue
		}
		var j backlog.Judged
		if data, err := os.ReadFile(file); err == nil && yaml.Unmarshal(data, &j) == nil {
			judged[id] = j
		}
	}
	cfg, err := backlog.ReadConfig(settings)
	if err != nil {
		return nil, err
	}
	// The issues pre found waiting on a person, for the report's opening
	// (ADR-0031).
	var waits []backlog.Wait
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "waits.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &waits); err != nil {
			return nil, fmt.Errorf("in/waits.yaml: %v", err)
		}
	}
	// What open issues were built on that changed, and those read again
	// for it (ADR-0032).
	var changes []backlog.Change
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "changes.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &changes); err != nil {
			return nil, fmt.Errorf("in/changes.yaml: %v", err)
		}
	}
	p, err := backlog.Decide(b, st.Repo, role, cfg, closes, read, judged, waits, changes)
	if err != nil {
		return nil, err
	}
	data, _ := yaml.Marshal(p)
	return p, os.WriteFile(file, data, 0o644)
}

// report writes the role's report issue, when the run did or proposes
// something, or found a closing wrong: its body what the run did, its
// record comment what the role did so far.
func report(f forge.Forge, role string, p *backlog.Plan) error {
	if !p.Changed && !slices.ContainsFunc(p.Decisions, func(d backlog.Decision) bool { return d.Mode != backlog.Off }) {
		return nil
	}
	id, err := f.KeepIssue(backlog.ReportTitle(role), p.ReportBody(), true)
	if err != nil || id == 0 {
		return err
	}
	return f.Sticky(forge.Target{Kind: "issue", ID: id}, backlog.FormatRecord(p.Record), backlog.RecordMarker(role), true)
}
