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
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

var engineBin, rolesDir string

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
	// The shipped roles, and the roles only the tests use (fixtures/roles):
	// a mechanism no shipped role needs is still tried on one of those.
	rolesDir = filepath.Join(dir, "roles")
	if err := linkRoles(rolesDir, "../../roles", "fixtures/roles"); err != nil {
		panic("gathering the roles: " + err.Error())
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
		Env    map[string]string   `yaml:"env"`         // variables set for the run, e.g. a PATH without a tool; $VAR expands
	} `yaml:"given"`
	Run struct {
		Role    string            `yaml:"role"`
		Target  map[string]int    `yaml:"target"`
		Branch  string            `yaml:"branch"` // the branch the targeted merge request comes from, as --branch
		NoApply bool              `yaml:"no-apply"`
		OpenMR  bool              `yaml:"open-merge-request"`
		PushMR  bool              `yaml:"push-to-merge-request"`
		Event   string            `yaml:"event"`
		Input   map[string]string `yaml:"input"`
		AI      string            `yaml:"ai"`
		Scope   []string          `yaml:"scope"`
		Tamper  string            `yaml:"tamper"`
		Then    string            `yaml:"then"`
		Route   string            `yaml:"route"`
		Item    int               `yaml:"item"`
		Gate    string            `yaml:"gate"`
		Doctor  bool              `yaml:"doctor"`
		Init    bool              `yaml:"init"`
		InitOpt []string          `yaml:"init-options"`  // workline init, with these options
		Setup   []string          `yaml:"setup"`         // workline setup, with these options
		Import  []string          `yaml:"issues-import"` // workline issues import, with these arguments
		Review  []string          `yaml:"review"`        // workline review, with these options
		Sample  []string          `yaml:"sample"`        // workline sample, with these options; then: apply writes what it found
		Reports bool              `yaml:"reports"`       // also write --sarif and --code-quality
		JobSum  bool              `yaml:"summary"`       // also write --summary, Markdown and HTML, the apply after it too
		Forge   string            `yaml:"forge"`         // a forge spec passed as --forge (local, cmd:…), instead of the simulated one
		Follow  []string          `yaml:"follow"`        // workline follow, with these options
	} `yaml:"run"`
	Expect struct {
		Status      string                       `yaml:"status"`
		Findings    []map[string]string          `yaml:"findings"`
		NoFindings  []map[string]string          `yaml:"no-findings"` // findings that must not be there
		Checks      []map[string]string          `yaml:"checks"`      // the doctor's checks, ok ones too, by rule, where, level, message, fix
		AgentCalls  *int                         `yaml:"agent-calls"`
		Applied     []string                     `yaml:"applied"`
		Refused     []string                     `yaml:"refused"`
		Files       map[string]map[string]any    `yaml:"files"`
		Pushed      map[string]map[string]string `yaml:"pushed"`         // branch -> path -> a text the remote's branch holds there
		PushedMsg   map[string]string            `yaml:"pushed-message"` // branch -> a text the message of the remote branch's tip holds
		NotPushed   []string                     `yaml:"not-pushed"`     // branches of the remote the run left where they were
		OnTop       map[string]string            `yaml:"on-top"`         // branch -> the branch of the remote its tip holds
		Forge       map[string]any               `yaml:"forge"`
		Steps       []string                     `yaml:"steps"`
		Calls       []map[string]string          `yaml:"calls"`
		SARIF       []map[string]any             `yaml:"sarif"`         // results, by rule, uri, line, level
		CodeQuality []map[string]any             `yaml:"code-quality"`  // issues, by check_name, path, line, severity
		LeftOut     []string                     `yaml:"left-out"`      // wheres found in neither report
		Notes       []string                     `yaml:"notes"`         // texts the agent's notes hold
		RefusedKept int                          `yaml:"refused-kept"`  // refused answers kept in the run folders
		CallsKept   int                          `yaml:"calls-kept"`    // agent calls recorded in the run folders
		RunFiles    map[string]map[string]string `yaml:"run-files"`     // a file of the run folder -> a text it holds
		Branches    map[string]map[string]string `yaml:"branches"`      // a local branch -> path -> a text it holds there
		Listed      []string                     `yaml:"issues-listed"` // texts `workline issues list` prints afterwards
		Summary     string                       `yaml:"summary"`       // a text the result's summary holds
		SummaryFile []string                     `yaml:"summary-file"`  // texts the --summary file holds, in this order
		SummaryHTML []string                     `yaml:"summary-html"`  // texts the --summary .html file holds, in this order
		Coverage    []map[string]string          `yaml:"coverage"`      // an import's map: its items, by lines, state, issue, words, why
		NotCovered  []map[string]string          `yaml:"not-covered"`   // an import's items left with no issue nor reason
	} `yaml:"expect"`
}

