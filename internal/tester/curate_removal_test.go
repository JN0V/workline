package tester

import (
	"regexp"
	"strings"
	"testing"
)

// curatorTask is the curator's task for a folder with no test measured.
func curatorTask(t *testing.T) string {
	t.Helper()
	return strings.ToLower(curateTaskText(t.TempDir(), "internal/x", nil, nil, History{}, Settings{RemovalsMax: 3}))
}

// Test kept for a past fix: Given a test whose commit message links a
// closed bug "#N" and which covers no line that another test misses,
// When the tester gardens the corpus, Then it proposes removing it.
func TestCurateTaskRemovesTestOfClosedBug(t *testing.T) {
	task := curatorTask(t)
	for _, want := range []string{"commit message", "closed", "bug", "#"} {
		if !strings.Contains(task, want) {
			t.Errorf("the curator's task does not say %q: a test whose commit message links a closed bug is removed when no other test misses its lines", want)
		}
	}
	if !strings.Contains(task, "no other test misses") && !strings.Contains(task, "no line that another test misses") &&
		!strings.Contains(task, "covers no line") {
		t.Errorf("the curator's task does not hold the removal to a test covering no line another test misses")
	}
}

// Test without a link: Given a test with no closed-bug link in its commit
// message, When the tester gardens the corpus, Then it never proposes
// removing it as obsolete.
func TestCurateTaskNeverRemovesTestWithoutLink(t *testing.T) {
	task := curatorTask(t)
	if !strings.Contains(task, "obsolete") {
		t.Fatalf("the curator's task does not speak of an obsolete test")
	}
	re := regexp.MustCompile(`(?s)(without|no)\b[^.]{0,80}\blink[^.]{0,200}\bnever|\bnever[^.]{0,200}(without|no)\b[^.]{0,80}\blink`)
	if !re.MatchString(task) {
		t.Errorf("the curator's task does not say a test with no closed-bug link is never proposed for removal as obsolete")
	}
}
