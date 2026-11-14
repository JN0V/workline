package forge

import (
	"os"
	"strings"
	"testing"
)

// Given a head whose workline/accepted success was set by an app that is
// not workline's, When the wake reads it, Then it is no yes: the query
// asks for the creator by workline's own App login, never any Bot.
func TestGitHubAppStatusAnotherApp(t *testing.T) {
	t.Setenv("WORKLINE_APP", "workline-app")
	log := ghStub(t, "*commits/c0ffee/statuses*) ;;\n")
	yes, err := (&github{repo: t.TempDir()}).AppStatus("c0ffee", Accepted)
	calls, _ := os.ReadFile(log)
	if yes || err != nil || strings.Contains(string(calls), `.creator.type == "Bot"`) ||
		!strings.Contains(string(calls), `.creator.login == "workline-app[bot]"`) {
		t.Errorf("%v, %v; calls:\n%s", yes, err, calls)
	}
}

// Given a head whose workline/accepted success was set by workline's App,
// When the wake reads it, Then it is the yes: the query pins the creator
// to the App's login, and what it finds is a yes.
func TestGitHubAppStatusTheApp(t *testing.T) {
	t.Setenv("WORKLINE_APP", "workline-app")
	log := ghStub(t, "*commits/c0ffee/statuses*creator.login*workline-app*) echo 42 ;;\n")
	yes, err := (&github{repo: t.TempDir()}).AppStatus("c0ffee", Accepted)
	calls, _ := os.ReadFile(log)
	if !yes || err != nil {
		t.Errorf("%v, %v; calls:\n%s", yes, err, calls)
	}
}
