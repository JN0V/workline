package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The tester's objection queue, in the templates and in workline's own
// workflows: the tests read workline-answer.yml, workline-objection.yml and
// workline-develop.yml, where the answer is woken.

var answerCIDirs = []string{"../../defaults/ci/github", "../../.github/workflows"}

// answerCIWorkflow is what these tests read of a GitHub workflow.
type answerCIWorkflow struct {
	On   map[string]any `yaml:"on"`
	Jobs map[string]struct {
		If    string `yaml:"if"`
		Steps []struct {
			Run string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func answerCIRead(t *testing.T, f string) answerCIWorkflow {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var wf answerCIWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	return wf
}

// answerCIHeartbeatReaches says whether the develop workflow's reconciler,
// the heartbeat's run, starts the tester's answer: it runs `workline test
// answer`, or starts workline-answer.yml, which then takes a manual start.
func answerCIHeartbeatReaches(t *testing.T, dir string) (bool, string) {
	t.Helper()
	develop := answerCIRead(t, filepath.Join(dir, "workline-develop.yml"))
	starts := false
	for _, j := range develop.Jobs {
		for _, s := range j.Steps {
			if strings.Contains(s.Run, "test answer") {
				return true, ""
			}
			if strings.Contains(s.Run, "workline-answer") {
				starts = true
			}
		}
	}
	if !starts {
		return false, "workline-develop.yml never runs `workline test answer` nor starts workline-answer.yml"
	}
	if _, ok := answerCIRead(t, filepath.Join(dir, "workline-answer.yml")).On["workflow_dispatch"]; !ok {
		return false, "workline-develop.yml starts workline-answer.yml, which takes no manual start (workflow_dispatch)"
	}
	return true, ""
}

// Queue drained
// Given a Draft with three unanswered objections,
// When one answer run ends,
// Then all three are answered, or the next heartbeat answers the rest with
// no person acting: the end of an answer starts the next one, or the
// heartbeat's reconciler starts the answer.
func TestObjectionQueueDrained(t *testing.T) {
	for _, dir := range answerCIDirs {
		answer := answerCIRead(t, filepath.Join(dir, "workline-answer.yml"))
		chained := false
		if run, ok := answer.On["workflow_run"].(map[string]any); ok {
			names, _ := run["workflows"].([]any)
			chained = slices.Contains(names, any("workline answer"))
		}
		restarts := false
		for _, j := range answer.Jobs {
			for _, s := range j.Steps {
				if strings.Contains(s.Run, "gh workflow run") {
					restarts = true
				}
			}
		}
		reaches, why := answerCIHeartbeatReaches(t, dir)
		if !chained && !restarts && !reaches {
			t.Errorf("%s: one answer run leaves the other objections waiting: the end of workline answer starts no next answer, and the heartbeat does not: %s", dir, why)
		}
	}
}

// Heartbeat resumes
// Given an unanswered objection and no new comment for an hour,
// When the heartbeat runs,
// Then the tester answers it: the reconciler of workline-develop.yml runs
// `workline test answer`, or starts workline-answer.yml, which then takes
// a manual start.
func TestObjectionHeartbeatResumes(t *testing.T) {
	for _, dir := range answerCIDirs {
		if reaches, why := answerCIHeartbeatReaches(t, dir); !reaches {
			t.Errorf("%s: an objection no new comment wakes stays unanswered: %s", dir, why)
		}
	}
}

// Reply wakes
// Given an answered objection,
// When the developer or a person replies in its thread,
// Then the tester runs and answers that reply: workline-objection.yml
// notices a reply on a tester's Draft (branch workline/tester/…), whose
// body opens with no `workline:objection`, from an App too (the
// developer's), so that its end wakes workline-answer.yml.
func TestObjectionReplyWakesTheTester(t *testing.T) {
	for _, dir := range answerCIDirs {
		objection := answerCIRead(t, filepath.Join(dir, "workline-objection.yml"))
		noticed := false
		for name, j := range objection.Jobs {
			cond := strings.Join(strings.Fields(j.If), " ")
			if !strings.Contains(cond, "workline/tester/") {
				continue
			}
			opensWithWord := strings.Contains(cond, "comment.body")
			if opensWithWord && !strings.Contains(cond, "in_reply_to_id") {
				continue // an objection's first comment alone
			}
			if strings.Contains(cond, "github.event.comment.user.type != 'Bot'") && !strings.Contains(cond, "|| ") {
				continue // a person's alone: the developer's reply is an App's
			}
			t.Logf("%s: job %s notices a reply on a tester's Draft", dir, name)
			noticed = true
		}
		if !noticed {
			t.Errorf("%s/workline-objection.yml: no job notices a reply in a thread on a tester's Draft (workline/tester/…) whose body opens with no `workline:objection`: it is skipped, and nothing wakes the answer", dir)
		}
	}
}

// answerCICode is a workflow file's text without its comment lines, to look
// for a name the workflow really uses.
func answerCICode(t *testing.T, f string) string {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var code []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		code = append(code, line)
	}
	return strings.Join(code, "\n")
}

// Queue drained: the answer reads the Draft's tip
// Given a Draft whose branch moved since the objection run (a re-run, a push),
// When the next answer run starts,
// Then workline-answer.yml reads the open pull request's tip (`headRefOid`),
// never the objection run's `workflow_run.head_sha`, which it leaves behind.
func TestObjectionAnswerReadsTheDraftsTip(t *testing.T) {
	for _, dir := range answerCIDirs {
		code := answerCICode(t, filepath.Join(dir, "workline-answer.yml"))
		if strings.Contains(code, "workflow_run.head_sha") {
			t.Errorf("%s/workline-answer.yml: reads workflow_run.head_sha, the head of the objection run, which a re-run or a push since leaves behind: it must read the open pull request's tip", dir)
		}
		if !strings.Contains(code, "headRefOid") {
			t.Errorf("%s/workline-answer.yml: never reads the open pull request's tip (headRefOid)", dir)
		}
	}
}

// Heartbeat resumes: the idle check counts the answers
// Given an unanswered objection and no new comment for an hour,
// When the heartbeat runs,
// Then its idle check counts `.answers` too, so that a Draft with an
// answer waiting is not taken for idle.
func TestObjectionHeartbeatIdleCountsAnswers(t *testing.T) {
	for _, dir := range answerCIDirs {
		code := answerCICode(t, filepath.Join(dir, "workline-develop.yml"))
		if !strings.Contains(code, ".answers") {
			t.Errorf("%s/workline-develop.yml: the heartbeat's idle check does not count `.answers`: an objection waiting for an answer is taken for idle", dir)
		}
	}
}

// answerCITriedWorkflow is what the state step's test reads of a workflow:
// each job's condition, and each step's condition and command.
type answerCITriedWorkflow struct {
	Jobs map[string]struct {
		If    string `yaml:"if"`
		Steps []struct {
			If  string `yaml:"if"`
			Run string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// answerCITriedCondition gives the condition of the step that writes the
// answer's state (`--tried`), and its job's, each on one line.
func answerCITriedCondition(t *testing.T, f string) (step, job string) {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var wf answerCITriedWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	for _, j := range wf.Jobs {
		for _, s := range j.Steps {
			if strings.Contains(s.Run, "--tried") {
				return strings.Join(strings.Fields(s.If), " "), strings.Join(strings.Fields(j.If), " ")
			}
		}
	}
	t.Fatalf("%s: no step writes the answer's state (`--tried`)", f)
	return "", ""
}

// Heartbeat resumes: an unreachable agent leaves no tried mark
// Given an unanswered objection whose answer ran with the agent unreachable
// (status 3),
// When the heartbeat runs,
// Then the tester answers it: the answer's state step is not run for that
// status, so that no tried mark stops the reconciler, and the template and
// workline's own workflow give that step the same condition.
func TestObjectionUnreachableAgentLeavesNoTriedMark(t *testing.T) {
	var steps []string
	for _, dir := range answerCIDirs {
		step, job := answerCITriedCondition(t, filepath.Join(dir, "workline-answer.yml"))
		steps = append(steps, step)
		both := step + " " + job
		if !strings.Contains(both, "status") || !strings.Contains(both, "3") {
			t.Errorf("%s/workline-answer.yml: the state step runs whatever the status (step if: %q, job if: %q): an unreachable agent (status 3) gets marked tried and the reconciler never retries it", dir, step, job)
		}
	}
	if len(steps) == 2 && steps[0] != steps[1] {
		t.Errorf("the two copies of the answer's state step differ: %s has %q, %s has %q", answerCIDirs[0], steps[0], answerCIDirs[1], steps[1])
	}
}
