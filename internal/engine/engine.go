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
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/role"
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
	Applied    []string          `json:"applied"`
	Refused    []string          `json:"refused"`
	Handoffs   []any             `json:"handoffs,omitempty"` // next roles asked for; routing runs them
	RunDir     string            `json:"run-dir"`
	ToApply    bool              `json:"to-apply,omitempty"` // judged with NoApply: `workline apply` still has work
}

// Exit codes of pre and post (docs/spec/role-contract.md, "Exit codes").
const (
	exitOK       = 0
	exitBlock    = 1
	exitHuman    = 2
	exitExternal = 3
	exitNothing  = 10
)

// Run executes one run and always returns a result with a status: an error
// inside the run becomes a blocking finding, never a silent pass.
func Run(o Options) *Result {
	res := &Result{Applied: []string{}, Refused: []string{}}
	if err := run(o, res); err != nil {
		res.Status = verdict.Block
		rule := "engine-error"
		if role.IsConfigError(err) {
			rule = "config-invalid"
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Message: err.Error()})
	}
	return res
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
	env := scriptEnv(runDir, r.Name, o)

	// 2. Prepare.
	code, err := script(r, "pre", o.Repo, env)
	if err != nil {
		return err
	}
	switch code {
	case exitOK:
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
		return nil
	case exitExternal:
		res.Status, res.Summary = verdict.BlockedExternal, "pre: an outside service failed"
		return nil
	default:
		return fmt.Errorf("pre exited with code %d", code)
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
		os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
		os.Remove(filepath.Join(runDir, "out", "verdict.yaml"))
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
	st := runState{Role: r.Name, RolesDir: o.RolesDir, Repo: o.Repo, Forge: o.Forge, Target: o.Target, Scope: o.Scope, Digest: digest, Targets: o.Targets}
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
		if err := st.save(runDir); err != nil {
			return err
		}
		res.Applied = append(res.Applied, in.Kind)
	}
	res.Handoffs = ap.handoffs
	return nil
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
		r, err := role.Load(st.RolesDir, st.Role)
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
	out, err := git(repo, strings.NewReader(diff), "apply", "--recount", "--numstat", "-")
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
		res.AgentCalls++
		err := ag.Propose(agent.Request{RunDir: runDir, Repo: o.Repo, Role: r, Tier: tier})
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
	v.Findings = append(a.findings, v.Findings...)
	a.verdict = v
	return a, nil
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
		return os.WriteFile(filepath.Join(a.runDir, "out", "note.md"), []byte(fmt.Sprint(in.Value)), 0o644)
	case "patch":
		return a.patch(in.Value)
	case "release":
		return a.release(in.Value)
	case "comment":
		body, _ := in.Value.(string)
		if err := a.needForge("comment"); err != nil {
			return err
		}
		if a.target == nil {
			return errors.New("a comment needs a target issue or merge request (--target)")
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
		body += fmt.Sprintf("\n\nFound by the %s role, outside the task it was working on.", a.role)
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
	files, err := git(a.repo, strings.NewReader(diff), "apply", "--recount", "--numstat", "-")
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
	if _, err := git(a.repo, strings.NewReader(diff), "apply", "--recount", "-"); err != nil {
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
		return fmt.Errorf("the %q flow needs a forge, which this engine cannot reach yet; set flow: direct", flow)
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
	return append(os.Environ(),
		"WORKLINE_RUN_DIR="+runDir,
		"WORKLINE_EVENT="+o.Event,
		"WORKLINE_AI="+ai,
		"WORKLINE_ROLE="+roleName,
		"WORKLINE_BIN="+self,
	)
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
