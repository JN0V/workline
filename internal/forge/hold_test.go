package forge

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// main's tip as GitHub files it: the runs a push started count, with
// other apps' checks and the commit statuses; a schedule's, an issue's
// or a workflow_run's suite, filed under the default branch's tip, never
// does, nor workline's own status, nor a suite with no run (ADR-0051).
func TestGitHubTip(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in\n*graphql*) printf '%s\\n' \"$TIP\" ;;\n*) echo main ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	suite := func(event, name, status, conclusion string) string {
		run := `null`
		if event != "" {
			run = `{"event":"` + event + `"}`
		}
		return `{"workflowRun":` + run + `,"checkRuns":{"nodes":[{"__typename":"CheckRun","name":"` + name + `","status":"` + status + `","conclusion":"` + conclusion + `"}]}}`
	}
	tip := func(suites, statuses string) string {
		return `{"data":{"repository":{"ref":{"target":{"oid":"abc1234def","checkSuites":{"nodes":[` + suites +
			`,{"workflowRun":null,"checkRuns":{"nodes":[]}}]},"status":{"contexts":[` + statuses + `]}}}}}}`
	}
	for _, c := range []struct {
		name, tip, state string
		failing, pending []string
	}{
		{"green", tip(suite("push", "test", "COMPLETED", "SUCCESS"), ``), "success", nil, nil},
		{"a schedule's red run left out", tip(suite("push", "test", "COMPLETED", "SUCCESS")+","+suite("schedule", "gardening", "COMPLETED", "FAILURE")+","+suite("issues", "again", "IN_PROGRESS", ""), ``), "success", nil, nil},
		{"a push's red run", tip(suite("push", "test", "COMPLETED", "FAILURE")+","+suite("push", "workline", "COMPLETED", "SUCCESS"), ``), "failure", []string{"test"}, nil},
		{"another app's run, not done", tip(suite("", "ci/external", "IN_PROGRESS", ""), ``), "pending", nil, []string{"ci/external"}},
		{"a status red, workline's own left out", tip(suite("push", "test", "COMPLETED", "SUCCESS"), `{"__typename":"StatusContext","context":"ci/legacy","state":"FAILURE"},{"__typename":"StatusContext","context":"workline/accepted","state":"PENDING"}`), "failure", []string{"ci/legacy"}, nil},
		{"none", tip(suite("workflow_run", "workline apply", "COMPLETED", "SUCCESS"), ``), "", nil, nil},
	} {
		t.Setenv("TIP", c.tip)
		got, err := g.Tip("")
		if err != nil || got.State != c.state || got.Branch != "main" || got.Commit != "abc1234def" ||
			!slices.Equal(got.Failing, c.failing) || !slices.Equal(got.Pending, c.pending) {
			t.Errorf("%s: %+v, %v; want %s, failing %v, pending %v", c.name, got, err, c.state, c.failing, c.pending)
		}
	}
}
