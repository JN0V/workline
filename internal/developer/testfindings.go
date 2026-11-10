package developer

// A test finding goes to the tester (#1265, ADR-0066): a tests-lens
// finding on a line of code is sent to the tester's Draft, an objection
// of kind `missing`, with no agent; the tester adds the test or says why
// none is needed. A finding sent is never asked again; code findings
// stay the developer's; the rounds never run out on test findings
// (roles/developer/docs/review-loop.md).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/tester"
)

// testsLens is the reviewer's lens whose findings on code are the
// tester's.
const testsLens = "tests"

// testFinding says whether a is a reviewer's finding of its tests lens,
// a row that blocks or warns, with no finding of another lens grouped
// under it on its line: such a row holds code to fix, the developer's.
func testFinding(a Asked) bool {
	if a.Lens != testsLens || (a.Holds != "blocks" && a.Holds != "warns") {
		return false
	}
	return !slices.ContainsFunc(alsoLenses(a.Text), func(l string) bool { return l != testsLens })
}

// alsoWords open, in a reviewer's row, the findings grouped under the
// one leading its line, each "<lens>: <title>", apart by "; "
// (internal/builtin/reviewer, message).
const alsoWords = " (also found on this line — "

// alsoLenses are the lenses of the findings a row's text groups under
// the one leading it; none for a row of one finding.
func alsoLenses(text string) []string {
	_, rest, ok := strings.Cut(text, alsoWords)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range strings.Split(rest, "; ") {
		if lens, _, ok := strings.Cut(item, ": "); ok && lens != "" && !strings.ContainsAny(lens, " ()") {
			out = append(out, lens)
		}
	}
	return out
}

