package reviewer

import (
	"fmt"
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
