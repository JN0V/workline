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
