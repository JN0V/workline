package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// Added is what a gate found a change adds, for what shows it on the
// merge request (ADR-0049, step 3): each check's count, and the findings
// themselves, placed and named by a fingerprint.
type Added struct {
	Base     string       `json:"base"` // the merge base they are counted since
	Checks   []CheckCount `json:"checks"`
	Findings []Finding    `json:"findings"`
}

// CheckCount is how one check of a gate ended on a change.
type CheckCount struct {
	ID     string `json:"id"`
	Output string `json:"output"` // exit or sarif
	New    int    `json:"new"`
	// Before: the findings there before the change, not counted; -1 when
	// the tool told the new ones itself, the rest not known, or none were read.
	Before int    `json:"before"`
	Failed bool   `json:"failed,omitempty"` // an exit check that did not pass
	Error  string `json:"error,omitempty"`  // the check did not work: nothing it found is known
	Fix    string `json:"fix,omitempty"`    // the command that fixes what it finds: a formatter's
}

// Finding is one result a change adds, as the tool wrote it.
type Finding struct {
	Check   string `json:"check"`
	Tool    string `json:"tool,omitempty"`
	Rule    string `json:"rule,omitempty"`
	Level   string `json:"level"` // SARIF's: error, warning, note
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message,omitempty"`
	// Fingerprint holds no line number: the finding's identity, as the
	// comparison reads it, and how many alike come before it in its file.
	// It stays while the finding only moves (GitLab's Code Quality reads
	// it).
	Fingerprint string `json:"fingerprint"`
}

// stamp names each result by its identity and its rank among those
// alike, in the order the tool wrote them: two results alike in a file are
// told apart, and neither changes while they only move.
func stamp(rs []*result, l *lines) {
	seen := map[string]int{}
	for _, r := range rs {
		id := r.identity(l, r.uri)
		sum := sha256.Sum256([]byte(id + "\x00" + strconv.Itoa(seen[id])))
		seen[id]++
		r.posted = hex.EncodeToString(sum[:])
	}
}

// added are the results to show: on a change compared, those marked new;
// told by the tool, every one.
func added(c Check, rs []*result) []Finding {
	var out []Finding
	for _, r := range rs {
		if r.state == "unchanged" {
			continue
		}
		out = append(out, Finding{Check: c.ID, Tool: r.tool, Rule: r.rule, Level: r.level, Path: r.uri, Line: r.line,
			Message: r.text, Fingerprint: r.posted})
	}
	return out
}
