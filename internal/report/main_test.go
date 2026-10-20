package report

import (
	"fmt"
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

// The count by kind: the most first, alike in the order first found, main
// red said first; past mainShown entries, the rest counted.
func TestMainIssueCountsByKindAndCaps(t *testing.T) {
	e := func(rule string) mainEntry { return mainEntry{Rule: rule} }
	got := byKind([]mainEntry{e("lint/godot")}, []mainEntry{e("dead-link"), e("doc-too-long"), e("doc-too-long"), e("dead-link"), e("doc-too-long")}, true)
	if want := "main's run red, 3 `doc-too-long`, 2 `dead-link`, 1 `lint/godot`"; got != want {
		t.Errorf("byKind = %q, want %q", got, want)
	}
	if got := byKind(nil, []mainEntry{e("b"), e("a")}, false); got != "1 `b`, 1 `a`" {
		t.Errorf("a tie keeps the order found: %q", got)
	}
	if got := byKind(nil, nil, true); got != "main's run red" {
		t.Errorf("red with no finding listed: %q", got)
	}
	var findings []verdict.Finding
	for i := range mainShown + 5 {
		findings = append(findings, verdict.Finding{Rule: "doc-too-long", Where: fmt.Sprintf("docs/d%d.md", i), Message: "too long"})
	}
	r := &line.Result{Event: line.MainEvent, Main: &line.MainRun{Head: "h"}, Steps: []line.Step{{Name: "documentalist", Status: verdict.Pass,
		Result: &engine.Result{Findings: findings}}}}
	body, _, _ := MainIssue(r, "")
	if n := strings.Count(body, "\n- `docs/d"); n != mainShown || !strings.Contains(body, "- and 5 more, not listed here") ||
		!strings.HasPrefix(body, fmt.Sprintf("**%d findings on the default branch**: %d `doc-too-long`.", mainShown+5, mainShown+5)) {
		t.Errorf("listed %d of %d, the rest not counted:\n%s", n, mainShown+5, body[:min(len(body), 300)])
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
