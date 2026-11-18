package developer

// A Draft made green (#969): the developer's pull request opened as a
// Draft — its proof red, not run whole, or its agent failed — is marked
// ready for review once every project check at its branch's head is
// green, no cap reached, as one proven green at first opens ready. Its
// round asks no agent: nothing is left to fix. A cap reached keeps it a
// Draft for a person, said; a check red, pending or cancelled keeps it
// a Draft, the red one a round of its own.

import (
	"fmt"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// readyBody is what the round says on its pull request, marking it ready.
func readyBody(head string) string {
	return fmt.Sprintf("**The developer**: every project check green at `%s`, no cap reached: marked ready for review. "+
		"The reviewer reads it now; the merge policy decides the rest.\n", head[:min(7, len(head))])
}

// readyPost ends a round that marks its Draft ready: the engine marks it
// (out/merge-request.yaml's `ready`), then applies the comment pre left.
func readyPost(runDir string, r round) int {
	if err := writeMergeRequest(runDir, map[string]any{"ready": true}); err != nil {
		return fail(err)
	}
	return writeVerdict(runDir, verdict.Verdict{Status: verdict.Pass,
		Summary: fmt.Sprintf("every project check green at its head, no cap reached: merge request %d marked ready for review", r.MR)})
}

// readied is what pre leaves for the engine when its Draft is ready: the
// round's comment saying so.
func readied(head string) []intent.Intention {
	return []intent.Intention{{Kind: "comment", Value: map[string]any{"sticky": "ready", "body": readyBody(head)}}}
}
