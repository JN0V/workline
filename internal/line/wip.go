package line

// Room in the next state (ADR-0059, decision 9): a step that moves an
// issue into a state at its WIP limit does not start. It waits, said
// once on the issue in the line's comment, and the next event or
// reconcile starts it when room appears: the line pulls, it never pushes
// work onto a full state.
//
// A limit counts the work its state's role can pull (ADR-0059,
// amended 2026-11-02): an issue in the developer's state that waits for
// its acceptance tests, or for another open issue, or has parts, is
// counted apart. It waits on a step before its own state, never on room
// after it; counted, it would keep that step waiting for room it alone
// fills: a deadlock (#854). A ready issue without its tests goes to
// Tests by rule (retest), once; put back by a person, it is said and
// left to them.

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

const (
	developerRole = "developer"
	testerRole    = "tester"
)

// tally is what a run counts the limits from besides the open issues:
// the issues a tester's open Draft is for, and those a proof kind is
// recorded on, each read at first need and kept for the run.
type tally struct {
	drafts map[int]bool // nil until read
	proofs map[int]bool // by issue, once read
}

func newTally() *tally { return &tally{proofs: map[int]bool{}} }

// drafted says whether a tester's Draft is open for issue id: the
// forge's open merge requests from the tester's branches, read once.
func (t *tally) drafted(b forge.Backlog, id int) (bool, error) {
	if t.drafts == nil {
		f, ok := b.(interface {
			OpenMergeRequests(prefix string) ([]string, error)
		})
		if !ok {
			return false, fmt.Errorf("this forge lists no merge requests: whether a ready issue's tests are written cannot be read")
		}
		branches, err := f.OpenMergeRequests(tester.TestsBranch)
		if err != nil {
			return false, err
		}
		t.drafts = map[int]bool{}
		for _, br := range branches {
			var n int // read as the developer reads it: the number the branch starts with
			if _, err := fmt.Sscanf(strings.TrimPrefix(br, tester.TestsBranch), "%d", &n); err == nil && strings.HasPrefix(br, tester.TestsBranch) {
				t.drafts[n] = true
			}
		}
	}
	return t.drafts[id], nil
}

// proved says whether a proof kind is recorded on issue id in place of a
// Draft (ADR-0059, decision 4): from comments when given, else read.
func (t *tally) proved(b forge.Backlog, id int, comments []string) (bool, error) {
	if p, ok := t.proofs[id]; ok {
		return p, nil
	}
	if comments == nil {
		var err error
		if comments, err = b.Comments(forge.Target{Kind: "issue", ID: id}); err != nil {
			return false, err
		}
	}
	_, ok := tester.ReadProof(comments)
	t.proofs[id] = ok
	return ok, nil
}

// testsFirst says whether an issue in state s waits for its acceptance
// tests before the developer takes it, as the developer reads it: the
// developer's state, where the project routes the tester.
func testsFirst(cfg *routing.Config, s routing.State) bool {
	return s.Role == developerRole && cfg.Runs(testerRole)
}

// untested says whether issue is, in state s, still waits for its tests:
// its Validation names scenarios, and neither a tester's open Draft nor
// a proof kind stands for them. comments: the issue's, when read already.
func untested(cfg *routing.Config, b forge.Backlog, t *tally, s routing.State, is forge.Issue, comments []string) (bool, error) {
	if !testsFirst(cfg, s) || len(tester.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) == 0 {
		return false, nil
	}
	if ok, err := t.drafted(b, is.ID); err != nil || ok {
		return false, err
	}
	ok, err := t.proved(b, is.ID, comments)
	return !ok, err
}

// apart says why issue is, in state s, is not counted in its limit: what
// it waits on is no room after s, so its state's role cannot pull it;
// "" when it counts.
func apart(cfg *routing.Config, b forge.Backlog, t *tally, s routing.State, is forge.Issue, open map[int]bool) (string, error) {
	if s.Role != developerRole {
		return "", nil
	}
	if len(backlog.Waiting(is, open)) > 0 || len(backlog.Parts(is)) > 0 {
		return "it waits on another issue", nil
	}
	if u, err := untested(cfg, b, t, s, is, nil); err != nil || u {
		return "it waits for its tests", err
	}
	return "", nil
}

// waits says why the step of state s waits: its next state is at its
// limit; "" when it has room or no limit. open is the open issues, listed
// here at first need and kept for the next call; t, what else the count
// reads, kept for the run.
func waits(cfg *routing.Config, b forge.Backlog, s routing.State, open *[]forge.Issue, t *tally) (string, error) {
	limit := cfg.Limit(s.Next)
	if s.Next == "" || limit == 0 {
		return "", nil
	}
	if *open == nil {
		issues, err := b.Issues()
		if err != nil {
			return "", err
		}
		*open = issues
	}
	next, _ := cfg.State(s.Next)
	var in []forge.Issue
	isOpen := map[int]bool{}
	for _, is := range *open {
		isOpen[is.ID] = true
		if st, ok := cfg.StateOf(is.Labels); ok && st.Name == s.Next {
			in = append(in, is)
		}
	}
	n, left := len(in), 0
	if n >= limit { // the count refined only when it would stop the step: its reads cost API calls
		for _, is := range in {
			why, err := apart(cfg, b, t, next, is, isOpen)
			if err != nil {
				return "", err
			}
			if why != "" {
				left++
			}
		}
	}
	n -= left
	if n < limit {
		return "", nil
	}
	holds := fmt.Sprintf("%s holds %d open %s, its limit %d (routing.wip)", s.Next, n, plural(n, "issue", "issues"), limit)
	if left > 0 {
		holds = fmt.Sprintf("%s holds %d open %s the %s can take, its limit %d (routing.wip), and %d waiting for %s tests or another issue, not counted",
			s.Next, n, plural(n, "issue", "issues"), next.Role, limit, left, plural(left, "its", "their"))
	}
	return holds + fmt.Sprintf(": the %s starts when it has room", s.Role), nil
}

