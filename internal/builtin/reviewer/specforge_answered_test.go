package reviewer

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// specRound is one read of issue 1's spec as the line does it: the spec the
// local forge hands the reviewer, the findings the lens raised settled, and
// the comment the reviewer writes on the issue.
func specRound(t *testing.T, repo string, s Settings, found []Finding) (state, Review, string) {
	t.Helper()
	q, _, ok := specOnForge(t.TempDir(), repo, s)
	if !ok || len(q.read) != 1 {
		t.Fatalf("the reviewer reads %d specs (asked: %v), want issue 1's alone", len(q.read), ok)
	}
	sp := q.read[0]
	st := state{Head: "h", From: "h", Base: "h", Lenses: []string{"scenarios"}, Spec: &sp}
	v, fallback, err := settled(t.TempDir(), repo, s, st, candidates{Related: found})
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range fallback {
		if m, isMap := in.Value.(map[string]any); isMap && in.Kind == "comment" && m["sticky"] == backlog.SpecKey {
			body, _ := m["body"].(string)
			return st, v, body
		}
	}
	t.Fatalf("the review writes no comment on the spec: %v", fallback)
	return st, v, ""
}

// Answer read as settled:
// Given a reviewer's finding on a scenario, answered by a person and
// written into the spec (in its Scope),
// When the reviewer reads it again and its lens raises the same finding
// (same scenario name, same lens),
// Then it does not hold for that finding: no important finding open, the
// issue held no more.
func TestAnswerWrittenIntoTheScopeSettlesTheFinding(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	// The local forge refuses every write in CI: a CI runner's own variables
	// never decide this test.
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("CI", "")
	t.Setenv("GITLAB_CI", "")
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	t.Setenv("WORKLINE_FORGE", "local")
	t.Setenv("WORKLINE_EVENT", "issue")
	t.Setenv("WORKLINE_TARGET", "issue:1")

	validation := "## Validation\n\n- **Two by three:** Given 2 and 3, When multiplied, Then 6.\n- **Empty list:** Given nothing, When multiplied, Then 1.\n\n"
	head := "## Need\n\nA calculator multiplies.\n\n## Verification\n\nThe product is right.\n\n"
	before := head + validation + "## Scope\n\ncalc/calc.go\n"
	after := head + validation + "## Scope\n\ncalc/calc.go\n\nTwo by three: only whole numbers are multiplied, a failing case is not asked here.\n"

	l := &forge.Local{Repo: repo}
	if id, err := l.Open("Mean", before, nil); err != nil || id != 1 {
		t.Fatalf("open issue: %d, %v", id, err)
	}
	s := Settings{SpecsPerRun: 5, SpecRounds: 3, ForgeWrites: true}
	finding := Finding{Lens: "scenarios", Severity: "important",
		Title: "Scenario “Two by three” names no failing case", Why: "it states only the happy path",
		Cause: Quote{Path: "#1", Quote: "**Two by three:** Given 2 and 3, When multiplied, Then 6."},
		Where: "#1:11", Related: true}

	// The first review holds it.
	st1, v1, first := specRound(t, repo, s, []Finding{finding})
	if open := specRecord(st1, v1).Open; open != 1 {
		t.Fatalf("the first review leaves %d important findings open, want 1", open)
	}
	if err := l.Sticky(forge.Target{Kind: "issue", ID: 1}, first, backlog.SpecMarker, true); err != nil {
		t.Fatal(err)
	}

	// A person answers it in the Scope.
	if err := l.SetBody(1, after); err != nil {
		t.Fatal(err)
	}

	// The reviewer reads it again; its lens raises the same finding.
	st2, v2, second := specRound(t, repo, s, []Finding{finding})
	if open := specRecord(st2, v2).Open; open != 0 {
		t.Errorf("the answer written in the Scope leaves %d important findings open, want 0", open)
	}
	if strings.Contains(second, "Holds it from ready") {
		t.Errorf("the comment still holds the issue from ready:\n%s", second)
	}
	is, err := l.Issue(1)
	if err != nil {
		t.Fatal(err)
	}
	if rule, why := backlog.SpecHold(*is, []string{second + "\n\n" + backlog.SpecMarker}); rule != "" {
		t.Errorf("the spec still holds the issue: %s: %s", rule, why)
	}
}
