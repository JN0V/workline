package agent

import (
	"strings"
	"testing"
)

func TestProposalsFrom(t *testing.T) {
	moved := "- patch: |\n    --- a/d.md\n    +++ b/d.md\n    @@ -1,3 +0,0 @@\n    -```\n    -run it\n    -```\n"
	for _, c := range []struct{ name, answer, want string }{
		{"plain", "- note: done\n", "note: done"},
		{"fenced", "```yaml\n- note: done\n```\n", "note: done"},
		{"prose around a fence", "Here it is:\n```yaml\n- note: done\n```\nThat is all.", "note: done"},
		{"a code block inside a patch", moved, "-run it"},
		{"a fenced answer holding a code block", "```yaml\n" + moved + "```\n", "-run it"},
	} {
		got, err := proposalsFrom(c.answer)
		if err != nil || !strings.Contains(string(got), c.want) {
			t.Errorf("%s: %q, %v", c.name, got, err)
		}
	}
	if _, err := proposalsFrom("I could not decide."); err == nil {
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
