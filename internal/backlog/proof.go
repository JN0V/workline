package backlog

// A proof kind, not always a new test (ADR-0059, decision 4): for a flaky
// test to make reliable, a refactor or a docs change, the tester records
// the kind of proof and its reason on the issue, and the issue moves on
// with no Draft; the developer accepts it in place of one. "No test
// possible" goes back to the product owner, which picks a kind (its
// ready's proof, kept in its state) or sends the issue back to its tests.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
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

// ProofKinds are the kinds recorded in place of a Draft, each with what
// proves the change.
var ProofKinds = map[string]string{
	KindFlaky: "the named test passes N runs of N",
	KindSuite: "the project's suite, unchanged and green",
	KindDocs:  "the documentalist's checks",
}

// Proof is a kind in place of a Draft: the tester's record, kept in its
// comment on the issue, or the product owner's pick after the tester found
// no test possible, kept in its state.
type Proof struct {
	Kind   string `json:"kind" yaml:"kind"`
	Reason string `json:"reason" yaml:"reason"`
	// Test and Runs: for flaky-n-of-n, the test the developer's proof
	// runs and how many times, each run green; no Runs, the developer's
	// flaky-runs setting.
	Test string `json:"test,omitempty" yaml:"test,omitempty"`
	Runs int    `json:"runs,omitempty" yaml:"runs,omitempty"`
	// Answers, the engine's, on the product owner's pick: the tester's
	// "no test possible" it answers (NoTest's digest). A tester's later
	// record replaces it.
	Answers string `json:"answers,omitempty" yaml:"answers,omitempty"`
	// By is the person of the project who named the kind in a comment
	// (PersonsPick): their word, done whatever the role's mode.
	By string `json:"by,omitempty" yaml:"by,omitempty"`
}

// TestName is a test's name as a proof-command takes it in `{test}`:
// letters, digits and `_ . / : -`, nothing a shell reads.
var TestName = regexp.MustCompile(`^[A-Za-z0-9_./:-]{1,200}$`)

// ProofMark marks the tester's record in its comment on the issue.
const ProofMark = "tester-proof "

// testerSticky marks the tester's one comment on an issue.
const testerSticky = "<!-- workline:sticky=tester/issue -->"

// namedLine is a line of a flaky-n-of-n note naming its test or its
// runs: `test: <name>`, `runs: <N>`.
var namedLine = regexp.MustCompile(`(?im)^\s*(test|runs):[ \t]*(\S+)[ \t]*$`)

// proofNote is a note naming a kind: `proof: <kind> — why`.
var proofNote = regexp.MustCompile(`(?is)^\s*proof:\s*([a-z-]+)\s*[—:-]*\s*(.*)$`)

// ProofNote reads the kind a note names, `proof: <kind> — why`, and its
// reason; for flaky-n-of-n, its `test:` and `runs:` lines. False when it
// names none.
func ProofNote(n string) (Proof, bool) {
	m := proofNote.FindStringSubmatch(n)
	if m == nil {
		return Proof{}, false
	}
	p, reason := Proof{Kind: strings.ToLower(m[1])}, m[2]
	if p.Kind == KindFlaky {
		for _, l := range namedLine.FindAllStringSubmatch(reason, -1) {
			if strings.EqualFold(l[1], "test") {
				p.Test = l[2]
			} else if n, err := strconv.Atoi(l[2]); err == nil && n > 0 {
				p.Runs = n
			}
		}
		reason = namedLine.ReplaceAllString(reason, "")
	}
	p.Reason = strings.TrimSpace(reason)
	return p, true
}

// ProofWrong says what is wrong with a kind given in place of a Draft —
// no reason, a kind that is none of ProofKinds, a flaky test with no name
// a test command takes —, with its rule; "" when nothing is.
func ProofWrong(p Proof) (rule, why string) {
	switch {
	case strings.TrimSpace(p.Reason) == "":
		return "proof-reason-missing", "the kind `" + p.Kind + "` comes with no reason: each kind says why no new test fits"
	case ProofKinds[p.Kind] == "":
		return "proof-kind-unknown", "`" + p.Kind + "` is no proof kind: " + KindFlaky + ", " + KindSuite + " or " + KindDocs
	case p.Kind == KindFlaky && p.Test == "":
		return "proof-test-missing", "it names no test (`test: <name>`): the developer's proof runs it N times"
	case p.Kind == KindFlaky && !TestName.MatchString(p.Test):
		return "proof-test-missing", "the test `" + p.Test + "` is no name a test command takes: letters, digits and `_ . / : -`"
	}
	return "", ""
}

