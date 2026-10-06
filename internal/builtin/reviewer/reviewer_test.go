package reviewer

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/builtin/committer"
)

func TestComment(t *testing.T) {
	for _, c := range []struct {
		file, line, text string
		ok               bool
	}{
		{"a.go", "\t// The bug was here.", "The bug was here.", true},
		{"a.go", "x := 1 // used to be 2", "used to be 2", true},
		{"a.go", `s := "a // b"`, "", false},
		{"a.py", "# previously None", "previously None", true},
		{"a.go", "x := 1", "", false},
		{"a.sql", "-- old", "old", true},
		{"Makefile", "# a target", "a target", true},
	} {
		text, ok, _ := comment(c.file, c.line)
		if ok != c.ok || text != c.text {
			t.Errorf("comment(%q, %q) = %q, %v; want %q, %v", c.file, c.line, text, ok, c.text, c.ok)
		}
	}
}

func TestLongCommentCountsWholeLines(t *testing.T) {
	var added []Line
	for i := range 12 {
		added = append(added, Line{"a.go", i + 1, fmt.Sprintf("\tField%d int // what it holds", i)})
	}
	c := Change{Added: map[string][]Line{"a.go": added}}
	found, err := mechanical(c, Settings{CommentBlockMax: 8}, committer.Settings{})
	if err != nil || len(found) != 0 {
		t.Errorf("comments after code are no block: %v %v", found, err)
	}
	for i := range added {
		added[i].Text = "// a line of a long comment"
	}
	if found, _ := mechanical(c, Settings{CommentBlockMax: 8}, committer.Settings{}); len(found) != 1 || found[0].Where != "a.go:1" {
		t.Errorf("twelve comment lines: %v", found)
	}
}

func TestStoryOverTwoLines(t *testing.T) {
	c := Change{Added: map[string][]Line{"a.go": {
		{"a.go", 7, "// Args splits the range. Fields used"},
		{"a.go", 8, "// to drop the empty words."},
		{"a.go", 9, "func Args() {}"},
	}}}
	found, err := mechanical(c, Settings{StoryWords: []string{`\bused to\b`}}, committer.Settings{})
	if err != nil || len(found) != 1 || found[0].Rule != "bug-story" || found[0].Where != "a.go:7" {
		t.Errorf("a story over two lines of one comment: %v %v", found, err)
	}
}

func TestParseDiffInsideAHunk(t *testing.T) {
	diff := "diff --git a/q.sql b/q.sql\n--- a/q.sql\n+++ b/q.sql\n@@ -3,2 +3,2 @@\n--- an old comment\n-select 1;\n+++ a new one\n+select 2;\n" +
		"diff --git a/gone.go b/gone.go\ndeleted file mode 100644\n--- a/gone.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-package gone\n"
	c := parseDiff(diff)
	if got := c.Added["q.sql"]; len(got) != 2 || got[0] != (Line{"q.sql", 3, "++ a new one"}) || got[1].At != 4 {
		t.Errorf("added: %v", c.Added)
	}
	if got := c.Removed["q.sql"]; len(got) != 2 || got[0].Text != "-- an old comment" {
		t.Errorf("removed: %v", c.Removed)
	}
	if got := c.Removed["gone.go"]; len(got) != 1 || got[0].Text != "package gone" {
		t.Errorf("a deleted file's lines: %v", c.Removed)
	}
}

// A removal — a hunk removing more lines than it adds — is placed in the
// new file, and the kept lines within three of it count as the change;
// those further do not, nor those beside a replacement or an addition.
func TestExposedBesideARemoval(t *testing.T) {
	diff := "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -6,3 +5,0 @@\n-x\n-y\n-z\n@@ -20,3 +18 @@\n-a\n-b\n-c\n+new\n" +
		"@@ -40 +38 @@\n-old\n+new\n@@ -50,0 +48 @@\n+added\n"
	c := parseDiff(diff)
	if got := c.Removals["a.go"]; !slices.Equal(got, []Place{{6, 5}, {18, 18}}) {
		t.Errorf("removals: %v", got)
	}
	for n, want := range map[int]bool{2: false, 3: true, 8: true, 9: false, 14: false, 15: true, 21: true, 22: false, 37: false, 47: false} {
		if c.exposedAt("a.go", n) != want {
			t.Errorf("line %d exposed: %v, want %v", n, !want, want)
		}
	}
}

func TestLocate(t *testing.T) {
	lines := []string{"func A() {", "\treturn  x / y", "}", "func B() {", "\treturn x / y", "}", "", "if !ok {", "\treturn w", "}"}
	for _, c := range []struct {
		quote string
		want  []Place
	}{
		{"return x / y", []Place{{2, 2}, {5, 5}}},       // one line, spaces aside
		{"func B() {\n  return x / y", []Place{{4, 5}}}, // two lines
		{"if !ok { return w }", []Place{{8, 10}}},       // three lines quoted as one
		{"}\nfunc B", []Place{{3, 4}}},                  // across a line break
		{"return x / z", nil},
	} {
		if got := locate(lines, c.quote); !slices.Equal(got, c.want) {
			t.Errorf("%q: %v, want %v", c.quote, got, c.want)
		}
	}
}

func TestRecord(t *testing.T) {
	r := Record{}.add([]string{"aa", "bb"}).add([]string{"bb", "cc"})
	back := parseRecord("text before\n" + r.String() + "\nafter")
	if !slices.Equal(back.Reviewed, []string{"aa", "bb", "cc"}) || back.Runs != 2 {
		t.Errorf("record read back: %+v", back)
	}
	if empty := parseRecord("no record here"); len(empty.Reviewed) != 0 || empty.Runs != 0 {
		t.Errorf("no record: %+v", empty)
	}
}

func TestAIFindingsByLens(t *testing.T) {
	lenses := `"lenses": ["correctness", "edge-cases", "tests"]`
	for _, c := range []struct {
		value  string
		blocks []string // the lenses whose finding blocks
		err    bool
	}{
		{`"warn"`, nil, false},
		{`"block"`, []string{"correctness", "edge-cases", "tests"}, false},
		{`{"correctness": "block", "edge-cases": "warn"}`, []string{"correctness"}, false},
		{`{}`, nil, false},
		{`null`, nil, false},
		{`"blocks"`, nil, true},
		{`""`, nil, true},
		{`{"correctness": "on"}`, nil, true},
		{`{"correctnes": "block"}`, nil, true},
		{`["correctness"]`, nil, true},
	} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
			t.Fatal(err)
		}
		data := fmt.Sprintf(`{%s, "ai-findings": %s}`, lenses, c.value)
		if err := os.WriteFile(filepath.Join(dir, "in", "settings.json"), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		s, err := settings(dir)
		if (err != nil) != c.err {
			t.Errorf("ai-findings %s: error %v, want one: %v", c.value, err, c.err)
			continue
		}
		if c.err {
			continue
		}
		for _, lens := range []string{"correctness", "edge-cases", "tests"} {
			if got, want := s.AIFindings.Blocks(lens), slices.Contains(c.blocks, lens); got != want {
				t.Errorf("ai-findings %s: the %s lens blocks = %v, want %v", c.value, lens, got, want)
			}
		}
	}
}
