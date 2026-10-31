package tester

// A proof kind, not always a new test (ADR-0059, decision 4): for a flaky
// test to make reliable, a refactor or a docs change, the tester records
// the kind of proof and its reason on the issue, and the issue moves on
// with no Draft; the developer accepts it in place of one. "No test
// possible" goes back to the product owner.

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// The proof kinds: new tests, the tester's Draft; the others in place of
// one; KindNone, no test possible.
const (
	KindNewTests = "new-tests"
	KindFlaky    = "flaky-n-of-n"
	KindSuite    = "existing-suite"
	KindDocs     = "docs-checks"
	KindNone     = "none"
)

// ProofKinds are the kinds a tester records in place of a Draft, each
// with what proves the change.
var ProofKinds = map[string]string{
	KindFlaky: "the named test passes N runs of N",
	KindSuite: "the project's suite, unchanged and green",
	KindDocs:  "the documentalist's checks",
}

// Proof is the tester's record of a kind in place of a Draft, kept in its
// comment on the issue.
type Proof struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

const proofMark = "tester-proof "

// proofNote is the agent's note naming a kind: `proof: <kind> — why`.
var proofNote = regexp.MustCompile(`(?is)^\s*proof:\s*([a-z-]+)\s*[—:-]*\s*(.*)$`)

// proofOf reads the kind a note names, and its reason; false when no note
// names one.
func proofOf(notes []string) (Proof, bool) {
	for _, n := range notes {
		if m := proofNote.FindStringSubmatch(n); m != nil {
			return Proof{Kind: strings.ToLower(m[1]), Reason: strings.TrimSpace(m[2])}, true
		}
	}
	return Proof{}, false
}

// ReadProof is the kind the tester recorded on an issue, from its
// comments; false when it recorded none.
func ReadProof(comments []string) (Proof, bool) {
	sticky := "<!-- workline:sticky=tester/issue -->"
	for _, c := range slices.Backward(comments) {
		if !strings.Contains(c, sticky) {
			continue
		}
		_, after, ok := strings.Cut(c, "<!-- workline:"+proofMark)
		if !ok {
			return Proof{}, false // its comment now says something else
		}
		data, _, _ := strings.Cut(after, " -->")
		var p Proof
		if json.Unmarshal([]byte(data), &p) == nil && ProofKinds[p.Kind] != "" {
			return p, true
		}
		return Proof{}, false
	}
	return Proof{}, false
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
	line, err := routing.Load(repo)
	if err != nil {
		return fail(err)
	}
	s, ok := line.StateOf(t.Labels)
	if !ok || s.Role != "tester" {
		return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "a proof kind recorded, the issue's state kept: it is not the tester's",
			Findings: []verdict.Finding{{Rule: "proof-recorded", Level: "info", Where: where, Message: p.Kind + ": " + p.Reason}}})
	}
	to, did, next, rule := s.Next, "records the proof of this issue: "+p.Kind+", no new test",
		"it is "+s.Next+": whoever builds it proves it so, "+ProofKinds[p.Kind], "proof-recorded"
	if p.Kind == KindNone {
		to, did, rule = "", "finds no test possible for this issue", "no-test-possible"
		for _, o := range line.States {
			if o.Role == "product-owner" {
				to = o.Name
			}
		}
		next = "back to the product owner (" + to + "): it picks a proof kind, or rewrites the scenarios"
	}
	add, remove, err := line.Move(to)
	if err != nil {
		return fail(err)
	}
	remove = slices.DeleteFunc(remove, func(l string) bool { return !slices.Contains(t.Labels, l) })
	body := backlog.Said("The tester", did, p.Reason, next)
	if p.Kind != KindNone {
		data, _ := json.Marshal(p)
		body += "\n<!-- workline:" + proofMark + string(data) + " -->"
	}
	intents = append(slices.DeleteFunc(intents, func(i intent.Intention) bool { return i.Kind == "note" }),
		intent.Intention{Kind: "label", Value: map[string]any{"issue": t.Issue, "add": add, "remove": remove}},
		intent.Intention{Kind: "comment", Value: map[string]any{"issue": t.Issue, "sticky": "issue", "body": body}})
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return fail(err)
	}
	return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("no new test: %s; the issue moves to %s", p.Kind, to),
		Findings: []verdict.Finding{{Rule: rule, Level: "info", Where: where, Message: p.Kind + ": " + p.Reason}}})
}
