package report

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/line"
)

// The default branch's issue holds main's findings and, with no release
// merge request open, the night's dry run's part: each writer keeps the
// other's as it is (ADR-0050).
func TestDryRunPartKeptInMainIssue(t *testing.T) {
	d := &line.DryRun{Head: "abcdef1234", Since: "v1.0.0", Holds: []line.Hold{{Step: "documentalist", Rule: "suspect", Where: "docs/a.md", Message: "src/a.go changed"}}}
	body, changed := WithDryRun("", d)
	if !changed || !strings.Contains(body, "### The next release") || !strings.Contains(body, "`docs/a.md` `documentalist suspect`") {
		t.Fatalf("the part not put in a new issue:\n%s", body)
	}
	if again, changed := WithDryRun(body, d); changed || again != body {
		t.Errorf("the same holds, the next night: changed")
	}
	moved := *d
	moved.Head = "fedcba9876"
	if _, changed := WithDryRun(body, &moved); changed {
		t.Errorf("the branch moved, the same holds: changed")
	}

	// Main's run after a merge rewrites the issue: the part stays, after its own.
	main := &line.Result{Event: line.MainEvent, Main: &line.MainRun{Head: "1234567890"}}
	rewritten, _, _ := MainIssue(main, body)
	if !strings.Contains(rewritten, "`docs/a.md` `documentalist suspect`") || !strings.Contains(rewritten, mainMarker) {
		t.Fatalf("main's run dropped the dry run's part:\n%s", rewritten)
	}
	if strings.Index(rewritten, mainMarker) > strings.Index(rewritten, dryRunStart) {
		t.Errorf("the dry run's part comes before main's own")
	}

	// Nothing holds any more: the part taken out, main's own left.
	none := &line.DryRun{Head: "fedcba9876", Since: "v1.0.0"}
	out, changed := WithDryRun(rewritten, none)
	if !changed || strings.Contains(out, dryRunStart) || !strings.Contains(out, mainMarker) {
		t.Errorf("the part not taken out, or main's lost:\n%s", out)
	}
	if _, changed := WithDryRun(out, none); changed {
		t.Errorf("nothing held, nothing holds: changed")
	}

	// An issue holding the part alone says nothing holds, still the issue
	// workline keeps.
	alone, changed := WithDryRun(body, none)
	if !changed || !strings.Contains(alone, "Nothing would hold the release now") || !strings.Contains(alone, dryRunEnd) {
		t.Errorf("an issue of the part alone, emptied:\n%s", alone)
	}
	if _, changed := WithDryRun(alone, none); changed {
		t.Errorf("said already that nothing holds: changed")
	}
}

// The comment on the release merge request is posted once something
// holds, edited when what it lists changes, never for a moved head alone.
func TestDryRunComment(t *testing.T) {
	none := &line.DryRun{Head: "abc1234", MergeRequest: 3}
	if _, changed, any := DryRunComment(none, ""); changed || any {
		t.Errorf("nothing holds, no comment: changed %v, any %v", changed, any)
	}
	d := &line.DryRun{Head: "abc1234", MergeRequest: 3, Holds: []line.Hold{{Step: "gate:release", Rule: "check-failed", Where: "versions", Message: "exit 1 @someone <b>"}}}
	body, changed, any := DryRunComment(d, "")
	if !changed || !any || !strings.Contains(body, "on this release's head") {
		t.Fatalf("something holds, no comment:\n%s", body)
	}
	if strings.Contains(body, "@someone") || strings.Contains(body, "<b>") {
		t.Errorf("a run's text not defused:\n%s", body)
	}
	if _, changed, _ := DryRunComment(&line.DryRun{Head: "def5678", MergeRequest: 3, Holds: d.Holds}, body); changed {
		t.Errorf("the head moved alone: changed")
	}
	if b, changed, _ := DryRunComment(none, body); !changed || !strings.Contains(b, "nothing would hold") {
		t.Errorf("nothing holds any more, not said:\n%s", b)
	}
}
