package documentalist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// Not vouched for: Given a doc the agent judges without vouching for it,
// When its patch also changes `verified`,
// Then the doc keeps its `verified` and gets `judged`.
func TestNotVouchedFor(t *testing.T) {
	dir := t.TempDir()
	doc := "---\nchecked: aaaaaaa\njudged: 1111111\nverified: human:alice\n---\n# Title\n\nBody line one.\nBody line two.\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "."},
		{"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	full := "bbbbbbb0000000000000000000000000000000ff"
	diff := "--- a/README.md\n+++ b/README.md\n@@ -3,7 +3,7 @@\n" +
		"-judged: 1111111\n" +
		"-verified: human:alice\n" +
		"+judged: " + full[:7] + "\n" +
		"+verified: agent:documentalist\n" +
		" ---\n" +
		" # Title\n" +
		" \n" +
		"-Body line one.\n" +
		"+Body line 1.\n" +
		" Body line two.\n"
	judged := map[string]map[string]string{"README.md": {"": full}}
	intents := []intent.Intention{{Kind: "patch", Value: diff}}
	j, err := judgePatches(dir, Settings{}, judged, map[string]string{}, nil, map[string]bool{"README.md": true}, intents, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(j.refused) > 0 {
		t.Fatalf("patch refused: %+v", j.refused)
	}
	if j.fix == nil {
		t.Fatal("no fix accepted")
	}
	got := j.fix.after["README.md"]
	if v := headerValue(got, "verified"); v != "human:alice" {
		t.Errorf("verified = %q, want it kept as human:alice", v)
	}
	if v := headerValue(got, "judged"); v != full[:7] {
		t.Errorf("judged = %q, want %q", v, full[:7])
	}
	if v := headerValue(got, "checked"); v != "aaaaaaa" {
		t.Errorf("checked = %q, want it left as aaaaaaa", v)
	}
	if !strings.Contains(got, "Body line 1.") {
		t.Errorf("the body fix was not applied:\n%s", got)
	}
}
