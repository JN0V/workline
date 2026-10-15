package engine

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// annotationsSummary is what the check says above its annotations.
const annotationsSummary = "The documentalist's findings on this head, each beside its doc's line. " +
	"None is a conversation to resolve, none holds the merge: fix the doc, or leave it. " +
	"The judge's job summary lists them all."

// annotateFix is what to give a token whose check was refused.
const annotateFix = "give the applying job's token `checks: write` (ci/github/workline-apply.yml), " +
	"and a GitHub App the permission Checks: read and write"

// Annotate shows the documentalist's findings of a merge request's own run
// on the head it judged, as the annotations of the check FindingsCheck,
// which open no conversation (#412). build makes them from the files as
// that head holds them, its branch fetched when the commit is not here.
// The run's findings are data its code could have written: only their
// places and words are shown, nothing is applied. What became of them is a
// finding, never an error: on a forge that takes no check (GitLab, whose
// Code Quality report shows them), nothing.
func Annotate(from FromMergeRequest, build func(read func(path string) ([]byte, error)) ([]forge.Annotation, int)) []verdict.Finding {
	not := func(why string) []verdict.Finding {
		return []verdict.Finding{{Rule: "findings-not-annotated", Where: forge.FindingsCheck, Level: "warn",
			Message: why + "; the findings stay in the judge's job summary"}}
	}
	f, err := forge.Open(from.Forge, from.Repo)
	if err != nil {
		return not(err.Error())
	}
	a, ok := f.(forge.Annotator)
	if !ok {
		return nil
	}
	if _, err := git(from.Repo, nil, "cat-file", "-e", from.Head+"^{commit}"); err != nil {
		mr, err := f.MergeRequest(from.ID)
		if err != nil {
			return not(forge.Said(err))
		}
		if !mr.Here {
			return not(fmt.Sprintf("merge request #%d comes from a fork, whose head is not fetched here", from.ID))
		}
		if _, err := fetchTip(from.Repo, mr.Branch, forge.KeepsBranches(f)); err != nil {
			return not(forge.Said(err))
		}
	}
	notes, left := build(func(path string) ([]byte, error) {
		return exec.Command("git", "-C", from.Repo, "cat-file", "blob", from.Head+":"+path).Output()
	})
	if len(notes) == 0 {
		return nil
	}
	summary := annotationsSummary
	if left > 0 {
		summary += fmt.Sprintf(" %d more are not written here.", left)
	}
	url, err := a.Annotate(from.Head, forge.FindingsCheck, summary, notes)
	if forge.Refused(err) {
		why := forge.Said(err)
		if !strings.Contains(why, "checks: write") {
			why += "; " + annotateFix
		}
		return not(why)
	}
	if err != nil {
		return not(forge.Said(err))
	}
	where := "on " + short(from.Head)
	if url != "" {
		where = url
	}
	n := len(notes) + left
	return []verdict.Finding{{Rule: "annotated", Where: forge.FindingsCheck, Level: "warn",
		Message: fmt.Sprintf("%d %s of the documentalist, beside their lines: %s", n, map[bool]string{true: "finding", false: "findings"}[n == 1], where)}}
}
