package backlog

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

const noTestComment = "**The tester** finds no test possible.\n\n<!-- workline:tester-proof {\"kind\":\"none\",\"reason\":\"wording only.\"} -->\n<!-- workline:sticky=tester/issue -->"

func poState(answers string) string {
	return FormatState(State{Confirmed: "abc", Proof: &Proof{Kind: KindDocs, Reason: "the docs checks", Answers: answers}}) + "\n\n" + StateMarker("product-owner")
}

func TestReadProofTakesThePickAnsweringTheTestersWord(t *testing.T) {
	_, digest, ok := NoTest([]string{noTestComment})
	if !ok || len(digest) != 12 {
		t.Fatalf("NoTest = %q, %v", digest, ok)
	}
	cases := []struct {
		name     string
		comments []string
		kind     string
	}{
		{"none alone: no proof", []string{noTestComment}, ""},
		{"the pick answers it", []string{poState(digest), noTestComment}, KindDocs},
		{"the pick answers an older word", []string{poState("000000000000"), noTestComment}, ""},
		{"the tester's kind wins", []string{poState(digest), "<!-- workline:tester-proof {\"kind\":\"existing-suite\",\"reason\":\"a refactor\"} -->\n<!-- workline:sticky=tester/issue -->"}, KindSuite},
		{"the tester's comment says something else", []string{poState(digest), "**The tester** objects.\n<!-- workline:sticky=tester/issue -->"}, ""},
	}
	for _, c := range cases {
		p, ok := ReadProof(c.comments, "product-owner")
		if got := map[bool]string{true: p.Kind}[ok]; got != c.kind {
			t.Errorf("%s: kind %q, want %q", c.name, got, c.kind)
		}
	}
}

func TestPersonsPickIsAPersonOfTheProjectsAfterTheTester(t *testing.T) {
	tester := forge.Note{Body: noTestComment, Bot: true}
	pick := func(author string, insider bool, body string) forge.Note {
		return forge.Note{Body: body, Author: author, Insider: insider}
	}
	cases := []struct {
		name  string
		notes []forge.Note
		by    string
	}{
		{"a person of the project", []forge.Note{tester, pick("bob", true, "proof: docs-checks — wording only")}, "bob"},
		{"the last one", []forge.Note{tester, pick("bob", true, "proof: docs-checks — wording"), pick("ann", true, "proof: existing-suite — a refactor")}, "ann"},
		{"an outsider", []forge.Note{tester, pick("eve", false, "proof: docs-checks — wording")}, ""},
		{"before the tester's word", []forge.Note{pick("bob", true, "proof: docs-checks — wording"), tester}, ""},
		{"no tester's word", []forge.Note{pick("bob", true, "proof: docs-checks — wording")}, ""},
		{"a comment naming no kind", []forge.Note{tester, pick("bob", true, "I think the docs checks prove it")}, ""},
	}
	for _, c := range cases {
		p, ok := PersonsPick(c.notes)
		if got := map[bool]string{true: p.By}[ok]; got != c.by {
			t.Errorf("%s: by %q, want %q", c.name, got, c.by)
		}
	}
	p, _ := PersonsPick([]forge.Note{tester, pick("bob", true, "proof: flaky-n-of-n — it races\ntest: TestIdle\nruns: 20")})
	if p.Kind != KindFlaky || p.Test != "TestIdle" || p.Runs != 20 || p.Reason != "it races" {
		t.Errorf("a flaky pick read as %+v", p)
	}
}

func TestSameTellsAKeptPersonsPick(t *testing.T) {
	pick := Proof{Kind: KindDocs, Reason: "wording", By: "bob"}
	kept := pick
	kept.Answers = "af1c2ed6ea0a"
	if !kept.Same(pick) {
		t.Error("the pick kept, read again: not the same")
	}
	var none *Proof
	agent := Proof{Kind: KindDocs, Reason: "wording"}
	other := pick
	other.Reason = "only words"
	if none.Same(pick) || agent.Same(Proof{Kind: KindDocs, Reason: "wording"}) || kept.Same(other) {
		t.Error("none kept, an agent's pick, or another person's word taken for the same")
	}
}

func TestProofWrong(t *testing.T) {
	cases := map[string]Proof{
		"":                     {Kind: KindSuite, Reason: "a refactor"},
		"proof-reason-missing": {Kind: KindSuite},
		"proof-kind-unknown":   {Kind: KindNone, Reason: "none fits"},
		"proof-test-missing":   {Kind: KindFlaky, Reason: "it races", Test: "a;rm"},
	}
	for want, p := range cases {
		if rule, _ := ProofWrong(p); rule != want {
			t.Errorf("%+v: rule %q, want %q", p, rule, want)
		}
	}
}

// commentsOnly is a backlog whose issues' comments are given: all
// checkNoTest reads from the forge.
type commentsOnly struct {
	forge.Backlog
	comments []string
}

func (c commentsOnly) Comments(forge.Target) ([]string, error) { return c.comments, nil }

