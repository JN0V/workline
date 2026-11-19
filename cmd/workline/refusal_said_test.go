package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// refusalForge is a fake forge holding one issue, and its state file.
func refusalForge(t *testing.T) string {
	t.Helper()
	state := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(state, []byte(`{"issues":[{"id":1,"title":"A need","labels":[],"comments":[]}],"merge-requests":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return state
}

// issueComments are the comments the fake forge keeps on its issue 1.
func issueComments(t *testing.T, state string) []string {
	t.Helper()
	data, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Issues []struct {
			Comments []json.RawMessage `json:"comments"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(data, &s); err != nil || len(s.Issues) != 1 {
		t.Fatalf("forge file unreadable: %v\n%s", err, data)
	}
	var out []string
	for _, c := range s.Issues[0].Comments {
		out = append(out, string(c))
	}
	return out
}

// Said on the item
// Given a role run that refuses or blocks, for a reason that is neither a
// config that cannot be read nor an engine error (a frozen test edited),
// When it concerns an issue,
// Then that issue carries one comment from that role saying what was
// refused and why.
func TestEveryRefusalIsSaidOnItsItem(t *testing.T) {
	state := refusalForge(t)
	res := &engine.Result{Status: verdict.Block, Summary: "the round is refused",
		Findings: []verdict.Finding{{Rule: "acceptance-test-edit", Message: "a frozen test is edited"}}}
	commentRefusal("developer", "fake:"+state, t.TempDir(), &forge.Target{Kind: "issue", ID: 1}, res)
	got := issueComments(t, state)
	if len(got) != 1 {
		t.Fatalf("the issue carries %d comments, want exactly 1: %q", len(got), got)
	}
	if !strings.Contains(got[0], "a frozen test is edited") || !strings.Contains(got[0], "developer") {
		t.Errorf("the comment does not say the role and what was refused and why: %s", got[0])
	}
}

// Updated in place
// Given the comment a role's refusal left on its issue,
// When the same role refuses again, for another reason,
// Then that comment is updated and no other is added.
func TestARefusedRoleKeepsOneComment(t *testing.T) {
	state := refusalForge(t)
	target := &forge.Target{Kind: "issue", ID: 1}
	first := &engine.Result{Status: verdict.Block, Summary: "refused",
		Findings: []verdict.Finding{{Rule: "acceptance-test-edit", Message: "a frozen test is edited"}}}
	second := &engine.Result{Status: verdict.Block, Summary: "refused again",
		Findings: []verdict.Finding{{Rule: "steering-file", Message: "a steering file is touched"}}}
	commentRefusal("developer", "fake:"+state, t.TempDir(), target, first)
	commentRefusal("developer", "fake:"+state, t.TempDir(), target, second)
	got := issueComments(t, state)
	if len(got) != 1 {
		t.Fatalf("the issue carries %d comments, want exactly 1: %q", len(got), got)
	}
	if !strings.Contains(got[0], "a steering file is touched") || strings.Contains(got[0], "a frozen test is edited") {
		t.Errorf("the comment is not updated to the second refusal: %s", got[0])
	}
}
