// Package scripts tests the maintainer's scripts: what scripts/try-on-copy.sh
// and the manual workflow running it say they do. A real try needs GitHub
// and a trial copy; these tests read the script and the workflow, and
// never reach a forge.
package scripts

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// tryScript reads scripts/try-on-copy.sh, failing the test when it is not there.
func tryScript(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("try-on-copy.sh")
	if err != nil {
		t.Fatalf("scripts/try-on-copy.sh: %v", err)
	}
	return string(data)
}

// tryWorkflow is .github/workflows/try.yml, as far as the tests read it.
type tryWorkflow struct {
	On          any `yaml:"on"`
	Concurrency any `yaml:"concurrency"`
	Jobs        map[string]struct {
		ContinueOnError any `yaml:"continue-on-error"`
		Steps           []struct {
			Run             string `yaml:"run"`
			ContinueOnError any    `yaml:"continue-on-error"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// loadTry reads .github/workflows/try.yml, failing the test when it is not there.
func loadTry(t *testing.T) tryWorkflow {
	t.Helper()
	data, err := os.ReadFile("../.github/workflows/try.yml")
	if err != nil {
		t.Fatalf(".github/workflows/try.yml: %v", err)
	}
	var w tryWorkflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("try.yml does not parse: %v", err)
	}
	return w
}

// Try a ref
// Given a pull request head,
// When the maintainer runs `scripts/try-on-copy.sh <ref>`,
// Then a scratch repository is made afresh from the trial copy at its
// fixed tag, with that commit's engine and that commit's workflow
// templates, and one issue runs end to end.
func TestTryOnCopyRefMakesAScratchFromTheTrialCopyWithTheCommitsEngineAndTemplates(t *testing.T) {
	s := tryScript(t)
	for _, want := range []string{
		"JN0V/TraceGlass-trial", // the trial copy, the base
		"gh repo create",        // a scratch repository, made afresh
		"cmd/workline",          // the engine built from the commit
		"defaults/ci/github",    // the commit's workflow templates
	} {
		if !strings.Contains(s, want) {
			t.Errorf("try-on-copy.sh never says %q", want)
		}
	}
	if !regexp.MustCompile(`(?i)tag`).MatchString(s) {
		t.Errorf("try-on-copy.sh names no fixed tag of the trial copy")
	}
	if !strings.Contains(s, "issue") {
		t.Errorf("try-on-copy.sh opens no issue to run end to end")
	}
}

// Try a release
// Given a published version,
// When the maintainer runs `scripts/try-on-copy.sh <version>`,
// Then the scratch repository downloads that release as the pin would,
// and one issue runs end to end.
func TestTryOnCopyVersionDownloadsTheReleaseAsThePinWould(t *testing.T) {
	s := tryScript(t)
	for _, want := range []string{
		"WORKLINE_VERSION", // the pin the templates carry
		"releases/download",
		"checksums.txt",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("try-on-copy.sh never says %q: a release is not downloaded as the pin does", want)
		}
	}
}

// Forge facts
// Given a try that ran,
// When it ends,
// Then it prints green or red and the run's link, green only if a pull
// request was opened, its checks are green and its labels are set.
func TestTryOnCopyPrintsGreenOrRedFromTheForgesFacts(t *testing.T) {
	s := tryScript(t)
	for _, want := range []string{"green", "red", "gh pr", "checks", "labels"} {
		if !strings.Contains(s, want) {
			t.Errorf("try-on-copy.sh never says %q", want)
		}
	}
	if !regexp.MustCompile(`(?i)url|actions/runs`).MatchString(s) {
		t.Errorf("try-on-copy.sh prints no link to the run")
	}
}

// From GitHub
// Given the manual try.yml workflow,
// When the maintainer starts it with a ref or a version,
// Then it runs the same script and its run concludes success or
// failure accordingly.
func TestTryWorkflowRunsTheScriptByHand(t *testing.T) {
	w := loadTry(t)
	on, ok := w.On.(map[string]any)
	if !ok {
		t.Fatalf("try.yml's `on` is %v: want a map holding workflow_dispatch", w.On)
	}
	dispatch, ok := on["workflow_dispatch"]
	if !ok || len(on) != 1 {
		t.Errorf("try.yml is started by %v: want workflow_dispatch alone, by hand", on)
	}
	if in, _ := dispatch.(map[string]any); in == nil || in["inputs"] == nil {
		t.Errorf("try.yml's workflow_dispatch takes no input: it cannot be given a ref or a version")
	}
	ran := false
	for name, j := range w.Jobs {
		if j.ContinueOnError != nil && j.ContinueOnError != false {
			t.Errorf("job %s: continue-on-error hides the script's failure", name)
		}
		for _, s := range j.Steps {
			if !strings.Contains(s.Run, "scripts/try-on-copy.sh") {
				continue
			}
			ran = true
			if s.ContinueOnError != nil && s.ContinueOnError != false {
				t.Errorf("job %s: continue-on-error hides the script's failure", name)
			}
			if strings.Contains(s.Run, "|| true") {
				t.Errorf("job %s: `|| true` makes the run succeed whatever the script says", name)
			}
		}
	}
	if !ran {
		t.Error("no step of try.yml runs scripts/try-on-copy.sh")
	}
}

// One at a time
// Given a try running,
// When a second one starts,
// Then try.yml queues it, and the script refuses with a message naming
// the running try.
func TestTryQueuesAndTheScriptRefusesWhileOneRuns(t *testing.T) {
	w := loadTry(t)
	switch c := w.Concurrency.(type) {
	case string:
		if c == "" {
			t.Error("try.yml's concurrency group is empty")
		}
	case map[string]any:
		if c["group"] == nil || c["group"] == "" {
			t.Error("try.yml's concurrency has no group")
		}
		if c["cancel-in-progress"] == true {
			t.Error("try.yml cancels the try running when a second starts: it must queue it")
		}
	default:
		t.Errorf("try.yml has no concurrency group: %v", w.Concurrency)
	}

	s := tryScript(t)
	if !strings.Contains(s, "try.yml") {
		t.Error("try-on-copy.sh never looks for a running try.yml")
	}
	if !regexp.MustCompile(`in_progress|queued`).MatchString(s) {
		t.Error("try-on-copy.sh never reads whether a run is in progress")
	}
	if !regexp.MustCompile(`(?m)exit [1-9]`).MatchString(s) {
		t.Error("try-on-copy.sh never exits with a failure: it does not refuse")
	}
}

// The copy kept
// Given a try ran,
// When the maintainer looks at the trial copy,
// Then it is as it was: only the scratch repository changed.
func TestTryOnCopyChangesOnlyTheScratchRepository(t *testing.T) {
	s := tryScript(t)
	if !strings.Contains(s, "JN0V/TraceGlass-trial") {
		t.Fatal("try-on-copy.sh never names the trial copy it starts from")
	}
	if !strings.Contains(s, "gh repo create") {
		t.Error("try-on-copy.sh makes no scratch repository: it would write to the copy")
	}
	for _, line := range strings.Split(s, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "#") || !strings.Contains(l, "TraceGlass-trial") {
			continue
		}
		for _, bad := range []string{"push", "repo delete", "--force", "--method", "-X ", "gh pr create", "gh issue", "gh workflow run"} {
			if strings.Contains(l, bad) {
				t.Errorf("try-on-copy.sh writes to the trial copy: %q", l)
			}
		}
	}
}
