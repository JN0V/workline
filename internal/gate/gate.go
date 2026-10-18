// Package gate runs a gate: a list of checks, each a command whose result is
// read against thresholds decided in advance (docs/spec/gates.md). No model is
// ever asked whether to pass. On a merge request, the thresholds read only
// the findings the change adds (ADR-0049): each SARIF check runs again at
// the merge base, and its results are compared.
package gate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/linters"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Check is one command of a gate.
type Check struct {
	ID       string         `yaml:"id"`
	Run      string         `yaml:"run"`
	Output   string         `yaml:"output"` // exit | sarif
	Max      map[string]int `yaml:"max"`    // per SARIF level: error, warning, note
	Optional bool           `yaml:"optional"`
	Timeout  string         `yaml:"timeout"` // e.g. "10m"; default 10 minutes
	// New says who tells a merge request's new findings, for a SARIF
	// check: rerun (the default), the engine, running it again at the
	// merge base; tool, the tool itself, given WORKLINE_BASE.
	New string `yaml:"new"`
	// BaseTimeout caps the run at the merge base; by default twice the
	// head's, a minute at least, the check's timeout at most.
	BaseTimeout string `yaml:"base-timeout"`
	// Config: the files the tool reads its config from, beside those its
	// name tells, the head's copied over the merge base.
	Config []string `yaml:"config"`
}

// Gate is one named gate.
type Gate struct {
	Criteria string            `yaml:"criteria"`
	Checks   []Check           `yaml:"checks"`
	Enforce  map[string]string `yaml:"enforce"`
}

const defaultTimeout = 10 * time.Minute

// Load reads the gates declared in <repo>/.workline/config.yaml.
func Load(repo, name string) (*Gate, error) {
	var cfg struct {
		Gates map[string]Gate `yaml:"gates"`
	}
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("no gate %q: .workline/config.yaml: %w", name, err)
	}
	if err := role.CheckConfig(data); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	g, ok := cfg.Gates[name]
	if !ok {
		return nil, fmt.Errorf("no gate named %q in .workline/config.yaml", name)
	}
	return &g, nil
}

// Validate refuses a gate whose verdict would be decided while reading the
// results: every counted output needs its thresholds up front.
func (g *Gate) Validate() []verdict.Finding {
	var bad []verdict.Finding
	if len(g.Checks) == 0 {
		bad = append(bad, verdict.Finding{Rule: "invalid-gate", Message: "a gate with no check proves nothing"})
	}
	invalid := func(c Check, msg string) {
		bad = append(bad, verdict.Finding{Rule: "invalid-gate", Where: c.ID, Message: msg})
	}
	for _, c := range g.Checks {
		switch {
		case c.ID == "" || c.Run == "":
			invalid(c, "each check needs an id and a command")
		case c.Output != "exit" && c.Output != "sarif":
			invalid(c, fmt.Sprintf("output %q is not supported (exit, sarif)", c.Output))
		case c.Output == "sarif" && len(c.Max) == 0:
			invalid(c, "a counted output needs thresholds (`max`), decided before the gate runs")
		case c.New != "" && c.New != "rerun" && c.New != "tool":
			invalid(c, fmt.Sprintf("new %q is not supported (rerun, tool)", c.New))
		case c.New == "rerun" && c.Output == "exit":
			invalid(c, "an exit code holds no findings to compare at the merge base; the command reads WORKLINE_BASE to tell the new ones itself")
		}
		for key, d := range map[string]string{"timeout": c.Timeout, "base-timeout": c.BaseTimeout} {
			if _, err := time.ParseDuration(d); d != "" && err != nil {
				invalid(c, fmt.Sprintf("%s %q is not a duration (90s, 10m)", key, d))
			}
		}
	}
	return bad
}

// Run runs every check and returns the gate's verdict, every finding
// counted: on a schedule, a release, or by hand. A check that could not
// run, or whose output cannot be read, fails the gate unless marked
// optional.
func Run(repo, name string) *verdict.Verdict { return run(repo, name, "") }

