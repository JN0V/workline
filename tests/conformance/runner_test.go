// Package conformance runs the cases in cases/ against an engine binary, from
// the outside, as docs/spec/conformance.md describes. Any engine that passes
// them obeys the contract.
//
// Cases listed in pending.txt are not expected to pass yet. They still run: a
// pending case that starts passing fails the suite until it is removed from
// the list, so the list never lies.
package conformance

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

var engineBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "workline-engine-")
	if err != nil {
		panic(err)
	}
	engineBin = os.Getenv("WORKLINE_ENGINE")
	if engineBin == "" {
		engineBin = filepath.Join(dir, "workline")
		build := exec.Command("go", "build", "-o", engineBin, "../../cmd/workline")
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			panic("building the engine: " + err.Error())
		}
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type repoSpec struct {
	Repo  string   `yaml:"repo"`
	Setup []string `yaml:"setup"`
}

type caseFile struct {
	Case  string `yaml:"case"`
	About string `yaml:"about"`
	Given struct {
		Repo   string              `yaml:"repo"`
		Repos  map[string]repoSpec `yaml:"repos"`
		Setup  []string            `yaml:"setup"`
		Config map[string]any      `yaml:"config"`
		Forge  map[string]any      `yaml:"forge"`
	} `yaml:"given"`
	Run struct {
		Role   string            `yaml:"role"`
		Event  string            `yaml:"event"`
		Input  map[string]string `yaml:"input"`
		AI     string            `yaml:"ai"`
		Scope  []string          `yaml:"scope"`
		Tamper string            `yaml:"tamper"`
		Then   string            `yaml:"then"`
		Route  string            `yaml:"route"`
		Item   int               `yaml:"item"`
		Gate   string            `yaml:"gate"`
	} `yaml:"run"`
	Expect struct {
		Status     string                    `yaml:"status"`
		Findings   []map[string]string       `yaml:"findings"`
		AgentCalls *int                      `yaml:"agent-calls"`
		Applied    []string                  `yaml:"applied"`
		Refused    []string                  `yaml:"refused"`
		Files      map[string]map[string]any `yaml:"files"`
		Forge      map[string]any            `yaml:"forge"`
	} `yaml:"expect"`
}

type result struct {
	Status   string `json:"status"`
	Findings []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Message string `json:"message"`
	} `json:"findings"`
	AgentCalls int      `json:"agent-calls"`
	Applied    []string `json:"applied"`
	Refused    []string `json:"refused"`
}

func TestConformance(t *testing.T) {
	pending := readPending(t)
	files, _ := filepath.Glob("cases/*/*.yaml")
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no cases found")
	}
	for _, f := range files {
		var c caseFile
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := yaml.Unmarshal(data, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		t.Run(c.Case, func(t *testing.T) {
			problems := runCase(t, &c)
			switch {
			case pending[c.Case] && len(problems) == 0:
				t.Errorf("passes now: remove it from pending.txt")
			case pending[c.Case]:
				t.Skipf("pending: %s", problems[0])
			case len(problems) > 0:
				t.Errorf("%s\n  %s", c.About, strings.Join(problems, "\n  "))
			}
		})
	}
}

