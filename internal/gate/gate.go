// Package gate runs a gate: a list of checks, each a command whose result is
// read against thresholds decided in advance (docs/spec/gates.md). No model is
// ever asked whether to pass.
package gate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
}

// Gate is one named gate.
type Gate struct {
	Criteria string            `yaml:"criteria"`
	Checks   []Check           `yaml:"checks"`
	Enforce  map[string]string `yaml:"enforce"`
}

// Load reads the gates declared in <repo>/.workline/config.yaml.
func Load(repo, name string) (*Gate, error) {
	var cfg struct {
		Gates map[string]Gate `yaml:"gates"`
	}
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("no gate %q: .workline/config.yaml: %w", name, err)
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
	for _, c := range g.Checks {
		switch {
		case c.ID == "" || c.Run == "":
			bad = append(bad, verdict.Finding{Rule: "invalid-gate", Where: c.ID, Message: "each check needs an id and a command"})
		case c.Output != "exit" && c.Output != "sarif":
			bad = append(bad, verdict.Finding{Rule: "invalid-gate", Where: c.ID, Message: fmt.Sprintf("output %q is not supported (exit, sarif)", c.Output)})
		case c.Output == "sarif" && len(c.Max) == 0:
			bad = append(bad, verdict.Finding{Rule: "invalid-gate", Where: c.ID, Message: "a counted output needs thresholds (`max`), decided before the gate runs"})
		}
	}
	return bad
}

// Run runs every check and returns the gate's verdict. A check that could not
// run, or whose output cannot be read, fails the gate unless marked optional.
func Run(repo, name string) *verdict.Verdict {
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
	v := &verdict.Verdict{Status: verdict.Pass}
	for _, c := range g.Checks {
		for _, f := range runCheck(repo, out, c) {
			if f.Rule == "check-error" && c.Optional {
				f.Level = "warn"
				f.Message += " (optional check: reported, not blocking)"
			}
			v.Findings = append(v.Findings, f)
		}
	}
	v.Status = verdict.Pass
	for _, f := range v.Findings {
		if f.Level == "" {
			v.Status = verdict.Block
		}
	}
	verdict.Enforce(v, g.Enforce)
	if v.Status == verdict.Pass {
		v.Summary = fmt.Sprintf("gate %s passed", name)
	} else {
		v.Summary = fmt.Sprintf("gate %s did not pass", name)
	}
	return v
}

func runCheck(repo, out string, c Check) []verdict.Finding {
	timeout := 10 * time.Minute
	if c.Timeout != "" {
		if d, err := time.ParseDuration(c.Timeout); err == nil {
			timeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", strings.ReplaceAll(c.Run, "{out}", out))
	cmd.Dir = repo
	log, _ := os.Create(filepath.Join(out, c.ID+".log"))
	if log != nil {
		defer log.Close()
		cmd.Stdout, cmd.Stderr = log, log
	}
	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return []verdict.Finding{{Rule: "check-error", Where: c.ID, Message: "timed out after " + timeout.String()}}
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	case err != nil:
		return []verdict.Finding{{Rule: "check-error", Where: c.ID, Message: err.Error()}}
	}
	if code == 126 || code == 127 {
		return []verdict.Finding{{Rule: "check-error", Where: c.ID, Message: fmt.Sprintf("the command could not run (exit %d: tool missing or not executable); see %s", code, filepath.Join(out, c.ID+".log"))}}
	}
	switch c.Output {
	case "exit":
		if code != 0 {
			return []verdict.Finding{{Rule: "check-failed", Where: c.ID, Message: fmt.Sprintf("exit %d; see %s", code, filepath.Join(out, c.ID+".log"))}}
		}
		return nil
	case "sarif":
		counts, err := sarifCounts(out)
		if err != nil {
			return []verdict.Finding{{Rule: "check-error", Where: c.ID, Message: err.Error()}}
		}
		var f []verdict.Finding
		for level, max := range c.Max {
			if n := counts[level]; n > max {
				f = append(f, verdict.Finding{Rule: "over-threshold", Where: c.ID,
					Message: fmt.Sprintf("%d %s findings, at most %d allowed", n, level, max)})
			}
		}
		return f
	}
	return nil
}

// sarifCounts counts results by level across the SARIF files written in out
// since the check started. Results without a level count as "warning", as the
// SARIF standard says.
func sarifCounts(out string) (map[string]int, error) {
	files, _ := filepath.Glob(filepath.Join(out, "*.sarif"))
	if len(files) == 0 {
		return nil, errors.New("the check wrote no SARIF file in {out}")
	}
	counts := map[string]int{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var s struct {
			Runs []struct {
				Results []struct {
					Level string `json:"level"`
				} `json:"results"`
			} `json:"runs"`
		}
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("%s is not readable SARIF: %v", filepath.Base(f), err)
		}
		for _, r := range s.Runs {
			for _, res := range r.Results {
				level := res.Level
				if level == "" {
					level = "warning"
				}
				counts[level]++
			}
		}
		os.Rename(f, f+".read") // counted once, even if another check writes SARIF later
	}
	return counts, nil
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
