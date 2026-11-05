package engine

import (
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// The short rules are kept with the verdict, every one: a person's
// reading takes a review not whole only when it is the one short rule, and
// never from a verdict an earlier workline kept, which named its first
// only (#1107).
func TestShortRulesKeptWhole(t *testing.T) {
	v := HeadVerdict{Status: verdict.Pass, Findings: []verdict.Finding{
		{Rule: notWhole, Message: "past parts-max"},
		{Rule: "lens-failed", Message: "the agent failed"},
	}}
	if got := v.shortRules(); !slices.Equal(got, []string{"lens-failed", notWhole}) {
		t.Errorf("shortRules = %q, want lens-failed and %s", got, notWhole)
	}
	_, why := v.class()
	rules, back := readRules(keptWhy(v.shortRules(), why))
	if !slices.Equal(rules, []string{"lens-failed", notWhole}) || back != why {
		t.Errorf("read back %q, %q; want both rules, %q", rules, back, why)
	}
	if !strings.HasPrefix(why, shortSaid+"(lens-failed)") {
		t.Errorf("why = %q, want lens-failed named first", why)
	}
	// Kept by an earlier workline: its text names the first rule only.
	rules, back = readRules("its review falls short (review-not-whole): past parts-max")
	if rules != nil || back != "its review falls short (review-not-whole): past parts-max" {
		t.Errorf("an earlier verdict read %q, %q; want no rules, its text", rules, back)
	}
	if got := keptWhy(nil, "the line passed"); got != "the line passed" {
		t.Errorf("a pass kept as %q, want its text alone", got)
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
