// Package evaluation grades roles with a real agent, on real cases, as
// docs/spec/conformance.md ("Evaluation") describes. Conformance proves the
// engine obeys the contract; evaluation measures how well a role does its job
// with a given model, and keeps the score over time in results.tsv.
//
// It calls a real agent and costs tokens, so it runs only when asked:
//
//	WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/
//
// WORKLINE_JUDGE names the agent grading the `judge` checks, of another
// provider than WORKLINE_EVAL's; without one, they are skipped.
package evaluation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
)

type caseFile struct {
	Case  string `yaml:"case"`
	About string `yaml:"about"`
	Given struct {
		Repo           string         `yaml:"repo"`            // a conformance fixture
		Setup          []string       `yaml:"setup"`           // shell lines run after it is built
		Config         map[string]any `yaml:"config"`          // .workline/config.yaml
		WorklineCommit string         `yaml:"workline-commit"` // this repository: that commit's diff, staged on its parent
		WorklineAt     string         `yaml:"workline-at"`     // this repository, at that commit
	} `yaml:"given"`
	Run struct {
		Role    string            `yaml:"role"`
		Event   string            `yaml:"event"`
		Message string            `yaml:"message"` // commit-msg: the message the author wrote
		Input   map[string]string `yaml:"input"`
	} `yaml:"run"`
	Grade []map[string]any `yaml:"grade"`
}

type result struct {
	Status     string `json:"status"`
	AgentCalls int    `json:"agent-calls"`
	Calls      []call `json:"calls"`
	Findings   []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Message string `json:"message"`
	} `json:"findings"`
	Applied []string `json:"applied"`
}

type call struct {
	Effort    string  `json:"effort"`
	Model     string  `json:"model"`
	TokensIn  int     `json:"tokens-in"`
	TokensOut int     `json:"tokens-out"`
	CostUSD   float64 `json:"cost-usd"`
}

// answeredBy says which models answered, in order, `>` marking a step up, the
// efforts asked, and what the calls used: a score means little without them.
func answeredBy(calls []call) (models, efforts, tokensIn, tokensOut, cost string) {
	var m, e []string
	total, in, out := 0.0, 0, 0
	for _, c := range calls {
		if len(m) == 0 || m[len(m)-1] != c.Model {
			m = append(m, c.Model)
		}
		if len(e) == 0 || e[len(e)-1] != c.Effort {
			e = append(e, c.Effort)
		}
		total += c.CostUSD
		in += c.TokensIn
		out += c.TokensOut
	}
	return strings.Join(m, ">"), strings.Join(e, ">"), fmt.Sprint(in), fmt.Sprint(out), fmt.Sprintf("%.4f", total)
}

// run is what a case produced, for the checks to read.
type run struct {
	repo, message, headBefore string
	before                    map[string]string // files as they were before the run, when a check needs them
	res                       result
	judgedBy                  string // the model that answered the judge checks
}

var (
	engineBin string
	record    sync.Mutex
)

func TestMain(m *testing.M) {
	if os.Getenv("WORKLINE_EVAL") == "" {
		os.Exit(m.Run()) // the checks' own tests only
	}
	dir, err := os.MkdirTemp("", "workline-eval-")
	if err != nil {
		panic(err)
	}
	engineBin = filepath.Join(dir, "workline")
	build := exec.Command("go", "build", "-o", engineBin, "../../cmd/workline")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("building the engine: " + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func TestEvaluation(t *testing.T) {
	if os.Getenv("WORKLINE_EVAL") == "" {
		t.Skip("set WORKLINE_EVAL=claude to run it: it calls a real agent and costs tokens")
	}
	files, _ := filepath.Glob("cases/*/*.yaml")
	sort.Strings(files)
	version := worklineVersion()
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
			t.Parallel()
			start := time.Now()
			r, err := play(t, &c)
			if err != nil {
				t.Fatal(err)
			}
			passed, failed, skipped := grade(&c, r)
			score := fmt.Sprintf("%d/%d", passed, passed+len(failed))
			t.Logf("%s — %s, %d agent calls; failed: %s", c.Case, score, r.res.AgentCalls, strings.Join(failed, "; "))
			if len(failed) > 0 { // what the run said, to see why
				t.Logf("  applied: %v", r.res.Applied)
				for _, f := range r.res.Findings {
					if f.Rule != "links-not-checked" {
						t.Logf("  %s %s: %.300s", f.Rule, f.Where, f.Message)
					}
				}
			}
			models, efforts, tokensIn, tokensOut, cost := answeredBy(r.res.Calls)
			line := strings.Join([]string{time.Now().UTC().Format(time.RFC3339), version, os.Getenv("WORKLINE_EVAL"), models, efforts,
				c.Case, score, fmt.Sprint(r.res.AgentCalls), tokensIn, tokensOut, cost, fmt.Sprintf("%.0f", time.Since(start).Seconds()),
				strings.Join(append(failed, skipped...), "; "), r.judgedBy}, "\t")
			record.Lock()
			defer record.Unlock()
			results := "results.tsv"
			if f := os.Getenv("WORKLINE_EVAL_RESULTS"); f != "" {
				results = f // a scheduled run, from another checkout, keeps the same history
			}
			out, err := os.OpenFile(results, os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			fmt.Fprintln(out, line)
		})
	}
}