// testerTests are the issue's frozen tests on its open Draft, sorted:
// where a test finding goes; none without a Draft or its head.
func (b Bounds) testerTests() []string {
	if b.Draft == 0 || b.Head == "" {
		return nil
	}
	var out []string
	for f, n := range b.Frozen {
		if n == b.Issue {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

// splitTests takes out of asked the test findings that are the tester's:
// a tests-lens row on a line of code, not of a frozen test (frozenNote's)
// nor of another test (the developer's own), with the tester's Draft
// open. One a resumed round asks again from its record stays the
// developer's: it was asked it already.
func splitTests(asked []Asked, b Bounds) (mine, theirs []Asked) {
	tests := b.tests()
	open := len(b.testerTests()) > 0
	for _, x := range asked {
		file, _, _ := strings.Cut(x.Where, ":")
		_, frozen := b.Frozen[file]
		if !testFinding(x) || x.Waited || !open || file == "" || frozen || tests.Test(file) {
			mine = append(mine, x)
			continue
		}
		theirs = append(theirs, x)
	}
	return mine, theirs
}

// testRequests is the comment that sends l to the tester's Draft: an
// objection of kind `missing` a finding, on the last line of the issue's
// first frozen test, its own key so that two are two threads.
func testRequests(repo string, b Bounds, mr int, l []Asked) intent.Intention {
	file := b.testerTests()[0]
	line := 1
	if data, err := readAt(repo, b.Head, file); err == nil {
		line = max(1, strings.Count(strings.TrimRight(string(data), "\n"), "\n")+1)
	}
	var comments []forge.LineComment
	var keys []string
	for _, x := range l {
		keys = append(keys, x.Key)
		why := fmt.Sprintf("the reviewer's tests lens, on pull request %d, `%s`: %s", mr, x.Where, x.Text)
		if x.Detail != "" {
			why += "\n\n" + x.Detail
		}
		sum := sha256.Sum256([]byte(x.Key))
		comments = append(comments, forge.LineComment{Path: file, Line: line, Key: "missing-" + hex.EncodeToString(sum[:6]),
			Body: fmt.Sprintf("%s\n- test: %s\n- kind: missing\n- why: %s", tester.ObjectionWord, file, why)})
	}
	batch := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return intent.Intention{Kind: "comment", Value: map[string]any{
		"merge-request": b.Draft, "sticky": "test-findings-" + hex.EncodeToString(batch[:4]), // a send its own comment, an earlier one kept
		"body": fmt.Sprintf("**The developer** sends the review's test findings on pull request %d to the tester, #%d's: it adds each test, or says why none is needed (#1265).\n\n%s\n%s\n%s",
			mr, b.Issue, forge.ReviewHere, forge.Unplaced(comments, "the engine that applied this run posts no comment on a line").Section(), forge.ReviewEnd),
		"review": map[string]any{"head": b.Head, "body": "", "comments": comments}}}
}

// testerLines are the test findings a round sends, as its task, its
// comment and its record say them.
func testerLines(l []Asked) []string {
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = fmt.Sprintf("%s %s `%s`: %s", x.Holds, x.Lens, x.Where, x.Text)
	}
	return out
}

// sendTests marks l sent in a's record.
func sendTests(a *Act, l []Asked) {
	for _, x := range l {
		if !slices.Contains(a.Tester, x.Key) {
			a.Tester = append(a.Tester, x.Key)
		}
	}
}

// toTesterOnly is what a review asking test findings alone writes, no
// agent and no round counted: the findings on the tester's Draft, the
// review read, the round's comment and the record.
func toTesterOnly(r round, b Bounds, repo string) []intent.Intention {
	a := r.Record
	sendTests(&a, r.ToTester)
	a.ReviewRuns = max(a.ReviewRuns, r.Runs)
	if r.ReviewAt != "" {
		a.ReviewAt = r.ReviewAt
	}
	if r.GatesAt != "" {
		a.GatesAt = r.GatesAt
	}
	if r.ChecksAt != "" {
		a.ChecksAt = r.ChecksAt
	}
	var body strings.Builder
	fmt.Fprintf(&body, "**The developer**: the review asks test findings alone, the tester's: sent to its Draft !%d, no round counted. "+
		"The tester adds each test or says why none is needed; a test it adds is taken by the next round.\n\nSent to the tester:\n\n", b.Draft)
	for _, t := range testerLines(r.ToTester) {
		fmt.Fprintf(&body, "- %s\n", t)
	}
	return []intent.Intention{
		testRequests(repo, b, r.MR, r.ToTester),
		{Kind: "comment", Value: map[string]any{"sticky": "round", "body": body.String()}},
		{Kind: "comment", Value: map[string]any{"sticky": RecordKey, "body": recordBody(a)}},
	}
}

// noneNeededWords read the tester's answer on the reviewed pull request
// that no test is needed, and the line the review found it on
// (tester.toReview).
var noneNeededWords = regexp.MustCompile("\\*\\*The tester\\*\\* answers a test finding of this review on `([^`]+)`, sent to its Draft \\(\\[the thread\\]\\([^)]*\\)\\): \\*\\*None needed\\*\\*")

// disputed are the review's rows sent to the tester that still block the
// merge, the tester answering on the pull request that no test is needed
// on their line: the reviewer and the tester do not agree, and nothing
// the developer may do settles it.
func disputed(rows []Asked, a Act, kept forge.Kept) []Asked {
	needless := map[string]bool{}
	for _, c := range kept.Each("<!-- workline:sticky=tester/" + tester.TesterAnswerKey) {
		if m := noneNeededWords.FindStringSubmatch(c); m != nil {
			needless[m[1]] = true
		}
	}
	var out []Asked
	for _, x := range rows {
		if x.Holds == "blocks" && slices.Contains(a.Tester, x.Key) && needless[x.Where] {
			out = append(out, x)
		}
	}
	return out
}

// blockedMove are the labels its issue takes, blocked: the line's, or
// workline's own with no line.
func blockedMove(repo string) (add, remove []string) {
	add, remove = []string{backlog.LabelBlocked}, []string{backlog.LabelReady}
	if line, err := routing.Load(repo); err == nil {
		if ad, rm, err := line.Move("blocked"); err == nil {
			add, remove = ad, rm
		}
	}
	return add, remove
}

// disputedOff is what a review blocking on test findings the tester
// holds needless writes, no agent and no round counted: its issue
// blocked, the reason on it and on the pull request, for a person who
// reads both; no round runs after.
func disputedOff(r round, repo string) []intent.Intention {
	a := r.Record
	a.Handed = true
	a.ReviewRuns = max(a.ReviewRuns, r.Runs)
	if r.ReviewAt != "" {
		a.ReviewAt = r.ReviewAt
	}
	add, remove := blockedMove(repo)
	n := len(r.Disputed)
	why := fmt.Sprintf("the review of merge request %d still blocks on %d test finding%s the tester answered no test is needed for: the reviewer and the tester do not agree",
		r.MR, n, plural(n))
	return []intent.Intention{
		{Kind: "label", Value: map[string]any{"issue": a.Issue, "add": add, "remove": remove, "card": "blocked"}},
		sayOn(a.Issue, backlog.Said(who, "blocks this issue", why,
			"a person on the loop reads the finding and the tester's answer on the pull request, settles it, and moves the issue back by its label")),
		{Kind: "comment", Value: map[string]any{"sticky": "round", "body": fmt.Sprintf(
			"**Blocked**: the review still blocks on %d test finding%s the tester answered no test is needed for. #%d is blocked, its reason on it: a person of the project reads both and settles it.\n\n%s",
			n, plural(n), a.Issue, askedList(r.Disputed))}},
		{Kind: "comment", Value: map[string]any{"sticky": RecordKey, "body": recordBody(a)}},
	}
}
