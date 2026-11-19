package developer

// The proof and the caps (ADR-0054, step 3): before its pull request
// opens, the code within its bounds is laid on a fresh checkout of the
// commit the run read — never its agent's worktree, whose ignored files,
// caches and git store its agent could have changed (ADR-0056, step 3) —
// and the project's test command runs there: the tester's own setting, so
// the acceptance tests on the branch run with the rest. Its agent's own
// runs are not read. Green, no cap reached, its agent not failed: the pull request opens
// ready. Red, not run whole, a cap reached or its agent failed: a Draft,
// which wakes no agent (ADR-0050 §4),
// saying what failed. Both are kept in the claim's record, for the auditor
// (ADR-0046).

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/kit/worktree"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/tester"
)

// Flaky is a flaky-n-of-n proof to run: the named test, Runs times.
type Flaky struct {
	Test string
	Runs int
}

// flakyOf is the flaky-n-of-n proof the tester recorded, its runs the
// record's or the flaky-runs setting; nil for another kind or none.
func flakyOf(k *backlog.Proof, s Settings) *Flaky {
	if k == nil || k.Kind != backlog.KindFlaky {
		return nil
	}
	n := k.Runs
	if n <= 0 {
		n = s.FlakyRuns
	}
	return &Flaky{Test: k.Test, Runs: n}
}

// again is the flaky-n-of-n proof a review round runs again: the one
// the issue run made, from its record.
func again(p *devmark.Proof) *Flaky {
	if p == nil || p.Runs == 0 {
		return nil
	}
	return &Flaky{Test: p.Test, Runs: p.Runs}
}

