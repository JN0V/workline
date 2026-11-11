package engine

// A merge the policy says yes to, the forge refusing it, is said where a
// person reads (#1354): on the merge request, and on each issue it closes,
// one comment each, edited as it goes. A conflict with the base names its
// files and says the request waits to be brought up to date with it; who
// brings it is not said here. Once it merges, or its auto-merge is on,
// the same comments say so: none is left saying it waits.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// RefusedKey is the sticky comment's key that says a refused merge, on
// the merge request and on each issue it closes.
const RefusedKey = "line/merge-refused"

// refusedMark is the data line the comment on an issue carries while its
// merge request's merge is refused, read back by the reconciler.
const refusedMark = "merge-refused "

// Refusal is what a refused merge's comment on an issue keeps for the
// reconciler: the merge request, its head, what the forge said, the
// conflict's base when it is one.
type Refusal struct {
	MR       int    `json:"mr"`
	Head     string `json:"head"`
	Said     string `json:"said"`
	Conflict string `json:"conflict,omitempty"`
}

// ReadRefusal is the refusal an issue's comments keep, false when none
// says one: a merge request merged since, or never refused.
func ReadRefusal(comments []string) (Refusal, bool) {
	sticky := forge.Marker("sticky=" + RefusedKey)
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
		if json.Unmarshal([]byte(data), &r) == nil {
			return r, true
		}
	}
	return Refusal{}, false
}

// sayRefused writes, on mr and on each issue it closes, that its merge,
// due (due says why), was refused at tip: err, the forge's refusal.
func sayRefused(f forge.Forge, mr forge.MergeRequest, tip, due string, err error) error {
	said, ref, at := forge.Said(err), forge.MergeRef(f, mr.ID), short(tip)
	c := forge.Conflicting(err)
	body := fmt.Sprintf("**Not merged**: its merge was due, %s, but the forge refused it at its head %s: %s.", due, at, said)
	issue := fmt.Sprintf("**Its pull request %s is not merged**: its merge was due, %s, but the forge refused it at its head %s: %s.", ref, due, at, said)
	if c != nil {
		where := ""
		if len(c.Files) > 0 {
			where = " in `" + strings.Join(c.Files, "`, `") + "`"
		}
		waits := fmt.Sprintf("It stays open, not merged: it waits to be brought up to date with %s, %s merged into `%s` or the branch rebased on it. Its new head is then weighed again, as any head: the line, then the merge policy.", c.Base, c.Base, mr.Branch)
		body = fmt.Sprintf("**Not merged: it conflicts with %s**: its merge was due, %s, but at its head %s, %s and its branch change the same lines%s.\n\n%s\n\n**What a person can do**: bring `%s` up to date with %s, or close this pull request.",
			c.Base, due, at, c.Base, where, waits, mr.Branch, c.Base)
		issue = fmt.Sprintf("**Its pull request %s is not merged: it conflicts with %s**: its merge was due, %s, but at its head %s, %s and its branch change the same lines%s.\n\n%s\n\n**What a person can do**: bring `%s` up to date with %s, or close %s.",
			ref, c.Base, due, at, c.Base, where, waits, mr.Branch, c.Base, ref)
	}
	if err := f.Sticky(forge.Target{Kind: "merge-request", ID: mr.ID}, backlog.Inert(body), forge.Marker("sticky="+RefusedKey), true); err != nil {
		return err
	}
	r := Refusal{MR: mr.ID, Head: tip, Said: said}
	if c != nil {
		r.Conflict = c.Base
	}
	data, _ := json.Marshal(r)
	issue += "\n\n<!-- workline:" + refusedMark + string(data) + " -->"
	for _, id := range forge.Closes(mr.Title, mr.Body) {
		if err := f.Sticky(forge.Target{Kind: "issue", ID: id}, backlog.Inert(issue), forge.Marker("sticky="+RefusedKey), true); err != nil {
			return err
		}
	}
	return nil
}

// sayUnrefused edits the comments sayRefused wrote, when there are any:
// at tip, merged, or its auto-merge on.
func sayUnrefused(f forge.Forge, mr forge.MergeRequest, tip string, merged bool) error {
	ref, at := forge.MergeRef(f, mr.ID), short(tip)
	body := fmt.Sprintf("**Merged**: at its head %s, the merge refused before went through.", at)
	issue := fmt.Sprintf("**Its pull request %s is merged**: at its head %s, the merge refused before went through.", ref, at)
	if !merged {
		body = fmt.Sprintf("**No longer refused**: at its head %s, auto-merge is on: the forge merges it once its rules are met.", at)
		issue = fmt.Sprintf("**Its pull request %s is no longer refused**: at its head %s, auto-merge is on: the forge merges it once its rules are met.", ref, at)
	}
	if err := f.Sticky(forge.Target{Kind: "merge-request", ID: mr.ID}, body, forge.Marker("sticky="+RefusedKey), false); err != nil {
		return err
	}
	for _, id := range forge.Closes(mr.Title, mr.Body) {
		if err := f.Sticky(forge.Target{Kind: "issue", ID: id}, issue, forge.Marker("sticky="+RefusedKey), false); err != nil {
			return err
		}
	}
	return nil
}
