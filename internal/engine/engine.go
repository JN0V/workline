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
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
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
		res.Findings = append(res.Findings, verdict.Finding{Rule: "engine-error", Message: err.Error()})
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
		res.Status, res.Summary = verdict.Pass, "nothing to do"
		if v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml")); err == nil && v.Summary != "" {
			res.Summary = v.Summary // the role's own words on why there is nothing to do
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

	// 3. Propose — only when pre asked a question and an agent is available.
	agentExternal := false
	if _, err := os.Stat(filepath.Join(runDir, "in", "task.md")); err == nil && ag != nil {
		res.AgentCalls++
		err := ag.Propose(agent.Request{RunDir: runDir, Repo: o.Repo, Role: r})
		switch {
		case err == nil:
		case errors.Is(err, agent.ErrUnavailable):
			agentExternal = true
			res.Findings = append(res.Findings, verdict.Finding{Rule: "agent-unavailable", Message: err.Error()})
		case errors.Is(err, agent.ErrInvalidOutput):
			res.Findings = append(res.Findings, verdict.Finding{Rule: "agent-invalid-output", Message: err.Error()})
		default:
			return err
		}
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return err
	}
	fallback, err := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	if err != nil {
		return err
	}
	intents = intent.Merge(fallback, intents)
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return err
	}
	line, err := routing.Load(o.Repo)
	if err != nil {
		return err
	}
	if refused, why := invalid(r, intents, line); len(refused) > 0 {
		res.Refused = refused // the set is refused whole; these are the kinds that caused it
		res.Findings = append(res.Findings, verdict.Finding{Rule: "intention-refused", Message: why})
		intents = nil
		_ = os.Remove(filepath.Join(runDir, "out", "intentions.yaml"))
	}

	// 4. Judge.
	code, err = script(r, "post", o.Repo, env)
	if err != nil {
		return err
	}
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		return fmt.Errorf("post wrote no readable verdict: %w", err)
	}
	if want := statusForExit(code); want == "" {
		return fmt.Errorf("post exited with code %d", code)
	} else if want != v.Status {
		return fmt.Errorf("post exited %d but its verdict says %q", code, v.Status)
	}
	verdict.Enforce(v, r.Enforcement(cfg))
	res.Status, res.Summary = v.Status, v.Summary
	res.Findings = append(res.Findings, v.Findings...)
	if agentExternal && res.Status != verdict.Pass {
		res.Status = verdict.BlockedExternal
		return nil
	}
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
	settings := r.MergedSettings(cfg)
	ap := applier{repo: o.Repo, runDir: runDir, targets: o.Targets, writes: r.Writes(settings), settings: settings}
	for _, in := range intents {
		if err := ap.apply(in); err != nil {
			return fmt.Errorf("apply %s: %w", in.Kind, err)
		}
		res.Applied = append(res.Applied, in.Kind)
	}
	res.Handoffs = ap.handoffs
	return nil
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
	for _, pat := range a.writes {
		if pat == clean {
			return true
		}
		if dir, ok := strings.CutSuffix(pat, "/**"); ok && strings.HasPrefix(clean, dir+"/") {
			return true
		}
		if ok, _ := filepath.Match(pat, clean); ok {
			return true
		}
	}
	return false
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
	files, err := git(a.repo, strings.NewReader(diff), "apply", "--numstat", "-")
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
	if _, err := git(a.repo, strings.NewReader(diff), "apply", "-"); err != nil {
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
			return nil // already released: applying again changes nothing
		}
		return fmt.Errorf("tag %s already exists on another commit", version)
	}
	if notes == "" {
		notes = version
	}
	_, err = git(a.repo, nil, "tag", "-a", version, "-m", notes)
	return err
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