// prove runs the project's test command on the patches laid on the commit
// the run read, stopped past the proof's seconds: a cap then. With a
// proof-command naming `{packages}`, on the folders of the patches' files
// and of also (the acceptance tests) alone; the pull request's own CI
// runs the whole suite. A flaky-n-of-n proof runs it n.Runs times on
// the one tree, `{test}` the named test; green only N runs of N, the
// test passing in each.
func prove(repo, at string, patches []intent.Intention, also []string, n *Flaky, s Settings) (devmark.Proof, *devmark.Cap) {
	// The flaky proof's test and runs first, kept whatever stops it: a
	// review round reads them again from the record (again).
	var p devmark.Proof
	if n != nil {
		p.Test, p.Runs = n.Test, max(1, n.Runs)
	}
	ts, err := testerSettingsIn(s.rolesDir, repo)
	if err != nil {
		p.Result, p.Why = devmark.Unproven, err.Error()
		return p, nil
	}
	p.Command = ts.Command
	scope := tester.Scope{Files: also}
	if n != nil {
		if !backlog.TestName.MatchString(n.Test) {
			p.Result, p.Why = devmark.Unproven, "the tester's flaky-n-of-n record names no test to run"
			return p, nil
		}
		scope.Test, scope.Runs = n.Test, p.Runs
	}
	if strings.Contains(s.ProofCommand, "{packages}") {
		scope.Command = s.ProofCommand
		for _, in := range patches {
			files, _ := touched(in.Value)
			scope.Files = append(scope.Files, files...)
		}
	}
	if at == "" {
		at = "HEAD"
	}
	// The CI tests the merge with main: so does the proof (#1899).
	merged, why, err := withMain(repo, at)
	if err != nil {
		p.Result, p.Why = devmark.Unproven, err.Error()
		return p, nil
	}
	if why != "" {
		p.Result, p.Why = devmark.Unproven, why
		return p, nil
	}
	at = merged
	var deadline time.Time
	if s.ProofSeconds > 0 {
		deadline = time.Now().Add(time.Duration(s.ProofSeconds) * time.Second)
	}
	start := time.Now()
	o, f := tester.OnChange(repo, at, patches, ts, deadline, scope)
	switch {
	case f != nil && f.Rule == "proof-over-time" && f.Where == "suite-max-seconds":
		// One run stopped at the tester's suite-max-seconds, before the
		// proof's own seconds (#1425): that setting named, where it hung.
		p.Ran = o.Runs
		p.Result, p.Why = devmark.Unproven, fmt.Sprintf("a run of the tests stopped at suite-max-seconds (%d s): %s", ts.SuiteMaxSeconds, f.Message)
		return p, &devmark.Cap{Setting: "suite-max-seconds", Spent: int(time.Since(start).Seconds()), Allows: ts.SuiteMaxSeconds}
	case f != nil && f.Rule == "proof-over-time":
		p.Ran = o.Runs
		p.Result, p.Why = devmark.Unproven, fmt.Sprintf("the test command ran past %d s: stopped", s.ProofSeconds)
		return p, &devmark.Cap{Setting: "proof-seconds", Spent: int(time.Since(start).Seconds()), Allows: s.ProofSeconds}
	case f != nil:
		p.Result, p.Why = devmark.Unproven, strings.TrimSpace(f.Where+": "+f.Message)
		return p, nil
	}
	p.Passed, p.Red, p.Tree, p.At, p.Missing = len(o.Passed), o.Red, "fresh", o.At, o.Missing
	p.Scope, p.Packages = devmark.Whole, o.Packages
	switch {
	case len(o.Packages) > 0:
		p.Scope, p.Command = devmark.Touched, s.ProofCommand
	case s.ProofCommand == "":
		p.Whole = "no proof-command set"
	case scope.Command == "":
		p.Whole = "its proof-command names no {packages}"
	default:
		p.Whole = "no file it touches lies in a folder holding tests"
	}
	p.Result = devmark.Red
	if o.Green() {
		p.Result = devmark.Green
	} else if len(o.Red) == 0 {
		p.Why = "the test command failed, no test red: " + o.Exit
		if len(o.Passed) == 0 {
			p.Result, p.Why = devmark.Unproven, "the report names no test"
		}
	}
	if n == nil {
		return p, nil
	}
	p.Ran = o.Runs
	reported, _ := o.Passes(n.Test)
	switch {
	case p.Result == devmark.Green && !reported:
		p.Result, p.Why = devmark.Unproven, fmt.Sprintf("the named test `%s` was not reported, run %d of %d", n.Test, o.Runs, p.Runs)
	case p.Result == devmark.Red && p.Why == "":
		p.Why = fmt.Sprintf("run %d of %d: %d test%s red on its code: %s", o.Runs, p.Runs, len(p.Red), devmark.Plural(len(p.Red)), strings.Join(p.Red, ", "))
	case p.Result != devmark.Green:
		p.Why = fmt.Sprintf("run %d of %d: %s", o.Runs, p.Runs, p.Why)
	}
	return p, nil
}

// mainRef is the ref of the project's main in repo: origin's default
// branch, else main, as fetched or local; "" when none is found.
func mainRef(repo string) string {
	var names []string
	if h, err := gitOut(repo, "symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD"); err == nil {
		names = append(names, strings.TrimPrefix(h, "origin/"))
	}
	names = append(names, "main")
	for _, n := range names {
		for _, r := range []string{"refs/remotes/origin/" + n, "refs/heads/" + n} {
			if _, err := gitOut(repo, "rev-parse", "-q", "--verify", r+"^{commit}"); err == nil {
				return r
			}
		}
	}
	return ""
}

// withMain is the commit the proof runs on: at with main merged in, as the
// CI tests it, a merge commit kept in repo's objects only. at itself when
// main is none, or already in it. why says a merge that conflicts: not
// proven, never green.
func withMain(repo, at string) (commit, why string, err error) {
	head, herr := gitOut(repo, "rev-parse", "--verify", "-q", "--end-of-options", at+"^{commit}")
	ref := mainRef(repo)
	if herr != nil || ref == "" {
		return at, "", nil // unread here: the fresh tree says it
	}
	tip, err := gitOut(repo, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return at, "", nil
	}
	if exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", tip, head).Run() == nil {
		return head, "", nil
	}
	name := strings.TrimPrefix(strings.TrimPrefix(ref, "refs/remotes/origin/"), "refs/heads/")
	tree, files, err := worktree.MergeTree(repo, head, tip)
	if err != nil {
		return "", "", fmt.Errorf("main merged into its code, in memory: %w", err)
	}
	if len(files) > 0 {
		return "", fmt.Sprintf("its code conflicts with %s in %s: the proof does not run on a merge that conflicts; a merge round settles it", name, strings.Join(files, ", ")), nil
	}
	c, err := mergeCommit(repo, tree, head, tip, "workline: "+name+" merged in, for the proof")
	if err != nil {
		return "", "", err
	}
	return c, "", nil
}