// RunNew runs the gate on a change: the thresholds read only the findings
// it adds since the merge base of since and HEAD, and a suppression it
// adds counts as one (ADR-0049).
func RunNew(repo, name, since string) *verdict.Verdict { return run(repo, name, since) }

func run(repo, name, since string) *verdict.Verdict {
	g, err := Load(repo, name)
	if err != nil {
		return &verdict.Verdict{Status: verdict.Block, Summary: "the gate could not be loaded",
			Findings: []verdict.Finding{{Rule: "invalid-gate", Message: err.Error()}}}
	}
	if bad := g.Validate(); len(bad) > 0 {
		return &verdict.Verdict{Status: verdict.Block, Summary: "the gate is not valid; nothing was run", Findings: bad}
	}
	out, err := outDir(repo, name)
	if err != nil {
		return &verdict.Verdict{Status: verdict.Block, Findings: []verdict.Finding{{Rule: "check-error", Message: err.Error()}}}
	}
	var ch *change
	if since != "" {
		if ch, err = changeSince(repo, since); err != nil {
			return &verdict.Verdict{Status: verdict.Block, Summary: fmt.Sprintf("gate %s did not run", name),
				Findings: []verdict.Finding{{Rule: "check-error", Where: name, Message: err.Error()}}}
		}
		defer ch.cleanUp()
	}
	v := &verdict.Verdict{}
	var told []string // how each check's new findings were told
	for _, c := range g.Checks {
		found, note := runCheck(repo, out, c, ch, g.Checks)
		for _, f := range found {
			if f.Rule == "check-error" && c.Optional {
				f.Level = "warn"
				f.Message += " (optional check: reported, not blocking)"
			}
			v.Findings = append(v.Findings, f)
		}
		if note != "" {
			told = append(told, note)
		}
	}
	if ch != nil {
		v.Findings = append(v.Findings, ch.suppressions(g.Checks)...)
	}
	enforce(v, g)
	if v.Status == verdict.Pass {
		v.Summary = fmt.Sprintf("gate %s passed", name)
	} else {
		v.Summary = fmt.Sprintf("gate %s did not pass", name)
	}
	if ch != nil {
		v.Summary += fmt.Sprintf("; only the findings the change adds since %s count", short(ch.base))
		if len(told) > 0 {
			v.Summary += ": " + strings.Join(told, ", ")
		}
	}
	return v
}

// enforce lowers or drops findings by the gate's `enforce`, keyed by a
// check's id or by a rule, then gives the verdict: a finding left at no
// level blocks. A check that did not run is still said, off or not.
func enforce(v *verdict.Verdict, g *Gate) {
	var kept []verdict.Finding
	for _, f := range v.Findings {
		level, ok := g.Enforce[f.Where]
		if !ok {
			level = g.Enforce[f.Rule]
		}
		switch {
		case level == "off" && f.Rule != "check-error":
			continue
		case level == "off", level == "warn":
			f.Level = "warn"
		}
		kept = append(kept, f)
	}
	v.Findings, v.Status = kept, verdict.Pass
	for _, f := range kept {
		if f.Level == "" {
			v.Status = verdict.Block
		}
	}
}

