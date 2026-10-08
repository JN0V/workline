package sample

import (
	"fmt"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
)

// The product owner's acts in the weekly sample (ADR-0033), when the
// project turns it on (the product owner's `weekly-sample`, off by
// default since ADR-0038): one in ten of those it did alone in the week,
// drawn from each issue's state, as the docs are, and written on a
// tracking issue of their own for a person to judge — an act found wrong
// is undone on its issue, which has that kind proposed there. Drawn with
// the forge and no AI, in the job holding the forge's token: --apply.

// ActsIssueTitle is the tracking issue of the product owner's acts.
const ActsIssueTitle = "workline: the weekly sample of the product owner's acts"

// productOwner is the role whose acts are sampled.
const productOwner = "product-owner"

const actsIntro = `Each week, one in ten of the acts the product owner did alone — closing,
renaming, splitting, moving to ` + "`ready`" + `, ordering… — is drawn from the record on
each issue's state, with the day and the autonomy level each was done at,
for a person to judge (ADR-0033). An act you find wrong: undo it on its
issue — reopen it, rename it back, take the label off — and the role
proposes that kind of act on that issue from its next run (ADR-0038).

A comment a week, below, with the level the acts done alone suggest. The
sample never changes the setting.

Written by ` + "`workline sample --apply`" + `, with no AI.`

// ActsRead is what the sample drew of the product owner's acts.
type ActsRead struct {
	Issue int     `json:"issue"` // the tracking issue written
	Level string  `json:"level"` // the level of its last act done alone
	Week  int     `json:"week"`  // the acts done alone in the week
	Drawn []Drawn `json:"drawn"`
	// Counted are the acts done alone at Level the issues keep, up to the
	// week's end; Undone, those a person undid.
	Counted int    `json:"counted"`
	Undone  int    `json:"undone"`
	Suggest string `json:"suggest,omitempty"`
	Why     string `json:"why,omitempty"`
}

// Drawn is one act drawn, and whether a person undid it.
type Drawn struct {
	backlog.Did
	Undone string `json:"undone,omitempty"` // what shows it undone
}

// applyActs draws the week's acts of the product owner and writes them on
// their tracking issue, when the project turned it on; nothing when no
// issue keeps an act of the role's.
func applyActs(res *Result, repo string, f forge.Forge) error {
	b, ok := f.(forge.Backlog)
	if !ok || !actsOn(repo) {
		return nil // a forge with no issues has no acts either
	}
	acts, err := backlog.ReadActs(b, productOwner)
	if err != nil || acts == nil {
		return err
	}
	_, from, to, err := Week(res.Week, time.Time{})
	if err != nil {
		return err
	}
	first, end := from.Format("2006-01-02"), to.Format("2006-01-02")
	r := &ActsRead{Level: acts.Level, Drawn: []Drawn{}}
	var week []int // the places in the record of the week's acts
	var keys []string
	for i, d := range acts.Did {
		if d.Day >= end {
			continue
		}
		if d.Level == acts.Level {
			r.Counted++
			if _, undone := acts.Undone(i); undone {
				r.Undone++
			}
		}
		if d.Day >= first {
			week = append(week, i)
			keys = append(keys, fmt.Sprintf("%d\x00%s\x00%s", d.Issue, d.Act, d.Day))
		}
	}
	r.Week = len(week)
	for _, k := range pick(res.Week, keys) {
		why, _ := acts.Undone(week[k])
		r.Drawn = append(r.Drawn, Drawn{Did: acts.Did[week[k]], Undone: why})
	}
	r.Suggest, r.Why = backlog.SuggestFromActs(r.Level, r.Counted, r.Undone)
	id, err := f.KeepIssue(ActsIssueTitle, actsIntro, true)
	if err != nil {
		return err
	}
	r.Issue = id
	if err := f.Sticky(forge.Target{Kind: "issue", ID: id}, actsReport(res.Week, first, to.AddDate(0, 0, -1).Format("2006-01-02"), r), forge.Marker("sample-acts:"+res.Week), true); err != nil {
		return err
	}
	res.Acts = r
	res.Applied = append(res.Applied, "comment")
	res.Findings = append(res.Findings, verdict.Finding{Rule: "acts-sample", Level: "info", Where: fmt.Sprintf("#%d", id),
		Message: fmt.Sprintf("%s by the product owner in %s; %d drawn, for a person to judge", actsWords(r.Week), res.Week, len(r.Drawn))})
	if r.Suggest != "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "acts-suggest", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: fmt.Sprintf("autonomy: %s suggested — %s; the setting is a person's", r.Suggest, r.Why)})
	}
	return nil
}

// actsWords counts acts done alone: "1 act done alone".
func actsWords(n int) string { return plural(n, "act") + " done alone" }

// actsReport is the week's comment on the acts' tracking issue.
func actsReport(week, first, last string, r *ActsRead) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n\n", week)
	if r.Week == 0 {
		fmt.Fprintf(&b, "No act done alone by the product owner from %s to %s: nothing to judge.\n", first, last)
	} else {
		fmt.Fprintf(&b, "%s by the product owner from %s to %s; %d drawn, one in ten, for a person to judge. An act you find wrong: undo it on its issue, and the role puts that kind back to propose at its next run.\n\n", actsWords(r.Week), first, last, len(r.Drawn))
		b.WriteString("| Issue | Act | Day | Level | Now |\n|---|---|---|---|---|\n")
		for _, d := range r.Drawn {
			now := "standing"
			if d.Undone != "" {
				now = "**undone**: " + strings.ReplaceAll(d.Undone, "|", `\|`)
			}
			fmt.Fprintf(&b, "| #%d | %s | %s | %s | %s |\n", d.Issue, d.Act, d.Day, d.Level, now)
		}
		b.WriteString("\n")
		for _, d := range r.Drawn {
			if d.Line != "" {
				fmt.Fprintf(&b, "- #%d: %s\n", d.Issue, d.Line)
			}
		}
	}
	fmt.Fprintf(&b, "\n%s at %s in the acts the issues keep, %d undone, up to %s. ", actsWords(r.Counted), r.Level, r.Undone, last)
	switch {
	case r.Suggest != "":
		fmt.Fprintf(&b, "**Suggested**: `autonomy: %s` — %s. Set it in the project's settings if you agree; the sample never changes it.\n", r.Suggest, r.Why)
	case r.Level == backlog.Cautious:
		b.WriteString("At cautious, the role does alone only what checks facts: nothing to suggest a level from.\n")
	case r.Counted < backlog.SuggestAfter:
		fmt.Fprintf(&b, "Too few acts done alone at %s to suggest a level: %d, at least %d.\n", r.Level, r.Counted, backlog.SuggestAfter)
	default:
		fmt.Fprintf(&b, "No other level suggested.\n")
	}
	return b.String()
}

// actsOn says whether the project turned the sample of the product
// owner's acts on: its `weekly-sample` setting (ADR-0038: off by default).
func actsOn(repo string) bool {
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil || cfg == nil {
		return false
	}
	on, _ := cfg.Roles[productOwner].Settings["weekly-sample"].(bool)
	return on
}