// play builds the case's repository and runs the role once, with the agent.
func play(t *testing.T, c *caseFile) (*run, error) {
	work := t.TempDir()
	repo := filepath.Join(work, "repo")
	env := hermetic()
	self, _ := filepath.Abs("../..")
	switch {
	case c.Given.WorklineCommit != "":
		sha := c.Given.WorklineCommit
		if err := sh(work, env, "git clone -q "+self+" repo && cd repo && git checkout -q "+sha+"^ && git -C "+self+" diff "+sha+"^ "+sha+" | git apply --index"); err != nil {
			return nil, err
		}
	case c.Given.WorklineAt != "":
		if err := sh(work, env, "git clone -q "+self+" repo && cd repo && git checkout -q "+c.Given.WorklineAt); err != nil {
			return nil, err
		}
	default:
		script, _ := filepath.Abs(filepath.Join("..", "conformance", "fixtures", "repos", c.Given.Repo+".sh"))
		if err := os.MkdirAll(repo, 0o755); err != nil {
			return nil, err
		}
		for _, s := range append([]string{"sh " + script}, c.Given.Setup...) {
			if err := sh(repo, env, s); err != nil {
				return nil, err
			}
		}
	}
	if c.Given.Config != nil {
		data, _ := yaml.Marshal(c.Given.Config)
		os.MkdirAll(filepath.Join(repo, ".workline"), 0o755)
		os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), data, 0o644)
	}
	r := &run{repo: repo, before: map[string]string{}}
	head, _ := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	r.headBefore = strings.TrimSpace(string(head))
	for _, g := range c.Grade {
		if p, ok := g["body-unchanged"].(string); ok {
			data, _ := os.ReadFile(filepath.Join(repo, p))
			r.before[p] = string(data)
		}
	}
	roles, _ := filepath.Abs("../../roles")
	args := []string{"run-role", c.Run.Role, "--event", c.Run.Event, "--repo", repo, "--roles", roles, "--ai", os.Getenv("WORKLINE_EVAL"), "--json"}
	msgFile := filepath.Join(work, "message")
	if c.Run.Message != "" {
		if err := os.WriteFile(msgFile, []byte(c.Run.Message+"\n"), 0o644); err != nil {
			return nil, err
		}
		args = append(args, "--input-file", "message="+msgFile)
	}
	for k, v := range c.Run.Input {
		args = append(args, "--input", k+"="+v)
	}
	cmd := exec.Command(engineBin, args...)
	cmd.Env = env
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	_ = cmd.Run()
	if err := json.Unmarshal(out.Bytes(), &r.res); err != nil {
		return nil, fmt.Errorf("the engine printed no result: %v\n%s", err, errOut.String())
	}
	if data, err := os.ReadFile(msgFile); err == nil {
		r.message = strings.TrimSpace(string(data))
	}
	return r, nil
}

// grade runs a case's checks; each one is one point. A judge check that
// could not be asked is skipped: it is no point, earned or lost.
func grade(c *caseFile, r *run) (passed int, failed, skipped []string) {
	for _, g := range c.Grade {
		for kind, v := range g {
			if kind == "judge" {
				why, err := judge(fmt.Sprint(v), c, r)
				switch {
				case err != nil:
					skipped = append(skipped, "judge skipped: "+err.Error())
				case why != "":
					failed = append(failed, "judge: "+why)
				default:
					passed++
				}
				continue
			}
			if why := check(kind, v, c, r); why != "" {
				failed = append(failed, kind+": "+why)
			} else {
				passed++
			}
		}
	}
	return passed, failed, skipped
}

