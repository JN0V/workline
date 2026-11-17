package line

// What `workline tick` asks the line (ADR-0069 §2): the decisions kept
// here, read by the tick, never written twice.

import (
	"fmt"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// BacklogDue says whether a backlog run is due, and why: Ready below its
// target (readyBelow), and Ready moved since the last backlog run, or that
// run readied an issue (backlogDue). last is that run as backlogDue reads
// it: `<started>/<updated>`, `none`, `waiting` or `unread`. Not below its
// target, or no backlog run on this line: false and "".
func BacklogDue(cfg *routing.Config, b forge.Backlog, open []forge.Issue, last string) (bool, string) {
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
