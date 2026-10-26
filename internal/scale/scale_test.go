package scale

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The two scales of ADR-0040, in the order's direction, and where none
// sorts: a priority after Low, an effort as Medium.
func TestScales(t *testing.T) {
	if !slices.Equal(Priority.Names, []string{"Urgent", "High", "Medium", "Low"}) || !slices.Equal(Effort.Names, []string{"Low", "Medium", "High"}) {
		t.Fatalf("scales = %v, %v", Priority.Names, Effort.Names)
	}
	if Priority.Rank(0) != Priority.Count()+1 || Priority.Rank(4) != 4 || Priority.Rank(9) != 5 {
		t.Errorf("priority ranks: none %d, Low %d", Priority.Rank(0), Priority.Rank(4))
	}
	if Effort.Rank(0) != 2 || Effort.Name(Effort.Rank(0)) != "Medium" {
		t.Errorf("an effort not estimated ranks %d, want Medium", Effort.Rank(0))
	}
	if Priority.Name(0) != "" || Priority.Name(5) != "" || Priority.Valid(0) || !Priority.Valid(4) || Effort.Valid(4) {
		t.Error("a level outside the scale has no name and is not valid")
	}
}

// Words are read by name, whatever their case: the levels' own, and those
// projects and forges use for them; a name no table knows is none.
func TestWords(t *testing.T) {
	for _, c := range []struct {
		s    Scale
		word string
		want int
	}{
		{Priority, "Urgent", 1}, {Priority, "urgent", 1}, {Priority, "Critical", 1}, {Priority, "critical-urgent", 1},
		{Priority, "HIGH", 2}, {Priority, "important-soon", 2}, {Priority, "important", 2},
		{Priority, "Medium", 3}, {Priority, "important-longterm", 3},
		{Priority, "Low", 4}, {Priority, "backlog", 4},
		{Effort, "Low", 1}, {Effort, "small", 1}, {Effort, "hours", 1},
		{Effort, "medium", 2}, {Effort, "days", 2},
		{Effort, "High", 3}, {Effort, "large", 3}, {Effort, "x-large", 3}, {Effort, "weeks", 3},
	} {
		if l, ok := c.s.Word(c.word); !ok || l != c.want {
			t.Errorf("%s %q = %d, %v; want %d", c.s.Axis, c.word, l, ok, c.want)
		}
	}
	for _, w := range []string{"awaiting-more-evidence", "Big", "", "small"} {
		if l, ok := Priority.Word(w); ok {
			t.Errorf("priority %q = %d; want none", w, l)
		}
	}
}

// A field's options are read by name, never by their position: an Effort
// field listing High first sorts High last.
func TestOptionsByName(t *testing.T) {
	f, err := Options(Effort, []string{"High", "Medium", "Low"})
	if err != nil {
		t.Fatal(err)
	}
	if f.Of("High") != 3 || f.Of("Low") != 1 || Effort.Rank(f.Of("High")) <= Effort.Rank(f.Of("Medium")) {
		t.Errorf("High = %d, Low = %d; High sorts last", f.Of("High"), f.Of("Low"))
	}
	if l, _ := f.Label(1); l.Name != "Low" {
		t.Errorf("effort Low is written %q", l.Name)
	}
	p, err := Options(Priority, []string{"Urgent", "High", "Medium", "Low"})
	if err != nil || p.Of("Urgent") != 1 || p.Of("Low") != 4 {
		t.Errorf("GitHub's Priority options: %+v, %v", p.Labels, err)
	}
}

// An option renamed to a name no table knows carries no level, whatever
// its place, and is said: never read as the level its position held.
func TestOptionRenamedFailsLoud(t *testing.T) {
	f, err := Options(Effort, []string{"Big", "Medium", "Low"})
	if err == nil || !strings.Contains(err.Error(), `"Big"`) || !strings.Contains(err.Error(), "effort") {
		t.Fatalf("err = %v; want it to name the option", err)
	}
	if f.Of("Big") != 0 {
		t.Errorf("Big = %d; want none", f.Of("Big"))
	}
	if _, ok := f.Label(3); ok {
		t.Error("no option carries High")
	}
	if f.Of("Medium") != 2 || f.Of("Low") != 1 {
		t.Errorf("the options known still read: %+v", f.Labels)
	}
}

