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
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/pathglob"
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
	Scope  []string      // paths the task is about; a patch outside is refused
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
	Status       string            `json:"status"`
	Summary      string            `json:"summary,omitempty"`
	Findings     []verdict.Finding `json:"findings,omitempty"`
	AgentCalls   int               `json:"agent-calls"`
	Calls        []agent.Call      `json:"calls,omitempty"` // each call: what was asked, what answered
	Applied      []string          `json:"applied"`
	Refused      []string          `json:"refused"`
	Handoffs     []any             `json:"handoffs,omitempty"` // next roles asked for; routing runs them
	Notes        []string          `json:"notes,omitempty"`    // what the agent left for a person: a round's notes each
	RunDir       string            `json:"run-dir"`
	ToApply      bool              `json:"to-apply,omitempty"`      // judged with NoApply: `workline apply` still has work
	MergeRequest int               `json:"merge-request,omitempty"` // the merge request the patches went to
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
		res.Findings = nil
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
		if err != nil || res.Status != verdict.Pass || o.NoApply || !changedSomething(res.Applied[applied:]) || round == maxRounds {
			return res
		}
	}
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
	if o.Forge == "" {
		o.Forge = cfg.Forge
	}
	env := scriptEnv(runDir, r.Name, o)
	if o.OpenMergeRequest {
		// The role decides what it proposes when enough of its merge requests wait.
		f, err := forge.Open(o.Forge, o.Repo)
		if err != nil {
			return err
		}
		if f == nil {
			return errors.New("--open-merge-request needs a forge (--forge)")
		}
		n, err := f.OpenMergeRequests(branchPrefix(r.Name))
		if err != nil {
			return err
		}
		env = append(env, fmt.Sprintf("WORKLINE_OPEN_MERGE_REQUESTS=%d", n))
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
		if !a.askedAgent || v.Status != verdict.Block {
			break
		}
		failures++
		var retry bool
		if retry, tier = askAgain(failures, r.Model.PromoteAfter, tier); !retry {
			break
		}
		var fb strings.Builder
		for _, f := range v.Findings {
			fmt.Fprintf(&fb, "- %s: %s\n", f.Rule, f.Message)
		}
		if err := os.WriteFile(filepath.Join(runDir, "out", "feedback.md"), []byte(fb.String()), 0o644); err != nil {
			return err
		}
		// The refused answer is kept beside its refusal, to be studied.
		os.Rename(filepath.Join(runDir, "out", "intentions.yaml"), filepath.Join(runDir, "out", fmt.Sprintf("refused-%d.yaml", failures)))
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
				intents, v = kept, nv
				v.Findings = append(left, v.Findings...)
			}
		}
	}
	res.Status, res.Summary = v.Status, v.Summary
	res.Findings = append(res.Findings, v.Findings...)
	if res.Status != verdict.Pass || len(intents) == 0 {
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
	intent.SortForApply(intents)
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return err
	}
	if o.Forge == "" {
		o.Forge = cfg.Forge
	}
	st := runState{Role: r.Name, RolesDir: o.RolesDir, Repo: o.Repo, Forge: o.Forge, Target: o.Target, Scope: o.Scope, Digest: digest, Targets: o.Targets,
		OpenMergeRequest: o.OpenMergeRequest, PushToMergeRequest: o.PushToMergeRequest}
	if err := st.save(runDir); err != nil {
		return err
	}
	if o.NoApply {
		res.Summary = fmt.Sprintf("judged, not applied (%d proposals); apply with: workline apply %s", len(intents), runDir)
		res.ToApply = true
		for _, in := range intents {
			if in.Kind == "handoff" {
				res.Findings = append(res.Findings, deferred(r.Name, in.Value))
			}
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
		return true, nil
	}
	return true, fmt.Errorf("pre exited with code %d", code)
}

