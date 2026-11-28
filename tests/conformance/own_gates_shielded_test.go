package conformance

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The gates step of workline's own head gates job runs the pull request's
// code through a shell: whatever fails in that shell outside the commands
// it guards (a failed write to the summary) must show as a warning, never
// turn the check red. The step carries continue-on-error, and a later
// step of the job reads its outcome and says it.

type ownGatesJobStep struct {
	Name            string         `yaml:"name"`
	ID              string         `yaml:"id"`
	If              string         `yaml:"if"`
	Uses            string         `yaml:"uses"`
	With            map[string]any `yaml:"with"`
	Env             map[string]any `yaml:"env"`
	Run             string         `yaml:"run"`
	ContinueOnError any            `yaml:"continue-on-error"`
}

// text is all that a step reads or runs, for what it refers to.
func (s ownGatesJobStep) text() string {
	out := s.If + "\n" + s.Run
	for k, v := range s.Env {
		out += fmt.Sprintf("\n%s=%v", k, v)
	}
	for k, v := range s.With {
		out += fmt.Sprintf("\n%s=%v", k, v)
	}
	return out
}

// ownGatesShield reads the gates job of workline.yml: the step with id
// gates, and the steps after it that read its outcome.
func ownGatesShield(t *testing.T) (gates ownGatesJobStep, readers []ownGatesJobStep) {
	t.Helper()
	data, err := os.ReadFile("../../.github/workflows/workline.yml")
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []ownGatesJobStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	steps := wf.Jobs["gates"].Steps
	at := -1
	for i, s := range steps {
		if s.ID == "gates" {
			at = i
			gates = s
		}
	}
	if at < 0 {
		t.Fatalf("workline.yml's gates job has no step with id gates: %+v", steps)
	}
	for _, s := range steps[at+1:] {
		if strings.Contains(s.text(), "steps.gates.outcome") {
			readers = append(readers, s)
		}
	}
	return gates, readers
}

func continuesOnError(s ownGatesJobStep) bool {
	return fmt.Sprint(s.ContinueOnError) == "true"
}

// Shell fails in the gates step
// Given a failure outside the guarded commands, such as a failed write to
// the summary,
// When the head gates job runs,
// Then the check shows a warning and not red: the gates step continues on
// error, and a later step reads its outcome and says a warning.
func TestOwnGatesStepShellFailureIsAWarning(t *testing.T) {
	gates, readers := ownGatesShield(t)
	if !continuesOnError(gates) {
		t.Errorf("the gates step %q has no continue-on-error: true, so a shell error outside its guarded commands turns the job red", gates.Name)
	}
	if len(readers) == 0 {
		t.Fatal("no step after the gates step reads steps.gates.outcome: a failure of the step itself is never said")
	}
	said := false
	for _, r := range readers {
		if strings.Contains(r.text(), "::warning") {
			said = true
		}
	}
	if !said {
		t.Errorf("the step reading steps.gates.outcome says no ::warning when the gates step failed")
	}
}

var ownGatesOutcomeWord = regexp.MustCompile(`\b(success|failure)\b`)

// Gates pass
// Given gates that pass,
// When the job runs,
// Then the check stays green: the step reading the gates step's outcome
// tells a success from a failure, so it fails nothing when the step
// succeeded, and the gates step still carries its shield.
func TestOwnGatesStepPassingGatesStayGreen(t *testing.T) {
	gates, readers := ownGatesShield(t)
	if !continuesOnError(gates) {
		t.Errorf("the gates step %q has no continue-on-error: true", gates.Name)
	}
	if len(readers) == 0 {
		t.Fatal("no step after the gates step reads steps.gates.outcome")
	}
	for _, r := range readers {
		if !ownGatesOutcomeWord.MatchString(r.text()) {
			t.Errorf("step %q reads steps.gates.outcome without telling success from failure:\n%s", r.Name, r.text())
		}
	}
}
