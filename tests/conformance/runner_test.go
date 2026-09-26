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
		Seen   map[string]any      `yaml:"models-seen"` // the models this machine saw answer, before the run
	} `yaml:"given"`
	Run struct {
		Role    string            `yaml:"role"`
		Target  map[string]int    `yaml:"target"`
		NoApply bool              `yaml:"no-apply"`
		Event   string            `yaml:"event"`
		Input   map[string]string `yaml:"input"`
		AI      string            `yaml:"ai"`
		Scope   []string          `yaml:"scope"`
		Tamper  string            `yaml:"tamper"`
		Then    string            `yaml:"then"`
		Route   string            `yaml:"route"`
		Item    int               `yaml:"item"`
		Gate    string            `yaml:"gate"`
		Reports bool              `yaml:"reports"` // also write --sarif and --code-quality
	} `yaml:"run"`
	Expect struct {
		Status     string                    `yaml:"status"`
		Findings   []map[string]string       `yaml:"findings"`
		NoFindings []map[string]string       `yaml:"no-findings"` // findings that must not be there
		AgentCalls *int                      `yaml:"agent-calls"`
		Applied    []string                  `yaml:"applied"`
		Refused    []string                  `yaml:"refused"`
		Files      map[string]map[string]any `yaml:"files"`
		Forge      map[string]any            `yaml:"forge"`
		Steps      []string                  `yaml:"steps"`
		Calls      []map[string]string       `yaml:"calls"`
		SARIF       []map[string]any `yaml:"sarif"`        // results, by rule, uri, line, level
		CodeQuality []map[string]any `yaml:"code-quality"` // issues, by check_name, path, line, severity
		LeftOut     []string         `yaml:"left-out"`     // wheres found in neither report
	} `yaml:"expect"`
}

