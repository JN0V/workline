//go:build workline_quick

package forge

import "time"

// An engine built for the conformance cases that prove a call is tried
// again (`engine-tags: [workline_quick]`): a call given 2 s, not a minute,
// tried again 10 ms after, not 2 s, so a case stalls for seconds, not
// minutes. A Retry-After is waited for as asked. Never in a shipped engine.
func init() {
	httpClient.Timeout = 2 * time.Second
	retry.pause = 10 * time.Millisecond
}