// runCheck runs one check and reads its result; on a change, a SARIF
// check runs again at the merge base and only the results the head adds
// count. The note says how many, for the summary.
func runCheck(repo, out string, c Check, ch *change, all []Check) ([]verdict.Finding, string) {
	timeout := defaultTimeout
	if c.Timeout != "" {
		timeout, _ = time.ParseDuration(c.Timeout)
	}
	env := []string{"WORKLINE_BASE="} // every finding: none the job had set either
	if ch != nil {
		env = []string{"WORKLINE_BASE=" + ch.base}
	}
	start := time.Now()
	code, failed := execute(repo, out, c, timeout, env)
	if failed != nil {
		return []verdict.Finding{*failed}, ""
	}
	switch c.Output {
	case "exit":
		if code != 0 {
			return []verdict.Finding{{Rule: "check-failed", Where: c.ID, Message: fmt.Sprintf("exit %d; see %s", code, filepath.Join(out, c.ID+".log"))}}, ""
		}
		return nil, ""
	}
	head, err := readSARIF(out, repo)
	if err != nil {
		return []verdict.Finding{{Rule: "check-error", Where: c.ID, Message: err.Error()}}, ""
	}
	counted := results(head)
	if ch == nil {
		return overThreshold(c, counted, ""), ""
	}
	if c.New == "tool" {
		return overThreshold(c, counted, ""), c.ID + " told by the tool"
	}
	took := time.Since(start)
	base, failed := rerun(ch, out, c, all, took, timeout)
	tookBase := time.Since(start) - took
	if failed != nil {
		return []verdict.Finding{*failed}, ""
	}
	var found []verdict.Finding
	// The same tool on both sides, by construction, but for one the
	// project's tree pins (go.mod's tool, package.json): said, as a
	// finding of a rule the new version brings looks new.
	if hv, bv := version(head), version(base); hv != "" && bv != "" && hv != bv {
		found = append(found, verdict.Finding{Rule: "tool-version-differs", Where: c.ID, Level: "warn",
			Message: fmt.Sprintf("the head ran %s, the merge base %s: a finding the new version adds counts as the change's; a tool's upgrade gets a pull request of its own", hv, bv)})
	}
	compare(counted, results(base), &lines{root: repo}, &lines{root: ch.tree}, ch.moved)
	compared := filepath.Join(out, "compared", c.ID)
	if err := writeCompared(compared, head); err != nil {
		return []verdict.Finding{{Rule: "check-error", Where: c.ID, Message: err.Error()}}, ""
	}
	var added []*result
	for _, r := range counted {
		if r.state == "new" {
			added = append(added, r)
		}
	}
	old := len(counted) - len(added)
	found = append(found, overThreshold(c, added, fmt.Sprintf(" (%d there before it, not counted; each marked in %s)", old, compared))...)
	return found, fmt.Sprintf("%s %d new, %d before it (%s, then %s at the merge base)", c.ID, len(added), old, took.Round(100*time.Millisecond), tookBase.Round(100*time.Millisecond))
}

// execute runs a check's command in dir, its output in out.
func execute(dir, out string, c Check, timeout time.Duration, env []string) (int, *verdict.Finding) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", strings.ReplaceAll(c.Run, "{out}", out))
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	logFile := filepath.Join(out, c.ID+".log")
	log, _ := os.Create(logFile)
	if log != nil {
		defer log.Close()
		cmd.Stdout, cmd.Stderr = log, log
	}
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return 0, &verdict.Finding{Rule: "check-error", Where: c.ID, Message: "timed out after " + timeout.String()}
	case errors.As(err, &exitErr):
		if code := exitErr.ExitCode(); code == 126 || code == 127 {
			return code, &verdict.Finding{Rule: "check-error", Where: c.ID, Message: fmt.Sprintf("the command could not run (exit %d: tool missing or not executable); see %s", code, logFile)}
		}
		return exitErr.ExitCode(), nil
	case err != nil:
		return 0, &verdict.Finding{Rule: "check-error", Where: c.ID, Message: err.Error()}
	}
	return 0, nil
}

// rerun runs a SARIF check again at the merge base, its output in a
// folder of its own, within its cap. A run there that fails is an error:
// what the change adds cannot be told.
func rerun(ch *change, out string, c Check, all []Check, took, timeout time.Duration) ([]sarifFile, *verdict.Finding) {
	atBase := func(err error) *verdict.Finding {
		return &verdict.Finding{Rule: "check-error", Where: c.ID, Message: fmt.Sprintf("at the merge base %s: %v; the findings the change adds cannot be told", short(ch.base), err)}
	}
	var rerun []Check
	for _, x := range all {
		if x.Output == "sarif" && x.New != "tool" {
			rerun = append(rerun, x)
		}
	}
	if err := ch.checkout(configFor(rerun)); err != nil {
		return nil, atBase(err)
	}
	cap := min(max(2*took, time.Minute), timeout)
	if c.BaseTimeout != "" {
		cap, _ = time.ParseDuration(c.BaseTimeout)
	}
	dir := filepath.Join(out, "base")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, atBase(err)
	}
	if _, failed := execute(ch.tree, dir, c, cap, []string{"WORKLINE_BASE=" + ch.base}); failed != nil {
		return nil, atBase(errors.New(failed.Message))
	}
	base, err := readSARIF(dir, ch.tree)
	if err != nil {
		return nil, atBase(err)
	}
	return base, nil
}

