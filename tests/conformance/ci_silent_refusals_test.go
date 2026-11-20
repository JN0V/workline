package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// silentStep is a step of a GitHub-style workflow, as the static test reads it.
type silentStep struct {
	Name string         `yaml:"name"`
	If   string         `yaml:"if"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

// silentErrorLine is the line a Verdict step prints before it fails.
const silentErrorLine = "::error title=workline::"

// silentGitHubFiles are the role workflows the done criterion lists, as
// the template holds them.
var silentGitHubFiles = []string{"workline.yml", "workline-bounds.yml", "workline-warn.yml",
	"workline-develop.yml", "workline-gardening.yml", "workline-answer.yml"}

// silentTemplateDir holds the GitHub templates. The workflows of
// workline's own CI are read-only for the change, so the test holds the
// templates, not them.
const silentTemplateDir = "../../defaults/ci/github"

// silentForgejoDir holds the Forgejo samples, held to the GitHub-style rules.
const silentForgejoDir = "../../defaults/ci/forgejo"

// silentForgejoFiles are the workflow files of the Forgejo samples.
func silentForgejoFiles() []string {
	var files []string
	for _, ext := range []string{"*.yml", "*.yaml"} {
		got, _ := filepath.Glob(silentForgejoDir + "/" + ext)
		files = append(files, got...)
	}
	return files
}

// silentJobs reads the steps of each job of a workflow file.
func silentJobs(t *testing.T, f string) map[string][]silentStep {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []silentStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	jobs := map[string][]silentStep{}
	for name, j := range wf.Jobs {
		jobs[name] = j.Steps
	}
	return jobs
}

// silentGitHubGaps reads one workflow file as text and YAML, and says, for
// each job that has a Verdict step, what it lacks: the reason said as an
// error line, the result file and the run folder kept on a failure. sig
// is the way each such job does both, to compare two copies.
func silentGitHubGaps(t *testing.T, f string) (said, kept []string, sig map[string]string) {
	t.Helper()
	jobs := silentJobs(t, f)
	names := make([]string, 0, len(jobs))
	for n := range jobs {
		names = append(names, n)
	}
	sort.Strings(names)
	sig = map[string]string{}
	for _, name := range names {
		var verdict *silentStep
		all := ""
		for i := range jobs[name] {
			s := &jobs[name][i]
			all += s.Run + "\n"
			if (s.Name == "Verdict" || s.Name == "Said") && strings.Contains(s.Run, "exit") {
				verdict = s
			}
		}
		if verdict == nil {
			continue
		}
		e := strings.Index(verdict.Run, silentErrorLine)
		hasError := e >= 0 && e < strings.LastIndex(verdict.Run, "exit")
		if !hasError {
			said = append(said, fmt.Sprintf("%s: job %s: its %s step exits with a status without printing `%s<reason>` first", f, name, verdict.Name, silentErrorLine))
		}
		if !strings.Contains(all, ".summary") {
			said = append(said, fmt.Sprintf("%s: job %s: no step reads the summary (.summary) the error line says", f, name))
		}
		// The result file and the run folder: the uploads whose path holds
		// the runs, else those of a result file; never the tally.
		var runs, results []silentStep
		for _, s := range jobs[name] {
			if !strings.Contains(s.Uses, "upload-artifact") {
				continue
			}
			path, artifact := fmt.Sprint(s.With["path"]), fmt.Sprint(s.With["name"])
			switch {
			case strings.Contains(path, "tally") || strings.Contains(artifact, "tally"):
			case strings.Contains(path, ".workline-runs"):
				runs = append(runs, s)
			case strings.Contains(path, ".json"):
				results = append(results, s)
			}
		}
		if len(runs) == 0 {
			runs = results
		}
		if len(runs) == 0 {
			kept = append(kept, fmt.Sprintf("%s: job %s: no upload keeps its result file nor its run folder", f, name))
		}
		var keptNames []string
		for _, s := range runs {
			if strings.Contains(s.If, "failure()") || strings.Contains(s.If, "always()") {
				keptNames = append(keptNames, fmt.Sprint(s.With["name"]))
				continue
			}
			kept = append(kept, fmt.Sprintf("%s: job %s: the artifact %v is uploaded `if: %s`, not kept when the job fails", f, name, s.With["name"], s.If))
		}
		sort.Strings(keptNames)
		sig[name] = fmt.Sprintf("error=%v kept=%v", hasError, keptNames)
	}
	return said, kept, sig
}

// silentText flattens a script's lines, whatever their nesting.
func silentText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case []any:
		var b strings.Builder
		for _, e := range x {
			b.WriteString(silentText(e) + "\n")
		}
		return b.String()
	}
	return fmt.Sprint(v)
}

// silentResolve is a GitLab job with what it extends under it.
func silentResolve(all map[string]any, name string, depth int) map[string]any {
	out := map[string]any{}
	job, _ := all[name].(map[string]any)
	if depth > 8 {
		return out
	}
	var parents []string
	switch e := job["extends"].(type) {
	case string:
		parents = []string{e}
	case []any:
		for _, p := range e {
			parents = append(parents, fmt.Sprint(p))
		}
	}
	for _, p := range parents {
		for k, v := range silentResolve(all, p, depth+1) {
			out[k] = v
		}
	}
	for k, v := range job {
		out[k] = v
	}
	return out
}

var silentExit = regexp.MustCompile(`exit\s+"?\$`)

// silentGitLabGaps reads the GitLab template, and says, for each job whose
// script exits with a status, whether it prints an ERROR: line first and
// whether its artifacts are kept `when: always`.
func silentGitLabGaps(t *testing.T) (said, kept []string) {
	t.Helper()
	const f = "../../defaults/ci/gitlab/workline.gitlab-ci.yml"
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var all map[string]any
	if err := yaml.Unmarshal(data, &all); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	names := make([]string, 0, len(all))
	for n, v := range all {
		if _, ok := v.(map[string]any); ok && !strings.HasPrefix(n, ".") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		job := silentResolve(all, name, 0)
		script := silentText(job["script"])
		exits := silentExit.FindAllStringIndex(script, -1)
		if len(exits) == 0 {
			continue
		}
		if e := strings.Index(script, "ERROR:"); e < 0 || e > exits[len(exits)-1][0] {
			said = append(said, fmt.Sprintf("%s: job %s: its script exits with a status without printing an `ERROR:` line first", f, name))
		}
		if art, _ := job["artifacts"].(map[string]any); fmt.Sprint(art["when"]) != "always" {
			kept = append(kept, fmt.Sprintf("%s: job %s: its artifacts are not kept `when: always`", f, name))
		}
	}
	return said, kept
}

// silentAllGitHubStyle lists every GitHub-style file the done criterion
// holds: the GitHub templates and the Forgejo samples.
func silentAllGitHubStyle() []string {
	var files []string
	for _, f := range silentGitHubFiles {
		files = append(files, filepath.Join(silentTemplateDir, f))
	}
	return append(files, silentForgejoFiles()...)
}

// Said in the log
// Given a CI run of any role that ends with a status not 0,
// When its Verdict step ends,
// Then the log shows the summary as an error line: on GitHub and Forgejo
// the Verdict step prints `::error title=workline::<reason>` before it
// exits, on GitLab the job's script prints an `ERROR:` line before it exits.
func TestSilentRefusalsSaidInTheLog(t *testing.T) {
	var gaps []string
	for _, f := range silentAllGitHubStyle() {
		said, _, _ := silentGitHubGaps(t, f)
		gaps = append(gaps, said...)
	}
	said, _ := silentGitLabGaps(t)
	gaps = append(gaps, said...)
	for _, g := range gaps {
		t.Error(g)
	}
}

// Kept when it fails
// Given the same run,
// When it blocked before it left a pending step,
// Then its result file and run folder are kept as an artifact: on GitHub
// and Forgejo the upload has `failure()` or `always()` in its condition, on
// GitLab the job's artifacts are kept `when: always`.
func TestSilentRefusalsKeptWhenItFails(t *testing.T) {
	var gaps []string
	for _, f := range silentAllGitHubStyle() {
		_, kept, _ := silentGitHubGaps(t, f)
		gaps = append(gaps, kept...)
	}
	_, kept := silentGitLabGaps(t)
	gaps = append(gaps, kept...)
	for _, g := range gaps {
		t.Error(g)
	}
}

// Both copies
// Given a workflow and its template,
// When either is read,
// Then both say the reason and keep the files the same way: the GitHub
// template and the Forgejo sample of the same file, where there is one.
func TestSilentRefusalsBothCopies(t *testing.T) {
	for _, f := range silentGitHubFiles {
		var sigs []map[string]string
		for _, dir := range []string{silentTemplateDir, silentForgejoDir} {
			path := filepath.Join(dir, f)
			if _, err := os.Stat(path); err != nil && dir == silentForgejoDir {
				continue
			}
			said, kept, sig := silentGitHubGaps(t, path)
			for _, g := range append(said, kept...) {
				t.Error(g)
			}
			sigs = append(sigs, sig)
		}
		if len(sigs) < 2 {
			continue
		}
		for name, own := range sigs[0] {
			if other, ok := sigs[1][name]; ok && !reflect.DeepEqual(own, other) {
				t.Errorf("%s: job %s: the GitHub template says and keeps `%s`, the Forgejo sample `%s`", f, name, own, other)
			}
		}
	}
}
