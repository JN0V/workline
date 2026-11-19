package conformance

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// dryRunScriptLines flattens a job's script: GitLab takes a list nested in
// the list (a YAML anchor's) as its lines, in order.
func dryRunScriptLines(v any) []string {
	var out []string
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			out = append(out, dryRunScriptLines(e)...)
		}
	case nil:
	default:
		out = append(out, fmt.Sprint(x))
	}
	return out
}

// The dry run holds no token
// Given a GitLab project whose `workline` environment sets GITLAB_TOKEN,
// When a schedule runs `workline:release:dry-run`,
// Then the release's gates run with GITLAB_TOKEN unset: the job's `unset`
// line names it, before `workline route release --dry-run`.
func TestGitLabReleaseDryRunUnsetsGitlabToken(t *testing.T) {
	data, err := os.ReadFile("../../defaults/ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	job, ok := top["workline:release:dry-run"].(map[string]any)
	if !ok {
		t.Fatal("no job workline:release:dry-run in the GitLab template")
	}
	lines := dryRunScriptLines(job["script"])
	unsetAt, routeAt := -1, -1
	var unsetLine string
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "unset ") && unsetAt < 0 {
			unsetAt, unsetLine = i, l
		}
		if strings.Contains(l, "workline route release --dry-run") && routeAt < 0 {
			routeAt = i
		}
	}
	if routeAt < 0 {
		t.Fatalf("workline:release:dry-run: its script does not run `workline route release --dry-run`: %v", lines)
	}
	if unsetAt < 0 || unsetAt > routeAt {
		t.Fatalf("workline:release:dry-run: no unset line before `workline route release --dry-run`: %v", lines)
	}
	names := map[string]bool{}
	for _, w := range strings.Fields(unsetLine) {
		names[w] = true
	}
	for _, want := range []string{"CLAUDE_CODE_OAUTH_TOKEN", "WORKLINE_GITLAB_TOKEN", "GITLAB_TOKEN"} {
		if !names[want] {
			t.Errorf("workline:release:dry-run: its unset line %q does not name %s: the release's gates run with it", unsetLine, want)
		}
	}
}

// The doc says it
// Given docs/gitlab/environment.md,
// When a person reads which jobs unset the tokens,
// Then the release's dry run is listed with the jobs that unset all three:
// the doc no longer says it keeps GITLAB_TOKEN.
func TestGitLabEnvironmentDocListsDryRunAmongUnsetters(t *testing.T) {
	data, err := os.ReadFile("../../docs/gitlab/environment.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if strings.Contains(doc, "not `GITLAB_TOKEN`") {
		t.Errorf("docs/gitlab/environment.md: still says the release's dry run keeps GITLAB_TOKEN (\"not `GITLAB_TOKEN`\")")
	}
	listed := false
	for _, para := range strings.Split(doc, "\n\n") {
		if strings.Contains(para, "dry run") && strings.Contains(para, "GITLAB_TOKEN") {
			listed = true
		}
	}
	if !listed {
		t.Errorf("docs/gitlab/environment.md: the release's dry run is not listed with GITLAB_TOKEN among the tokens its jobs unset")
	}
}
