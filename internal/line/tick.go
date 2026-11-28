package line

// What `workline tick` asks the line (ADR-0069 §2): the decisions kept
// here, read by the tick, never written twice.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/scale"
)

// BacklogDue says whether a backlog run is due, and why: an Urgent issue
// (priority 1, read in the project's own priority labels or workline's) with
// no state or in `refine`, labelled since the last backlog run started, one whose labels are
// unread in unread; then Ready below its target (readyBelow) and moved since
// that run, or it readied one (backlogDue); else false and "". last is that
// run: `<started>/<updated>`, `none`, `waiting`, `unread`.
func BacklogDue(cfg *routing.Config, repo string, b forge.Backlog, open []forge.Issue, last string) (due bool, why string, unread error) {
	effortScheme(repo, b, open) // the scheme read, or workline's own labels kept: the product owner's run says a bad setting
	start, ok := labeledAt(strings.Split(last, "/")[0])
	for _, is := range open {
		if st, in := cfg.StateOf(is.Labels); (ok || last == "none" || last == "") && (!in || st.Name == "refine") &&
			backlog.Priority(is) == 1 && slices.Contains(cfg.Events["schedule"], "product-owner") {
			events, err := b.LabelEvents(is.ID)
			if err != nil {
				unread = errors.Join(unread, fmt.Errorf("#%d, %s: its labels: %w", is.ID, scale.Priority.Name(1), err))
			}
			for _, e := range events {
				if at, ok := labeledAt(e.Created); ok && e.Added && scale.Used.Priority.Family.Of(e.Label) == 1 && at.After(start) {
					return true, fmt.Sprintf("#%d is %s, labelled since the last backlog run: one is due", is.ID, scale.Priority.Name(1)), unread
				}
			}
		}
	}
	if !readyBelow(cfg, open) {
		return false, "", unread
	}
	n := len(slices.DeleteFunc(slices.Clone(open), func(is forge.Issue) bool { st, ok := cfg.StateOf(is.Labels); return !ok || st.Name != readyState }))
	due, said := backlogDue(cfg, b, open, last)
	return due, fmt.Sprintf("Ready %d below %d: %s", n, cfg.Limit(readyState), said), unread
}
