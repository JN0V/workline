package conformance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// answerRefusalStep is a step of workline-answer.yml, as these tests read it.
type answerRefusalStep struct {
	Name string         `yaml:"name"`
	If   string         `yaml:"if"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
	Env  map[string]any `yaml:"env"`
}

// answerRefusalJobs reads the steps of each job of an answer workflow.
func answerRefusalJobs(t *testing.T, f string) map[string][]answerRefusalStep {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []answerRefusalStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	jobs := map[string][]answerRefusalStep{}
	for name, j := range wf.Jobs {
		jobs[name] = j.Steps
	}
	return jobs
}

// answerRefusalRun runs a step's shell as GitHub does (bash -e), in dir,
// with fake gh and workline on PATH and git a no-op; it returns the step's
// outputs and its log.
func answerRefusalRun(t *testing.T, dir, script, workline string, env ...string) (string, string) {
	t.Helper()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	gh := "#!/bin/sh\ncase \"$1 $2\" in\n  \"pr list\") echo 7 ;;\n  *) echo \"unexpected: $*\" >&2; exit 9 ;;\nesac\n"
	for name, body := range map[string]string{"gh": gh, "workline": workline} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "output")
	_ = os.Remove(out)
	run := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", "git() { :; }\n"+script)
	run.Dir = dir
	run.Env = append(os.Environ(), append([]string{"PATH=" + bin + ":" + os.Getenv("PATH"),
		"BRANCH=workline/tester/issue-9", "HEAD_SHA=abc123", "GITHUB_OUTPUT=" + out,
		"GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary"), "WORKLINE_CALLS=" + filepath.Join(dir, "calls")}, env...)...)
	said, _ := run.CombinedOutput()
	got, _ := os.ReadFile(out)
	calls, _ := os.ReadFile(filepath.Join(dir, "calls"))
	return string(got), string(said) + string(calls)
}

// A refused answer said, in the template and in workline's own workflow
// Given a tester's answer the engine refuses (exit 1, no pending run, its
// run folder holding out/on-block.yaml),
// When the judge and apply jobs of workline-answer.yml run,
// Then the run is named for apply, the artifact is kept with answer.json,
// the Verdict step prints the reason as an error, and apply finds the run
// where two upload paths put it, then applies it: the comment in the thread.
func TestAnswerRefusalSaidInJobAndThread(t *testing.T) {
	const run = "20261118T083850.123456789-tester"
	// The engine refusing: the run folder with its comment, the JSON on stdout.
	refuse := fmt.Sprintf("#!/bin/sh\necho \"workline $*\" >> \"$WORKLINE_CALLS\"\n"+
		"case \"$1\" in apply) exit 0 ;; esac\n"+
		"mkdir -p \"$WORKLINE_RUNS_DIR/%[1]s/out\" && echo '- reply: refused' > \"$WORKLINE_RUNS_DIR/%[1]s/out/on-block.yaml\"\n"+
		"echo '{\"status\": \"block\", \"summary\": \"the answer is refused\", \"run-dir\": \"'\"$WORKLINE_RUNS_DIR/%[1]s\"'\", "+
		"\"findings\": [{\"rule\": \"rewrite-other-file\", \"message\": \"tests/test_add.sh exists already\"}]}'\n"+
		"echo \"the tester's answer is refused\" >&2\nexit 1\n", run)
	for _, dir := range answerCIDirs {
		f := filepath.Join(dir, "workline-answer.yml")
		jobs := answerRefusalJobs(t, f)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			var answer, upload, verdict, apply *answerRefusalStep
			for i := range jobs["judge"] {
				s := &jobs["judge"][i]
				switch {
				case s.Name == "The answer (proposed only)":
					answer = s
				case strings.Contains(s.Uses, "upload-artifact") && fmt.Sprint(s.With["name"]) == "workline-answer":
					upload = s
				case s.Name == "Verdict":
					verdict = s
				}
			}
			for i := range jobs["apply"] {
				if s := &jobs["apply"][i]; strings.Contains(s.Run, "workline apply") {
					apply = s
				}
			}
			if answer == nil || upload == nil || verdict == nil || apply == nil {
				t.Fatalf("%s: a step missing: answer %v, upload %v, Verdict %v, apply %v", f, answer != nil, upload != nil, verdict != nil, apply != nil)
			}
			work := t.TempDir()
			out, log := answerRefusalRun(t, work, answer.Run, refuse, "WORKLINE_RUNS_DIR="+filepath.Join(work, ".workline-runs"))
			if !strings.Contains(out, "status=1") || !strings.Contains(out, "run="+run) {
				t.Errorf("%s: a refused answer's run is not named for apply\noutputs:\n%s\nlog:\n%s", f, out, log)
			}
			if c := upload.If; !strings.Contains(c, "always()") || strings.Contains(c, "outputs.run") {
				t.Errorf("%s: the answer artifact is uploaded `if: %s`: a refused answer may not be kept", f, c)
			}
			if p := fmt.Sprint(upload.With["path"]); !strings.Contains(p, ".workline-runs") || !strings.Contains(p, "answer.json") {
				t.Errorf("%s: the answer artifact's path %q does not keep the runs and answer.json", f, p)
			}
			_, log = answerRefusalRun(t, work, verdict.Run, refuse, "STATUS=1")
			if !strings.Contains(log, "::error title=workline::the answer is refused") || !strings.Contains(log, "rewrite-other-file: tests/test_add.sh exists already") {
				t.Errorf("%s: the Verdict step does not say the reason as an error\n%s", f, log)
			}
			// Two upload paths root the artifact at the workspace: the
			// download holds .workline-runs/<run>.
			runsDir := fmt.Sprint(apply.Env["WORKLINE_RUNS_DIR"])
			if !strings.HasSuffix(runsDir, "/.workline-runs") {
				t.Fatalf("%s: apply's WORKLINE_RUNS_DIR %q is not the artifact's .workline-runs", f, runsDir)
			}
			_, log = answerRefusalRun(t, work, apply.Run, refuse, "WORKLINE_RUNS_DIR="+filepath.Join(work, ".workline-runs"), "RUN="+run, "APP=")
			if !strings.Contains(log, "workline apply "+run+" --from-merge-request 7") {
				t.Errorf("%s: apply does not apply the refused run\n%s", f, log)
			}
		})
	}
}
