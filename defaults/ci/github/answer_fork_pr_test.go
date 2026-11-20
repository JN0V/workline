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

var (
	forkFlowExpr   = regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)
	forkFlowOutput = regexp.MustCompile(`^(?:steps|needs)\.[\w-]+\.outputs\.([\w-]+)$`)
	forkFlowNumber = regexp.MustCompile(`--(?:from-)?merge-request (\d+)`)
)

// runAnswerAgainstPRs runs the answer workflow's steps as GitHub runs them
// (bash -e, step by step, each step's env and outputs passed on), with a
// `gh` that lists prs (a JSON list) for any `gh pr list`, through the
// step's own --jq, and a `workline` and a `git` that only record they were
// called. It returns the calls of workline, a line each.
func runAnswerAgainstPRs(t *testing.T, file, prs, branch string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	type step struct {
		ID   string         `yaml:"id"`
		Name string         `yaml:"name"`
		Run  string         `yaml:"run"`
		Env  map[string]any `yaml:"env"`
	}
	var w struct {
		Jobs map[string]struct {
			Steps []step `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	dir := t.TempDir()
	bin, work, temp := filepath.Join(dir, "bin"), filepath.Join(dir, "work"), filepath.Join(dir, "temp")
	runDir := filepath.Join(temp, "workline-runs", ".workline-runs", "20260101T000000.000000001-tester")
	for _, d := range []string{bin, work, temp, runDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	calls, out, list := filepath.Join(dir, "calls"), filepath.Join(dir, "output"), filepath.Join(dir, "prs.json")
	stubs := map[string]string{
		"gh": `#!/bin/bash
if [ "$1" = pr ] && [ "$2" = list ]; then
  expr=.
  args=("$@")
  for ((i = 0; i < ${#args[@]}; i++)); do
    case "${args[i]}" in --jq|-q) expr="${args[i+1]}" ;; esac
  done
  jq -r "$expr" "$PRS"
fi
exit 0
`,
		"git": "#!/bin/bash\nexit 0\n",
		"workline": `#!/bin/bash
echo "workline $*" >> "$CALLS"
case " $* " in
  *" --state "*) echo "due s1" ;;
  *" --json "*) echo '{"pending":["/w/.workline-runs/20260101T000000.000000001-tester"]}' ;;
esac
exit 0
`,
	}
	for name, body := range stubs {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(list, []byte(prs), 0o644); err != nil {
		t.Fatal(err)
	}
	outputs := map[string]string{}
	resolve := func(v string) string {
		return forkFlowExpr.ReplaceAllStringFunc(v, func(m string) string {
			expr := forkFlowExpr.FindStringSubmatch(m)[1]
			switch {
			case forkFlowOutput.MatchString(expr):
				return outputs[forkFlowOutput.FindStringSubmatch(expr)[1]]
			case strings.Contains(expr, "inputs.branch"):
				return branch
			case expr == "runner.temp":
				return temp
			case expr == "github.workspace":
				return work
			}
			return ""
		})
	}
	do := func(job string, want func(step) bool) {
		for _, s := range w.Jobs[job].Steps {
			if !want(s) {
				continue
			}
			cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", s.Run)
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "CALLS="+calls, "PRS="+list,
				"GITHUB_OUTPUT="+out, "GITHUB_STEP_SUMMARY="+filepath.Join(dir, "summary"),
				"GITHUB_REF_NAME=main", "RUNNER_TEMP="+temp, "GITHUB_WORKSPACE="+work,
				"WORKLINE_RUNS_DIR="+filepath.Join(work, ".workline-runs"),
				"GH_TOKEN=t", "GH_REPO=o/r", "GITHUB_REPOSITORY=o/r")
			for k, v := range s.Env {
				cmd.Env = append(cmd.Env, k+"="+resolve(fmt.Sprint(v)))
			}
			if said, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s: the step %q failed: %v\n%s", file, s.Name, err, said)
			}
			got, _ := os.ReadFile(out)
			for _, l := range strings.Split(string(got), "\n") {
				if k, v, ok := strings.Cut(l, "="); ok {
					outputs[k] = v
				}
			}
		}
	}
	do("propose", func(s step) bool { return s.ID == "head" })
	if outputs["head"] != "" {
		do("propose", func(s step) bool { return s.ID == "answer" })
	}
	if outputs["state"] != "" {
		if outputs["run"] != "" {
			do("apply", func(s step) bool { return strings.Contains(s.Run, "workline apply") })
		}
		if outputs["status"] != "" && outputs["status"] != "3" {
			do("apply", func(s step) bool { return s.Name == "The answer's state" })
		}
	}
	got, _ := os.ReadFile(calls)
	return string(got)
}

// Same-named fork branch
// Given a Draft pull request in this repository (7) and a fork pull
// request (11, listed first) on the same branch name,
// When the answer workflow runs,
// Then the answer is read and applied on the Draft's pull request only:
// the answer, the apply and the state it marks all name pull request 7.
// It reads both copies of the workflow, the template and the project's own.
func TestAnswerWorkflowKeepsToTheDraftNotAForkPullRequest(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	prs := `[{"number":11,"headRefOid":"ffffffffffffffffffffffffffffffffffffffff","isCrossRepository":true},` +
		`{"number":7,"headRefOid":"dddddddddddddddddddddddddddddddddddddddd","isCrossRepository":false}]`
	for _, file := range answerWorkflows {
		t.Run(file, func(t *testing.T) {
			calls := runAnswerAgainstPRs(t, file, prs, "workline/tester/issue-9")
			for _, c := range []struct{ what, marker string }{
				{"the answer", "--no-apply"},
				{"the apply", " apply "},
				{"the state mark", "--tried"},
			} {
				found := false
				for _, l := range strings.Split(calls, "\n") {
					if !strings.HasPrefix(l, "workline ") || !strings.Contains(l, c.marker) {
						continue
					}
					found = true
					if m := forkFlowNumber.FindStringSubmatch(l); m == nil || m[1] != "7" {
						t.Errorf("%s: %s is not on the Draft's pull request 7: %q", file, c.what, l)
					}
				}
				if !found {
					t.Errorf("%s: %s never ran; workline was called:\n%s", file, c.what, calls)
				}
			}
		})
	}
}
