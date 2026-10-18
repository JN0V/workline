package agent

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// What a lens answered on PR #137, in the shapes that broke it (#138): a
// text starting with a backtick, written plain, does not read; the same
// text as a block scalar reads exactly as written — quotes, fences, `: `
// and ` #` included — the answer's point being to quote code exactly.
func TestProposalsFromCodeInText(t *testing.T) {
	plain := "- finding:\n    severity: nit\n    title: The agreed proposal is read from the first fence\n" +
		"    why: `ProposalComment` writes the agent's text before the engine's block\n" +
		"    cause: {path: internal/backlog/backlog.go, quote: \"if strings.HasPrefix(l, \\\"```\\\") {\"}\n"
	if _, _, err := proposalsFrom(plain); err == nil || !strings.Contains(err.Error(), "cannot start any token") {
		t.Fatalf("a plain text starting with a backtick read: %v", err)
	}
	why := "`ProposalComment` writes the agent's `Why` first: if it holds a\n```yaml\nfence, # the engine reads that one\n"
	quote := "if strings.HasPrefix(l, \"```\") {\n\tfirst = i // a fence: the answer's\n"
	block := "- finding:\n    severity: nit\n    title: \"The agreed proposal: read from the first fence\"\n" +
		"    why: |\n" + indent(why) + "    cause:\n      path: internal/backlog/backlog.go\n      quote: |\n" + indent(quote)
	got, _, err := proposalsFrom(block)
	if err != nil {
		t.Fatalf("a block scalar did not read: %v", err)
	}
	var list []map[string]map[string]any
	if err := yaml.Unmarshal(got, &list); err != nil {
		t.Fatal(err)
	}
	f := list[0]["finding"]
	if f["why"] != why || f["cause"].(map[string]any)["quote"] != quote {
		t.Errorf("the text did not read as written: %q, %q", f["why"], f["cause"])
	}
	// Its first line starting with a tab, the reader takes it for
	// indentation: the engine says the block's indentation (Mend).
	if got, _, err := proposalsFrom("- note: |\n    \tx\n"); err != nil || string(got) != "- note: |2\n    \tx\n" {
		t.Errorf("a block scalar starting with a tab: %q, %v", got, err)
	}
}

func indent(s string) string {
	var b strings.Builder
	for _, l := range strings.SplitAfter(s, "\n") {
		if l != "" {
			b.WriteString("        " + l)
		}
	}
	return b.String()
}

func TestProposalsFrom(t *testing.T) {
	moved := "- patch: |\n    --- a/d.md\n    +++ b/d.md\n    @@ -1,3 +0,0 @@\n    -```\n    -run it\n    -```\n"
	for _, c := range []struct{ name, answer, want string }{
		{"plain", "- note: done\n", "note: done"},
		{"fenced", "```yaml\n- note: done\n```\n", "note: done"},
		{"prose around a fence", "Here it is:\n```yaml\n- note: done\n```\nThat is all.", "note: done"},
		{"a code block inside a patch", moved, "-run it"},
		{"a fenced answer holding a code block", "```yaml\n" + moved + "```\n", "-run it"},
	} {
		got, _, err := proposalsFrom(c.answer)
		if err != nil || !strings.Contains(string(got), c.want) {
			t.Errorf("%s: %q, %v", c.name, got, err)
		}
	}
	if got, _, err := proposalsFrom(" []\n"); err != nil || string(got) != "[]\n" {
		t.Errorf("an empty list is an answer proposing nothing: %q, %v", got, err)
	}
	if _, _, err := proposalsFrom("I could not decide."); err == nil {
		t.Error("prose alone is not a proposal")
	}
}

func TestClaudeAnswer(t *testing.T) {
	var call Call
	r, ok := claudeAnswer([]byte(`{"result":"- ok: 1","is_error":false,"total_cost_usd":0.0134,
		"modelUsage":{"claude-sonnet-5":{"inputTokens":9,"cacheReadInputTokens":6000,"cacheCreationInputTokens":400,"outputTokens":900},
		"claude-haiku-4-5-20251001":{"inputTokens":100,"outputTokens":40}}}`), &call)
	if !ok || r.Result != "- ok: 1" || call.Model != "claude-sonnet-5" || call.CostUSD != 0.0134 ||
		call.TokensIn != 6509 || call.TokensCached != 6000 || call.TokensOut != 940 {
		t.Errorf("got %+v, %+v, %v", r, call, ok)
	}
	// Sonnet asked and answering "[]" in 10 tokens, the side Haiku writing
	// 40: Sonnet answered (the evaluation's clean rename, #126).
	short := Call{Asked: "sonnet"}
	claudeAnswer([]byte(`{"result":"[]","modelUsage":{"claude-haiku-4-5-20251001":{"inputTokens":300,"outputTokens":40},
		"claude-sonnet-5-5":{"inputTokens":4000,"outputTokens":10}}}`), &short)
	if short.Model != "claude-sonnet-5-5" || short.TokensOut != 50 {
		t.Errorf("a short answer: %+v", short)
	}
	// No model asked by name: the one writing the most.
	free := Call{}
	claudeAnswer([]byte(`{"result":"[]","modelUsage":{"claude-haiku-4-5-20251001":{"outputTokens":40},"claude-sonnet-5-5":{"outputTokens":10}}}`), &free)
	if free.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("no model asked: %+v", free)
	}
	if _, ok := claudeAnswer([]byte("- ok: 1"), &call); ok {
		t.Error("plain text read as Claude's JSON")
	}
}

