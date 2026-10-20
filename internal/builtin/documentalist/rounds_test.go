package documentalist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// A release's earlier rounds (#572): what they asked is not asked again,
// the comment they proposed is the newest, their docs stay in the fix.
func TestEarlierRounds(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	first, second := t.TempDir(), t.TempDir()
	for _, d := range []string{first, second} {
		must(os.MkdirAll(filepath.Join(d, "in"), 0o755))
		must(os.MkdirAll(filepath.Join(d, "out"), 0o755))
	}
	must(os.WriteFile(filepath.Join(first, "in", "judged.yaml"), []byte("docs/a.md: {\"\": abc}\n"), 0o644))
	must(os.WriteFile(filepath.Join(second, "in", "judged.yaml"), []byte("docs/b.md: {\"\": abc}\n"), 0o644))
	must(intent.Write(filepath.Join(first, "out", "intentions.yaml"), []intent.Intention{
		{Kind: "comment", Value: map[string]any{"sticky": releaseKeptKey, "body": "first"}}}))
	must(intent.Write(filepath.Join(second, "out", "intentions.yaml"), []intent.Intention{{Kind: "patch", Value: "--- a/docs/b.md\n+++ b/docs/b.md\n"}}))
	must(writeYAML(filepath.Join(first, "out", "release-fix.yaml"), releaseFix{Title: "docs: bring the docs in line with the code",
		Docs: map[string]string{"docs/a.md": "still true, `checked` moved"}, Withheld: []verdict.Finding{{Rule: "removal-uncited", Where: "docs/a.md"}}}))
	t.Setenv("WORKLINE_EARLIER_ROUNDS", first+" "+second)

	docs := map[string]*suspectDoc{"docs/a.md": {}, "docs/b.md": {}, "docs/c.md": {}, "docs/d.md": {note: "(its own)"}}
	if err := holdEarlierRounds(earlierRounds(), docs); err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]string{"docs/a.md": askedEarlier, "docs/b.md": askedEarlier, "docs/c.md": "", "docs/d.md": "(its own)"} {
		if docs[p].note != want {
			t.Errorf("%s: note %q, want %q", p, docs[p].note, want)
		}
	}

	// The second round proposed no comment: the first's is the newest.
	if body, ok, err := earlierSticky(releaseKeptKey); err != nil || !ok || body != "first" {
		t.Errorf("earlierSticky = %q, %v, %v; want the first round's", body, ok, err)
	}
	if _, ok, _ := earlierSticky("release-night"); ok {
		t.Error("earlierSticky found a comment no round proposed")
	}

	before, err := earlierFix()
	if err != nil || before == nil {
		t.Fatalf("earlierFix = %v, %v", before, err)
	}
	outcome, withheld := before.withEarlier(map[string]string{"docs/c.md": "fixed"}, map[string]map[string]string{"docs/c.md": nil}, nil)
	if outcome["docs/a.md"] == "" || outcome["docs/c.md"] != "fixed" || len(withheld) != 1 {
		t.Errorf("withEarlier = %v, %v: want the earlier doc, this round's, and the earlier place withheld", outcome, withheld)
	}
	if got := before.titled("docs: bring the product docs up to date for the release"); !strings.Contains(got, "what the release waits for") {
		t.Errorf("titled = %q: two kinds of task make one fix", got)
	}
	if got := before.titled("docs: bring the docs in line with the code"); got != "docs: bring the docs in line with the code" {
		t.Errorf("titled = %q: the same task keeps its title", got)
	}
	var none *releaseFix
	if o, w := none.withEarlier(map[string]string{"x": "y"}, nil, nil); len(o) != 1 || w != nil {
		t.Errorf("no earlier round changed the fix: %v, %v", o, w)
	}
}