type result struct {
	Status   string `json:"status"`
	Findings []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Message string `json:"message"`
		Fix     string `json:"fix"`
	} `json:"findings"`
	Checks []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Level   string `json:"level"`
		Message string `json:"message"`
		Fix     string `json:"fix"`
	} `json:"checks"`
	AgentCalls int              `json:"agent-calls"`
	Calls      []map[string]any `json:"calls"`
	RunDir     string           `json:"run-dir"`
	Pending    []string         `json:"pending"` // runs a line judged and did not apply
	Notes      []string         `json:"notes"`
	Summary    string           `json:"summary"`
	Coverage   struct {
		Items      []map[string]any `json:"items"`
		NotCovered []map[string]any `json:"not-covered"`
	} `json:"coverage"`
	Applied []string `json:"applied"`
	Refused []string `json:"refused"`
	Steps   []struct {
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
	// The user's config folder stays out: their own lists and defaults are not the case's.
	env := append(hermeticEnv(), "WORKLINE_MODELS_SEEN="+seen, "XDG_CONFIG_HOME="+filepath.Join(work, "config"))
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
	forgeSpec := ""
	if forgeFile != "" {
		forgeSpec = "fake:" + forgeFile
	}
	if c.Run.Forge != "" {
		forgeSpec = c.Run.Forge
	}
	if c.Run.Forge == "gitlab" {
		// GitLab's REST API, simulated over the same file (gitlab_test.go).
		if forgeFile == "" {
			return []string{"run.forge gitlab needs given.forge"}
		}
		srv := httptest.NewServer(&gitlabMock{file: forgeFile})
		defer srv.Close()
		env = append(env, "CI_API_V4_URL="+srv.URL+"/api/v4", "CI_PROJECT_ID=1", "CI_PROJECT_PATH=", "GITLAB_TOKEN=conformance")
	}
	roles := rolesDir
	args := []string{"run-role", c.Run.Role, "--event", c.Run.Event, "--repo", repo, "--roles", roles, "--json"}
	switch {
	case c.Run.Gate != "":
		args = []string{"gate", c.Run.Gate, "--repo", repo, "--json"}
	case c.Run.Doctor:
		args = []string{"doctor", "--repo", repo, "--json"}
	case c.Run.Init:
		args = append([]string{"init", "--repo", repo, "--roles", roles, "--json"}, c.Run.InitOpt...)
	case c.Run.Setup != nil:
		args = append(append([]string{"setup"}, c.Run.Setup...), "--json")
	case c.Run.Review != nil:
		args = append(append([]string{"review"}, c.Run.Review...), "--repo", repo, "--roles", roles, "--json")
	case c.Run.Import != nil:
		args = append(append([]string{"issues", "import"}, c.Run.Import...), "--repo", repo, "--json")
	case c.Run.Sample != nil:
		// The read writes nothing to the forge: it is not given one.
		args = append(append([]string{"sample"}, c.Run.Sample...), "--repo", repo, "--out", filepath.Join(work, "sample.json"), "--json")
		for i, a := range args {
			if strings.HasPrefix(a, "fake:") {
				p, _ := filepath.Abs(filepath.Join("fixtures", "agents", strings.TrimPrefix(a, "fake:")+".yaml"))
				args[i] = "fake:" + p
			}
		}
	case c.Run.Route == "ready" && c.Run.Item != 0:
		args = []string{"item", "ready", fmt.Sprint(c.Run.Item), "--repo", repo, "--json"}
		if forgeSpec != "" {
			args = append(args, "--forge", forgeSpec)
		}
	case c.Run.Route != "":
		args = []string{"route", c.Run.Route, "--repo", repo, "--roles", roles, "--json"}
	case c.Run.Follow != nil:
		args = append(append([]string{"follow"}, c.Run.Follow...), "--repo", repo, "--json")
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
	if forgeSpec != "" && c.Run.Item == 0 && c.Run.Sample == nil {
		args = append(args, "--forge", forgeSpec)
	}
	for kind, id := range c.Run.Target {
		args = append(args, "--target", fmt.Sprintf("%s:%d", kind, id))
	}
	if c.Run.Branch != "" {
		args = append(args, "--branch", c.Run.Branch)
	}
	for _, s := range c.Run.Scope {
		args = append(args, "--scope", s)
	}
	if c.Run.OpenMR {
		args = append(args, "--open-merge-request")
	}
	if c.Run.PushMR {
		args = append(args, "--push-to-merge-request")
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
	summaryFile, summaryHTML := filepath.Join(work, "summary.md"), filepath.Join(work, "summary.html")
	if c.Run.JobSum {
		args = append(args, "--summary", summaryFile, "--summary", summaryHTML)
	}
	// Judged on one machine, applied on another (CI's two jobs): each its own
	// cache, the first gone by the time the second applies.
	// With the roles built into the engine, as CI installs it: no --roles.
	judgeCache := ""
	if c.Run.Then == "resume-elsewhere" {
		judgeCache = t.TempDir()
		env = append(env, "XDG_CACHE_HOME="+judgeCache)
		for i, a := range args {
			if a == "--roles" {
				args = append(args[:i:i], args[i+2:]...)
				break
			}
		}
	}
	cmd := exec.Command(engineBin, args...)
	cmd.Dir = repo
	cmd.Env = env
	if c.Run.Setup != nil { // the machine's git config, which setup changes, is the case's own
		global := filepath.Join(work, "gitconfig")
		os.WriteFile(global, nil, 0o644)
		cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL="+global)
	}
	for k, v := range c.Given.Env {
		cmd.Env = append(cmd.Env, k+"="+os.Expand(v, func(name string) string { return lookup(cmd.Env, name) }))
	}
	tips := map[string]string{} // the remote's branches before the run
	for _, b := range c.Expect.NotPushed {
		out, _ := exec.Command("git", "-C", repo, "ls-remote", "origin", "refs/heads/"+b).Output()
		tips[b] = string(out)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run() // the exit code mirrors the status, which is checked below
	var r result
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		return []string{fmt.Sprintf("engine printed no result: %v\n%s", err, stderr.String())}
	}
	if c.Run.Then == "resume" || c.Run.Then == "resume-elsewhere" {
		dirs := r.Pending
		if len(dirs) == 0 && r.RunDir != "" {
			dirs = []string{r.RunDir}
		}
		if len(dirs) == 0 {
			return []string{"the first run reported no run folder to resume"}
		}
		resumeArgs := append(append([]string{"apply"}, dirs...), "--json")
		if c.Run.JobSum {
			resumeArgs = append(resumeArgs, "--summary", summaryFile, "--summary", summaryHTML)
		}
		resume := exec.Command(engineBin, resumeArgs...)
		resume.Env = env
		if judgeCache != "" {
			os.RemoveAll(judgeCache)
			resume.Env = append(env, "XDG_CACHE_HOME="+t.TempDir())
		}
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
	if c.Run.Sample != nil && c.Run.Then == "apply" {
		// The write, as CI's job holding the forge's token and no AI key.
		applyArgs := []string{"sample", "--apply", filepath.Join(work, "sample.json"), "--repo", repo, "--forge", forgeSpec, "--json"}
		if c.Run.JobSum {
			applyArgs = append(applyArgs, "--summary", summaryFile, "--summary", summaryHTML)
		}
		apply := exec.Command(engineBin, applyArgs...)
		apply.Dir, apply.Env = repo, cmd.Env
		stdout.Reset()
		apply.Stdout, apply.Stderr = &stdout, &stderr
		_ = apply.Run()
		var r2 result
		if err := json.Unmarshal(stdout.Bytes(), &r2); err != nil {
			return []string{fmt.Sprintf("sample --apply printed no result: %v\n%s", err, stderr.String())}
		}
		r2.AgentCalls += r.AgentCalls
		r2.Calls = append(r.Calls, r2.Calls...)
		r2.Findings = append(r.Findings, r2.Findings...)
		r = r2
	}
	problems := compare(c, &r, repo)
	for b, before := range tips {
		if out, _ := exec.Command("git", "-C", repo, "ls-remote", "origin", "refs/heads/"+b).Output(); string(out) != before {
			problems = append(problems, fmt.Sprintf("origin's %s moved: %q, was %q", b, out, before))
		}
	}
	if len(c.Expect.Listed) > 0 {
		list := exec.Command(engineBin, "issues", "list", "--repo", repo)
		list.Env = env
		out, err := list.CombinedOutput()
		for _, text := range c.Expect.Listed {
			if err != nil || !strings.Contains(string(out), text) {
				problems = append(problems, fmt.Sprintf("workline issues list does not print %q (%v):\n%s", text, err, out))
			}
		}
	}
	if c.Run.Reports {
		problems = append(problems, compareReports(c, sarifFile, cqFile)...)
	}
	if len(c.Expect.SummaryFile) > 0 {
		problems = append(problems, compareSummaryFile(c.Expect.SummaryFile, summaryFile)...)
	}
	if len(c.Expect.SummaryHTML) > 0 {
		problems = append(problems, compareSummaryFile(c.Expect.SummaryHTML, summaryHTML)...)
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
			if f.Rule == want["rule"] && strings.Contains(f.Where, want["where"]) && strings.Contains(f.Message, want["message"]) && strings.Contains(f.Fix, want["fix"]) {
				found = true
			}
		}
		if !found {
			p = append(p, fmt.Sprintf("missing finding %v (got %v)", want, r.Findings))
		}
	}
	for _, want := range e.Checks {
		found := false
		for _, ch := range r.Checks {
			if ch.Rule == want["rule"] && (want["level"] == "" || ch.Level == want["level"]) && strings.Contains(ch.Where, want["where"]) &&
				strings.Contains(ch.Message, want["message"]) && strings.Contains(ch.Fix, want["fix"]) {
				found = true
			}
		}
		if !found {
			p = append(p, fmt.Sprintf("missing check %v (got %v)", want, r.Checks))
		}
	}
	for _, bad := range e.NoFindings {
		for _, f := range r.Findings {
			if f.Rule == bad["rule"] && f.Where == bad["where"] && strings.Contains(f.Message, bad["message"]) {
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
	for branch, files := range e.Pushed {
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		for path, text := range files {
			out, err := exec.Command("git", "-C", repo, "show", "origin/"+branch+":"+path).Output()
			if err != nil || !strings.Contains(string(out), text) {
				p = append(p, fmt.Sprintf("origin's %s does not hold %q in %s", branch, text, path))
			}
		}
	}
	for branch, files := range e.Branches {
		for path, text := range files {
			out, err := exec.Command("git", "-C", repo, "show", "refs/heads/"+branch+":"+path).Output()
			if err != nil || !strings.Contains(string(out), text) {
				p = append(p, fmt.Sprintf("the local branch %s does not hold %q in %s", branch, text, path))
			}
		}
	}
	for branch, text := range e.PushedMsg {
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%B", "origin/"+branch).Output()
		if err != nil || !strings.Contains(string(out), text) {
			p = append(p, fmt.Sprintf("the tip of origin's %s does not say %q in its message: %s", branch, text, out))
		}
	}
	for branch, base := range e.OnTop {
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		if exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", "origin/"+base, "origin/"+branch).Run() != nil {
			p = append(p, fmt.Sprintf("origin's %s is not on top of origin's %s", branch, base))
		}
	}
	if e.RefusedKept > 0 {
		kept, _ := filepath.Glob(filepath.Join(repo, ".git", "workline", "runs", "*", "out", "refused-*.yaml"))
		if len(kept) != e.RefusedKept {
			p = append(p, fmt.Sprintf("%d refused answers kept, want %d", len(kept), e.RefusedKept))
		}
	}
	if e.CallsKept > 0 {
		n := 0
		files, _ := filepath.Glob(filepath.Join(repo, ".git", "workline", "runs", "*", "out", "calls.jsonl"))
		for _, f := range files {
			data, _ := os.ReadFile(f)
			n += strings.Count(string(data), "\n")
		}
		if n != e.CallsKept {
			p = append(p, fmt.Sprintf("%d agent calls kept, want %d", n, e.CallsKept))
		}
	}
	for name, want := range e.RunFiles {
		found, _ := filepath.Glob(filepath.Join(repo, ".git", "workline", "runs", "*", filepath.FromSlash(name)))
		if len(found) == 0 {
			p = append(p, fmt.Sprintf("no run folder keeps %s", name))
			continue
		}
		if text, ok := want["contains"]; ok {
			held := false
			for _, f := range found {
				data, _ := os.ReadFile(f)
				held = held || strings.Contains(string(data), text)
			}
			if !held {
				p = append(p, fmt.Sprintf("%s does not hold %q", name, text))
			}
		}
		if text, ok := want["not-contains"]; ok {
			for _, f := range found {
				if data, _ := os.ReadFile(f); strings.Contains(string(data), text) {
					p = append(p, fmt.Sprintf("%s holds %q", name, text))
				}
			}
		}
	}
	if e.Summary != "" && !strings.Contains(r.Summary, e.Summary) {
		p = append(p, fmt.Sprintf("summary %q does not hold %q", r.Summary, e.Summary))
	}
	p = append(p, mapped("coverage", e.Coverage, r.Coverage.Items)...)
	p = append(p, mapped("not-covered", e.NotCovered, r.Coverage.NotCovered)...)
	for _, text := range e.Notes {
		if !strings.Contains(strings.Join(r.Notes, "\n"), text) {
			p = append(p, fmt.Sprintf("no note holds %q (notes: %q)", text, r.Notes))
		}
	}
	for path, want := range e.Files {
		data, err := os.ReadFile(filepath.Join(repo, path))
		if exists, ok := want["exists"].(bool); ok && exists != (err == nil) {
			p = append(p, fmt.Sprintf("%s exists = %v, want %v", path, err == nil, exists))
		}
		texts, _ := want["contains"].([]any) // one text, or a list
		if text, ok := want["contains"].(string); ok {
			texts = []any{text}
		}
		for _, text := range texts {
			if !strings.Contains(string(data), fmt.Sprint(text)) {
				p = append(p, fmt.Sprintf("%s does not contain %q", path, text))
			}
		}
		if text, ok := want["lacks"].(string); ok && strings.Contains(string(data), text) {
			p = append(p, fmt.Sprintf("%s contains %q", path, text))
		}
	}
	return p
}

// linkRoles makes dir a folder of roles: a link to each role of each source.
func linkRoles(dir string, sources ...string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, src := range sources {
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			abs, _ := filepath.Abs(filepath.Join(src, e.Name()))
			if err := os.Symlink(abs, filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
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

// lookup is the value a variable has in env, the last one set winning.
func lookup(env []string, name string) string {
	v := ""
	for _, kv := range env {
		if k, val, ok := strings.Cut(kv, "="); ok && k == name {
			v = val
		}
	}
	return v
}

// hermeticEnv keeps the machine's git config and hooks out of the tests,
// and the CI the suite may run in: a case sets CI's variables itself.
func hermeticEnv() []string {
	fixtures, _ := filepath.Abs("fixtures")
	var env []string
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); name != "CI" && name != "GITHUB_ACTIONS" && name != "GITLAB_CI" {
			env = append(env, kv)
		}
	}
	return append(env, "FIXTURES="+fixtures, // for a cmd: agent's script
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
// id, `comments` is a count, `labels` the exact set, `comment-contains` /
// `comment-lacks` texts some comment holds, or none does, `branch`, `base`
// `title`, the `reason` it was closed for, its `milestone` and the `parent`
// it is a sub-issue of (0 for none) an item's, `blocked-by` the issues it
// waits on in the forge's own relation, `closed` whether it
// is, `body-contains` a text its body holds, and
// `absent: true` no item with that id.
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
			if absent, _ := wm["absent"].(bool); absent {
				if found != nil {
					p = append(p, fmt.Sprintf("forge: %s %v exists, want none", kind, wm["id"]))
				}
				continue
			}
			if found == nil {
				p = append(p, fmt.Sprintf("forge: no %s with id %v", kind, wm["id"]))
				continue
			}
			if t, ok := wm["body-contains"]; ok && !strings.Contains(fmt.Sprint(found["body"]), fmt.Sprint(t)) {
				p = append(p, fmt.Sprintf("forge: %s %v body = %q, want it to hold %q", kind, wm["id"], found["body"], t))
			}
			if t, ok := wm["body-lacks"]; ok && strings.Contains(fmt.Sprint(found["body"]), fmt.Sprint(t)) {
				p = append(p, fmt.Sprintf("forge: %s %v body = %q, want it without %q", kind, wm["id"], found["body"], t))
			}
			if w, ok := wm["closed"]; ok && fmt.Sprint(found["closed"] == true) != fmt.Sprint(w) {
				p = append(p, fmt.Sprintf("forge: %s %v closed = %v, want %v", kind, wm["id"], found["closed"] == true, w))
			}
			if found["parent"] == nil {
				found["parent"] = 0 // not a sub-issue
			}
			for _, k := range []string{"branch", "base", "title", "reason", "milestone", "parent"} {
				if w, ok := wm[k]; ok && fmt.Sprint(found[k]) != fmt.Sprint(w) {
					p = append(p, fmt.Sprintf("forge: %s %v %s = %v, want %v", kind, wm["id"], k, found[k], w))
				}
			}
			if n, ok := wm["comments"]; ok {
				c, _ := found["comments"].([]any)
				if fmt.Sprint(len(c)) != fmt.Sprint(n) {
					p = append(p, fmt.Sprintf("forge: %s %v has %d comments, want %v", kind, wm["id"], len(c), n))
				}
			}
			comments, _ := found["comments"].([]any)
			held := func(text string) bool {
				for _, c := range comments {
					if strings.Contains(fmt.Sprint(c), text) {
						return true
					}
				}
				return false
			}
			if t, ok := wm["comment-contains"]; ok && !held(fmt.Sprint(t)) {
				p = append(p, fmt.Sprintf("forge: no comment on %s %v holds %q (comments: %v)", kind, wm["id"], t, comments))
			}
			if t, ok := wm["comment-lacks"]; ok && held(fmt.Sprint(t)) {
				p = append(p, fmt.Sprintf("forge: a comment on %s %v holds %q", kind, wm["id"], t))
			}
			if w, ok := wm["blocked-by"]; ok {
				gb, _ := found["blocked-by"].([]any)
				if fmt.Sprint(gb) != fmt.Sprint(w) {
					p = append(p, fmt.Sprintf("forge: %s %v blocked-by = %v, want %v", kind, wm["id"], gb, w))
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

// compareSummaryFile checks the --summary file holds each text, in order.
func compareSummaryFile(want []string, file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return []string{fmt.Sprintf("no summary file: %v", err)}
	}
	rest := string(data)
	for _, w := range want {
		i := strings.Index(rest, w)
		if i < 0 {
			return []string{fmt.Sprintf("the summary file does not hold %q after what came before:\n%s", w, data)}
		}
		rest = rest[i+len(w):]
	}
	return nil
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

// mapped says which wanted entries of an import's map no entry matches:
// each key equal, but words and why, which an entry need only hold.
func mapped(name string, want []map[string]string, got []map[string]any) []string {
	var p []string
	for _, w := range want {
		found := false
		for _, g := range got {
			ok := true
			for k, v := range w {
				have := fmt.Sprint(g[k])
				if k == "words" || k == "why" {
					ok = ok && strings.Contains(have, v)
				} else {
					ok = ok && have == v
				}
			}
			found = found || ok
		}
		if !found {
			p = append(p, fmt.Sprintf("%s: no entry %v (got %v)", name, w, got))
		}
	}
	return p
}
