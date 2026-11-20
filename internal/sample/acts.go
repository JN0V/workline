package sample

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// The product owner's acts in the weekly sample of every role's (ADR-0046,
// folding in ADR-0033's issue of its own): drawn one in ten per kind like
// any role's, each saying when a person undid it, and the level its acts
// done alone suggest, said in the week's comment and never set. Its former
// tracking issue, still open, is told where the weeks went, and closed.

// productOwner is the role whose acts and level are read.
const productOwner = "product-owner"

// formerActsTitle is the product owner's own tracking issue, before it
// was folded into every role's: closed, never opened again.
const formerActsTitle = "workline: the weekly sample of the product owner's acts"

// Autonomy is what the product owner's acts done alone suggest of its
// level (ADR-0033): over those at the level of its last act, up to the
// window's end — how many, how many a person undid, the level they
// suggest and why.
type Autonomy struct {
	Level   string `json:"level"`
	Counted int    `json:"counted"`
	Undone  int    `json:"undone"`
	Until   string `json:"until"` // the last day counted, YYYY-MM-DD
	Suggest string `json:"suggest,omitempty"`
	Why     string `json:"why,omitempty"`
}

// actsWords counts acts done alone: "1 act done alone".
func actsWords(n int) string { return plural(n, "act") + " done alone" }

// autonomyLines say the product owner's level, for the week's comment.
func autonomyLines(au *Autonomy) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n**The product owner's level**: %s at %s in the acts the issues keep, %d undone, up to %s. ", actsWords(au.Counted), au.Level, au.Undone, au.Until)
	switch {
	case au.Suggest != "":
		fmt.Fprintf(&b, "**Suggested**: `autonomy: %s` — %s. Set it in the project's settings if you agree; the sample never changes it.\n", au.Suggest, au.Why)
	case au.Level == backlog.Cautious:
		b.WriteString("At cautious, the role does alone only what checks facts: nothing to suggest a level from.\n")
	case au.Counted < backlog.SuggestAfter:
		fmt.Fprintf(&b, "Too few acts done alone at %s to suggest a level: %d, at least %d.\n", au.Level, au.Counted, backlog.SuggestAfter)
	default:
		b.WriteString("No other level suggested.\n")
	}
	return b.String()
}

// autonomyFinding says the level suggested, when one is.
func autonomyFinding(au *Autonomy) []verdict.Finding {
	if au == nil || au.Suggest == "" {
		return nil
	}
	return []verdict.Finding{{Rule: "acts-suggest", Level: "info",
		Message: fmt.Sprintf("the product owner's autonomy: %s suggested — %s; the setting is a person's", au.Suggest, au.Why)}}
}

// olderWords say the product owner's acts read without the record
// ADR-0046 completes: as their issue reads now.
func olderWords(refs []string) string {
	if len(refs) == 1 {
		return fmt.Sprintf("the product owner's act on %s, read on its issue as it reads now and the last commit of its day, the model that acted unknown: an engine before ADR-0046 kept no `at`, `model` nor `body` in its record", refs[0])
	}
	return fmt.Sprintf("the product owner's acts on %s, read on their issues as they read now and the last commit of their day, the model that acted unknown: an engine before ADR-0046 kept no `at`, `model` nor `body` in their record", strings.Join(refs, ", "))
}

// foldedSays is what the product owner's former tracking issue is told.
const foldedSays = "The product owner's acts are folded into #%d, the weekly sample of every role's acts (ADR-0046): " +
	"drawn there each week, one in ten of each kind, each re-checked by the auditor and saying when a person undid it, " +
	"with the level they suggest. This issue gets no more weeks.\n\nWritten by `workline sample --apply`, with no AI."

// foldActs closes the product owner's former tracking issues still open,
// each told once where the weeks went (ADR-0046).
func foldActs(res *Result, f forge.Forge, former []int, id int) error {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil
	}
	for _, n := range former {
		if err := f.Comment(forge.Target{Kind: "issue", ID: n}, fmt.Sprintf(foldedSays, id), forge.Marker("sample-folded")); err != nil {
			return err
		}
		if err := b.Close(n, 0); err != nil {
			return err
		}
		res.Applied = append(res.Applied, "close")
		res.Findings = append(res.Findings, verdict.Finding{Rule: "sample-folded", Level: "info", Where: fmt.Sprintf("#%d", n),
			Message: fmt.Sprintf("the product owner's own weekly issue, folded into #%d: told so, and closed", id)})
	}
	return nil
}

// ownDraw are the roles that still have a draw of their own: the shared
// draw skips them, so no act is drawn twice. A role leaves the list when
// its code moves to the shared record.
var ownDraw = []string{docsRole, productOwner, reviewRole, commitRole, testRole, devRole}

// recordedActs are the acts of the days [first, end) every role without a
// draw of its own recorded on the issues, open or closed, oldest first:
// one draw for any role, read and never written.
func recordedActs(f forge.Forge, b forge.Backlog, first, end string) ([]Act, []string, error) {
	all, err := b.AllIssues()
	if err != nil {
		return nil, nil, err
	}
	var out []Act
	var unread []string
	for _, is := range all {
		ref := fmt.Sprintf("#%d", is.ID)
		kept, err := forge.ReadKept(f, forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			unread = append(unread, fmt.Sprintf("the acts recorded on %s: the issue could not be read: %v", ref, err))
			continue
		}
		for i, r := range engine.ReadActs(kept, forge.Target{Kind: "issue", ID: is.ID}) {
			if r.Role == "" || r.Kind == "" || slices.Contains(ownDraw, r.Role) {
				continue
			}
			if r.Day < first || (end != "" && r.Day >= end) {
				continue
			}
			var line string
			if len(r.Items) > 0 {
				line = fmt.Sprintf("read %d items", len(r.Items))
			}
			out = append(out, Act{Role: r.Role, Kind: r.Kind, Ref: ref, Day: r.Day, By: strings.Join(r.Models, ", "), Line: line, Commit: r.Commit,
				key: fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d", ref, r.Role, r.Kind, r.Day, i)})
		}
	}
	return out, unread, nil
}