// testerSettings are the tester's settings, the project's over the role's:
// its test command and report are the project's tests, one setting for
// both roles.
func testerSettings(repo string) (tester.Settings, error) {
	return testerSettingsIn("", repo)
}

// testerSettingsIn are the tester's settings, its role read in rolesDir,
// "" the one the engine gives a role's run.
func testerSettingsIn(rolesDir, repo string) (tester.Settings, error) {
	if rolesDir == "" {
		rolesDir = os.Getenv("WORKLINE_ROLES_DIR")
	}
	r, err := role.Load(rolesDir, "tester")
	if err != nil {
		return tester.Settings{}, fmt.Errorf("the tester's role, whose test-command the proof runs: %w", err)
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return tester.Settings{}, err
	}
	return tester.SettingsFrom(r.MergedSettings(cfg))
}

// Used is what the run's agent calls used, as the agents report it: the
// turns, the fresh tokens (agent.Call.Fresh: what ai-max-tokens holds),
// the cache reads apart, the seconds waited, the times it used each tool.
type Used struct {
	Turns   int
	Tokens  int
	Cached  int
	Seconds int
	Tools   map[string]int
	// mostTurns: the most turns one call took, each call told
	// ai-max-turns whole (claude's --max-turns is per call); timeCut:
	// the engine stopped waiting, the run's ai-max-seconds spent;
	// noTurns, noTokens: the calls that reported none; failed: why the
	// last call failed, its time aside (an earlier one asked again).
	mostTurns         int
	timeCut           bool
	noTurns, noTokens int
	failed            string
}

// used reads the run's calls (out/calls.jsonl), as the engine counts them.
func used(runDir string) Used {
	var u Used
	data, err := os.ReadFile(filepath.Join(runDir, "out", "calls.jsonl"))
	if err != nil {
		return u
	}
	seconds := 0.0
	for _, l := range strings.Split(string(data), "\n") {
		var c agent.Call
		if json.Unmarshal([]byte(l), &c) != nil {
			continue
		}
		u.Turns += c.Turns
		u.mostTurns = max(u.mostTurns, c.Turns)
		for name, n := range c.Tools {
			if u.Tools == nil {
				u.Tools = map[string]int{}
			}
			u.Tools[name] += n
		}
		u.Tokens += c.Fresh()
		u.Cached += c.TokensCached
		seconds += c.Seconds
		u.timeCut = u.timeCut || c.TimeCut
		if c.Turns == 0 {
			u.noTurns++
		}
		if c.TokensIn+c.TokensOut == 0 {
			u.noTokens++
		}
		u.failed = c.Failed // the last call says: one asked again that answered did not fail
	}
	u.Seconds = int(math.Round(seconds))
	return u
}

// agentCaps are the caps on its agent the run reached (ADR-0056): its
// time over the run (the engine stopped waiting and killed it), its turns
// in one call (each is told ai-max-turns), its fresh tokens over the run,
// as it reports them, at or past what each allows; none set, none reached.
func agentCaps(u Used, s Settings) []devmark.Cap {
	var caps []devmark.Cap
	if u.timeCut && s.MaxSeconds > 0 {
		caps = append(caps, devmark.Cap{Setting: "ai-max-seconds", Spent: u.Seconds, Allows: s.MaxSeconds})
	}
	if s.MaxTurns > 0 && u.mostTurns >= s.MaxTurns {
		caps = append(caps, devmark.Cap{Setting: "ai-max-turns", Spent: u.mostTurns, Allows: s.MaxTurns})
	}
	if s.MaxTokens > 0 && u.Tokens >= s.MaxTokens {
		caps = append(caps, devmark.Cap{Setting: "ai-max-tokens", Spent: u.Tokens, Allows: s.MaxTokens})
	}
	return caps
}

