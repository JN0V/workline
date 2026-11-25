package devmark

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// The reviewer's comments a round reads its findings from (its summary,
// and the comment of its lenses that warn when the review is split), and
// the count of its runs their records hold.
var (
	ReviewerMarkers = []string{forge.Marker("sticky=reviewer/summary"), forge.Marker("sticky=reviewer/warn")}
	ReviewerRuns    = regexp.MustCompile(`<!-- workline:record reviewed=\S*(?: patches=\S*)? runs=(\d+)`)
	// ReviewerAt: the head its summary read. A run its rules hold counts
	// no run, its findings new all the same (#1040).
	ReviewerAt = regexp.MustCompile(`<!-- workline:at head=([0-9a-f]+)`)
	// reviewerFindings: the hidden record of its findings, a JSON list of
	// lens, holds, where, severity and text (the reviewer's reviewer.go).
	reviewerFindings = regexp.MustCompile(`<!-- workline:findings (\[.*\]) -->`)
	// ReviewerKept: the head on which the judge the reviewer asked said
	// a task keeps the behaviour, written only on a yes, on the
	// line above the head its summary read: never a quote in its table.
	ReviewerKept = regexp.MustCompile(`<!-- workline:kept head=([0-9a-f]+) -->\n<!-- workline:at head=([0-9a-f]+)`)
	// ReviewerOwed: its summary's record owes every lens, its rules alone
	// ran (the reviewer's record.go writes it).
	ReviewerOwed = regexp.MustCompile(`<!-- workline:record reviewed=\S*(?: patches=\S*)? runs=\d+ owed=all`)
)

// Round is one round of the review loop, as the record keeps it.
type Round struct {
	N      int      `json:"n"`
	Asked  []string `json:"asked,omitempty"`  // the findings asked, each as its row reads
	Heard  int      `json:"heard,omitempty"`  // a person's comments read
	Nits   int      `json:"nits,omitempty"`   // nits said, not asked
	Result string   `json:"result,omitempty"` // its proof's result; no-code when it wrote none
	Whys   bool     `json:"whys,omitempty"`   // the five whys ran
	Person bool     `json:"person,omitempty"` // past the last round: its issue blocked (ADR-0059)
	// Bounds, a round with no fix: held, nothing to wait for; blocked, a
	// project check stays red, or its fix was held off a green head:
	// blocked for a person (#938); objected, its objection waits for the
	// tester (#987). Asked, in a record before ADR-0065's change 2: it
	// asked the product owner for files outside the Scope (#921).
	// Result not-pushed: its fix, red on a green head, kept off it.
	Bounds string         `json:"bounds,omitempty"`
	Turns  int            `json:"turns,omitempty"`  // what its agent used, as it reported them
	Tokens int            `json:"tokens,omitempty"` // fresh, what ai-max-tokens holds
	Cached int            `json:"cached,omitempty"` // read from cache, apart
	Tools  map[string]int `json:"tools,omitempty"`
	// Beside: the new test files its fix wrote beside the Scope
	// (ADR-0063, §9).
	Beside []string `json:"beside,omitempty"`
	// Beyond: the files its fix wrote beyond the Scope, each with its
	// agent's reason (ADR-0065, change 2).
	Beyond []Beyond `json:"beyond,omitempty"`
	// Tester: the review's test findings it sent to the tester (#1265).
	Tester []string `json:"tester,omitempty"`
	// Merge: a round that merged the base in (ADR-0067): settled, or
	// failed, nothing pushed.
	Merge string `json:"merge,omitempty"`
}

// Asked is a finding a round asks the agent to fix: a reviewer's row that
// blocks, or warns and is Important, or a person's comment.
type Asked struct {
	Key   string `json:"key"` // lens, file, title: the same finding round after round
	Lens  string `json:"lens,omitempty"`
	Where string `json:"where,omitempty"`
	Text  string `json:"text"`
	Holds string `json:"holds"` // blocks, warns, or a person
	// Detail is the finding whole, from the reviewer's comment on its
	// line: its row shows the title alone.
	Detail string `json:"detail,omitempty"`
	// Check: a project check red, whose log the round reads (#917).
	Check *forge.HeadCheck `json:"check,omitempty"`
	// Waited: asked again from the record's last round (#1022): the
	// developer's still, a test finding too.
	Waited bool `json:"waited,omitempty"`
}

// RecordUnread says why a reviewer comment's findings record cannot be
// read, "" when it reads, or when the comment shows no finding either.
func RecordUnread(comment string) string {
	m := reviewerFindings.FindStringSubmatch(comment)
	if m == nil {
		if strings.Contains(comment, "|---") {
			return "its comment shows a table but holds no findings record (workline:findings): no finding read from the table's words"
		}
		return ""
	}
	if _, err := decodeRows(m[1]); err != nil {
		return "its findings record does not parse: " + err.Error()
	}
	return ""
}

type reviewRow struct {
	Lens, Holds, Where, Severity, Text string
}

// decodeRows is the one decoding RecordUnread and ReviewRows share, so a
// mistyped field is said unread, never read as no finding.
func decodeRows(raw string) ([]reviewRow, error) {
	var list []reviewRow
	err := json.Unmarshal([]byte(raw), &list)
	return list, err
}

// ReviewRows reads the findings of the reviewer's comment from its hidden
// findings record, never from its table's words (#705): what blocks, a
// lens's or a rule's, and what warns and is Important; the nits counted.
// Another that warns asks nothing: left. A comment with no record asks
// nothing.
func ReviewRows(comment string) (asked []Asked, nits int) {
	m := reviewerFindings.FindStringSubmatch(comment)
	if m == nil {
		return nil, 0
	}
	list, err := decodeRows(m[1])
	if err != nil {
		return nil, 0
	}
	for _, x := range list {
		switch {
		case x.Severity == "Nit":
			nits++
			continue
		case x.Holds == "blocks": // a lens's, or a rule's: a gate's, said bare
		case x.Holds != "warns" || x.Severity != "Important":
			continue
		}
		msg := x.Text
		if x.Severity != "" {
			msg = x.Severity + ": " + msg
		}
		asked = append(asked, ReviewFinding(x.Holds, x.Lens, x.Where, msg))
	}
	return asked, nits
}

// ReviewFinding is a reviewer's finding asked, its key the same round
// after round: lens, file, title.
func ReviewFinding(holds, lens, where, msg string) Asked {
	title := strings.TrimSpace(strings.TrimPrefix(msg, "Important:"))
	if t, _, ok := strings.Cut(title, " — "); ok {
		title = t
	}
	file, _, _ := strings.Cut(where, ":")
	return Asked{Key: lens + "|" + file + "|" + title, Lens: lens, Where: where, Text: msg, Holds: holds}
}
