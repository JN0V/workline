package line

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/release"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
)

// What holds a release on the forge (ADR-0051, cut 1), each a finding's
// rule.
const (
	RuleHeld        = "release-held"         // an open issue labelled forge.HoldLabel
	RuleBaseRed     = "release-base-red"     // a check red on the tip the release is cut from
	RuleBasePending = "release-base-pending" // a check not done there yet
	RuleBaseNone    = "release-base-unchecked"
	RuleHoldsUnread = "release-holds-unread"
)

// releaseHolds reads what holds the release on the forge, on `release` or
// a release tool's merge request (ADR-0017): each open issue labelled
// forge.HoldLabel, and the tip of the branch it is cut from until every
// check there is green, read, never run again. What could not be read
// holds it too; with no forge, it is said. It returns the findings, and
// what holds in a few words; none when the run is no release's.
func releaseHolds(event string, o engine.Options) ([]verdict.Finding, string) {
	if event != "release" && event != "merge-request" {
		return nil, ""
	}
	spec := o.Forge
	if spec == "" {
		if cfg, err := role.LoadProjectConfig(o.Repo); err == nil {
			spec = cfg.Forge
		}
	}
	var f forge.Forge
	var err error
	if spec != "" && spec != "none" {
		f, err = forge.Open(spec, o.Repo)
	}
	unread := func(level, why string) ([]verdict.Finding, string) {
		return []verdict.Finding{{Rule: RuleHoldsUnread, Level: level, Where: "release", Message: why}}, ""
	}
	// The merge request says where it goes; unread, the branch it goes
	// into is not known, and never taken for the default one.
	branch, base := o.Branch, ""
	var mrErr error
	if f != nil && o.Target != nil && o.Target.Kind == "merge-request" {
		var mr forge.MergeRequest
		if mr, mrErr = f.MergeRequest(o.Target.ID); mrErr == nil {
			base = mr.Base
			if branch == "" {
				branch = mr.Branch
			}
		}
	}
	if event == "merge-request" {
		// Not known to be the release, not held: said, a warning, as a
		// forge's hiccup would otherwise hold every merge request.
		if why := errors.Join(err, mrErr); branch == "" && why != nil && o.Target != nil {
			return unread("warn", fmt.Sprintf("whether %s is a release tool's is unknown, so what holds a release is not read: %v; give its branch with --branch", o.Target, why))
		}
		rs, err := release.Load(o.Repo)
		if err != nil || !rs.IsBranch(branch) {
			return nil, ""
		}
	}
	rf, ok := f.(forge.Releasing)
	switch {
	case err != nil:
		return unread(verdict.Block, "what holds the release could not be read: "+err.Error())
	case f == nil:
		return unread("warn", fmt.Sprintf("no forge: neither an issue labelled %s nor the checks on the tip the release is cut from are read", forge.HoldLabel))
	case !ok:
		return unread("warn", fmt.Sprintf("forge %s does not say what holds a release: neither an issue labelled %s nor the checks on the tip it is cut from are read", spec, forge.HoldLabel))
	}
	var found []verdict.Finding
	var held []string
	issues, err := rf.Holding()
	if err != nil {
		found = append(found, verdict.Finding{Rule: RuleHoldsUnread, Where: "release",
			Message: fmt.Sprintf("the issues labelled %s could not be read, so the release is held: %v", forge.HoldLabel, err)})
		held = append(held, "its issues unread")
	}
	for _, is := range issues {
		found = append(found, verdict.Finding{Rule: RuleHeld, Where: fmt.Sprintf("#%d", is.ID),
			Message: fmt.Sprintf("#%d %q is open, labelled %s: it holds the release; take the label off, or close the issue, to let it go", is.ID, is.Title, forge.HoldLabel)})
		held = append(held, fmt.Sprintf("#%d", is.ID))
	}
	if mrErr != nil {
		found = append(found, verdict.Finding{Rule: RuleHoldsUnread, Where: o.Target.String(),
			Message: fmt.Sprintf("%s could not be read, so the branch it goes into is not known, nor its checks read: the release is held: %v", o.Target, mrErr)})
		return found, strings.Join(append(held, "its merge request unread"), ", ")
	}
	tip, err := rf.Tip(base)
	name := tip.Branch
	if name == "" {
		name = base
	}
	if name == "" {
		name = "the default branch"
	}
	at := fmt.Sprintf("%s's tip", name)
	if c := tip.Commit; c != "" {
		at = fmt.Sprintf("%s's tip, %.7s,", name, c)
	}
	switch {
	case err != nil:
		found = append(found, verdict.Finding{Rule: RuleHoldsUnread, Where: name,
			Message: fmt.Sprintf("the checks on %s's tip could not be read, so the release is held: %v", name, err)})
		held = append(held, name+"'s checks unread")
	case tip.State == "failure":
		found = append(found, verdict.Finding{Rule: RuleBaseRed, Where: name,
			Message: fmt.Sprintf("%s is red: %s; the release waits for it green: fix it, or run its red checks again", at, strings.Join(tip.Failing, ", "))})
		held = append(held, name+" red")
	case tip.State == "pending":
		found = append(found, verdict.Finding{Rule: RuleBasePending, Where: name,
			Message: fmt.Sprintf("%s has checks not done: %s; the release waits for them green, read again once they end", at, strings.Join(tip.Pending, ", "))})
		held = append(held, name+"'s checks not done")
	case tip.State == "":
		found = append(found, verdict.Finding{Rule: RuleBaseNone, Level: "warn", Where: name,
			Message: fmt.Sprintf("%s carries no check: nothing there to read as green", at)})
	}
	return found, strings.Join(held, ", ")
}