// waiting is what stateOf returns for a step that waits: the line's
// comment to write, its record unchanged but for the state it waits on;
// nil when the comment says it already, once is enough.
func waiting(res *Result, s routing.State, why string, rec Record, where string, id int) *StateWrite {
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-waits", Level: "info", Where: where,
		Message: fmt.Sprintf("its state, %s, waits to wake the %s: %s", s.Name, s.Role, why)})
	if rec.Waits == s.Next {
		return nil
	}
	rec.Waits = s.Next
	return &StateWrite{Issue: id, Record: rec, Waits: why}
}

// testsState is the state whose role, the tester, moves an issue into
// s: where a ready issue without its tests goes back; false when the
// table has none, or the project does not route the tester on issue.
func testsState(cfg *routing.Config, s routing.State) (routing.State, bool) {
	for _, t := range cfg.States {
		if t.Next == s.Name && t.Role == testerRole {
			_, role, _ := cfg.Wakes([]string{t.Label}, IssueEvent)
			return t, role == testerRole
		}
	}
	return routing.State{}, false
}

// retest checks issue is, in the developer's state s, for its tests: one
// still waiting for them goes to the tester's state, by rule, no agent —
// the step it skipped, not a judgement on its need. When a person put it
// back after the line's move, it is said once and left to them
// (ADR-0059, amended: a move undone is theirs to make again); said, its
// step is not run: the developer would only refuse it. held: false when
// its tests are there, the step then the state's as ever.
func retest(res *Result, cfg *routing.Config, b forge.Backlog, t *tally, s routing.State, is forge.Issue, rec Record, had bool, labeled string, comments []string) (w *StateWrite, held bool, err error) {
	to, ok := testsState(cfg, s)
	if !ok || !testsFirst(cfg, s) {
		return nil, false, nil
	}
	if u, err := untested(cfg, b, t, s, is, comments); err != nil || !u {
		return nil, false, err
	}
	where := fmt.Sprintf("#%d", is.ID)
	why := fmt.Sprintf("its %s names scenarios, and no tester's Draft nor proof kind stands for them: the %s does not take it before its tests",
		work.Name(work.Validation), s.Role)
	if had && rec.Retest == s.Name {
		if rec.RetestKept == labeled {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "state-untested-left", Level: "info", Where: where,
				Message: fmt.Sprintf("it waits for its tests, left to a person, who put %s back: the %s is not woken, nor is it counted in %s's limit", s.Label, s.Role, s.Name)})
			return nil, true, nil
		}
		why = fmt.Sprintf("a person put %s back after the line moved it to %s; %s", s.Label, to.Name, why)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-untested-kept", Level: "info", Where: where,
			Message: why + ": left as it is, not counted in " + s.Name + "'s limit"})
		rec.RetestKept = labeled
		return &StateWrite{Issue: is.ID, Record: rec, RetestKept: why}, true, nil
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-untested", Level: "warn", Where: where, Message: why + ": to " + to.Name})
	now := Record{State: s.Name, Body: bodyHash(is.Body), Labeled: labeled, Hops: rec.Hops, Retest: s.Name}
	return &StateWrite{Issue: is.ID, Record: now, Retest: why}, true, nil
}

// retestBody writes a ready issue's move to the tester's state — its
// labels, its card to the board's tests column — and is the line's
// comment saying why; or, for one a person put back, the comment alone.
func retestBody(f forge.Forge, cfg *routing.Config, repo string, t forge.Target, w StateWrite) (string, error) {
	left, _ := cfg.State(w.Record.Retest)
	to, _ := testsState(cfg, left)
	if w.RetestKept != "" {
		return backlog.Said("workline's line", "finds this issue without its tests, and leaves it", w.RetestKept,
			fmt.Sprintf("a person decides: moves it to %s by its label `%s`, or writes its tests; meanwhile it is not counted in %s's limit", to.Name, to.Label, left.Name)) +
			"\n\n" + recordMarker(w.Record), nil
	}
	add, remove, err := cfg.Move(to.Name)
	if err != nil {
		return "", err
	}
	card := board.Along(f, repo, to.Name)
	add, remove = card.Fold(add, remove)
	if err := f.Label(t, add, remove); err != nil {
		return "", err
	}
	card.Write(f, t.ID)
	return backlog.Said("workline's line", "moves this issue to "+to.Name+", for its tests", w.Retest,
		fmt.Sprintf("the %s writes them, then the issue is %s again; to undo, put `%s` back", to.Role, left.Name, left.Label)) +
		"\n\n" + recordMarker(w.Record), nil
}

// anticipate counts an issue whose step started in its next state, so
// the steps after it in the same run see it there.
func anticipate(cfg *routing.Config, is *forge.Issue, next string) {
	if s, ok := cfg.State(next); ok && next != "" {
		is.Labels = []string{s.Label}
	}
}
