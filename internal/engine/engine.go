// Package engine runs one role once: check, prepare, propose, judge, apply
// (docs/spec/role-contract.md, "One run").
package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/JN0V/assembly-line/internal/agent"
	"github.com/JN0V/assembly-line/internal/intent"
	"github.com/JN0V/assembly-line/internal/role"
	"github.com/JN0V/assembly-line/internal/verdict"
)

// Options describe one run.
type Options struct {
	Repo     string            // repository the role works on
	RolesDir string            // folder holding the roles
	Role     string            // role name
	Event    string            // event the role runs on
	AI       string            // agent spec, see agent.Parse
	Inputs   map[string]string // name -> value, written to in/input/<name>
	Targets  map[string]string // name -> file an intention writes back to (e.g. the hook's message file)

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
		if err := ag.Propose(runDir); err != nil {
			if !errors.Is(err, agent.ErrUnavailable) {
				return err
			}
			agentExternal = true
			res.Findings = append(res.Findings, verdict.Finding{Rule: "agent-unavailable", Message: err.Error()})
		}
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return err
	}
	if refused := invalid(r, intents); len(refused) > 0 {
		res.Refused = intent.Kinds(intents)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "intention-refused", Message: fmt.Sprintf("not allowed for this role: %v", refused)})
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
	for _, in := range intents {
		if err := apply(in, runDir, o.Targets); err != nil {
			return fmt.Errorf("apply %s: %w", in.Kind, err)
		}
		res.Applied = append(res.Applied, in.Kind)
	}
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

// invalid returns the kinds that are not in the catalogue or not allowed for the role.
func invalid(r *role.Role, in []intent.Intention) []string {
	var bad []string
	for _, i := range in {
		if !intent.Catalogue[i.Kind] || !r.Allows(i.Kind) {
			bad = append(bad, i.Kind)
		}
	}
	return bad
}

// apply carries out one intention. This version knows the local ones.
func apply(in intent.Intention, runDir string, targets map[string]string) error {
	switch in.Kind {
	case "commit-message":
		msg, ok := in.Value.(string)
		if !ok {
			return errors.New("value must be text")
		}
		if err := os.WriteFile(filepath.Join(runDir, "out", "commit-message"), []byte(msg), 0o644); err != nil {
			return err
		}
		if t := targets["message"]; t != "" {
			return os.WriteFile(t, []byte(msg+"\n"), 0o644)
		}
		return nil
	case "note":
		return os.WriteFile(filepath.Join(runDir, "out", "note.md"), []byte(fmt.Sprint(in.Value)), 0o644)
	}
	return fmt.Errorf("not implemented yet in this engine")
}

func newRunDir(repo, roleName string) (string, error) {
	id := fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102T150405.000000000"), roleName)
	dir := filepath.Join(repo, ".assembly", "runs", id)
	for _, d := range []string{"in/input", "out"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return "", err
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
		"ASSEMBLY_RUN_DIR="+runDir,
		"ASSEMBLY_EVENT="+o.Event,
		"ASSEMBLY_AI="+ai,
		"ASSEMBLY_ROLE="+roleName,
		"ASSEMBLY_BIN="+self,
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
