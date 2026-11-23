package github

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// ownAnswerWorkflow is the project's own answer workflow.
const ownAnswerWorkflow = "../../../.github/workflows/workline-answer.yml"

// runAnswerLookupSteps runs, as GitHub runs them (bash -e, pipefail), the
// three steps of the own answer workflow that look the pull request up
// from the branch name: the answer (propose job), the apply and the
// answer's state (apply job). The head step is not run: each step gets
// the head, the run and the state a head step that found the Draft would
// have passed on. `gh pr list` lists prs (a JSON list) through the step's
// own --jq; `workline` and `git` only record they were called. It returns
// the calls of workline, a line each.
func runAnswerLookupSteps(t *testing.T, prs string) string {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	data, err := os.ReadFile(ownAnswerWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	type step struct {
		ID   string `yaml:"id"`
		Name string `yaml:"name"`
		Run  string `yaml:"run"`
	}
	var w struct {
		Jobs map[string]struct {
			Steps []step `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("%s: %v", ownAnswerWorkflow, err)
	}
	var answer, apply, state string
	for _, s := range w.Jobs["propose"].Steps {
		if s.ID == "answer" {
			answer = s.Run
		}
	}
	for _, s := range w.Jobs["apply"].Steps {
		switch {
		case strings.Contains(s.Run, "workline apply"):
			apply = s.Run
		case s.Name == "The answer's state":
			state = s.Run
		}
	}
	if answer == "" || apply == "" || state == "" {
		t.Fatalf("%s: a step is missing (answer %t, apply %t, state %t)", ownAnswerWorkflow, answer != "", apply != "", state != "")
	}

	dir := t.TempDir()
	bin, work := filepath.Join(dir, "bin"), filepath.Join(dir, "work")
	const run = "20260101T000000.000000001-tester"
	runs := filepath.Join(dir, "runs")
	for _, d := range []string{bin, work, filepath.Join(runs, run)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	calls, list := filepath.Join(dir, "calls"), filepath.Join(dir, "prs.json")
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
	for _, script := range []string{answer, apply, state} {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
		cmd.Dir = work
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "CALLS="+calls, "PRS="+list,
			"GITHUB_OUTPUT="+filepath.Join(dir, "output"), "GITHUB_STEP_SUMMARY="+filepath.Join(dir, "summary"),
			"GITHUB_REF_NAME=main", "GH_TOKEN=t", "GH_REPO=o/r", "GITHUB_REPOSITORY=o/r",
			"WORKLINE_RUNS_DIR="+runs, "BRANCH=workline/tester/x",
			"HEAD_SHA=dddddddddddddddddddddddddddddddddddddddd", "RUN="+run, "STATE=s1", "APP=")
		if said, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("a step failed: %v\n%s", err, said)
		}
	}
	got, _ := os.ReadFile(calls)
	return string(got)
}

// Fork PR ignored
// Given an open fork pull request from a branch named workline/tester/x
// and the repository's own pull request closed,
// When the answer step runs,
// Then it finds no pull request and posts nothing: the answer, the apply
// and the state mark call workline not at all.
func TestAnswerStepsIgnoreAForkPullRequest(t *testing.T) {
	calls := runAnswerLookupSteps(t, `[{"number":11,"headRefOid":"ffffffffffffffffffffffffffffffffffffffff","isCrossRepository":true}]`)
	if calls != "" {
		t.Errorf("workline was called on a fork's pull request:\n%s", calls)
	}
}

// Own PR chosen
// Given a fork pull request listed before the repository's own,
// When the answer step runs,
// Then it answers on the repository's own pull request: the answer, the
// apply and the state mark all name pull request 7.
func TestAnswerStepsChooseTheOwnPullRequest(t *testing.T) {
	calls := runAnswerLookupSteps(t, `[{"number":11,"headRefOid":"ffffffffffffffffffffffffffffffffffffffff","isCrossRepository":true},`+
		`{"number":7,"headRefOid":"dddddddddddddddddddddddddddddddddddddddd","isCrossRepository":false}]`)
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
				t.Errorf("%s is not on the repository's own pull request 7: %q", c.what, l)
			}
		}
		if !found {
			t.Errorf("%s never ran; workline was called:\n%s", c.what, calls)
		}
	}
}
