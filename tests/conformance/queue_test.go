package conformance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// A merge queue's or train's run guards the merged result with the gates
// alone, never an agent (ADR-0050); the checks the queue requires report
// on its commit, or it waits until its timeout. The head's review and
// verdict, and the person's yes, stay the merge request's (ADR-0042):
// nothing is applied, kept or followed from the queue's run.

// A queue's group as GitHub gives it to a workflow (`merge_group`), and a
// pull request's push, for the same workflow read both ways.
var (
	mergeGroup = map[string]any{
		"github.event_name":                 "merge_group",
		"github.repository":                 "owner/repo",
		"github.event.merge_group.base_sha": "b0b0",
		"github.event.merge_group.head_sha": "4e4d",
		"github.event.merge_group.head_ref": "gh-readonly-queue/main/pr-7-c0c0",
	}
	pullRequestPush = map[string]any{
		"github.event_name":                             "pull_request",
		"github.event.action":                           "synchronize",
		"github.repository":                             "owner/repo",
		"github.event.pull_request.number":              "7",
		"github.event.pull_request.draft":               false,
		"github.event.pull_request.base.sha":            "b0b0",
		"github.event.pull_request.head.sha":            "c0c0",
		"github.event.pull_request.head.repo.full_name": "owner/repo",
	}
)

