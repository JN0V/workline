package developer

// The proof and the caps (ADR-0054, step 3): before its pull request
// opens, the code within its bounds is laid on the commit the run read, in
// a checkout apart, and the project's test command runs there — the
// tester's own setting, so the acceptance tests on the branch run with the
// rest. Green, and no cap reached: the pull request opens ready. Red, not
// run whole, or a cap reached: a Draft, which wakes no agent (ADR-0050 §4),
// saying what failed. Both are kept in the claim's record, for the auditor
// (ADR-0046).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// The proof's results.
const (
	Green    = "green"
	Red      = "red"
	Unproven = "unproven" // the test command did not run whole
)

// Proof is what the project's tests said on the developer's code.
type Proof struct {
	Result  string   `json:"result"`
	Command string   `json:"command,omitempty"`
	Passed  int      `json:"passed"`
	Red     []string `json:"red,omitempty"`
	// Why: unproven, why it did not run whole; red with no test red, why
	// the command failed.
	Why string `json:"why,omitempty"`
}

// Cap is a bound on the run reached: the setting, what was spent, what it
// allows.
type Cap struct {
	Setting string `json:"setting"`
	Spent   int    `json:"spent"`
	Allows  int    `json:"allows"`
}

// prove runs the project's test command on the patches laid on the commit
// the run read, stopped past the proof's seconds: a cap then.
func prove(repo, at string, patches []intent.Intention, s Settings) (Proof, *Cap) {
	ts, err := testerSettings(repo)
	if err != nil {
		return Proof{Result: Unproven, Why: err.Error()}, nil
	}
	p := Proof{Command: ts.Command}
	if at == "" {
		at = "HEAD"
	}
	var deadline time.Time
	if s.ProofSeconds > 0 {
		deadline = time.Now().Add(time.Duration(s.ProofSeconds) * time.Second)
	}
	start := time.Now()
	o, f := tester.OnChange(repo, at, patches, ts, deadline)
	switch {
	case f != nil && f.Rule == "proof-over-time":
		p.Result, p.Why = Unproven, fmt.Sprintf("the test command ran past %d s: stopped", s.ProofSeconds)
		return p, &Cap{Setting: "proof-seconds", Spent: int(time.Since(start).Seconds()), Allows: s.ProofSeconds}
	case f != nil:
		p.Result, p.Why = Unproven, strings.TrimSpace(f.Where+": "+f.Message)
		return p, nil
	}
	p.Passed, p.Red = len(o.Passed), o.Red
	p.Result = Red
	if o.Green() {
		p.Result = Green
	} else if len(o.Red) == 0 {
		p.Why = "the test command failed, no test red: " + o.Exit
		if len(o.Passed) == 0 {
			p.Result, p.Why = Unproven, "the report names no test"
		}
	}
	return p, nil
}

// testerSettings are the tester's settings, the project's over the role's:
// its test command and report are the project's tests, one setting for
// both roles.
func testerSettings(repo string) (tester.Settings, error) {
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), "tester")
	if err != nil {
		return tester.Settings{}, fmt.Errorf("the tester's role, whose test-command the proof runs: %w", err)
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return tester.Settings{}, err
	}
	return tester.SettingsFrom(r.MergedSettings(cfg))
}

// tokensCap is ai-max-tokens reached: what the run's calls spent
// (out/calls.jsonl), as the engine counts it, at or past what it allows;
// nil when not, or no cap is set.
func tokensCap(runDir string, allows int) *Cap {
	if allows <= 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(runDir, "out", "calls.jsonl"))
	if err != nil {
		return nil
	}
	spent := 0
	for _, l := range strings.Split(string(data), "\n") {
		var c agent.Call
		if json.Unmarshal([]byte(l), &c) == nil {
			spent += c.TokensIn + c.TokensOut
		}
	}
	if spent < allows {
		return nil
	}
	return &Cap{Setting: "ai-max-tokens", Spent: spent, Allows: allows}
}

