package conformance

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// workline's own pull requests read its Go code with golangci-lint, a gate
// of its line (ADR-0049, step 5). Each tool a gate of the merge requests'
// line runs is installed in workline.yml's gates job before any gate runs,
// on a pull request and on main after a merge alike: an exact version, the
// archive's SHA-256 written in the workflow and checked, in the runner's
// own folder, never with sudo. A tool taken at its latest version, or
// checked against a file downloaded beside it, could read other findings
// from one run to the next, and the merge base's kept findings would not
// stand for the head's.

var (
	pinnedVersion = regexp.MustCompile(`\b[0-9]+\.[0-9]+\.[0-9]+\b`)
	pinnedSHA256  = regexp.MustCompile(`\b[0-9a-f]{64}\b`)
)

func TestOwnGatesToolsArePinned(t *testing.T) {
	data, err := os.ReadFile("../../.workline/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Routing struct {
			Events map[string][]string `yaml:"events"`
		} `yaml:"routing"`
		Gates map[string]struct {
			Checks []struct {
				ID  string `yaml:"id"`
				Run string `yaml:"run"`
			} `yaml:"checks"`
		} `yaml:"gates"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	var tools []string
	for _, step := range cfg.Routing.Events["merge-request"] {
		name, ok := strings.CutPrefix(step, "gate:")
		if !ok {
			continue
		}
		g, ok := cfg.Gates[name]
		if !ok {
			t.Errorf("the merge requests' line runs gate %s, which gates: does not declare", name)
			continue
		}
		for _, c := range g.Checks {
			if f := strings.Fields(c.Run); len(f) > 0 && !slices.Contains(tools, f[0]) {
				tools = append(tools, f[0])
			}
		}
	}
	if !slices.Contains(tools, "golangci-lint") {
		t.Fatalf("workline's own merge requests run no golangci-lint gate (tools %v)", tools)
	}

	data, err = os.ReadFile("../../.github/workflows/workline.yml")
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []engineStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	steps := wf.Jobs["gates"].Steps
	first, engine := -1, -1
	for i, s := range steps {
		if first < 0 && (strings.Contains(s.Run, "--gates-only") || strings.Contains(s.Run, "workline route main")) {
			first = i
		}
		if s.Uses == engineAction {
			engine = i
			// golangci-lint loads the code with Go: set up, the engine
			// taken from the cache or built.
			if s.With["go"] != "true" {
				t.Errorf("gates job: the engine's action leaves no Go for golangci-lint when it takes the engine from the cache (with %v)", s.With)
			}
		}
	}
	if first < 0 || engine < 0 {
		t.Fatalf("workline.yml's gates job runs no gate, or takes no engine: %+v", steps)
	}
	for _, tool := range tools {
		at := slices.IndexFunc(steps, func(s engineStep) bool {
			return strings.Contains(s.Run, "curl") && strings.Contains(s.Run, tool)
		})
		if at < 0 {
			t.Errorf("gates job: %s, which a gate runs, is not installed", tool)
			continue
		}
		s := steps[at]
		pins := s.Run
		for k, v := range s.Env {
			pins += "\n" + k + "=" + v
		}
		switch {
		case at > first:
			t.Errorf("gates job: %s is installed after the gates ran", tool)
		case s.If != steps[engine].If:
			t.Errorf("gates job: %s is installed only %q, the gates run %q", tool, s.If, steps[engine].If)
		case !pinnedVersion.MatchString(pins) || strings.Contains(pins, "latest"):
			t.Errorf("gates job: %s is installed at no exact version:\n%s", tool, pins)
		case !pinnedSHA256.MatchString(pins) || !strings.Contains(s.Run, "sha256sum -c"):
			t.Errorf("gates job: %s's archive is not checked against a SHA-256 the workflow holds:\n%s", tool, pins)
		case strings.Contains(s.Run, "sudo") || !strings.Contains(s.Run, "$RUNNER_TEMP"):
			t.Errorf("gates job: %s is installed outside the runner's own folder:\n%s", tool, s.Run)
		}
	}
}
