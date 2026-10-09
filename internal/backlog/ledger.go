package backlog

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
)

// What the role holds is kept on each issue, never in a report (ADR-0038):
// the acts it proposes there, waiting on a person's yes; the acts it did
// alone, a person may undo; what it did, each with its day — all in the
// issue's own state comment, its one comment, edited in place. An issue
// waiting on a person bears the label workline:proposed (GitLab:
// workline::proposed): a saved filter on it is what waits on a person.
// What the role knows of the whole backlog is read from these, each run;
// none of it is kept anywhere else.

// The labels of a proposal (ADR-0038): the role's, on an issue waiting on
// a person; and a person's yes, read in either spelling.
const (
	LabelProposed  = "workline:proposed"
	scopedProposed = "workline::proposed"
	scopedAccepted = "workline::accepted"
)

// ProposedLabel is the label the role puts on an issue waiting on a person,
// as the forge names it: scoped on GitLab.
func ProposedLabel(f any) string {
	if forge.ScopedLabels(f) {
		return scopedProposed
	}
	return LabelProposed
}

// AcceptedLabel is the label a person agrees with, as the role creates it
// on the forge: scoped on GitLab, so it replaces workline::proposed where
// the plan has scoped labels exclusive. Either spelling is a yes.
func AcceptedLabel(f any) string {
	if forge.ScopedLabels(f) {
		return scopedAccepted
	}
	return LabelAccepted
}

// acceptedLabels are a person's yes, in both spellings.
var acceptedLabels = []string{LabelAccepted, scopedAccepted}

// Proposed says whether an issue bears the role's label, either spelling.
func Proposed(is forge.Issue) bool {
	return slices.Contains(is.Labels, LabelProposed) || slices.Contains(is.Labels, scopedProposed)
}

// UndoneMax is how many acts of a kind done alone a person undoes, across
// the issues, before that kind is proposed on every issue; one undone is
// proposed again on its own issue only (ADR-0038, amended).
const (
	UndoneMax      = 3
	UndoneMaxLimit = 20
)

// ProposalsMax is how many issues may wait on a person at once before the
// role slows down: past it, a run reads only the issues a person answered
// on (ADR-0038: silence is never a yes).
const (
	ProposalsMax      = 10
	ProposalsMaxLimit = 100
)

// plusOne is a comment that says nothing but agreement or a vote: never an
// answer to read.
var plusOne = regexp.MustCompile(`^(?:[+-]1|:[+-]1:|:thumbsup:|:thumbsdown:|👍|👎|\s)+$`)

