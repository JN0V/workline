package tester

import (
	"testing"

	"github.com/JN0V/workline/internal/backlog"
)

func TestRefused(t *testing.T) {
	sticky := "\n<!-- workline:sticky=tester/issue -->"
	for _, c := range []struct {
		name     string
		comments []string
		want     string
	}{
		{"refused after the agent", []string{backlog.Said("The tester", refusedRun, "passes on the base.", "next") + sticky}, "passes on the base"},
		{"refused before the agent", []string{backlog.Said("The tester", refusedPre, "no named scenario", "next") + sticky}, "no named scenario"},
		{"its tests written", []string{backlog.Said("The tester", "wrote the acceptance tests of this issue, as a Draft", "each scenario", "next") + sticky}, ""},
		{"a person's words, no marker", []string{backlog.Said("The tester", refusedRun, "passes on the base", "next")}, ""},
		{"the last word counts", []string{
			backlog.Said("The tester", refusedRun, "passes on the base", "next") + sticky,
			backlog.Said("The tester", "wrote the acceptance tests of this issue, as a Draft", "each scenario", "next") + sticky}, ""},
		{"none", nil, ""},
	} {
		if got := Refused(c.comments); got != c.want {
			t.Errorf("%s: Refused = %q, want %q", c.name, got, c.want)
		}
	}
}
