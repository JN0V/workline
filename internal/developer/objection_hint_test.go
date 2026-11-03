package developer

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// No text instead
// Given an acceptance test that calls a command against its documented rule,
// When the agent explains the problem in a text note,
// Then the run says an objection was due, not only "refused at apply".
func TestNoTextInstead(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	b := Bounds{
		Issue:  7,
		Scope:  []string{"pkg/run.go"},
		Frozen: map[string]int{"pkg/run_test.go": 7},
	}
	note := intent.Intention{Kind: "comment", Value: map[string]any{
		"issue": 7,
		"body":  "pkg/run_test.go runs the command without the flag its no-terminal rule requires: it cannot pass.",
	}}
	_, _, said := bounded(repo, b, nil, []intent.Intention{note})
	if len(said) == 0 {
		t.Fatalf("the text note was not refused at apply")
	}
	hinted := false
	for _, s := range said {
		if strings.Contains(s, ObjectionsFile) {
			hinted = true
		}
	}
	if !hinted {
		t.Fatalf("the refusal does not say an objection was due (naming %s): %q", ObjectionsFile, said)
	}
}
