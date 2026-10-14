package documentalist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// The project's command runs on the fixed docs, the doc as its argument;
// one that does not end in time has failed, and the run goes on.
func TestCheckExamples(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-qm", "docs: a"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	docs := map[string]string{"a.md": "# A\n\n```sh\nmake\n```\n"}
	if r := checkExamples(repo, `grep -q make "$1"`, docs, []string{"a.md"}); !r.ok {
		t.Errorf("a command passing on the fixed doc: %+v", r)
	}
	if r := checkExamples(repo, `echo "$1: no rule make"; exit 2`, docs, []string{"a.md"}); r.ok || r.out != "a.md: no rule make" {
		t.Errorf("a command failing: %+v, want its output", r)
	}
	was := exampleTimeout
	exampleTimeout = 500 * time.Millisecond
	defer func() { exampleTimeout = was }()
	if r := checkExamples(repo, "exec sleep 30", docs, nil); r.ok || !strings.Contains(r.out, "did not end within") {
		t.Errorf("a command that does not end: %+v", r)
	}
	if out, _ := git(repo, "worktree", "list"); strings.Count(out, "\n") > 0 {
		t.Errorf("checkouts left behind: %s", out)
	}
}
