package documentalist

import (
	"fmt"
	"strconv"
	"time"

	"github.com/JN0V/workline/internal/verdict"
)

// Freshness says how long a doc stays trusted without being read again.
type Freshness struct {
	StaleAfterDays int `json:"stale-after-days"`
}

// staleDocs reports the docs last confirmed too long ago: the newest commit
// their `checked` names is older than stale-after-days. A doc is dated by its
// confirmation, not by its last edit: a doc nobody reads again drifts from a
// world that changed around it, even when its sources did not (Google's
// freshness dates). Docs already suspect or pending are being handled, and a
// commit that cannot be read is reported by the suspect check.
func staleDocs(docs []*Doc, pl *places, f Freshness, now time.Time, skip func(string) bool) []verdict.Finding {
	if f.StaleAfterDays <= 0 {
		return []verdict.Finding{{Rule: "setting-missing", Level: "block",
			Message: "freshness.stale-after-days is not set, so no doc was checked for freshness; set it, or turn the rule off with `enforce`"}}
	}
	var out []verdict.Finding
	for _, d := range docs {
		if skip(d.Path) {
			continue
		}
		var newest time.Time
		var commit string
		for name, checked := range d.Checked {
			where, err := pl.get(name)
			if err != nil || checked == "" || checked == "HEAD" {
				continue
			}
			ct, err := git(where.dir, "log", "-1", "--format=%ct", checked+"^{commit}", "--")
			if err != nil {
				continue
			}
			sec, err := strconv.ParseInt(ct, 10, 64)
			if err != nil {
				continue
			}
			if t := time.Unix(sec, 0); t.After(newest) {
				newest, commit = t, checked
			}
		}
		if newest.IsZero() {
			continue
		}
		days := int(now.Sub(newest).Hours() / 24)
		if days > f.StaleAfterDays {
			out = append(out, verdict.Finding{Rule: "stale", Where: d.Path,
				Message: fmt.Sprintf("last confirmed %d days ago (checked: %s, %s), past freshness.stale-after-days (%d): read it again against its sources and what it describes, then move `checked`",
					days, commit, newest.UTC().Format("2006-01-02"), f.StaleAfterDays)})
		}
	}
	return out
}
