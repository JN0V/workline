package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/defaults/ci"
	"github.com/JN0V/workline/internal/routing"
)

// pinLine is a CI file's pin of the engine, as the templates write it.
var pinLine = regexp.MustCompile(`(?m)^\s*WORKLINE_VERSION:\s*["']?(v\d+\.\d+\.\d+)["']?`)

// customised is the comment line with which a person says a copy differs from
// its template on purpose: doctor then does not compare its content.
var customised = regexp.MustCompile(`(?m)^\s*#\s*workline:customised\b`)

// release reads a release's vMAJOR.MINOR.PATCH, nil for anything else:
// "(devel)", a pseudo-version, a release candidate.
func release(v string) []int {
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if !strings.HasPrefix(v, "v") || len(parts) != 3 {
		return nil
	}
	out := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || p != strconv.Itoa(n) {
			return nil
		}
		out[i] = n
	}
	return out
}

// eventTemplates is the GitHub template that runs each event workline routes
// in CI (defaults/ci/github): the events a person's machine runs are not here.
var eventTemplates = map[string]string{
	"merge-request": "workline.yml",
	"main":          "workline.yml",
	"schedule":      "workline-gardening.yml",
	"release":       "workline-gardening.yml",
	"issue":         "workline-develop.yml",
}

// ciTemplates says, of a repository whose CI runs workline, each CI file
// pinned to a release older than this engine, and each event routed that
// no GitHub workflow runs, with the template to copy (#1016). A repository
// set up months ago otherwise runs its old roles, and the new ones never.
func ciTemplates(r *Report, o Options, line *routing.Config) {
	github, _ := filepath.Glob(filepath.Join(o.Repo, ".github", "workflows", "*.y*ml"))
	slices.Sort(github)
	files := append(slices.Clone(github), filepath.Join(o.Repo, ".gitlab-ci.yml"))
	engine := release(o.Engine)
	pinned := false
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		m := pinLine.FindSubmatch(data)
		if m == nil {
			continue
		}
		pinned = true
		pin := release(string(m[1]))
		if engine == nil || pin == nil || slices.Compare(pin, engine) > 0 {
			continue
		}
		rel, _ := filepath.Rel(o.Repo, f)
		if slices.Compare(pin, engine) == 0 {
			if tpl, err := ci.Templates.ReadFile(strings.TrimPrefix(templateOf(rel), "defaults/ci/")); err == nil && !customised.Match(data) && withoutPin(tpl) != withoutPin(data) {
				r.add(Check{Area: "repository", Rule: "workflow-behind", Where: rel, Level: Warn,
					Message: fmt.Sprintf("%s differs from the template of workline %s beyond its pin: it was re-pinned, not re-copied, and runs what that template has since changed", rel, o.Engine),
					Fix:     fmt.Sprintf("re-copy %s to %s, then reapply your changes (%s); if the difference is on purpose, add a `# workline:customised` comment line to %s to silence this", templateOf(rel), rel, templateURL(o.Engine, templateOf(rel)), rel)})
			}
			continue
		}
		r.add(Check{Area: "repository", Rule: "workflow-outdated", Where: rel, Level: Warn,
			Message: fmt.Sprintf("%s runs workline %s, this engine is %s: its roles run as they were then, and what came since never runs", rel, m[1], o.Engine),
			Fix:     fmt.Sprintf("set WORKLINE_VERSION: %s in it, and compare it with %s", o.Engine, templateURL(o.Engine, templateOf(rel)))})
	}
	if !pinned || len(github) == 0 {
		return // workline runs in no CI here, or not on GitHub's
	}
	events := make([]string, 0, len(line.Events))
	for e, steps := range line.Events {
		if len(steps) > 0 && eventTemplates[e] != "" {
			events = append(events, e)
		}
	}
	slices.Sort(events)
	for _, e := range events {
		if ciRuns(o.Repo, "workline route "+e) {
			continue
		}
		t := eventTemplates[e]
		r.add(Check{Area: "repository", Rule: "workflow-missing", Where: ".github/workflows/" + t, Level: Warn,
			Message: fmt.Sprintf("`%s` is routed to %s, but no workflow runs `workline route %s`: those steps never run in CI", e, strings.Join(line.Events[e], ", "), e),
			Fix:     "copy " + templateURL(o.Engine, t) + " to .github/workflows/" + t})
	}
}

// withoutPin is a CI file's content, its pin line and line ends set aside.
func withoutPin(data []byte) string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if !pinLine.MatchString(l) {
			out = append(out, strings.TrimRight(l, " \t"))
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// templateOf is the template a workflow was copied from: its own name, a
// GitHub workflow's; workline.gitlab-ci.yml, GitLab's pipeline.
func templateOf(rel string) string {
	if rel == ".gitlab-ci.yml" {
		return "defaults/ci/gitlab/workline.gitlab-ci.yml"
	}
	return "defaults/ci/github/" + filepath.Base(rel)
}

// templateURL is where a template of this engine's release is read; a build
// that is no release, its default branch's.
func templateURL(engine, t string) string {
	if !strings.Contains(t, "/") {
		t = "defaults/ci/github/" + t
	}
	ref := "main"
	if release(engine) != nil {
		ref = engine
	}
	return "https://github.com/JN0V/workline/blob/" + ref + "/" + t
}
