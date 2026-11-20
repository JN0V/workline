package documentalist

import (
	"strings"
	"testing"
)

func refFindings(r Reader, doc string) (msg string, rules map[string]bool) {
	tree := Tree{Docs: map[string]string{"docs/a.md": doc}, Files: map[string]bool{"docs/a.md": true}}
	r.Pages = []string{"docs/**"}
	rules = map[string]bool{}
	for _, p := range ReaderChecks(tree, r) {
		rules[p.Rule] = true
		if p.Rule == "reference-unlinked" {
			msg = p.Message
		}
	}
	return msg, rules
}

func TestReaderCustomReferencePattern(t *testing.T) {
	r := Reader{References: []string{`RFC-\d+`}}
	msg, _ := refFindings(r, "# A\n\nSee RFC-12 for why.\n")
	if !strings.Contains(msg, "`RFC-12`") {
		t.Errorf("bare RFC-12 not flagged: %q", msg)
	}
	msg, _ = refFindings(r, "# A\n\nSee ADR-3 and #5 for why.\n")
	if msg != "" {
		t.Errorf("ADR-3 flagged under a custom pattern: %q", msg)
	}
}

func TestReaderCustomReferenceKeepsItsOwnFirstCharacter(t *testing.T) {
	r := Reader{References: []string{`\[\d+\]`}}
	msg, _ := refFindings(r, "# A\n\nSee [12] and \"RFC-1\".\n")
	if !strings.Contains(msg, "`[12]`") {
		t.Errorf("pattern text altered: %q", msg)
	}
	r = Reader{References: []string{`RFC-\d+`}}
	msg, _ = refFindings(r, "# A\n\nCited as \"RFC-12\" and —RFC-13.\n")
	if !strings.Contains(msg, "2 decision") {
		t.Errorf("quote and dash boundaries missed: %q", msg)
	}
}

func TestReaderInvalidReferencePatterns(t *testing.T) {
	for _, bad := range []string{`(`, ``, `\d*`, `x?`} {
		r := Reader{References: []string{bad}}
		msg, rules := refFindings(r, "# A\n\nSome prose here.\n")
		if !rules["setting-invalid"] {
			t.Errorf("%q: not said invalid", bad)
		}
		if msg != "" {
			t.Errorf("%q: prose flagged: %q", bad, msg)
		}
	}
}
