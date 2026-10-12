package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// lineReview posts a sticky comment's review, {head, body, comments:
// [{path, line, key, body}]}, on the merge request's lines, in the forge's
// own review (#277), and returns the sticky comment's body with what
// became of them where it says forge.ReviewHere: a link to the review, and each
// comment not on its line in full, with why. A forge that takes no comment
// on a line, or refuses them, leaves them all there; the refusal is said
// in the run's findings too, never an error: only a forge out of reach is.
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
	if len(r.Comments) == 0 {
		return strings.Replace(body, forge.ReviewHere, "", 1), nil
	}
	for i := range r.Comments {
		r.Comments[i].Body = backlog.Inert(r.Comments[i].Body)
	}
	lr, ok := a.forge.(forge.LineReviewer)
	if !ok {
		done := forge.Reviewed{}
		for _, c := range r.Comments {
			done.Placed = append(done.Placed, forge.Placed{LineComment: c, State: forge.LineRefused, Why: "this forge takes no comment on a line"})
		}
		return strings.Replace(body, forge.ReviewHere, reviewSection(done), 1), nil
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
	return strings.Replace(body, forge.ReviewHere, reviewSection(done), 1), nil
}

// reviewSection says where the comments on lines are, and gives in full
// those not on their line, grouped by why.
func reviewSection(r forge.Reviewed) string {
	var b strings.Builder
	posted, open := 0, 0
	var off []forge.Placed
	for _, p := range r.Placed {
		switch p.State {
		case forge.LinePosted:
			posted++
		case forge.LineOpen:
			open++
		default:
			off = append(off, p)
		}
	}
	if on := posted + open; on > 0 {
		review := "the review"
		if r.URL != "" {
			review = "[the review](" + r.URL + ")"
		}
		var parts []string
		if posted > 0 {
			parts = append(parts, fmt.Sprintf("%d posted now", posted))
		}
		if open > 0 {
			parts = append(parts, fmt.Sprintf("%d still open from before, not posted again", open))
		}
		fmt.Fprintf(&b, "**On their lines**: %d %s, in %s (%s): reply to each, or resolve it, there.\n\n",
			on, plural(on, "comment", "comments"), review, strings.Join(parts, ", "))
	}
	var whys []string
	by := map[string][]forge.Placed{}
	for _, p := range off {
		if _, ok := by[p.Why]; !ok {
			whys = append(whys, p.Why)
		}
		by[p.Why] = append(by[p.Why], p)
	}
	for _, why := range whys {
		fmt.Fprintf(&b, "**Here only**, not on their lines — %s:\n\n", strings.TrimSuffix(why, "."))
		for _, p := range by[why] {
			fmt.Fprintf(&b, "`%s:%d`\n\n> %s\n\n", p.Path, p.Line, strings.ReplaceAll(strings.TrimSpace(p.Body), "\n", "\n> "))
		}
	}
	return strings.TrimSuffix(b.String(), "\n\n")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
