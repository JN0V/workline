package backlog

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestFollowCarriesLinesThroughADiff(t *testing.T) {
	cases := []struct {
		name       string
		diff       string
		from, to   int
		nfrom, nto int
		changed    bool
	}{
		{"a line added above", "@@ -2,0 +3 @@\n", 7, 8, 8, 9, false},
		{"a line removed above", "@@ -2 +1,0 @@\n", 7, 8, 6, 7, false},
		{"a line added after them", "@@ -8,0 +9 @@\n", 7, 8, 7, 8, false},
		{"a line added between them", "@@ -7,0 +8 @@\n", 7, 8, 7, 9, true},
		{"their last line rewritten in two", "@@ -8 +8,2 @@\n", 7, 8, 7, 9, true},
		{"their first line rewritten", "@@ -7 +7 @@\n", 7, 8, 7, 8, true},
		{"the file deleted", "@@ -1,10 +0,0 @@\n", 7, 8, 1, -1, true},
		{"elsewhere, below", "@@ -12,2 +12,3 @@\n", 7, 8, 7, 8, false},
	}
	for _, c := range cases {
		nf, nt, ch := follow(hunks(c.diff), c.from, c.to)
		if nf != c.nfrom || nt != c.nto || ch != c.changed {
			t.Errorf("%s: follow = %d, %d, %v; want %d, %d, %v", c.name, nf, nt, ch, c.nfrom, c.nto, c.changed)
		}
	}
}

func TestRewrittenReadsOnlyAPersonsChange(t *testing.T) {
	body := "Intro.\n\n## Need\n\n" + DraftLine("product-owner") + "\n\nEvery row  exported.\n\n## Scope\n\nsrc/a.go\nBlocked by #3. <!-- workline:blocked-by -->"
	st := &State{}
	st.Keep(body)
	if got := Rewritten(st, "Intro, fixed.\n\n## Need\n\nEvery row exported.\n\n## Scope\n\nsrc/a.go"); len(got) != 0 {
		t.Errorf("a typo above, spaces, the engine's lines gone: Rewritten = %v, want none", got)
	}
	if got := Rewritten(st, "## Need\n\nEvery row exported as JSON.\n\n## Scope\n\nsrc/b.go"); !slices.Equal(got, []string{"Need", "Scope"}) {
		t.Errorf("Rewritten = %v, want Need and Scope", got)
	}
	if got := Rewritten(&State{}, "## Need\n\nAnything."); got != nil {
		t.Errorf("a state that kept no sections: Rewritten = %v, want none", got)
	}
	if got := Rewritten(&State{Sections: map[string]string{"Scope": "src/a.go"}}, "## Need\n\nWritten now.\n\n## Scope\n\nsrc/a.go"); got != nil {
		t.Errorf("a Need written where there was none: Rewritten = %v, want none", got)
	}
}

func TestImportedReadsItsFileAndLines(t *testing.T) {
	body := "1. Keep the last row.\n\nOpened from `ROADMAP.md`, lines 7 to 8 by the product-owner role.\n\n<!-- workline:import=ROADMAP.md:0a1b2c3d4e5f -->"
	path, from, to, ok := Imported(body)
	if !ok || path != "ROADMAP.md" || from != 7 || to != 8 {
		t.Errorf("Imported = %q %d %d %v", path, from, to, ok)
	}
	if _, _, _, ok := Imported("Opened from `ROADMAP.md`, lines 7 to 8 by the product-owner role."); ok {
		t.Error("no import key: not an imported issue")
	}
}

func TestSectionsAndLinesOfOneIssueAreTwoChanges(t *testing.T) {
	p := &Plan{hand: &Hand{}, open: map[int]bool{4: true, 5: true}}
	p.readChanges([]Change{
		{Issue: 4, What: []string{"Need"}, Touch: []Touch{{Issue: 5, How: TouchPart}}},
		{Issue: 4, Path: "ROADMAP.md", Lines: "7 to 9", Touch: []Touch{{Issue: 4, How: TouchImport}}},
	})
	if len(p.Record.Changes) != 2 {
		t.Errorf("changes kept = %v, want both: one does not replace the other", p.Record.Changes)
	}
}

func TestTouchedPartsFirstThenWaitingThenSharing(t *testing.T) {
	open := []forge.Issue{
		{ID: 9, Children: []int{11}},
		{ID: 11},
		{ID: 12, BlockedBy: []int{9}},
		{ID: 13, Body: "Blocked by #9."},
		{ID: 14},
		{ID: 15},
	}
	sources := map[int][]string{9: {"src/a.go#F"}, 11: {"src/a.go"}, 14: {"src/a.go"}, 15: {"README.md"}}
	got := Touched(9, open, sources, true)
	want := []Touch{{Issue: 11, How: TouchPart}, {Issue: 12, How: TouchWaits}, {Issue: 13, How: TouchWaits}, {Issue: 14, How: TouchSources, Files: "src/a.go"}}
	if !slices.Equal(got, want) {
		t.Errorf("its Scope changed: Touched = %v, want %v", got, want)
	}
	if got := Touched(9, open, sources, false); !slices.Equal(got, want[:3]) {
		t.Errorf("its Need alone changed: Touched = %v, want %v", got, want[:3])
	}
}
