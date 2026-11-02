package tester

// A bug reproduced first (ADR-0060, decision 4): its tests must fail on
// the base, which removes any doubt that the bug is there, before any
// code fixes it. A bug whose every test passes on the base is not
// reproduced: no Draft, the issue keeps its state — never ready —,
// labelled workline:needs-repro, and its reporter is asked how to
// reproduce it, in the tester's one comment. Past the product owner's
// delay with no answer, the line blocks it, "not reproduced" (ADR-0059).
// Never another kind: a bug not shown is not turned into a feature.

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// notShown says whether a refused proof is a bug not reproduced: the
// issue is a bug, and every blocking finding is a test green on the base.
func notShown(t task, proof verdict.Verdict) bool {
	if kind, _ := backlog.KindOf(t.Labels); kind != backlog.KindBug {
		return false
	}
	green := 0
	for _, f := range proof.Findings {
		switch {
		case f.Level != "":
		case f.Rule == "test-green-on-base":
			green++
		default:
			return false
		}
	}
	return green > 0
}

// unreproduced ends the run of a bug its tests do not show on the base:
// no Draft, the mark workline:needs-repro put on, the reporter asked in
// the tester's one comment; the run passes, the issue in its state.
func unreproduced(runDir string, t task, intents []intent.Intention, proof verdict.Verdict) int {
	var green []string
	for _, f := range proof.Findings {
		if f.Rule == "test-green-on-base" {
			green = append(green, "`"+f.Where+"`")
		}
	}
	who := "its reporter"
	if t.Author != "" && !backlog.Bot(t.Author) {
		who = "@" + t.Author
	}
	why := fmt.Sprintf("each test written from its scenarios passes on the base, %s (%s): the bug does not show there, and a fix with no test that failed first proves nothing",
		short(t.Base), strings.Join(green, ", "))
	next := who + ", how do you reproduce it? The steps, the version and what you see, in a comment or in its steps. " +
		"With no answer within the product owner's delay (acts.ask.wait), the line blocks it as " + backlog.NotReproduced + "; it is never turned into a feature"
	body := backlog.Said("The tester", "could not reproduce this bug", why, next)
	label := map[string]any{"issue": t.Issue, "add": []string{backlog.LabelNeedsRepro}}
	if slices.Contains(t.Labels, backlog.LabelNeedsRepro) {
		label = nil // asked already: the mark's date, the delay's start, kept
	}
	intents = slices.DeleteFunc(intents, func(i intent.Intention) bool { return i.Kind == "note" || i.Kind == "patch" })
	if label != nil {
		intents = append(intents, intent.Intention{Kind: "label", Value: label})
	}
	intents = append(intents, intent.Intention{Kind: "comment", Value: map[string]any{"issue": t.Issue, "sticky": "issue", "body": body}})
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents); err != nil {
		return fail(err)
	}
	return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "a bug not reproduced: its tests pass on the base; its reporter asked, no Draft",
		Findings: []verdict.Finding{{Rule: "bug-not-reproduced", Level: "warn", Where: fmt.Sprintf("#%d", t.Issue), Message: why}}})
}

// bugProof says why a proof kind does not fit a bug: only new tests, red
// on the base, show it (ADR-0060); "" when the issue is no bug, or the
// kind is none (no test possible: back to the product owner).
func bugProof(t task, p Proof) string {
	if kind, _ := backlog.KindOf(t.Labels); kind != backlog.KindBug || p.Kind == KindNone || p.Kind == KindNewTests {
		return ""
	}
	return fmt.Sprintf("#%d is a bug: only a test that fails on the base shows it, never %s; write that test, or say none is possible", t.Issue, p.Kind)
}
