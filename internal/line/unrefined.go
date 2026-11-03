package line

// An issue left unrefined (ADR-0059, amended for #951): the product
// owner read it in refine, on its body as it is, and left it lacking a
// section — nothing drafted nor asked, nothing waiting on a person. Its
// run is not "acted": the line wakes it again on the same body, at most
// RefineRetries times, then blocks the issue, the reason on it. Never on
// the same code as its last read (#980): that read again would get the
// same input; the issue is blocked at once, the code it lacks said.

import (
	"fmt"
	"slices"
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
// sources: the code its state names now, the next read's.
func unrefined(is forge.Issue, notes []forge.Note) (lacks string, sources []string) {
	st, found, err := backlog.ReadState(forge.Bodies(notes), backlog.Keeper)
	if !found || err != nil || st.Body == "" || st.Body != backlog.BodyDigest(is.Body) {
		return "", nil // not read on this body: its first read is due, or a person's change
	}
	if st.Label != "" || st.Left != "" || st.Aside != "" || len(st.Split) > 0 || backlog.Readied(is) || backlog.HasDraft(is.Body, st) {
		return "", nil
	}
	if len(backlog.Unheard(st, notes, is)) > 0 {
		return "", nil // a person wrote since its read: theirs is read next
	}
	if _, asked := backlog.Unanswered(notes, backlog.Keeper); asked {
		return "", nil // its question waits on its reporter: the delay wakes it (ADR-0060)
	}
	missing := backlog.Missing(is.Body)
	if len(missing) == 0 {
		return "", nil
	}
	return strings.Join(missing, ", "), st.Sources
}

// givenKey is what Record.Given keeps of the code a read is given: its
// sources, sorted, one a line (no path holds a newline), "-" for none.
func givenKey(sources []string) string {
	if len(sources) == 0 {
		return "-"
	}
	s := slices.Clone(sources)
	slices.Sort(s)
	return strings.Join(s, "\n")
}

// givenNow is the code the product owner's state names on an issue, as
// Record.Given keeps it: "" when its state does not read.
func givenNow(comments []string) string {
	st, found, err := backlog.ReadState(comments, backlog.Keeper)
	if err != nil {
		return ""
	}
	if !found {
		return givenKey(nil)
	}
	return givenKey(st.Sources)
}

// codeSaid says the code a read was given, for the line's reason: the
// paths its sources and its text name.
func codeSaid(shown []string) string {
	if len(shown) == 0 {
		return "no code: its text names no file of the repository, nor did the product owner"
	}
	return "only the code of `" + strings.Join(slices.Compact(slices.Sorted(slices.Values(shown))), "`, `") + "`"
}

// refineAgain is the record of an issue woken again for being left
// unrefined, and what the line says of it; past RefineRetries, or on the
// same code its last read had (sources), the reason it is blocked
// instead, the code it was given said (shown).
func refineAgain(rec Record, lacks string, sources, shown []string) (Record, string, string) {
	if rec.Given != "" && rec.Given == givenKey(sources) {
		return rec, "", fmt.Sprintf("the product owner read it on this text with %s, and left it lacking %s, nothing asked: read again, it would get the same; a person names the files it is about in a comment, or completes it",
			codeSaid(shown), lacks)
	}
	if rec.Retry >= RefineRetries {
		return rec, "", fmt.Sprintf("the product owner read it %d times on the same text and left it lacking %s, nothing asked: a person completes it, or names its files in a comment",
			rec.Retry+1, lacks)
	}
	rec.Retry++
	return rec, fmt.Sprintf("the product owner's last read left it lacking %s, nothing asked: woken again, %d of %d", lacks, rec.Retry, RefineRetries), ""
}