func results(files []sarifFile) []*result {
	var out []*result
	for _, f := range files {
		out = append(out, f.results...)
	}
	return out
}

// version is the tools and versions a check's runs say, joined.
func version(files []sarifFile) string {
	seen := map[string]bool{}
	var out []string
	for _, f := range files {
		for _, v := range f.tools {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// overThreshold counts results by level against the check's `max`, and
// names the first of those counted.
func overThreshold(c Check, rs []*result, before string) []verdict.Finding {
	counts := map[string]int{}
	for _, r := range rs {
		counts[r.level]++
	}
	kind := ""
	if before != "" {
		kind = "new "
	}
	levels := make([]string, 0, len(c.Max))
	for level := range c.Max {
		levels = append(levels, level)
	}
	sort.Strings(levels)
	var f []verdict.Finding
	for _, level := range levels {
		if n, most := counts[level], c.Max[level]; n > most {
			noun := "findings"
			if n == 1 {
				noun = "finding"
			}
			f = append(f, verdict.Finding{Rule: "over-threshold", Where: c.ID,
				Message: fmt.Sprintf("%d %s%s %s, at most %d allowed%s%s", n, kind, level, noun, most, before, first(rs, level))})
		}
	}
	return f
}

// first names the first results of a level, where they are.
func first(rs []*result, level string) string {
	var named []string
	for _, r := range rs {
		if r.level != level {
			continue
		}
		if len(named) == 5 {
			named = append(named, "…")
			break
		}
		at := r.uri
		if r.line > 0 {
			at = fmt.Sprintf("%s:%d", r.uri, r.line)
		}
		named = append(named, strings.TrimSpace(at+" "+r.rule))
	}
	if len(named) == 0 {
		return ""
	}
	return ": " + strings.Join(named, ", ")
}

// suppressions are the suppressions the change adds, each a finding, for
// a check that runs the tool they silence; one moved, removed elsewhere in
// the change, is none.
func (ch *change) suppressions(checks []Check) []verdict.Finding {
	var f []verdict.Finding
	gone := map[string]int{}
	for k, n := range ch.gone {
		gone[k] = n
	}
	tools := make([][]string, len(checks))
	for i, c := range checks {
		tools[i] = linters.Ran(c.Run)
	}
	for _, l := range ch.added {
		for i, c := range checks {
			s, ok := linters.Suppressed(l.file, l.text, tools[i])
			if !ok {
				continue
			}
			if k := squeeze(l.text); gone[k] > 0 {
				gone[k]--
				break
			}
			f = append(f, verdict.Finding{Rule: "suppression-added", Where: fmt.Sprintf("%s:%d", l.file, l.n),
				Message: fmt.Sprintf("the change adds %s, which hides what check %s counts: it counts as a finding", s.Name, c.ID)})
			break
		}
	}
	return f
}

// configNames are the names of the files the tools a check runs read
// their config from.
func (c Check) configNames() []string {
	var out []string
	for _, t := range linters.Ran(c.Run) {
		out = append(out, linters.Configs(t)...)
	}
	return out
}

// outDir creates the gate's output folder inside the git directory.
func outDir(repo, name string) (string, error) {
	base := filepath.Join(repo, ".workline", "gates")
	if gd, err := exec.Command("git", "-C", repo, "rev-parse", "--absolute-git-dir").Output(); err == nil {
		base = filepath.Join(strings.TrimSpace(string(gd)), "workline", "gates")
	}
	dir := filepath.Join(base, time.Now().UTC().Format("20060102T150405.000000000")+"-"+name)
	return dir, os.MkdirAll(dir, 0o755)
}