func TestParseClaude(t *testing.T) {
	for spec, want := range map[string]claude{
		"claude":                      {},
		"claude:opus":                 {model: "opus"},
		"claude:claude-sonnet-5@high": {model: "claude-sonnet-5", effort: "high"},
		"claude:@max":                 {effort: "max"},
	} {
		a, err := Parse(spec)
		if err != nil || a != want {
			t.Errorf("%s: got %#v, %v", spec, a, err)
		}
	}
	for _, spec := range []string{"claude:", "claude:opus@loud"} {
		if _, err := Parse(spec); err == nil {
			t.Errorf("%s: accepted", spec)
		}
	}
}

// An item holding several keys where one kind goes is refused, the reader
// told where they go: a lens wrote a finding's `decision` and `cause` as an
// item of their own (#490, #492) and, told only to send valid YAML, sent
// it again so.
func TestProposalsFromSaysWhereKeysGo(t *testing.T) {
	for _, c := range []struct{ answer, want string }{
		{"- note: x\n- decision: \"Should it be 0?\"\n  cause: {path: \"#4\", quote: left open}\n",
			"proposal 2 holds 2 kinds (cause, decision); each holds exactly one: cause and decision are no kind, but keys of a `finding`, written under `- finding:`, indented, beside its other keys"},
		{"- finding: {title: t}\n  decision: \"Should it be 0?\"\n",
			"proposal 1 holds 2 kinds (decision, finding); each holds exactly one: decision is no kind, but a key of a `finding`"},
		{"- comment: hello\n  label: {add: [bug]}\n", "proposal 1 holds 2 kinds (comment, label); each holds exactly one: write each as an item of its own"},
		{"- note: hello\n  colour: red\n", "colour is no kind, but a key of a proposal, written under its kind, indented, as the answer's shape shows it"},
		{"- note: hello\n  why: because\n", "why is no kind, but a key of a proposal, written under its kind"}, // a key of many kinds
		{"- {}\n", "proposal 1 holds no kind; each holds exactly one"},
		// A cause's key beside its finding goes under the cause, not beside
		// the finding's keys: no finding named.
		{"- finding: {title: t}\n  path: a.go\n", "path is no kind, but a key of a proposal, written under its kind"},
	} {
		if _, _, err := proposalsFrom(c.answer); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.answer, err, c.want)
		}
	}
	// Only a kind the role writes is named: a role writing no finding is not
	// told to write one.
	answer := "- note: x\n  cause: {path: a.go, quote: x}\n"
	if _, _, err := proposalsFrom(answer, "note", "comment"); err == nil || strings.Contains(err.Error(), "finding") ||
		!strings.Contains(err.Error(), "cause is no kind, but a key of a proposal") {
		t.Errorf("a role writing no finding: %v", err)
	}
	if _, _, err := proposalsFrom(answer, "finding", "note"); err == nil || !strings.Contains(err.Error(), "under `- finding:`") {
		t.Errorf("a role writing findings: %v", err)
	}
	if _, _, err := proposalsFrom("- finding: {title: t}\n  path: a.go\n", "finding"); err == nil || strings.Contains(err.Error(), "`finding`") {
		t.Errorf("a cause's key told to go beside the finding's: %v", err)
	}
}

// The keys of a shape's top, its comment's included; not a nested map's,
// nor a word in a quoted text.
func TestShapeKeys(t *testing.T) {
	top := shapeKeys(contracts["finding"])
	for _, k := range []string{"lens", "severity", "title", "why", "cause", "symptom", "fix", "seen", "claim", "decision"} {
		if !top[k] {
			t.Errorf("%s is a finding's key", k)
		}
	}
	for _, k := range []string{"path", "quote", "line", "words"} {
		if top[k] {
			t.Errorf("%s is no key at a finding's top", k)
		}
	}
	if keys := shapeKeys(`- refine: {issue: 12, need: "one line: who needs what"}`); keys["line"] || !keys["need"] {
		t.Errorf("a word in a quoted text read as a key: %v", keys)
	}
}
