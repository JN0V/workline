package devmark

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// Passed is a ready issue the pick passed over, and why.
type Passed struct {
	Issue int    `json:"issue"`
	Why   string `json:"why"`
	// Next is what the issue waits for when the developer's own pick
	// holds it, said on the issue (ADR-0059, decision 1); none for a
	// person's hold, or an open pull request that says it already.
	Next string `json:"-"`
}

// Act is the record of a pick, hidden in the comment it leaves on the
// issue: the commit the run read, the issue taken, the ready issues before
// it passed over and why.
type Act struct {
	Act    string   `json:"act"`
	At     string   `json:"at,omitempty"`
	Issue  int      `json:"issue"`
	Passed []Passed `json:"passed,omitempty"`
	// Refused: each file its agent's patches reached past its bounds, and
	// the bound (ADR-0054, step 2).
	Refused []Refusal `json:"refused,omitempty"`
	// Unread: its agent's words that did not read, as they came, for its
	// pull request's body (agentWords); never kept in the record.
	Unread []string `json:"-"`
	// Proof: what the project's tests said on its code; Caps: the bounds
	// on the run it reached (step 3).
	Proof *Proof `json:"proof,omitempty"`
	Caps  []Cap  `json:"caps,omitempty"`
	// Turns, Tokens, Cached and Tools: what its agent used, as it
	// reported them (ADR-0056, steps 4, 5 and 7): its fresh tokens, what
	// ai-max-tokens holds, its cache reads apart, the times it used each
	// tool; 0 or none, not reported.
	Turns  int            `json:"turns,omitempty"`
	Tokens int            `json:"tokens,omitempty"`
	Cached int            `json:"cached,omitempty"`
	Tools  map[string]int `json:"tools,omitempty"`
	// AgentFailed: why its agent's call failed, its time aside; what it
	// edited is proven all the same, its pull request a Draft.
	AgentFailed string `json:"agent-failed,omitempty"`
	// NoCode: its agent wrote no code, so no pull
	// request; the claim waits to be let go (ADR-0059, decision 1).
	NoCode bool `json:"no-code,omitempty"`
	// Body: the issue's body as the claim read it (BodyMark): a claim
	// with no code is let go when the body changes since (#1084).
	Body string `json:"body,omitempty"`
	// Unopened: the times its pull request did not open for this issue,
	// each giving the claim back (#856); read at the pick from the claim's
	// comment, carried on. At the second, the issue is blocked.
	Unopened int `json:"unopened,omitempty"`
	// From: the tester's Draft's commit its code starts from, the
	// acceptance tests on it, those for the issue in Acceptance; Code and
	// Tests: the files its code and its tests wrote, in commits apart
	// (step 4).
	From       string   `json:"from,omitempty"`
	Acceptance []string `json:"acceptance,omitempty"`
	// NoTest: the tester found that no acceptance test applies to the
	// issue and handed it over: no frozen test, said in its pull request
	// (#1941).
	NoTest bool `json:"no-test,omitempty"`
	// Merged: the base's commit merged into From, which was behind it,
	// for its code to start on the base's paths; its branch starts from
	// that merge (#1941).
	Merged string `json:"merged,omitempty"`
	// RedUnproven: why the tester's red proof of those tests was not
	// settled by its judge (ADR-0065): said on its pull request.
	RedUnproven string `json:"red-unproven,omitempty"`
	// Unproven: the scenarios the tester said no test proves, read from
	// its Draft's record at each run, for its pull request's body; never
	// kept in the record.
	Unproven []string `json:"-"`
	Code     []string `json:"code,omitempty"`
	Tests    []string `json:"tests,omitempty"`
	// The review loop (step 6): the rounds run, the reviewer's runs and
	// a person's comments read already, the findings asked before (to
	// tell one still there), the five whys run, a person handed it.
	Round      int `json:"round,omitempty"`
	ReviewRuns int `json:"review-runs,omitempty"`
	// ReviewAt: the head of the reviewer's summary its last round read;
	// another is a new review, its runs counted or not (#1040).
	ReviewAt string   `json:"review-at,omitempty"`
	Heard    []string `json:"heard,omitempty"`
	Seen     []string `json:"seen,omitempty"`
	Rounds   []Round  `json:"rounds,omitempty"`
	Whys     bool     `json:"whys,omitempty"`
	Handed   bool     `json:"handed,omitempty"`
	// GatesAt: the head of the gates' report its last round asked the
	// findings of: the same report is not asked again.
	GatesAt string `json:"gates-at,omitempty"`
	// ChecksAt: the head its last round read the project's red checks
	// at: the same head is never asked twice (#917).
	ChecksAt string `json:"checks-at,omitempty"`
	// Widen: its last round's wait, for a person or the tester (#938,
	// #987).
	Widen *Widen `json:"widen,omitempty"`
	// Beside: the new test files its code wrote beside the Scope
	// (ADR-0063, §9): listed on its pull request, and added to the Scope
	// at the product owner's next run.
	Beside []string `json:"beside,omitempty"`
	// Beyond: the files its code wrote beyond the Scope, each with its
	// agent's reason (ADR-0065, change 2): listed on its pull request,
	// for the reviewer's `intent` lens to judge.
	Beyond []Beyond `json:"beyond,omitempty"`
	// Tester: the keys of the review's test findings sent to the tester
	// (#1265): never asked again, of the developer nor the tester.
	Tester []string `json:"tester,omitempty"`
	// Conflict: the head a merge round failed at (ADR-0067): no second
	// one there.
	Conflict string `json:"conflict,omitempty"`
}

