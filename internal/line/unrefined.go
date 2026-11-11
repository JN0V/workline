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

// unrefinedWhy opens the reason the line gives an issue it blocks for
// being left unrefined: its record says so when it is read again (#1206).
const unrefinedWhy = "the product owner read it"

// unrefinedBlock says whether the line blocked the issue of rec from
// refine for being left unrefined, not on a run's size: the block a
// rewrite of its text lifts (#1206).
func unrefinedBlock(rec Record) bool {
	return rec.State == BlockedState && rec.From == refineState && rec.SizeOf == "" && rec.Backstop == 0 && strings.HasPrefix(rec.Why, unrefinedWhy+" ")
}

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

// unreadBody says whether the product owner's state on an issue says it
// never read it (#1041): no state, or one never judged, no body read.
// The line woke it, so that run read nothing — an import left for the
// catch-up (#1036) —: not acted. A state that does not read is the
// role's to say (state-broken).
func unreadBody(notes []forge.Note) bool {
	st, found, err := backlog.ReadState(forge.Bodies(notes), backlog.Keeper)
	if err != nil {
		return false
	}
	return !found || st.Judged == "" && st.Body == ""
}

// commentsUnheard is how many of a person's comments issue is holds
// (backlog.Counted), and whether the product owner's state says it has
// not read them all (#1331): an answer to the reviewer's question, or to
// its own. No state: none unheard, its first read due anyway.
func commentsUnheard(is forge.Issue, notes []forge.Note) (int, bool) {
	counted, unheard, _ := commentsHeld(is, notes)
	return counted, unheard
}

// commentsHeld is commentsUnheard, and whether the product owner's last
// run held its `ready` for those comments, nothing of them written into
// the spec (#1331): they are read once more.
func commentsHeld(is forge.Issue, notes []forge.Note) (counted int, unheard, held bool) {
	counted = len(backlog.Counted(notes, is))
	st, found, err := backlog.ReadState(forge.Bodies(notes), backlog.Keeper)
	if !found || err != nil {
		return counted, false, false
	}
	unheard = backlog.Heard(st, counted) < counted
	return counted, unheard, unheard && st.AnswerHeld == counted
}

// noTestUnsettled says whether the product owner left an issue in refine
// for a person after the tester's "no test possible" (#1053): its word
// on it says so, and that "no test possible" still stands on its text,
// no proof picked. The tester's reason, one line.
func noTestUnsettled(is forge.Issue, comments []string) (string, bool) {
	st, found, err := backlog.ReadState(comments, backlog.Keeper)
	if !found || err != nil || !strings.HasPrefix(st.Silent, backlog.NoTestSilent) || !backlog.NoTestStands(is, comments, backlog.Keeper) {
		return "", false
	}
	reason, _, _ := backlog.NoTest(comments)
	return strings.Join(strings.Fields(reason), " "), true
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
		return rec, "", fmt.Sprintf(unrefinedWhy+" on this text with %s, and left it lacking %s, nothing asked: read again, it would get the same; a person names the files it is about in a comment, or completes it",
			codeSaid(shown), lacks)
	}
	if rec.Retry >= RefineRetries {
		return rec, "", fmt.Sprintf(unrefinedWhy+" %d times on the same text and left it lacking %s, nothing asked: a person completes it, or names its files in a comment",
			rec.Retry+1, lacks)
	}
	rec.Retry++
	return rec, fmt.Sprintf("the product owner's last read left it lacking %s, nothing asked: woken again, %d of %d", lacks, rec.Retry, RefineRetries), ""
}
