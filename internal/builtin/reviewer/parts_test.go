package reviewer

import (
	"fmt"
	"strings"
	"testing"
)

// bodyLines are the lines of a part's diff that are the file's, not a
// header: what every part together must hold once each.
func bodyLines(parts [][]unit) []string {
	var out []string
	for _, p := range parts {
		for _, u := range p {
			for _, l := range strings.SplitAfter(u.text, "\n") {
				if l == "" || u.whole {
					if u.whole && l != "" {
						out = append(out, l)
					}
					continue
				}
				if strings.HasPrefix(l, "@@") || strings.HasPrefix(l, "diff --git") || strings.HasPrefix(l, "--- ") || strings.HasPrefix(l, "+++ ") || strings.HasPrefix(l, "new file") || strings.HasPrefix(l, "index ") {
					continue
				}
				out = append(out, l)
			}
		}
	}
	return out
}

func newFileDiff(path string, n int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\nnew file mode 100644\nindex 0000000..1111111\n--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n", path, path, path, n)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "+line %d\n", i)
	}
	return b.String()
}

// TestPackCutsAHunkToFillAPart: a hunk longer than what is left of a part
// is cut to fill it; each piece its file's header and a hunk header of its
// own, numbered where its lines are; every line read once (#446).
func TestPackCutsAHunkToFillAPart(t *testing.T) {
	units := fileDiffs(newFileDiff("a.go", 10) + newFileDiff("b.go", 40))
	if len(units) != 2 || units[0].path != "a.go" || units[1].path != "b.go" {
		t.Fatalf("units = %+v", units)
	}
	parts := pack(units, 0, 30)
	if len(parts) != 3 {
		t.Fatalf("parts = %d, want 3: a.go and the head of b.go, then two more pieces of b.go", len(parts))
	}
	got := bodyLines(parts)
	if len(got) != 50 {
		t.Fatalf("lines read = %d, want 50, each once", len(got))
	}
	for _, p := range parts {
		lines := 0
		for _, u := range p {
			lines += u.lines()
		}
		if lines > 30 {
			t.Errorf("a part holds %d lines, past 30", lines)
		}
	}
	second := parts[1][0].text
	if !strings.Contains(second, "+++ b/b.go\n@@ -0,0 +") || !strings.HasPrefix(second, "diff --git a/b.go b/b.go\n") {
		t.Errorf("a piece lacks its header: %q", second[:80])
	}
	// The second piece starts where the first ended.
	first := parts[0][1].text
	n := strings.Count(first, "+line ")
	if want := fmt.Sprintf("@@ -0,0 +%d,", n+1); !strings.Contains(second, want) {
		t.Errorf("second piece's hunk header: want %q in %q", want, second)
	}
}

// TestPackKeepsAFileThatFitsAPartWhole: a file that fits a part of its own
// is not cut to fill the one before.
func TestPackKeepsAFileThatFitsAPartWhole(t *testing.T) {
	units := fileDiffs(newFileDiff("a.go", 20) + newFileDiff("b.go", 20))
	parts := pack(units, 0, 30)
	if len(parts) != 2 || len(parts[0]) != 1 || len(parts[1]) != 1 {
		t.Fatalf("parts = %d, want a.go then b.go, each whole", len(parts))
	}
}

// TestPackCutsAWholeFileByLines: a file given whole, larger than a part,
// is cut by its lines, each piece saying which.
func TestPackCutsAWholeFileByLines(t *testing.T) {
	var body strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&body, " line %03d of the file\n", i)
	}
	u := unit{path: "c.go", whole: true, label: "c.go", text: body.String()}
	parts := pack([]unit{u}, u.size()/3, 0)
	if len(parts) < 3 {
		t.Fatalf("parts = %d, want 3 or more", len(parts))
	}
	if got := bodyLines(parts); len(got) != 100 {
		t.Fatalf("lines read = %d, want 100", len(got))
	}
	if l := parts[1][0].label; !strings.HasPrefix(l, "c.go, lines ") || !strings.HasSuffix(l, " of 100") {
		t.Errorf("label = %q", l)
	}
	if l := parts[len(parts)-1][0].label; !strings.Contains(l, "-100 of 100") {
		t.Errorf("last label = %q", l)
	}
}

// TestPackReadsALineLongerThanAPartAlone: never a loop, never a line lost.
func TestPackReadsALineLongerThanAPartAlone(t *testing.T) {
	long := strings.Repeat("x", 500)
	diff := "diff --git a/d.go b/d.go\n--- a/d.go\n+++ b/d.go\n@@ -1,2 +1,2 @@\n-" + long + "\n+" + long + "y\n"
	parts := pack(fileDiffs(diff), 100, 0)
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want each line alone", len(parts))
	}
	if got := bodyLines(parts); len(got) != 2 {
		t.Fatalf("lines read = %d, want 2", len(got))
	}
	if !strings.Contains(parts[1][0].text, "@@ -2,0 +1,1 @@") {
		t.Errorf("second piece: %q", parts[1][0].text[:60])
	}
}

// TestPackOneDiffWithNoHunkWhole: a file whose diff has no hunk (binary)
// is read as it is, alone.
func TestPackOneDiffWithNoHunkWhole(t *testing.T) {
	diff := "diff --git a/e.png b/e.png\nindex 1..2 100644\nBinary files a/e.png and b/e.png differ\n"
	parts := pack(fileDiffs(diff), 10, 1)
	if len(parts) != 1 || parts[0][0].text != diff {
		t.Fatalf("parts = %+v", parts)
	}
}

// TestFileDiffsJoinAsTheDiff: split by file, the diff reads back as it
// came, so a change that fits one call is asked as before.
func TestFileDiffsJoinAsTheDiff(t *testing.T) {
	diff := newFileDiff("a.go", 3) + "diff --git a/b.go b/b.go\n--- a/b.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-gone\n"
	units := fileDiffs(diff)
	var b strings.Builder
	for _, u := range units {
		b.WriteString(u.text)
	}
	if b.String() != diff || units[1].path != "b.go" {
		t.Fatalf("joined = %q, paths %q %q", b.String(), units[0].path, units[1].path)
	}
}
