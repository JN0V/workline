package agent

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNotice(t *testing.T) {
	file := filepath.Join(t.TempDir(), "seen.yaml")
	haiku := Call{Agent: "claude", Asked: "haiku", Model: "claude-haiku-4-5-20251001"}
	if n := Notice(file, haiku); n != "" {
		t.Errorf("first model seen noticed: %s", n)
	}
	if n := Notice(file, haiku); n != "" {
		t.Errorf("same model noticed: %s", n)
	}
	next := haiku
	next.Model = "claude-haiku-5"
	if n := Notice(file, next); !strings.Contains(n, "claude-haiku-5, no longer claude-haiku-4-5-20251001") {
		t.Errorf("change not noticed: %q", n)
	}
	if n := Notice(file, next); n != "" {
		t.Errorf("a change noticed twice: %s", n)
	}
	if n := Notice(file, Call{Agent: "claude", Asked: "sonnet", Model: "claude-sonnet-5"}); n != "" {
		t.Errorf("another alias's first model noticed: %s", n)
	}
	if n := Notice(file, Call{Agent: "fake", Model: "x"}); n != "" {
		t.Errorf("a call that asked nothing noticed: %s", n)
	}
}
