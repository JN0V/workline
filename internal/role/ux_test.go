package role

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// uxInstruction reads the UX role's prompt, lowercased.
func uxInstruction(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "roles", "ux", "instruction.md"))
	if err != nil {
		t.Fatalf("the UX role's prompt: %v", err)
	}
	return strings.ToLower(string(data))
}

// Disagreeing screens: Given a planted pair of screens that name the same
// action two ways, When the UX role reads the change, Then it flags the
// disagreement, quoting both screens.
func TestDisagreeingScreens(t *testing.T) {
	r, err := Load(filepath.Join("..", "..", "roles"), "ux")
	if err != nil {
		t.Fatalf("the UX role does not load: %v", err)
	}
	if !slices.Contains(r.Events, "merge-request") {
		t.Errorf("the UX role reads a change: events %v lack merge-request", r.Events)
	}
	p := uxInstruction(t)
	for _, want := range []string{"disagree", "same action", "quote"} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt does not ask to flag screens that disagree on the same action, quoting both: lacks %q", want)
		}
	}
}

// Unclear wording: Given a planted screen with an unclear message, When
// the UX role reads the change, Then it flags the wording, quoting the
// message and the issue's Need.
func TestUnclearWording(t *testing.T) {
	p := uxInstruction(t)
	for _, want := range []string{"wording", "message", "need", "quote"} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt does not ask to flag unclear wording, quoting the message and the Need: lacks %q", want)
		}
	}
}

// Findings never override the tools: Given the accessibility checks
// passing, then failing, When the UX role adds findings, Then the gate's
// verdict is the tools', unchanged by the findings.
func TestFindingsNeverOverrideTheTools(t *testing.T) {
	r, err := Load(filepath.Join("..", "..", "roles"), "ux")
	if err != nil {
		t.Fatalf("the UX role does not load: %v", err)
	}
	for _, k := range []string{"patch", "label", "commit-message"} {
		if slices.Contains(r.Intentions, k) {
			t.Errorf("the UX role proposes %q: its findings are advisory, never changing the code or the verdict", k)
		}
	}
	if len(r.Writes(r.Settings)) != 0 {
		t.Errorf("the UX role writes %v: it only reads", r.Writes(r.Settings))
	}
	if !strings.Contains(uxInstruction(t), "advisory") {
		t.Errorf("the prompt does not say the findings are advisory, the tools' verdict standing")
	}
}

// No agent: Given a project run with no agent, When the line runs on a
// change, Then the accessibility checks run, and no reading is made.
func TestNoAgent(t *testing.T) {
	r, err := Load(filepath.Join("..", "..", "roles"), "ux")
	if err != nil {
		t.Fatalf("the UX role does not load: %v", err)
	}
	if r.WithoutAI != "pass" {
		t.Errorf("without-ai = %q, want pass: no reading, the checks' verdict stands", r.WithoutAI)
	}
}
