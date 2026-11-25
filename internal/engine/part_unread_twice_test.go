package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
)

// unreadTwice is an agent whose every answer is text that is not a YAML
// list: each call leaves its answer in out/agent-answer.txt and fails with
// its own parse error. It keeps the feedback each call was given.
type unreadTwice struct {
	told []string
	errs []string
}

func (a *unreadTwice) Propose(req agent.Request) (agent.Call, error) {
	fb, _ := os.ReadFile(filepath.Join(req.RunDir, "out", "feedback.md"))
	a.told = append(a.told, string(fb))
	reason := "extra call"
	if n := len(a.told); n <= len(a.errs) {
		reason = a.errs[n-1]
	}
	call := agent.Call{Agent: "fake"}
	if err := os.WriteFile(filepath.Join(req.RunDir, "out", "agent-answer.txt"), []byte("Here is my review of the change.\n"), 0o644); err != nil {
		return call, err
	}
	return call, &unreadError{reason}
}

// unreadError is the agent's ErrInvalidOutput, with the parse error it says.
type unreadError struct{ why string }

func (e *unreadError) Error() string { return agent.ErrInvalidOutput.Error() + ": " + e.why }
func (e *unreadError) Unwrap() error { return agent.ErrInvalidOutput }

// Still bad
// Given a lens answers text that is not a YAML list, and its second answer
// is not valid either,
// When the engine reads the lens's parts,
// Then the lens is asked twice, the second time with the first parse
// error, and the part is left unanswered, its reason naming both parse
// errors, which the reviewer says as `lens-failed`.
func TestPartUnreadTwiceNamesBothParseErrors(t *testing.T) {
	runDir := t.TempDir()
	task := filepath.Join(runDir, "in", "parts", "1-correctness", "task.md")
	for _, d := range []string{filepath.Dir(task), filepath.Join(runDir, "out")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(task, []byte("Review the change.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const first, second = "first parse error: line 1", "second parse error: line 3"
	ag := &unreadTwice{errs: []string{first, second}}
	r := &role.Role{Name: "reviewer", PartIntentions: []string{"finding"}, Model: role.Model{PromoteAfter: 1}}
	res := &Result{}
	if err := askParts(r, Options{}, ag, runDir, []string{task}, res); err != nil {
		t.Fatal(err)
	}
	if len(ag.told) != 2 {
		t.Fatalf("the lens was asked %d times, want 2", len(ag.told))
	}
	if ag.told[0] != "" || !strings.Contains(ag.told[1], first) {
		t.Errorf("feedback given with the calls: %q, want none, then one holding %q", ag.told, first)
	}
	why, err := os.ReadFile(filepath.Join(runDir, "in", "parts", "1-correctness", "unanswered"))
	if err != nil {
		t.Fatalf("the part is not left unanswered: %v", err)
	}
	if !strings.HasPrefix(string(why), "invalid:") {
		t.Errorf("unanswered = %q, want its kind, invalid, first", why)
	}
	for _, want := range []string{first, second} {
		if !strings.Contains(string(why), want) {
			t.Errorf("unanswered = %q, want it to name the parse error %q", why, want)
		}
	}
}
