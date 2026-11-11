// Package productowner holds the deterministic steps of the product owner
// role (roles/product-owner): pre lists the open issues with what the engine
// knows of each; the acts the agent proposes are checked when the engine
// applies them (internal/backlog). The role's scripts call them through
// `workline builtin product-owner pre|post`.
package productowner

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/codepick"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/testdef"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

const (
	exitNothing  = 10
	exitExternal = 3
	bodyMax      = 3000 // characters of a comment given to the agent; half, of a related issue's body
	commentsMax  = 5    // its last comments given
	relatedMax   = 6    // other issues on the same code, given whole
)

// Settings are the role's settings pre reads, as merged by the engine.
type Settings struct {
	IssuesPerRun int `json:"issues-per-run"` // issues read in one run
	CodeLinesMax int `json:"code-lines-max"` // lines of code given in one run, all files together
	CodeBytesMax int `json:"code-bytes-max"` // bytes of code given each issue read; 0: no bound (#1152)
	Acts         struct {
		Ask struct {
			Rounds int `json:"rounds"` // times an issue's reporter is written to, then a person
		} `json:"ask"`
		CloseObsolete struct {
			Mode   string   `json:"mode"`
			Days   *int     `json:"days"`   // an announcement waits this long (ADR-0024)
			Exempt []string `json:"exempt"` // labels that keep an issue from it
		} `json:"close-obsolete"`
	} `json:"acts"`
}

// obsolete is close-obsolete's setting, as the engine reads it.
func (s Settings) obsolete() backlog.Setting {
	o := s.Acts.CloseObsolete
	return backlog.Setting{Mode: o.Mode, Days: o.Days, Exempt: o.Exempt}
}

// rounds is how many times an issue's reporter is written to, as the
// engine counts them (backlog.RoundsMax).
func (s Settings) rounds() int {
	return backlog.RoundsMax(map[string]backlog.Setting{"ask": {Rounds: s.Acts.Ask.Rounds}})
}

