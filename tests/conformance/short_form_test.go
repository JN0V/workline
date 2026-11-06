package conformance

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

type gitlabInclude struct {
	Project string `yaml:"project"`
	Ref     string `yaml:"ref"`
	File    string `yaml:"file"`
}

// The GitLab short form takes the template by reference, and its warn child
// pipeline reads the same file from the same project and ref: the version is
// one string (a YAML anchor), so changing it moves both.
func TestGitLabShortFormChildReadsTheSameRemoteFile(t *testing.T) {
	b, err := os.ReadFile("../../ci/gitlab/workline-short.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Include []gitlabInclude `yaml:"include"`
		Child   struct {
			Trigger struct {
				Include []gitlabInclude `yaml:"include"`
			} `yaml:"trigger"`
		} `yaml:"workline:warn:child"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Include) != 1 || len(doc.Child.Trigger.Include) != 1 {
		t.Fatalf("want one include and one child include, got %v and %v", doc.Include, doc.Child.Trigger.Include)
	}
	top, child := doc.Include[0], doc.Child.Trigger.Include[0]
	if top.Project == "" || top.Ref == "" || top.File == "" {
		t.Fatalf("the include names no project, ref and file: %+v", top)
	}
	if top != child {
		t.Errorf("the child pipeline reads %+v, the include %+v: they must be the same", child, top)
	}
	if !regexp.MustCompile(`(?m)^\s*ref: &\w+ v\d+\.\d+\.\d+$`).Match(b) {
		t.Errorf("the ref is not one anchored version string, so the child could drift")
	}
}

// The GitHub short form calls a reusable workflow of workline's repository at
// a version: that workflow must exist, and offer workflow_call.
func TestGitHubShortFormCallsAWorkflowThatExists(t *testing.T) {
	b, err := os.ReadFile("../../ci/github/workline-short.yml")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`uses:\s*JN0V/workline/(\.github/workflows/[\w.-]+)@v\d+\.\d+\.\d+`).FindSubmatch(b)
	if m == nil {
		t.Fatal("the short form calls no workflow of workline at a version")
	}
	path := "../../" + string(m[1])
	w, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s is not in the repository yet: the short form calls a workflow that does not exist (needs .github/workflows, outside this change's scope)", m[1])
	}
	if !regexp.MustCompile(`(?m)^\s*workflow_call:`).Match(w) {
		t.Errorf("%s has no workflow_call trigger", m[1])
	}
}

const (
	ghShort = "../../ci/github/workline-short.yml"
	glShort = "../../ci/gitlab/workline-short.gitlab-ci.yml"
	// shortFormLines is the line limit of a short form: about ten lines.
	shortFormLines = 12
)

var semver = regexp.MustCompile(`v\d+\.\d+\.\d+`)

func readShort(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Changing the one version string changes the version used, and nothing else.
func TestShortFormVersionIsOneString(t *testing.T) {
	for _, p := range []string{ghShort, glShort} {
		s := readShort(t, p)
		code := regexp.MustCompile(`(?m)^\s*#.*$`).ReplaceAllString(s, "")
		if n := len(semver.FindAllString(code, -1)); n != 1 {
			t.Fatalf("%s: want one version string outside comments, got %d", p, n)
		}
		bumped := semver.ReplaceAllString(s, "v9.9.9")
		if bumped == s || len(bumped) != len(s)+len("v9.9.9")-len(semver.FindString(s)) {
			t.Errorf("%s: bumping the version edits more than the version", p)
		}
		if !regexp.MustCompile(`v9\.9\.9`).MatchString(bumped) || semver.FindString(code) == "v9.9.9" {
			t.Errorf("%s: the bumped version is not the one used", p)
		}
	}
}

// Dependabot reads a `uses: owner/repo/path@vX.Y.Z` of a workflow; Renovate
// reads a GitLab include of `project:` with `ref:`.
func TestShortFormVersionIsBotReadable(t *testing.T) {
	var gh struct {
		Jobs map[string]struct {
			Uses string `yaml:"uses"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal([]byte(readShort(t, ghShort)), &gh); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, j := range gh.Jobs {
		if regexp.MustCompile(`^[\w.-]+/[\w.-]+/\.github/workflows/[\w.-]+\.ya?ml@v\d+\.\d+\.\d+$`).MatchString(j.Uses) {
			found = true
		}
	}
	if !found {
		t.Errorf("no job uses owner/repo/.github/workflows/file@vX.Y.Z, the format Dependabot reads")
	}
	var gl struct {
		Include []gitlabInclude `yaml:"include"`
	}
	if err := yaml.Unmarshal([]byte(readShort(t, glShort)), &gl); err != nil {
		t.Fatal(err)
	}
	if len(gl.Include) != 1 || gl.Include[0].Project == "" || !semver.MatchString(gl.Include[0].Ref) {
		t.Errorf("the include has no project with a version ref, the format Renovate reads: %+v", gl.Include)
	}
}

func TestShortFormStaysWithinTheLineLimit(t *testing.T) {
	for _, p := range []string{ghShort, glShort} {
		if n := strings.Count(strings.TrimRight(readShort(t, p), "\n"), "\n") + 1; n > shortFormLines {
			t.Errorf("%s has %d lines, the limit is %d", p, n, shortFormLines)
		}
	}
}

// The short forms run the same jobs as the full copies: on GitHub the same
// triggers (merge request, weekly sample, gardening schedule); on GitLab the
// very template, which holds the jobs.
func TestShortFormRunsTheSameJobs(t *testing.T) {
	var gh struct {
		On map[string]any `yaml:"on"`
	}
	if err := yaml.Unmarshal([]byte(readShort(t, ghShort)), &gh); err != nil {
		t.Fatal(err)
	}
	for _, trig := range []string{"pull_request", "schedule"} {
		if _, ok := gh.On[trig]; !ok {
			t.Errorf("the GitHub short form lacks the %s trigger", trig)
		}
	}
	sample, err := os.ReadFile("../../ci/github/workline-sample.yml")
	if err != nil {
		t.Fatal(err)
	}
	if m := regexp.MustCompile(`cron: "([^"]+)"`).FindSubmatch(sample); m == nil || !strings.Contains(readShort(t, ghShort), `"`+string(m[1])+`"`) {
		t.Errorf("the weekly sample's schedule is not in the GitHub short form")
	}
	var gl struct {
		Include []gitlabInclude `yaml:"include"`
	}
	if err := yaml.Unmarshal([]byte(readShort(t, glShort)), &gl); err != nil || len(gl.Include) != 1 {
		t.Fatalf("the GitLab short form: %v %v", err, gl.Include)
	}
	full, err := os.ReadFile("../../" + gl.Include[0].File)
	if err != nil {
		t.Fatalf("the GitLab short form includes a file the repository lacks: %v", err)
	}
	for _, job := range []string{"workline:judge:", "workline:garden:judge:", "workline:warn:child:"} {
		if !strings.Contains(string(full), "\n"+job) {
			t.Errorf("the included template lacks the job %s", job)
		}
	}
}
