//go:build workline_alias

package work

// An engine built for the conformance cases that prove the table is the
// one place a heading is read (ADR-0039, `engine-tags: [workline_alias]`):
// how a person accepts the need takes a new name in front, the shipped one
// still read, as a new name would. Never in a shipped engine.
func init() { Headings = Headings.With(Validation, "Accepted when") }