// Pre lists the open issues in in/task.md. An issue without a state comment
// gets one, through a fallback comment, and is judged from the next run; one
// whose state comment does not read is left out, and said.
func Pre(runDir, repo string) int {
	role := os.Getenv("WORKLINE_ROLE")
	if role == "" {
		role = "product-owner"
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	b, ok := f.(forge.Backlog)
	if f == nil || !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no backlog to keep",
			Findings: []verdict.Finding{{Rule: "no-forge", Message: "the product owner keeps a forge's issues; " + forge.Missing}}})
	}
	open, err := b.Issues()
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	if os.Getenv("WORKLINE_EVENT") == "import" {
		return preImport(runDir, repo, role, b, open)
	}
	head, err := exec.Command("git", "-C", repo, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return fail(fmt.Errorf("the commit the run is on: %v", err))
	}
	commit := strings.TrimSpace(string(head))
	// One issue's state woke the role (ADR-0059): that issue's comments
	// and labels alone are read, here and when its acts are decided —
	// each open issue's would spend the token's API budget at every step
	// of the line (#804). The listing of the open issues, one call a
	// hundred, stays: the order, what blocks, what is open.
	var scope []int
	if n := issueEvent(); n > 0 {
		scope = []int{n}
		if err := os.WriteFile(filepath.Join(runDir, "in", backlog.ScopeFile), []byte(strconv.Itoa(n)), 0o644); err != nil {
			return fail(err)
		}
	}
	inScope := func(is forge.Issue) bool { return scope == nil || slices.Contains(scope, is.ID) }
	// What the run reads, kept with each act it does, for the weekly
	// sample to re-check it on that text and that code (ADR-0046).
	thisRun := backlog.Read{At: commit, Bodies: map[int]string{}}
	for _, is := range open {
		thisRun.Bodies[is.ID] = backlog.BodyDigest(is.Body)
	}
	if data, err := yaml.Marshal(thisRun); err != nil {
		return fail(err)
	} else if err := os.WriteFile(filepath.Join(runDir, "in", backlog.ReadFile), data, 0o644); err != nil {
		return fail(err)
	}
	s := Settings{IssuesPerRun: 8, CodeLinesMax: 1500, CodeBytesMax: 16000}
	raw := map[string]any{}
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json")); err == nil {
		if err := json.Unmarshal(data, &s); err != nil {
			return fail(fmt.Errorf("settings: %v", err))
		}
		_ = json.Unmarshal(data, &raw)
	}
	// How far the role goes (ADR-0026): a level or a number out of range
	// stops the run, never read as a default.
	cfg, err := backlog.ReadConfig(raw)
	if err != nil {
		return fail(fmt.Errorf("settings: %v", err))
	}
	// The reviewer after the role in a line reads each spec before it is
	// ready (#128): its findings given to the agent, ready held on them.
	line, err := routing.Load(repo)
	if err != nil {
		return fail(err)
	}
	cfg.SpecReview = line.Follows(role, backlog.SpecReviewer)
	cfg.Tester = line.Runs("tester")
	var fallback []intent.Intention
	var findings []verdict.Finding
	for _, name := range cfg.Gone {
		findings = append(findings, verdict.Finding{Rule: "setting-gone", Level: "warn",
			Message: name + " is no longer read: " + backlog.GoneSettings[name] + "; take it off the project's settings"})
	}
	// Where the project keeps each axis, when its settings ask (ADR-0040):
	// read there for the whole run, and said once. A family the setting
	// names and the project lacks stops the run, said.
	scheme, where, err := backlog.Where(b, cfg, open)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "where the project keeps its priorities is not found",
			Findings: []verdict.Finding{{Rule: "values-where", Level: "block", Message: err.Error()}}})
	}
	scale.Use(scheme)
	findings = append(findings, where...)
	judged := 0
	tracked := trackedFiles(repo)
	var readIDs []string // the issues read, for the plan (backlog.Decide)
	// answered: a person's comment on an issue, read first — waiting on
	// them, set aside or neither; again: an act proposed only for the cap;
	// then the issues never read, a person's newest first, an import's or a
	// bot's oldest first; behind: read and left with nothing under an older
	// reading, read again once.
	var answered, again, never, catchUp, changed, behind, rest []due
	heard := map[int]forge.Note{} // a person's comment on an issue with no proposal waiting: the last
	// What the role holds, read from each open issue's state (ADR-0038):
	// its proposals, its acts a person may undo, the old report's record
	// the first time.
	ledger, err := backlog.ReadLedger(b, role, open, scope)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	answers := map[int]*backlog.Answer{} // a person's comments read as "revise", for the plan
	revisions := backlog.RevisionsMax(cfg.Acts)
	milestones, err := b.Milestones()
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	next := backlog.NextMilestone(repo, milestones) // where an issue that slipped goes
	var capped []int                                // an act proposed only for the cap, or recorded without its act: read again
	for _, q := range ledger.Record.Proposed {
		if (q.Capped || q.Undrafted()) && q.Issue > 0 {
			capped = append(capped, q.Issue)
		}
	}
	now := clock.Now()
	var all []forge.Issue // every issue, open and closed, read once when a parent needs its parts
	// A link the role set whose blocker closed (ADR-0028): its reason is
	// gone, and the engine takes it off, with no agent; a person's link is
	// left. A blocker closed without delivering is said: what waited on it
	// may need a person's look.
	stale := ledger.Stale(open)
	if len(stale) > 0 {
		if all, err = b.AllIssues(); errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		} else if err != nil {
			return fail(err)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(stale)) {
		var says []string
		for _, bl := range stale[id] {
			how := "closed"
			i := slices.IndexFunc(all, func(is forge.Issue) bool { return is.ID == bl })
			switch {
			case i < 0:
				how = "gone from the forge"
			case all[i].Reason == "not_planned":
				how = "closed as not planned"
			case all[i].Reason == "duplicate":
				how = "closed as a duplicate"
			}
			says = append(says, fmt.Sprintf("#%d is %s", bl, how))
			if how != "closed" {
				findings = append(findings, verdict.Finding{Rule: "blocker-not-delivered", Level: "warn", Where: fmt.Sprintf("#%d", id),
					Message: fmt.Sprintf("it waited on #%d, %s: the link the role set is taken off; whether #%d still stands without it is a person's to say", bl, how, id)})
			}
		}
		fallback = append(fallback, intent.Intention{Kind: "undepend", Value: map[string]any{
			"issue": id, "blocked-by": stale[id], "own": true,
			"why": strings.Join(says, ", ") + ": the wait the role set is over."}})
	}
	judgedPass := os.Getenv("WORKLINE_JUDGED") == "answered" // the engine asked pre's questions to a judge
	asked := false                                           // a question for the judge written: the judge first
	notesOf := map[int][]forge.Note{}                        // each issue's comments, for what waits on a person
	timedOut := map[int]backlog.Silence{}                    // a question unanswered past acts.ask.wait, for the role's assumption (ADR-0060)
	draftsTaken := map[int]bool{}                            // drafts unanswered past it, taken as written with no agent
	var waits []backlog.Wait                                 // the issues waiting on a person, for the summary (ADR-0031)
	var kindless []forge.Issue                               // read before, no kind label: their kind read alone (#878)
	for _, is := range open {
		if backlog.Tracking(is, role) || !inScope(is) {
			// Its old report, the weekly sample, a role's issue kept in
			// place: workline's own, never read as the backlog's (#429).
			// Out of an event's scope: its comments unread (#804).
			continue
		}
		notes := ledger.Notes[is.ID]
		comments := forge.Bodies(notes)
		notesOf[is.ID] = notes
		st, found, err := backlog.ReadState(comments, role)
		_, _, noTest := backlog.NoTest(comments)
		if _, picked := backlog.ReadProof(comments, role); found && err == nil && noTest && !picked {
			// A pick already done for an earlier "no test possible" is
			// not taken again for the tester's new word.
			if pp, ok := backlog.PersonsPick(notes); ok && !st.Proof.Same(pp) {
				// The tester found no test possible, and a person of the
				// project named the kind in a comment: their pick, done
				// with no agent (ADR-0059, decision 4).
				fallback = append(fallback, intent.Intention{Kind: "ready", Value: map[string]any{
					"issue": is.ID, "proof": pp, "own": true,
					"why": fmt.Sprintf("@%s named its proof in a comment, the tester having found no test possible.", pp.By)}})
			}
		}
		// Only the text as last proposed, not answered since — an answer
		// may change it, the agent reads it first — and only while the body
		// still lacks what it adds.
		p := backlog.LastProposal(comments, role)
		agreed := ""
		unheard := false // an answer to it the agent reads first (#390)
		if p != nil {
			_, adds, _ := backlog.Refine(is.Body, *p, role)
			if len(adds) > 0 {
				agreed = backlog.Agreement(notes, is, role)
			}
			// An answer may change it, unless it is the agreement itself:
			// the reporter's or the project's, not read yet — a bot's or a
			// stranger's leaves it as it stands.
			unheard = len(adds) > 0 && agreed == "" && backlog.AnswerUnheard(st, notes, is, role)
			if len(adds) == 0 || unheard {
				p = nil
			}
		}
		if found && err == nil && agreed != "" && !backlog.Accepted(is) {
			// Its reporter, or a person of the project, replied `agreed` to
			// the text last proposed: the engine writes what the body still
			// lacks, with no agent; ready, a split or a rename stay the
			// project's, by the label (ADR-0021).
			whose := "its reporter"
			if agreed != is.Author {
				whose = "a person of the project"
			}
			lp := backlog.LastProposal(comments, role)
			fallback = append(fallback, intent.Intention{Kind: "refine", Value: map[string]any{
				"issue": is.ID, "need": lp.Need, "example": lp.Example, "steps": lp.Steps, "verification": lp.Verification, "validation": lp.Validation,
				"scope": lp.Scope, "sources": lp.Sources, "own": true, "agreed": agreed,
				"why": fmt.Sprintf("the text proposed to its reporter, agreed to by @%s (%s) in a reply.", agreed, whose)}})
		}
		who, closing := "", false // who accepted it; a closing it proposed, accepted
		if found && err == nil && backlog.Accepted(is) {
			// A person of the project labelled it workline:accepted: their
			// yes to what the role proposes on it, done as its state holds
			// it, with no agent (ADR-0038).
			var werr error
			if who, werr = acceptedBy(b, is); errors.Is(werr, forge.ErrUnreachable) {
				fmt.Fprintln(os.Stderr, werr)
				return exitExternal
			}
			for _, q := range st.Proposed {
				if q.Doable() && !ledger.Aside[is.ID] {
					c := q.Proposal
					closing = closing || c.Do == "close"
					fallback = append(fallback, intent.Intention{Kind: c.Do, Value: map[string]any{
						"issue": is.ID, "reason": c.Reason, "ticked": who, "own": true,
						"why": "accepted by " + who + " (" + backlog.AcceptedLabel(b) + ")"}})
				}
			}
		}
		if found && err == nil && !backlog.Accepted(is) && !ledger.Aside[is.ID] {
			// A move back to refine an earlier run proposed: no longer a
			// person's yes to wait for (ADR-0059, amended), done with no
			// agent, its reason as proposed.
			for _, q := range st.Proposed {
				if c := q.Proposal; q.Doable() && c.Do == "unready" {
					fallback = append(fallback, intent.Intention{Kind: "unready", Value: map[string]any{
						"issue": is.ID, "own": true, "why": c.Why}})
				}
			}
		}
		if found && err == nil && backlog.Accepted(is) && p != nil {
			// A refined text proposed to an outsider, agreed to by a person
			// of the project with the label: the engine writes what the body
			// still lacks, with no agent (ADR-0021).
			fallback = append(fallback, intent.Intention{Kind: "refine", Value: map[string]any{
				"issue": is.ID, "need": p.Need, "example": p.Example, "steps": p.Steps, "verification": p.Verification, "validation": p.Validation,
				"scope": p.Scope, "sources": p.Sources, "own": true,
				"why": "the text proposed to its reporter, agreed to by a person (" + backlog.AcceptedLabel(b) + ")"}})
		}
		// The tester found no test possible for it as it stands, and no kind
		// is picked: a ready alone sends it back to its tests unchanged, so
		// neither the label nor the spec review moves it (#815).
		noTestStands := backlog.NoTestStands(is, comments, role)
		if found && err == nil && backlog.Accepted(is) && !closing {
			// A person accepted its drafts, with the label: the engine moves
			// it to ready if its sections are there, with no agent — once
			// the agent read an answer to the text proposed (#390).
			// Never a split parent (#1208): its parts are what is built.
			if !unheard && !noTestStands && len(backlog.SplitInto(is, st)) == 0 {
				fallback = append(fallback, intent.Intention{Kind: "ready", Value: map[string]any{
					"issue": is.ID, "why": "its drafts accepted (" + backlog.AcceptedLabel(b) + ")", "ticked": who, "own": true}})
			}
		} else if r, cleared := backlog.SpecCleared(is, comments); found && err == nil && cfg.SpecReview && cleared &&
			!backlog.Readied(is) && !noTestStands && len(backlog.NotReady(is, st, false)) == 0 && len(backlog.SplitInto(is, st)) == 0 &&
			len(backlog.Unheard(st, notes, is)) == 0 {
			// Never a split parent, read for its parts (#1208): its parts
			// are what is built, a review before its split no ready. Nor
			// one holding a person's comment not read yet (#1331): the
			// agent writes what it settles first.
			// The reviewer read its spec as it is and found no important
			// finding open: the hold lifts, and the engine moves it to
			// ready, with no agent (#128).
			fallback = append(fallback, intent.Intention{Kind: "ready", Value: map[string]any{
				"issue": is.ID, "why": fmt.Sprintf("the reviewer read its spec (round %d): no important finding open", r.Round), "own": true}})
		}
		switch {
		case !found:
			fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{
				"issue": is.ID, "sticky": "state", "body": backlog.FormatState(backlog.State{Confirmed: commit})}})
			if backlog.ByPerson(is) && (len(ledger.States) > 0 || is.ID == issueEvent()) {
				// A person's issue opened since the role's last run: read in
				// this one, its first state written before any act on it
				// (ADR-0018, amended). On a backlog the role never ran on,
				// every issue waits its turn, but the one an event names.
				never = append(never, due{is, &backlog.State{Confirmed: commit}, comments, notes})
			} else if is.ID == issueEvent() {
				// An import or a bot's issue its own state woke the role on
				// (ADR-0059): read now, as the line wakes the role once a
				// body — left for the catch-up, it sat in refine with
				// nothing said (#1036).
				catchUp = append(catchUp, due{is, &backlog.State{Confirmed: commit}, comments, notes})
			}
			continue
		case err != nil:
			// Held, said each run (#302): never read as no state, which
			// would write a new one over what the role held there.
			findings = append(findings, verdict.Finding{Rule: "state-broken", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
				Message: "its state comment does not read (" + err.Error() + "): the issue is held — not judged, nothing written on it — until a person mends that comment, or deletes it for a state anew"})
			continue
		}
		if perr := backlog.ProposalUnread(comments, role); perr != nil && len(backlog.NotReady(is, nil, true)) > 0 {
			// The text last proposed to its reporter does not read (#302):
			// an agreement to it writes nothing; said while the body still
			// lacks a section, a person's to complete.
			findings = append(findings, verdict.Finding{Rule: "proposal-unread", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
				Message: "the engine's block in the text last proposed to its reporter does not read (" + perr.Error() + "): an agreement to it writes nothing in the body, which waits on a person to copy the sections in"})
		}
		// A parent (ADR-0029): what its parts delivered, kept in one
		// comment, with no agent; a person accepts it once all are closed.
		if parts := backlog.Parts(is); len(parts) > 0 {
			ev, err := evidence(b, repo, is, parts, role, &all)
			if errors.Is(err, forge.ErrUnreachable) {
				fmt.Fprintln(os.Stderr, err)
				return exitExternal
			}
			if err != nil {
				return fail(err)
			}
			fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{
				"issue": is.ID, "sticky": backlog.EvidenceKey, "body": ev.Body}})
			for _, p := range ev.Unread {
				findings = append(findings, verdict.Finding{Rule: "closers-unread", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
					Message: fmt.Sprintf("the forge did not say what closed #%d (%v): said on the parent as not read, never taken for closed by hand", p.ID, p.Unread)})
			}
			if len(ev.TestUnread) > 0 {
				findings = append(findings, verdict.Finding{Rule: "proof-test-unread", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
					Message: fmt.Sprintf("its %s names %s as proof, and the code could not be read for it: the item is said not proved, never the test missing", work.Name(work.Verification), strings.Join(ev.TestUnread, ", "))})
			}
			if len(ev.NoTest) > 0 {
				findings = append(findings, verdict.Finding{Rule: "proof-test-missing", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
					Message: fmt.Sprintf("its %s rests on %s as proof, and the code at this commit holds no such test: the item is said not proved", work.Name(work.Verification), strings.Join(ev.NoTest, ", "))})
			}
			if ev.AllClosed {
				alone := "the role never closes it alone"
				if kind, _ := backlog.KindOf(is.Labels); kind == backlog.KindBug {
					alone = "the role closes it alone only on its reproduction test green on the default branch"
				}
				msg := fmt.Sprintf("its %d parts are closed: for a person to accept, by closing it or with %s — %s", ev.Parts, backlog.AcceptedLabel(b), alone)
				if len(ev.Undone) > 0 || len(ev.Unproved) > 0 {
					msg += fmt.Sprintf("; not delivered: %d part(s), not proved: %d %s item(s)", len(ev.Undone), len(ev.Unproved), work.Name(work.Verification))
				}
				findings = append(findings, verdict.Finding{Rule: "parent-to-accept", Level: "info", Where: fmt.Sprintf("#%d", is.ID), Message: msg})
				// Proposed to a person, its yes one label (ADR-0029,
				// amended): never left open with nothing asked (#1208).
				fallback = append(fallback, intent.Intention{Kind: "close", Value: map[string]any{
					"issue": is.ID, "reason": "completed", "parent": true, "own": true,
					"why": fmt.Sprintf("its %d parts are closed; what each delivered, and what is missing, is in the comment on its parts", ev.Parts)}})
			}
			if backlog.Accepted(is) && !closing {
				// A yes left on it with no closing proposed (#1213): put
				// early, or one to an act done already, never a yes to
				// what its parts delivered: checked as one to its closing,
				// it is taken off, said, never left to stand for it.
				fallback = append(fallback, intent.Intention{Kind: "close", Value: map[string]any{
					"issue": is.ID, "reason": "completed", "parent": true, "own": true, "ticked": who,
					"why": "accepted by " + who + " (" + backlog.AcceptedLabel(b) + ")"}})
			}
		}
		// An issue announced obsolete (ADR-0024): kept open when someone
		// wrote or took the label off, closed once due if a second judge
		// agrees — with no agent of the role's.
		o := backlog.ReadObsolete(repo, is, notes, st, role, s.obsolete(), now)
		if w := backlog.ObsoleteWait(is.ID, o); w != nil && s.obsolete().Mode != backlog.Off {
			waits = append(waits, *w) // due, a judge's: said stuck until closed or kept
		}
		switch {
		case o.Announcement == nil:
		case o.Keep != "":
			fallback = append(fallback, intent.Intention{Kind: "keep", Value: map[string]any{
				"issue": is.ID, "why": o.Keep, "say": o.Say, "own": true}})
		case !o.Due || s.obsolete().Mode == backlog.Off:
		case judgedPass:
			j, ok := readJudged(runDir, is.ID)
			switch {
			case !ok || j.Yes == nil:
				why := "no agent to judge with"
				if ok && j.Error != "" {
					why = j.Error
				}
				findings = append(findings, verdict.Finding{Rule: "judge-unavailable", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
					Message: "announced obsolete, due, and nobody wrote: not closed without a second judge (" + why + "); it waits for the next run"})
			case *j.Yes:
				a := o.Announcement
				fallback = append(fallback, intent.Intention{Kind: "close", Value: map[string]any{
					"issue": is.ID, "reason": "obsolete", "why": a.Why, "announced": a.Announced, "own": true,
					"quote": map[string]any{"path": a.Quote.Path, "issue": a.Quote.Issue, "text": a.Quote.Text}}})
			default:
				fallback = append(fallback, intent.Intention{Kind: "keep", Value: map[string]any{
					"issue": is.ID, "why": "a second judge did not agree — " + j.Says(), "say": true, "own": true}})
			}
		case os.Getenv("WORKLINE_AI") == "none":
			findings = append(findings, verdict.Finding{Rule: "judge-unavailable", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
				Message: "announced obsolete, due, and nobody wrote: not closed without a second judge (no agent); it waits for a run with one"})
		default:
			if err := askJudge(runDir, repo, is, *o.Announcement); err != nil {
				return fail(err)
			}
			asked = true
		}
		if backlog.Released(repo, is.Milestone) {
			// Its milestone's release is tagged: it slipped, and the engine
			// moves it to the next, with no agent (or proposes, with none).
			fallback = append(fallback, intent.Intention{Kind: "milestone", Value: map[string]any{
				"issue": is.ID, "milestone": next, "from": is.Milestone, "own": true,
				"why": fmt.Sprintf("slipped: %s is released (its tag exists).", is.Milestone)}})
		}
		// Who counts (ADR-0038): its reporter's and the project's people's
		// comments, never a bot's, a stranger's or a "+1"; those the role
		// has not read yet.
		counted := backlog.Counted(notes, is)
		fresh := backlog.Unheard(st, notes, is)
		aside := ledger.Aside[is.ID] || backlog.StillAside(st, is, notes)
		var back []forge.Note // the comments that brought it back, read under an older way of reading
		if aside && !ledger.Aside[is.ID] {
			back, err = backlog.BroughtBack(b, is, st, notes)
			if errors.Is(err, forge.ErrUnreachable) {
				fmt.Fprintln(os.Stderr, err)
				return exitExternal
			}
			if err != nil {
				back = nil // a forge that keeps no label events: nothing known brought it back
			}
		}
		d := due{is, st, comments, notes}
		if scope == nil && os.Getenv("WORKLINE_EVENT") != "issue" && !aside && is.ID != ledger.Report && unkinded(is, st) {
			kindless = append(kindless, is) // for the kind's light read (#878)
		}
		// Questions time out, as lazy consensus (ADR-0060): one unanswered
		// past the level's delay, dated by the forge, read once for the
		// role's assumption; drafts no person accepted past it, all the
		// issue lacks, taken as written with no agent; an answer after an
		// assumption marks it, and is read as asking to revise.
		silence, silent := backlog.Unanswered(notes, role)
		silent = silent && silence.Due(cfg.Wait, st, now) && (st.Told < silence.Round || is.ID == issueEvent()) && !aside
		if silent {
			timedOut[is.ID] = silence
		}
		answeredBy := ""
		if len(fresh) > 0 && !aside {
			answeredBy = backlog.AnsweredAfter(is, st, notes)
		}
		if answeredBy != "" {
			fallback = append(fallback, intent.Intention{Kind: "assume", Value: map[string]any{
				"issue": is.ID, "answered": answeredBy, "own": true,
				"why": fmt.Sprintf("@%s wrote after the role's assumptions: their word stands over them.", answeredBy)}})
		}
		if !aside && !backlog.Readied(is) && backlog.OnlyDrafts(is, st) {
			taken, err := draftsDue(b, is, notes, cfg.Wait, now)
			if errors.Is(err, forge.ErrUnreachable) {
				fmt.Fprintln(os.Stderr, err)
				return exitExternal
			}
			if taken {
				draftsTaken[is.ID] = true
				fallback = append(fallback, intent.Intention{Kind: "ready", Value: map[string]any{
					"issue": is.ID, "own": true,
					"why": fmt.Sprintf("its drafts got no answer for %s (acts.ask.wait): taken as written, for a person to correct", cfg.Wait.Say())}})
			}
		}
		switch {
		case agreed != "" || backlog.Accepted(is) && !unheard:
			// Agreed to, or accepted: done this run, not read again for it.
			// Accepted after an answer to the text proposed, it is read
			// below for that answer: its refine is written in the issue,
			// and the label moves it to ready at the next run (#390).
			rest = append(rest, d)
		case len(back) > 0:
			// A person's comment brought it back, read under an older way
			// of reading, and it stayed aside: read again once, as a fix
			// reaches the issues it failed on (ADR-0038, amended).
			heard[is.ID] = back[len(back)-1]
			behind = append(behind, d)
		case aside:
			// Set aside by a person: nothing proposed until it changes.
			rest = append(rest, d)
		case st.Label != "" && len(fresh) > 0 && st.Revisions >= revisions:
			// Its drafts revised as many times as allowed: a person's from
			// here; said in its own comment, edited, no new one on the issue
			// (ADR-0038; ADR-0059, decision 1).
			findings = append(findings, verdict.Finding{Rule: "left-to-a-person", Level: "info", Where: fmt.Sprintf("#%d", is.ID),
				Message: fmt.Sprintf("@%s wrote on it; its proposal was revised %d times already: left to a person, said in its comment there", fresh[len(fresh)-1].Author, st.Revisions)})
			left := *st
			n := len(counted)
			left.Heard, left.Left = &n, fresh[len(fresh)-1].Author
			fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{
				"issue": is.ID, "sticky": "state", "update-only": true, "body": backlog.FormatState(left)}})
			rest = append(rest, d)
		case st.Label != "" && len(fresh) > 0, answeredBy != "":
			// A person's comment on its proposal: "revise" — read first,
			// the drafts rewritten in place, 👀 on each, one line in reply.
			// An answer after its assumptions is one too (ADR-0060).
			var ids []string
			for _, n := range fresh {
				ids = append(ids, n.ID)
			}
			answers[is.ID] = &backlog.Answer{Notes: ids, Who: fresh[len(fresh)-1].Author, Revision: st.Revisions + 1}
			answered = append(answered, d)
		case len(fresh) > 0:
			// A person's comment on an issue with no proposal waiting on
			// them — set aside, read and left, never read: their word, read
			// first with the answers; no 👀 nor reply, nothing was asked of
			// them (ADR-0038, amended).
			heard[is.ID] = fresh[len(fresh)-1]
			answered = append(answered, d)
		case silent:
			// Its question unanswered past the delay: read once, for the
			// role's assumption (ADR-0060).
			answered = append(answered, d)
		case slices.Contains(capped, is.ID):
			again = append(again, d)
		case st.Judged == "" && backlog.ByPerson(is):
			never = append(never, d)
		case st.Judged == "":
			catchUp = append(catchUp, d)
		case cfg.SpecReview && backlog.SpecOpen(is, comments) != nil && backlog.SpecOpen(is, comments).Body != st.Answered:
			// The reviewer's findings on its spec, not given yet: answered
			// at this refine (#128).
			changed = append(changed, d)
		case cfg.SpecReview && len(backlog.Unrevised(is, st, comments)) > 0:
			// A review answered by asking a person for a section the role
			// rewrites now, its Verification or Scope: read again once,
			// the section rewritten (ADR-0060).
			changed = append(changed, d)
		case rewriteUnread(is, st, cfg):
			// Its own Validation names no scenario the tester reads: the
			// role rewrites it, told once a body (#855).
			changed = append(changed, d)
		case noTestUnread(comments, st, role):
			// The tester found no test possible: the role picks the
			// proof, or sends it back to its tests (ADR-0059, decision 4).
			changed = append(changed, d)
		case sourcesChanged(repo, st),
			st.Body != "" && backlog.BodyDigest(is.Body) != st.Body, // a person changed its body
			st.Closed != "": // a person reopened what the role closed
			changed = append(changed, d)
		case backlog.LeftBehind(is, st, comments, role):
			// Read and left with nothing under an older way of reading:
			// read again once, so a fix of the engine reaches it.
			behind = append(behind, d)
		default:
			rest = append(rest, d)
		}
	}
	if asked {
		return 0 // the judge first: pre runs again with its answers
	}
	// What the order holds back (ADR-0028), with no agent: the first ready
	// issue offered, those waiting on an open one, the cycles.
	report, isOpen := ledger.Report, map[int]bool{}
	for _, is := range open {
		isOpen[is.ID] = true
	}
	bl := backlog.ReadBacklog(open, report)
	for _, c := range bl.Cycles {
		findings = append(findings, verdict.Finding{Rule: "dependency-cycle", Level: "warn", Where: fmt.Sprintf("#%d", c[0]),
			Message: backlog.CycleText(c) + ": a cycle, reported and never followed; none of them is offered first until a person takes a link off"})
	}
	for _, id := range bl.Blocked {
		findings = append(findings, verdict.Finding{Rule: "waiting", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: fmt.Sprintf("waits on %s, open: ordered after it, never offered first", issueRefs(bl.Waiting[id]))})
	}
	first := map[int]bool{}
	for _, is := range backlog.Next(open, report, cfg.NextMax) {
		first[is.ID] = true
		findings = append(findings, verdict.Finding{Rule: "next-ready", Level: "info", Where: fmt.Sprintf("#%d", is.ID),
			Message: is.Title + placed(is)})
	}
	if scale.Used.Detected {
		// The ready issues missing a value, counted; the role's values
		// stuck on one level, said (ADR-0040).
		findings = append(findings, unestimated(open, report, isOpen)...)
		findings = append(findings, stuck(ledger.States, isOpen)...)
		if scale.Used.Priority.Hidden() || scale.Used.Effort.Hidden() {
			findings = append(findings, verdict.Finding{Rule: "values-hidden", Level: "info",
				Message: "Values hidden: the fields are visible to organization members only, and this repository is public."})
		}
	}
	// What waits on a person (ADR-0031), with no agent: a reporter's
	// answer, from the day of the last round; a ready issue nothing
	// started, from the day the forge says it got the label — those
	// next aside. A day the forge does not say is said.
	unknown := func(id int, what string) {
		findings = append(findings, verdict.Finding{Rule: "stuck-unknown", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: "the forge does not say " + what + ": not said stuck, whatever it waits"})
	}
	for _, is := range open {
		if is.ID == report || !inScope(is) {
			continue // out of an event's scope: when it got ready, unread (#804)
		}
		if notes, ok := notesOf[is.ID]; ok {
			switch w, known := backlog.AskedWait(is.ID, notes, role); {
			case !known:
				unknown(is.ID, "when its reporter was last written to")
			case w != nil:
				waits = append(waits, *w)
			}
		}
		if first[is.ID] || !backlog.Offered(is, isOpen) {
			continue
		}
		t, err := b.Trail(is.ID, backlog.LabelReady)
		if errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		}
		if err != nil {
			findings = append(findings, verdict.Finding{Rule: "stuck-unknown", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
				Message: "when it got " + backlog.LabelReady + " and what names it could not be read (" + err.Error() + "): not said stuck"})
			continue
		}
		switch w, known := backlog.ReadyWait(is.ID, t); {
		case !known:
			unknown(is.ID, "when it got "+backlog.LabelReady)
		case w != nil:
			waits = append(waits, *w)
		}
	}
	// What is stuck, for the job's summary (ADR-0031, ADR-0038): each
	// issue waiting on a person past stuck-days, an announcement due.
	board := backlog.MakeBoard(open, report, waits, ledger.Record.Proposed, cfg, now)
	for _, w := range board.Stuck {
		findings = append(findings, verdict.Finding{Rule: "stuck", Level: "info", Where: fmt.Sprintf("#%d", w.Issue), Message: board.StuckLine(w)})
	}
	// What open issues were built on that changed (ADR-0032), with no
	// agent, when the project turns it on (`changed-needs`): a person's
	// rewrite of an issue's Need or Scope, the lines of a file an issue
	// was imported from. The parts of the issue changed, and the issue
	// imported, are read first, with the change.
	var found []backlog.Change
	var rebase []due
	front := map[int]string{}
	if cfg.ChangedNeeds {
		var unread []verdict.Finding
		found, front, rebase, unread = changesFound(repo, open, report, slices.Concat(answered, again, never, catchUp, changed, behind, rest), cfg.Archived)
		findings = append(findings, unread...)
	}
	var ahead []due
	pick := func(list []due) []due {
		var left []due
		for _, d := range list {
			if front[d.is.ID] != "" {
				ahead = append(ahead, d)
			} else {
				left = append(left, d)
			}
		}
		return left
	}
	again, never, catchUp, changed, behind, rest = pick(again), pick(never), pick(catchUp), pick(changed), pick(behind), pick(rest)
	slices.SortFunc(ahead, func(a, b due) int { return a.is.ID - b.is.ID })
	// A person's issue never read, newest first: what a person just asked
	// for is not left behind the catch-up of an import (ADR-0018,
	// amended); an import's or a bot's, oldest first. One left with
	// nothing, newest first too.
	slices.SortFunc(never, func(a, b due) int { return b.is.ID - a.is.ID })
	slices.SortFunc(behind, func(a, b due) int { return b.is.ID - a.is.ID })
	// The order of a run's reading: a person's comment — an answer to a
	// proposal, or their word on any other issue —, what a change asks
	// for, an act proposed only for the cap, a person's new issue, what
	// changed since it was read, one left with nothing under an older way
	// of reading, then the catch-up. An issue whose code did not change is
	// not read again (ADR-0018). Without an agent, nothing is read, and no
	// issue is said to be. Past proposals-max issues waiting on a person,
	// only those a person commented on are read: the role slows down
	// rather than piling up proposals (ADR-0038).
	toRead := slices.Concat(answered, ahead, again, never, changed, behind, catchUp)
	if n := issueEvent(); n != 0 {
		// One issue's state woke the role (ADR-0059): that issue alone is
		// read, whatever changed; the rest wait for their own event, or
		// the night.
		var one []due
		rest0 := slices.Clone(rest) // nothing but the woken state to read it for
		for _, d := range slices.Concat(toRead, rest) {
			if d.is.ID == n {
				one = append(one, d)
			}
		}
		rest = slices.DeleteFunc(slices.Concat(toRead, rest), func(d due) bool { return d.is.ID == n })
		if draftsTaken[n] && slices.ContainsFunc(rest0, func(d due) bool { return d.is.ID == n }) {
			// Woken for its drafts past the delay alone: taken as written,
			// with no agent (ADR-0060).
			rest = append(rest, one...)
			one = nil
		}
		toRead = one
	} else if waiting := ledger.Waiting(open); waiting >= cfg.ProposalsMax {
		findings = append(findings, verdict.Finding{Rule: "proposals-waiting", Level: "warn",
			Message: fmt.Sprintf("%d issues wait on your answer, the most allowed (proposals-max: %d): the role reads only those you answered until fewer wait", waiting, cfg.ProposalsMax)})
		for _, d := range toRead[len(answered):] {
			rest = append(rest, d)
		}
		toRead = answered
	}
	if os.Getenv("WORKLINE_AI") == "none" {
		toRead = nil
		kindless = nil
	}
	issuePages := forge.IssuePages(f) // an issue cited by its page is cited
	said := ""                        // the releases and open milestones, read once
	// The tester's open Drafts, where the project routes it: a ready
	// issue from before the line, with none, is offered to its tests
	// (#788); nil, none offered.
	var drafts []string
	if line.Runs("tester") && slices.ContainsFunc(toRead, func(d due) bool { return slices.Contains(d.is.Labels, backlog.LabelReady) }) {
		open, err := f.OpenMergeRequests(tester.TestsBranch)
		if errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		}
		if err != nil {
			return fail(err)
		}
		drafts = append([]string{}, open...)
	}
	// build writes the task of the first n issues to read: each issue with
	// its state once read, recorded only once the agent answers; the
	// others in the backlog's order, given whole when on the same code, by
	// title otherwise; the code they name; nothing is written yet. Each
	// issue no kind tells is asked its kind by name (#895).
	kindAct, kindMax := kindMode(cfg, ledger.Demoted(cfg.UndoneMax))
	build := func(n int) (taskBuilt, error) {
		t := taskBuilt{wasRead: map[int]bool{}}
		var task strings.Builder
		// The code each issue read names, by the picker the tester and
		// the reviewer share (ADR-0063 §8): code-lines-max shared among
		// them, a piece given for one not given again for the next.
		picker, err := codepick.New(repo, "")
		if err != nil {
			return t, err
		}
		tests := testsOf(repo)
		var code []string          // the files the issues name
		var picks []codepick.Pick  // each issue's, in order
		var pickedFor []int        // the issue each pick is for
		codeLeft := s.CodeLinesMax  // lines of code still to share
		noCode := false            // an issue read names no file: the repository's folders are listed
		left := slices.Clone(rest)
		for i, d := range toRead {
			if i >= n {
				left = append(left, d)
				continue
			}
			t.wasRead[d.is.ID] = true
			t.ids = append(t.ids, strconv.Itoa(d.is.ID))
			written, aboutIt := writtenAt(repo, d.is, d.st, tracked, cfg.Archived)
			share := codeLeft / max(min(n, len(toRead))-i, 1)
			a := askOf(picker, tests, d.is, d.st, d.comments, aboutIt, share)
			a.Bytes = s.CodeBytesMax // each issue's: the context budget bounds the run
			if d.is.ID == issueEvent() {
				// Run again for the code its agent said it lacked
				// (ADR-0063, §6): those files, whole, or their first
				// lines (#1146). The smallest in bytes first: a short
				// file whole, not left out behind a long one's first
				// lines.
				a.Needs = needed(runDir)
				slices.SortStableFunc(a.Needs, func(x, y string) int { return neededBytes(picker, x) - neededBytes(picker, y) })
			}
			pick := picker.Pick(a)
			codeLeft -= share - pick.Left
			picks, pickedFor = append(picks, pick), append(pickedFor, d.is.ID)
			if c := pick.Cut(); c != "" {
				var caps, raise []string // the caps that cut it, each by its setting
				for _, by := range pick.CutBy() {
					if by == codepick.BytesName {
						caps = append(caps, fmt.Sprintf("%s (%d bytes)", by, s.CodeBytesMax))
					} else {
						caps = append(caps, fmt.Sprintf("%s (its share: %d of %d lines)", by, share, s.CodeLinesMax))
					}
					raise = append(raise, verdict.Setting("product-owner", by))
				}
				t.cuts = append(t.cuts, verdict.Finding{Rule: verdict.CutRule, Level: "warn", Where: fmt.Sprintf("#%d", d.is.ID),
					Message: fmt.Sprintf("the code given to the product owner's agent for it is cut past %s: %s; raise %s",
						strings.Join(caps, " and "), c, strings.Join(raise, ", or "))})
			}
			files := pick.Named()
			noCode = noCode || len(files) == 0
			review := written + kindAsked(d.is, d.st, kindAct)
			if r := backlog.SpecOpen(d.is, d.comments); cfg.SpecReview && r != nil {
				review += specFindings(d.is, d.st, d.comments, r)
			}
			if reason, _, ok := backlog.NoTest(d.comments); ok {
				if _, picked := backlog.ReadProof(d.comments, role); !picked {
					review += noTestSaid(reason, backlog.RetestSpent(d.comments))
				}
			}
			if unread, own := backlog.ScenariosUnread(d.is, d.st); unread && cfg.Tester && !backlog.Readied(d.is) {
				review += backlog.ScenariosUnreadTask(d.is, own)
			}
			review += markedSaid(d.is)
			if d.is.ID == issueEvent() && slices.Contains(d.is.Labels, backlog.LabelToRefine) {
				// The line's refine step on it: its size predicted at
				// ready (ADR-0064, §2), no call of its own.
				review += predictSaid(b, open, d.is, d.st, cfg.SplitOff(), testsFirst(b, d.is, cfg.Tester))
			}
			if slices.Contains(d.is.Labels, backlog.LabelBlocked) && d.is.ID == issueEvent() {
				review += blockedSaid(runDir, d.is, d.comments, cfg.SplitOff(), func() string { return spentSaid(b, open, d.is, d.comments) })
				if unrefinedRead(runDir) {
					// Blocked on its own read in refine, its text rewritten
					// and nothing decided on it: back to refine in the same
					// run, not blocked a week more (#1206). Never on a block
					// a person or another role made: a rewrite lifts none.
					fallback = append(fallback, intent.Intention{Kind: "unblock", Value: map[string]any{
						"issue": d.is.ID, "to": "refine", "own": true, "if-refined": true,
						"why": "its text rewritten on this read, nothing else decided on it: the read it was blocked on is gone, so it is read again in refine as it stands now"}})
				}
			} else if files := backlog.ScopeAsked(d.comments); len(files) > 0 && slices.Contains(d.is.Labels, backlog.LabelToRefine) {
				// The developer's first step sent it back to ask for files
				// outside the Scope (ADR-0063, §9): widened, or ready as it is.
				review += fmt.Sprintf("Sent back to refine by the developer: its code needs `%s`, outside the issue's %s; its comment says why. Decide one, its `why` one plain line: to widen it, `refine` with `scope`, the %[2]s rewritten whole with these files in it, and `sources`, then `ready`; or `ready` as it is, why the %[2]s must not take them in the `why`: the developer asked the same files again, a person decides.\n",
					strings.Join(files, "`, `"), work.Name(work.Scope))
			}
			if kept := keptMissing(d.is, d.comments); len(kept) > 0 {
				// New test files the developer kept beside the Scope
				// (ADR-0063, §9): folded into it at this run.
				review += fmt.Sprintf("The developer kept new test files beside its %s: `%s`. Add them to the %[1]s with a `refine`, the %[1]s whole and its `sources`; the engine adds any you leave out of a %[1]s you write.\n",
					work.Name(work.Scope), strings.Join(kept, "`, `"))
			}
			if readyOpen(d.is, d.st, cfg) {
				review += fmt.Sprintf("Ready: its %s, %s and %s are there, %s and %s a person's — propose `ready` when it is evident: one need, scenarios that prove it, nothing left open in its comments; the engine checks it. The %s is the tester's, written from its tests.\n",
					work.Name(work.Need), work.Name(work.Validation), work.Name(work.Scope), work.Name(work.Need), work.Name(work.Validation), work.Name(work.Verification))
			} else if drafts != nil && toTests(d.is, d.comments, cfg, drafts) {
				review += "Its tests: ready from before the line of states, no Draft of its tests — propose `ready` to send it to its tests (" + backlog.LabelNeedsTests + ") when it is still current and one run's work; `unready` with its reason when not.\n"
			}
			if sl, ok := timedOut[d.is.ID]; ok {
				review += timedOutSaid(sl, cfg.Wait)
			}
			if who := backlog.AnsweredAfter(d.is, d.st, d.notes); who != "" && answers[d.is.ID] != nil {
				review += fmt.Sprintf("@%s wrote after the assumptions you took on it (its %s section, below): their word stands over them. Rewrite what you built on an assumption they answered: your own sections, with `refine`; `unready` if it is ready and no longer holds. Never their %s.\n",
					who, backlog.AssumptionsName, work.Name(work.Need))
			}
			if a := answers[d.is.ID]; a != nil {
				review += revise(d.is, d.st, a, revisions)
			} else if hn, ok := heard[d.is.ID]; ok {
				review += commented(hn, d.is, d.st, d.notes)
			}
			// The issues it cites, open or closed: a closed one is history,
			// never work to come.
			if ids := backlog.CitedIssues(d.is.Body+"\n"+strings.Join(forge.Bodies(backlog.Counted(d.notes, d.is)), "\n"), issuePages); len(ids) > 0 {
				if all == nil && slices.ContainsFunc(ids, func(n int) bool { return !isOpen[n] }) {
					var err error
					if all, err = b.AllIssues(); err != nil {
						return t, err
					}
				}
				closed := map[int]string{}
				for _, is := range all {
					if is.Closed {
						closed[is.ID] = is.Reason
					}
				}
				if cites := backlog.IssueStates(ids, isOpen, closed); cites != "" {
					review = "Cites: " + cites + " — a closed issue is done or dropped: never a part to wait for nor work to come.\n" + review
				}
			}
			writeIssue(&task, role, s.rounds(), d.is, d.st, d.notes, files, isOpen, ledger.OwnBlockers(d.is), front[d.is.ID]+review)
			for _, f := range picker.Where(askOf(picker, tests, d.is, d.st, d.comments, aboutIt, 0)) {
				if !slices.Contains(code, f) {
					code = append(code, f)
				}
			}
			read := *d.st
			counted := len(backlog.Counted(d.notes, d.is))
			read.Judged, read.Comments, read.Heard, read.Rules = commit, 0, &counted, backlog.Reading
			if answers[d.is.ID] != nil {
				read.Revisions++
			}
			if r := backlog.SpecOpen(d.is, d.comments); cfg.SpecReview && r != nil {
				read.Answered, read.Revised = r.Body, backlog.Revisable(d.is, d.st, r)
			}
			if _, digest, ok := backlog.NoTest(d.comments); ok {
				read.NoTest = digest
			}
			if rewriteUnread(d.is, d.st, cfg) {
				read.Unread = backlog.BodyDigest(d.is.Body)
			}
			if sl, ok := timedOut[d.is.ID]; ok {
				read.Told = sl.Round // told once: not read again for it
			}
			read.Keep(d.is.Body)
			t.reads = append(t.reads, intent.Intention{Kind: "comment", Value: map[string]any{
				"issue": d.is.ID, "sticky": "state", "update-only": true, "if-answered": true, "body": backlog.FormatState(read)}})
		}
		if said == "" {
			said = releases(repo, b)
		}
		intro := fmt.Sprintf("The run is on commit %s. %d open issues to read against the code.\n\n%s\n%s", commit, len(t.ids), said, modes(cfg, ledger.Demoted(cfg.UndoneMax))+estimates(cfg))
		if pages := forge.FilePages(f); pages != "" {
			// What it writes cites a decision or a doc as a link a reader
			// follows (AGENTS.md, "Writing user docs").
			intro += fmt.Sprintf("Links: a file of the repository you cite — a decision, a doc, code — is a link to %s<its path>; an issue is cited as #<number>.\n\n", pages)
		}
		// An issue not read in this run, on the same code as one read, is
		// given whole: it may be the original a duplicate is closed
		// against, and a duplicate's original is quoted (ADR-0018).
		var related strings.Builder
		var others []string
		shown := 0
		// The rest in the backlog's order: nearest milestone, priority,
		// number, an issue after those it waits on (ADR-0028).
		ordered := make([]forge.Issue, len(left))
		byID := map[int]due{}
		for i, d := range left {
			ordered[i], byID[d.is.ID] = d.is, d
		}
		backlog.Order(ordered)
		for _, is := range ordered {
			d := byID[is.ID]
			files := picker.Where(askOf(picker, tests, d.is, d.st, d.comments, "", 0))
			if shown < relatedMax && slices.ContainsFunc(files, func(f string) bool { return slices.Contains(code, f) }) {
				shown++
				fmt.Fprintf(&related, "## #%d %s\n\n%s\n\n", d.is.ID, d.is.Title, clip(d.is.Body, bodyMax/2))
				continue
			}
			others = append(others, fmt.Sprintf("- #%d %s%s", d.is.ID, d.is.Title, place(d.is, isOpen)))
		}
		if shown > 0 {
			fmt.Fprintf(&task, "# Other open issues on the same code\n\nNot to judge in this run: given whole, as the original a duplicate would be closed against.\n\n%s", related.String())
		}
		if len(others) > 0 {
			fmt.Fprintf(&task, "# The other open issues, titles only\n\nNot read in this run; a duplicate may be one of them.\n\n%s\n\n", strings.Join(others, "\n"))
		}
		writeCode(&task, repo, picks, pickedFor)
		for i, pk := range picks {
			// Its code read: each file it names given, in part at least.
			if fs := pk.Named(); len(fs) > 0 && len(pk.Paths()) == len(fs) {
				t.codeRead = append(t.codeRead, strconv.Itoa(pickedFor[i]))
			}
			if given := pk.Paths(); len(given) > 0 {
				t.codeGiven = append(t.codeGiven, strconv.Itoa(pickedFor[i])+" "+strings.Join(given, " "))
			}
		}
		slices.SortFunc(t.codeRead, func(a, b string) int { x, _ := strconv.Atoi(a); y, _ := strconv.Atoi(b); return x - y })
		if noCode {
			writeFolders(&task, tracked)
		}
		t.text = intro + task.String()
		return t, nil
	}
	// What the agent reads fits the role's context budget (#427): the task
	// holds as many issues as issues-per-run lets, one fewer while its
	// prompt would be over the budget, as the engine measures it. The
	// issues left out are read first at the next run: nothing of them is
	// recorded. One issue alone over it is left to the engine, which says
	// the agent was not asked.
	n := min(len(toRead), s.IssuesPerRun)
	room := fits(runDir, repo, role)
	var t taskBuilt
	// The kind's light read (#878): the issues read before, no kind label,
	// by title and the start of their body, within what the kind act's
	// cap leaves once the n issues read in full are counted.
	waiting := ledger.Waiting(open) >= cfg.ProposalsMax
	light := lightKinds(kindless, toRead[:n], kindAct, kindMax, waiting)
	for n > 0 {
		var err error
		if t, err = build(n); errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		} else if err != nil {
			return fail(err)
		}
		light = lightKinds(kindless, toRead[:n], kindAct, kindMax, waiting)
		if room == nil || n == 1 || room(t.text+kindSection(light)) {
			break
		}
		n--
	}
	lightText := kindSection(light)
	t.text += lightText // the kind's light read rides on the full one: one call
	if cut := toRead[n:min(len(toRead), s.IssuesPerRun)]; len(cut) > 0 {
		var ids []string
		for _, d := range cut {
			ids = append(ids, fmt.Sprintf("#%d", d.is.ID))
		}
		findings = append(findings, verdict.Finding{Rule: "issues-cut", Level: "info",
			Message: fmt.Sprintf("%d of the issues to read left for the next run, read first then (%s): with them, the task would be over the role's context budget", len(cut), strings.Join(ids, ", "))})
	}
	findings = append(findings, t.cuts...)
	fallback = append(fallback, t.reads...)
	readIDs, judged = t.ids, len(t.ids)
	wasRead := t.wasRead
	if wasRead == nil {
		wasRead = map[int]bool{}
	}
	// An answer not read this run waits for the next: no 👀, no reply.
	for id := range answers {
		if !wasRead[id] {
			delete(answers, id)
		}
	}
	if data, err := yaml.Marshal(answers); err != nil {
		return fail(err)
	} else if err := os.WriteFile(filepath.Join(runDir, "in", "answers.yaml"), data, 0o644); err != nil {
		return fail(err)
	}
	// The issues read again with the change, said in the summary; the
	// sections of an issue not read kept now, so a change flags once — after
	// the agent's answer when it was given an issue to read for it.
	given := false
	for i := range found {
		for k, t := range found[i].Touch {
			if (t.How == backlog.TouchPart || t.How == backlog.TouchImport) && wasRead[t.Issue] {
				found[i].Touch[k].Read, given = true, true
			}
		}
	}
	for _, d := range rebase {
		if wasRead[d.is.ID] {
			continue
		}
		st := *d.st
		st.Sections = backlog.Basis(d.is.Body)
		fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{
			"issue": d.is.ID, "sticky": "state", "update-only": true, "if-answered": given, "body": backlog.FormatState(st)}})
	}
	if len(found) > 0 {
		if data, err := yaml.Marshal(found); err != nil {
			return fail(err)
		} else if err := os.WriteFile(filepath.Join(runDir, "in", "changes.yaml"), data, 0o644); err != nil {
			return fail(err)
		}
		for _, c := range found {
			findings = append(findings, verdict.Finding{Rule: "need-changed", Level: "info", Where: fmt.Sprintf("#%d", c.Issue),
				Message: changeLine(c)})
		}
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "issues-read"), []byte(strings.Join(readIDs, "\n")), 0o644); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", backlog.CodeReadFile), []byte(strings.Join(t.codeRead, "\n")), 0o644); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", backlog.CodeGivenFile), []byte(strings.Join(t.codeGiven, "\n")), 0o644); err != nil {
		return fail(err)
	}
	if len(fallback) > 0 {
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
			return fail(err)
		}
	}
	if judged == 0 && lightText != "" {
		// Nothing to read in full: the kinds alone, a task of their own,
		// read with its own short instruction (model.tasks.kind, #878).
		if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(kindIntro(commit, len(light))+lightText), 0o644); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte(kindTask+"\n"), 0o644); err != nil {
			return fail(err)
		}
		return writeFindings(runDir, findings)
	}
	if judged == 0 {
		// No task.md: the agent is not asked; the state comments are
		// written, a person's answers and acts undone read (ADR-0038).
		os.Remove(filepath.Join(runDir, "in", "task.md"))
		if len(fallback) > 0 || len(found) > 0 || len(ledger.Signs) > 0 {
			return writeFindings(runDir, findings)
		}
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no issue to judge", Findings: findings})
	}
	// The full read's instruction, never the kinds' alone.
	if err := os.Remove(filepath.Join(runDir, "in", "task-kind")); err != nil && !os.IsNotExist(err) {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(t.text), 0o644); err != nil {
		return fail(err)
	}
	return writeFindings(runDir, findings)
}

