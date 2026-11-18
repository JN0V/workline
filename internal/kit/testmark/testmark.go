// Package testmark is what the tester leaves on the forge that the line,
// the engine and other roles read: its branches, its subjects, its
// objections' threads, its refusal, its proof (ADR-0070 §6 e). The tester
// writes with the same words.
package testmark

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// The tester's branches: an issue's tests, one an issue
// (workline/tester/issue-<n>), a folder curated.
const (
	TestsBranch  = "workline/tester/issue-"
	CurateBranch = "workline/tester/" + CurateKey
)

// TestsSubject and CurateSubject open the subjects of the tester's
// commits of its tests and of a removal: its merge request's title.
const (
	TestsSubject  = "test: the acceptance tests of #"
	CurateSubject = "test: curate "
)

// IsAnswered says whether a comment is the tester's answer to an objection.
func IsAnswered(body string) bool { return strings.Contains(body, AnswerMark) }

// ObjectionWord opens an objection's first comment.
const ObjectionWord = "workline:objection"

// The tester's own comments in a thread carry AnswerMark, a person's turn
// PersonMark, an objection it could not read UnreadMark (the builder's
// turn, not an answer counted): workline's own account may write for
// other roles too.
var (
	AnswerMark = forge.Marker("tester-answer")
	PersonMark = forge.Marker("tester-person-decides") // before ADR-0059: read, never written
	// SettleMark: past the tester's answers, the product owner settles
	// (ADR-0059, decision 6); BlockedMark: past it, the issue Blocked.
	SettleMark  = forge.Marker("tester-product-owner-settles")
	BlockedMark = forge.Marker("tester-objection-blocked")
	UnreadMark  = forge.Marker("tester-unreadable")
)

// ObjectionsWait says whether one of a Draft's conversations on a line is
// an objection the tester has not answered, settled nor blocked (#1265):
// what wakes its answer run where no forge's job does, the local forge's
// reconcile. The run reads them again, who wrote each among them.
func ObjectionsWait(threads []forge.Thread) bool {
	return slices.ContainsFunc(threads, ObjectionWaits)
}

// ObjectionWaits says whether one conversation on a line is an objection
// the tester has not answered, settled nor blocked.
func ObjectionWaits(t forge.Thread) bool {
	if t.Resolved || len(t.Comments) == 0 || (!strings.HasPrefix(strings.TrimSpace(t.Comments[0].Body), ObjectionWord) && !PersonsThread(t)) {
		return false
	}
	last := t.Comments[len(t.Comments)-1].Body
	return !slices.ContainsFunc([]string{AnswerMark, UnreadMark, SettleMark, PersonMark, BlockedMark}, func(m string) bool { return strings.Contains(last, m) })
}

// PersonsThread says whether a conversation on a line was opened by a
// person of the project (#1358): a review comment on the Draft, no
// objection's form, no engine's marker. The tester reads it as a test
// missing, naming no scenario.
func PersonsThread(t forge.Thread) bool {
	if len(t.Comments) == 0 {
		return false
	}
	c := t.Comments[0]
	body := strings.TrimSpace(c.Body)
	return c.Insider && !c.Bot && body != "" && !strings.HasPrefix(body, ObjectionWord) && !strings.Contains(body, "<!-- workline:")
}

// CurateKey opens the key of a curator's merge request, one a folder
// (workline/tester/curate-<folder>).
const CurateKey = "curate-"

// What the tester says, in its one comment on an issue, when it writes
// no tests for it: before asking the agent, or after.
const (
	RefusedPre = "writes no tests for this issue yet"
	RefusedRun = "wrote no tests for this issue this run"
)

// Refused is why the tester's last word on an issue refused it, read from
// its one comment there; "" when that word is another, or there is none.
// An issue waiting for its tests that the tester refused is blocked by
// the line, never left as acted (#794).
func Refused(comments []string) string {
	sticky := forge.Marker("sticky=tester/issue")
	for _, c := range slices.Backward(comments) {
		if !strings.Contains(c, sticky) {
			continue
		}
		if !strings.Contains(c, "**The tester** "+RefusedRun) && !strings.Contains(c, "**The tester** "+RefusedPre) {
			return ""
		}
		_, why, ok := strings.Cut(c, "- **Why**: ")
		if why, _, _ = strings.Cut(why, "\n"); !ok || strings.TrimSpace(why) == "" {
			return "the tester wrote no tests for it"
		}
		return strings.TrimSuffix(strings.TrimSpace(why), ".")
	}
	return ""
}

// ReadProof is the kind an issue is proved by in place of a Draft: the
// tester's, or the product owner's after "no test possible".
func ReadProof(comments []string) (backlog.Proof, bool) {
	return backlog.ReadProof(comments, "product-owner")
}

// Untested says whether a ready issue still waits for its tests (#788):
// labelled workline:ready from before the line of states, its Validation
// naming scenarios, no Draft of its tests open (drafts: the tester's open
// branches), no proof kind recorded, not claimed nor blocked. The product
// owner's ready, keeping it, sends it to its tests.
func Untested(is forge.Issue, comments, drafts []string) bool {
	if !slices.Contains(is.Labels, backlog.LabelReady) || slices.Contains(is.Labels, backlog.LabelNeedsTests) ||
		slices.Contains(is.Labels, backlog.LabelBlocked) || backlog.Parked(is) || slices.Contains(is.Labels, "workline:developing") ||
		slices.Contains(drafts, TestsBranch+fmt.Sprint(is.ID)) {
		return false
	}
	if _, ok := ReadProof(comments); ok {
		return false
	}
	return len(work.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) > 0
}
