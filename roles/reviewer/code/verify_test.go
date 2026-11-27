package reviewer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A verifier that cannot be prepared (here the diff: no such commits)
// leaves the review as it was, and says it gave no answer (#2308).
func TestVerifierNotPreparedGivesNoAnswer(t *testing.T) {
	runDir := t.TempDir()
	os.MkdirAll(filepath.Join(runDir, "in"), 0o755)
	os.WriteFile(filepath.Join(runDir, "in", "settings.json"), []byte(`{"verifier": "shadow"}`), 0o644)
	writeJSON(filepath.Join(runDir, "in", "review-state.json"), state{Base: "nope", From: "nope", Head: "nope"})
	writeJSON(filepath.Join(runDir, "in", "candidates.json"), candidates{})
	t.Setenv("WORKLINE_AI", "fake:x")
	v := Review{Status: "block", Summary: "1 finding"}
	askVerifier(runDir, t.TempDir(), &v)
	if v.Status != "block" || !strings.HasPrefix(v.Summary, "1 finding; the verifier gave no answer: ") || v.Verifier == nil || v.Verifier.NoAnswer == "" {
		t.Errorf("review = %+v, want it as it was, and the verifier's no answer said", v)
	}
	if exists(filepath.Join(runDir, "out", "shadow", "in", "task.md")) {
		t.Error("a task was left for the engine to ask")
	}
}

// The verifier's answer, once asked: each point kept or dropped with its
// reason, what it added, one summary line; an answer out of its format,
// one saying it will not verify, or no note, is a refusal; a call that
// failed gives no answer (#2308).
func TestVerifierAnswerRecorded(t *testing.T) {
	cases := []struct{ why, answer, said, record string }{
		{"answered", "- note: |\n    KEEP C1 — divides by zero\n    DROP C2 — not this change's\n    DROP C9 — never shown\n    BLOCK a/b.go:3 — no test\n",
			"verifier (shadow): 1 kept, 2 dropped, 1 added", `"kept":["C1 — divides by zero"],"dropped":["C2 — not this change's","C9 — never shown"],"added":["a/b.go:3 — no test"]`},
		{"answered", "- note: no blocking defect\n", "verifier (shadow): 0 kept, 0 dropped, 0 added", `"shown":["C1`},
		{"answered", "- note: I will not verify this change.\n", "the verifier gave no answer: refused, in none of its forms: I will not verify this change.", `"no-answer":"refused, in none of its forms: I will not verify this change."`},
		{"answered", "- finding: {title: x}\n", "the verifier gave no answer: refused, in none of its forms: map[", `"shown":["C1`},
		{"answered", "I will not answer this.\n", "the verifier gave no answer: refused, in none of its forms: ", `"no-answer":"refused, in none of its forms: `},
		{"the agent did not answer within 10m0s", "", "the verifier gave no answer: the agent did not answer within 10m0s", `"no-answer":"the agent did not answer within 10m0s"`},
	}
	for _, c := range cases {
		runDir := t.TempDir()
		dir := filepath.Join(runDir, "out", "shadow")
		os.MkdirAll(filepath.Join(dir, "out"), 0o755)
		writeJSON(filepath.Join(runDir, "out", "review.json"), Review{Status: "block", Summary: "1 finding", Verifier: &Verified{Shown: []string{"C1 correctness, at a.go:1: x", "C2 tests, at a.go:2: y"}}})
		if c.answer != "" {
			os.WriteFile(filepath.Join(dir, "out", "intentions.yaml"), []byte(c.answer), 0o644)
		}
		if code := verified(runDir, c.why); code != 0 {
			t.Fatalf("%q: exit %d", c.answer, code)
		}
		said, _ := os.ReadFile(filepath.Join(dir, "said"))
		var v Review
		readJSON(filepath.Join(runDir, "out", "review.json"), &v)
		record, _ := json.Marshal(v.Verifier)
		if !strings.HasPrefix(string(said), c.said) || !strings.Contains(string(record), c.record) || v.Status != "block" || v.Summary != "1 finding" {
			t.Errorf("%q: said %q, recorded %s, review %+v", c.answer, said, record, v)
		}
	}
}