// taskBuilt is a run's task, written once it fits: its text, each issue
// read with its state as read, recorded once the agent answers.
type taskBuilt struct {
	text    string
	reads   []intent.Intention
	ids     []string
	wasRead map[int]bool
	// codeRead are the issues read whose files were all given: an effort
	// is estimated on them only (ADR-0040).
	codeRead []string
	// codeGiven are the files given, a line an issue read: its number,
	// then the paths (backlog.CodeGivenFile).
	codeGiven []string
	// cuts are what code-lines-max left out of the code each issue names,
	// said to the person (#1159).
	cuts []verdict.Finding
}

// askedAgainChars is the room kept in the prompt for an answer asked again:
// what was refused, about a thousand characters when it did not read.
const askedAgainChars = 2000

// fits says whether a task keeps the agent's prompt within the role's
// context budget (#427), measured as the engine measures it — the same
// facets, agent.Prompt, agent.Tokens —, room kept for an answer asked
// again; nil when the role or its budget is not known: the engine then
// refuses a prompt too large, and says so.
func fits(runDir, repo, name string) func(task string) bool {
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), name)
	if err != nil || r.Context.Budget <= 0 {
		return nil
	}
	// its issue's size tier multiplies it (ADR-0064)
	budget := r.Context.Budget * role.SizeFactor()
	r.Context.Budget = 0 // measured here, refused by the engine only
	return func(task string) bool {
		if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task), 0o644); err != nil {
			return true
		}
		system, user, err := agent.Prompt(agent.Request{RunDir: runDir, Repo: repo, Role: r})
		if err != nil {
			return true // the engine says why when it asks
		}
		return agent.Tokens(len(system)+len(user)+askedAgainChars) <= budget
	}
}

