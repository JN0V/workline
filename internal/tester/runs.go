package tester

import (
	"time"

	"github.com/JN0V/workline/internal/verdict"
)

// RunsWithin is how long the tester waits, before its agent, for the test
// command to show it runs on the base, and doctor for it on HEAD: a setup
// fault (an SDK not found, a tool missing) ends a run in seconds (#1333).
const RunsWithin = 60 * time.Second

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
