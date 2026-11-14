package conformance

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v3"
)

// sampleRecordStep is a step of the weekly sample's workflow, as the test
// below reads it.
type sampleRecordStep struct {
	ID  string            `yaml:"id"`
	Env map[string]string `yaml:"env"`
	Run string            `yaml:"run"`
}

// Given a merged developer pull request whose record the App wrote, When the
// weekly sample runs in CI, Then its row is judged, not `not-read`: the step
// that reads the sample names the App, as the developer's own workflow does,
// so the records the App wrote are the engine's own account's, and read.
func TestSampleReadStepNamesTheApp(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/workline-sample.yml")
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Env   map[string]string  `yaml:"env"`
			Steps []sampleRecordStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	read, ok := wf.Jobs["read"]
	if !ok {
		t.Fatal("workline-sample.yml: no job read")
	}
	found := false
	for _, s := range read.Steps {
		if s.ID != "sample" {
			continue
		}
		found = true
		const want = "${{ vars.WORKLINE_APP_ID }}"
		got := s.Env["WORKLINE_APP_ID"]
		if got == "" {
			got = read.Env["WORKLINE_APP_ID"]
		}
		if got != want {
			t.Errorf("workline-sample.yml, read, the sample's step: WORKLINE_APP_ID = %q, want %q, the App whose comments keep the developer's and the tester's records", got, want)
		}
	}
	if !found {
		t.Error("workline-sample.yml, read: no step with the id sample")
	}
}
