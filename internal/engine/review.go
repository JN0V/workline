package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/builtin/inspector"
	"github.com/JN0V/workline/internal/builtin/reviewer"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// lineReview posts a sticky comment's review, {head, body, comments:
// [{path, line, key, body}]}, on the merge request's lines, in the forge's
// own review (#277), and returns the sticky comment's body with what
// became of them in its review region (forge.FillReview): a link to the
// review, and each comment not on its line in full, with why. A forge that
// takes no comment on a line, or refuses them, leaves them all there; the
// refusal is said in the run's findings too, never an error: only a forge
// out of reach is.
func (a *applier) lineReview(t forge.Target, v any, body string) (string, error) {
	var r struct {
		Head     string              `json:"head"`
		Body     string              `json:"body"`
		Comments []forge.LineComment `json:"comments"`
	}
	data, _ := json.Marshal(v)
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("a review reads {head, body, comments: [{path, line, key, body}]}: %v", err)
	}
	if t.Kind != "merge-request" {
		return "", errors.New("a review goes on a merge request")
	}
	for _, c := range r.Comments {
		if c.Path == "" || c.Line < 1 || strings.TrimSpace(c.Body) == "" {
			return "", fmt.Errorf("a comment on a line needs its path, its line and a body: %+v", c)
		}
	}
	for i := range r.Comments {
		r.Comments[i].Body = backlog.Inert(r.Comments[i].Body)
	}
	lr, ok := a.forge.(forge.LineReviewer)
	if !ok || len(r.Comments) == 0 {
		return forge.FillReview(body, forge.Unplaced(r.Comments, "this forge takes no comment on a line").Section()), nil
	}
	done, err := lr.Review(t.ID, r.Head, backlog.Inert(r.Body), r.Comments)
	if err != nil {
		return "", err
	}
	for _, p := range done.Placed {
		if p.State == forge.LineRefused {
			a.findings = append(a.findings, verdict.Finding{Rule: "line-comment-refused", Level: "warn", Where: fmt.Sprintf("%s:%d", p.Path, p.Line),
				Message: "not posted on its line, in full in the summary comment instead: " + p.Why})
		}
	}
	return forge.FillReview(body, done.Section()), nil
}

// mergedRecord is a sticky comment's body, the record in it merged with
// the one the comment holds by then (#375): a run applied after this one
// read it wrote it, a push having been judged before the run before it
// was applied. read is the record the run read; no comment yet, or a
// forge that lists none, leaves body as it is. The reviewer's summary
// comment, or the inspector's triage (#580): the last workline's own
// account wrote, the one a run reads and Sticky edits (ADR-0051). When
// that comment names a later run than body (forge.Newer), its text is
// kept and body's record merged into it (#598).
func (a *applier) mergedRecord(t forge.Target, key, body, read string) (string, error) {
	if _, ok := a.forge.(forge.Backlog); !ok {
		return body, nil
	}
	kept, err := forge.ReadKept(a.forge, t)
	if err != nil {
		return "", err
	}
	marker := forge.Marker("sticky=" + a.role + "/" + key)
	found, ok := kept.Last(marker)
	if ok && forge.Newer(found, body) {
		// A later run's comment, applied first (#598): its text stays,
		// this run's record merged into it.
		merged := reviewer.MergeInto(found, body, read, marker)
		if a.role == "inspector" {
			merged = inspector.MergeInto(found, body, read, marker)
		}
		if merged != "" {
			a.findings = append(a.findings, verdict.Finding{Rule: "applied-after-a-newer-run", Level: "warn",
				Message: "the comment holds a later run's text, applied before this one: kept, this run's record merged into it"})
			return merged, nil
		}
		// A record that does not read: the later run's comment is left
		// as it is, never the older text written over it; this run's
		// commits are read again at the next push.
		a.findings = append(a.findings, verdict.Finding{Rule: "applied-after-a-newer-run", Level: "warn",
			Message: "the comment holds a later run's text, applied before this one: kept as it is, this run's record not merged (a record does not read); what it read is read again"})
		return strings.TrimSpace(strings.Replace(found, marker, "", 1)), nil
	}
	switch {
	case !ok:
		return body, nil
	case a.role == "inspector":
		return inspector.MergeComment(body, read, found), nil
	}
	return reviewer.MergeSummary(body, read, found), nil
}
