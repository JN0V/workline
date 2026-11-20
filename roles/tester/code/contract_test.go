package tester

import (
	"reflect"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/contract"
)

// The tester joins the shipped registry offering every field of what the
// doctor asks of the project's tests: none left nil.
func TestTheTesterOffersEveryTestsField(t *testing.T) {
	tt, name := contract.Builtins().Tests()
	if tt == nil || name != "tester" {
		t.Fatalf("Tests = %v, %q: the tester offers none", tt, name)
	}
	v := reflect.ValueOf(*tt)
	for i := range v.NumField() {
		if v.Field(i).IsNil() {
			t.Errorf("the tester leaves contract.Tests.%s nil", v.Type().Field(i).Name)
		}
	}
}

// What the doctor asks: the settings read first, an error when they do
// not read; else the command, and whether it runs on rev.
func TestCommandRunsForTheDoctor(t *testing.T) {
	repo, run := gitRepo(t, map[string]string{"a.txt": "a\n"})
	rev := run(repo, "rev-parse", "HEAD")
	if c, f, err := commandRuns(repo, rev, map[string]any{"test-command": "true", "suite-max-seconds": "lots"}); err == nil || c != "" || f != nil {
		t.Errorf("unreadable settings: %q, %v, %v; want an error alone", c, f, err)
	}
	if c, f, err := commandRuns(repo, rev, map[string]any{"test-command": "true", "test-report": "report.xml"}); err != nil || c != "true" || f != nil {
		t.Errorf("a command that runs: %q, %v, %v", c, f, err)
	}
	if c, f, err := commandRuns(repo, rev, map[string]any{"test-command": "echo SDK location not found >&2; exit 1", "test-report": "report.xml"}); err != nil || c == "" || f == nil || f.Rule != "test-command-fails" {
		t.Errorf("a command that fails: %q, %+v, %v", c, f, err)
	}
}

// What the developer's proof asks: the settings as another role reads
// them, what was read kept on an error; a run on settings that do not
// read is a finding, never a run.
func TestTheTestsAsAnotherRoleAsks(t *testing.T) {
	m := map[string]any{"test-command": "go test", "suite-max-seconds": 90}
	if ts, err := settingsFor(m); err != nil || ts.Command != "go test" || ts.SuiteMaxSeconds != 90 {
		t.Errorf("settingsFor = %+v, %v", ts, err)
	}
	bad := map[string]any{"test-command": "go test", "suite-max-seconds": "lots"}
	ts, err := settingsFor(bad)
	if err == nil {
		t.Errorf("unreadable settings read: %+v", ts)
	}
	if _, f := onChangeFor(t.TempDir(), "HEAD", nil, ts, time.Time{}, contract.TestScope{}); f == nil || f.Rule != "settings-unread" {
		t.Errorf("a run on unreadable settings: %+v", f)
	}
}