// proofFindings say the proof and the caps in the run's verdict.
func proofFindings(where string, p Proof, caps []Cap) []verdict.Finding {
	var out []verdict.Finding
	switch p.Result {
	case Green:
		out = append(out, verdict.Finding{Rule: "proof-green", Level: "info", Where: where,
			Message: fmt.Sprintf("%d test%s passed on its code, none red: `%s`", p.Passed, plural(p.Passed), p.Command)})
	case Red:
		msg := fmt.Sprintf("%d test%s red on its code: %s", len(p.Red), plural(len(p.Red)), strings.Join(p.Red, ", "))
		if p.Why != "" {
			msg = p.Why
		}
		out = append(out, verdict.Finding{Rule: "proof-red", Level: "warn", Where: where, Message: msg + "; its pull request opens as a Draft"})
	default:
		out = append(out, verdict.Finding{Rule: "proof-not-run", Level: "warn", Where: where,
			Message: "the project's tests did not run whole on its code (" + p.Why + "); its pull request opens as a Draft"})
	}
	for _, c := range caps {
		out = append(out, verdict.Finding{Rule: "cap-reached", Level: "warn", Where: c.Setting,
			Message: fmt.Sprintf("the run spent %d, %s allows %d: its pull request opens as a Draft", c.Spent, c.Setting, c.Allows)})
	}
	return out
}

// proofLines are the Verification's lines: what ran and its result, each
// cap reached.
func proofLines(p Proof, caps []Cap) string {
	var b strings.Builder
	switch p.Result {
	case Green:
		fmt.Fprintf(&b, "- `%s`: %d test%s passed, none red.\n", p.Command, p.Passed, plural(p.Passed))
	case Red:
		fmt.Fprintf(&b, "- `%s`: %d passed, %d red.\n", p.Command, p.Passed, len(p.Red))
		for _, t := range p.Red {
			fmt.Fprintf(&b, "  - red: `%s`\n", t)
		}
		if p.Why != "" {
			fmt.Fprintf(&b, "  - %s\n", p.Why)
		}
	default:
		fmt.Fprintf(&b, "- The project's tests did not run whole: %s\n", p.Why)
	}
	for _, c := range caps {
		fmt.Fprintf(&b, "- Cap reached: %s, %d spent of %d.\n", c.Setting, c.Spent, c.Allows)
	}
	return b.String()
}

// mergeRequest is out/merge-request.yaml: the branch's key, the issue's
// title, a body saying what the proof said; ready only when green with no
// cap reached. The body's other sections are step 4's.
func mergeRequest(a Act, title string) map[string]any {
	ready := a.Proof != nil && a.Proof.Result == Green && len(a.Caps) == 0
	var b strings.Builder
	fmt.Fprintf(&b, "Closes #%d\n\n", a.Issue)
	if ready {
		fmt.Fprintf(&b, "The developer's code for #%d, proven on the project's tests.\n\n", a.Issue)
	} else {
		fmt.Fprintf(&b, "The developer's code for #%d, **a Draft**: not proven green, or a cap reached. "+
			"A person takes it from here, or removes the claim on the issue.\n\n", a.Issue)
	}
	fmt.Fprintf(&b, "## %s\n\n", work.Name(work.Verification))
	if a.Proof != nil {
		b.WriteString(proofLines(*a.Proof, a.Caps))
	}
	return map[string]any{"key": fmt.Sprint(a.Issue), "title": title, "body": b.String(), "draft": !ready}
}

// issueTitle is the issue's title, from the task pre wrote; "" when not
// read, the engine's default title then.
func issueTitle(runDir string, issue int) string {
	data, err := os.ReadFile(filepath.Join(runDir, "in", "task.md"))
	if err != nil {
		return ""
	}
	first, _, _ := strings.Cut(string(data), "\n")
	t, _ := strings.CutPrefix(first, fmt.Sprintf("# Issue #%d: ", issue))
	if t == first {
		return ""
	}
	return strings.TrimSpace(t)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
