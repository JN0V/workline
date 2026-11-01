package tester

// A proof kind, not always a new test (ADR-0059, decision 4): for a flaky
// test to make reliable, a refactor or a docs change, the tester records
// the kind of proof and its reason on the issue, and the issue moves on
// with no Draft; the developer accepts it in place of one. "No test
// possible" goes back to the product owner.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// The proof kinds and the record, the backlog's: the product owner picks
// one too, after "no test possible".
const (
	KindNewTests = backlog.KindNewTests
	KindFlaky    = backlog.KindFlaky
	KindSuite    = backlog.KindSuite
	KindDocs     = backlog.KindDocs
	KindNone     = backlog.KindNone
)

// Proof is the tester's record of a kind in place of a Draft.
type Proof = backlog.Proof

// ProofKinds are the kinds a tester records in place of a Draft.
var ProofKinds = backlog.ProofKinds

// TestName is a test's name as a proof-command takes it in `{test}`.
var TestName = backlog.TestName

// proofOf reads the kind a note names, and its reason; false when no note
// names one.
func proofOf(notes []string) (Proof, bool) {
	for _, n := range notes {
		if p, ok := backlog.ProofNote(n); ok {
			return p, true
		}
	}
	return Proof{}, false
}

// ReadProof is the kind an issue is proved by in place of a Draft: the
// tester's, or the product owner's after "no test possible".
func ReadProof(comments []string) (Proof, bool) {
	return backlog.ReadProof(comments, "product-owner")
}

// proved ends a run whose agent wrote no test and named a kind instead:
// recorded on the issue, and the issue moved on — to the state after its
// own for a kind in ProofKinds, back to the product owner's for none.
func proved(runDir, repo string, t task, intents []intent.Intention, p Proof) int {
	where := fmt.Sprintf("#%d", t.Issue)
	if p.Reason == "" {
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "a proof kind with no reason",
			Findings: []verdict.Finding{{Rule: "proof-reason-missing", Where: where, Message: "the note names the kind `" + p.Kind + "` and no reason: each kind says why no new test fits"}}})
	}
	if p.Kind != KindNone && ProofKinds[p.Kind] == "" {
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "an unknown proof kind",
			Findings: []verdict.Finding{{Rule: "proof-kind-unknown", Where: where,
				Message: fmt.Sprintf("`%s` is no proof kind: %s, %s or %s, or %s when no test is possible", p.Kind, KindFlaky, KindSuite, KindDocs, KindNone)}}})
	}
	if p.Kind == KindFlaky && !TestName.MatchString(p.Test) {
		msg := "the note names no test (`test: <name>`): the developer's proof runs it N times"
		if p.Test != "" {
			msg = "the test `" + p.Test + "` is no name a test command takes: letters, digits and `_ . / : -`"
		}
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "a flaky-n-of-n proof with no test to run",
			Findings: []verdict.Finding{{Rule: "proof-test-missing", Where: where, Message: msg}}})
	}
	line, err := routing.Load(repo)
	if err != nil {
		return fail(err)
	}
	// Moved only from the tester's own state: an issue labelled ready by
	// hand keeps it, the kind recorded all the same for the developer.
	s, ok := line.StateOf(t.Labels)
	ours := ok && s.Role == "tester" && s.Next != ""
	to, did, rule := s.Next, "records the proof of this issue: "+p.Kind+", no new test", "proof-recorded"
	next := "it is " + s.Next + ": whoever builds it proves it so, " + ProofKinds[p.Kind]
	if !ours {
		to, next = "", "whoever builds it proves it so, "+ProofKinds[p.Kind]+"; its state kept, not the tester's"
	}
	if p.Kind == KindNone {
		p.Body = t.Body // the body it read: sent again unchanged, Blocked (#815)
		to, did, rule = "", "finds no test possible for this issue", "no-test-possible"
		for _, o := range line.States {
			if o.Role == "product-owner" {
				to = o.Name
			}
		}
		if to == "" {
			return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no test possible, and no state of the product owner's to send it to",
				Findings: []verdict.Finding{{Rule: "no-product-owner-state", Where: where,
					Message: "the routing's states name none the product owner's: the issue keeps its state; add one, or have a person settle it"}}})
		}
		next = "back to the product owner (" + to + "): it picks a proof kind, or rewrites the scenarios; a person of the project may name one in a comment, `proof: <kind> — why`"
	}
	var move []intent.Intention
	if to != "" {
		add, remove, err := line.Move(to)
		if err != nil {
			return fail(err)
		}
		remove = slices.DeleteFunc(remove, func(l string) bool { return !slices.Contains(t.Labels, l) })
		move = []intent.Intention{{Kind: "label", Value: map[string]any{"issue": t.Issue, "add": add, "remove": remove}}}
	}
	// The record, none too: the product owner's pick answers it.
	data, _ := json.Marshal(p)
	body := backlog.Said("The tester", did, p.Reason, next) + "\n<!-- workline:" + backlog.ProofMark + string(data) + " -->"
	intents = append(append(slices.DeleteFunc(intents, func(i intent.Intention) bool { return i.Kind == "note" }), move...),
		intent.Intention{Kind: "comment", Value: map[string]any{"issue": t.Issue, "sticky": "issue", "body": body}})
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return fail(err)
	}
	summary := fmt.Sprintf("no new test: %s; the issue moves to %s", p.Kind, to)
	if to == "" {
		summary = fmt.Sprintf("no new test: %s, recorded; the issue's state kept", p.Kind)
	}
	return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: summary,
		Findings: []verdict.Finding{{Rule: rule, Level: "info", Where: where, Message: p.Kind + ": " + p.Reason}}})
}

