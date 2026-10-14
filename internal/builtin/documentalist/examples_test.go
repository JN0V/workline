package documentalist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A fenced block a fix adds or changes is an example it writes; one moved
// unchanged is not, nor one whose only change brings a line count the
// engine found off to its number (#121).
func TestExamplesWritten(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "Clock.h"), []byte(strings.Repeat("x\n", 569)), 0o644); err != nil {
		t.Fatal(err)
	}
	files := map[string]bool{"src/Clock.h": true}
	head := "---\nsources: [src/Clock.h]\nchecked: abc1234\n---\n# Clock\n\n"
	listing := "```\nsrc/\n    Clock.h           (524 lines)   # the clock\n```\n"
	example := "```cpp\nClock clock;\nclock.begin();\n```\n"
	for _, c := range []struct {
		name, old, now, want string
	}{
		{"added", head + "Some prose.\n", head + "Some prose.\n\n" + example, "9-12 cpp"},
		{"changed", head + example, head + strings.Replace(example, "begin", "start", 1), "7-10 cpp"},
		{"moved unchanged", head + example + "\nProse.\n", head + "Prose.\n\n" + example, ""},
		{"prose only", head + "One hour.\n" + example, head + "Two hours.\n" + example, ""},
		{"a count brought to the engine's", head + listing, head + strings.Replace(listing, "524", "569", 1), ""},
		{"a count and code", head + listing, head + strings.Replace(strings.Replace(listing, "524", "569", 1), "the clock", "the time", 1), "7-10"},
		{"left open", head + "Prose.\n", head + "Prose.\n\n```sh\nmake\n", "9-10 sh"},
	} {
		var got []string
		for _, f := range examplesWritten(dir, files, "d.md", c.old, c.now) {
			got = append(got, strings.TrimSpace(fmtFence(f)))
		}
		if strings.Join(got, "; ") != c.want {
			t.Errorf("%s: examples written %q, want %q", c.name, got, c.want)
		}
	}
}

func fmtFence(f fence) string {
	return strings.TrimSpace(strings.Join([]string{linesOf([]fence{{from: f.from, to: f.to}}, false), f.lang}, " "))
}

// The places a fix changed are counted without the examples it wrote,
// said apart; a doc whose fix only wrote an example is fixed in no place.
func TestPlacesSaidApartFromExamples(t *testing.T) {
	head := "---\nsources: [src/a.go]\nchecked: abc1234\n---\n# A\n\n"
	before := head + "One hour.\n\nMore.\n"
	example := "```go\nttl := 7200\n```\n"
	for _, c := range []struct {
		name, now, want string
	}{
		{"prose and an example apart", head + "Two hours.\n\n" + example + "\nMore.\n", "1 place and 1 code example"},
		{"an example only", head + "One hour.\n\n" + example + "\nMore.\n", "1 code example"},
		{"prose only", head + "Two hours.\n\nMore, said again.\n", "2 places"},
	} {
		j := &judgedFixes{before: map[string]string{"d.md": before}, after: map[string]string{"d.md": c.now},
			examples: map[string][]fence{}}
		if w := examplesWritten(t.TempDir(), nil, "d.md", before, c.now); len(w) > 0 {
			j.examples["d.md"] = w
		}
		if got := j.placesSaid("d.md"); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// Each changed line of a diff is placed in the doc it makes.
func TestNewLineOf(t *testing.T) {
	old := "a\nb\nc\nd\n"
	files, err := parseDiff("--- a/d.md\n+++ b/d.md\n@@ -1,3 +1,4 @@\n a\n+x\n+y\n-b\n c\n")
	if err != nil {
		t.Fatal(err)
	}
	at := newLineOf(old, placed(old, files[0]))
	if at[[2]int{0, 1}] != 2 || at[[2]int{0, 2}] != 3 || at[[2]int{0, 3}] != 4 {
		t.Errorf("new lines %v, want x at 2, y at 3, b's place at 4", at)
	}
}
