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
	title := fmt.Sprintf("%d %s", len(notes), map[bool]string{true: "finding", false: "findings"}[len(notes) == 1])
	url, err := a.Annotate(from.Head, forge.FindingsCheck, title, summary, notes)
	if err != nil {
		why := forge.Said(err)
		if forge.Refused(err) && !strings.Contains(why, "checks: write") {
			why += "; " + annotateFix
		}
		if url != "" { // the check written, some of its annotations not
			why = "only part of them written, " + url + ": " + why
		}
		return not(why)
	}
	where := "on " + short(from.Head)
	if url != "" {
		where = url
	}
	n := len(notes) + left
	return []verdict.Finding{{Rule: "annotated", Where: forge.FindingsCheck, Level: "warn",
		Message: fmt.Sprintf("%d %s of the documentalist, beside their lines: %s", n, map[bool]string{true: "finding", false: "findings"}[n == 1], where)}}
}

// GatesMarker keys the gates' summary comment on a merge request: one,
// edited on each push.
var GatesMarker = forge.Marker("sticky=gates/summary")

// GatesToShow is what of the gates a merge request is shown: the
// annotations and their check's words, the summary comment once the
// check's page is known, and whether there is anything to say.
type GatesToShow struct {
	Notes          []forge.Annotation
	Title, Summary string
	Comment        func(url string) string
	Worth          bool
	New            int
}

// ShowGates shows on a merge request what its own run found the change
// adds to the gates' tools (ADR-0049, step 3): the annotations of the
// check GatesCheck on the head judged, which open no conversation, then
// one summary comment, edited on each push. With nothing to say, no check
// is written, and a comment an earlier push left is edited to say so;
// none is posted. What became of them is a finding, never an error; a
// forge that takes no check (GitLab, whose Code Quality report shows
// them) is shown nothing here.
func ShowGates(from FromMergeRequest, g GatesToShow) []verdict.Finding {
	not := func(why string) verdict.Finding {
		return verdict.Finding{Rule: "gates-not-shown", Where: forge.GatesCheck, Level: "warn",
			Message: why + "; the gates' findings stay in their job's summary"}
	}
	f, err := forge.Open(from.Forge, from.Repo)
	if err != nil {
		return []verdict.Finding{not(err.Error())}
	}
	a, ok := f.(forge.Annotator)
	if !ok {
		return nil
	}
	// The merge request the run names must have the head judged as its
	// head still: the number is the run's word, and its branch moved on,
	// the run of its new head shows its own.
	mr, err := f.MergeRequest(from.ID)
	if err != nil {
		return []verdict.Finding{not(forge.Said(err))}
	}
	if !mr.Here {
		return []verdict.Finding{not(fmt.Sprintf("merge request #%d comes from a fork, whose head is not fetched here", from.ID))}
	}
	tip, err := fetchTip(from.Repo, mr.Branch, forge.KeepsBranches(f))
	if err != nil {
		return []verdict.Finding{not(forge.Said(err))}
	}
	if tip != from.Head {
		return []verdict.Finding{{Rule: "moved-on", Level: "warn", Where: forge.GatesCheck,
			Message: fmt.Sprintf("merge request #%d's branch is at %s, not at %s, the head its run judged: the gates' findings are not shown; the run of its new head shows its own", from.ID, short(tip), short(from.Head))}}
	}
	var said []verdict.Finding
	url := ""
	if g.Worth && len(g.Notes) > 0 {
		if url, err = a.Annotate(from.Head, forge.GatesCheck, g.Title, g.Summary, g.Notes); err != nil {
			why := forge.Said(err)
			if forge.Refused(err) && !strings.Contains(why, "checks: write") {
				why += "; " + annotateFix
			}
			if url != "" { // the check written, some of its annotations not
				why = "only part of them written, " + url + ": " + why
			}
			said = append(said, not(why))
		}
	}
	to := forge.Target{Kind: "merge-request", ID: from.ID}
	if err := f.Sticky(to, g.Comment(url), GatesMarker, g.Worth); err != nil {
		return append(said, not("the summary comment was not written: "+forge.Said(err)))
	}
	if !g.Worth {
		return said
	}
	where := "beside their lines on " + short(from.Head)
	if url != "" {
		where = "beside their lines, " + url
	}
	return append(said, verdict.Finding{Rule: "gates-shown", Where: forge.GatesCheck, Level: "warn",
		Message: fmt.Sprintf("%d new %s of the gates, %s; counted in the summary comment on %s", g.New,
			map[bool]string{true: "finding", false: "findings"}[g.New == 1], where, to)})
}
