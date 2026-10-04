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

func TestLocate(t *testing.T) {
	lines := []string{"func A() {", "\treturn  x / y", "}", "func B() {", "\treturn x / y", "}"}
	if got := locate(lines, "return x / y"); !slices.Equal(got, []int{2, 5}) {
		t.Errorf("one line, spaces aside: %v", got)
	}
	if got := locate(lines, "func B() {\n  return x / y"); !slices.Equal(got, []int{4}) {
		t.Errorf("two lines: %v", got)
	}
	if got := locate(lines, "return x / z"); got != nil {
		t.Errorf("not there: %v", got)
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
