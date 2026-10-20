package backlog

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// The issues workline keeps for its own work are told apart from the
// backlog's (#429): by title for the old report and the weekly sample, by
// the engine's line for an issue a role keeps in place, by its key for a
// demotion; a role's finding, an import and a person's issue are work.
func TestTracking(t *testing.T) {
	for _, c := range []struct {
		is   forge.Issue
		want bool
	}{
		{forge.Issue{Title: "Backlog — product owner"}, true},
		{forge.Issue{Title: "workline: the weekly sample of every role's acts"}, true},
		{forge.Issue{Title: "workline: the weekly sample of the docs vouched for"}, true},
		{forge.Issue{Title: "workline: the weekly sample of the product owner's acts"}, true},
		{forge.Issue{Title: "Docs waiting for a person", Body: "Two docs.\n\nKept up to date by the documentalist role."}, true},
		{forge.Issue{Title: "workline: its merge requests wait", Body: "Close it.\n\n<!-- workline:merge-demoted=abc123 -->"}, true},
		{forge.Issue{Title: "workline: findings on the default branch", Body: "Two.\n\n<!-- workline:main-findings {\"keys\":[]} -->"}, true},
		{forge.Issue{Title: "workline: the gardening fails", Body: "A person's issue, titled so."}, false},
		{forge.Issue{Title: "Rows lost", Body: "Found.\n\n<!-- workline:opened-by=reviewer -->"}, false},
		{forge.Issue{Title: "Roadmap item", Body: "Item.\n\n<!-- workline:import=docs/ROADMAP.md:3 -->"}, false},
		{forge.Issue{Title: "Mentions it", Body: "The line \"Kept up to date by the documentalist role.\" is odd."}, false},
	} {
		if got := Tracking(c.is, "product-owner"); got != c.want {
			t.Errorf("Tracking(%q) = %v, want %v", c.is.Title, got, c.want)
		}
	}
	if !Bot("workline-jn0v[bot]") || !Bot("project_12_bot_ab12") || Bot("ann") {
		t.Error("Bot: a [bot] login and a GitLab project bot are bots, a person is not")
	}
}