// A project's label families, read by name: numbers from the family's
// own start, words through the tables.
func TestLabelFamilies(t *testing.T) {
	for _, c := range []struct {
		s      Scale
		prefix string
		labels []string
		want   []int
	}{
		{Priority, "P", []string{"P0", "P1", "P2", "P3"}, []int{1, 2, 3, 4}},
		{Priority, "P", []string{"P1", "P2", "P3", "P4"}, []int{1, 2, 3, 4}},
		{Priority, "P", []string{"P0", "P1", "P2", "P3", "P4", "P5"}, []int{1, 2, 3, 4, 4, 4}},
		{Priority, "priority::", []string{"priority::1", "priority::2", "priority::3", "priority::4"}, []int{1, 2, 3, 4}},
		{Priority, "Priority:", []string{"Priority:0", "Priority:1", "Priority:2", "Priority:3"}, []int{1, 2, 3, 4}},
		{Priority, "workline:priority/", []string{"workline:priority/2", "workline:priority/3", "workline:priority/4"}, []int{2, 3, 4}},
		{Priority, "priority/", []string{"priority/critical-urgent", "priority/important-soon", "priority/important-longterm", "priority/backlog"}, []int{1, 2, 3, 4}},
		{Priority, "P-", []string{"P-critical", "P-high", "P-medium", "P-low"}, []int{1, 2, 3, 4}},
		{Priority, "Priority/", []string{"Priority/Critical", "Priority/High", "Priority/Medium", "Priority/Low"}, []int{1, 2, 3, 4}},
		{Effort, "effort", []string{"effort1: hours", "effort2: days", "effort3: weeks"}, []int{1, 2, 3}},
		{Effort, "effort/", []string{"effort/small", "effort/medium", "effort/large", "effort/x-large"}, []int{1, 2, 3, 3}},
		{Effort, "effort::", []string{"effort::low", "effort::medium", "effort::high"}, []int{1, 2, 3}},
	} {
		f, err := Labels(c.s, c.prefix, c.labels)
		if err != nil {
			t.Errorf("%s: %v", c.prefix, err)
		}
		var got []int
		for _, l := range c.labels {
			got = append(got, f.Of(l))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s %v = %v; want %v", c.prefix, c.labels, got, c.want)
		}
	}
}

// A family holds only its own labels: a prefix ending in a letter takes
// numbers only; a word no table knows is said and carries none; the first
// label of a level is the one written.
func TestFamilyMembers(t *testing.T) {
	f, err := Labels(Priority, "P", []string{"Performance", "P-critical", "P1a", "P3", "P4", "P5", "bug"})
	if err != nil {
		t.Errorf("only P3 to P5 belong: %v", err)
	}
	if f.Of("Performance") != 0 || f.Of("P-critical") != 0 || f.Of("P1a") != 0 || len(f.Labels) != 3 {
		t.Errorf("family P = %+v", f.Labels)
	}
	if f.Of("P3") != 3 || f.Of("P5") != 4 {
		t.Errorf("P3, P5, with no P0 = %d, %d; want Medium, Low: a family's first label never made moves none", f.Of("P3"), f.Of("P5"))
	}
	if l, _ := f.Label(4); l.Name != "P4" {
		t.Errorf("Low is written %q; want P4, its lowest", l.Name)
	}
	g, err := Labels(Priority, "priority/", []string{"priority/awaiting-more-evidence", "priority/important-soon"})
	if err == nil || !strings.Contains(err.Error(), "priority/awaiting-more-evidence") {
		t.Errorf("err = %v; want it to name the label no table knows", err)
	}
	if g.Of("priority/awaiting-more-evidence") != 0 || g.Of("priority/important-soon") != 2 {
		t.Errorf("family priority/ = %+v", g.Labels)
	}
	h, _ := Labels(Priority, "P", []string{"P10", "P2", "P9", "P1"})
	if l, _ := h.Label(4); l.Name != "P9" {
		t.Errorf("Low = %+v", h.Labels)
	}
	if h.Of("P1") != 1 || h.Of("P2") != 2 || h.Of("P9") != 4 || h.Of("P10") != 4 {
		t.Errorf("numbers compared as numbers: %+v", h.Labels)
	}
}

