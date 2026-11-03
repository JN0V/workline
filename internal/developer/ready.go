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
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// draftReady says whether the developer's Draft is ready for review at
// head: its record holds no cap reached, nor its agent's failure that no
// round followed, and the project's checks there are all done and green;
// with no project check on the forge, its last proof green. Not ready,
// kept says why when a person is the one to mark it, "" else.
func draftReady(f forge.Forge, a Act, head string) (ready bool, kept string) {
	switch {
	case len(a.Caps) > 0:
		names := make([]string, len(a.Caps))
		for i, c := range a.Caps {
			names[i] = c.Setting
		}
		return false, "its record holds a cap reached (" + strings.Join(names, ", ") + "): a person marks it ready"
	case a.AgentFailed != "" && a.Round == 0:
		return false, "its agent failed (" + a.AgentFailed + "), and no round followed: a person marks it ready"
	case head == "":
		return false, ""
	}
	pc, ok := f.(forge.ProjectChecked)
	if !ok {
		return false, ""
	}
	cs, err := pc.ProjectChecks(head)
	if err != nil {
		return false, "" // said as the checks unread
	}
	for _, c := range cs {
		if c.State != "success" {
			return false, "" // red: a round; pending: its end wakes one
		}
	}
	if len(cs) == 0 {
		return a.Proof != nil && a.Proof.Result == Green, ""
	}
	return true, ""
}

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
