package devmark

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// DraftReady says whether the developer's Draft is ready for review at
// head: its record holds no cap reached, nor its agent's failure that no
// round followed, and the project's checks there are all done and green;
// with no project check on the forge, its last proof green. Not ready,
// kept says why when a person is the one to mark it, "" else.
func DraftReady(f forge.Forge, a Act, head string) (ready bool, kept string) {
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

// DraftHolder says who acts next on mr, the developer's open Draft, when
// no round is due on it (#1271): a person, why said, when its record
// holds a cap reached, its agent's failure no round followed, or its
// rounds handed off; "" when it is the developer's, its next round woken
// by its checks' end, a red one, or a comment of a person's.
func DraftHolder(f forge.Forge, mr forge.MergeRequest) (string, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: mr.ID})
	if err != nil {
		return "", err
	}
	body, ok := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	a, read := ReadAct(body)
	switch {
	case !ok || !read:
		return "", ErrRecordUnread
	case a.Handed:
		return fmt.Sprintf("past round %d, the developer handed it to a person", a.Round), nil
	}
	_, why := DraftReady(f, a, "")
	return why, nil
}

// ConflictFailed says a merge round failed at head, as the developer's
// record on mr keeps it: no second one there, a person's.
func ConflictFailed(f forge.Forge, mr forge.MergeRequest, head string) (bool, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: mr.ID})
	if err != nil {
		return false, err
	}
	body, ok := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	a, read := ReadAct(body)
	return ok && read && a.Conflict == head, nil
}
