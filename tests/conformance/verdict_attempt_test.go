package conformance

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Re-run reads its own verdict: Given a judge job re-run after a failure,
// When the verdict job runs, Then it reads the verdict of the latest attempt:
// the artifact is named per attempt, or the download selects the attempt.
func TestReRunReadsItsOwnVerdict(t *testing.T) {
	type step struct {
		Uses string         `yaml:"uses"`
		Run  string         `yaml:"run"`
		With map[string]any `yaml:"with"`
	}
	type file struct {
		Jobs map[string]struct {
			Steps []step `yaml:"steps"`
		} `yaml:"jobs"`
	}
	load := func(f string) file {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf file
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		return wf
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		perAttempt := false
		for _, job := range load(dir + "workline.yml").Jobs {
			for _, s := range job.Steps {
				if strings.HasPrefix(s.Uses, "actions/upload-artifact") {
					name, _ := s.With["name"].(string)
					if strings.HasPrefix(name, "workline-verdict") && strings.Contains(name, "run_attempt") {
						perAttempt = true
					}
				}
			}
		}
		selects := false
		for _, job := range load(dir + "workline-apply.yml").Jobs {
			for _, s := range job.Steps {
				if strings.Contains(s.Run, "workline-verdict") && strings.Contains(s.Run, "run_attempt") {
					selects = true
				}
			}
		}
		if !perAttempt && !selects {
			t.Errorf("%s: the verdict artifact is not named per attempt (github.run_attempt), nor the download made for the latest attempt", dir)
		}
	}
}
