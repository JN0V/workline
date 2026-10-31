package tester

// The tester's moment (ADR-0055, step 4): a scheduled run takes one ready
// issue whose tests nobody wrote yet, and writes them as `test write`
// does. Whoever builds next starts from that Draft
// (roles/tester/docs/hand-over.md).

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// taskKey is the task of the Draft holding issue n's tests, its branch
// workline/tester/issue-<n> (roles/tester/docs/hand-over.md): the engine lists the
// open ones by it.
func taskKey(n int) string { return fmt.Sprintf("issue-%d", n) }

// pick is the issue a scheduled run writes the tests of: the first, in
// the backlog's order, waiting for its tests (needs-tests, or ready;
// waiting on no open issue, no parts), whose Validation names scenarios, with no Draft
// of the tester open; 0 when none waits. waiting is the tasks of the
// tester's open Drafts, as the engine lists them.
func pick(f forge.Backlog, waiting []string) (int, error) {
	issues, err := f.Issues()
	if err != nil {
		return 0, err
	}
	open := map[int]bool{}
	for _, is := range issues {
		open[is.ID] = true
	}
	backlog.Order(issues)
	for _, is := range issues {
		if !backlog.Readied(is) || len(backlog.Waiting(is, open)) > 0 || len(backlog.Parts(is)) > 0 || slices.Contains(waiting, taskKey(is.ID)) {
			continue
		}
		if len(Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) > 0 {
			return is.ID, nil
		}
	}
	return 0, nil
}

// scheduled picks the run's issue: its number, or the exit of a run that
// stops here, its verdict final.
func scheduled(runDir, repo string) (id, code int) {
	n, ok := os.LookupEnv("WORKLINE_OPEN_MERGE_REQUESTS")
	if !ok || n == "" {
		return 0, final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no Draft can be opened",
			Findings: []verdict.Finding{{Rule: "no-merge-request", Message: "the scheduled tester opens each issue's tests as a Draft, and skips an issue whose Draft is open: run the schedule with --open-merge-request and a forge"}}})
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return 0, fail(err)
	}
	if f == nil {
		return 0, final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge to read the issues from",
			Findings: []verdict.Finding{{Rule: "no-forge", Message: "the tester reads the ready issues and opens a Draft on a forge; " + forge.Missing}}})
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return 0, final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the forge lists no issues",
			Findings: []verdict.Finding{{Rule: "no-backlog", Message: "the scheduled tester picks a ready issue from the forge's open issues, which this forge does not list: run `workline test write --issue <n>`"}}})
	}
	id, err = pick(b, strings.Fields(os.Getenv("WORKLINE_OPEN_MERGE_REQUEST_TASKS")))
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return 0, exitExternal
	}
	if err != nil {
		return 0, fail(err)
	}
	if id == 0 {
		return 0, final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no ready issue waits for its tests"})
	}
	return id, 0
}