type result struct {
	Status   string `json:"status"`
	Findings []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Message string `json:"message"`
	} `json:"findings"`
	AgentCalls int              `json:"agent-calls"`
	Calls      []map[string]any `json:"calls"`
	RunDir     string           `json:"run-dir"`
	Pending    []string         `json:"pending"` // runs a line judged and did not apply
	Applied    []string         `json:"applied"`
	Refused    []string         `json:"refused"`
	Steps      []struct {
		Name string `json:"name"`
	} `json:"steps"`
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
	work := t.TempDir()
	seen := filepath.Join(work, "models-seen.yaml")
	env := append(hermeticEnv(), "WORKLINE_MODELS_SEEN="+seen)
	if c.Given.Seen != nil {
		data, _ := yaml.Marshal(c.Given.Seen)
		if err := os.WriteFile(seen, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

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

	forgeFile := ""
	if len(c.Given.Forge) > 0 {
		forgeFile = filepath.Join(work, "forge.json")
		data, _ := json.Marshal(c.Given.Forge)
		os.WriteFile(forgeFile, data, 0o644)
	}
	roles, _ := filepath.Abs("../../roles")
	args := []string{"run-role", c.Run.Role, "--event", c.Run.Event, "--repo", repo, "--roles", roles, "--json"}
	switch {
	case c.Run.Gate != "":
		args = []string{"gate", c.Run.Gate, "--repo", repo, "--json"}
	case c.Run.Route == "ready" && c.Run.Item != 0:
		args = []string{"item", "ready", fmt.Sprint(c.Run.Item), "--repo", repo, "--json"}
		if forgeFile != "" {
			args = append(args, "--forge", "fake:"+forgeFile)
		}
	case c.Run.Route != "":
		args = []string{"route", c.Run.Route, "--repo", repo, "--roles", roles, "--json"}
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
	if forgeFile != "" && c.Run.Item == 0 {
		args = append(args, "--forge", "fake:"+forgeFile)
	}
	for kind, id := range c.Run.Target {
		args = append(args, "--target", fmt.Sprintf("%s:%d", kind, id))
	}
	for _, s := range c.Run.Scope {
		args = append(args, "--scope", s)
	}
	if c.Run.NoApply {
		args = append(args, "--no-apply")
	}
	if c.Run.Tamper != "" {
		args = append(args, "--test-tamper-before-apply")
	}
	sarifFile, cqFile := filepath.Join(work, "findings.sarif"), filepath.Join(work, "code-quality.json")
	if c.Run.Reports {
		args = append(args, "--sarif", sarifFile, "--code-quality", cqFile)
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
	if c.Run.Then == "resume" {
		dirs := r.Pending
		if len(dirs) == 0 && r.RunDir != "" {
			dirs = []string{r.RunDir}
		}
		if len(dirs) == 0 {
			return []string{"the first run reported no run folder to resume"}
		}
		resume := exec.Command(engineBin, append(append([]string{"apply"}, dirs...), "--json")...)
		resume.Env = env
		stdout.Reset()
		resume.Stdout, resume.Stderr = &stdout, &stderr
		_ = resume.Run()
		var r2 result
		if err := json.Unmarshal(stdout.Bytes(), &r2); err != nil {
			return []string{fmt.Sprintf("resume printed no result: %v\n%s", err, stderr.String())}
		}
		r2.AgentCalls += r.AgentCalls
		r2.Calls = append(r.Calls, r2.Calls...)
		r = r2
	}
	problems := compare(c, &r, repo)
	if c.Run.Reports {
		problems = append(problems, compareReports(c, sarifFile, cqFile)...)
	}
	if len(c.Expect.Forge) > 0 {
		problems = append(problems, compareForge(c.Expect.Forge, forgeFile)...)
	}
	return problems
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
	for _, bad := range e.NoFindings {
		for _, f := range r.Findings {
			if f.Rule == bad["rule"] && f.Where == bad["where"] {
				p = append(p, fmt.Sprintf("finding %v should not be there: %s", bad, f.Message))
			}
		}
	}
	if e.AgentCalls != nil && *e.AgentCalls != r.AgentCalls {
		p = append(p, fmt.Sprintf("agent calls = %d, want %d", r.AgentCalls, *e.AgentCalls))
	}
	if e.Calls != nil {
		if len(r.Calls) != len(e.Calls) {
			p = append(p, fmt.Sprintf("calls = %v, want %v", r.Calls, e.Calls))
		} else {
			for i, want := range e.Calls {
				for k, v := range want {
					if fmt.Sprint(r.Calls[i][k]) != v {
						p = append(p, fmt.Sprintf("call %d: %s = %v, want %s", i+1, k, r.Calls[i][k], v))
					}
				}
			}
		}
	}
	if e.Applied != nil && fmt.Sprint(e.Applied) != fmt.Sprint(r.Applied) {
		p = append(p, fmt.Sprintf("applied = %v, want %v", r.Applied, e.Applied))
	}
	if e.Steps != nil {
		var got []string
		for _, s := range r.Steps {
			got = append(got, s.Name)
		}
		if fmt.Sprint(got) != fmt.Sprint(e.Steps) {
			p = append(p, fmt.Sprintf("steps = %v, want %v", got, e.Steps))
		}
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
	fixtures, _ := filepath.Abs("fixtures")
	return append(os.Environ(), "FIXTURES="+fixtures, // for a cmd: agent's script
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

// compareForge checks the simulated forge's state: for each listed item, by
// id, `comments` is a count and `labels` the exact set.
func compareForge(want map[string]any, file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return []string{"no forge state to check: " + err.Error()}
	}
	var got map[string][]map[string]any
	json.Unmarshal(data, &got)
	var p []string
	for kind, items := range want {
		list, _ := items.([]any)
		for _, w := range list {
			wm, _ := w.(map[string]any)
			var found map[string]any
			for _, g := range got[kind] {
				if fmt.Sprint(g["id"]) == fmt.Sprint(wm["id"]) {
					found = g
				}
			}
			if found == nil {
				p = append(p, fmt.Sprintf("forge: no %s with id %v", kind, wm["id"]))
				continue
			}
			if n, ok := wm["comments"]; ok {
				c, _ := found["comments"].([]any)
				if fmt.Sprint(len(c)) != fmt.Sprint(n) {
					p = append(p, fmt.Sprintf("forge: %s %v has %d comments, want %v", kind, wm["id"], len(c), n))
				}
			}
			if l, ok := wm["labels"]; ok {
				gl, _ := found["labels"].([]any)
				if fmt.Sprint(gl) != fmt.Sprint(l) {
					p = append(p, fmt.Sprintf("forge: %s %v labels = %v, want %v", kind, wm["id"], gl, l))
				}
			}
		}
	}
	return p
}

// compareReports checks the SARIF and Code Quality files a run wrote.
func compareReports(c *caseFile, sarifFile, cqFile string) []string {
	var p []string
	var sarif struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
				PartialFingerprints map[string]string `json:"partialFingerprints"`
			} `json:"results"`
		} `json:"runs"`
	}
	var cq []struct {
		CheckName   string `json:"check_name"`
		Severity    string `json:"severity"`
		Fingerprint string `json:"fingerprint"`
		Location    struct {
			Path  string `json:"path"`
			Lines struct {
				Begin int `json:"begin"`
			} `json:"lines"`
		} `json:"location"`
	}
	for file, into := range map[string]any{sarifFile: &sarif, cqFile: &cq} {
		data, err := os.ReadFile(file)
		if err == nil {
			err = json.Unmarshal(data, into)
		}
		if err != nil {
			return []string{fmt.Sprintf("%s: %v", filepath.Base(file), err)}
		}
	}
	if sarif.Version != "2.1.0" || len(sarif.Runs) != 1 {
		p = append(p, fmt.Sprintf("SARIF version %q with %d runs, want 2.1.0 with one run", sarif.Version, len(sarif.Runs)))
		return p
	}
	var got, gotCQ []map[string]any
	uris := map[string]bool{}
	for _, r := range sarif.Runs[0].Results {
		if len(r.Locations) != 1 || len(r.PartialFingerprints) == 0 {
			p = append(p, fmt.Sprintf("SARIF result %s has %d locations and %d fingerprints, want one and some", r.RuleID, len(r.Locations), len(r.PartialFingerprints)))
			continue
		}
		l := r.Locations[0].PhysicalLocation
		uris[l.ArtifactLocation.URI] = true
		got = append(got, map[string]any{"rule": r.RuleID, "uri": l.ArtifactLocation.URI, "line": l.Region.StartLine, "level": r.Level})
	}
	for _, i := range cq {
		if i.Fingerprint == "" {
			p = append(p, "a Code Quality issue has no fingerprint")
		}
		uris[i.Location.Path] = true
		gotCQ = append(gotCQ, map[string]any{"check_name": i.CheckName, "path": i.Location.Path, "line": i.Location.Lines.Begin, "severity": i.Severity})
	}
	has := func(list []map[string]any, want map[string]any) bool {
		for _, g := range list {
			ok := true
			for k, v := range want {
				ok = ok && fmt.Sprint(g[k]) == fmt.Sprint(v)
			}
			if ok {
				return true
			}
		}
		return false
	}
	for _, w := range c.Expect.SARIF {
		if !has(got, w) {
			p = append(p, fmt.Sprintf("SARIF lacks %v (got %v)", w, got))
		}
	}
	for _, w := range c.Expect.CodeQuality {
		if !has(gotCQ, w) {
			p = append(p, fmt.Sprintf("Code Quality lacks %v (got %v)", w, gotCQ))
		}
	}
	for _, w := range c.Expect.LeftOut {
		if uris[w] {
			p = append(p, fmt.Sprintf("%s is in a report, and should be left out", w))
		}
	}
	return p
}
