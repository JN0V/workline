package github

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Kept when it fails
// Given the tester's answer run on the CI,
// When its status is not 0,
// Then its result file (answer.json) and run folder are kept as an
// artifact: the upload runs for a failed run too, and takes both.
func TestAnswerKeepsItsResultAndRunsWhenItFails(t *testing.T) {
	data, err := os.ReadFile("workline-answer.yml")
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]struct {
			Steps []struct {
				If   string         `yaml:"if"`
				Uses string         `yaml:"uses"`
				With map[string]any `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	found := false
	for job, j := range w.Jobs {
		for _, s := range j.Steps {
			if !strings.Contains(s.Uses, "upload-artifact") || fmt.Sprint(s.With["name"]) != "workline-answer" {
				continue
			}
			found = true
			path := fmt.Sprint(s.With["path"])
			if !strings.Contains(path, "answer.json") {
				t.Errorf("%s: the artifact keeps %q, not the result file answer.json", job, path)
			}
			if !strings.Contains(path, ".workline-runs") {
				t.Errorf("%s: the artifact keeps %q, not the run folder .workline-runs", job, path)
			}
			if !strings.Contains(s.If, "always()") && !strings.Contains(s.If, "failure()") && !strings.Contains(s.If, "outputs.status") {
				t.Errorf("%s: the upload runs `if: %s`, which leaves a failed run's result and runs unkept", job, s.If)
			}
		}
	}
	if !found {
		t.Fatal("no upload of the workline-answer artifact found")
	}
}
