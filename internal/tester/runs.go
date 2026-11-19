package tester

import (
	"time"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// RunsWithin is how long the tester waits, before its agent, for the test
// command to show it runs on the base, and doctor for it on HEAD: a setup
// fault (an SDK not found, a tool missing) ends a run in seconds (#1333).
const RunsWithin = 60 * time.Second

// commandRuns is Runs on rev for the doctor, the settings merged read
// first, within RunsWithin (contract.Tests).
func commandRuns(repo, rev string, merged map[string]any) (string, *verdict.Finding, error) {
	s, err := SettingsFrom(merged)
	if err != nil {
		return "", nil, err
	}
	return s.Command, Runs(repo, rev, s, RunsWithin), nil
}

// Runs runs the test command on rev, apart, for at most within, and says
// whether it runs: nil when it wrote its report, red tests or not, or was
// still running at the end, stopped then, or exited 0 with no report (a
// project with no test yet: Gradle's, writing its first); else the finding of why not,
// its error lines quoted. No test is proven by it: it costs at most
// within, where an agent's call costs tokens first (#1333).
func Runs(repo, rev string, s Settings, within time.Duration) *verdict.Finding {
	if v := unset(s); v != nil {
		return &v.Findings[0]
	}
	if stop := time.Now().Add(within); s.deadline.IsZero() || stop.Before(s.deadline) {
		s.deadline = stop
	}
	r, f := run(repo, rev, nil, "", s, "on "+short(rev))
	if f == nil || f.Rule == "proof-over-time" || f.Rule == "report-missing" && r.failure == exitedZero {
		return nil
	}
	if f.Rule == "report-missing" {
		f.Rule = "test-command-fails"
	}
	return f
}

// settingsFor is SettingsFrom for another role (contract.Tests): its
// command and suite's seconds, what was read kept on an error.
func settingsFor(merged map[string]any) (contract.TestSettings, error) {
	s, err := SettingsFrom(merged)
	return contract.TestSettings{Command: s.Command, SuiteMaxSeconds: s.SuiteMaxSeconds, Merged: merged}, err
}

// onChangeFor is OnChange for another role (contract.Tests), its
// settings read again from the merged ones settingsFor read.
func onChangeFor(repo, rev string, patches []intent.Intention, ts contract.TestSettings, deadline time.Time, scope contract.TestScope) (contract.TestOutcome, *verdict.Finding) {
	s, err := SettingsFrom(ts.Merged)
	if err != nil {
		return contract.TestOutcome{}, &verdict.Finding{Rule: "settings-unread", Where: "tester", Message: err.Error()}
	}
	return OnChange(repo, rev, patches, s, deadline, scope)
}
