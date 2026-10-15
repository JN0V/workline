package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
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
// forge that lists none, leaves body as it is.
func (a *applier) mergedRecord(t forge.Target, key, body, read string) (string, error) {
	b, ok := a.forge.(forge.Backlog)
	if !ok {
		return body, nil
	}
	comments, err := b.Comments(t)
	if err != nil {
		return "", err
	}
	marker := forge.Marker("sticky=" + a.role + "/" + key)
	for _, c := range comments { // the first, the one a run reads
		if strings.Contains(c, marker) {
			return reviewer.MergeSummary(body, read, c), nil
		}
	}
	return body, nil
}
