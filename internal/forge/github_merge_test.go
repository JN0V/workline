package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A head's checks summed up, workline's own statuses left out: a verdict
// kept failing, or a yes not given, is no check of the change (ADR-0042).
func TestSumChecks(t *testing.T) {
	for _, c := range []struct {
		name string
		all  []checkContext
		want string
	}{
		{"none", nil, ""},
		{"green", []checkContext{{Type: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"}, {Type: "StatusContext", Context: "ci", State: "SUCCESS"}}, "success"},
		{"one runs", []checkContext{{Type: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"}, {Type: "CheckRun", Status: "IN_PROGRESS"}}, "pending"},
		{"one failed", []checkContext{{Type: "CheckRun", Status: "IN_PROGRESS"}, {Type: "CheckRun", Status: "COMPLETED", Conclusion: "FAILURE"}}, "failure"},
		{"workline's own left out", []checkContext{{Type: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Type: "StatusContext", Context: VerdictStatus, State: "FAILURE"}, {Type: "StatusContext", Context: Accepted, State: "FAILURE"}}, "success"},
	} {
		if got := sumChecks(c.all); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// The base holds a yes only when it requires workline/accepted from the
// App itself: unpinned, or the App unknown, any token setting statuses
// could give a yes (ADR-0042).
func TestGitHubGuarded(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in\n*rules/branches*) printf '%s\\n' \"$RULES\" ;;\n*) ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	for _, c := range []struct {
		rules, app, want string
	}{
		{`{"context":"workline/accepted","app":"12"}`, "12", ""},
		{`{"context":"workline/accepted","app":""}`, "12", "from any source"},
		{`{"context":"workline/accepted","app":"12"}`, "", "id is not known"},
		{`{"context":"workline/accepted","app":"99"}`, "12", "from another app"},
		{`{"context":"test","app":""}`, "12", "does not require the status workline/accepted"},
	} {
		t.Setenv("RULES", c.rules)
		t.Setenv("WORKLINE_APP_ID", c.app)
		why, err := g.Guarded("main")
		if err != nil || (c.want == "") != (why == "") || !strings.Contains(why, c.want) {
			t.Errorf("rules %s: %q, %v; want %q", c.rules, why, err, c.want)
		}
	}
}

// GitHub turns no auto-merge on for a pull request it could merge now, a
// check it does not require still running ("unstable status", as on
// workline#341): it is merged at once, at the head checked.
func TestGitHubAutoMergeMergesWhatItCouldMergeNow(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	script := "#!/bin/sh\necho \"$*\" >> " + log + "\ncase \"$*\" in\n" +
		"*enablePullRequestAutoMerge*) echo \"gh: Pull request Pull request is in $STATE status\" >&2; exit 1 ;;\n" +
		"*squashMergeAllowed*) printf '%s\\n' '{\"data\":{\"repository\":{\"squashMergeAllowed\":true,\"viewerDefaultMergeMethod\":\"SQUASH\",\"pullRequest\":{\"id\":\"PR_1\"}}}}' ;;\n" +
		"*) ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	for _, state := range []string{"clean", "unstable", "has_hooks"} {
		os.Remove(log)
		t.Setenv("STATE", state)
		merged, err := g.AutoMerge(MergeRequest{ID: 341}, "abc123", true)
		calls, _ := os.ReadFile(log)
		if err != nil || !merged || !strings.Contains(string(calls), "pulls/341/merge") || !strings.Contains(string(calls), "sha=abc123") {
			t.Errorf("%s status: merged %v, %v; calls:\n%s", state, merged, err, calls)
		}
	}
	t.Setenv("STATE", "blocked")
	os.Remove(log)
	if merged, err := g.AutoMerge(MergeRequest{ID: 341}, "abc123", true); merged || err == nil {
		t.Errorf("blocked status: merged %v, %v; want an error, no merge", merged, err)
	}
}
