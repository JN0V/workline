package tick

import (
	"slices"
	"testing"
)

// TestDueFailedNightlyListsBacklog: Given a `workline gardening nightly`
// run that failed an hour ago and Ready below its target, When the tick
// runs, Then backlog is listed: a failed run kept no backlog.
func TestDueFailedNightlyListsBacklog(t *testing.T) {
	po := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
	failed := run("workline gardening nightly", "workline-gardening.yml", "completed", "2026-11-16T01:20:00Z")
	failed.Success = false
	runs := append(slices.Clone(done), failed)
	r := Due(setUp(t, po, 6, runs, ""))
	if got := jobsOf(r); !slices.Contains(got, "backlog") {
		t.Errorf("jobs %v, want backlog listed; said %v, unread %v", got, r.Said, r.Unread)
	}
}

// TestDueDocsAndBacklogRunKeepsBacklog: Given a project on the installed
// template, a `workline gardening docs and backlog` run that succeeded an
// hour ago (the full Sunday run, the product owner too) and Ready below
// its target, When the tick runs, Then backlog is not listed.
func TestDueDocsAndBacklogRunKeepsBacklog(t *testing.T) {
	po := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
	sunday := run("workline gardening docs and backlog", "workline-gardening.yml", "completed", "2026-11-16T01:20:00Z")
	sunday.Success = true
	runs := append(slices.Clone(done), sunday)
	r := Due(setUp(t, po, 6, runs, ""))
	if got := jobsOf(r); slices.Contains(got, "backlog") {
		t.Errorf("jobs %v, want no backlog; said %v, unread %v", got, r.Said, r.Unread)
	}
}