var actData = regexp.MustCompile(`<!-- ` + ActMark + ` (\{.*\}) -->`)

// Line is the hidden line a comment keeps the record in.
func (a Act) Line() string {
	data, _ := json.Marshal(a)
	return fmt.Sprintf("<!-- %s %s -->", ActMark, data)
}

// BodyMark is an issue body's short digest, as a claim's record keeps
// it (Act.Body): the line and the developer read a change by it. The
// engine's own "Blocked by" line (backlog.BlockedByMarker) is left out:
// the engine recording what an issue waits on is no change of a
// person's, and starts no round (#1271).
func BodyMark(body string) string {
	lines := strings.Split(body, "\n")
	lines = slices.DeleteFunc(lines, func(l string) bool { return strings.Contains(l, backlog.BlockedByMarker) })
	return Digest(strings.Join(lines, "\n"))
}

// Digest is the short digest of a text, its spaces around trimmed.
func Digest(text string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])[:12]
}

// BodyChanged says whether body is not the one mark, a record's
// BodyMark, was kept of. A record from before the engine's line was left
// out kept the whole body's digest: read as the same body too.
func BodyChanged(mark, body string) bool {
	return mark != BodyMark(body) && mark != Digest(body)
}

// ReadAct reads a pick's record from a comment of the developer's.
func ReadAct(body string) (Act, bool) {
	m := actData.FindStringSubmatch(body)
	var a Act
	if m == nil || json.Unmarshal([]byte(m[1]), &a) != nil || a.Act == "" {
		return Act{}, false
	}
	return a, true
}

// ReadRecord reads the record of its code on its merge request id, from
// workline's own comment; ok is false when there is none it can read.
func ReadRecord(f forge.Forge, id int) (Act, bool, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		return Act{}, false, err
	}
	body, ok := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	if !ok {
		return Act{}, false, nil
	}
	a, ok := ReadAct(body)
	return a, ok && a.Act == "code", nil
}

// RoundLines are its review rounds as the auditor reads them: each, what
// was asked and how it ended; "" for none (ADR-0046).
func RoundLines(a Act) string {
	var b strings.Builder
	for _, x := range a.Rounds {
		fmt.Fprintf(&b, "- Round %d: %s", x.N, RoundResult(x))
		if x.Whys {
			b.WriteString(", the five whys run")
		}
		if x.Nits > 0 {
			fmt.Fprintf(&b, ", %d nit%s not asked", x.Nits, Plural(x.Nits))
		}
		b.WriteString(".\n")
		for _, t := range x.Asked {
			fmt.Fprintf(&b, "  - asked: %s\n", t)
		}
	}
	return b.String()
}

// RoundResult says how a review round ended.
func RoundResult(x Round) string {
	switch {
	case x.Person:
		return "its issue blocked"
	case x.Result == "no-code":
		return "no fix"
	case x.Result == "not-pushed":
		return "its fix not pushed, red on a green head"
	}
	return "its fix pushed, the tests " + x.Result
}

// ClaimActIn is the last developer's record among comments; the zero Act
// when none reads.
func ClaimActIn(comments []string) Act {
	var last Act
	for _, c := range comments {
		if a, ok := ReadAct(c); ok {
			last = a
		}
	}
	return last
}

// NoCodeOnOtherBody says whether a claim's record a wrote no code on a
// body other than body (#1084); false for a record from before Act.Body.
func NoCodeOnOtherBody(a Act, body string) bool {
	return a.NoCode && a.Body != "" && BodyChanged(a.Body, body)
}

// SpentOnSameBody says whether a claim's record a ran on the body it has
// now (or one it did not keep): no code, or its pull request gone, merged
// part of it or closed by a person, not by workline (callers ask with
// none open). Past claim-hours, the line blocks it for a person.
func SpentOnSameBody(b forge.Backlog, a Act, body string) bool {
	switch c, ok := b.(forge.Closings); {
	case a.Body != "" && BodyChanged(a.Body, body):
		return false
	case a.NoCode:
		return true
	case !ok || len(a.Code) == 0:
		return false
	default:
		ours, err := c.ClosedWith(fmt.Sprint(Branch, a.Issue), forge.Marker("closed"))
		return err == nil && !ours
	}
}

// ActMark opens the hidden line of the developer's record on an issue
// it took, what the auditor reads (ADR-0046).
const ActMark = "workline:developer-act"