// acceptedBy names who labelled an issue workline:accepted, as the forge's
// label events say: "@name", or "a person of the project" when the forge
// does not say.
func acceptedBy(b forge.Backlog, is forge.Issue) (string, error) {
	events, err := b.LabelEvents(is.ID)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) {
			return "", err
		}
		events = nil
	}
	for i := len(events) - 1; i >= 0; i-- {
		if e := events[i]; e.Added && backlog.Accepted(forge.Issue{Labels: []string{e.Label}}) && e.Author != "" {
			return "@" + e.Author, nil
		}
	}
	return "a person of the project", nil
}

// revise tells the agent a person commented on what the role proposes on
// an issue (ADR-0038): which comments, the revision it is, and how it
// answers — its own drafts and the builder's part rewritten whole, a
// person's Need and Validation kept (ADR-0060).
func revise(is forge.Issue, st *backlog.State, a *backlog.Answer, max int) string {
	mine := backlog.Own(is, st)
	if lift(is, st) || slices.Contains(mine, work.ByMeaning(is.Body)[work.Need].Heading) {
		if ok, _ := backlog.RolesTop(is, st); ok {
			mine = append(mine, "its text on top, a role's: taken from the top into the `need` you give")
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "A person of the project commented on what you proposed on it (@%s, the last comments below): revision %d of %d. Read them as asking you to revise: ", a.Who, a.Revision, max)
	if len(mine) > 0 {
		fmt.Fprintf(&b, "give a `refine` with the whole new text of the sections that are still yours (%s), answering them; ", strings.Join(mine, ", "))
	}
	fmt.Fprintf(&b, "propose again what still stands, changed as they ask, and nothing they turned down. A %s, %s, %s or %s a person wrote, or a section they deleted, is theirs: never written. When they ask for nothing you can change, say why in a `note`.\n",
		work.Name(work.Need), work.Name(work.Validation), work.Name(work.Example), work.Name(work.Steps))
	return b.String()
}

// commented tells the agent a person commented on an issue with no
// proposal waiting on them (ADR-0038, amended): their word comes first.
// One they had set aside is brought back by it: said so — who, when, their
// words —, and read as any issue at the role's level unless the comment
// says otherwise. One whose body changed since it was set aside is back
// for that change, not the comment: the plain wording.
func commented(n forge.Note, is forge.Issue, st *backlog.State, notes []forge.Note) string {
	if st.Aside == "" || st.Aside != backlog.BodyDigest(is.Body) {
		// The comments it has not read, quoted (#1331): left among the
		// issue's comments below, an answer was taken as settled already.
		fresh := backlog.Unheard(st, notes, is)
		if len(fresh) == 0 {
			fresh = []forge.Note{n}
		}
		by := ""
		if n.Author != "" {
			by = " (@" + n.Author + ")"
		}
		var b strings.Builder
		fmt.Fprintf(&b, "A person commented since you last read it%s: read their word first — what it says decides. "+
			"What it settles that the body leaves open — a value, a bound, a choice, an answer to the reviewer's question — goes into the %s with a `refine`, its whole new text: the tester and the developer build from the body, never from its comments. Not read before:\n",
			by, work.Name(work.Scope))
		for _, f := range fresh {
			fmt.Fprintf(&b, "> %s\n\n", strings.ReplaceAll(clip(marker.ReplaceAllString(f.Body, ""), quoteMax), "\n", "\n> "))
		}
		if st.AnswerHeld > 0 && st.AnswerHeld == len(backlog.Counted(notes, is)) {
			// Its ready held for them last run (#1331): read again, line by
			// line against the spec, before a ready taken this time.
			fmt.Fprintf(&b, "Your `ready` on it was held last run: you read these comments and wrote nothing of them. Check each against the spec, line by line: "+
				"what one settles that the spec leaves out or says otherwise goes into the %s with a `refine`. Propose `ready` again only if the spec already says all of it: it is taken this time.\n", work.Name(work.Scope))
		}
		return b.String()
	}
	on := ""
	if len(n.Created) >= len("2006-01-02") {
		if _, err := time.Parse("2006-01-02", n.Created[:10]); err == nil {
			on = " on " + n.Created[:10]
		}
	}
	quote := strings.ReplaceAll(clip(marker.ReplaceAllString(n.Body, ""), quoteMax), "\n", "\n> ")
	return fmt.Sprintf("Brought back by a comment from @%s%s:\n> %s\n\n", n.Author, on, quote) +
		"Read their word first; then treat it as any issue at your level: complete what it lacks — draft its missing sections —, propose, or move it to `ready`, as the modes above say. " +
		"Unless the comment says otherwise (not now, leave it, they will do it themselves): then propose nothing, and say so in a `note`.\n"
}

// testsOf is the project's one definition of a test file (ADR-0065);
// unread, the generic one.
func testsOf(repo string) testdef.Def {
	d, _ := role.TestDef(os.Getenv("WORKLINE_ROLES_DIR"), repo)
	return d
}

// quoteMax bounds a person's comment quoted where it brought an issue
// back; the comment is given whole among the issue's comments.
const quoteMax = 600

// evidence reads a parent's parts — every issue listed once a run, into
// all — and what closed each part delivered, and writes its report
// (ADR-0029).
func evidence(b forge.Backlog, repo string, parent forge.Issue, ids []int, role string, all *[]forge.Issue) (backlog.Evidence, error) {
	if *all == nil {
		list, err := b.AllIssues()
		if err != nil {
			return backlog.Evidence{}, err
		}
		*all = append([]forge.Issue{}, list...)
	}
	var parts []backlog.Part
	for _, id := range ids {
		p := backlog.Part{ID: id, Gone: true}
		for _, is := range *all {
			if is.ID == id {
				p.Issue, p.Gone = is, false
			}
		}
		if p.Delivered() {
			// What closed it is evidence beside the closing: a forge that
			// refuses to say fails no run, and the part says it was not
			// read — never "by hand"; one that does not answer stops it.
			closers, err := b.Closers(id)
			if errors.Is(err, forge.ErrUnreachable) {
				return backlog.Evidence{}, err
			}
			p.Closers, p.Unread = closers, err
		}
		parts = append(parts, p)
	}
	tests := testsOf(repo)
	return backlog.ReadEvidence(parent, parts, role, backlog.CodeTests(repo, tests), tests), nil
}

// issueRefs says issue numbers as references: "#12, #13".
func issueRefs(ids []int) string {
	var out []string
	for _, n := range ids {
		out = append(out, fmt.Sprintf("#%d", n))
	}
	return strings.Join(out, ", ")
}

// askJudge writes the question a second judge answers on an issue
// announced obsolete and due (ADR-0024): the issue, the evidence, the code
// quoted as it is now, around the quote; the model that proposed it named,
// for the judge to stand apart from it.
func askJudge(runDir, repo string, is forge.Issue, a backlog.Announcement) error {
	dir := filepath.Join(runDir, "in", "judge", fmt.Sprintf("%s%d", backlog.JudgeKeyPrefix, is.ID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	code := ""
	if a.Quote.Path != "" {
		if data, err := exec.Command("git", "-C", repo, "show", "HEAD:"+a.Quote.Path).Output(); err == nil {
			lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			from, to := 1, len(lines)
			if f, t, _, ok := backlog.Locate(repo, a.Quote.Path, a.Quote.Text); ok {
				from, to = max(1, f-judgeContext), min(len(lines), t+judgeContext)
			}
			for i := from; i <= to; i++ {
				lines[i-1] = fmt.Sprintf("%4d  %s", i, lines[i-1])
			}
			code = strings.Join(lines[from-1:to], "\n")
		}
	}
	data, _ := json.Marshal(map[string]string{"question": backlog.JudgeQuestion, "author": a.By,
		"material": backlog.JudgeMaterial(repo, is, a, code)}) // JSON, which YAML reads: code may start a line with a tab
	return os.WriteFile(filepath.Join(dir, "question.yaml"), data, 0o644)
}

// judgeContext is the lines given around a quote to the judge.
const judgeContext = 60

// readJudged reads the judge's answer on an announced issue, as the engine
// wrote it; false when there is none.
func readJudged(runDir string, id int) (backlog.Judged, bool) {
	var j backlog.Judged
	data, err := os.ReadFile(filepath.Join(runDir, "in", "judge", fmt.Sprintf("%s%d", backlog.JudgeKeyPrefix, id), "answer.yaml"))
	if err != nil || yaml.Unmarshal(data, &j) != nil {
		return j, false
	}
	return j, true
}

// writeIssue gives one issue to the agent: what the engine knows of it, its
// body, its last comments, the engine's own left out.
func writeIssue(b *strings.Builder, role string, rounds int, is forge.Issue, st *backlog.State, notes []forge.Note, files []string, isOpen map[int]bool, own []int, change string) {
	comments := forge.Bodies(notes)
	fmt.Fprintf(b, "## #%d %s\n\n", is.ID, is.Title)
	if len(is.Labels) > 0 {
		fmt.Fprintf(b, "Labels: %s\n", strings.Join(is.Labels, ", "))
	}
	sources := "none named yet"
	if len(st.Sources) > 0 {
		sources = strings.Join(st.Sources, ", ")
	}
	if is.Milestone != "" {
		fmt.Fprintf(b, "Milestone: %s\n", is.Milestone)
	}
	if set := backlog.PriorityLabels(is); len(set) > 0 {
		whose := ""
		if st.Priority != backlog.HiddenLevel && (len(set) != 1 || set[0] != st.Priority) {
			whose = " (a person's: kept)"
		}
		fmt.Fprintf(b, "Priority: %s%s\n", scale.Value(backlog.Priority(is)), whose)
	}
	if set := backlog.EffortLabels(is); len(set) > 0 {
		whose := ""
		if st.Effort != backlog.HiddenLevel && (len(set) != 1 || set[0] != st.Effort) {
			whose = " (a person's: kept)"
		}
		value := scale.EffortValue(backlog.Effort(is))
		if scale.Used.Effort.Weight {
			// The weight the issue holds, a person's 2 as it is.
			value = fmt.Sprintf("%s (weight %d)", scale.Effort.Name(backlog.Effort(is)), *is.Weight)
		}
		fmt.Fprintf(b, "Effort: %s%s\n", value, whose)
	} else if scale.Used.Effort.Weight && len(backlog.Parts(is)) > 0 {
		// GitLab sums a parent's weight from its parts' (ADR-0040).
		b.WriteString("Effort: not estimated here: a parent, GitLab sums its parts' weights\n")
	}
	if by := backlog.OpenedBy(is.Body); by != "" {
		// A role's finding, opened through the one way (ADR-0018): the
		// product owner takes it from here, as any other issue.
		fmt.Fprintf(b, "Opened by: the %s role, on a finding of its own — a draft to refine: its %s and %s are not a person's yet\n", by, work.Name(work.Need), work.Name(work.Validation))
	} else if is.Author != "" {
		whose := ", of the project"
		switch {
		case backlog.Bot(is.Author):
			whose = ", a bot: nobody to ask — the project's to complete" // #429
		case !is.Insider:
			whose = ", without write access to the project"
		}
		fmt.Fprintf(b, "Opened by: %s%s\n", is.Author, whose)
	}
	if lift(is, st) {
		// A role's text no person edited, and no Need yet (ADR-0039,
		// amended): the engine tells so, from what it recorded.
		fmt.Fprintf(b, "Description: the text above its sections is a role's, and no person edited it — give it as the `need` draft, rewritten in plain words: the engine takes it from the top, its %s said once\n", work.Name(work.Need))
	}
	if st.Title != "" && st.Title != is.Title {
		b.WriteString("Title: a person's, set after the role's (kept)\n")
	}
	if parts := backlog.SplitInto(is, st); len(parts) > 0 {
		// Split already (ADR-0022): its children are issues of their own;
		// the parent is a person's to accept as they close (ADR-0029).
		var ids []string
		for _, id := range parts {
			state := "closed"
			if isOpen[id] {
				state = "open"
			}
			ids = append(ids, fmt.Sprintf("#%d (%s)", id, state))
		}
		fmt.Fprintf(b, "Split into: %s — not split again, never closed by the role: a person accepts it once its parts are closed\n", strings.Join(ids, ", "))
	}
	if a, _ := backlog.LastAnnouncement(notes, role); a != nil && !slices.Contains(st.Kept, a.Key()) {
		fmt.Fprintf(b, "Announced obsolete on %s (%s): closed by the engine once its delay passed, if nobody wrote and a second judge agrees — do not propose closing it again\n", a.Announced, a.Quote.Path)
	}
	if len(st.Kept) > 0 {
		b.WriteString("Kept open after an announcement as obsolete: someone wrote, took the label off, or a judge disagreed — never announced again on the same quote; only on code changed since, if it truly solves it\n")
	}
	if blockers := backlog.Blockers(is); len(blockers) > 0 {
		// What it waits on (ADR-0028): a person's link is kept; a closed
		// blocker holds nothing back.
		var said []string
		for _, n := range blockers {
			state := "closed"
			if isOpen[n] {
				state = "open"
			}
			if slices.Contains(own, n) {
				state += ", set by the role"
			}
			said = append(said, fmt.Sprintf("#%d (%s)", n, state))
		}
		fmt.Fprintf(b, "Waits on: %s — ordered after the open ones; a depend only adds what is missing; an undepend only takes off a link set by the role\n", strings.Join(said, ", "))
	}
	fmt.Fprintf(b, "Sections: %s\n", sections(is, st))
	if st.Silent != "" && !strings.HasPrefix(st.Silent, backlog.NoTestSilent) && !backlog.HeldForScenarios(st.Silent) {
		// Read before and left unrefined (#951): refined now, from its
		// text and the code given, or asked — never left so again.
		fmt.Fprintf(b, "Left unrefined when last read: %s. Refine it now from its text and the code given — or the repository's folders —, or ask its reporter.\n", strings.TrimSuffix(st.Silent, "."))
	}
	fmt.Fprintf(b, "Sources: %s. Confirmed at: %s.\n", sources, st.Confirmed)
	if len(files) > 0 {
		fmt.Fprintf(b, "Code it names, given below: %s.\n", strings.Join(files, ", "))
	}
	if e := backlog.ReadExchange(comments, role); e.Rounds > 0 {
		// The conversation with its reporter so far: what was asked or
		// proposed is shown below, among the comments (ADR-0021).
		answer := "not answered since: nothing more is written to them before an answer"
		if e.Answered {
			answer = "answered since: read the answer — refine, mark ready, or ask what is still missing, never a question asked before"
		}
		fmt.Fprintf(b, "Written to its reporter: %d of %d times; %s.\n", e.Rounds, rounds, answer)
	}
	if backlog.Outsider(is) && !backlog.Accepted(is) {
		fmt.Fprintf(b, "Its reporter is outside the project: a `refine` is proposed to them in a comment, not written in the body; `why` says what you understood of the issue; a `split`, a `rename` or `ready` is proposed to the project on the issue.\n")
	}
	if change != "" {
		b.WriteString(change)
	}
	b.WriteString("\n")
	// Its body whole (#1331): a section is rewritten whole, and a line
	// past a cut, a person's answer written there, was lost with it.
	fmt.Fprintf(b, "%s\n\n", strings.TrimSpace(is.Body))
	type said struct {
		who, text string
		person    bool // a person's who counts (backlog.Counted)
	}
	var kept []said
	for _, n := range notes {
		switch {
		case backlog.Round(n.Body, role):
			kept = append(kept, said{"", "(The product owner wrote:) " + marker.ReplaceAllString(n.Body, ""), false})
		default:
			if _, engine := backlog.EngineMarker(n.Body); !engine {
				person := len(backlog.Counted([]forge.Note{n}, is)) == 1
				kept = append(kept, said{author(n, is), marker.ReplaceAllString(n.Body, ""), person}) // a person's, quoting the engine's maybe
			}
		}
	}
	// The last commentsMax, and every comment of a person who counts
	// before them (#1331): an answer left out was rewritten away.
	if len(kept) > commentsMax {
		var left []said
		for i, c := range kept {
			if c.person || i >= len(kept)-commentsMax {
				left = append(left, c)
			}
		}
		if out := len(kept) - len(left); out > 0 {
			fmt.Fprintf(b, "(%d earlier comments left out, none a person's)\n\n", out)
		}
		kept = left
	}
	for _, c := range kept {
		fmt.Fprintf(b, "Comment%s:\n> %s\n\n", c.who, strings.ReplaceAll(clip(c.text, bodyMax), "\n", "\n> "))
	}
}

// noTestUnread says whether the tester found no test possible for an
// issue and the role was not given it yet, nor a kind picked for it.
func noTestUnread(comments []string, st *backlog.State, role string) bool {
	_, digest, ok := backlog.NoTest(comments)
	if !ok || digest == st.NoTest {
		return false
	}
	_, picked := backlog.ReadProof(comments, role)
	return !picked
}

// noTestSaid tells the agent the tester found no test possible, and what
// it may do: pick the proof in place of new tests, send the issue back to
// its tests with scenarios a test checks (ADR-0059, decision 4), or, the
// tester's reason its own, send it back unchanged once (#1053); spent,
// the retest is not offered.
func noTestSaid(reason string, spent bool) string {
	retest := "If the reason is the tester's own, not the issue's — code missing from its task, a run that failed — send it back to its tests unchanged, once: `ready` with `retest: true` and why. "
	if spent {
		retest = "It was sent back to its tests unchanged once already, and the tester found none again: no second retest. "
	}
	return fmt.Sprintf("No test possible, says the tester: %s\n"+
		"Pick its proof in place of new tests — `ready` with `proof: {kind: <kind>, reason: <why it proves the need>}`, the kind one of: %s (%s), %s (%s), %s (%s; add `test: <its name>` and, if not the project's setting, `runs: <N>`). "+
		"Or rewrite what keeps a test from being written — its %s, yours to rewrite; its %s where it is your draft, so a test checks it — with a `refine` giving each section whole, or ask its reporter: a `ready` with no proof sends it back to its tests beside that rewrite. %s"+
		"A `ready` alone is refused, and the line blocks the issue for a person.\n",
		strings.TrimSpace(reason), backlog.KindSuite, backlog.ProofKinds[backlog.KindSuite], backlog.KindDocs, backlog.ProofKinds[backlog.KindDocs],
		backlog.KindFlaky, backlog.ProofKinds[backlog.KindFlaky], work.Name(work.Scope), work.Name(work.Validation), retest)
}

// rewriteUnread says whether the agent is told to rewrite an issue's
// Validation in a form the tester reads (#855): the line routes the
// tester, the issue is in refine, its Validation the role's own and
// unread, and the agent not told yet for this body.
func rewriteUnread(is forge.Issue, st *backlog.State, cfg backlog.Config) bool {
	unread, own := backlog.ScenariosUnread(is, st)
	return unread && own && cfg.Tester && !backlog.Readied(is) && st.Unread != backlog.BodyDigest(is.Body)
}

// markedSaid tells the agent which of an issue's Need and Validation its
// writer marked `Draft:` (#1019): unfinished, they hold it from ready
// until rewritten — its to do — or a person accepts them; "" for none.
func markedSaid(is forge.Issue) string {
	if backlog.Accepted(is) || backlog.Readied(is) {
		return ""
	}
	var names []string
	for _, m := range []work.Meaning{work.Need, work.Validation} {
		if backlog.MarkedDraft(is.Body, m) {
			names = append(names, work.ByMeaning(is.Body)[m].Heading)
		}
	}
	if len(names) == 0 {
		return ""
	}
	opens := " opens"
	if len(names) > 1 {
		opens = " each open"
	}
	return "Its " + strings.Join(names, " and ") + opens + " with `Draft:`: its writer marked it unfinished, so the issue is not ready as it stands. Yours to rewrite: answer with a `refine` giving each such section whole, the `Draft:` gone, its meaning kept.\n"
}

// author says who wrote a comment, as the agent is told: its reporter, a
// person of the project, a bot, or someone else — whose word decides
// nothing (ADR-0021); "" when the forge does not say.
func author(n forge.Note, is forge.Issue) string {
	switch {
	case n.Author == "":
		return ""
	case n.Bot:
		return " by @" + n.Author + ", a bot"
	case n.Author == is.Author:
		return " by @" + n.Author + ", its reporter"
	case n.Insider:
		return " by @" + n.Author + ", of the project"
	}
	return " by @" + n.Author + ", outside the project"
}

// place says an issue's milestone and priority, when it has them.
func place(is forge.Issue, isOpen map[int]bool) string {
	var out []string
	if w := backlog.Waiting(is, isOpen); len(w) > 0 {
		var ids []string
		for _, n := range w {
			ids = append(ids, fmt.Sprintf("#%d", n))
		}
		out = append(out, "waits on "+strings.Join(ids, ", "))
	}
	if is.Milestone != "" {
		out = append(out, is.Milestone)
	}
	if n := backlog.Priority(is); n > 0 {
		out = append(out, scale.Said(n))
	}
	if len(out) == 0 {
		return ""
	}
	return " (" + strings.Join(out, ", ") + ")"
}

// placed says an issue's milestone and priority after its title, only
// those it has: " — milestone v1.0, priority 2", or nothing.
func placed(is forge.Issue) string {
	var out []string
	if is.Milestone != "" {
		out = append(out, "milestone "+is.Milestone)
	}
	switch n := backlog.Priority(is); {
	case scale.Used.Priority.Hidden() || scale.Used.Effort.Hidden():
		// Visible to organization members only, on a public repository:
		// the order kept, the values never shown (ADR-0040).
	case scale.Used.Detected && scale.Used.Effort.Said == "":
		// No effort kept anywhere: its priority by name, or none said.
		out = append(out, scale.PriorityName(n))
	case scale.Used.Detected:
		// Its cell of the value-vs-effort grid, a value missing said
		// (ADR-0040).
		out = append(out, scale.Cell(n, backlog.Effort(is)))
	case n > 0:
		out = append(out, scale.Said(n))
	}
	if len(out) == 0 {
		return ""
	}
	return " — " + strings.Join(out, ", ")
}

// unestimated counts the ready issues offered that miss a priority or an
// effort, as Next says them (ADR-0040): "3 of 9 ready issues have no
// priority: they follow their number".
func unestimated(open []forge.Issue, report int, isOpen map[int]bool) []verdict.Finding {
	ready, noPriority, noEffort := 0, 0, 0
	for _, is := range open {
		if is.ID == report || !backlog.Offered(is, isOpen) {
			continue
		}
		ready++
		if backlog.Priority(is) == 0 {
			noPriority++
		}
		if backlog.Effort(is) == 0 {
			noEffort++
		}
	}
	var says []string
	if noPriority > 0 {
		says = append(says, fmt.Sprintf("%d of %d ready issues have no priority: they follow their number", noPriority, ready))
	}
	if noEffort > 0 && scale.Used.Effort.Said != "" {
		which := "ready issues "
		if len(says) > 0 {
			which = ""
		}
		says = append(says, fmt.Sprintf("%d of %d %shave no effort, read as %s", noEffort, ready, which, scale.Effort.Name(scale.Effort.None)))
	}
	if len(says) == 0 {
		return nil
	}
	return []verdict.Finding{{Rule: "next-unestimated", Level: "info", Message: strings.Join(says, "; ") + "."}}
}

// stuck says an axis whose values the role set sit on one level, past 10
// of them, over 60% (ADR-0040): "Effort: Medium on 8 of the 11 issues the
// role estimated."
func stuck(states map[int]*backlog.State, isOpen map[int]bool) []verdict.Finding {
	var priorities, efforts []int
	for _, id := range slices.Sorted(maps.Keys(states)) {
		if st := states[id]; st != nil && isOpen[id] {
			if st.Priority > 0 {
				priorities = append(priorities, st.Priority)
			}
			if st.Effort > 0 {
				efforts = append(efforts, st.Effort)
			}
		}
	}
	var out []verdict.Finding
	for _, a := range []struct {
		s      scale.Scale
		levels []int
		verb   string
	}{{scale.Priority, priorities, "ordered"}, {scale.Effort, efforts, "estimated"}} {
		if say := scale.Stuck(a.s, a.levels); say != "" {
			out = append(out, verdict.Finding{Rule: "estimates-stuck", Level: "info",
				Message: fmt.Sprintf("%s%s: %s issues the role %s.", strings.ToUpper(a.s.Axis[:1]), a.s.Axis[1:], say, a.verb)})
		}
	}
	return out
}

// estimates tells the agent how the role sets an issue's priority and
// effort where the project's settings say it keeps them (ADR-0040): once,
// only where empty; an effort beside the order, on code given only.
func estimates(cfg backlog.Config) string {
	on := func(kind string) bool { s, ok := cfg.Acts[kind]; return ok && s.Mode != backlog.Off }
	if !scale.Used.Detected || !on("order") && !on("effort") {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Priority and effort\n\nEach is set once, only where the issue has none: a value there is a person's, or yours from an earlier run, never estimated again.\n\n")
	if on("order") {
		var levels []string
		for l := 1; l <= scale.Priority.Count(); l++ {
			levels = append(levels, fmt.Sprintf("%d %s", l, scale.Priority.Name(l)))
		}
		fmt.Fprintf(&b, "- Priority, from the issue and its code: %s. Data lost, a regression, others blocked first; what can wait last.\n", strings.Join(levels, ", "))
	}
	if on("effort") && scale.Used.Effort.Said != "" {
		var levels []string
		for l := 1; l <= scale.Effort.Count(); l++ {
			levels = append(levels, fmt.Sprintf("%d %s (%s)", l, scale.Effort.Name(l), scale.EffortAnchor(l)))
		}
		fmt.Fprintf(&b, "- Effort, how hard it is to build: %s. Write it in the issue's order, `order: {issue: 12, priority: 2, effort: 1, why: \"…\"}`, the priority left out when the issue has one; only for an issue whose code is given below.\n", strings.Join(levels, ", "))
	}
	return b.String() + "\n"
}

// sections says which of the four sections an issue's body has — an
// example or the steps to reproduce after them, never required —, and
// which are drafts no person made theirs yet.
func sections(is forge.Issue, st *backlog.State) string {
	body := is.Body
	have := work.ByMeaning(body)
	var there, missing []string
	for _, m := range append(slices.Clone(work.Required), work.Example, work.Steps) {
		text, name := strings.TrimSpace(have[m].Text), have[m].Heading
		switch {
		case m == work.Need && text == "" && backlog.NeedOnTop(is, st):
			// A person's text on top is its Need (ADR-0039, amended).
			there = append(there, work.Name(m)+" (the text on top, a person's)")
		case text == "" && !slices.Contains(work.Required, m):
		case text == "":
			missing = append(missing, work.Name(m))
		case backlog.IsDraft(body, m, st):
			there = append(there, name+" (draft)")
		case backlog.MarkedDraft(body, m) && !backlog.Accepted(is):
			there = append(there, name+" (its writer's draft, opening with `Draft:`: yours to rewrite)") // #1019
		case backlog.Drafted(name):
			there = append(there, name+" (a person's)") // not marked a draft: theirs
		default:
			there = append(there, name)
		}
	}
	out := "none"
	if len(there) > 0 {
		out = strings.Join(there, ", ")
	}
	if len(missing) > 0 {
		out += "; missing: " + strings.Join(missing, ", ")
	}
	return out
}

// trackedFiles lists the files of the commit the run is on.
func trackedFiles(repo string) map[string]bool {
	out, _ := exec.Command("git", "-C", repo, "ls-files").Output()
	files := map[string]bool{}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		files[f] = true
	}
	return files
}

// provenance is the line the engine ends an imported issue with.
var provenance = regexp.MustCompile(`(?m)^Opened from .* by the [\w -]+ role\.\r?$`)

// marker is the engine's hidden text in a body: the file it was imported
// from, a key.
var marker = regexp.MustCompile(`<!-- workline:[^>]*-->`)

// folders are the folders of the commit's files, each above them.
func folders(tracked map[string]bool) map[string]bool {
	out := map[string]bool{}
	for f := range tracked {
		for d := path.Dir(f); d != "." && d != "/" && !out[d]; d = path.Dir(d) {
			out[d] = true
		}
	}
	return out
}

// foldersMax bounds the folders a task lists: those nearest the root.
const foldersMax = 150

// writeFolders lists the repository's folders, nearest the root first,
// for an issue that names no file: the agent names its Scope's files
// from them instead of leaving it out.
func writeFolders(b *strings.Builder, tracked map[string]bool) {
	var list []string
	for d := range folders(tracked) {
		list = append(list, d)
	}
	slices.SortFunc(list, func(a, b string) int {
		if n := strings.Count(a, "/") - strings.Count(b, "/"); n != 0 {
			return n
		}
		return strings.Compare(a, b)
	})
	more := ""
	if len(list) > foldersMax {
		more = fmt.Sprintf("- and %d deeper folders\n", len(list)-foldersMax)
		list = list[:foldersMax]
	}
	slices.Sort(list)
	fmt.Fprintf(b, "# The repository's folders\n\nAn issue above names no file: name the files or folders its %s touches from these, in `sources` (a folder named is given with its files at the next run); no line is quoted from a folder.\n\n- %s\n%s\n", work.Name(work.Scope), strings.Join(list, "\n- "), more)
}

// askOf is what the picker is asked for an issue: its title, its body
// and a person's comments, its sources, the lines of the file it was
// imported from that name its item (writtenAt: aboutIt), never that file
// itself, which is where it was written, not its code (#548, #1063).
func askOf(picker *codepick.Picker, tests testdef.Def, is forge.Issue, st *backlog.State, comments []string, aboutIt string, budget int) codepick.Ask {
	from, _, _, _ := backlog.Imported(is.Body)
	body := marker.ReplaceAllString(provenance.ReplaceAllString(is.Body, ""), "")
	for _, c := range comments {
		if !strings.Contains(c, "<!-- workline:") { // a person's: an answer may name the code
			body += "\n" + c
		}
	}
	text := is.Title + "\n" + body + "\n" + aboutIt
	var paths []string
	for _, src := range st.Sources {
		paths = append(paths, backlog.RepoPath(src))
	}
	// A link to a file on the forge is that file.
	for _, m := range backlog.PathLike.FindAllString(text, -1) {
		if p := backlog.RepoPath(m); p != m && picker.Tracked(p) && !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	a := codepick.Ask{Text: text, Paths: paths, Tests: tests, Budget: budget, BudgetName: "code-lines-max"}
	if from != "" {
		a.Skip = []string{from}
	}
	return a
}

// writeCode gives the code each issue read names, as the picker planned
// it, each file with its last commits: what changed it, so an issue the
// code solved can be told.
func writeCode(b *strings.Builder, repo string, picks []codepick.Pick, ids []int) {
	head := false
	for i, pk := range picks {
		if len(pk.Named()) == 0 {
			continue
		}
		if !head {
			b.WriteString("# Code the issues name, as it is at this commit\n\nFound for each issue by the names and paths it writes: each definition of a name whole, the lines naming it, a file named whole when it fits; the rest said.\n")
			head = true
		}
		fmt.Fprintf(b, "\n## The code #%d names\n", ids[i])
		pk.Write(b, codepick.Style{Numbered: true, About: func(f string) string {
			log, _ := exec.Command("git", "-C", repo, "log", "-5", "--format=%h %as %s", "--", f).Output()
			return fmt.Sprintf("\nLast commits:\n%s\n", strings.TrimSpace(string(log)))
		}})
	}
	if head {
		b.WriteString("\n")
	}
}

// modes tells the agent how far the role goes in this run (ADR-0026): each
// kind of act's mode, and where it comes from — the level, the project's
// setting, or a person's undoing — so it writes nothing that is dropped.
func modes(cfg backlog.Config, demoted []string) string {
	var b strings.Builder
	level := cfg.Level
	if cfg.LevelFrom != "" {
		level += " (" + cfg.LevelFrom + ")" // ADR-0060, §3: the role's own, the project's default, the role's
	}
	fmt.Fprintf(&b, "# What you may do in this run\n\nAutonomy: %s — %s. Each kind of act, its mode and where it comes from (level; setting: the project's; demoted: a person undid one of its acts):\n\n", level, levelSays[cfg.Level])
	for _, m := range cfg.Modes(demoted) {
		fmt.Fprintf(&b, "- %s: %s\n", m.Kind, m.Say())
	}
	b.WriteString("\n`act`: done by the engine once checked. `propose`: proposed on the issue for a person to accept, never done by the role — propose only what you would do. `off`: dropped — do not write it.\n\n")
	b.WriteString("Whose each section is: the task marks a draft of yours `(draft)`; a section not marked a draft is a person's — they wrote it or made it theirs. Never hold an act back for not knowing who wrote a section.\n\n")
	b.WriteString(waitSays(cfg.Wait))
	return b.String()
}

// waitSays tells the agent how long a question waits for its answer
// (ADR-0060): for ever, a delay after which it is woken to assume, or
// none — an `assume` beside each `ask`.
func waitSays(w backlog.Delay) string {
	switch {
	case w.Never:
		return "A question you ask waits for its answer (acts.ask.wait never): write no `assume`.\n\n"
	case w.D == 0:
		return fmt.Sprintf("A question you ask does not wait (acts.ask.wait 0): beside each `ask`, give an `assume` for the same issue — your best answer to it, one plain sentence, written on the issue under %s for the person to correct — and go on as if it were the answer: refine, `ready` when evident. Your drafts are taken as written at once.\n\n", backlog.AssumptionsName)
	}
	return fmt.Sprintf("A question you ask waits %s for its answer (acts.ask.wait): the task says when one got none; then give an `assume`, your best answer, and go on. Drafts with no answer for as long are taken as written by the engine.\n\n", w.Say())
}

// timedOutSaid tells the agent its question got no answer for the delay
// (ADR-0060): take the best assumption, and go on.
func timedOutSaid(s backlog.Silence, w backlog.Delay) string {
	return fmt.Sprintf("Unanswered: your question of %s got no answer in %s (acts.ask.wait). Give an `assume` with your best answer to it — one plain sentence, from what the issue, its comments and the code show, written on the issue under %s for the person to correct — then go on as if it were the answer: `refine` what it lacks, `ready` when evident. Ask nothing more: the question stands. The person's %s and %s are never rewritten.\n",
		s.Since.UTC().Format("2006-01-02 15:04 UTC"), w.Say(), backlog.AssumptionsName, work.Name(work.Need), work.Name(work.Validation))
}

// draftsDue says whether an issue's drafts no person accepted got no
// answer for the delay (ADR-0060): at once with none; otherwise from the
// forge's time of its workline:draft label, or a person's comment after.
func draftsDue(b forge.Backlog, is forge.Issue, notes []forge.Note, w backlog.Delay, now time.Time) (bool, error) {
	switch {
	case w.Never:
		return false, nil
	case w.D == 0:
		return true, nil
	}
	t, err := b.Trail(is.ID, backlog.LabelDraft)
	if err != nil {
		return false, err
	}
	since, known := backlog.DraftsSince(t.Labeled, notes, is)
	return known && w.Passed(since, now), nil
}

// levelSays is what each autonomy level means for the agent (ADR-0038).
var levelSays = map[string]string{
	"cautious":     "you note and propose: what you would do waits on a person",
	"normal":       "you complete what is missing, and an evident issue moves to ready alone",
	"enterprising": "you complete what is missing, and issues move to ready much more often",
}

// issueEvent is the issue an `issue` event names, its state having woken
// the role (ADR-0059); -1 on that event with none named, nothing read; 0
// on any other event.
func issueEvent() int {
	if os.Getenv("WORKLINE_EVENT") != "issue" {
		return 0
	}
	kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	n, err := strconv.Atoi(num)
	if kind != "issue" || err != nil || n <= 0 {
		return -1
	}
	return n
}

// needed are the files the line gives a run again for the code its agent
// said it lacked (the input `needs`, comma-separated): none on a first
// run.
func needed(runDir string) []string {
	data, err := os.ReadFile(filepath.Join(runDir, "in", "input", engine.NeedsInput))
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range strings.Split(string(data), ",") {
		if p = strings.TrimSpace(p); p != "" && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// neededBytes is the size of a file the agent said it lacked, as the
// picker finds it (a line suffix off, a short name resolved): the budget
// is in bytes too (code-bytes-max). 0: none of the repository.
func neededBytes(picker *codepick.Picker, w string) int {
	n := 0
	if f := work.InRepo(strings.TrimSpace(work.LineOff(w)), picker.Files()); f != "" {
		for _, l := range picker.Lines(f) {
			n += len(l) + 1
		}
	}
	return n
}

// readyOpen says whether the agent is told an issue may be ready: its four
// sections there, Need and Validation a person's, not ready yet, and ready
// not off — the engine checks it, and proposes an outsider's (ADR-0038).
func readyOpen(is forge.Issue, st *backlog.State, cfg backlog.Config) bool {
	if s, ok := cfg.Acts["ready"]; !ok || s.Mode == backlog.Off || backlog.Readied(is) || slices.Contains(is.Labels, backlog.LabelBlocked) {
		return false
	}
	return len(backlog.NotReady(is, st, false)) == 0
}

// blockWhy reads why a role blocked an issue, from the last comment
// saying it blocks it (backlog.Said's way); "" when none says.
// unrefinedRead says whether the line woke the product owner on a
// blocked issue it had blocked from refine on that role's own read, left
// lacking (engine.RevisitUnrefinedInput).
func unrefinedRead(runDir string) bool {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", engine.RevisitUnrefinedInput))
	return strings.TrimSpace(string(data)) == "true"
}

var blockWhy = regexp.MustCompile(`(?m)^- \*\*Why\*\*: (.+)$`)

// blockedSaid tells the agent, on a blocked issue the line woke it on,
// why it was blocked, what is new, and what it may decide (ADR-0059,
// amended): back to the state it came from or to refine, or kept
// blocked, its reason said afresh.
func blockedSaid(runDir string, is forge.Issue, comments []string, splitOff bool, spent func() string) string {
	input := func(name string) string {
		data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
		return strings.TrimSpace(string(data))
	}
	why := "a role blocked it; its comments say why"
	for _, c := range slices.Backward(comments) {
		if !strings.Contains(c, "blocks this issue") {
			continue
		}
		// The line's own comment on a block it did not make points to
		// the one that says why: that one is read.
		if m := blockWhy.FindStringSubmatch(c); m != nil && !strings.HasPrefix(m[1], backlog.BlockedElsewhere) {
			why = strings.TrimSuffix(strings.TrimSpace(m[1]), ".")
			break
		}
	}
	news := input(engine.RevisitInput)
	if news == "" {
		news = "the line woke you on it"
	}
	from := input(engine.RevisitFromInput)
	to := backlog.UnblockTo(from)
	where := "`to: refine`"
	if len(to) > 1 {
		where = fmt.Sprintf("`to: %s`, the state it was blocked from, or `to: refine` when its need must be read again", to[1])
	}
	if role := input(engine.SizeInput); role != "" {
		// A run too big for its role (ADR-0064, step 3): the same input
		// crosses the same, so the Scope is the product owner's to narrow,
		// or the issue a person's; at the normal size, granted large when
		// its need is atomic (step 4). Told the facts it judges on: what
		// crossed, with the run's numbers, and what the issue spent so far
		// against its normal (#1193, round 6's RC3).
		grant := fmt.Sprintf("; or, its need atomic — no smaller %s nor part of it passes its own scenarios apart — `size` with `tier: %s` — every size target of its runs ×%d — then `unblock` it the same way (where the modes below say `size: propose`, it is only proposed: then `blocked`, `person: true`)",
			work.Name(work.Scope), backlog.TierLarge, backlog.TierFactor(backlog.TierLarge))
		tier := input(engine.SizeTierInput)
		if tier != "" {
			grant = fmt.Sprintf(" (its size tier is %s already, `%s`: a tier only goes up, and past %s it is a person's, so no `size`)", tier, backlog.TierLabel(tier), backlog.TierLarge)
		}
		if role == engine.SizePredicted {
			// Predicted past its normal at refine, a bug readied for its
			// tests first (#1212): written now, its split is asked.
			split := "split it" + strings.TrimPrefix(splitSaid(is, splitOff), "; or split it")
			if noSplit(is, splitOff, "") != "" {
				split = "no split" + splitSaid(is, splitOff)
			}
			return fmt.Sprintf("Blocked (%s), on its size: you predicted it past its normal at refine, and readied it for its tests first; the tester wrote them, on its Draft, so it is split or granted now, before the developer takes it. Read again because %s. Spent on it so far: %s. Decide one, its `why` one plain line: %s%s; or, its tests showing it within its normal after all, `unblock` it, %s; or `blocked`, `person: true`: a person decides. Never `ready` on it.\n",
				backlog.LabelBlocked, news, spent(), split, strings.Replace(grant, "the same way", where, 1), where)
		}
		rule := ""
		if tier == "" && noSplit(is, splitOff, "") == "" {
			// Both offered: which one, said once (ADR-0064, §3).
			rule = fmt.Sprintf(" Split or grant: split when its parts each pass their own scenarios apart; when none can — one cause, a change no half of proves — grant it, never split it. Its %s or %s saying its parts fail apart is evidence to weigh, not the test.",
				work.Name(work.Need), work.Name(work.Scope))
		}
		return fmt.Sprintf("Blocked (%s), on its size. What crossed: %s. Read again because %s. Spent on it so far: %s. A crossing measures one run of the %s, not the need: a run again on the same input crosses the same, and sent back as it is, the issue waits for a person. Decide one, its `why` one plain line: narrow its %s — `refine` with `scope`, the %[6]s rewritten whole to the fewer files its need needs, and `sources` — then `unblock` it, %[7]s%[8]s%[10]s; or `blocked`, why its need cannot be narrowed, `person: true`: a person splits it, grants it `%[9]s`, or raises the setting the line names.%[11]s Never `ready` on it.\n",
			backlog.LabelBlocked, why, news, spent(), strings.ReplaceAll(role, "-", " "), work.Name(work.Scope), where, grant, backlog.LabelSizeXLarge, splitSaid(is, splitOff), rule)
	}
	alone := ""
	if unrefinedRead(runDir) {
		// Blocked on its own read in refine: a rewrite lifts it (#1206).
		alone = " A `refine` with neither beside it sends it back to refine, read again as rewritten."
	}
	if files := backlog.ScopeAsked(comments); len(files) > 0 && slices.Contains(to, "ready") {
		// The developer asks for files outside the Scope (#921): its
		// Scope is the product owner's to widen, or to keep as it is.
		return fmt.Sprintf("Blocked (%s): %s. Read again because %s. The developer's pull request needs `%s`, outside the issue's %s; its comment says why. Decide one, its `why` one plain line: to widen it, `refine` with `scope`, the %[5]s rewritten whole with these files in it, and `sources`, then `unblock` it `to: ready`: the developer resumes on its pull request with the new bounds; or `blocked`, why the %[5]s must not take them — another issue's need, a file no developer changes — `person: true` when only a person can decide. Never `ready` on it.\n",
			backlog.LabelBlocked, why, news, strings.Join(files, "`, `"), work.Name(work.Scope))
	}
	return fmt.Sprintf("Blocked (%s): %s. Read again because %s. Decide one, its `why` one plain line: `unblock` it, %s, when what blocked it is gone; `blocked`, the reason it stays as things are now, `person: true` when only a person can lift it — a decision, an access, a fix outside this repository; or a `close` by the rules above.%s Never `ready` on it.\n",
		backlog.LabelBlocked, why, news, where, alone)
}

// keptMissing are the new test files the developer kept beside the
// issue's Scope (ADR-0063, §9) that its Scope does not name yet.
func keptMissing(is forge.Issue, comments []string) []string {
	scope := work.ByMeaning(is.Body)[work.Scope].Text
	return slices.DeleteFunc(backlog.ScopeKept(comments), func(f string) bool { return strings.Contains(scope, "`"+f+"`") })
}

// toTests says whether the agent is told a ready issue from before the
// line of states may go to its tests (#788): untested, as the tester
// reads it, and ready not off.
func toTests(is forge.Issue, comments []string, cfg backlog.Config, drafts []string) bool {
	if s, ok := cfg.Acts["ready"]; !ok || s.Mode == backlog.Off {
		return false
	}
	return tester.Untested(is, comments, drafts)
}

// lift says whether the agent is told an issue's text on top is a role's
// to take into the Need it drafts: no person edited it, and the issue has
// no Need yet (ADR-0039, amended).
func lift(is forge.Issue, st *backlog.State) bool {
	ok, _ := backlog.RolesTop(is, st)
	return ok && strings.TrimSpace(work.ByMeaning(is.Body)[work.Need].Text) == ""
}

// releases says where the project stands: its last release, and the
// milestones open, for the issues to be put in.
func releases(repo string, b forge.Backlog) string {
	last := "none yet"
	if out, err := exec.Command("git", "-C", repo, "describe", "--tags", "--abbrev=0").Output(); err == nil {
		last = strings.TrimSpace(string(out))
	}
	open := "none"
	if ms, err := b.Milestones(); err == nil && len(ms) > 0 {
		// In the backlog's order: by due date, then by title.
		sort.SliceStable(ms, func(i, j int) bool { return backlog.CompareDue(ms[i], ms[j]) < 0 })
		var said []string
		for _, m := range ms {
			if m.Due != "" {
				said = append(said, fmt.Sprintf("%s (due %s)", m.Title, m.Due))
			} else {
				said = append(said, m.Title)
			}
		}
		open = strings.Join(said, ", ")
	}
	return fmt.Sprintf("Last release: %s. Open milestones: %s.\n", last, open)
}

// sourcesChanged says whether a commit since the issue was last read
// touched one of its sources; an issue with none named is not read again.
func sourcesChanged(repo string, st *backlog.State) bool {
	var paths []string
	for _, s := range st.Sources {
		p, _, _ := strings.Cut(s, "#")
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		return false
	}
	out, err := exec.Command("git", append([]string{"-C", repo, "log", "-1", "--format=%h", st.Judged + "..HEAD", "--"}, paths...)...).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + " […]"
}

// Post passes: each act is checked when the engine applies it, against the
// code and the forge as they are then.
func Post(runDir, repo string) int {
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		v = &verdict.Verdict{}
	}
	v.Status, v.Summary = verdict.Pass, "the acts are checked as they are applied"
	return write(runDir, *v)
}

func writeFindings(runDir string, findings []verdict.Finding) int {
	if len(findings) == 0 {
		return 0
	}
	return write(runDir, verdict.Verdict{Status: verdict.Pass, Findings: findings})
}

func final(runDir string, v verdict.Verdict) int {
	if code := write(runDir, v); code != 0 {
		return code
	}
	return exitNothing
}

func write(runDir string, v verdict.Verdict) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return 0
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "product-owner:", err)
	return 1
}

// preImport gives the agent a share of a file — lines from to to, as the
// command importing it cut them — to tell its items still to do, each to
// open as an issue, its text quoted (docs/spec/backlog-acts.md, "Importing
// a file"). The file's format is the agent's to read, not pre's.
func preImport(runDir, repo, role string, b forge.Backlog, open []forge.Issue) int {
	input := func(name string) string {
		data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
		return strings.TrimSpace(string(data))
	}
	file := input("file")
	var from, to int
	fmt.Sscan(input("from"), &from)
	fmt.Sscan(input("to"), &to)
	if file == "" || from < 1 || to < from {
		return fail(fmt.Errorf("an import names a file and its lines: --input file=<path> from=<n> to=<n>"))
	}
	data, err := exec.Command("git", "-C", repo, "show", "HEAD:"+file).Output()
	if err != nil {
		return fail(fmt.Errorf("%s is not in the commit the run is on", file))
	}
	lines := strings.Split(string(data), "\n")
	if from > len(lines) {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "nothing left to read in " + file})
	}
	to = min(to, len(lines))
	// The closed issues it opened from this share: an item one holds is not
	// opened again, and the plan says so (ADR-0018, amended).
	all, err := b.AllIssues()
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	closed := backlog.ClosedImports(all, file, lines, from, to)
	var t strings.Builder
	fmt.Fprintf(&t, "Import: lines %d to %d of `%s`, of %d. Each item still to do becomes an issue.\n\n", from, to, file, len(lines))
	var titles []string
	for _, is := range open {
		if is.Title != backlog.ReportTitle(role) {
			titles = append(titles, fmt.Sprintf("- #%d %s", is.ID, is.Title))
		}
	}
	if len(titles) > 0 {
		fmt.Fprintf(&t, "# The open issues, titles only\n\nAn item one of them already holds is not opened again.\n\n%s\n\n", strings.Join(titles, "\n"))
	}
	if len(closed) > 0 {
		var held []string
		for _, c := range closed {
			why := "closed"
			if c.Issue.Reason != "" {
				why += " as " + strings.ReplaceAll(c.Issue.Reason, "_", " ")
			}
			at := "lines " + c.Lines
			if c.From == c.To {
				at = "line " + c.Lines
			}
			held = append(held, fmt.Sprintf("- #%d %s — %s; %s", c.Issue.ID, c.Issue.Title, why, at))
		}
		fmt.Fprintf(&t, "# The closed issues opened from these lines\n\nAn item one of them holds is not opened again: answer its lines with a `skip`, `held`, naming the issue. Reopening one is a person's.\n\n%s\n\n", strings.Join(held, "\n"))
	}
	fmt.Fprintf(&t, "# `%s`, lines %d to %d\n\n```\n", file, from, to)
	for i := from; i <= to; i++ {
		fmt.Fprintf(&t, "%5d  %s\n", i, lines[i-1])
	}
	t.WriteString("```\n")
	t.WriteString(elsewhere(lines, from, to))
	if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("import\n"), 0o644); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(t.String()), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

const (
	mentionsPerID = 4   // lines given for one id, those of a table or a heading first
	mentionsIDs   = 40  // ids looked for in one share
	mentionChars  = 240 // characters of a line given
)

var itemID = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:-[A-Z][A-Z0-9]*)*-[0-9]+\b`)

// elsewhere gives the lines outside a share that name an id the share
// holds: a file often says an item is done far from the item itself — a
// table of what shipped, a summary — and a share alone would open it again.
// Which line says what is the agent's to read; pre only finds them.
func elsewhere(lines []string, from, to int) string {
	said := mentions(lines, from, to, from, to)
	if said == "" {
		return ""
	}
	return "\n# What the rest of the file says of this share's items\n\nThe lines outside the share that name an id the share holds. An item they say is done, merged or dropped is not opened.\n\n" + said
}

// mentions lists, for each id lines from to to hold, the lines of the
// file outside out to outTo that name it, those of a table or a heading
// first.
func mentions(lines []string, from, to, out, outTo int) string {
	var ids []string
	for _, l := range lines[from-1 : to] {
		for _, id := range itemID.FindAllString(l, -1) {
			if !slices.Contains(ids, id) && len(ids) < mentionsIDs {
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	var b strings.Builder
	for _, id := range ids {
		named := regexp.MustCompile(`(^|[^A-Za-z0-9-])` + regexp.QuoteMeta(id) + `($|[^0-9])`)
		var structure, prose []int
		for i, l := range lines {
			if i+1 >= out && i+1 <= outTo || !named.MatchString(l) {
				continue
			}
			if t := strings.TrimSpace(l); strings.HasPrefix(t, "|") || strings.HasPrefix(t, "#") {
				structure = append(structure, i)
			} else {
				prose = append(prose, i)
			}
		}
		at := append(structure, prose...)
		if len(at) == 0 {
			continue
		}
		more := ""
		if len(at) > mentionsPerID {
			more = fmt.Sprintf("    (%d more lines name it)\n", len(at)-mentionsPerID)
			at = at[:mentionsPerID]
		}
		sort.Ints(at)
		fmt.Fprintf(&b, "- %s\n", id)
		for _, i := range at {
			fmt.Fprintf(&b, "%6d  %s\n", i+1, clip(lines[i], mentionChars))
		}
		b.WriteString(more)
	}
	return b.String()
}

const (
	aroundLines = 20 // lines given each side of an imported item, its section's at most
	fromFileMax = 60 // lines of the file it was imported from given an issue
)

// writtenAt gives an issue imported from a file the lines around its item
// in that file as it reads at the commit — its section, from its heading —
// and the lines elsewhere in it that name the item's id: where it was
// written, facts to refine from rather than ask its author (#548). The
// file is never code the issue names (named): a file the commit does not
// hold, or the project archived, gives nothing; nor do lines gone since.
// aboutIt is the lines elsewhere that name the item's id.
func writtenAt(repo string, is forge.Issue, st *backlog.State, tracked map[string]bool, archived []string) (text, aboutIt string) {
	path, from, to, ok := backlog.Imported(is.Body)
	if !ok || !tracked[path] || pathglob.Any(archived, path) {
		return "", ""
	}
	data, err := exec.Command("git", "-C", repo, "show", "HEAD:"+path).Output()
	if err != nil {
		return "", ""
	}
	if st.Confirmed != "" { // the lines it was opened from, followed to this commit
		_, now, nfrom, nto, changed, err := backlog.LinesChange(repo, path, from, to, st.Confirmed, st.Confirmed, "HEAD")
		if err != nil || changed && now == "" {
			return "", ""
		}
		from, to = nfrom, nto
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if from < 1 || to < from || to > len(lines) {
		return "", ""
	}
	heading := func(i int) bool { return strings.HasPrefix(strings.TrimSpace(lines[i-1]), "#") }
	start := from
	for start > 1 && start > from-aroundLines && !heading(start) {
		start--
	}
	end := to
	for end < len(lines) && end < to+aroundLines && !heading(end+1) {
		end++
	}
	end = min(end, start+fromFileMax-1)
	var b strings.Builder
	fmt.Fprintf(&b, "Imported from `%s`: its item is at lines %d to %d as the file reads at this commit. Around it, and where the file names its id, are facts to refine from — never asked of its reporter:\n\n```\n", path, from, to)
	for i := start; i <= end; i++ {
		fmt.Fprintln(&b, strings.TrimRight(fmt.Sprintf("%6d  %s", i, clip(lines[i-1], mentionChars)), " "))
	}
	if more := to - end; more > 0 {
		fmt.Fprintf(&b, "(%d more lines of the item left out)\n", more)
	}
	b.WriteString("```\n")
	aboutIt = mentions(lines, from, to, start, end)
	if aboutIt != "" {
		b.WriteString("\nIts id elsewhere in the file:\n\n" + aboutIt)
	}
	return b.String(), aboutIt
}

