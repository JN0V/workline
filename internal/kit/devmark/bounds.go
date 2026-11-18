package devmark

import (
	"fmt"
	"strings"
)

// Refusal is a file a patch reached and post refused, and why: kept in the
// record the claim's comment holds, for the auditor (ADR-0046).
type Refusal struct {
	File string `json:"file"`
	Rule string `json:"rule"`
	// Detail is why the bounds could not be read, for bounds-unread.
	Detail string `json:"detail,omitempty"`
}

// Beyond is a file a change wrote beyond its issue's Scope, and the
// one-line reason its agent gave, "" when it gave none.
type Beyond struct {
	File string `json:"file"`
	Why  string `json:"why,omitempty"`
}

// BeyondLines are a record's files beyond the Scope, as its pull request
// lists them: what the auditor samples (ADR-0046); "" for none.
func BeyondLines(l []Beyond) string {
	var b strings.Builder
	for _, x := range l {
		why := x.Why
		if why == "" {
			why = "no reason given by its agent"
		}
		fmt.Fprintf(&b, "- `%s`: %s\n", x.File, InertText(strings.Join(strings.Fields(why), " ")))
	}
	return b.String()
}
