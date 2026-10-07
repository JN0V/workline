package backlog

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestParts(t *testing.T) {
	is := forge.Issue{ID: 9, Children: []int{12, 12, 9}, Body: "Text.\n\n- [ ] #30 not a part\n\n## Sub-issues\n\n- [x] #11\n- [ ] #12\n* [ ] #9"}
	if got := Parts(is); !slices.Equal(got, []int{11, 12}) {
		t.Errorf("Parts = %v, want [11 12]: the relation and the Sub-issues list; itself and other lists left out", got)
	}
	if got := SplitInto(is, &State{Split: []int{13, 11}}); !slices.Equal(got, []int{11, 12, 13}) {
		t.Errorf("SplitInto = %v, want [11 12 13]: the parts and the state's split", got)
	}
	if got := Parts(forge.Issue{ID: 5, Children: []int{7, 7}}); !slices.Equal(got, []int{7}) {
		t.Errorf("Parts = %v, want [7]: a child the forge lists twice is one part", got)
	}
	if got := Parts(forge.Issue{ID: 4, Body: "No parts."}); len(got) != 0 {
		t.Errorf("Parts = %v, want none", got)
	}
}

func TestVerificationItems(t *testing.T) {
	cases := []struct {
		body string
		want []string
	}{
		{"## Verification\n\nConformance cases:\n- a parent gets one comment;\n1. A test passes.\n- [ ] A box item\n\n## Scope\n\nx", []string{"a parent gets one comment;", "A test passes.", "A box item"}},
		{"### Verification\n\nA test writes three rows\nand reads them back.", []string{"A test writes three rows and reads them back."}},
		{"## Verification\n\n_No response_", nil},
		{"No sections.", nil},
	}
	for _, c := range cases {
		if got := VerificationItems(c.body); !slices.Equal(got, c.want) {
			t.Errorf("VerificationItems(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

func TestProofIsAQuote(t *testing.T) {
	done := Part{ID: 11, Issue: forge.Issue{Closed: true, Reason: "completed", Body: "## Verification\n\nA **test** writes `three` rows."}}
	if proof("A test writes three rows;", []Part{done}) == "" {
		t.Error("a quote, emphasis, case and final punctuation aside, should prove the item")
	}
	if proof("A test writes four rows.", []Part{done}) != "" {
		t.Error("other words prove nothing")
	}
	refused := done
	refused.Issue.Reason = "not_planned"
	if proof("A test writes three rows.", []Part{refused}) != "" {
		t.Error("a part not delivered proves nothing")
	}
}

func TestCloserNotReadIsSaid(t *testing.T) {
	part := Part{ID: 11, Issue: forge.Issue{Closed: true, Reason: "completed", Title: "Keep the last row"}, Unread: errors.New("the forge refused closers: unknown operation")}
	ev := ReadEvidence(forge.Issue{ID: 9}, []Part{part}, "product-owner", nil)
	if !strings.Contains(ev.Body, "| #11 Keep the last row | closed as completed | not read: the forge did not say |") || strings.Contains(ev.Body, "by hand") {
		t.Errorf("a closer the forge refused to say is said not read, never by hand:\n%s", ev.Body)
	}
	if strings.Contains(ev.Body, "unknown operation") {
		t.Errorf("the forge's error stays out of the comment, in the finding:\n%s", ev.Body)
	}
	if len(ev.Unread) != 1 || ev.Unread[0].ID != 11 {
		t.Errorf("Unread = %v, want #11", ev.Unread)
	}
}

func TestTestNames(t *testing.T) {
	cases := []struct {
		item string
		want []string
	}{
		{"`TestWriteRows` passes", []string{"TestWriteRows"}},
		{"the test `keeps-the-last-row` and `src/export/csv_test.go::TestQuote`", []string{"keeps-the-last-row", "src/export/csv_test.go::TestQuote"}},
		{"`tests/export.spec.ts` and `test_quote` cover it", []string{"tests/export.spec.ts", "test_quote"}},
		{"conformance case `parent-x` passes", []string{"parent-x"}},
		{"in that case `WriteRows` returns; the spec `x` says so", nil},
		{"`WriteRows` keeps the last row; `make check` is green", nil},
		{"A test writes three rows.", nil},
	}
	for _, c := range cases {
		if got := TestNames(c.item); !slices.Equal(got, c.want) {
			t.Errorf("TestNames(%q) = %q, want %q", c.item, got, c.want)
		}
	}
}

func TestCodeTestsReadsHead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.MkdirAll(filepath.Join(dir, "tests", "export"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tests", "export", "rows_test.go"), []byte("package export\n\nfunc TestRows(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "test"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	find := CodeTests(dir)
	for name, want := range map[string]string{
		"tests/export/rows_test.go":          "tests/export/rows_test.go",
		"tests/export/rows_test.go::TestRows": "tests/export/rows_test.go",
		"TestRows":                           "tests/export/rows_test.go",
		"tests/export":                       "", // a folder, no test file
		"tests/export/cols_test.go":          "",
		"TestCols":                           "",
	} {
		if got, err := find(name); got != want || err != nil {
			t.Errorf("CodeTests(%q) = %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestCodeTestsUnreadIsNotMissing(t *testing.T) {
	find := CodeTests(t.TempDir()) // not a repository: git fails
	for _, name := range []string{"TestWriteRows", "src/export/csv_test.go"} {
		if path, err := find(name); err == nil {
			t.Errorf("CodeTests(%q) = %q, no error: a git that failed is read as a test missing", name, path)
		}
	}
}
