// Package reviewmark is what the reviewer's words are, as other roles
// and the sample read them (ADR-0070 §6 e).
package reviewmark

import (
	"encoding/json"
	"regexp"
	"strconv"
)

// Answered is a finding the judge answered: kept (yes) or dropped (no), at
// the head the review read, with the models of the lens that found it and
// of its judge. A line each in the summary comment, for the merge
// request's life: the weekly sample draws the reviewer's acts from them
// (ADR-0046). What it was judged on keys it too, so the same finding on
// the same patch is not judged again, its verdict kept (ADR-0050, step 6):
// the patch id of the merge request's change to the cause's file, and the
// finding's own key; with the independence of a yes, the reason of a no.
// A line written before them keys nothing: its finding is judged again.
type Answered struct {
	Lens    string `json:"lens"`
	Where   string `json:"where"`
	At      string `json:"at"`
	Patch   string `json:"patch,omitempty"`
	Finding string `json:"finding,omitempty"`
	Level   string `json:"level,omitempty"`
	Why     string `json:"why,omitempty"`
	Yes     bool   `json:"yes"`
	Finder  string `json:"finder"`
	Judge   string `json:"judge"`
	Title   string `json:"title"`
}

// SummaryMax is the most characters the summary comment holds: GitHub
// takes 65,536, the engine adds to the review region when it applies. The
// lines of the findings judged give way first, the oldest first.
const SummaryMax = 60000

// SummaryKey keys the reviewer's one comment on a merge request; WarnKey,
// the run apart's, the lenses that warn, when the review is split
// (ADR-0050): each run its own comment and record, written by its own
// applying job, which no other edits at the same time.
const (
	SummaryKey = "summary"
	WarnKey    = "warn"
)

// judgedHead opens the lines of the findings judged, saying how many
// the comment no longer holds; an engine before them wrote none.
var (
	judgedHead = regexp.MustCompile(`<!-- workline:judged-lines gone=(\d+) -->`)
	judgedLine = regexp.MustCompile(`<!-- workline:judged (.*?) -->`)
)

// ReadJudged reads the lines of the findings judged in a summary comment:
// each line; how many the comment no longer holds — past its size, or a
// line that does not read; and whether it holds them at all: one an
// engine before them wrote does not (ADR-0046).
func ReadJudged(comment string) (judged []Answered, gone int, held bool) {
	if m := judgedHead.FindStringSubmatch(comment); m != nil {
		gone, _ = strconv.Atoi(m[1])
		held = true
	}
	for _, m := range judgedLine.FindAllStringSubmatch(comment, -1) {
		var j Answered
		if err := json.Unmarshal([]byte(m[1]), &j); err != nil || j.Lens == "" || j.Where == "" {
			gone++
			continue
		}
		judged = append(judged, j)
	}
	return judged, gone, held
}
