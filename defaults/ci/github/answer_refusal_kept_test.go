package github

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// answerStep is a step of the answer workflow's propose job.
type answerStep struct {
	Name string         `yaml:"name"`
	If   string         `yaml:"if"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
	Env  map[string]any `yaml:"env"`
}

// answerProposeSteps reads the steps of the propose job of an answer workflow.
func answerProposeSteps(t *testing.T, file string) []answerStep {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]struct {
			Steps []answerStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return w.Jobs["propose"].Steps
}

// The reason in the job
// Given an answer the tester refuses, on the CI,
// When the answer job ends,
// Then the Verdict step prints the summary as an error annotation, and
// the job keeps answer.json and the run's folder as an artifact, a
// refused answer included (it leaves no pending run). It reads the
// template the project ships.
func TestAnswerRefusalIsAnnotatedAndKept(t *testing.T) {
	file := "workline-answer.yml"
	var upload, verdict *answerStep
	steps := answerProposeSteps(t, file)
	for i := range steps {
		s := &steps[i]
		switch {
		case strings.Contains(s.Uses, "upload-artifact") && fmt.Sprint(s.With["name"]) == "workline-answer":
			upload = s
		case s.Name == "Verdict":
			verdict = s
		}
	}
	if upload == nil || verdict == nil {
		t.Fatalf("%s: propose job lacks its answer artifact upload (%v) or its Verdict step (%v)", file, upload != nil, verdict != nil)
	}
	if c := upload.If; !strings.Contains(c, "always()") && !strings.Contains(c, "failure()") && !strings.Contains(c, "outputs.status") {
		t.Errorf("%s: the answer artifact is uploaded `if: %s`, which a refused answer (no pending run) never meets", file, c)
	}
	path := fmt.Sprint(upload.With["path"])
	for _, want := range []string{".workline-runs", "answer.json"} {
		if !strings.Contains(path, want) {
			t.Errorf("%s: the answer artifact's path %q does not keep %s", file, path, want)
		}
	}
	text := verdict.Run
	for _, v := range verdict.Env {
		text += "\n" + fmt.Sprint(v)
	}
	if !strings.Contains(verdict.Run, "::error") {
		t.Errorf("%s: the Verdict step prints no error annotation:\n%s", file, verdict.Run)
	}
	if !strings.Contains(text, "answer.json") && !strings.Contains(strings.ToLower(text), "summary") {
		t.Errorf("%s: the Verdict step does not read the answer's summary:\n%s", file, text)
	}
}
