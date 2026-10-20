package conformance

import (
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// workline's own pull requests read its Go code with golangci-lint, a gate
// of its line (ADR-0049, step 5). Each tool a gate of the merge requests'
// line runs is installed in workline.yml's gates job before any gate runs,
// on a pull request and on main alike: an exact version, the archive's
// SHA-256 written in the workflow and checked, in the runner's own folder,
// never with sudo, so that every run, and the base main kept, read alike.

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
			if tool := toolOf(c.Run); tool != "" && !slices.Contains(tools, tool) {
				tools = append(tools, tool)
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
		if tool == "go" { // set up by the engine's action, checked above
			continue
		}
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
		case !checksDigest(s):
			t.Errorf("gates job: %s's archive is not checked against a SHA-256 the workflow holds:\n%s", tool, pins)
		case strings.Contains(s.Run, "sudo") || !strings.Contains(s.Run, "$RUNNER_TEMP"):
			t.Errorf("gates job: %s is installed outside the runner's own folder:\n%s", tool, s.Run)
		}
	}
}

// toolOf is the tool a check's command runs: its first word past any
// `NAME=value` set before it, by its base name.
func toolOf(run string) string {
	for _, w := range strings.Fields(run) {
		if name, _, ok := strings.Cut(w, "="); ok && name != "" && !strings.ContainsAny(name, "/-.") {
			continue
		}
		return path.Base(w)
	}
	return ""
}

// checksDigest says whether a step checks what it downloaded against a
// SHA-256 it holds: the line running `sha256sum -c` holds the digest, or
// a variable the step's env sets to one; never a file fetched beside it.
func checksDigest(s engineStep) bool {
	for _, l := range strings.Split(s.Run, "\n") {
		if !strings.Contains(l, "sha256sum -c") || strings.Contains(l, "curl") {
			continue
		}
		if pinnedSHA256.MatchString(l) {
			return true
		}
		for k, v := range s.Env {
			if pinnedSHA256.MatchString(v) && (strings.Contains(l, "$"+k+" ") || strings.Contains(l, "${"+k+"}")) {
				return true
			}
		}
	}
	return false
}

func TestOwnGatesToolsArePinnedReadsTheInstall(t *testing.T) {
	sha := strings.Repeat("ab", 32)
	for run, want := range map[string]string{
		"golangci-lint run ./...":       "golangci-lint",
		"GOFLAGS=-mod=mod go vet ./...": "go",
		"./bin/lint --all":              "lint",
	} {
		if got := toolOf(run); got != want {
			t.Errorf("toolOf(%q) = %q, want %q", run, got, want)
		}
	}
	for _, c := range []struct {
		name string
		step engineStep
		want bool
	}{
		{"the digest held in env", engineStep{Env: map[string]string{"SHA256": sha}, Run: `echo "$SHA256  $f" | sha256sum -c -`}, true},
		{"the digest on the line", engineStep{Run: `echo "` + sha + `  f.tar.gz" | sha256sum -c`}, true},
		{"a checksums file fetched beside it", engineStep{Env: map[string]string{"SHA256": sha}, Run: "# " + sha + "\ncurl -fsSL $u/checksums.txt | grep f.tar.gz | sha256sum -c -"}, false},
		{"a digest in env, not the one checked", engineStep{Env: map[string]string{"SHA256": sha}, Run: "sha256sum -c sums.txt"}, false},
	} {
		if got := checksDigest(c.step); got != c.want {
			t.Errorf("%s: checksDigest = %v, want %v", c.name, got, c.want)
		}
	}
}
