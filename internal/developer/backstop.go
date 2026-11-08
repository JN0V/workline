package developer

// The per-issue backstop on a review round (ADR-0064, §5, the panel of
// 2026-11-08): an issue whose runs reached its backstop gets no round
// more. The round is not run, said on its pull request, kept open, and
// its issue goes to the blocked state for a person, as past the last
// round (handOff).

import (
	"fmt"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/routing"
)

// Backstop says why issue's runs reached its backstop, what a person
// does about it; "" when they did not. Set by the engine's command to
// the line's reading (line.BackstopSaid), which keeps the record and the
// normals; nil, never checked.
var Backstop func(f forge.Forge, repo string, issue int) (string, error)

// BranchIssue is the issue a developer's branch is for, 0 for another's.
func BranchIssue(branch string) int { return developerIssue(branch) }

// stopped is what a round its issue's backstop stops writes: the issue
// blocked, why on it and on the pull request. No round is counted.
func stopped(r round, repo string, issue int) []intent.Intention {
	add, remove := []string{backlog.LabelBlocked}, []string{backlog.LabelReady}
	if line, err := routing.Load(repo); err == nil {
		if ad, rm, err := line.Move("blocked"); err == nil {
			add, remove = ad, rm
		}
	}
	return []intent.Intention{
		{Kind: "label", Value: map[string]any{"issue": issue, "add": add, "remove": remove, "card": "blocked"}},
		sayOn(issue, backlog.Said(who, "blocks this issue", r.Backstop,
			fmt.Sprintf("no review round runs on merge request %d, kept open; no product owner reads it again", r.MR))),
		{Kind: "comment", Value: map[string]any{"sticky": "round", "body": fmt.Sprintf(
			"**Blocked at its backstop**: no review round runs. #%d is blocked, its reason on it; this pull request stays open.\n\n%s.", issue, r.Backstop)}},
	}
}
