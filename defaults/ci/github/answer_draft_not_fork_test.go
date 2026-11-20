package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// answerWorkflows are the two copies of the answer workflow: the
// template the project ships and the project's own workflow.
var answerWorkflows = []string{
	"workline-answer.yml",
	"../../../.github/workflows/workline-answer.yml",
}

// runAnswerJobs plays, with bash as the CI does, the steps of the answer
// workflow that look a pull request up (the propose job's, then the apply
// job's) on the branch workline/tester/1, for a forge whose `gh pr list`
// returns the JSON list `prs`. Each step's inputs are the outputs of the
// steps before it, as the workflow wires them. gh, git and workline are
// stubs: gh answers from `prs` through jq (as its --jq does), workline
// says a state is due and a run is pending; git and workline record
// their calls. It returns the calls recorded, one a line.
func runAnswerJobs(t *testing.T, file, prs string) string {
	t.Helper()
	sh, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	type step struct {
		ID  string         `yaml:"id"`
		Run string         `yaml:"run"`
		Env map[string]any `yaml:"env"`
	}
	var w struct {
		Env  map[string]any `yaml:"env"`
		Jobs map[string]struct {
			Outputs map[string]string `yaml:"outputs"`
			Steps   []step            `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("%s: %v", file, err)
	}

	const branch = "workline/tester/1"
	const run = "20261120T000000.000000001-tester"
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	ws := filepath.Join(dir, "ws")
	tmp := filepath.Join(dir, "tmp")
	for _, d := range []string{bin, ws, filepath.Join(tmp, "workline-runs", ".workline-runs", run)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	calls := filepath.Join(dir, "calls")
	list := filepath.Join(dir, "prs.json")
	if err := os.WriteFile(list, []byte(prs), 0o644); err != nil {
		t.Fatal(err)
	}
	stubs := map[string]string{
		"gh": "[ \"$1 $2\" = \"pr list\" ] || { echo \"gh: unexpected call: $*\" >&2; exit 1; }\n" +
			"jq=\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = --jq ]; then jq=$2; fi\n  shift\ndone\n" +
			"if [ -n \"$jq\" ]; then exec jq -r \"$jq\" \"$PRS\"; fi\ncat \"$PRS\"\n",
		"git": "echo \"git $*\" >> \"$CALLS\"\nexit 0\n",
		"workline": "echo \"workline $*\" >> \"$CALLS\"\ncase \"$*\" in\n" +
			"*\" --state\"*) echo 'due S1';;\n" +
			"*--no-apply*) echo '{\"pending\":[\"/w/.workline-runs/" + run + "\"]}';;\nesac\nexit 0\n",
	}
	for name, body := range stubs {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	vals := map[string]string{
		"github.workspace": ws, "runner.temp": tmp, "github.token": "tok",
		"inputs.branch": branch,
	}
	expr := regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)
	resolve := func(s string) string {
		return expr.ReplaceAllStringFunc(s, func(m string) string {
			for _, alt := range strings.Split(expr.FindStringSubmatch(m)[1], "||") {
				alt = strings.TrimSpace(alt)
				v := vals[alt]
				if strings.HasPrefix(alt, "'") {
					v = strings.Trim(alt, "'")
				}
				if v != "" {
					return v
				}
			}
			return ""
		})
	}
	n := 0
	play := func(s step) {
		n++
		outFile := filepath.Join(dir, fmt.Sprintf("output-%d", n))
		script := filepath.Join(dir, fmt.Sprintf("step-%d.sh", n))
		if err := os.WriteFile(script, []byte(s.Run), 0o644); err != nil {
			t.Fatal(err)
		}
		env := []string{
			"PATH=" + bin + ":" + os.Getenv("PATH"),
			"HOME=" + dir, "GIT_CONFIG_GLOBAL=" + os.DevNull,
			"CALLS=" + calls, "PRS=" + list,
			"GITHUB_OUTPUT=" + outFile,
			"GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary"),
			"RUNNER_TEMP=" + tmp,
		}
		for _, e := range []map[string]any{w.Env, s.Env} {
			for k, v := range e {
				env = append(env, k+"="+resolve(fmt.Sprint(v)))
			}
		}
		cmd := exec.Command(sh, "--noprofile", "--norc", "-eo", "pipefail", script)
		cmd.Dir = ws
		cmd.Env = env
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("%s: the step %q ended with an error: %v\n%s", file, s.ID, runErr, out)
		}
		made, _ := os.ReadFile(outFile)
		for _, line := range strings.Split(string(made), "\n") {
			if k, v, ok := strings.Cut(line, "="); ok && s.ID != "" {
				vals["steps."+s.ID+".outputs."+k] = v
			}
		}
	}
	// The steps that look a pull request up or call the engine on one.
	looksUp := func(s step) bool {
		return strings.Contains(s.Run, "gh pr list") || strings.Contains(s.Run, "workline test answer") ||
			strings.Contains(s.Run, "workline apply")
	}

	for _, s := range w.Jobs["propose"].Steps {
		if looksUp(s) {
			play(s)
		}
	}
	for k, v := range w.Jobs["propose"].Outputs {
		vals["needs.propose.outputs."+k] = resolve(v)
	}
	for _, s := range w.Jobs["apply"].Steps {
		if looksUp(s) {
			play(s)
		}
	}
	made, _ := os.ReadFile(calls)
	return string(made)
}

// Same-named fork branch
// Given a Draft pull request in this repository (number 7) and a fork
// pull request (number 5) on the same branch name, listed first,
// When the answer workflow runs,
// Then the answer is read and applied on the Draft's pull request only:
// the state read, the answer, the apply and the state marked all name
// pull request 7, none the fork's 5. It reads both copies of the
// workflow, the template and the project's own.
// The scenarios No fork (the Draft alone: answered on it) and Only a
// fork pull request (nothing answered, exit 0) describe what the head
// step already does: they have no test of their own, as one that passes
// on the code as it is proves nothing.
func TestAnswerAppliesOnTheDraftNotAForkPullRequest(t *testing.T) {
	prs := `[{"number":5,"headRefOid":"forkfork","isCrossRepository":true},` +
		`{"number":7,"headRefOid":"draft777","isCrossRepository":false}]`
	target := regexp.MustCompile(`--(?:from-)?merge-request (\d+)`)
	for _, file := range answerWorkflows {
		t.Run(file, func(t *testing.T) {
			calls := runAnswerJobs(t, file, prs)
			seen := map[string]bool{}
			for _, line := range strings.Split(calls, "\n") {
				m := target.FindStringSubmatch(line)
				if m == nil {
					continue
				}
				if m[1] != "7" {
					t.Errorf("%s: %q reads or applies pull request %s, want the Draft's 7", file, line, m[1])
				}
				switch {
				case strings.Contains(line, "--state"):
					seen["state read"] = true
				case strings.Contains(line, "--no-apply"):
					seen["answer"] = true
				case strings.Contains(line, "workline apply"):
					seen["apply"] = true
				case strings.Contains(line, "--tried"):
					seen["state marked"] = true
				}
			}
			for _, want := range []string{"state read", "answer", "apply", "state marked"} {
				if !seen[want] {
					t.Errorf("%s: no %s call on the Draft's pull request 7; calls:\n%s", file, want, calls)
				}
			}
		})
	}
}