// specFindings tells the agent the reviewer's findings open on an issue's
// spec (#128): what holds it from ready, and how it answers them — the
// sections it may rewrite, the builder's part always (ADR-0060); a
// person's Need or Validation asked of its reporter.
func specFindings(is forge.Issue, st *backlog.State, comments []string, r *backlog.SpecReview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "The reviewer's findings on its spec, round %d: %d important open, in %s. It stays out of ready until none is open. ", r.Round, r.Open, strings.Join(r.In, ", "))
	mine := backlog.Revisable(is, st, r)
	if len(mine) > 0 {
		fmt.Fprintf(&b, "Answer them with a `refine` giving the whole new text of the sections they lie in that are yours to rewrite: %s. ", strings.Join(mine, ", "))
	}
	theirs := slices.DeleteFunc(slices.Clone(r.In), func(n string) bool { return work.Has(mine, work.MeaningOf(n)) })
	testers := slices.DeleteFunc(slices.Clone(theirs), func(n string) bool { return !work.Testers(work.MeaningOf(n)) })
	theirs = slices.DeleteFunc(theirs, func(n string) bool { return work.Testers(work.MeaningOf(n)) })
	if len(theirs) > 0 {
		fmt.Fprintf(&b, "%s: a person's, never rewritten: `ask` its reporter. ", strings.Join(theirs, ", "))
	}
	if len(testers) > 0 {
		// The tester writes the test plan (ADR-0066): its finding is the
		// tester's, read when it writes the tests.
		fmt.Fprintf(&b, "%s: the tester's, answered as it writes the tests: leave it, ask no one. ", strings.Join(testers, ", "))
	}
	b.WriteString("The reviewer reads the spec again after your answer.\n")
	for i := len(comments) - 1; i >= 0; i-- {
		if strings.Contains(comments[i], backlog.SpecMarker) {
			fmt.Fprintf(&b, "\n> %s\n", strings.ReplaceAll(clip(marker.ReplaceAllString(comments[i], ""), bodyMax), "\n", "\n> "))
			break
		}
	}
	return b.String()
}