// unreported says a cap its agent did not report the use of: a cmd:
// agent call that writes no turns or tokens to WORKLINE_CALL is not held
// to them, said, never read as under them (principle 12). One call
// reporting does not vouch for another.
func unreported(where string, u Used, s Settings) []verdict.Finding {
	var not []string
	if s.MaxTurns > 0 && u.noTurns > 0 {
		not = append(not, "ai-max-turns")
	}
	if s.MaxTokens > 0 && u.noTokens > 0 {
		not = append(not, "ai-max-tokens")
	}
	if len(not) == 0 {
		return nil
	}
	held := "; no cap held it"
	if s.MaxSeconds > 0 {
		held = fmt.Sprintf("; its time held it, ai-max-seconds %d", s.MaxSeconds)
	}
	return []verdict.Finding{{Rule: "cap-not-checked", Level: "info", Where: where,
		Message: "its agent reported no use of " + strings.Join(not, " nor ") + ": not checked this run" + held}}
}

// agentFailed says its agent's call failed: its edits so far are bounded
// and proven all the same, the pull request a Draft, its reason said.
func agentFailed(where, why string) verdict.Finding {
	return verdict.Finding{Rule: "agent-failed", Level: "warn", Where: where,
		Message: "its agent failed (" + why + "): what it edited is proven all the same; its pull request opens as a Draft"}
}

// proofFindings say the proof and the caps in the run's verdict.
func proofFindings(where string, p devmark.Proof, caps []devmark.Cap) []verdict.Finding {
	var out []verdict.Finding
	switch p.Result {
	case devmark.Green:
		out = append(out, verdict.Finding{Rule: "proof-green", Level: "info", Where: where,
			Message: fmt.Sprintf("%d test%s passed on its code, none red: `%s`%s", p.Passed, devmark.Plural(p.Passed), p.Command, p.NOfN())})
	case devmark.Red:
		msg := fmt.Sprintf("%d test%s red on its code: %s", len(p.Red), devmark.Plural(len(p.Red)), strings.Join(p.Red, ", "))
		if p.Why != "" {
			msg = p.Why
		}
		msg += missingSaid(p.Missing)
		out = append(out, verdict.Finding{Rule: "proof-red", Level: "warn", Where: where, Message: msg + "; its pull request opens as a Draft"})
	default:
		out = append(out, verdict.Finding{Rule: "proof-not-run", Level: "warn", Where: where,
			Message: "the project's tests did not run whole on its code (" + p.Why + ")" + missingSaid(p.Missing) + "; its pull request opens as a Draft"})
	}
	for _, c := range caps {
		out = append(out, verdict.Finding{Rule: "cap-reached", Level: "warn", Where: c.Setting,
			Message: fmt.Sprintf("the run spent %d, %s allows %d; its pull request opens as a Draft", c.Spent, c.Setting, c.Allows)})
	}
	return out
}

// missingSaid is what a red finding adds for the tools missing where the
// proof ran: "" when none.
func missingSaid(tools []string) string {
	if len(tools) == 0 {
		return ""
	}
	return "; not found where it ran: " + strings.Join(tools, ", ")
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

// ProveMerge runs the project's test command whole on at, a merge of the
// base into a request made by the engine, on a fresh checkout of it: the
// proof that two changes green apart are green together (#1370). Whole,
// never the proof-command's folders: a clash may sit in a file neither
// change touched. The developer's proof-seconds bound it. rolesDir: where
// the roles are read, as the engine resolves them.
func ProveMerge(rolesDir, repo, at string) devmark.Proof {
	var s Settings
	if r, err := role.Load(rolesDir, "developer"); err == nil {
		if cfg, err := role.LoadProjectConfig(repo); err == nil {
			s = SettingsFrom(r.MergedSettings(cfg))
		}
	}
	s.ProofCommand, s.rolesDir = "", rolesDir
	p, _ := prove(repo, at, nil, nil, nil, s)
	return p
}
