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

// BacklogDue says whether a backlog run is due, and why: Ready below its
// target (readyBelow), and Ready moved since the last backlog run, or that
// run readied an issue (backlogDue). last is that run as backlogDue reads
// it: `<started>/<updated>`, `none`, `waiting` or `unread`. Not below its
// target, or no backlog run on this line: false and "". Before all, due for
// an Urgent issue, priority 1, so that the product owner reads it at the next
// tick, not the next night: one it has not read yet, with no state or in
// `refine`, its priority label put on since the last backlog run started.
func BacklogDue(cfg *routing.Config, b forge.Backlog, open []forge.Issue, last string) (bool, string) {
	start, ok := labeledAt(strings.Split(last, "/")[0])
	for _, is := range open {
		if st, in := cfg.StateOf(is.Labels); (ok || last == "none" || last == "") && (!in || st.Name == "refine") &&
			slices.Contains(scale.Own.Levels(is.Labels), 1) && slices.Contains(cfg.Events["schedule"], "product-owner") {
			events, _ := b.LabelEvents(is.ID)
			for _, e := range events {
				if at, ok := labeledAt(e.Created); ok && e.Added && scale.Own.Of(e.Label) == 1 && at.After(start) {
					return true, fmt.Sprintf("#%d is %s, labelled since the last backlog run: one is due", is.ID, scale.Priority.Name(1))
				}
			}
		}
	}
	if !readyBelow(cfg, open) {
		return false, ""
	}
	n := 0
	for _, is := range open {
		if st, ok := cfg.StateOf(is.Labels); ok && st.Name == readyState {
			n++
		}
	}
	due, said := backlogDue(cfg, b, open, last)
	return due, fmt.Sprintf("Ready %d below %d: %s", n, cfg.Limit(readyState), said)
}