// Counted are an issue's comments that count (ADR-0038): its reporter's
// and the project's people's — never a bot's, an outsider's, an author
// the forge does not name, the engine's own, nor a bare "+1".
func Counted(notes []forge.Note, is forge.Issue) []forge.Note {
	var out []forge.Note
	for _, n := range notes {
		if _, engine := EngineMarker(n.Body); engine || strings.Contains(n.Body, "<!-- workline:") {
			continue
		}
		if n.Bot || n.Author == "" || (n.Author != is.Author && !n.Insider) || plusOne.MatchString(strings.TrimSpace(n.Body)) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// Heard is how many counted comments the state says the role read; a
// state written before it counted them (no `heard`) counted every
// comment not the engine's (`comments`): those it holds are taken as
// read, never more than there are now.
func Heard(st *State, counted int) int {
	switch {
	case st == nil:
		return counted
	case st.Heard == nil:
		return min(st.Comments, counted)
	}
	return *st.Heard
}

// ID is a pending proposal's key: its issue and kind, or the text an
// issue to open would be opened from.
func (q Pending) ID() string {
	if q.Key != "" {
		return q.Key
	}
	return fmt.Sprintf("%d/%s", q.Issue, q.Act)
}

// Undrafted says whether a proposal on an issue was recorded with its line
// alone, the act not kept — by an engine before acts were recorded: its
// issue is read again, and the act drafted then.
func (q Pending) Undrafted() bool { return q.Proposal == nil && q.Key == "" && q.Issue > 0 }

// Doable says whether the engine can do a proposal a person accepted: what
// it would do recorded, and not one only a person or the import does.
func (q Pending) Doable() bool {
	c := q.Proposal
	return c != nil && c.Do != "open" && !c.Spent && !c.Advice && !(c.Do == "milestone" && c.Milestone == "")
}

// Answer is a person's comment on an issue waiting on them, read as
// "revise" (ADR-0038): the issue read again with it, 👀 put on each, the
// drafts rewritten in place, one line in reply.
type Answer struct {
	Notes    []string `yaml:"notes,flow"` // the comments read, by the forge's id
	Who      string   `yaml:"who"`        // the last one's author
	Revision int      `yaml:"revision"`   // this revision's number, from 1
}

// Ledger is what the role holds, read from each open issue's state: the
// record it keeps there, as one; and, the first run after a report
// issue, that report's record, moved to the issues (ADR-0038).
type Ledger struct {
	Report int    // an open report issue of an earlier engine: its record read once, then it is closed
	Record Record // the issues' records, as one
	Broken error  // the old report's record does not read
	States map[int]*State
	Notes  map[int][]forge.Note
	// Aside are the issues a person took workline:proposed off since the
	// last run, without accepting: "not now", that issue only, until it
	// changes.
	Aside map[int]bool
	// Undone are the role's acts a person undid since the last run, other
	// than closings; Standing, those still watched (ADR-0026).
	Undone   []Undo
	Standing []Done
	// Own are the blockers the role set on each open issue: a depend's, a
	// split's after (ADR-0028).
	Own map[int][]int
	// Signs are what a person did that a run with no agent still writes:
	// a yes, an answer, a label taken off, an act undone, the report to
	// move.
	Signs []string
}

// Demoted are the kinds of act proposed on every issue: those a person
// undid at least max times, across the issues (UndoneMax).
func (l *Ledger) Demoted(max int) []string {
	return demoted(append(slices.Clone(l.Record.Undone), l.Undone...), l.Record.Wrong, max)
}

// demoted counts the acts undone, closings reopened among them, by kind:
// a kind undone max times or more is proposed everywhere.
func demoted(undone []Undo, wrong []Closing, max int) []string {
	if max <= 0 {
		max = UndoneMax
	}
	count := map[string]int{}
	for _, u := range undone {
		count[u.Act]++
	}
	for _, c := range wrong {
		count[c.Act]++
	}
	var out []string
	for _, kind := range ActKinds {
		if count[kind] >= max {
			out = append(out, kind)
		}
	}
	return out
}

// UndoneOn are the kinds of act a person undid on an issue: proposed
// there from then on, whatever the level (ADR-0038, amended).
func (l *Ledger) UndoneOn(id int) []string {
	var out []string
	for _, u := range append(slices.Clone(l.Record.Undone), l.Undone...) {
		if u.Issue == id && !slices.Contains(out, u.Act) {
			out = append(out, u.Act)
		}
	}
	for _, c := range l.Record.Wrong {
		if c.Issue == id && !slices.Contains(out, c.Act) {
			out = append(out, c.Act)
		}
	}
	return out
}

// Waiting are the open issues that wait on a person: the role's label on
// them.
func (l *Ledger) Waiting(open []forge.Issue) int {
	n := 0
	for _, is := range open {
		if st := l.States[is.ID]; st != nil && st.Label != "" && !l.Aside[is.ID] {
			n++
		}
	}
	return n
}

// ReadLedger reads what the role holds on each open issue, from its state
// comment; the record of an earlier engine's report, the first time; the
// acts a person undid since.
func ReadLedger(f forge.Backlog, role string, open []forge.Issue) (*Ledger, error) {
	l := &Ledger{States: map[int]*State{}, Notes: map[int][]forge.Note{}, Aside: map[int]bool{}}
	byID := map[int]forge.Issue{}
	for _, is := range open {
		if is.Title == ReportTitle(role) {
			l.Report = is.ID
			continue
		}
		byID[is.ID] = is
		notes, err := f.Notes(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil, err
		}
		l.Notes[is.ID] = notes
		st, found, err := ReadState(forge.Bodies(notes), role)
		if !found || err != nil {
			continue
		}
		l.States[is.ID] = st
		if aside, err := setAside(f, is, st); err != nil {
			return nil, err
		} else if aside {
			l.Aside[is.ID] = true
			l.Signs = append(l.Signs, fmt.Sprintf("#%d set aside", is.ID))
		}
		l.add(is.ID, st)
		if Accepted(is) && (len(st.Proposed) > 0 || st.Label != "") {
			l.Signs = append(l.Signs, fmt.Sprintf("#%d accepted", is.ID))
		}
		if st.Closed != "" {
			l.Signs = append(l.Signs, fmt.Sprintf("#%d reopened", is.ID))
		}
		if st.Label != "" && Heard(st, len(Counted(notes, is))) < len(Counted(notes, is)) {
			l.Signs = append(l.Signs, fmt.Sprintf("#%d answered", is.ID))
		}
	}
	if l.Report != 0 {
		if err := l.readReport(f, role, byID); err != nil {
			return nil, err
		}
	}
	var err error
	if l.Standing, l.Undone, l.Own, err = findUndone(f, role, byID, l.Record.Done); err != nil {
		return nil, err
	}
	for _, u := range l.Undone {
		l.Signs = append(l.Signs, fmt.Sprintf("#%d: %s undone", u.Issue, u.Act))
	}
	slices.Sort(l.Signs)
	l.Signs = slices.Compact(l.Signs)
	return l, nil
}

// setAside says whether a person took the role's label off an issue since
// the last run without accepting: "not now" (ADR-0038). A label a bot took
// off, as the forge says, is not a person's answer.
func setAside(f forge.Backlog, is forge.Issue, st *State) (bool, error) {
	if st.Label == "" || slices.Contains(is.Labels, st.Label) || Accepted(is) {
		return false, nil
	}
	events, err := f.LabelEvents(is.ID)
	if err != nil {
		return false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		if e := events[i]; e.Label == st.Label && !e.Added {
			return !e.Bot, nil
		}
	}
	return true, nil
}

// add puts an issue's record in the ledger's, each entry with its issue; a
// proposal set aside is a person's "not now": dropped.
func (l *Ledger) add(id int, st *State) {
	r := &l.Record
	if !l.Aside[id] {
		for _, q := range st.Proposed {
			q.Issue = id
			r.Proposed = append(r.Proposed, q)
		}
	}
	for _, d := range st.Done {
		d.Issue = id
		r.Done = append(r.Done, d)
	}
	for _, u := range st.Undone {
		u.Issue = id
		r.Undone = append(r.Undone, u)
	}
	for _, d := range st.Did {
		d.Issue = id
		r.Did = append(r.Did, d)
	}
	if st.Closed != "" {
		r.Closed = append(r.Closed, Closing{Issue: id, Act: st.Closed})
	}
}

// readReport reads the record an earlier engine kept on its report issue,
// once: each issue's part moves to that issue's state at the end of the
// run, and the report is closed (ADR-0038). Its ticks, its pause, its
// measure are not read: they lived on the report alone.
func (l *Ledger) readReport(f forge.Backlog, role string, open map[int]forge.Issue) error {
	notes, err := f.Notes(forge.Target{Kind: "issue", ID: l.Report})
	if err != nil {
		return err
	}
	var old Record
	for i := len(notes) - 1; i >= 0; i-- {
		if !strings.Contains(notes[i].Body, RecordMarker(role)) {
			continue
		}
		text, ok := yamlBlock(notes[i].Body)
		if !ok {
			l.Broken = fmt.Errorf("no YAML block")
		} else if err := yaml.Unmarshal([]byte(text), &old); err != nil {
			l.Broken = err
		}
		break
	}
	if l.Broken != nil {
		return nil
	}
	l.Signs = append(l.Signs, fmt.Sprintf("#%d, the old report, to close", l.Report))
	r := &l.Record
	held := func(id int) bool { // the issue's state holds a record already: the report's is older
		st := l.States[id]
		return st != nil && (len(st.Proposed) > 0 || len(st.Done) > 0 || len(st.Did) > 0 || len(st.Undone) > 0 || st.Closed != "")
	}
	for _, q := range old.Proposed {
		if q.Issue > 0 && !held(q.Issue) && !l.Aside[q.Issue] {
			r.Proposed = append(r.Proposed, q)
		}
	}
	for _, d := range old.Done {
		if !held(d.Issue) {
			r.Done = append(r.Done, d)
		}
	}
	for _, u := range old.Undone {
		if !held(u.Issue) {
			r.Undone = append(r.Undone, u)
		}
	}
	for _, c := range old.Wrong { // a closing found wrong: undone, on its issue
		if !held(c.Issue) {
			r.Undone = append(r.Undone, Undo{Issue: c.Issue, Act: c.Act, Evidence: fmt.Sprintf("#%d reopened", c.Issue)})
		}
	}
	for _, d := range old.Did {
		if !held(d.Issue) {
			r.Did = append(r.Did, d)
		}
	}
	for _, c := range old.Closed {
		if !held(c.Issue) {
			r.Closed = append(r.Closed, c)
		}
	}
	return nil
}

// stateOf is what the plan holds for one issue, in its state's fields.
func (p *Plan) stateOf(id int, st *State) {
	r := p.Record
	st.Proposed, st.Done, st.Undone, st.Did = nil, nil, nil, nil
	for _, q := range r.Proposed {
		if q.Issue == id && q.Key == "" {
			q.Issue = 0
			st.Proposed = append(st.Proposed, q)
		}
	}
	for _, d := range r.Done {
		if d.Issue == id {
			d.Issue = 0
			st.Done = append(st.Done, d)
		}
	}
	for _, u := range r.Undone {
		if u.Issue == id {
			u.Issue = 0
			st.Undone = append(st.Undone, u)
		}
	}
	for _, c := range r.Wrong {
		if c.Issue == id {
			st.Undone = append(st.Undone, Undo{Act: c.Act, Evidence: fmt.Sprintf("#%d reopened", id), Day: today()})
		}
	}
	for _, d := range r.Did {
		if d.Issue == id {
			d.Issue = 0
			st.Did = append(st.Did, d)
		}
	}
	st.Closed = ""
	for _, c := range r.Closed {
		if c.Issue == id {
			st.Closed = c.Act
		}
	}
}

func today() string { return time.Now().UTC().Format(dateLayout) }

// Persist writes what the run leaves on each issue (ADR-0038): its record
// in its state comment, edited in place; the label workline:proposed on an
// issue waiting on a person, off one that no longer waits, off one a
// person accepted; 👀 on a person's comments read, and one line in reply.
// The old report, read this run, is closed with a link to the filter.
// Each write is idempotent: a resumed run writes the same.
func (p *Plan) Persist(f forge.Forge, role string) error {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil
	}
	ids := slices.Clone(p.Visit)
	for _, q := range p.Record.Proposed {
		ids = append(ids, q.Issue)
	}
	for _, c := range p.Record.Closed {
		ids = append(ids, c.Issue)
	}
	for _, d := range p.Record.Did {
		ids = append(ids, d.Issue)
	}
	for _, d := range p.Decisions {
		if d.Mode != Off && d.Act.Issue > 0 {
			ids = append(ids, d.Act.Issue)
		}
	}
	for id := range p.Answers {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	proposed := ProposedLabel(f)
	ensured := false
	for _, id := range ids {
		if id <= 0 || id == p.Report {
			continue
		}
		is, err := f.Issue(id)
		if errors.Is(err, forge.ErrUnreachable) {
			return err
		}
		if err != nil {
			continue // gone from the forge since: deleted, moved
		}
		t := forge.Target{Kind: "issue", ID: id}
		comments, err := b.Comments(t)
		if err != nil {
			return err
		}
		st, found, err := ReadState(comments, role)
		if !found || err != nil {
			continue
		}
		before := FormatState(*st)
		p.stateOf(id, st)
		var add, remove []string
		switch {
		case is.Closed:
			if st.Label != "" && slices.Contains(is.Labels, st.Label) {
				remove = append(remove, st.Label)
			}
			st.Label, st.Proposed = "", nil
		case st.Label != "" && !slices.Contains(is.Labels, st.Label) && !Accepted(*is) && p.SetAside[id]:
			// Taken off by a person: not now, this issue only, until it
			// changes (ADR-0038).
			st.Label, st.Proposed, st.Aside = "", nil, BodyDigest(is.Body)
		case Accepted(*is):
			// Their yes, on every plan: the role takes its label off itself.
			for _, l := range []string{st.Label, LabelProposed, scopedProposed} {
				if l != "" && slices.Contains(is.Labels, l) && !slices.Contains(remove, l) {
					remove = append(remove, l)
				}
			}
			st.Label = ""
		default:
			if st.Aside != "" && (BodyDigest(is.Body) != st.Aside || p.Answers[id] != nil) {
				st.Aside = "" // the issue changed since: proposed again, if anything
			}
			waits := st.Aside == "" && (len(st.Proposed) > 0 || (HasDraft(is.Body, st) && !slices.Contains(is.Labels, LabelReady)))
			switch {
			case waits && st.Label == "":
				if !ensured {
					if err := b.EnsureLabel(proposed, "fbca04", proposedSays(AcceptedLabel(f))); err != nil {
						return err
					}
					ensured = true
				}
				add, st.Label = append(add, proposed), proposed
			case waits && !slices.Contains(is.Labels, st.Label):
				add = append(add, st.Label) // taken off by a bot, or lost: put back
			case !waits && st.Label != "":
				if slices.Contains(is.Labels, st.Label) {
					remove = append(remove, st.Label)
				}
				st.Label = ""
			}
		}
		if !is.Closed && !HasDraft(is.Body, st) {
			// Its labels say what is true of it: no draft of the role's
			// left, no workline:draft; set aside, its body a person's, the
			// role's way to ready is off it too (ADR-0038).
			for _, l := range []string{LabelDraft, LabelToRefine} {
				if slices.Contains(is.Labels, l) && (l == LabelDraft || st.Aside != "") && !slices.Contains(remove, l) {
					remove = append(remove, l)
				}
			}
		}
		if after := FormatState(*st); after != before {
			if err := f.Sticky(t, after, StateMarker(role), false); err != nil {
				return err
			}
		}
		if len(add) > 0 || len(remove) > 0 {
			if err := f.Label(t, add, remove); err != nil {
				return err
			}
		}
	}
	for _, id := range sortedKeys(p.Answers) {
		a := p.Answers[id]
		for _, n := range a.Notes {
			if err := b.React(id, n, forge.Eyes); err != nil {
				return err
			}
		}
		if len(a.Notes) == 0 {
			continue
		}
		if err := f.Comment(forge.Target{Kind: "issue", ID: id}, Inert(p.reply(id, a)), forge.Marker(role+"/reply="+a.Notes[len(a.Notes)-1])); err != nil {
			return err
		}
	}
	if p.Report != 0 && p.Moved {
		where := forge.LabelFilter(f, proposed)
		if where == "" {
			where = "the issues bearing the label `" + proposed + "`"
		} else {
			where = fmt.Sprintf("[the issues bearing `%s`](%s)", proposed, where)
		}
		say := fmt.Sprintf("The %s keeps no report any more: what it proposes is on each issue, under its one comment there, and what waits on you is %s. Each night's summary is in the CI job's summary. What this report recorded moved to each issue.", strings.ReplaceAll(role, "-", " "), where)
		if err := f.Comment(forge.Target{Kind: "issue", ID: p.Report}, say, forge.Marker(role+"/moved")); err != nil {
			return err
		}
		if err := b.Close(p.Report, 0); err != nil {
			return err
		}
	}
	return nil
}

// sortedKeys are a map's issue numbers, lowest first.
func sortedKeys[V any](m map[int]V) []int {
	ids := make([]int, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// reply is the role's one line to a person's comments it read: what it
// revised, or that nothing changed.
func (p *Plan) reply(id int, a *Answer) string {
	var did []string
	for _, d := range p.Decisions {
		if d.Act.Issue != id || d.Mode == Off {
			continue
		}
		switch {
		case d.Mode == Act && d.Act.Do == "refine" && !d.Act.ToReporter:
			did = append(did, "rewrote "+and(d.Act.Added)+" in place")
		case d.Mode == Act:
			did = append(did, strings.TrimSuffix(describe(d.Act, "Closed"), "."))
		case d.Mode == Propose:
			did = append(did, "proposes: "+strings.TrimSuffix(offer(d.Act), "."))
		}
	}
	who := ""
	if a.Who != "" {
		who = "@" + a.Who + ", "
	}
	if len(did) == 0 {
		return fmt.Sprintf("%sread (revision %d of %d): nothing changed; its comment above still says what it proposes.", who, a.Revision, p.revisionsMax())
	}
	return fmt.Sprintf("%sread (revision %d of %d): %s.", who, a.Revision, p.revisionsMax(), strings.Join(did, "; "))
}

// revisionsMax is how many times the role revises its drafts for a
// person's comments: the rounds (acts.ask.rounds) less the first draft.
func (p *Plan) revisionsMax() int { return max(1, RoundsMax(p.settings)-1) }

// RevisionsMax is the same, from the settings.
func RevisionsMax(settings map[string]Setting) int { return max(1, RoundsMax(settings)-1) }

// proposedSays is the description of the label workline:proposed, as the
// forge's list shows it: 100 characters at most, GitHub's limit.
func proposedSays(accepted string) string {
	return "The product owner proposes here: " + accepted + " to agree, a comment to revise, off for not now"
}

// slashLine is a line a GitLab quick action would run: a `/word` at its
// start.
var slashLine = regexp.MustCompile(`(?m)^(\s*)/`)

// Inert is a text the role writes on a forge with every line that starts
// with `/` escaped, outside fenced code: GitLab runs a quick action at a
// line's start when a comment or a description is written, and when it is
// edited (ADR-0038). The escape shows as the `/` it was.
func Inert(text string) string {
	lines := strings.Split(text, "\n")
	fence := 0 // the backticks of the fenced block the line is in; 0 outside
	for i, l := range lines {
		t := strings.TrimSpace(l)
		n := len(t) - len(strings.TrimLeft(t, "`"))
		indent := len(strings.ReplaceAll(l[:len(l)-len(strings.TrimLeft(l, " \t"))], "\t", "    "))
		switch {
		case indent > 3:
			// Indented code, or a line inside a fenced block: never a
			// fence (CommonMark).
		case fence == 0 && n >= 3 && !strings.Contains(t[n:], "`"):
			// A fence opens only with no backtick after it: "```x```" is
			// inline code, its line text like any other.
			fence = n
			continue
		case fence > 0 && n >= fence && strings.Trim(t, "`") == "":
			fence = 0
			continue
		}
		if fence == 0 {
			lines[i] = slashLine.ReplaceAllString(l, `$1\/`)
		}
	}
	return strings.Join(lines, "\n")
}
