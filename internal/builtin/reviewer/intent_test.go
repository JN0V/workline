package reviewer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// The issues a change closes, as GitHub and GitLab read the keywords; a
// reference closes nothing.
func TestCloses(t *testing.T) {
	got := closes("Closes #4.\n\nfixes: #7, Resolved #4", "Refs #126\nimplements #9\nsee owner/repo#3 and close#5")
	if fmt.Sprint(got) != "[4 7 9]" {
		t.Errorf("closes = %v, want [4 7 9]", got)
	}
}

// A claim is found in the author's words only, never in the head the
// engine writes; a trailer is left out, a dashed key of the author kept.
func TestSaidByAuthor(t *testing.T) {
	said := testimonyHead + "### The commits\n\n- abc fix: x\n  Follow-up: no change in behaviour.\n"
	if len(locate(saidByAuthor(said), "Check each claim against the code.")) != 0 {
		t.Error("the head read as the author's")
	}
	if len(locate(saidByAuthor(said), "no change in behaviour.")) == 0 {
		t.Error("the author's words not found")
	}
	if !trailer.MatchString("  Co-Authored-By: A <a@b>") || trailer.MatchString("  Follow-up: no change") {
		t.Error("trailers")
	}
}

// On a merge request the turn runs over every lens, one with nothing to
// read passed over: a run count names the same place whether an issue is
// closed or not.
func TestLensesInTurnPassOver(t *testing.T) {
	t.Setenv("WORKLINE_EVENT", "merge-request")
	s := Settings{Lenses: []string{"correctness", "intent", "claims"}, LensesPerPush: 1}
	skip := map[string]string{"intent": "no issue"}
	for runs, want := range []string{"correctness", "claims", "claims", "correctness"} {
		got, _, err := lenses(t.TempDir(), s, Record{Runs: runs}, skip)
		if err != nil || len(got) != 1 || got[0] != want {
			t.Errorf("runs %d: %v %v, want %s", runs, got, err, want)
		}
	}
}

// An issue is given by its Need, Verification and Scope; one without them,
// whole.
func TestIssueText(t *testing.T) {
	text := issueText(&forge.Issue{ID: 4, Title: "Mean", Body: "## Need\n\nA mean.\n\n## Validation\n\nRead it.\n\n## Scope\n\ncalc/\n"})
	if !strings.Contains(text, "#### Need\n\nA mean.") || !strings.Contains(text, "#### Scope\n\ncalc/") || strings.Contains(text, "Read it.") {
		t.Errorf("sections:\n%s", text)
	}
	if text := issueText(&forge.Issue{ID: 5, Title: "Bug", Body: "It crashes on []."}); !strings.Contains(text, "It crashes on [].") {
		t.Errorf("no section, the body whole:\n%s", text)
	}
}
