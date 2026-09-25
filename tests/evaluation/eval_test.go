// Package evaluation grades roles with a real agent, on real cases, as
// docs/spec/conformance.md ("Evaluation") describes. Conformance proves the
// engine obeys the contract; evaluation measures how well a role does its job
// with a given model, and keeps the score over time in results.tsv.
//
// It calls a real agent and costs tokens, so it runs only when asked:
//
//	WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/
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
	Findings   []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Message string `json:"message"`
	} `json:"findings"`
	Applied []string `json:"applied"`
}

// run is what a case produced, for the checks to read.
type run struct {
	repo, message, headBefore string
	before                    map[string]string // files as they were before the run, when a check needs them
	res                       result
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
			passed, failed := grade(&c, r)
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
			line := strings.Join([]string{time.Now().UTC().Format(time.RFC3339), version, os.Getenv("WORKLINE_EVAL"), c.Case,
				score, fmt.Sprint(r.res.AgentCalls), fmt.Sprintf("%.0f", time.Since(start).Seconds()), strings.Join(failed, "; ")}, "\t")
			record.Lock()
			defer record.Unlock()
			out, err := os.OpenFile("results.tsv", os.O_APPEND|os.O_WRONLY, 0o644)
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

// grade runs a case's checks; each one is one point.
func grade(c *caseFile, r *run) (int, []string) {
	passed := 0
	var failed []string
	for _, g := range c.Grade {
		for kind, v := range g {
			if why := check(kind, v, c, r); why != "" {
				failed = append(failed, kind+": "+why)
			} else {
				passed++
			}
		}
	}
	return passed, failed
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

// worklineVersion is the commit being evaluated, marked when the tree has changes.
func worklineVersion() string {
	head, _ := exec.Command("git", "-C", "../..", "rev-parse", "--short", "HEAD").Output()
	v := strings.TrimSpace(string(head))
	if dirty, _ := exec.Command("git", "-C", "../..", "status", "--porcelain", "--", "cmd", "internal", "roles").Output(); len(dirty) > 0 {
		v += "+changes"
	}
	return v
}
