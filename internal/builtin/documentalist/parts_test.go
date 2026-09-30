package documentalist

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func claimOf(t *testing.T, part int, text string) claim {
	t.Helper()
	var c claim
	if err := yaml.Unmarshal([]byte(text), &c); err != nil {
		t.Fatal(err)
	}
	c.part = part
	return c
}

func TestQuotedAtIgnoresWhitespaceOnly(t *testing.T) {
	doc := "line one\nAccess tokens   last\n  one hour.\nline four\n"
	if !quotedAt(doc, lineRange{2, 3}, "Access tokens last one hour.") {
		t.Fatal("a quote spread over the lines it cites, spaced otherwise, is found")
	}
	if quotedAt(doc, lineRange{3, 4}, "Access tokens last one hour.") {
		t.Fatal("a quote not at the lines it cites is not found")
	}
	if quotedAt(doc, lineRange{4, 9}, "line four") {
		t.Fatal("lines past the end are cited by no quote")
	}
}

func TestLineRangeReadsNumbersAndRanges(t *testing.T) {
	for text, want := range map[string]lineRange{"10": {10, 10}, `"12-14"`: {12, 14}, `"3 – 5"`: {3, 5}} {
		var r lineRange
		if err := yaml.Unmarshal([]byte(text), &r); err != nil || r != want {
			t.Errorf("%s: got %v %v, want %v", text, r, err, want)
		}
	}
	var r lineRange
	if err := yaml.Unmarshal([]byte(`"14-12"`), &r); err == nil {
		t.Error("a range ending before it starts is refused")
	}
}

// The parts' claims are put together by the doc's lines: contradicted and
// supported by no other part is wrong; supported by another part, a
// conflict; partial, to be read together; supported alone, nothing to fix.
func TestFixEntryPutsClaimsTogether(t *testing.T) {
	v := &partsVerdict{kept: []claim{
		claimOf(t, 0, `{lines: "3", status: contradicted, quote: a, source: {path: x.go, lines: "1", quote: p}, why: wrong alone}`),
		claimOf(t, 0, `{lines: "5-6", status: contradicted, quote: b, source: {path: x.go, lines: "2", quote: q}, why: disputed}`),
		claimOf(t, 1, `{lines: "6", status: supported, quote: b, source: {path: y.go, lines: "3", quote: r}}`),
		claimOf(t, 1, `{lines: "8", status: partial, quote: c, why: the rest elsewhere}`),
		claimOf(t, 1, `{lines: "9", status: supported, quote: d, source: {path: y.go, lines: "4", quote: s}}`),
	}}
	entry := v.fixEntry("docs/a.md", "1\n2\n3\n4\n5\n6\n7\n8\n9\n", t.TempDir())
	wrong, _, _ := strings.Cut(entry, "### Found wrong by one part")
	for _, want := range []string{"### Found wrong\n\n- The doc, line 3", "Why: wrong alone"} {
		if !strings.Contains(wrong, want) {
			t.Errorf("the wrong passages lack %q:\n%s", want, entry)
		}
	}
	if strings.Contains(wrong, "lines 5-6") {
		t.Errorf("a contradiction another part supports is not simply wrong:\n%s", entry)
	}
	for _, want := range []string{"### Found wrong by one part, and supported by another\n\n- The doc, lines 5-6", "Supported by y.go, line 3", "### To be read together", "- The doc, line 8"} {
		if !strings.Contains(entry, want) {
			t.Errorf("the fix task lacks %q:\n%s", want, entry)
		}
	}
	if strings.Contains(entry, "line 9") {
		t.Errorf("a passage supported alone needs no fix:\n%s", entry)
	}
	only := &partsVerdict{kept: []claim{claimOf(t, 0, `{lines: "9", status: supported, quote: d, source: {path: y.go, lines: "4", quote: s}}`)}}
	if e := only.fixEntry("docs/a.md", "x\n", t.TempDir()); e != "" {
		t.Errorf("nothing wrong, nothing to fix; got:\n%s", e)
	}
}

func TestUncoveredNamesWhatNoPartSpokeOf(t *testing.T) {
	doc := "---\nsources: [x.go]\n---\n# A\n\nCall `Refresh` first.\n\nThen `Revoke`.\n\n```\n`Inside` a block\n```\n"
	kept := []claim{claimOf(t, 0, `{lines: "6", status: supported, quote: x, source: {path: x.go, lines: "1", quote: y}}`)}
	got := uncovered(doc, kept)
	if got != "line 8 names `Revoke`" {
		t.Fatalf("uncovered = %q, want only line 8: line 6 is spoken of, and code blocks name nothing", got)
	}
}

func TestJudgedInPartsPatch(t *testing.T) {
	got, err := judgedInPartsPatch("d.md", "---\nsources: [x]\nchecked: abc1234\n---\n# D\n", "def5678")
	want := "--- a/d.md\n+++ b/d.md\n@@ -4,1 +4,2 @@\n----\n+judged-in-parts: def5678\n+---\n"
	if err != nil || got != want {
		t.Fatalf("got %q %v, want %q", got, err, want)
	}
	got, _ = judgedInPartsPatch("d.md", "---\njudged-in-parts: abc1234\n---\n", "def5678")
	if !strings.Contains(got, "@@ -2,1 +2,1 @@\n-judged-in-parts: abc1234\n+judged-in-parts: def5678\n") {
		t.Fatalf("an earlier record is replaced, not added to: %q", got)
	}
}
