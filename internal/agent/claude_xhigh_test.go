package agent

import (
	"strings"
	"testing"
)

// Given Claude Code 2.1.325, When I run a role with `--ai claude:sonnet@xhigh`,
// Then it runs and its call records the effort `xhigh`: the spec is accepted,
// the effort is kept as asked, and Claude is passed `--effort xhigh`.
func TestParseClaudeXhighForced(t *testing.T) {
	a, err := Parse("claude:sonnet@xhigh")
	if err != nil {
		t.Fatalf("claude:sonnet@xhigh refused: %v", err)
	}
	if a != (claude{model: "sonnet", effort: "xhigh"}) {
		t.Errorf("got %#v", a)
	}
	if got := claudeEffort["xhigh"]; got != "xhigh" {
		t.Errorf("claudeEffort[xhigh] = %q, want xhigh", got)
	}
}

// Given any agent, When I force the effort `huge`, Then workline refuses it
// and lists the known efforts, `xhigh` among them.
func TestParseClaudeUnknownEffortListsXhigh(t *testing.T) {
	_, err := Parse("claude:sonnet@huge")
	if err == nil {
		t.Fatal("claude:sonnet@huge accepted")
	}
	for _, want := range []string{`"huge"`, "known:", "none", "low", "medium", "high", "xhigh", "max"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}
