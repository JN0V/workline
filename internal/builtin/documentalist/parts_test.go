package documentalist

import (
	"fmt"
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

// Two claims speak of the same words when their quotes overlap where they
// stand in the doc, not when their lines merely do (#440): a sentence spans
// lines, and the words beside it are another passage.
func TestSameWordsNotSameLines(t *testing.T) {
	doc := "# Service\n" +
		"At most 100 requests are served at once; past that, the service answers\n" +
		"`503 Service Unavailable` rather than queueing them; the service logs it. Each\n" +
		"one lasts the configured session TTL, 24 hours by default.\n"
	at := func(lines, quote string) claim {
		return claimOf(t, 0, fmt.Sprintf("{lines: %q, status: supported, quote: %q}", lines, quote))
	}
	for _, tc := range []struct {
		a, b claim
		same bool
	}{
		{at("2", "At most 100 requests are served at once"), at("2-3", "past that, the service answers `503 Service Unavailable`"), false},
		{at("2", "At most 100 requests are served at once"), at("2-3", "served at once; past that"), true},
		{at("3-4", "Each one lasts the configured session TTL"), at("4", "24 hours by default"), false},
		{at("3-4", "Each one lasts the configured session TTL"), at("4", "the configured session TTL, 24 hours"), true},
		{at("3", "the service logs it"), at("2-3", "the service"), true}, // there twice: the second place counts
		{at("2", "not in the doc"), at("2", "At most 100"), true},        // a quote not placed: the lines decide
		{at("2", "not in the doc"), at("4", "24 hours by default"), false},
	} {
		if got := sameWords(doc, tc.a, tc.b); got != tc.same {
			t.Errorf("%q (%s) and %q (%s): same words %v, want %v", tc.a.Quote, tc.a.Lines, tc.b.Quote, tc.b.Lines, got, tc.same)
		}
		if got := sameWords(doc, tc.b, tc.a); got != tc.same {
			t.Errorf("%q and %q, the other way round: same words %v, want %v", tc.b.Quote, tc.a.Quote, got, tc.same)
		}
	}
}

// A quote a part read and judged partial for a passage is no proof the
// passage is wrong (#467), unless it stands elsewhere in the code too, or
// a part found those words wrong.
func TestPartialQuoteIsNoProof(t *testing.T) {
	src := "package s\n" + // 1
		"\n" + // 2
		"func topic(c Config) string {\n" + // 3
		"\tt := fmt.Sprintf(\"%s/%s/availability\",\n" + // 4
		"\t\tc.Prefix, c.Node)\n" + // 5
		"\treturn adopt(t) // fmt.Sprintf(\"%s/%s/will\")\n" + // 6
		"}\n" // 7
	quote := `"%s/%s/availability", c.Prefix, c.Node)`
	if got := quoteLines("s.go", src, quote); len(got) != 1 || got[0] != (lineRange{4, 5}) {
		t.Fatalf("the quote stands at lines 4-5, whitespace aside: got %v", got)
	}
	if got := quoteLines("s.go", src, `fmt.Sprintf("%s/%s/will")`); len(got) != 0 {
		t.Fatalf("a place wholly in a comment is not where a quote stands in the code: got %v", got)
	}
	block := citedBlock{from: 12, to: 12}
	for _, tc := range []struct {
		name    string
		partial []partialCite
		quote   string
		refused bool
	}{
		{"the quote a part judged partial", []partialCite{{Lines: "12", Path: "s.go", Source: "4-5"}}, quote, true},
		{"read wider than the quote", []partialCite{{Lines: "11-13", Path: "s.go", Source: "3-6"}}, quote, true},
		{"read narrower than the quote", []partialCite{{Lines: "12", Path: "s.go", Source: "4"}}, quote, false},
		{"another passage of the doc", []partialCite{{Lines: "20", Path: "s.go", Source: "4-5"}}, quote, false},
		{"another file", []partialCite{{Lines: "12", Path: "t.go", Source: "4-5"}}, quote, false},
		{"lines the part did not cite", []partialCite{{Lines: "12", Path: "s.go", Source: "4-5"}}, "return adopt(t)", false},
		{"no part judged it partial", nil, quote, false},
	} {
		cc := &citeContext{partial: map[string][]partialCite{"d.md": tc.partial}}
		why := cc.partialOnly("d.md", "s.go", src, tc.quote, block)
		if (why != "") != tc.refused {
			t.Errorf("%s: refused %v (%q), want %v", tc.name, why != "", why, tc.refused)
		}
	}
	// A quote standing elsewhere in the code too may be what the fix read.
	twice := src + "var x = fmt.Sprintf(\"%s/%s/availability\", c.Prefix, c.Node)\n"
	cc := &citeContext{partial: map[string][]partialCite{"d.md": {{Lines: "12", Path: "s.go", Source: "4-5"}}}}
	if why := cc.partialOnly("d.md", "s.go", twice, quote, block); why != "" {
		t.Errorf("a quote standing outside the lines a part judged partial is refused: %s", why)
	}
}

// Only a partial claim on lines no part found anything wrong in is kept,
// with its source lines, for the judge of the fix (#467).
func TestPartialCitesLeaveWhatAPartFoundWrong(t *testing.T) {
	v := &partsVerdict{kept: []claim{
		claimOf(t, 0, `{lines: "2", status: partial, quote: "At most 100 requests", source: {path: a.go, lines: "8", quote: "Max = 100"}}`),
		claimOf(t, 0, `{lines: "2-3", status: contradicted, quote: "a 503", source: {path: a.go, lines: "8", quote: "Max = 100"}}`), // other words, its quote among the same lines
		claimOf(t, 0, `{lines: "4", status: partial, quote: "the configured session TTL", source: {path: a.go, lines: "10-11", quote: "TTL = 24"}}`),
		claimOf(t, 1, `{lines: "4", status: partial, quote: "Each one lasts"}`),
		claimOf(t, 1, `{lines: "4", status: supported, quote: "Each one lasts", source: {path: b.go, lines: "17", quote: "ttl: c.TTL"}}`),
	}}
	got := v.partialCites()
	want := []partialCite{{Lines: "4", Path: "a.go", Source: "10-11"}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("partial cites: got %v, want %v", got, want)
	}
	if lines, src, ok := got[0].ranges(); !ok || lines != (lineRange{4, 4}) || src != (lineRange{10, 11}) {
		t.Fatalf("read back: %v %v %v", lines, src, ok)
	}
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

// The parts' claims are put together by the doc's words (here, quotes the
// doc does not hold: by its lines, TestSameWordsNotSameLines): contradicted and
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
