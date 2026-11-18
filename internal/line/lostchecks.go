package line

// The checks GitHub lost (#1719): an open line pull request whose head
// carries no check of the project's, a quarter of an hour after its push,
// had its `opened` event lost. The reconciler brings the checks back by
// pushing the branch merged with its base as the App; read with no agent.
// A reconcile that only lists (CI's read token) pushes nothing: it says.

import (
	"fmt"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// ChecksLostAfter is how long a head may go with no check from the
// project's CI before the line takes it as lost.
const ChecksLostAfter = 15 * time.Minute

// checksBack walks the developer's open pull requests and brings back the
// checks of each whose head has none; it returns how many it pushed.
func checksBack(res *Result, f forge.Forge, o engine.Options) int {
	m, okM := f.(forge.Merger)
	pc, okC := f.(forge.ProjectChecked)
	if !okM || !okC || forge.KeepsBranches(f) || o.NoApply || o.DryRun {
		return 0
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return 0 // said by the rounds' read
	}
	n := 0
	for _, mr := range mrs {
		if !mr.Here || mr.Draft || !strings.HasPrefix(mr.Branch, developer.Branch) {
			continue
		}
		where := fmt.Sprintf("merge request %d", mr.ID)
		st, err := m.MergeState(mr)
		if err != nil || st.Head == "" || len(st.Pushes) == 0 {
			continue
		}
		last, err := time.Parse(time.RFC3339, st.Pushes[len(st.Pushes)-1].At)
		if err != nil || clock().Sub(last) < ChecksLostAfter {
			continue
		}
		cs, err := pc.ProjectChecks(st.Head)
		if err != nil || len(cs) > 0 {
			continue // unread, or its checks are there, pending or done
		}
		if o.Inputs[ReconcileInput] == "list" {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "checks-lost", Level: "warn", Where: where,
				Message: "its head has no check of the project's, past " + ChecksLostAfter.String() + " since its push: the opened event was lost; a reconcile that writes brings them back"})
			continue
		}
		pushed, said, err := engine.BringChecksBack(o.Repo, mr)
		switch {
		case err != nil:
			res.Findings = append(res.Findings, verdict.Finding{Rule: "checks-lost", Level: "warn", Where: where,
				Message: "its head has no check of the project's, and they could not be brought back: " + err.Error()})
		case pushed:
			n++
			res.Findings = append(res.Findings, verdict.Finding{Rule: "checks-back", Level: "info", Where: where, Message: said})
		case said != "":
			res.Findings = append(res.Findings, verdict.Finding{Rule: "checks-lost", Level: "warn", Where: where, Message: said})
		}
	}
	return n
}
