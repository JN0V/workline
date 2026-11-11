package engine

// A merge the policy says yes to, the forge refusing it, is said where a
// person reads (#1354): on the merge request, and on each issue it closes,
// one comment each, edited as it goes. A conflict with the base names its
// files and says the request waits to be brought up to date with it; who
// brings it is not said here. Once it merges, or its auto-merge is on,
// the same comments say so. Each is said as true at the head it names:
// a head pushed since is weighed again, as any head.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// RefusedKey is the sticky comment's key that says a refused merge, on
// the merge request and on each issue it closes.
const RefusedKey = developer.RefusedKey

// refusedMark is the data line the comment on an issue carries while its
// merge request's merge is refused, read back by the reconciler.
const refusedMark = "merge-refused "

// refusedSticky is the marker of the comment on a merge request; on an
// issue, of the comment about merge request mr (mr > 0), one for each
// merge request that closes it.
func refusedSticky(mr int) string {
	if mr > 0 {
		return forge.Marker(fmt.Sprintf("sticky=%s mr=%d", RefusedKey, mr))
	}
	return forge.Marker("sticky=" + RefusedKey)
}

// Refusal is what a refused merge's comment on an issue keeps for the
// reconciler: the merge request, its head, what the forge said, the
// conflict's base when it is one.
type Refusal struct {
	MR       int    `json:"mr"`
	Head     string `json:"head"`
	Said     string `json:"said"`
	Conflict string `json:"conflict,omitempty"`
}

// ReadRefusal is the refusal of merge request mr an issue's comments
// keep, false when none says one: merged since, or never refused.
func ReadRefusal(comments []string, mr int) (Refusal, bool) {
	sticky := refusedSticky(mr)
	for _, c := range comments {
		if !strings.Contains(c, sticky) {
			continue
		}
		_, after, ok := strings.Cut(c, "<!-- workline:"+refusedMark)
		if !ok {
			return Refusal{}, false
		}
		data, _, _ := strings.Cut(after, " -->")
		var r Refusal
		if json.Unmarshal([]byte(data), &r) == nil && r.MR == mr {
			return r, true
		}
	}
	return Refusal{}, false
}

// sayRefused writes, on mr and on each issue it closes, that its merge,
// due (due says why), was refused at tip: err, the forge's refusal. A
// comment not written is said as a finding: the run goes on.
func sayRefused(f forge.Forge, mr forge.MergeRequest, tip, due string, err error, say func(rule, level, msg string)) {
	said, ref, at := forge.Said(err), forge.MergeRef(f, mr.ID), short(tip)
	c := forge.Conflicting(err)
	body := fmt.Sprintf("**Not merged**: its merge was due, %s, but the forge refused it at its head %s: %s.", due, at, said)
	issue := fmt.Sprintf("**Its pull request %s is not merged**: its merge was due, %s, but the forge refused it at its head %s: %s.", ref, due, at, said)
	if c != nil {
		where := ""
		if len(c.Files) > 0 {
			where = " in `" + strings.Join(c.Files, "`, `") + "`"
		}
		waits := fmt.Sprintf("A head that conflicts with %s is not merged: it waits to be brought up to date with %s, %s merged into `%s` or the branch rebased on it. A new head is weighed again, as any head: the line, then the merge policy.", c.Base, c.Base, c.Base, mr.Branch)
		switch {
		case c.Failed:
			// The developer's merge round failed at this head (ADR-0067).
			waits = fmt.Sprintf("The developer's round could not merge %[1]s in: its issue is blocked for a person, who merges %[1]s into `%[2]s`, settles the conflict and pushes, never a rebase. A new head is weighed again, as any head.", c.Base, mr.Branch)
		case c.Commit != "":
			waits = fmt.Sprintf("It waits to be brought up to date with %[1]s: the developer's next review round merges %[1]s in, settles the conflict keeping both changes, and pushes a merge commit, never a rebase; its checks and the reviewer read it, then it merges (ADR-0067).", c.Base)
		}
		body = fmt.Sprintf("**Not merged: it conflicts with %s**: its merge was due, %s, but at its head %s, %s and its branch change the same lines%s.\n\n%s\n\n**What a person can do**: bring `%s` up to date with %s, or close this pull request.",
			c.Base, due, at, c.Base, where, waits, mr.Branch, c.Base)
		issue = fmt.Sprintf("**Its pull request %s is not merged: it conflicts with %s**: its merge was due, %s, but at its head %s, %s and its branch change the same lines%s.\n\n%s\n\n**What a person can do**: bring `%s` up to date with %s, or close %s.",
			ref, c.Base, due, at, c.Base, where, waits, mr.Branch, c.Base, ref)
	}
	unsaid := func(err error) {
		say("refusal-unsaid", "warn", "its refused merge could not be said on it or its issue: "+forge.Said(err))
	}
	body = backlog.Inert(body)
	if c != nil && c.Commit != "" && !c.Failed {
		body += "\n\n" + developer.ConflictMark(tip, c.Commit) // the merge round's record
	}
	if err := f.Sticky(forge.Target{Kind: "merge-request", ID: mr.ID}, body, refusedSticky(0), true); err != nil {
		unsaid(err)
		return
	}
	r := Refusal{MR: mr.ID, Head: tip, Said: said}
	if c != nil {
		r.Conflict = c.Base
	}
	data, _ := json.Marshal(r)
	issue += "\n\n<!-- workline:" + refusedMark + string(data) + " -->"
	for _, id := range forge.Closes(mr.Title, mr.Body) {
		if err := f.Sticky(forge.Target{Kind: "issue", ID: id}, backlog.Inert(issue), refusedSticky(mr.ID), true); err != nil {
			unsaid(err)
			return
		}
	}
}

// sayUnrefused edits the comments sayRefused wrote, when mr holds one:
// at tip, merged, or its auto-merge on. A merge request never refused
// costs one read of its comments. A comment not edited is said as a
// finding: the run goes on.
func sayUnrefused(f forge.Forge, mr forge.MergeRequest, tip string, merged bool, say func(rule, level, msg string)) {
	unsaid := func(err error) {
		say("refusal-unsaid", "warn", "the merge refused before could not be said through on it or its issue: "+forge.Said(err))
	}
	t := forge.Target{Kind: "merge-request", ID: mr.ID}
	if b, ok := f.(forge.Backlog); ok {
		comments, err := b.Comments(t)
		if err != nil {
			unsaid(err)
			return
		}
		if !strings.Contains(strings.Join(comments, "\n"), refusedSticky(0)) {
			return
		}
	}
	ref, at := forge.MergeRef(f, mr.ID), short(tip)
	body := fmt.Sprintf("**Merged**: at its head %s, the merge refused before went through.", at)
	issue := fmt.Sprintf("**Its pull request %s is merged**: at its head %s, the merge refused before went through.", ref, at)
	if !merged {
		body = fmt.Sprintf("**No longer refused**: at its head %s, auto-merge is on: the forge merges it once its rules are met.", at)
		issue = fmt.Sprintf("**Its pull request %s is no longer refused**: at its head %s, auto-merge is on: the forge merges it once its rules are met.", ref, at)
	}
	if err := f.Sticky(t, body, refusedSticky(0), false); err != nil {
		unsaid(err)
		return
	}
	for _, id := range forge.Closes(mr.Title, mr.Body) {
		if err := f.Sticky(forge.Target{Kind: "issue", ID: id}, issue, refusedSticky(mr.ID), false); err != nil {
			unsaid(err)
			return
		}
	}
}
