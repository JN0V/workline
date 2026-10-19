package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/release"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
)

// The release's docs at night (ADR-0050, §7). On each move of a release
// tool's merge request, a role holding the release runs with no agent: what
// needs none still holds the release (AIWaitsNight). Its agent judges the
// docs due once a night, with the gardening, on that merge request's head,
// or when a person asks: `workline route release --target merge-request:<n>`.

// OpenRelease is the release tool's merge request open into the branch
// checked out, as the forge lists it: from this repository, its branch one
// of the project's release branches (ADR-0017). nil when none is open, or
// with no forge to ask. One role's fix goes to one branch,
// workline/<role>/release: with several open, the first is taken, the
// others said.
func OpenRelease(o Options) (*forge.MergeRequest, []verdict.Finding, error) {
	spec := o.Forge
	if spec == "" {
		cfg, err := role.LoadProjectConfig(o.Repo)
		if err != nil {
			return nil, nil, err
		}
		spec = cfg.Forge
	}
	if spec == "" || spec == "none" {
		return nil, nil, nil
	}
	f, err := forge.Open(spec, o.Repo)
	if err != nil || f == nil {
		return nil, nil, err
	}
	rs, err := release.Load(o.Repo)
	if err != nil {
		return nil, nil, err
	}
	base, _ := git(o.Repo, nil, "symbolic-ref", "-q", "--short", "HEAD")
	open, err := f.ListMergeRequests()
	if err != nil {
		return nil, nil, err
	}
	var found []forge.MergeRequest
	for _, mr := range open {
		if mr.Here && rs.IsBranch(mr.Branch) && (base == "" || mr.Base == "" || mr.Base == base) {
			found = append(found, mr)
		}
	}
	if len(found) == 0 {
		return nil, nil, nil
	}
	var said []verdict.Finding
	if len(found) > 1 {
		var others []string
		for _, mr := range found[1:] {
			others = append(others, forge.MergeRef(f, mr.ID))
		}
		said = append(said, verdict.Finding{Rule: "releases-open", Level: "warn", Where: forge.MergeRef(f, found[0].ID),
			Message: fmt.Sprintf("several release merge requests are open (%s too): the docs due are judged for this one, whose fix goes to the one branch of each role's release fix", strings.Join(others, ", "))})
	}
	return &found[0], said, nil
}

// OnReleaseHead checks out the head of the release merge request a run on
// `release` targets, detached, when it is not checked out already: its
// docs are judged on its files. back puts the clone where it was; a
// working tree holding changes is refused. branch is the merge request's,
// "" when it is not a release tool's: nothing is checked out.
func OnReleaseHead(o Options) (back func(), branch string, err error) {
	back = func() {}
	if o.Target == nil || o.Target.Kind != "merge-request" {
		return back, "", nil
	}
	spec := o.Forge
	if spec == "" {
		cfg, err := role.LoadProjectConfig(o.Repo)
		if err != nil {
			return back, "", err
		}
		spec = cfg.Forge
	}
	f, err := forge.Open(spec, o.Repo)
	if err != nil || f == nil {
		return back, "", err
	}
	rs, err := release.Load(o.Repo)
	if err != nil {
		return back, "", err
	}
	branch = o.Branch
	if branch == "" {
		mr, err := f.MergeRequest(o.Target.ID)
		if err != nil {
			return back, "", err
		}
		branch = mr.Branch
	}
	if !rs.IsBranch(branch) {
		return back, "", nil
	}
	tip, err := fetchTip(o.Repo, branch, forge.KeepsBranches(f))
	if err != nil {
		return back, "", err
	}
	if head, _ := git(o.Repo, nil, "rev-parse", "HEAD"); head == tip {
		return back, branch, nil
	}
	if back, err = onHead(o.Repo, tip); err != nil {
		return func() {}, "", fmt.Errorf("the release's docs are judged on %s's head, %s: %w", branch, short(tip), err)
	}
	return back, branch, nil
}

// ReleaseHeld says whether a run on the release blocked only for its docs:
// those due, and their fix in a merge request of its own, to merge first
// (ADR-0017). At night that is said on the release's merge request, and
// fails nothing of the night's.
func ReleaseHeld(r *Result) bool {
	if r == nil || r.Status != verdict.Block {
		return false
	}
	for _, f := range r.Findings {
		if (f.Level == "" || f.Level == "block") && !slices.Contains(heldFor, f.Rule) {
			return false
		}
	}
	return true
}

// heldFor are the rules of what holds a release for its docs.
var heldFor = []string{"suspect", "due", "fixed-elsewhere", "fix-in-comment"}
