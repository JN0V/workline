package report

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
)

// A finding the gates found main gained stays listed while main holds
// it, read back from the issue's own marker; once gone, it leaves the
// issue. A doc's finding counted otherwise is the same finding.
func TestMainIssueKeepsWhatMainHolds(t *testing.T) {
	gateStep := func(present []string, found ...gate.Finding) line.Step {
		return line.Step{Name: "gate:merge", Status: verdict.Pass, Gate: &verdict.Verdict{Status: verdict.Pass},
			New: &gate.Added{Findings: found, Present: present}}
	}
	docs := func(n string) line.Step {
		return line.Step{Name: "documentalist", Status: verdict.Pass, Result: &engine.Result{
			Findings: []verdict.Finding{{Rule: "doc-too-long", Where: "docs/a.md", Message: n + " lines; the budget is 200"}}}}
	}
	first := &line.Result{Event: line.MainEvent, Main: &line.MainRun{Before: "aaaaaaa1", Head: "bbbbbbb2"},
		Steps: []line.Step{gateStep([]string{"fp1", "old"}, gate.Finding{Check: "lint", Rule: "godot", Path: "x.go", Line: 3, Fingerprint: "fp1"}), docs("250")}}
	body, changed, any := MainIssue(first, "")
	if !changed || !any || !strings.Contains(body, "`x.go:3` `lint/godot`") || !strings.Contains(body, "(since `bbbbbbb`)") {
		t.Fatalf("first run: changed %v, any %v:\n%s", changed, any, body)
	}
	// The count by kind on top, the lists folded.
	if !strings.HasPrefix(body, "**2 findings on the default branch**: 1 `lint/godot`, 1 `doc-too-long`.\n") ||
		strings.Count(body, "<details>") != 2 || strings.Count(body, "</details>") != 2 {
		t.Errorf("not the count by kind, then the lists folded:\n%s", body)
	}
	// The next merge adds nothing; the doc grew. Nothing new nor gone.
	next := &line.Result{Event: line.MainEvent, Main: &line.MainRun{Before: "bbbbbbb2", Head: "ccccccc3"},
		Steps: []line.Step{gateStep([]string{"fp1", "old"}), docs("260")}}
	again, changed, _ := MainIssue(next, body)
	if changed || !strings.Contains(again, "`x.go:3` `lint/godot`") || !strings.Contains(again, "(since `bbbbbbb`)") {
		t.Errorf("nothing new: changed %v:\n%s", changed, again)
	}
	// Fixed on main: gone from what it lists.
	fixed := &line.Result{Event: line.MainEvent, Main: &line.MainRun{Before: "ccccccc3", Head: "ddddddd4"},
		Steps: []line.Step{gateStep([]string{"old"}), docs("260")}}
	gone, changed, any := MainIssue(fixed, again)
	if !changed || !any || strings.Contains(gone, "godot") {
		t.Errorf("fixed: changed %v, any %v:\n%s", changed, any, gone)
	}
}

// What a run wrote cannot close the marker it is read back from.
func TestMainIssueMarkerHoldsNoCommentEnd(t *testing.T) {
	r := &line.Result{Event: line.MainEvent, Main: &line.MainRun{Head: "h"}, Steps: []line.Step{{Name: "gate:merge",
		Status: verdict.Pass, Gate: &verdict.Verdict{Status: verdict.Pass},
		New: &gate.Added{Findings: []gate.Finding{{Check: "lint", Message: "a --> b <!-- c", Fingerprint: "f"}}, Present: []string{"f"}}}}}
	body, _, _ := MainIssue(r, "")
	_, marker, _ := strings.Cut(body, mainMarker)
	if strings.Count(marker, "-->") != 1 || readMainState(body).Gates[0].Message != "a --> b <!-- c" {
		t.Errorf("marker %q", marker)
	}
}
