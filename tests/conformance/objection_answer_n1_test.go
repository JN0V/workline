package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// answerN1Step is the run of workline-answer.yml's step name in job.
func answerN1Step(t *testing.T, f, job, name string) string {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	for _, s := range wf.Jobs[job].Steps {
		if s.Name == name {
			return s.Run
		}
	}
	t.Fatalf("%s: job %s has no step %q", f, job, name)
	return ""
}

// answerN1Run runs a step's shell as GitHub does (bash -e), with fake gh
// and workline on PATH; it returns the step's outputs and its log.
func answerN1Run(t *testing.T, script, workline string, env ...string) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	gh := "#!/bin/sh\ncase \"$1 $2\" in\n  \"pr list\") case \"$*\" in *headRefOid*) echo \"7 abc123\" ;; *) echo 7 ;; esac ;;\n  *) echo \"unexpected: $*\" >&2; exit 9 ;;\nesac\n"
	for name, body := range map[string]string{"gh": gh, "workline": workline} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(tmp, "output")
	run := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
	run.Dir = tmp
	run.Env = append(os.Environ(), append([]string{"PATH=" + bin + ":" + os.Getenv("PATH"),
		"BRANCH=workline/tester/issue-9", "GITHUB_OUTPUT=" + out, "GITHUB_STEP_SUMMARY=" + filepath.Join(tmp, "summary"),
		"WORKLINE_CALLS=" + filepath.Join(tmp, "calls")}, env...)...)
	said, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("the step failed: %v\n%s", err, said)
	}
	got, _ := os.ReadFile(out)
	calls, _ := os.ReadFile(filepath.Join(tmp, "calls"))
	return string(got), string(said) + string(calls)
}

// The pinned release reads the answer workflow (ADR-0071's N-1 rule)
// Given workline-answer.yml run by a release that knows neither
// `test answer --state` nor `--tried` (v0.78.4: "flag provided but not
// defined", exit 64),
// When an objection wakes it,
// Then the answer is due, as before the state, said in the log, and apply
// writes no state; and an engine that reads the state still decides.
func TestObjectionAnswerStateUnreadIsDue(t *testing.T) {
	// v0.78.4's answer to an unknown flag.
	old := "#!/bin/sh\necho \"workline $*\" >> \"$WORKLINE_CALLS\"\nfor a; do case \"$a\" in --state|--tried) echo \"workline test: flag provided but not defined: ${a#-}\" >&2; exit 64 ;; esac; done\n"
	said := func(line string) string {
		return "#!/bin/sh\necho \"workline $*\" >> \"$WORKLINE_CALLS\"\necho '" + line + "'\n"
	}
	for _, dir := range answerCIDirs {
		f := filepath.Join(dir, "workline-answer.yml")
		head := answerN1Step(t, f, "judge", "The Draft's head")
		state := answerN1Step(t, f, "apply", "The answer's state")
		t.Run(filepath.Base(dir), func(t *testing.T) {
			out, log := answerN1Run(t, head, old)
			if !strings.Contains(out, "head=abc123") || !strings.Contains(out, "state=unread") {
				t.Errorf("an engine without --state: the answer is not due\noutputs:\n%s\nlog:\n%s", out, log)
			}
			if !strings.Contains(log, "state not read") {
				t.Errorf("an engine without --state: the log does not say the state was not read\n%s", log)
			}
			_, log = answerN1Run(t, state, old, "STATE=unread")
			if strings.Contains(log, "--tried") {
				t.Errorf("a state not read is marked tried\n%s", log)
			}
			out, log = answerN1Run(t, head, said("due at=abc123 threads=5.1"))
			if !strings.Contains(out, "state=at=abc123 threads=5.1") {
				t.Errorf("an engine with --state: its state not passed on\n%s\n%s", out, log)
			}
			if out, log = answerN1Run(t, head, said("tried at=abc123 threads=5.1")); strings.Contains(out, "head=") {
				t.Errorf("a state already tried is answered again\n%s\n%s", out, log)
			}
			if _, log = answerN1Run(t, state, said("tried x"), "STATE=at=abc123 threads=5.1"); !strings.Contains(log, "--tried at=abc123 threads=5.1") {
				t.Errorf("a state read is not marked tried\n%s", log)
			}
		})
	}
}
