//go:build workline_agent_time

package engine

import "time"

// An engine built for the conformance cases on what a run's wait holds
// across its calls (`engine-tags: [workline_agent_time]`): the wait counts
// the time inside its agents' calls alone, never the engine's own steps
// between them — a post, its scans — which a loaded machine stretches past
// a cap of a few seconds before the next call is made. Never in a shipped
// engine.
func init() {
	waited = func(res *Result) time.Duration { return res.inCalls }
}
