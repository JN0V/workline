package conformance

import (
	enc "encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// roomStepOf is the shell of the apply job's `room` step in file.
func roomStepOf(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				ID  string `yaml:"id"`
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	for _, s := range wf.Jobs["apply"].Steps {
		if s.ID == "room" {
			return s.Run
		}
	}
	t.Fatalf("%s: apply has no step `room`", file)
	return ""
}

// runRoomStep runs the room step's shell as GitHub runs it, the line and
// `gh` stand-ins: the line lists due as the issues due, the forge lists
// runs, each a {displayTitle, status}, and what `gh` was asked is
// returned. The apply job of issue ends `times` times in a row.
func runRoomStep(t *testing.T, script, issue string, due []int, runs []map[string]string, times int) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, r := range runs {
		r["name"] = r["displayTitle"]
		r["event"] = "workflow_dispatch"
	}
	store, _ := enc.Marshal(runs)
	if err := os.WriteFile(filepath.Join(dir, "runs.json"), store, 0o644); err != nil {
		t.Fatal(err)
	}
	said, _ := enc.Marshal(map[string]any{"due": due, "summary": fmt.Sprintf("reconcile: %d issues due", len(due))})
	writeExec(t, filepath.Join(bin, "workline"), "#!/bin/bash\ncat <<'EOF'\n"+string(said)+"\nEOF\n")
	calls := filepath.Join(dir, "calls")
	// gh run list: the stored runs, those of --status when given, through --jq;
	// any other call is only logged.
	writeExec(t, filepath.Join(bin, "gh"), `#!/bin/bash
echo "$*" >> "`+calls+`"
if [ "$1" = run ] && [ "$2" = list ]; then
  status=""; expr=""
  args=("$@")
  for ((i = 0; i < ${#args[@]}; i++)); do
    case "${args[i]}" in
      --status) status="${args[i+1]}" ;;
      --jq|-q) expr="${args[i+1]}" ;;
    esac
  done
  runs=$(jq -c --arg s "$status" '[.[] | select($s == "" or .status == $s)]' "`+filepath.Join(dir, "runs.json")+`")
  if [ -n "$expr" ]; then printf '%s' "$runs" | jq -r "$expr"; else printf '%s\n' "$runs"; fi
  exit 0
fi
exit 0
`)
	for range times {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+filepath.Join(dir, "summary"),
			"GH_TOKEN=t", "ISSUE="+issue, "GITHUB_REPOSITORY=o/r", "GITHUB_REF_NAME=main",
			"GITHUB_WORKFLOW_REF=o/r/.github/workflows/workline-develop.yml@refs/heads/main")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("the room step failed: %v\n%s", err, out)
		}
	}
	got, _ := os.ReadFile(calls)
	return string(got)
}

// startedFor says whether the calls started a run of issue n.
func startedFor(calls string, n int) bool {
	return regexp.MustCompile(`(?m)^workflow run .*-f issue=` + fmt.Sprint(n) + `$`).MatchString(strings.TrimSpace(calls)) ||
		strings.Contains(calls, fmt.Sprintf("-f issue=%d\n", n))
}

// Given issue A (5) has a develop run in progress, When another develop
// run's apply job ends (the issue 7's) and the line lists A as due, Then
// no new develop run starts for A.
func TestRoomStepDoesNotDispatchAnIssueRunning(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	for _, f := range []string{"../../defaults/ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		script := roomStepOf(t, f)
		runs := []map[string]string{
			{"displayTitle": "workline develop #5", "status": "in_progress"},
			{"displayTitle": "workline develop #7", "status": "in_progress"},
			{"displayTitle": "workline develop #9", "status": "completed"},
		}
		calls := runRoomStep(t, script, "7", []int{5}, runs, 1)
		if startedFor(calls, 5) {
			t.Errorf("%s: a run of #5 started though one of it is in progress: %q", f, calls)
		}
	}
}

// Given issue A (5) has a develop run queued, When three apply jobs end
// within a minute and the line lists A as due each time, Then A still has
// one develop run queued: none of the three starts another.
func TestRoomStepDoesNotDispatchAnIssueQueued(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	for _, f := range []string{"../../defaults/ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		script := roomStepOf(t, f)
		runs := []map[string]string{
			{"displayTitle": "workline develop #5", "status": "queued"},
			{"displayTitle": "workline develop #9", "status": "completed"},
		}
		calls := runRoomStep(t, script, "7", []int{5}, runs, 3)
		if startedFor(calls, 5) {
			t.Errorf("%s: a run of #5 started though one of it is queued: %q", f, calls)
		}
	}
}
