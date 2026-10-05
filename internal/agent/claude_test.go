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
