package report

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
)

func gatesRun(checks []gate.CheckCount, found []gate.Finding, own ...verdict.Finding) *line.Result {
	return &line.Result{Steps: []line.Step{{Name: "gate:lint", Status: verdict.Block,
		Gate: &verdict.Verdict{Status: verdict.Block, Findings: own},
		New:  &gate.Added{Base: "2b1f39f0123", Checks: checks, Findings: found}}}}
}

// Many findings of one rule make one entry, its first places named; the
// rules past the reviewer's cap are counted; each finding is beside its
// line but a formatter's, whose command is said; a place outside the
// repository is none; the run's words are defused.
func TestGatesShown(t *testing.T) {
	var found []gate.Finding
	for i := 1; i <= 4; i++ {
		found = append(found, gate.Finding{Check: "golangci-lint", Tool: "golangci-lint", Rule: "errcheck", Level: "error",
			Path: "a.go", Line: i, Message: "Error return value of `os.Remove` is not checked @someone <b>", Fingerprint: fmt.Sprint(i)})
	}
	for i := 1; i <= 11; i++ {
		found = append(found, gate.Finding{Check: "golangci-lint", Tool: "golangci-lint", Rule: fmt.Sprintf("rule%02d", i), Level: "warning", Path: "b.go", Line: i})
	}
	found = append(found,
		gate.Finding{Check: "golangci-lint", Rule: "outside", Level: "error", Path: "../etc/passwd", Line: 1},
		gate.Finding{Check: "format", Tool: "gofmt", Rule: "gofmt", Level: "error", Path: "c.go", Line: 3},
		gate.Finding{Check: "format", Tool: "gofmt", Rule: "gofmt", Level: "error", Path: "d.go", Line: 9})
	g := Gates(gatesRun([]gate.CheckCount{
		{ID: "golangci-lint", Output: "sarif", New: 16, Before: 360},
		{ID: "format", Output: "sarif", New: 2, Before: 0, Fix: "gofmt -w ."},
		{ID: "vet", Output: "exit", Failed: true, Before: -1},
	}, found, verdict.Finding{Rule: "suppression-added", Where: "a.go:9", Message: "the change adds //nolint"}), "abc1234def")
	if !g.Worth() || g.New != 18 || g.Title() != "18 new findings" {
		t.Fatalf("worth %v, new %d, title %q", g.Worth(), g.New, g.Title())
	}
	// 4 errcheck, 11 rules, the suppression: the one outside and the
	// formatter's are not written.
	if len(g.Notes) != 16 || g.Left != 0 {
		t.Fatalf("%d notes, %d left: %+v", len(g.Notes), g.Left, g.Notes)
	}
	for _, n := range g.Notes {
		if n.Path == "c.go" || n.Path == "d.go" || strings.Contains(n.Path, "..") {
			t.Errorf("annotated %+v", n)
		}
	}
	if n := g.Notes[0]; n.Path != "a.go" || n.Line != 1 || n.Level != "failure" || n.Title != "errcheck (golangci-lint)" {
		t.Errorf("first note %+v", n)
	}
	c := g.Comment("https://github.com/o/r/runs/7", 10)
	for _, want := range []string{
		"### Gates: 18 new findings",
		"On `abc1234`, only what this change adds since `2b1f39f` counts",
		"| `lint` | `golangci-lint` | 16 | 360 |",
		"| `lint` | `vet` | did not pass | – |",
		"- **`format`**: 2 new findings in 2 files: run `gofmt -w .`, then push",
		"- **`vet`** did not pass",
		"- **`errcheck`**, 4 new: `a.go:1`, `a.go:2`, `a.go:3`, …. Error return value of `os.Remove` is not checked @\u200bsomeone &lt;b&gt;",
		"- **`rule09`**, 1 new: `b.go:9`",
		"- 4 more rules, 4 findings, not listed here: the reviewer's `findings-max` (10)",
		"the check [workline gates](https://github.com/o/r/runs/7)",
	} {
		if !strings.Contains(c, want) {
			t.Errorf("the comment lacks %q:\n%s", want, c)
		}
	}
	if strings.Contains(c, "@someone") || strings.Contains(c, "rule10") {
		t.Errorf("a mention left as it is, or a rule past the cap listed:\n%s", c)
	}
	// Past no cap, the suppression too; no check written, none named.
	if c := g.Comment("", 20); !strings.Contains(c, "- **`suppression-added`**, 1 new: `a.go:9`. the change adds //nolint") || strings.Contains(c, "the check") {
		t.Errorf("past no cap, the comment:\n%s", c)
	}
	long := Gates(gatesRun([]gate.CheckCount{{ID: "lint", Output: "sarif", New: 1}},
		[]gate.Finding{{Check: "lint", Rule: strings.Repeat("r", 300), Level: "error", Path: "a.go", Line: 1}}), "abc1234")
	if n := long.Notes[0]; len(n.Title) > 255 {
		t.Errorf("a title of %d bytes: GitHub takes 255", len(n.Title))
	}
}

// With no new finding and every check passing, there is nothing to say:
// the comment, edited only where a push left one, says so.
func TestGatesShownNothingNew(t *testing.T) {
	g := Gates(gatesRun([]gate.CheckCount{{ID: "golangci-lint", Output: "sarif", Before: 360}}, nil), "abc1234")
	if g.Worth() || len(g.Notes) != 0 {
		t.Fatalf("worth %v, %d notes", g.Worth(), len(g.Notes))
	}
	if c := g.Comment("", 10); !strings.Contains(c, "### Gates: no new finding") || !strings.Contains(c, "| `lint` | `golangci-lint` | 0 | 360 |") {
		t.Errorf("comment:\n%s", c)
	}
}

// Past the annotations a check is written with, the rest are counted.
func TestGatesShownCapped(t *testing.T) {
	var found []gate.Finding
	for i := range maxAnnotations + 7 {
		found = append(found, gate.Finding{Check: "lint", Rule: "errcheck", Level: "error", Path: "a.go", Line: i + 1})
	}
	g := Gates(gatesRun([]gate.CheckCount{{ID: "lint", Output: "sarif", New: len(found)}}, found), "abc1234")
	if len(g.Notes) != maxAnnotations || g.Left != 7 || !strings.Contains(g.Summary(), "7 more are not written here") {
		t.Errorf("%d notes, %d left; summary %q", len(g.Notes), g.Left, g.Summary())
	}
}
