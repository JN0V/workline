// Package devmark is what the developer leaves on the forge and in its
// run that the line, the engine and the sample read: its branches, its
// record's key and parts, its merge's words and marks, its proof and its
// results (ADR-0070 §6 e). The developer writes with the same words.
package devmark

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Branch opens its branches' names: workline/developer/<issue>.
const Branch = "workline/developer/"

// BranchIssue is the issue a developer's branch is for, 0 for another's.
func BranchIssue(branch string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(branch, Branch))
	if err != nil || !strings.HasPrefix(branch, Branch) {
		return 0
	}
	return n
}

// RecordKey is the sticky key of the record on its pull request.
const RecordKey = "code"

// ErrRecordUnread: the developer's record on its merge request, workline's
// own comment, was not read; on GitHub with an App, WORKLINE_APP_ID names
// the account its comments are trusted from.
var ErrRecordUnread = errors.New("the developer's record on it was not read as workline's own comment (with a GitHub App, WORKLINE_APP_ID names it)")

// The kinds of the issue a developer's pull request closes.
const (
	KindFix     = "fix"
	KindTask    = "task"
	KindFeature = "feature"
)

// Unreproduced marks the merge policy's word on a fix whose bug no test
// red on the base showed: it waits for a person's yes (ADR-0060).
const Unreproduced = "unreproduced"

// Unreviewed marks the merge policy's word on the developer's code no
// reviewer's verdict reads at its head: it waits for a person's yes, or
// a review of its head (ADR-0065, change 2).
const Unreviewed = "unreviewed"

// RefusedKey is the sticky `follow` says a refused merge in, on the
// request (#1354): a conflict's carries the record a merge round reads.
const RefusedKey = "line/merge-refused"

// ConflictMark is the record of a conflict of head with base, the tip of
// its base branch, the next merge round reads.
func ConflictMark(head, base string) string {
	return fmt.Sprintf("<!-- workline:conflict head=%s base=%s -->", head, base)
}

// MergeBundle is the run's git bundle of the settled merge's objects,
// under its directory: in CI the apply starts from another clone, which
// gets the run directory only (#1704).
const MergeBundle = "out/merge.bundle"

// The proof's results.
const (
	Green    = "green"
	Red      = "red"
	Unproven = "unproven" // the test command did not run whole
)
