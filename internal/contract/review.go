package contract

import (
	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// Review is what the engine and other roles ask of the role that reviews
// a change: the reviewer's.
type Review struct {
	// ReviewedAll says whether the record a summary comment holds covers
	// every commit base..head, by its id or its change's patch id (#1254).
	ReviewedAll func(repo, base, head, summary string) (bool, error)
	// RulesOn runs the reviewer's rules, no agent, on what head adds to
	// base in repo, as the project sets them: the findings that block.
	// Another role runs them on what it writes before the review (#1040).
	RulesOn func(repo, base, head string) ([]verdict.Finding, error)
	// ServeReads serves reads a judge asked, as a judge of the code's are:
	// the text served, the lines it holds, each read refused with why.
	ServeReads func(repo, rev, cause string, reads []judge.Read, left, linesMax int) (served string, lines int, refused []string)
}
