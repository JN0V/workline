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
