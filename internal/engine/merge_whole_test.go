package engine

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// A review not whole is named alone only when no agent was missing nor
// failed: a person's reading takes that one only (#1107).
func TestShortNamesNotWholeLast(t *testing.T) {
	v := HeadVerdict{Status: verdict.Pass, Findings: []verdict.Finding{
		{Rule: notWhole, Message: "past parts-max"},
		{Rule: "lens-failed", Message: "the agent failed"},
	}}
	if got := shortRule(v.short()); got != "lens-failed" {
		t.Errorf("shortRule = %q, want lens-failed", got)
	}
	v.Findings = v.Findings[:1]
	if got := shortRule(v.short()); got != notWhole {
		t.Errorf("shortRule = %q, want %s", got, notWhole)
	}
	if got := shortRule("the line passed, every agent's review whole"); got != "" {
		t.Errorf("shortRule of a pass = %q, want none", got)
	}
}

// Each label goes by its own events: the whole yes taken off leaves the
// plain one on, and a non-writer's label is no yes.
func TestTheYesEachLabelItsOwn(t *testing.T) {
	dana := func(label string, added bool, at string) forge.LabelEvent {
		return forge.LabelEvent{Label: label, Added: added, Writer: true, Note: forge.Note{Author: "dana", Created: at}}
	}
	events := []forge.LabelEvent{
		dana(labelAcceptedWhole, true, "t1"),
		dana("workline:accepted", true, "t2"),
		dana(labelAcceptedWhole, false, "t3"),
	}
	yes, _ := theYes(events, accepted)
	if yes == nil || yes.Label != "workline:accepted" {
		t.Fatalf("yes = %+v, want the plain label still on", yes)
	}
	if whole, _ := theYes(events, acceptedWhole); whole != nil {
		t.Errorf("whole = %+v, want none: taken off", whole)
	}
	zed := forge.LabelEvent{Label: labelAcceptedWhole, Added: true, Note: forge.Note{Author: "zed", Created: "t4"}}
	whole, ignored := theYes(append(events, zed), acceptedWhole)
	if whole != nil || ignored == nil || ignored.Author != "zed" {
		t.Errorf("whole, ignored = %+v, %+v: want no yes, zed's ignored", whole, ignored)
	}
}