// vague are the words a rewrite reaches for when it drops the author's.
var vague = []string{"refactor", "update", "updates", "improve", "improves", "enhance", "misc", "changes"}

func check(kind string, v any, c *caseFile, r *run) string {
	switch kind {
	case "status":
		if r.res.Status != fmt.Sprint(v) {
			return fmt.Sprintf("status %s", r.res.Status)
		}
	case "agent-calls-max":
		if n, _ := v.(int); r.res.AgentCalls > n {
			return fmt.Sprintf("%d calls", r.res.AgentCalls)
		}
	case "subject-max":
		n, _ := v.(int)
		if subject := strings.SplitN(r.message, "\n", 2)[0]; len([]rune(subject)) > n {
			return fmt.Sprintf("%q is %d characters", subject, len([]rune(subject)))
		}
	case "keeps-words":
		min, _ := v.(float64)
		if kept := keptWords(c.Run.Message, r.message); kept < min {
			return fmt.Sprintf("%.0f%% of the author's words kept in %q", kept*100, firstLine(r.message))
		}
	case "no-vague-words":
		for _, w := range vague {
			if hasWord(r.message, w) && !hasWord(c.Run.Message, w) {
				return fmt.Sprintf("%q brought in, in %q", w, firstLine(r.message))
			}
		}
	case "file-contains", "file-lacks":
		for path, text := range v.(map[string]any) {
			data, _ := os.ReadFile(filepath.Join(r.repo, path))
			if strings.Contains(string(data), fmt.Sprint(text)) != (kind == "file-contains") {
				return fmt.Sprintf("%s and %q", path, text)
			}
		}
	case "checked-is-head":
		data, _ := os.ReadFile(filepath.Join(r.repo, fmt.Sprint(v)))
		m := regexp.MustCompile(`(?m)^checked: ([0-9a-f]{7,40})$`).FindStringSubmatch(string(data))
		if m == nil || !strings.HasPrefix(r.headBefore, m[1]) {
			return fmt.Sprintf("%s does not record %.7s", v, r.headBefore)
		}
	case "body-unchanged":
		p := fmt.Sprint(v)
		data, _ := os.ReadFile(filepath.Join(r.repo, p))
		if body(string(data)) != body(r.before[p]) {
			return p + " was reworded"
		}
	case "lines-max":
		for path, n := range v.(map[string]any) {
			data, _ := os.ReadFile(filepath.Join(r.repo, path))
			if got := strings.Count(string(data), "\n"); got > n.(int) {
				return fmt.Sprintf("%s has %d lines", path, got)
			}
		}
	case "new-files-min":
		out, _ := exec.Command("git", "-C", r.repo, "ls-files", "--others", "--exclude-standard").Output()
		if n, _ := v.(int); len(strings.Fields(string(out))) < n {
			return fmt.Sprintf("%d new files", len(strings.Fields(string(out))))
		}
	default:
		return "unknown check"
	}
	return ""
}

// keptWords is the share of the author's subject words found in the rewrite.
func keptWords(original, rewrite string) float64 {
	words := func(s string) []string {
		return strings.FieldsFunc(strings.ToLower(firstLine(s)), func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r > 127)
		})
	}
	has := map[string]bool{}
	for _, w := range words(rewrite) {
		has[w] = true
	}
	orig := words(original)
	kept := 0
	for _, w := range orig {
		if has[w] {
			kept++
		}
	}
	if len(orig) == 0 {
		return 1
	}
	return float64(kept) / float64(len(orig))
}

