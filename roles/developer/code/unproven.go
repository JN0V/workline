package developer

// The scenarios the tester said no test proves, as its pull request lists
// them under `## Unproven` for the reviewer's tests lens.

import (
	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/kit/testmark"
)

// UnprovenHeading heads the scenarios the tester said no test proves, on
// its pull request: the reviewer's tests lens reads them there.
const UnprovenHeading = "Unproven"

const unprovenLead = "Scenarios of the issue no test proves, as the tester said, each with its reason: the reviewer's tests lens weighs whether a test can prove it."

// unprovenLines are the scenarios its pull request lists, inert: a
// name's @ pings no one.
func unprovenLines(us []testmark.Unproven) []string {
	var out []string
	for _, u := range us {
		l := "- **" + devmark.InertText(u.Scenario) + "**"
		if u.Why != "" {
			l += ": " + devmark.InertText(u.Why)
		}
		out = append(out, l)
	}
	return out
}
