package line

// A verdict blocked at the tester's green proof (#1502): no role reads
// the verdict's tester step, so the block lived only in the judge job's
// log. The line says it on the pull request, once a head, and its
// issue waits for a named person: the proof could not run here.

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// testerStep is how the verdict names the tester's step.
const testerStep = "(tester block)"

// testerBlock reads the line's verdict on mr's head: why, and the head,
// when it is blocked at the tester's step; ok false when it is not, or
// is not read.
func testerBlock(b forge.Backlog, repo string, mr forge.MergeRequest) (why, head string, ok bool) {
	m, isMerger := b.(forge.Merger)
	if !isMerger || mr.Draft {
		return "", "", false
	}
	head = TipOf(repo, mr.Branch)
	if head == "" {
		return "", "", false
	}
	class, said, err := m.Verdict(head)
	if err != nil || class != "red" || !strings.Contains(said, testerStep) {
		return "", "", false
	}
	return said, head, true
}

// testerBlockSaid comments on mr that its verdict is blocked at the
// tester's step, once for a head (its marker).
func testerBlockSaid(b forge.Backlog, repo string, mr forge.MergeRequest) error {
	f, ok := b.(forge.Forge)
	if !ok {
		return nil
	}
	why, head, ok := testerBlock(b, repo, mr)
	if !ok {
		return nil
	}
	body := fmt.Sprintf("**workline's line** — the verdict on %s is blocked at the tester step, the green proof of the change: %s\n\n"+
		"No role reads this step: a person decides whether the proof can run here, or the developer changes the code.",
		short(head), why)
	return f.Comment(forge.Target{Kind: "merge-request", ID: mr.ID}, backlog.Inert(body), forge.Marker("tester-block="+head))
}
