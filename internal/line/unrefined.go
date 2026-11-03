package line

// An issue left unrefined (ADR-0059, amended for #951): the product
// owner read it in refine, on its body as it is, and left it lacking a
// section — nothing drafted nor asked, nothing waiting on a person. Its
// run is not "acted": the line wakes it again on the same body, at most
// RefineRetries times, then blocks the issue, the reason on it.

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// RefineRetries is how many times the line wakes the product owner
// again on an issue it left unrefined, its body the same: past it, the
// issue is blocked for a person.
const RefineRetries = 2

// unrefined says what an issue in refine lacks when the product owner's
// last read of its body left it so, nothing waiting on a person: ""
// when it was not read on this body, is complete, or waits on a person —
// a question, drafts, a proposal, a split, a person's comment, set aside.
func unrefined(is forge.Issue, notes []forge.Note) string {
	st, found, err := backlog.ReadState(forge.Bodies(notes), backlog.Keeper)
	if !found || err != nil || st.Body == "" || st.Body != backlog.BodyDigest(is.Body) {
		return "" // not read on this body: its first read is due, or a person's change
	}
	if st.Label != "" || st.Left != "" || st.Aside != "" || len(st.Split) > 0 || backlog.Readied(is) || backlog.HasDraft(is.Body, st) {
		return ""
	}
	if _, asked := backlog.Unanswered(notes, backlog.Keeper); asked {
		return "" // its question waits on its reporter: the delay wakes it (ADR-0060)
	}
	lacks := backlog.Missing(is.Body)
	if len(lacks) == 0 {
		return ""
	}
	return strings.Join(lacks, ", ")
}

// refineAgain is the record of an issue woken again for being left
// unrefined, and what the line says of it; past RefineRetries, the
// reason it is blocked instead.
func refineAgain(rec Record, lacks string) (Record, string, string) {
	if rec.Retry >= RefineRetries {
		return rec, "", fmt.Sprintf("the product owner read it %d times on the same text and left it lacking %s, nothing asked: a person completes it, or names its files in a comment",
			rec.Retry+1, lacks)
	}
	rec.Retry++
	return rec, fmt.Sprintf("the product owner's last read left it lacking %s, nothing asked: woken again, %d of %d", lacks, rec.Retry, RefineRetries), ""
}