func hasWord(s, w string) bool {
	return regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b`).MatchString(firstLine(s))
}

func firstLine(s string) string { return strings.SplitN(s, "\n", 2)[0] }

// body drops a doc's frontmatter.
func body(s string) string {
	if rest, ok := strings.CutPrefix(s, "---\n"); ok {
		if _, after, ok := strings.Cut(rest, "\n---\n"); ok {
			return after
		}
	}
	return s
}

func sh(dir string, env []string, script string) error {
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir, cmd.Env = dir, env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%q: %v\n%s", script, err, out)
	}
	return nil
}

// hermetic keeps the machine's git config and hooks out, as the conformance
// suite does; the agent's own login stays, since it is what is evaluated.
func hermetic() []string {
	return append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
		"GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid",
		"GIT_AUTHOR_DATE=2026-01-03T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-03T00:00:00Z",
	)
}

// evaluated are the paths a score depends on: the engine, the roles (not
// their READMEs, which no agent reads) and the cases. A commit touching
// nothing else does not change what is evaluated.
var evaluated = []string{"cmd", "internal", "roles", ":(exclude)roles/*/README.md", "go.mod", "tests/evaluation/cases"}

// worklineVersion is the last commit changing what is evaluated, marked when
// the tree has changes there. tests/evaluation/schedule/run.sh counts runs by it.
func worklineVersion() string {
	last, _ := exec.Command("git", append([]string{"-C", "../..", "log", "-1", "--format=%h", "--"}, evaluated...)...).Output()
	v := strings.TrimSpace(string(last))
	if dirty, _ := exec.Command("git", append([]string{"-C", "../..", "status", "--porcelain", "--"}, evaluated...)...).Output(); len(dirty) > 0 {
		v += "+changes"
	}
	return v
}

// provider is the agent a spec names: claude for claude:haiku@low. A command
// is whatever it runs; whoever names one vouches for its provider.
func provider(spec string) string {
	p, _, _ := strings.Cut(spec, ":")
	return p
}

// judge asks the agent WORKLINE_JUDGE names a yes-or-no question on what the
// role produced: "" when it says yes, its reason when it says no. It never
// shares the graded agent's provider, which shares its blind spots
// (docs/spec/model-grid.md, "Independent judgement").
func judge(question string, c *caseFile, r *run) (string, error) {
	spec := os.Getenv("WORKLINE_JUDGE")
	if spec == "" || spec == "none" {
		return "", fmt.Errorf("no WORKLINE_JUDGE")
	}
	if p := provider(spec); p != "cmd" && p == provider(os.Getenv("WORKLINE_EVAL")) {
		return "", fmt.Errorf("WORKLINE_JUDGE is of the graded agent's provider, %s", p)
	}
	ag, err := agent.Parse(spec)
	if err != nil {
		return "", err
	}
	judgeRole, err := role.Load(".", "judge")
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "workline-judge-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, "in"), 0o755)
	os.MkdirAll(filepath.Join(dir, "out"), 0o755)
	var task strings.Builder
	fmt.Fprintf(&task, "## Question\n\n%s\n\n## The case\n\n%s\n\n", question, c.About)
	if c.Run.Message != "" {
		fmt.Fprintf(&task, "## What the author wrote\n\n```\n%s\n```\n\n## What the role made of it\n\n```\n%s\n```\n\n", c.Run.Message, r.message)
	}
	_ = exec.Command("git", "-C", r.repo, "add", "--intent-to-add", ".").Run() // new docs show in the diff
	diff, _ := exec.Command("git", "-C", r.repo, "diff", "HEAD").Output()
	if len(diff) > 60000 {
		diff = append(diff[:60000], "\n[cut]\n"...)
	}
	fmt.Fprintf(&task, "## The change, as the working tree holds it after the role\n\n```diff\n%s```\n", diff)
	if err := os.WriteFile(filepath.Join(dir, "in", "task.md"), []byte(task.String()), 0o644); err != nil {
		return "", err
	}
	call, err := ag.Propose(agent.Request{RunDir: dir, Repo: dir, Role: judgeRole})
	r.judgedBy = call.Model
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "out", "intentions.yaml"))
	if err != nil {
		return "", err
	}
	var notes []map[string]string
	if err := yaml.Unmarshal(data, &notes); err != nil || len(notes) == 0 || notes[0]["note"] == "" {
		return "", fmt.Errorf("the judge answered no note: %.200s", data)
	}
	verdict, why, _ := strings.Cut(notes[0]["note"], ":")
	switch strings.ToLower(strings.TrimSpace(verdict)) {
	case "yes":
		return "", nil
	case "no":
		return strings.Join(strings.Fields(why), " "), nil // one line of results.tsv
	}
	return "", fmt.Errorf("the judge said neither yes nor no: %.200s", notes[0]["note"])
}