// GitLab's weight, a whole number: read in three bands, written 1, 3, 5.
func TestWeight(t *testing.T) {
	for w, want := range map[int]int{-1: 0, 0: 1, 1: 1, 2: 2, 3: 2, 4: 3, 5: 3, 8: 3, 13: 3} {
		if got := Weight(w); got != want {
			t.Errorf("weight %d = %d; want %d", w, got, want)
		}
	}
	for l, want := range map[int]int{0: 0, 1: 1, 2: 3, 3: 5, 4: 0} {
		if got := WeightOf(l); got != want {
			t.Errorf("effort %d is written %d; want %d", l, got, want)
		}
	}
	for l := 1; l <= Effort.Count(); l++ {
		if Weight(WeightOf(l)) != l {
			t.Errorf("effort %d does not read back", l)
		}
	}
}

// The engine reads and writes workline's own labels, as before ADR-0040,
// and reads no effort.
func TestUsedIsWorklinesOwn(t *testing.T) {
	f := Used.Priority.Family
	if f.Prefix != "workline:priority/" || len(Used.Effort.Family.Labels) != 0 || Used.Effort.Said != "" {
		t.Fatalf("used = %+v", Used)
	}
	issue := []string{"bug", "workline:priority/3", "workline:priority/0", "workline:priority/5", "workline:priority/2", "priority/high"}
	if got := f.Levels(issue); !slices.Equal(got, []int{3, 2}) || f.First(issue) != 2 || f.First([]string{"bug"}) != 0 {
		t.Errorf("levels = %v, first %d", got, f.First(issue))
	}
	colors := []string{"b60205", "d93f0b", "fbca04", "c5def5"}
	for l := 1; l <= 4; l++ {
		label, ok := f.Label(l)
		want := "workline:priority/" + strconv.Itoa(l)
		if !ok || label.Name != want || label.Color != colors[l-1] {
			t.Errorf("level %d is written %+v", l, label)
		}
	}
	label, _ := f.Label(2)
	if d := label.Describe("product owner"); d != "Priority 2 of 4, 1 the most pressing: set by the product owner or a person" {
		t.Errorf("description = %q", d)
	}
	if got := f.Others(2); !slices.Equal(got, []string{"workline:priority/1", "workline:priority/3", "workline:priority/4"}) {
		t.Errorf("others = %v", got)
	}
	if _, ok := f.Label(0); ok {
		t.Error("no label carries none")
	}
	if Value(2) != "2" || Value(0) != "none" || Said(2) != "priority 2" || Said(0) != "no priority" {
		t.Errorf("said: %q, %q, %q, %q", Value(2), Value(0), Said(2), Said(0))
	}
}

// The table is the one place the engine names a level: no Go file of the
// engine outside this package holds, in a string, a level's name, a label
// of the families it reads, or a priority said as a number by hand — so
// moving to the forge's names is one change here.
func TestLevelsNamedOnlyInTheTable(t *testing.T) {
	var words []string
	for _, s := range []Scale{Priority, Effort} {
		for _, n := range s.Names {
			words = append(words, regexp.QuoteMeta(n))
		}
	}
	named := []*regexp.Regexp{
		regexp.MustCompile(`\b(` + strings.Join(words, "|") + `)\b`),
		regexp.MustCompile(`(?i)priority[: ]+(to )?%d`),
	}
	for _, f := range []Family{Used.Priority.Family, Used.Effort.Family} {
		if f.Prefix != "" {
			named = append(named, regexp.MustCompile(regexp.QuoteMeta(f.Prefix)))
		}
	}
	root := filepath.Join("..", "..")
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
				filepath.ToSlash(filepath.Dir(path)) == "../../internal/scale" {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				text, err := strconv.Unquote(lit.Value)
				if err != nil {
					text = lit.Value
				}
				for _, re := range named {
					if m := re.FindString(text); m != "" {
						t.Errorf("%s: names a level, %q: read it from the table (internal/scale)", fset.Position(lit.Pos()), m)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