// runCase returns what differs from the expectation; empty means it passes.
func runCase(t *testing.T, c *caseFile) []string {
	if len(c.Given.Forge) > 0 || c.Run.Then != "" || c.Run.Route != "" || len(c.Run.Scope) > 0 {
		return []string{"the runner does not support forges, resume, routing or scopes yet"}
	}
	work := t.TempDir()
	env := hermeticEnv()

	names := make([]string, 0, len(c.Given.Repos))
	for n := range c.Given.Repos {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		dir := filepath.Join(work, n)
		if err := build(dir, c.Given.Repos[n].Repo, c.Given.Repos[n].Setup, env); err != nil {
			return []string{err.Error()}
		}
		env = append(env, n+"="+dir)
	}
	repo := filepath.Join(work, "repo")
	if err := build(repo, c.Given.Repo, c.Given.Setup, env); err != nil {
		return []string{err.Error()}
	}
	if c.Given.Config != nil {
		data, _ := yaml.Marshal(c.Given.Config)
		os.MkdirAll(filepath.Join(repo, ".workline"), 0o755)
		os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), data, 0o644)
	}

	roles, _ := filepath.Abs("../../roles")
	args := []string{"run-role", c.Run.Role, "--event", c.Run.Event, "--repo", repo, "--roles", roles, "--json"}
	if c.Run.Gate != "" {
		args = []string{"gate", c.Run.Gate, "--repo", repo, "--json"}
	}
	ai := c.Run.AI
	if strings.HasPrefix(ai, "fake:") {
		p, _ := filepath.Abs(filepath.Join("fixtures", "agents", strings.TrimPrefix(ai, "fake:")+".yaml"))
		ai = "fake:" + p
	}
	if ai != "" {
		args = append(args, "--ai", ai)
	}
	for k, v := range c.Run.Input {
		args = append(args, "--input", k+"="+v)
	}
	if c.Run.Tamper != "" {
		args = append(args, "--test-tamper-before-apply")
	}
	cmd := exec.Command(engineBin, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run() // the exit code mirrors the status, which is checked below
	var r result
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		return []string{fmt.Sprintf("engine printed no result: %v\n%s", err, stderr.String())}
	}
	return compare(c, &r, repo)
}

func compare(c *caseFile, r *result, repo string) []string {
	var p []string
	e := c.Expect
	if e.Status != "" && e.Status != r.Status {
		p = append(p, fmt.Sprintf("status = %s, want %s (findings: %v)", r.Status, e.Status, r.Findings))
	}
	for _, want := range e.Findings {
		found := false
		for _, f := range r.Findings {
			if f.Rule == want["rule"] && strings.Contains(f.Where, want["where"]) {
				found = true
			}
		}
		if !found {
			p = append(p, fmt.Sprintf("missing finding %v (got %v)", want, r.Findings))
		}
	}
	if e.AgentCalls != nil && *e.AgentCalls != r.AgentCalls {
		p = append(p, fmt.Sprintf("agent calls = %d, want %d", r.AgentCalls, *e.AgentCalls))
	}
	if e.Applied != nil && fmt.Sprint(e.Applied) != fmt.Sprint(r.Applied) {
		p = append(p, fmt.Sprintf("applied = %v, want %v", r.Applied, e.Applied))
	}
	if e.Refused != nil && fmt.Sprint(e.Refused) != fmt.Sprint(r.Refused) {
		p = append(p, fmt.Sprintf("refused = %v, want %v", r.Refused, e.Refused))
	}
	for path, want := range e.Files {
		data, err := os.ReadFile(filepath.Join(repo, path))
		if exists, ok := want["exists"].(bool); ok && exists != (err == nil) {
			p = append(p, fmt.Sprintf("%s exists = %v, want %v", path, err == nil, exists))
		}
		if text, ok := want["contains"].(string); ok && !strings.Contains(string(data), text) {
			p = append(p, fmt.Sprintf("%s does not contain %q", path, text))
		}
	}
	if len(e.Forge) > 0 {
		p = append(p, "the runner cannot check forge state yet")
	}
	return p
}

// build creates a fixture repository in dir and runs the case's setup in it.
func build(dir, fixture string, setup, env []string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	script, _ := filepath.Abs(filepath.Join("fixtures", "repos", fixture+".sh"))
	steps := append([]string{"sh " + script}, setup...)
	for _, s := range steps {
		cmd := exec.Command("sh", "-c", s)
		cmd.Dir, cmd.Env = dir, env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("fixture %s: %q failed: %v\n%s", fixture, s, err, out)
		}
	}
	return nil
}

// hermeticEnv keeps the machine's git config and hooks out of the tests.
func hermeticEnv() []string {
	return append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
		"GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid",
		"GIT_AUTHOR_DATE=2026-01-03T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-03T00:00:00Z",
	)
}

func readPending(t *testing.T) map[string]bool {
	out := map[string]bool{}
	f, err := os.Open("pending.txt")
	if os.IsNotExist(err) {
		return out
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			out[line] = true
		}
	}
	return out
}
