package line

// When Ready is below its target, is a backlog run worth starting
// (#1549)? Not while nothing changes: the job that left Ready below its
// target would start one on every apply, each readying nothing. A run is
// due when Ready dropped since the last backlog run started, or that run
// readied an issue, so more may follow; else none, said in the summary.

import (
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// backlogDue says whether the backlog run Ready's drop below its target
// asks for is due, and why, in words for the summary: Ready dropped since
// the last backlog run started, an issue's Ready label taken off since,
// or that run readied one, its label put on while it ran. Only the issues
// in Ready, or in a state Ready cannot be reached from (taken, blocked),
// are read: one left Ready, or one readied, sits there.
func backlogDue(cfg *routing.Config, b forge.Backlog, open []forge.Issue, last string) (bool, string) {
	switch last {
	case "", "none":
		return true, "Ready is below its target, and no backlog run before: one is due"
	case "waiting":
		return false, "Ready is below its target, and a backlog run waits already: none added"
	case "unread":
		return false, "Ready is below its target, but the last backlog run could not be read: none started"
	}
	from, to, ok := strings.Cut(last, "/")
	start, ok1 := labeledAt(from)
	end, ok2 := labeledAt(to)
	if !ok || !ok1 || !ok2 {
		return false, "Ready is below its target, but the last backlog run is not a time (" + last + "): none started"
	}
	ready, ok := cfg.State(readyState)
	if !ok {
		return false, ""
	}
	upstream := upstreamOf(cfg, readyState)
	for _, is := range open {
		st, ok := cfg.StateOf(is.Labels)
		if !ok || upstream[st.Name] {
			continue
		}
		events, err := b.LabelEvents(is.ID)
		if err != nil {
			return false, "Ready is below its target, but the issues' labels could not be read: no backlog run started: " + forge.Said(err)
		}
		for _, e := range events {
			at, ok := labeledAt(e.Created)
			if !ok || e.Label != ready.Label {
				continue
			}
			switch {
			case !e.Added && at.After(start):
				return true, "Ready dropped below its target since the last backlog run: one is due"
			case e.Added && !at.Before(start) && !at.After(end):
				return true, "Ready is below its target, and the last backlog run readied an issue: one more is due"
			}
		}
	}
	return false, "Ready is below its target, but the last backlog run (ended " + end.UTC().Format(time.RFC3339) +
		") readied none, and none has left Ready since: no backlog run started"
}

// upstreamOf names the states whose next steps lead to state name: an
// issue there has not reached it yet.
func upstreamOf(cfg *routing.Config, name string) map[string]bool {
	up := map[string]bool{}
	for _, s := range cfg.States {
		var seen []string
		for n := s.Next; n != "" && !slices.Contains(seen, n); {
			if n == name {
				up[s.Name] = true
				break
			}
			seen = append(seen, n)
			next, _ := cfg.State(n)
			n = next.Next
		}
	}
	return up
}