// checkProof checks the kind a ready picks: only where the tester's last
// word is "no test possible", the kind's reason given, a flaky test
// named; the pick then records which word it answers.
func (p *Plan) checkProof(f forge.Backlog, c *Proposal) (rule, why string) {
	if f == nil {
		return "proof-not-asked", "no forge to read the tester's word on it from"
	}
	comments, err := f.Comments(forge.Target{Kind: "issue", ID: c.Issue})
	if err != nil {
		return "no-state", err.Error()
	}
	_, digest, ok := NoTest(comments)
	if !ok {
		return "proof-not-asked", "not moved to ready: a proof kind is the tester's to record; the product owner picks one only where the tester found no test possible — a ready alone sends it to its tests"
	}
	if rule, why := ProofWrong(*c.Proof); rule != "" {
		return rule, "not moved to ready: " + why
	}
	if c.Proof.By != "" {
		// A person's pick, read again from the forge: never the agent's
		// word that a person named it.
		notes, err := f.Notes(forge.Target{Kind: "issue", ID: c.Issue})
		if err != nil {
			return "no-state", err.Error()
		}
		if pp, ok := PersonsPick(notes); !ok || pp.By != c.Proof.By || pp.Kind != c.Proof.Kind {
			return "proof-not-a-persons", "not moved to ready: no comment of @" + c.Proof.By + " on it names the kind `" + c.Proof.Kind + "`"
		}
		c.Ticked = c.Proof.By
	}
	c.Proof.Answers = digest
	return "", ""
}

// Same says whether a kept pick is that one, read again: the same person,
// kind and words; false on none kept.
func (p *Proof) Same(o Proof) bool {
	return p != nil && p.By != "" && p.By == o.By && p.Kind == o.Kind && p.Reason == o.Reason && p.Test == o.Test && p.Runs == o.Runs
}

// PersonsPick is the proof kind a person of the project named on an issue
// after the tester's comment, `proof: <kind> — why`, the last one: the
// pick with no agent (ADR-0059, decision 4). False with none.
func PersonsPick(notes []forge.Note) (Proof, bool) {
	at := slices.IndexFunc(notes, func(n forge.Note) bool { return strings.Contains(n.Body, testerSticky) })
	if at < 0 {
		return Proof{}, false
	}
	for _, n := range slices.Backward(notes[at+1:]) {
		if n.Bot || !n.Insider || n.Author == "" {
			continue
		}
		if p, ok := ProofNote(n.Body); ok {
			p.By = n.Author
			return p, true
		}
	}
	return Proof{}, false
}

// testerRecord is the tester's record on an issue, from its comment; false
// when it holds none, or its comment now says something else.
func testerRecord(comments []string) (Proof, bool) {
	for _, c := range slices.Backward(comments) {
		if !strings.Contains(c, testerSticky) {
			continue
		}
		_, after, ok := strings.Cut(c, "<!-- workline:"+ProofMark)
		if !ok {
			return Proof{}, false
		}
		data, _, _ := strings.Cut(after, " -->")
		var p Proof
		if json.Unmarshal([]byte(data), &p) == nil && (ProofKinds[p.Kind] != "" || p.Kind == KindNone) {
			return p, true
		}
		return Proof{}, false
	}
	return Proof{}, false
}

// NoTest is the tester's "no test possible" on an issue, and its digest:
// what a product owner's pick answers. False when the tester's last word
// on it is another.
func NoTest(comments []string) (reason, digest string, ok bool) {
	p, ok := testerRecord(comments)
	if !ok || p.Kind != KindNone {
		return "", "", false
	}
	sum := sha256.Sum256([]byte(p.Reason))
	return p.Reason, hex.EncodeToString(sum[:])[:12], true
}

// ReadProof is the kind an issue is proved by in place of a Draft: the
// tester's record, or, where the tester found no test possible, the kind
// the product owner picked for that finding (its state, `role`'s). False
// when none is recorded.
func ReadProof(comments []string, role string) (Proof, bool) {
	p, ok := testerRecord(comments)
	if !ok {
		return Proof{}, false
	}
	if p.Kind != KindNone {
		return p, true
	}
	_, digest, _ := NoTest(comments)
	st, found, err := ReadState(comments, role)
	if !found || err != nil || st.Proof == nil || st.Proof.Answers != digest || ProofKinds[st.Proof.Kind] == "" {
		return Proof{}, false
	}
	return *st.Proof, true
}