// sentBack ends pre with no agent on an issue sent to its tests again as
// it stood when the tester found no test possible, no proof picked (#815):
// a second "no test possible" would send it back to refine, to be readied
// again unchanged. It goes to Blocked once, said on it, the record kept;
// false when the issue is not so.
func sentBack(runDir, repo string, f forge.Forge, is forge.Issue) (int, bool) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return 0, false
	}
	comments, err := b.Comments(forge.Target{Kind: "issue", ID: is.ID})
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal, true
	}
	if err != nil {
		return fail(err), true
	}
	p, ok := backlog.SentBackUnchanged(is, comments, "product-owner")
	if !ok {
		return 0, false
	}
	line, err := routing.Load(repo)
	if err != nil {
		return fail(err), true
	}
	add, remove := []string{backlog.LabelBlocked}, []string{backlog.LabelReady, backlog.LabelNeedsTests, backlog.LabelToRefine}
	if ad, rm, err := line.Move(BlockedState); err == nil {
		add, remove = ad, rm
	}
	remove = slices.DeleteFunc(remove, func(l string) bool { return !slices.Contains(is.Labels, l) })
	why := "it came back to its tests as it stood when the tester found no test possible, and no proof was picked; " +
		"written again, the tests would send it back to refine, and refine to its tests. The tester said: " + p.Reason
	data, _ := json.Marshal(p)
	body := backlog.Said("The tester", "leaves this issue blocked", why,
		"a person of the project names its proof in a comment, `proof: <kind> — why`, or edits its "+work.Name(work.Validation)+
			" so a test checks it, then takes "+backlog.LabelBlocked+" off") +
		"\n<!-- workline:" + backlog.ProofMark + string(data) + " -->"
	intents := []intent.Intention{
		{Kind: "label", Value: map[string]any{"issue": is.ID, "add": add, "remove": remove}},
		{Kind: "comment", Value: map[string]any{"issue": is.ID, "sticky": "issue", "body": body}},
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), intents); err != nil {
		return fail(err), true
	}
	v := verdict.Verdict{Status: verdict.Pass, Summary: "sent to its tests again unchanged after no test possible: blocked",
		Findings: []verdict.Finding{{Rule: "sent-back-unchanged", Level: "warn", Where: fmt.Sprintf("#%d", is.ID), Message: why}}}
	if data, err = json.Marshal(v); err != nil {
		return fail(err), true
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", saidFile), data, 0o644); err != nil {
		return fail(err), true
	}
	return 0, true
}