// askParts asks each part of a question (in/parts/<name>/task.md), sorted by
// name, in a context of its own, on the tier model.tasks.part names, and puts
// each answer back as in/parts/<name>/answer.yaml. A part answers only
// claims, which inform and are never applied: a part that fails, or answers
// anything else, is said, and its answer never given back as one. Without an
// agent, no part is asked.
func askParts(r *role.Role, o Options, ag agent.Agent, runDir string, tasks []string, res *Result) error {
	if ag == nil {
		return nil
	}
	sort.Strings(tasks)
	asked := *r // a part's needs, and the one intention it may answer with
	asked.Model = r.Model.For("part")
	asked.Intentions = []string{"claim"}
	gone := "" // the agent could not be reached: the other parts would fail the same
	for _, task := range tasks {
		name := filepath.Base(filepath.Dir(task))
		unanswered := func(why string) {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "part-unanswered", Where: name, Level: "warn",
				Message: "this part of the question got no answer that can be read (" + why + "): what it holds was judged by no one"})
		}
		if gone != "" {
			unanswered("not asked: " + gone)
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
		err = callAgent(ag, agent.Request{RunDir: dir, Repo: o.Repo, Role: &asked, Tier: asked.Model.Tier}, "part", runDir, res)
		switch {
		case errors.Is(err, agent.ErrUnavailable):
			gone = err.Error()
			unanswered(gone)
			continue
		case errors.Is(err, agent.ErrInvalidOutput):
			// One item written wrong spoils the whole list for a YAML
			// reader: the items are read one by one, the broken ones kept
			// as unreadable claims, which the role counts as dropped.
			raw, _ := os.ReadFile(filepath.Join(dir, "out", "agent-answer.txt"))
			read, broken := claimsOneByOne(string(raw))
			if read == 0 {
				unanswered(err.Error())
				continue
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: "part-partly-read", Where: name, Level: "warn",
				Message: fmt.Sprintf("the answer to this part was not valid as a whole: %d claims read one by one, %d that could not be read counted as dropped", read, len(broken))})
			if err := intent.Write(filepath.Join(dir, "out", "intentions.yaml"), append(claimsRead(string(raw)), broken...)); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		claims, err := intent.Read(filepath.Join(dir, "out", "intentions.yaml"))
		if err != nil {
			unanswered(err.Error())
			continue
		}
		var other []string
		for _, c := range claims {
			if c.Kind != "claim" {
				other = append(other, c.Kind)
			}
		}
		if len(other) > 0 {
			unanswered("it answered " + strings.Join(other, ", ") + "; a part answers with claims only")
			continue
		}
		answer := []byte("[]\n") // nothing to say of this share is an answer too
		if len(claims) > 0 {
			list := make([]map[string]any, len(claims))
			for i, c := range claims {
				list[i] = map[string]any{c.Kind: c.Value}
			}
			if answer, err = yaml.Marshal(list); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(task), "answer.yaml"), answer, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// answerItems cuts an answer into its top-level list items: each starts
// with "- " at the start of a line and runs to the next.
func answerItems(answer string) []string {
	var items []string
	for _, l := range strings.Split(answer, "\n") {
		switch {
		case strings.HasPrefix(l, "- "):
			items = append(items, l+"\n")
		case len(items) > 0 && (strings.HasPrefix(l, " ") || l == ""):
			items[len(items)-1] += l + "\n"
		}
	}
	return items
}

// claimsRead are the items of an answer that read as a claim each.
func claimsRead(answer string) []intent.Intention {
	var out []intent.Intention
	for _, item := range answerItems(answer) {
		var one []map[string]any
		if yaml.Unmarshal([]byte(item), &one) == nil && len(one) == 1 && len(one[0]) == 1 && one[0]["claim"] != nil {
			out = append(out, intent.Intention{Kind: "claim", Value: one[0]["claim"]})
		}
	}
	return out
}

// claimsOneByOne counts the items that read as claims, and returns the
// others as claims with nothing to check, which the role drops.
func claimsOneByOne(answer string) (int, []intent.Intention) {
	read := len(claimsRead(answer))
	var broken []intent.Intention
	for _, item := range answerItems(answer) {
		var one []map[string]any
		if yaml.Unmarshal([]byte(item), &one) != nil || len(one) != 1 || len(one[0]) != 1 || one[0]["claim"] == nil {
			broken = append(broken, intent.Intention{Kind: "claim", Value: map[string]any{"unreadable": strings.TrimSpace(item)}})
		}
	}
	return read, broken
}

// callAgent asks the agent once and records the call: in the result, and
// with the run (out/calls.jsonl), what it cost, to be read afterwards.
func callAgent(ag agent.Agent, req agent.Request, task, runDir string, res *Result) error {
	res.AgentCalls++
	start := time.Now()
	call, err := ag.Propose(req)
	call.Seconds = math.Round(time.Since(start).Seconds()*10) / 10
	call.Task = task
	res.Calls = append(res.Calls, call)
	if data, err := json.Marshal(call); err == nil {
		if f, err := os.OpenFile(filepath.Join(runDir, "out", "calls.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			f.Write(append(data, '\n'))
			f.Close()
		}
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
	ap := applier{repo: st.Repo, runDir: runDir, targets: st.Targets, writes: r.Writes(settings), settings: settings,
		forge: f, target: st.Target, runID: filepath.Base(runDir), role: r.Name}
	done := map[int]bool{}
	for _, i := range st.Applied {
		done[i] = true
	}
	for i, in := range intents {
		if done[i] {
			res.Applied = append(res.Applied, in.Kind)
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
		res.Applied = append(res.Applied, in.Kind)
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
	branch, here, err := f.MergeRequestBranch(st.Target.ID)
	if err != nil {
		return verdict.Finding{}, err
	}
	why := "it comes from a fork, where this job cannot push"
	if here {
		why, err = commitOnto(st, branch, mr.Title)
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
// origin, and pushes it, without force. It says why when it could not: the
// patches do not apply there, or the branch moved on meanwhile.
func commitOnto(st runState, branch, title string) (why string, err error) {
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
	if _, err := git(st.Repo, nil, "fetch", "-q", "origin", branch); err != nil {
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	// The patches ride along to the branch's tip; git refuses if they collide there.
	if _, err := git(st.Repo, nil, "checkout", "-q", "--detach", "FETCH_HEAD"); err != nil {
		back()
		return "the docs changed on the branch since this run read them", nil
	}
	for _, s := range [][]string{append([]string{"add", "--"}, st.Written...), {"commit", "-q", "-m", title, "-m", OwnTrailer + ": " + st.Role}} {
		if _, err := git(st.Repo, nil, s...); err != nil {
			back()
			return "", err
		}
	}
	if _, err := git(st.Repo, nil, "push", "-q", "origin", "HEAD:refs/heads/"+branch); err != nil {
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
	base, err := git(st.Repo, nil, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return 0, errors.New("a merge request is opened from a branch; this run is on a detached HEAD")
	}
	branch := branchPrefix(st.Role) + slugify(mr.Key)
	steps := [][]string{
		{"checkout", "-q", "-B", branch},
		append([]string{"add", "--"}, st.Written...),
		{"commit", "-q", "-m", mr.Title, "-m", OwnTrailer + ": " + st.Role},
		{"push", "-q", "--force", "origin", branch},
	}
	for _, s := range steps {
		if _, err := git(st.Repo, nil, s...); err != nil {
			// Back where it was: the patches in the working tree, unstaged.
			git(st.Repo, nil, append([]string{"reset", "-q", "--"}, st.Written...)...)
			git(st.Repo, nil, "checkout", "-q", base)
			if s[0] == "push" {
				return 0, fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
			}
			return 0, err
		}
	}
	if _, err := git(st.Repo, nil, "checkout", "-q", base); err != nil {
		return 0, err
	}
	return f.OpenMergeRequest(branch, base, mr.Title, mr.Body)
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
		case errors.Is(err, agent.ErrUnavailable):
			a.external = true
			a.findings = append(a.findings, verdict.Finding{Rule: "agent-unavailable", Message: err.Error()})
		case errors.Is(err, agent.ErrInvalidOutput):
			a.findings = append(a.findings, verdict.Finding{Rule: "agent-invalid-output", Message: err.Error()})
		default:
			return nil, err
		}
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		a.findings = append(a.findings, verdict.Finding{Rule: "agent-invalid-output", Message: err.Error()})
		intents = nil
	}
	fallback, err := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	if err != nil {
		return nil, err
	}
	intents = intent.Merge(fallback, intents)
	os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return nil, err
	}
	res.Refused = []string{}
	if refused, why := invalid(r, intents, line); len(refused) > 0 {
		res.Refused = refused // the set is refused whole; these are the kinds that caused it
		a.findings = append(a.findings, verdict.Finding{Rule: "intention-refused", Message: why})
		intents = nil
		os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	}
	if bad := outOfBounds(o.Repo, intents, r.Writes(settings), o.Scope); len(bad) > 0 {
		res.Refused = []string{"patch"}
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
	for _, i := range in {
		switch {
		case i.Kind == "claim":
			bad, why = append(bad, i.Kind), append(why, "a claim answers a part of a question (in/parts), never the question itself")
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
	writes       []string       // paths the role may write
	settings     map[string]any // the role's merged settings
	written      []string       // files changed by this run's patches
	handoffs     []any          // next roles asked for, recorded for routing
	forge        forge.Forge    // nil when the project has no forge
	target       *forge.Target
	runID, role  string
	index        int // position of the intention being applied, for its marker
}

func (a *applier) marker() string { return forge.Marker(fmt.Sprintf("run=%s/%d", a.runID, a.index)) }

func (a *applier) needForge(kind string) error {
	if a.forge == nil {
		return fmt.Errorf("a %s needs a forge; set `forge:` in .workline/config.yaml or pass --forge", kind)
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
	case "release":
		return a.release(in.Value)
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
		if a.target == nil {
			return errors.New("a comment needs a target issue or merge request (--target)")
		}
		if sticky {
			key, _ := m["sticky"].(string)
			if key == "" {
				return errors.New("a sticky comment needs a key: {body, sticky: key}")
			}
			only, _ := m["update-only"].(bool)
			return a.forge.Sticky(*a.target, body, forge.Marker("sticky="+a.role+"/"+key), !only)
		}
		return a.forge.Comment(*a.target, body, a.marker())
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
		body += fmt.Sprintf("\n\nOpened by the %s role.", a.role)
		if a.forge == nil {
			return a.localIssue(title, body)
		}
		_, err := a.forge.OpenIssue(title, body, a.marker())
		return err
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

// release commits what this run wrote, then tags it. Only the direct flow
// exists locally; the merge-request flow needs a forge.
func (a *applier) release(v any) error {
	m, ok := v.(map[string]any)
	version, _ := m["version"].(string)
	notes, _ := m["notes"].(string)
	if !ok || version == "" {
		return errors.New("expected {version, notes}")
	}
	if flow, _ := a.settings["flow"].(string); flow != "direct" {
		return fmt.Errorf("the %q flow is not built yet; set flow: direct", flow)
	}
	if len(a.written) > 0 {
		if _, err := git(a.repo, nil, append([]string{"add", "--"}, a.written...)...); err != nil {
			return err
		}
		if _, err := git(a.repo, nil, "commit", "-q", "-m", "chore(release): "+version); err != nil {
			return err
		}
	}
	head, err := git(a.repo, nil, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if tagged, err := git(a.repo, nil, "rev-parse", "-q", "--verify", version+"^{commit}"); err == nil {
		if tagged == head {
			return a.publish(version, notes) // tagged already: only the forge may be missing it
		}
		return fmt.Errorf("tag %s already exists on another commit", version)
	}
	if notes == "" {
		notes = version
	}
	if _, err = git(a.repo, nil, "tag", "-a", version, "-m", notes); err != nil {
		return err
	}
	return a.publish(version, notes)
}

// publish puts the release on the forge, when there is one.
func (a *applier) publish(version, notes string) error {
	if a.forge == nil {
		return nil
	}
	return a.forge.Release(version, notes)
}

// localIssue records an issue in .workline/issues/ for a project without a forge.
func (a *applier) localIssue(title, body string) error {
	dir := filepath.Join(a.repo, ".workline", "issues")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var slug strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			slug.WriteRune(r)
		case slug.Len() > 0 && !strings.HasSuffix(slug.String(), "-"):
			slug.WriteRune('-')
		}
	}
	path := filepath.Join(dir, strings.Trim(slug.String(), "-")+".md")
	if _, err := os.Stat(path); err == nil {
		return nil // already reported
	}
	return os.WriteFile(path, []byte("# "+title+"\n\n"+body+"\n"), 0o644)
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
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
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
