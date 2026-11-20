package tester

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// blockedReply runs judgeAnswer on an answer that is refused, and returns
// the key of the comment its reply to the builder is kept in.
func blockedReply(t *testing.T, a answer, notes []string) any {
	t.Helper()
	t.Setenv("WORKLINE_AI", "claude")
	runDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(runDir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	var intents []intent.Intention
	for _, n := range notes {
		intents = append(intents, intent.Intention{Kind: "note", Value: n})
	}
	if len(intents) > 0 {
		if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
			t.Fatal(err)
		}
	}
	if code := judgeAnswer(runDir, t.TempDir(), a, Settings{}); code != 1 {
		t.Fatalf("exit %d, want 1 (blocked)", code)
	}
	said, err := intent.Read(filepath.Join(runDir, "out", "on-block.yaml"))
	if err != nil || len(said) != 1 || said[0].Kind != "comment" {
		t.Fatalf("on-block.yaml: %+v, %v; want one comment", said, err)
	}
	m, ok := said[0].Value.(map[string]any)
	if !ok {
		t.Fatalf("the reply is %T, want a map", said[0].Value)
	}
	return m["sticky"]
}

// Said on the item; Updated in place
// Given a tester's answer that is refused (no reason, then five whys
// missing, two other reasons),
// When the tester's reply is written for the builder's thread each time,
// Then the reply is one sticky comment, kept under the same key whatever
// the reason, so a second refusal updates it and adds no other.
func TestBlockedAnswerReplyIsOneStickyComment(t *testing.T) {
	a := answer{Thread: "t1", File: "test_x.py", URL: "https://example.invalid/mr/1"}
	first := blockedReply(t, a, nil)
	b := a
	b.FiveWhys = true
	second := blockedReply(t, b, []string{"because the test holds"})
	if first == nil || first == "" {
		t.Fatalf("the reply of a refused answer is no sticky comment: key %v", first)
	}
	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Errorf("two refusals, two keys (%v, %v): a second reply is added, not updated", first, second)
	}
}
