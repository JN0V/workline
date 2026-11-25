package reviewer

// The scenarios the tester said no test proves, as the developer's pull
// request lists them under `## Unproven`: given to the tests lens, which
// finds a test missing where one can prove the scenario. The usual path
// settles it — the tester adds the test or says none is needed, the
// judge weighs —, a person only for what the judge refuses.

import (
	"os"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// developerBranch opens the branches of the developer's pull requests:
// only there is the list the engine's own.
const developerBranch = "workline/developer/"

// unprovenAsked marks a record whose tests lens was given the list
// once: never asked again, once the tester and the judge settled it.
const unprovenAsked = "<!-- workline:unproven=asked -->"

// unprovenOf is the list the developer's pull request the run reviews
// holds, cut past limit lines as the author's testimony is; "" on another
// branch, with no forge, or none listed.
func unprovenOf(repo string, limit int) string {
	t := mergeRequest()
	if t == nil {
		return ""
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil || f == nil {
		return ""
	}
	mr, err := f.MergeRequest(t.ID)
	if err != nil || !strings.HasPrefix(mr.Branch, developerBranch) {
		return ""
	}
	list, _ := cutLines(unprovenListed(mr.Body), limit, "the unproven scenarios are cut here, past testimony-lines-max")
	return list
}

// unprovenHeading is the developer's heading for them.
const unprovenHeading = "Unproven"

// testsLens is the lens they are given to.
const testsLens = "tests"

// unprovenListed are the bullets under `## Unproven` in the pull
// request's text, "" for none.
func unprovenListed(text string) string {
	var out []string
	in := false
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "#"):
			in = strings.TrimSpace(strings.TrimLeft(t, "#")) == unprovenHeading
		case in && strings.HasPrefix(t, "- "):
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

// unprovenText is what the tests lens is told of them, after its own
// question; "" for none.
func unprovenText(list string) string {
	if list == "" {
		return ""
	}
	return "\n\nThe tester said no test proves these scenarios of the issue, each with its reason. " +
		"For each the change makes testable, a test missing is a finding of this lens, Important: its cause the code that does the scenario, quoted, " +
		"the scenario named. None where the reason shows no test can check it.\n\n" + list
}