func TestGitHubMergeQueueRunsTheGatesAlone(t *testing.T) {
	type step struct {
		Name string         `yaml:"name"`
		ID   string         `yaml:"id"`
		Uses string         `yaml:"uses"`
		With map[string]any `yaml:"with"`
		Env  map[string]any `yaml:"env"`
		If   string         `yaml:"if"`
		Run  string         `yaml:"run"`
	}
	type job struct {
		If          string            `yaml:"if"`
		Concurrency map[string]any    `yaml:"concurrency"`
		Env         map[string]string `yaml:"env"`
		Steps       []step            `yaml:"steps"`
	}
	data, err := os.ReadFile("../../ci/github/workline.yml")
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		On          map[string]any `yaml:"on"`
		Concurrency map[string]any `yaml:"concurrency"`
		Jobs        map[string]job `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(wf.On["merge_group"]) != "map[types:[checks_requested]]" {
		t.Fatalf("workline.yml: no merge_group trigger (%v): a queue's required checks would never report", wf.On["merge_group"])
	}
	// One run per group, never sharing a pull request's lock: GitHub keeps
	// one run pending per group, and cancels the one pending before.
	for where, c := range map[string]map[string]any{"the workflow": wf.Concurrency, "judge": wf.Jobs["judge"].Concurrency} {
		group := expand(t, fmt.Sprint(c["group"]), mergeGroup)
		if !strings.HasSuffix(group, "gh-readonly-queue/main/pr-7-c0c0") {
			t.Errorf("workline.yml, %s: a queue's group runs in concurrency group %q, shared by every group", where, group)
		}
	}
	for _, c := range []struct {
		name  string
		event map[string]any
		head  string
		// route: what each job's line is run with, and never with
		route, never map[string][]string
	}{
		{
			name: "a merge queue's group", event: mergeGroup, head: "4e4d",
			route: map[string][]string{
				"gates": {"--gates-only", "--input queue=true", "--input range=b0b0..4e4d"},
				"judge": {"--ai none", "--forge none", "--input queue=true", "--input range=b0b0..4e4d", "--gates-from"},
			},
			never: map[string][]string{"judge": {"--verdict", "--target", "--push-to-merge-request", "lenses=all", "--sarif"}},
		},
		{
			name: "a pull request's push", event: pullRequestPush, head: "c0c0",
			route: map[string][]string{
				"gates": {"--gates-only", "--input range=b0b0..c0c0"},
				"judge": {"--ai \"$ai\"", "--target merge-request:7", "--input range=b0b0..c0c0", "--verdict workline-verdict.json"},
			},
			never: map[string][]string{"gates": {"queue="}, "judge": {"queue="}},
		},
	} {
		// The repository gives the agent's token, the App's key elsewhere;
		// the line has a gate, which ran.
		ctx := map[string]any{"env.HAS_CLAUDE": "true", "env.APP_KEY_IN_REACH": "false",
			"steps.any.outputs.any": "true", "steps.gates.outputs.ran": "true", "needs.gates.outputs.ran": "true"}
		for k, v := range c.event {
			ctx[k] = v
		}
		for _, name := range []string{"gates", "judge"} {
			j := wf.Jobs[name]
			if j.If != "" && !truthy(eval(t, j.If, ctx)) {
				t.Errorf("%s: job %s does not run (if %q): the check the queue requires never reports", c.name, name, j.If)
				continue
			}
			routes := 0
			for _, s := range j.Steps {
				if s.If != "" && !truthy(eval(t, s.If, ctx)) {
					continue
				}
				what := s.Name
				if what == "" {
					what = s.Uses
				}
				run := expand(t, s.Run, ctx)
				if strings.HasPrefix(s.Uses, "actions/checkout@") {
					if ref := expand(t, fmt.Sprint(s.With["ref"]), ctx); ref != c.head {
						t.Errorf("%s, job %s: checks out %q, want %s", c.name, name, ref, c.head)
					}
				}
				if strings.Contains(run, "workline route merge-request") {
					routes++
					for _, want := range c.route[name] {
						if !strings.Contains(run, want) {
							t.Errorf("%s, job %s: the line runs without %s:\n%s", c.name, name, want, run)
						}
					}
					for _, not := range c.never[name] {
						if strings.Contains(run, not) {
							t.Errorf("%s, job %s: the line runs with %s:\n%s", c.name, name, not, run)
						}
					}
				}
				if c.event["github.event_name"] != "merge_group" {
					continue
				}
				// The queue's run: no agent, no token for it, nothing left for a
				// job that writes, no verdict kept for a head.
				if strings.Contains(run, "npm install") {
					t.Errorf("%s, job %s: step %q installs the agent", c.name, name, what)
				}
				if env, _ := yaml.Marshal(s.Env); strings.Contains(string(env), "secrets.") {
					t.Errorf("%s, job %s: step %q holds a secret: %s", c.name, name, what, env)
				}
				switch {
				case strings.HasPrefix(s.Uses, "actions/upload-artifact@") && fmt.Sprint(s.With["name"]) != "workline-gates":
					t.Errorf("%s, job %s: uploads %v, for a job that writes", c.name, name, s.With["name"])
				case strings.HasPrefix(s.Uses, "github/codeql-action/"):
					t.Errorf("%s, job %s: uploads to code scanning", c.name, name)
				}
				// The job's verdict is the line's, the queue's run's.
				if s.Name == "Verdict" && name == "judge" {
					ctx["steps.queue.outputs.status"] = "1"
					if got := expand(t, s.Run, ctx); !strings.Contains(got, "status=1\n") {
						t.Errorf("%s: judge's verdict is not the queue's run's:\n%s", c.name, got)
					}
				}
			}
			if routes != 1 {
				t.Errorf("%s, job %s: %d steps run the line, want one", c.name, name, routes)
			}
		}
	}
	// What a queue's run leaves is no pull request's: nothing applied,
	// kept or followed, nor commented as a fork's. One job sets, with the
	// App, the yes the base requires on the group's commit, as the head
	// queued holds it; and only after a queue's run.
	for _, c := range []struct {
		event string
		want  map[string][]string // a workflow's jobs that run
	}{
		{"merge_group", map[string][]string{"workline-apply.yml": {"queue"}}},
		{"pull_request", map[string][]string{"workline-apply.yml": {"apply", "follow", "verdict"}}},
	} {
		run := map[string]any{"github.event_name": "workflow_run", "github.repository": "owner/repo",
			"github.event.workflow_run.name": "workline", "github.event.workflow_run.event": c.event,
			"github.event.workflow_run.conclusion": "failure", "github.event.workflow_run.head_repository.full_name": "owner/repo",
			"vars.WORKLINE_APP_ID": "1"}
		for _, f := range []string{"../../ci/github/workline-apply.yml", "../../ci/github/workline-fork.yml"} {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			var w struct {
				Jobs map[string]job `yaml:"jobs"`
			}
			if err := yaml.Unmarshal(data, &w); err != nil {
				t.Fatal(err)
			}
			var ran []string
			for name, j := range w.Jobs {
				if j.If == "" || truthy(eval(t, j.If, run)) {
					ran = append(ran, name)
				}
			}
			slices.Sort(ran)
			if want := c.want[filepath.Base(f)]; !slices.Equal(ran, want) {
				t.Errorf("after a run on %s, %s runs %v, want %v", c.event, filepath.Base(f), ran, want)
			}
			q, ok := w.Jobs["queue"]
			if !ok || c.event != "merge_group" {
				continue
			}
			follows := 0
			for _, s := range q.Steps {
				if !strings.Contains(s.Run, "workline follow") {
					continue
				}
				follows++
				if !strings.Contains(s.Run, `--merge-group "$GROUP" --head "$HEAD_SHA"`) ||
					fmt.Sprint(s.Env["GROUP"]) != "${{ github.event.workflow_run.head_branch }}" || fmt.Sprint(s.Env["HEAD_SHA"]) != "${{ github.event.workflow_run.head_sha }}" {
					t.Errorf("%s, job queue: does not set the yes on the group's commit from the group's branch: %s %v", filepath.Base(f), s.Run, s.Env)
				}
				if env, _ := yaml.Marshal(s.Env); strings.Contains(string(env), "CLAUDE") {
					t.Errorf("%s, job queue: the agent's token in reach of the App's", filepath.Base(f))
				}
			}
			if follows != 1 {
				t.Errorf("%s, job queue: %d steps run workline follow, want one", filepath.Base(f), follows)
			}
		}
	}
}

func TestGitLabMergeTrainRunsTheGatesAlone(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	// rules is a job's own, else the one it extends.
	rules := func(name string) []any {
		for seen := 0; name != "" && seen < 10; seen++ {
			job, _ := top[name].(map[string]any)
			if r, ok := job["rules"].([]any); ok {
				return r
			}
			name, _ = job["extends"].(string)
		}
		return nil
	}
	// The project's variables all set; a merge request of the project.
	project := map[string]any{"$CI_PIPELINE_SOURCE": "merge_request_event", "$WORKLINE_GITLAB_TOKEN": "glpat",
		"$CLAUDE_CODE_OAUTH_TOKEN": "sk", "$CI_MERGE_REQUEST_SOURCE_PROJECT_ID": "1", "$CI_PROJECT_ID": "1",
		"$CI_COMMIT_BRANCH": "", "$CI_DEFAULT_BRANCH": "main"}
	every := []string{"workline:gates", "workline:judge", "workline:apply", "workline:warn", "workline:warn:apply", "workline:merge"}
	for _, c := range []struct {
		kind string // CI_MERGE_REQUEST_EVENT_TYPE
		jobs []string
	}{
		{"merge_train", []string{"workline:gates"}},
		{"merged_result", every}, // the merge request's own pipeline, on each push
		{"detached", every},
	} {
		ctx := map[string]any{"$CI_MERGE_REQUEST_EVENT_TYPE": c.kind}
		for k, v := range project {
			ctx[k] = v
		}
		var ran []string
		for name, v := range top {
			job, ok := v.(map[string]any)
			if !ok || strings.HasPrefix(name, ".") || job["script"] == nil {
				continue
			}
			for _, r := range rules(name) {
				m, _ := r.(map[string]any)
				if cond, _ := m["if"].(string); cond != "" && truthy(eval(t, cond, ctx)) {
					ran = append(ran, name)
					break
				}
			}
		}
		slices.Sort(ran)
		want := slices.Clone(c.jobs)
		slices.Sort(want)
		if !slices.Equal(ran, want) {
			t.Errorf("a %s pipeline runs %v, want %v", c.kind, ran, want)
		}
		// Every job a job needs runs beside it: GitLab refuses the
		// pipeline otherwise.
		for _, name := range ran {
			job, _ := top[name].(map[string]any)
			needs, _ := job["needs"].([]any)
			for _, n := range needs {
				if !slices.Contains(ran, fmt.Sprint(n)) {
					t.Errorf("a %s pipeline: %s needs %v, which does not run", c.kind, name, n)
				}
			}
		}
	}
	// The gates' line: on a train, against the target's tip, said a
	// queue's run; else against the merge request's diff base, as before.
	gates, _ := top["workline:gates"].(map[string]any)
	var script strings.Builder
	for _, l := range gates["script"].([]any) {
		if anchored, ok := l.([]any); ok { // an anchor's lines, flattened as GitLab does
			for _, a := range anchored {
				fmt.Fprintln(&script, a)
			}
			continue
		}
		fmt.Fprintln(&script, l)
	}
	dir := t.TempDir()
	fake := "#!/bin/sh\necho \"$*\" > \"$ROUTE_SEEN\"\n"
	if err := os.WriteFile(filepath.Join(dir, "workline"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		kind, target string
		want, not    []string
	}{
		{kind: "merge_train", target: "t1t1", want: []string{"--input queue=true", "--input range=t1t1..c0c0"}},
		{kind: "merge_train", want: []string{"--input queue=true", "--input range=d1d1..c0c0"}}, // no target's tip given: the diff base
		{kind: "merged_result", target: "t1t1", want: []string{"--input range=d1d1..c0c0"}, not: []string{"queue="}},
		{kind: "detached", want: []string{"--input range=d1d1..c0c0"}, not: []string{"queue="}},
	} {
		seen := filepath.Join(dir, "seen-"+c.kind+c.target)
		cmd := exec.Command("sh", "-ec", script.String())
		cmd.Dir = dir // where the job's files go
		cmd.Env = []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"), "ROUTE_SEEN=" + seen,
			"CI_MERGE_REQUEST_EVENT_TYPE=" + c.kind, "CI_MERGE_REQUEST_DIFF_BASE_SHA=d1d1", "CI_COMMIT_SHA=c0c0"}
		if c.target != "" {
			cmd.Env = append(cmd.Env, "CI_MERGE_REQUEST_TARGET_BRANCH_SHA="+c.target)
		}
		out, err := cmd.CombinedOutput()
		args, _ := os.ReadFile(seen)
		if err != nil {
			t.Errorf("workline:gates, a %s pipeline: %v\n%s", c.kind, err, out)
			continue
		}
		for _, w := range c.want {
			if !strings.Contains(string(args), w) {
				t.Errorf("workline:gates, a %s pipeline, target %q: the line runs without %s: %s", c.kind, c.target, w, args)
			}
		}
		for _, n := range c.not {
			if strings.Contains(string(args), n) {
				t.Errorf("workline:gates, a %s pipeline: the line runs with %s: %s", c.kind, n, args)
			}
		}
	}
}

// expand replaces each ${{ … }} of a workflow's text by its value, as
// GitHub does before running it.
func expand(t *testing.T, text string, ctx map[string]any) string {
	return regexp.MustCompile(`\$\{\{(.*?)\}\}`).ReplaceAllStringFunc(text, func(m string) string {
		v := eval(t, strings.TrimSpace(m[3:len(m)-2]), ctx)
		switch v := v.(type) {
		case nil:
			return ""
		case bool:
			return fmt.Sprint(v)
		}
		return fmt.Sprint(v)
	})
}

// eval reads the expressions the templates write — GitHub's `if:` and
// ${{ }}, GitLab's `rules: if:` —: names looked up in ctx (null when not
// there), quoted strings, ==, !=, !, &&, ||, parentheses, and the status
// functions, a run neither cancelled nor failed. && and || give one of
// their operands, as GitHub's do.
func eval(t *testing.T, expr string, ctx map[string]any) any {
	t.Helper()
	tokens := regexp.MustCompile(`\s*(\$?[A-Za-z_][\w.\-]*\(\)|\$?[A-Za-z_][\w.\-]*|\d+|'[^']*'|"[^"]*"|==|!=|&&|\|\||!|\(|\))`).FindAllStringSubmatch(expr, -1)
	var toks []string
	for _, m := range tokens {
		toks = append(toks, m[1])
	}
	if blank := regexp.MustCompile(`\s+`); blank.ReplaceAllString(strings.Join(toks, ""), "") != blank.ReplaceAllString(expr, "") {
		t.Fatalf("expression %q: not read whole (%q)", expr, toks)
	}
	pos := 0
	var or func() any
	primary := func() any {
		if pos >= len(toks) {
			t.Fatalf("expression %q: ends early", expr)
		}
		tok := toks[pos]
		pos++
		switch {
		case tok == "(":
			v := or()
			if pos >= len(toks) || toks[pos] != ")" {
				t.Fatalf("expression %q: no closing parenthesis", expr)
			}
			pos++
			return v
		case strings.HasPrefix(tok, "'") || strings.HasPrefix(tok, `"`):
			return tok[1 : len(tok)-1]
		case tok == "true" || tok == "false":
			return tok == "true"
		case tok == "always()" || tok == "success()":
			return true
		case tok == "cancelled()" || tok == "failure()":
			return false
		case strings.HasSuffix(tok, "()"):
			t.Fatalf("expression %q: function %s not read here", expr, tok)
		}
		return ctx[tok]
	}
	var unary func() any
	unary = func() any {
		if pos < len(toks) && toks[pos] == "!" {
			pos++
			return !truthy(unary())
		}
		return primary()
	}
	cmp := func() any {
		a := unary()
		for pos < len(toks) && (toks[pos] == "==" || toks[pos] == "!=") {
			op := toks[pos]
			pos++
			b := unary()
			eq := a != nil && b != nil && fmt.Sprint(a) == fmt.Sprint(b) || a == nil && b == nil
			a = eq == (op == "==")
		}
		return a
	}
	and := func() any {
		a := cmp()
		for pos < len(toks) && toks[pos] == "&&" {
			pos++
			b := cmp()
			if truthy(a) {
				a = b
			}
		}
		return a
	}
	or = func() any {
		a := and()
		for pos < len(toks) && toks[pos] == "||" {
			pos++
			b := and()
			if !truthy(a) {
				a = b
			}
		}
		return a
	}
	v := or()
	if pos != len(toks) {
		t.Fatalf("expression %q: %q left unread", expr, toks[pos:])
	}
	return v
}

// truthy: neither null, false nor an empty string.
func truthy(v any) bool {
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	}
	return true
}
