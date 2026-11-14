package wake

import (
	"fmt"
	"strings"
	"testing"
)

// A heavy week
// Given a range whose triggers would pass the forge's body limit,
// When the wake keeps its issue,
// Then the issue is edited, under the limit, and says what it left out.
func TestIssueHeavyWeekStaysUnderTheLimit(t *testing.T) {
	var entries []Entry
	for i := 0; i < 3000; i++ {
		entries = append(entries, Entry{
			Kind: Contract, Key: fmt.Sprintf("contract:adr:docs/adr/%04d.md", i),
			Text: fmt.Sprintf("ADR added: `docs/adr/%04d-%s.md`", i, strings.Repeat("x", 60)),
			At:   "aaaaaaa1", Day: "2026-01-01",
		})
	}
	body, changed, any := Issue(&Result{Head: "aaaaaaa1", Day: "2026-01-01", Entries: entries}, "")
	if !changed || !any {
		t.Fatalf("changed %v any %v", changed, any)
	}
	if len(body) >= 65536 {
		t.Errorf("the body is %d bytes, not under GitHub's 65536 limit", len(body))
	}
	low := strings.ToLower(body)
	if !strings.Contains(low, "left out") && !strings.Contains(low, "omitted") {
		t.Errorf("the issue does not say what it left out")
	}
}
