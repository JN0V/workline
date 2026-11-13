package conformance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// ci-github-thread-reply-wakes-the-round (#1438): a person's reply in a
// review thread on a line of the developer's pull request is noticed by
// workline-objection.yml, which holds no token, as an objection is; its
// end wakes workline-develop.yml's whose, which runs as the default
// branch holds it: a round, the project's runs not read. The App's reply
// is noticed by no job, and wakes nothing. In the templates and in
// workline's own workflows.
func TestCIGitHubThreadReplyWakesTheRound(t *testing.T) {
	type step struct {
		ID  string            `yaml:"id"`
		Run string            `yaml:"run"`
		Env map[string]string `yaml:"env"`
	}
	type job struct {
		If          string            `yaml:"if"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []step            `yaml:"steps"`
	}
	read := func(f string) (on map[string]any, jobs map[string]job) {
		t.Helper()
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			On   map[string]any `yaml:"on"`
			Jobs map[string]job `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		return wf.On, wf.Jobs
	}
	norm := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, dir := range []string{"../../ci/github", "../../.github/workflows"} {
		on, jobs := read(filepath.Join(dir, "workline-objection.yml"))
		replied := jobs["replied"]
		if _, ok := on["pull_request_review_comment"]; !ok || len(replied.Permissions) != 0 || replied.Permissions == nil ||
			!strings.Contains(norm(replied.If), "startsWith(github.event.pull_request.head.ref, 'workline/developer/')") ||
			!strings.Contains(norm(replied.If), "github.event.comment.user.type != 'Bot'") ||
			!strings.Contains(norm(replied.If), "github.event.pull_request.head.repo.full_name == github.repository") ||
			!strings.Contains(norm(replied.If), "github.event.comment.author_association") {
			t.Errorf("%s/workline-objection.yml, replied: does not notice a person's reply on the developer's pull request alone, with no token: %q %v", dir, replied.If, replied.Permissions)
		}
		_, jobs = read(filepath.Join(dir, "workline-develop.yml"))
		whose := jobs["whose"]
		if !strings.Contains(norm(whose.If), "(github.event_name == 'workflow_run' && github.event.workflow_run.name == 'workline objection' && github.event.workflow_run.conclusion == 'success' && github.event.workflow_run.pull_requests[0] != null)") {
			t.Errorf("%s/workline-develop.yml, whose: the objection workflow's end does not wake it: %q", dir, whose.If)
		}
		var pr step
		for _, s := range whose.Steps {
			if s.ID == "pr" {
				pr = s
			}
		}
		for _, c := range []struct {
			name, jobs string // jobs: the objection run's jobs, as the API lists them
			round      bool
		}{
			{"a person's reply", `{"jobs":[{"name":"noticed","conclusion":"skipped"},{"name":"replied","conclusion":"success"}]}`, true},
			{"no reply noticed", `{"jobs":[{"name":"noticed","conclusion":"skipped"},{"name":"replied","conclusion":"skipped"}]}`, false},
		} {
			t.Run(filepath.Base(dir)+"/"+c.name, func(t *testing.T) {
				tmp := t.TempDir()
				bin := filepath.Join(tmp, "bin")
				if err := os.MkdirAll(bin, 0o755); err != nil {
					t.Fatal(err)
				}
				// gh: the run's jobs, filtered by the step's own jq; the
				// pull request open, a Draft or not, from the developer.
				gh := fmt.Sprintf(`#!/bin/sh
case "$1 $2" in
  "api repos/o/r/actions/runs/77/jobs") shift 2; [ "$1" = --jq ] && printf '%%s' '%s' | jq -r "$2" ;;
  "api "*) echo "unexpected: $*" >&2; exit 9 ;;
  "pr view") echo "workline/developer/26 false" ;;
esac
`, c.jobs)
				if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(gh), 0o755); err != nil {
					t.Fatal(err)
				}
				out := filepath.Join(tmp, "output")
				run := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", pr.Run)
				run.Dir = tmp
				run.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GH_REPO=o/r", "PR=33", "SHA=abc123",
					"RUN=workline objection", "RUN_ID=77", "GITHUB_OUTPUT="+out)
				said, err := run.CombinedOutput()
				if err != nil {
					t.Fatalf("the step failed: %v\n%s", err, said)
				}
				got, _ := os.ReadFile(out)
				if round := strings.Contains(string(got), "branch=workline/developer/26"); round != c.round {
					t.Errorf("round asked: %v, want %v\n%s", round, c.round, said)
				}
			})
		}
	}
}
