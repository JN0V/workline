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
