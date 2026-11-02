package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// What a role asks applied when its merge request does not open (#856):
// read from out/merge-request.yaml, labels and comments alone, what
// failed put in the comment in place of the mark, on one line, its
// backticks unable to end the code span.
func TestUnopenedSaysWhy(t *testing.T) {
	run := t.TempDir()
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(s string) {
		if err := os.WriteFile(filepath.Join(run, "out", "merge-request.yaml"), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("key: \"3\"\nunopened:\n  - label: {issue: 3, remove: [workline:developing]}\n  - comment: {issue: 3, sticky: claim, body: \"did not open: {why}\"}\n")
	in, err := unopened(run)
	if err != nil || len(in) != 2 || in[0].Kind != "label" || in[1].Kind != "comment" {
		t.Fatalf("read: %v %v", in, err)
	}
	if m, _ := in[0].Value.(map[string]any); m["issue"] != 3 {
		t.Errorf("issue not a number: %#v", m["issue"])
	}
	said := sayWhy(in[1], errors.New("git checkout: exit status 1: error: Your local changes\n\tlib.sh `x`"))
	body, _ := said.Value.(map[string]any)["body"].(string)
	if want := "did not open: `git checkout: exit status 1: error: Your local changes lib.sh 'x'`"; body != want {
		t.Errorf("body %q, want %q", body, want)
	}
	if b, _ := in[1].Value.(map[string]any)["body"].(string); !strings.Contains(b, intent.WhyMark) {
		t.Errorf("the role's intention changed in place: %q", b)
	}
	write("unopened:\n  - patch: {file: x, content: y}\n")
	if _, err := unopened(run); err == nil {
		t.Error("a patch read as what to apply when not opened")
	}
}
