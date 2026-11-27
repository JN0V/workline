package line

// What `workline tick` asks the line (ADR-0069 §2): the decisions kept
// here, read by the tick, never written twice.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/scale"
)

// BacklogDue says whether a backlog run is due, and why: an Urgent issue
// (#2322, priority 1) with no state or in `refine`, labelled since the last
// backlog run started, its labels unread an error; then Ready below its target
// (readyBelow) and moved since that run, or it readied one (backlogDue); else
// false and "". last is that run: `<started>/<updated>`, `none`, `waiting`, `unread`.
func BacklogDue(cfg *routing.Config, b forge.Backlog, open []forge.Issue, last string) (bool, string, error) {
	start, ok := labeledAt(strings.Split(last, "/")[0])
	for _, is := range open {
		if st, in := cfg.StateOf(is.Labels); (ok || last == "none" || last == "") && (!in || st.Name == "refine") &&
			slices.Contains(scale.Own.Levels(is.Labels), 1) && slices.Contains(cfg.Events["schedule"], "product-owner") {
			events, err := b.LabelEvents(is.ID)
			if err != nil {
				return false, "", fmt.Errorf("#%d, %s: its labels: %w", is.ID, scale.Priority.Name(1), err)
			}
			for _, e := range events {
				if at, ok := labeledAt(e.Created); ok && e.Added && scale.Own.Of(e.Label) == 1 && at.After(start) {
					return true, fmt.Sprintf("#%d is %s, labelled since the last backlog run: one is due", is.ID, scale.Priority.Name(1)), nil
				}
			}
		}
	}
	if !readyBelow(cfg, open) {
		return false, "", nil
	}
	n := 0
	for _, is := range open {
		if st, ok := cfg.StateOf(is.Labels); ok && st.Name == readyState {
			n++
		}
	}
	due, said := backlogDue(cfg, b, open, last)
	return due, fmt.Sprintf("Ready %d below %d: %s", n, cfg.Limit(readyState), said), nil
}