func TestCheckNoTestLiftedOnlyByARefineThatRewrites(t *testing.T) {
	is := forge.Issue{ID: 9, Body: "## Need\n\nEvery sale counted.\n\n## Verification\n\nA test.\n\n## Validation\n\nAnn reads it calmly.\n\n## Scope\n\nsrc/a.go"}
	f := commentsOnly{comments: []string{"**The tester** finds no test possible.\n\n<!-- workline:tester-proof {\"kind\":\"none\",\"reason\":\"a feeling.\",\"body\":\"" +
		BodyDigest(is.Body) + "\"} -->\n<!-- workline:sticky=tester/issue -->"}}
	rewrite := Proposal{Do: "refine", Issue: 9, Validation: "- **Three rows:** Given three sales, When Ann exports, Then she counts three.", Revise: []string{"Validation"}}
	same := rewrite
	same.Validation = "Ann reads it calmly."
	outsider := rewrite
	outsider.ToReporter = true
	cases := []struct {
		name string
		d    []Decision
		lift bool
	}{
		{"no refine", nil, false},
		{"a refine done, rewriting", []Decision{{Mode: Act, Act: rewrite}}, true},
		{"a refine proposed only", []Decision{{Mode: Propose, Act: rewrite}}, false},
		{"a refine to its reporter", []Decision{{Mode: Act, Act: outsider}}, false},
		{"a refine writing the same body", []Decision{{Mode: Act, Act: same}}, false},
	}
	for _, c := range cases {
		p := &Plan{Decisions: c.d}
		rule, _ := p.checkNoTest(f, "product-owner", is, &Proposal{Do: "ready", Issue: 9})
		if got := rule == ""; got != c.lift {
			t.Errorf("%s: rule %q, lifted %v, want %v", c.name, rule, got, c.lift)
		}
	}
}

// noTestSaid is the tester's "no test possible" on a body, retried or not.
func noTestSaid(reason, body string, retried bool) string {
	r := ""
	if retried {
		r = `,"retried":true`
	}
	return "**The tester** finds no test possible.\n\n<!-- workline:tester-proof {\"kind\":\"none\",\"reason\":\"" + reason + "\",\"body\":\"" + body + "\"" + r +
		"} -->\n<!-- workline:sticky=tester/issue -->"
}

func TestCheckRetest(t *testing.T) {
	is := forge.Issue{ID: 9, Body: "## Need\n\nEvery sale counted.\n\n## Validation\n\n- **Three rows:** Given three sales, When Ann exports, Then she counts three."}
	read := BodyDigest(is.Body)
	rewrite := Proposal{Do: "refine", Issue: 9, Validation: "- **Four rows:** Given four sales, When Ann exports, Then she counts four.", Revise: []string{"Validation"}}
	cases := []struct {
		name     string
		comments []string
		d        []Decision
		proof    *Proof
		rule     string
		retest   bool
	}{
		{"answered", []string{noTestSaid("no code.", read, false)}, nil, nil, "", true},
		{"a proof beside it", []string{noTestSaid("no code.", read, false)}, nil, &Proof{Kind: KindSuite, Reason: "covered"}, "retest-with-proof", true},
		{"no tester word", nil, nil, nil, "retest-not-asked", true},
		{"spent", []string{noTestSaid("no code.", read, true)}, nil, nil, "retest-spent", true},
		{"rewritten since", []string{noTestSaid("no code.", "0ldb0dy0ld00", false)}, nil, nil, "", false},
		{"rewritten in this run", []string{noTestSaid("no code.", read, false)}, []Decision{{Mode: Act, Act: rewrite}}, nil, "", false},
	}
	for _, c := range cases {
		p := &Plan{Decisions: c.d}
		pr := &Proposal{Do: "ready", Issue: 9, Retest: true, Proof: c.proof}
		rule, _ := p.checkRetest(is, c.comments, "product-owner", pr)
		if rule != c.rule || (rule == "" && pr.Retest != c.retest) {
			t.Errorf("%s: rule %q retest %v, want %q %v", c.name, rule, pr.Retest, c.rule, c.retest)
		}
		if rule == "" && pr.Retest && pr.Retested == "" {
			t.Errorf("%s: no word recorded as retested", c.name)
		}
	}
}

func TestRetestedNotByAStaleRetest(t *testing.T) {
	state := func(key string) string {
		return FormatState(State{Confirmed: "d8c4dff", Retest: key}) + "\n" + StateMarker("product-owner")
	}
	first := []string{noTestSaid("no code.", "b0dy00000001", false)}
	key, _ := RetestKey(first)
	if !Retested(append([]string{state(key)}, first...), "product-owner") {
		t.Fatal("the word retested reads as not retested")
	}
	// The same reason, on a rewritten body: another word, not retested.
	later := []string{state(key), noTestSaid("no code.", "b0dy00000002", false)}
	if Retested(later, "product-owner") {
		t.Error("a stale retest answers a later word on another body")
	}
}
