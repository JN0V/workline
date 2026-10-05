// Package backlog decides what becomes of a role's acts on a project's
// issues (docs/spec/backlog-acts.md): each is checked against the code and
// the forge, then done, proposed to a person, or dropped. Closing is the
// first act built.
package backlog

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// Modes of an act.
const (
	Act     = "act"
	Propose = "propose"
	Off     = "off"
)

// State is what the engine knows of an issue, kept in one comment on it.
type State struct {
	Sources   []string `yaml:"sources"`
	Confirmed string   `yaml:"confirmed"`
	Judged    string   `yaml:"judged,omitempty"`   // the commit the role last read it at
	Comments  int      `yaml:"comments,omitempty"` // people's comments when it was last read
	Body      string   `yaml:"body,omitempty"`     // a digest of its body, when the role last read or wrote it
	Priority  int      `yaml:"priority,omitempty"` // the priority the role last set: another on the issue is a person's
	Title     string   `yaml:"title,omitempty"`    // the title the role last set: another on the issue is a person's
	Split     []int    `yaml:"split,omitempty"`    // the children the role split it into: it is not split again
}

// StateMarker marks the comment holding an issue's state.
func StateMarker(role string) string { return forge.Marker("sticky=" + role + "/state") }

// RecordMarker marks the report's comment holding what the role did.
func RecordMarker(role string) string { return forge.Marker("sticky=" + role + "/acts") }

// ReportTitle is the title of the role's report issue.
func ReportTitle(role string) string { return "Backlog — " + strings.ReplaceAll(role, "-", " ") }

var fenced = regexp.MustCompile("(?s)```yaml\n(.*?)```")

// readBlock decodes the fenced YAML block of the comment carrying marker
// into v. found is false when no comment carries it; err when it does not
// read.
func readBlock(comments []string, marker string, v any) (found bool, err error) {
	for _, c := range comments {
		if !strings.Contains(c, marker) {
			continue
		}
		m := fenced.FindStringSubmatch(c)
		if m == nil {
			return true, fmt.Errorf("no YAML block")
		}
		dec := yaml.NewDecoder(strings.NewReader(m[1]))
		dec.KnownFields(true)
		return true, dec.Decode(v)
	}
	return false, nil
}

// ReadState reads an issue's state from its comments.
func ReadState(comments []string, role string) (*State, bool, error) {
	var s State
	found, err := readBlock(comments, StateMarker(role), &s)
	if found && err == nil && s.Confirmed == "" {
		err = fmt.Errorf("no commit it was confirmed at")
	}
	return &s, found, err
}

// FormatState is the body of an issue's state comment, its marker left to
// the forge's Sticky.
func FormatState(s State) string {
	data, _ := yaml.Marshal(struct {
		Sources   []string `yaml:"sources,flow"`
		Confirmed string   `yaml:"confirmed"`
		Judged    string   `yaml:"judged,omitempty"`
		Comments  int      `yaml:"comments,omitempty"`
		Body      string   `yaml:"body,omitempty"`
		Priority  int      `yaml:"priority,omitempty"`
		Title     string   `yaml:"title,omitempty"`
		Split     []int    `yaml:"split,flow,omitempty"`
	}{s.Sources, s.Confirmed, s.Judged, s.Comments, s.Body, s.Priority, s.Title, s.Split})
	return "What workline knows of this issue; edited by the engine, not by hand.\n\n```yaml\n" + string(data) + "```"
}

// Record is what the role did, kept on its report issue.
type Record struct {
	Closed   []Closing `yaml:"closed,omitempty"`
	Wrong    []Closing `yaml:"wrong,omitempty"`        // closings found wrong: their issue open again
	Propose  []string  `yaml:"propose,flow,omitempty"` // kinds of act back to propose, until the person says
	Proposed []Pending `yaml:"proposed,omitempty"`     // acts proposed, kept until their issue is closed or proposed again
}

// Pending is an act proposed to a person, as the report says it.
type Pending struct {
	Issue int    `yaml:"issue"`
	Act   string `yaml:"act"`
	Line  string `yaml:"line"`
	Key   string `yaml:"key,omitempty"` // an issue to open: the text it would be opened from (ImportKey)
	// Capped: proposed only because the run's cap was reached, not left to
	// a person by its mode: its issue is read again at the next run.
	Capped bool `yaml:"capped,omitempty"`
}

// Closing is one issue the role closed.
type Closing struct {
	Issue int    `yaml:"issue"`
	Act   string `yaml:"act"`
}

// Quote is the evidence an act cites: a text in a file, or in an issue.
type Quote struct {
	Path  string `yaml:"path"`
	Issue int    `yaml:"issue"`
	Text  string `yaml:"text"`
}

// Proposal is an act on an issue the agent proposed: a closing, or the code it
// is about named as its sources.
type Proposal struct {
	Do          string   `yaml:"do"` // close or sources: the intention's kind
	Issue       int      `yaml:"issue"`
	Reason      string   `yaml:"reason,omitempty"`
	DuplicateOf int      `yaml:"duplicate-of,omitempty"`
	Sources     []string `yaml:"sources,omitempty"`
	Milestone   string   `yaml:"milestone,omitempty"`
	Title       string   `yaml:"title,omitempty"` // an issue to open, or an issue's new title (rename)
	Into        []Child  `yaml:"into,omitempty"`  // the children of a split
	Quote       *Quote   `yaml:"quote"`
	Why         string   `yaml:"why"`
	// Refining: the sections written, Need and Validation as drafts; Added,
	// the engine's, says which the body did not have yet.
	Scope        string   `yaml:"scope,omitempty"`
	Verification string   `yaml:"verification,omitempty"`
	Need         string   `yaml:"need,omitempty"`
	Validation   string   `yaml:"validation,omitempty"`
	Added        []string `yaml:"added,omitempty"`
	Questions    string   `yaml:"questions,omitempty"` // asking the reporter; for an outsider's refine, what it still needs
	// The engine's, for the conversation with the reporter: the round this
	// comment would be, and whether a refine is proposed to the reporter in
	// a comment rather than written in the body (an outsider's issue).
	Round      int  `yaml:"round,omitempty"`
	ToReporter bool `yaml:"to-reporter,omitempty"`
	Spent      bool `yaml:"spent,omitempty"` // the rounds spent: proposed to a person
	// Ordering: the priority set (1 to 4); a milestone left because it is
	// released (From, the engine's); and, the engine's, the issue's
	// priority and milestone before the run, written in the report.
	Priority int    `yaml:"priority,omitempty"`
	From     string `yaml:"from,omitempty"`
	Before   string `yaml:"before,omitempty"`
}

// Child is one part of a split need: an issue of its own, with its four
// sections (ADR-0022).
type Child struct {
	Title        string   `yaml:"title"`
	Need         string   `yaml:"need"`
	Verification string   `yaml:"verification"`
	Validation   string   `yaml:"validation"`
	Scope        string   `yaml:"scope"`
	Sources      []string `yaml:"sources,omitempty"`
}

// Kind is the kind of act, as settings name it.
func (c Proposal) Kind() string {
	if c.Do != "close" {
		return c.Do
	}
	return "close-" + c.Reason
}

// where names the act's issue in a finding: its number, or the title of
// one to open.
func (c Proposal) where() string {
	if c.Do == "open" {
		return c.Title
	}
	return fmt.Sprintf("#%d", c.Issue)
}

// key tells one proposal from another in the report: its issue and kind,
// or for an issue to open, the text it would be opened from.
func (c Proposal) key() string {
	if c.Do == "open" && c.Quote != nil {
		return ImportKey(*c.Quote)
	}
	return fmt.Sprintf("%d/%s", c.Issue, c.Kind())
}

// Kinds are the intentions that are acts on the backlog.
var Kinds = []string{"open", "close", "sources", "milestone", "order", "refine", "ready", "ask", "split", "rename"}

// theirs are the acts an outsider's issue is proposed for, not done: it is
// theirs (ADR-0018).
var theirs = map[string]string{"ready": "moving it to ready", "split": "splitting it", "rename": "renaming it"}

// moves are the acts that move an issue in the backlog's order: capped
// together, at a share of the open issues a run (ADR-0018).
var moves = []string{"milestone", "order"}

// Decision is what becomes of one act.
type Decision struct {
	Index  int      `yaml:"index"` // the intention's place in the run
	Mode   string   `yaml:"mode"`  // act, propose, or off (dropped)
	Act    Proposal `yaml:"act"`
	Capped bool     `yaml:"capped,omitempty"` // proposed only for the cap
}

// Plan is what a run does with its acts, decided once, so a resumed run
// does the same.
type Plan struct {
	Decisions []Decision        `yaml:"decisions"`
	Findings  []verdict.Finding `yaml:"findings"`
	Record    Record            `yaml:"record"`
	Report    int               `yaml:"report"`  // the report issue, 0 when none is open yet
	Changed   bool              `yaml:"changed"` // the record changed: wrong closings found, proposals settled
	open      map[int]bool      // the open issues, read once
	issues    map[int]forge.Issue
	bodies    []string        // their bodies, to find an import again
	seen      map[string]bool // the imports this run decided
	read      []int           // the issues this run read
}

// Setting is a kind of act's mode and cap.
type Setting struct {
	Mode   string
	Max    int
	Rounds int // asking: the times an issue's reporter is written to, then a person (ask only)
}

// Settings reads the role's `acts` setting.
func Settings(settings map[string]any) map[string]Setting {
	out := map[string]Setting{}
	acts, _ := settings["acts"].(map[string]any)
	for kind, v := range acts {
		m, _ := v.(map[string]any)
		s := Setting{Mode: Propose}
		if mode, ok := m["mode"].(string); ok {
			s.Mode = mode
		}
		if n, ok := m["max"].(int); ok {
			s.Max = n
		}
		if n, ok := m["rounds"].(int); ok {
			s.Rounds = n
		}
		out[kind] = s
	}
	return out
}

var closeReasons = []string{"duplicate", "obsolete"}

// maxSources bounds the files an issue names.
const maxSources = 5

// MovedPercent reads the role's `moved-percent-max` setting: the share of
// the open issues a run may move, in percent; a fifth when it is not set.
func MovedPercent(settings map[string]any) int {
	if n, ok := settings["moved-percent-max"].(int); ok && n > 0 {
		return n
	}
	return 20
}

// Decide plans the acts proposed, given at their place in the run.
// read lists the issues the run read: a proposal made only for a cap is
// dropped once its issue was read again, decided again or not. A run
// moves at most movedPercent of the open issues (milestone and order).
func Decide(f forge.Backlog, repo, role string, settings map[string]Setting, closes map[int]Proposal, read []int, movedPercent int) (*Plan, error) {
	p := &Plan{read: read}
	if err := p.readRecord(f, role); err != nil {
		return nil, err
	}
	dropped := func(c Proposal, rule, msg string) {
		p.Findings = append(p.Findings, verdict.Finding{Rule: rule, Where: c.where(), Message: msg})
	}
	done := map[string]int{}
	moved := map[int]bool{} // the issues moved in this run
	backlogSize := 0
	for id := range p.open {
		if id != p.Report {
			backlogSize++
		}
	}
	movedMax := max(1, (backlogSize*movedPercent+99)/100)
	var indexes []int
	for i := range closes {
		indexes = append(indexes, i)
	}
	slices.Sort(indexes)
	decided := map[string]bool{}
	for _, i := range indexes {
		c := closes[i]
		if (c.Do == "ready" || c.Do == "milestone" || c.Do == "order" || c.Do == "split" || c.Do == "rename") && decided[c.key()] {
			p.Decisions = append(p.Decisions, Decision{Index: i, Mode: Off, Act: c}) // the engine's own and the agent's: one
			if c.Do == "split" || c.Do == "rename" {
				// Both checked against the state before either is applied:
				// a second would split or rename it again.
				dropped(c, "once-a-run", c.Do+": one an issue a run; the first that passed its check is kept")
			}
			continue
		}
		once := c.Do == "split" || c.Do == "rename"
		if !once {
			decided[c.key()] = true
		}
		mode, why := p.check(f, repo, role, &c)
		if once && why == "" {
			decided[c.key()] = true // the first that passes its check is kept
		}
		d := Decision{Index: i, Mode: Off, Act: c}
		switch {
		case why != "":
			dropped(c, mode, why)
		default:
			s, ok := settings[c.Kind()]
			if !ok {
				s = Setting{Mode: Propose}
			}
			d.Mode = s.Mode
			if d.Mode == Act && slices.Contains(p.Record.Propose, c.Kind()) {
				d.Mode = Propose
			}
			if what, ok := theirs[c.Do]; ok && d.Mode == Act && !p.issues[c.Issue].Insider && !Accepted(p.issues[c.Issue]) {
				// The reporter has no write access: the issue is theirs
				// (ADR-0018), the act is for a person to decide.
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "reporter-outside", Where: c.where(),
					Message: "opened by someone without write access to the project: " + what + " is proposed, not done"})
			}
			if c.Do == "milestone" && c.Milestone == "" {
				d.Mode = Propose // slipped, no open milestone to move it to: a person's to place
			}
			if rounds := RoundsMax(settings); d.Mode == Act && c.Round > rounds {
				// Its reporter written to as many times as allowed: a person
				// takes it from here, in the report.
				d.Mode, c.Spent = Propose, true
				d.Act = c
				p.Findings = append(p.Findings, verdict.Finding{Rule: "asks-spent", Where: c.where(),
					Message: fmt.Sprintf("its reporter was written to %d times (acts.ask.rounds): what is left is proposed to a person in the report", rounds)})
			}
			if d.Mode == Act && s.Max > 0 && done[c.Kind()] >= s.Max {
				d.Mode, d.Capped = Propose, c.From == ""
				p.Findings = append(p.Findings, verdict.Finding{Rule: "act-cap", Where: c.where(),
					Message: fmt.Sprintf("%s: at most %d a run; this one is proposed", c.Kind(), s.Max)})
			}
			if d.Mode == Act && slices.Contains(moves, c.Do) && !moved[c.Issue] {
				if len(moved) >= movedMax {
					// The engine's own move (a slip) is found again at the
					// next run: its issue is not read again for it.
					d.Mode, d.Capped = Propose, c.From == ""
					p.Findings = append(p.Findings, verdict.Finding{Rule: "moved-cap", Where: c.where(),
						Message: fmt.Sprintf("a run moves at most %d of the %d open issues (moved-percent-max: %d); this move is proposed", movedMax, backlogSize, movedPercent)})
				} else {
					moved[c.Issue] = true
				}
			}
			if d.Mode == Act {
				done[c.Kind()]++
				if c.Do == "refine" && !c.ToReporter {
					// A ready later in the run reads the body as this leaves it.
					is := p.issues[c.Issue]
					is.Body, _, _ = Refine(is.Body, c, role)
					p.issues[c.Issue] = is
				}
				if c.Do == "close" {
					p.Record.Closed = append(p.Record.Closed, Closing{Issue: c.Issue, Act: c.Kind()})
				}
			}
		}
		p.Decisions = append(p.Decisions, d)
	}
	p.keepProposed()
	return p, nil
}

// keepProposed carries over the acts proposed by earlier runs whose issue
// is still open — or, for an issue to open, that no open issue holds yet —
// and that this run did not decide again, then adds this run's: a proposal
// stays in the report until a person settles it.
func (p *Plan) keepProposed() {
	decided := map[string]bool{}
	for _, d := range p.Decisions {
		if d.Mode != Off {
			decided[d.Act.key()] = true
		}
	}
	var kept []Pending
	for _, q := range p.Record.Proposed {
		key, settled := q.Key, false
		if key == "" {
			key, settled = fmt.Sprintf("%d/%s", q.Issue, q.Act), !p.open[q.Issue]
		} else {
			marker := forge.Marker(key)
			settled = slices.ContainsFunc(p.bodies, func(b string) bool { return strings.Contains(b, marker) })
		}
		if q.Capped && slices.Contains(p.read, q.Issue) {
			settled = true // read again: the agent decided it anew, or not at all
		}
		if !settled && !decided[key] {
			kept = append(kept, q)
		}
	}
	for _, d := range p.Decisions {
		if d.Mode == Propose {
			q := Pending{Issue: d.Act.Issue, Act: d.Act.Kind(), Line: describe(d.Act, "Close"), Capped: d.Capped}
			if d.Act.Do == "open" {
				q.Key = d.Act.key()
			}
			kept = append(kept, q)
		}
	}
	if len(kept) != len(p.Record.Proposed) {
		p.Changed = true
	}
	p.Record.Proposed = kept
}

// readRecord finds the report issue and what it records, and looks for
// wrong closings: an issue the role closed, open again, puts that kind of
// act back to propose.
func (p *Plan) readRecord(f forge.Backlog, role string) error {
	open, err := f.Issues()
	if err != nil {
		return err
	}
	p.open, p.seen, p.issues = map[int]bool{}, map[string]bool{}, map[int]forge.Issue{}
	for _, is := range open {
		p.open[is.ID] = true
		p.issues[is.ID] = is
		p.bodies = append(p.bodies, is.Body)
		if is.Title == ReportTitle(role) {
			p.Report = is.ID
		}
	}
	if p.Report == 0 {
		return nil
	}
	comments, err := f.Comments(forge.Target{Kind: "issue", ID: p.Report})
	if err != nil {
		return err
	}
	if _, err := readBlock(comments, RecordMarker(role), &p.Record); err != nil {
		// What it did cannot be read: nothing is done, everything proposed.
		p.Findings = append(p.Findings, verdict.Finding{Rule: "record-broken", Where: fmt.Sprintf("#%d", p.Report),
			Message: "the record of what the role did does not read (" + err.Error() + "); every act is proposed until it is repaired"})
		p.Record = Record{Propose: []string{"close-duplicate", "close-obsolete"}}
		return nil
	}
	isOpen := p.open
	var kept []Closing
	for _, c := range p.Record.Closed {
		if !isOpen[c.Issue] {
			kept = append(kept, c)
			continue
		}
		p.Changed = true
		p.Record.Wrong = append(p.Record.Wrong, c)
		if !slices.Contains(p.Record.Propose, c.Act) {
			p.Record.Propose = append(p.Record.Propose, c.Act)
		}
		p.Findings = append(p.Findings, verdict.Finding{Rule: "wrong-closing", Where: fmt.Sprintf("#%d", c.Issue),
			Message: fmt.Sprintf("closed by the role (%s), open again: %s is back to propose until a person sets it to act", c.Act, c.Act)})
	}
	p.Record.Closed = kept
	return nil
}

// check says why a closing cannot be done, as a finding's rule and
// message, or nothing.
func (p *Plan) check(f forge.Backlog, repo, role string, c *Proposal) (rule, why string) {
	if c.Do == "open" {
		// An issue opened from a file: its text is the file's, quoted, never
		// written by the agent; opened once.
		if t := strings.TrimSpace(c.Title); t == "" || len(t) > 120 || strings.Contains(t, "\n") {
			return "open-title", "an issue's title is one line, 120 characters at most"
		}
		if c.Quote == nil || c.Quote.Path == "" || strings.TrimSpace(c.Quote.Text) == "" {
			return "no-quote", "an issue opened from a file quotes its text: {path, text}"
		}
		if !p.found(f, repo, *c.Quote) {
			return "no-quote", fmt.Sprintf("the quote is not found in %s", c.Quote.Path)
		}
		key := forge.Marker(ImportKey(*c.Quote))
		if p.seen[key] || slices.ContainsFunc(p.bodies, func(b string) bool { return strings.Contains(b, key) }) {
			return "already-open", "an open issue already holds this text"
		}
		p.seen[key] = true
		return "", ""
	}
	if rule, why := p.checkRefining(repo, role, c); rule != "" {
		return rule, why
	}
	if rule, why := p.checkSplitRename(repo, c); rule != "" {
		return rule, why
	}
	if c.Do == "milestone" || c.Do == "order" {
		// Ordering says nothing of an issue's truth: no quote, its state
		// readable all the same.
		is, ok := p.issues[c.Issue]
		if !ok {
			return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
		}
		c.Before = Before(is)
		switch {
		case c.Do == "order" && (c.Priority < 1 || c.Priority > Levels):
			return "priority-level", fmt.Sprintf("a priority is a level from 1, the most pressing, to %d", Levels)
		case c.Do == "order":
		case c.From != "" && (c.From != is.Milestone || !Released(repo, c.From)):
			return "not-slipped", fmt.Sprintf("#%d is not in the released milestone %q", c.Issue, c.From)
		case c.From != "" && c.Milestone == "":
			// slipped, no milestone to move it to: proposed
		default:
			if t := strings.TrimSpace(c.Milestone); t == "" || len(t) > 60 || strings.ContainsAny(t, "\n") {
				return "milestone-title", "a milestone is a title of one line, 60 characters at most"
			}
		}
	} else if c.Do == "sources" {
		if len(c.Sources) == 0 || len(c.Sources) > maxSources {
			return "sources-unknown", fmt.Sprintf("an issue names 1 to %d sources", maxSources)
		}
		for _, s := range c.Sources {
			path, _, _ := strings.Cut(s, "#")
			if exec.Command("git", "-C", repo, "cat-file", "-e", "HEAD:"+path).Run() != nil {
				return "sources-unknown", fmt.Sprintf("%s is not in the commit the run is on", path)
			}
		}
		if c.Quote != nil && c.Quote.Path != "" && !slices.ContainsFunc(c.Sources, func(s string) bool { return strings.HasPrefix(s, c.Quote.Path) }) {
			return "no-quote", "the quote naming the sources comes from one of them"
		}
	} else if c.Do == "close" && !slices.Contains(closeReasons, c.Reason) {
		return "close-reason", fmt.Sprintf("closing as %q: a role closes a duplicate or an obsolete issue; refusing a need is a person's (principle 1)", c.Reason)
	}
	if c.Reason == "duplicate" && (c.DuplicateOf <= 0 || c.DuplicateOf == c.Issue) {
		return "close-reason", "a duplicate names its original (duplicate-of)"
	}
	comments, err := f.Comments(forge.Target{Kind: "issue", ID: c.Issue})
	if err != nil {
		return "no-state", err.Error()
	}
	st, found, err := ReadState(comments, role)
	if !found {
		return "no-state", "the issue has no state comment yet: it is not acted on before the engine has one"
	} else if err != nil {
		return "state-broken", "the issue's state comment does not read (" + err.Error() + "): nothing is written on it"
	}
	if c.Do == "order" {
		set := PriorityLabels(p.issues[c.Issue])
		switch {
		case !(len(set) == 0 && st.Priority == 0) && !(len(set) == 1 && set[0] == st.Priority):
			return "priority-kept", "its priority was set by a person (a label the role did not set, or took off): it is kept"
		case len(set) == 1 && set[0] == c.Priority:
			return "priority-same", fmt.Sprintf("it has priority %d already", c.Priority)
		}
	}
	if c.Do == "ask" || c.ToReporter {
		if rule, why := conversation(role, comments, c); rule != "" {
			return rule, why
		}
	}
	switch {
	case c.Do == "split" && len(st.Split) > 0:
		return "already-split", fmt.Sprintf("it was split already, into %s: an issue is split once", issueList(st.Split))
	case c.Do == "rename" && st.Title != "" && st.Title != p.issues[c.Issue].Title:
		return "title-kept", "its title was set by a person after the role's: it is kept"
	}
	if c.Do == "milestone" || c.Do == "order" || c.Do == "refine" || c.Do == "ready" || c.Do == "ask" || c.Do == "split" || c.Do == "rename" {
		return "", "" // writing a plan or asking says nothing of the issue's truth: no quote
	}
	if c.Quote == nil || strings.TrimSpace(c.Quote.Text) == "" {
		return "no-quote", "no quote: an act cites the code or the issue it rests on"
	}
	if !p.found(f, repo, *c.Quote) {
		return "no-quote", fmt.Sprintf("the quote %q is not found where it says", c.Quote.Text)
	}
	return "", ""
}

// checkRefining checks a refine, a ready or an ask against the issue as it
// is: what refining would add, whether it is ready, whether its reporter
// was asked already.
func (p *Plan) checkRefining(repo, role string, c *Proposal) (rule, why string) {
	is, ok := p.issues[c.Issue]
	switch {
	case c.Do != "refine" && c.Do != "ready" && c.Do != "ask":
		return "", ""
	case !ok:
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	}
	switch c.Do {
	case "refine":
		if c.Scope != "" || len(c.Sources) > 0 {
			if len(c.Sources) == 0 || len(c.Sources) > maxSources {
				return "sources-unknown", fmt.Sprintf("a scope names 1 to %d files (sources)", maxSources)
			}
			for _, s := range c.Sources {
				path, _, _ := strings.Cut(s, "#")
				if exec.Command("git", "-C", repo, "cat-file", "-e", "HEAD:"+path).Run() != nil {
					return "sources-unknown", fmt.Sprintf("%s is not in the commit the run is on", path)
				}
			}
		}
		// An outsider's issue is theirs (ADR-0021): the refined text is
		// proposed to its reporter in a comment until they, or a person of
		// the project, agree. A role's finding is the line's own draft.
		c.ToReporter = !is.Insider && !Accepted(is) && OpenedBy(is.Body) == ""
		_, added, kept := Refine(is.Body, *c, role)
		c.Added = added
		if len(added) == 0 {
			return "nothing-to-refine", "every section it writes is in the body already: none is rewritten"
		}
		if len(kept) > 0 {
			p.Findings = append(p.Findings, verdict.Finding{Rule: "section-kept", Where: c.where(),
				Message: "already in the body, left as it is: " + strings.Join(kept, ", ")})
		}
	case "ready":
		if missing := NotReady(is.Body, Accepted(is)); len(missing) > 0 {
			return "not-ready", "it stays to refine: " + strings.Join(missing, "; ")
		}
	case "ask":
		if strings.TrimSpace(c.Questions) == "" {
			return "ask-empty", "an ask holds the questions"
		}
	}
	return "", ""
}

// maxChildren bounds the children of one split.
const maxChildren = 6

// checkSplitRename checks a split or a rename against the issue as it is
// (ADR-0022): a title of one line, a child with its four sections and the
// files its scope names.
func (p *Plan) checkSplitRename(repo string, c *Proposal) (rule, why string) {
	if c.Do != "split" && c.Do != "rename" {
		return "", ""
	}
	is, ok := p.issues[c.Issue]
	if !ok {
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	}
	if c.Do == "rename" {
		c.Before = is.Title
		switch t := strings.TrimSpace(c.Title); {
		case !oneLine(t):
			return "rename-title", "a title is one line, 120 characters at most"
		case t == strings.TrimSpace(is.Title):
			return "title-same", "it has this title already"
		}
		return "", ""
	}
	if len(c.Into) < 2 || len(c.Into) > maxChildren {
		return "split-size", fmt.Sprintf("a split makes 2 to %d issues", maxChildren)
	}
	keys := map[string]bool{}
	for _, ch := range c.Into {
		if !oneLine(strings.TrimSpace(ch.Title)) {
			return "split-title", "each child's title is one line, 120 characters at most"
		}
		if key := SplitKey(c.Issue, ch.Title); keys[key] {
			return "split-title", fmt.Sprintf("%q: two children have this title; each is a need of its own", ch.Title)
		} else {
			keys[key] = true
		}
		for _, sec := range []struct{ name, text string }{{"need", ch.Need}, {"verification", ch.Verification}, {"validation", ch.Validation}, {"scope", ch.Scope}} {
			if strings.TrimSpace(sec.text) == "" {
				return "split-sections", fmt.Sprintf("%q has no %s: each child has its four sections", ch.Title, sec.name)
			}
		}
		if len(ch.Sources) == 0 || len(ch.Sources) > maxSources {
			return "sources-unknown", fmt.Sprintf("%q: a scope names 1 to %d files (sources)", ch.Title, maxSources)
		}
		for _, s := range ch.Sources {
			path, _, _ := strings.Cut(s, "#")
			if out, err := exec.Command("git", "-C", repo, "cat-file", "-t", "HEAD:"+path).Output(); path == "" || err != nil || strings.TrimSpace(string(out)) != "blob" {
				return "sources-unknown", fmt.Sprintf("%q is not a file in the commit the run is on", path)
			}
		}
	}
	return "", ""
}

func oneLine(t string) bool {
	return t != "" && utf8.RuneCountInString(t) <= 120 && !strings.Contains(t, "\n")
}

func issueList(ids []int) string {
	var out []string
	for _, id := range ids {
		out = append(out, fmt.Sprintf("#%d", id))
	}
	return strings.Join(out, ", ")
}

// SplitKey is the marker key of a split's child: its parent and its
// title, so a split run again finds the children it opened.
func SplitKey(parent int, title string) string {
	return fmt.Sprintf("split=%d/%s", parent, strings.TrimPrefix(TitleKey(title), "issue=title#"))
}

// ChildBody is a split's child's body: a line naming its parent, then its
// four sections, Need and Validation as drafts.
func ChildBody(parent int, ch Child, role string) string {
	body, _, _ := Refine(fmt.Sprintf("Part of #%d.", parent), Proposal{Need: ch.Need, Verification: ch.Verification, Validation: ch.Validation, Scope: ch.Scope}, role)
	return body
}

// SubIssuesHeading heads the task list of a parent's children, on a forge
// without sub-issues.
const SubIssuesHeading = "## Sub-issues"

// ListChildren is a parent's body with its children as a task list under
// SubIssuesHeading: added after its text, or the list already there
// rewritten.
func ListChildren(body string, ids []int) string {
	var list []string
	for _, id := range ids {
		list = append(list, fmt.Sprintf("- [ ] #%d", id))
	}
	text := strings.Join(list, "\n")
	if _, ok := work.Sections(body)["Sub-issues"]; ok {
		return fill(body, "Sub-issues", text)
	}
	return strings.TrimRight(body, " \r\n") + "\n\n" + SubIssuesHeading + "\n\n" + text
}

// Sections are an issue's, in the order they are written.
var Sections = []string{"Need", "Verification", "Validation", "Scope"}

// drafted are the sections the role writes as drafts: a person's to state.
var drafted = []string{"Need", "Validation"}

// DraftMarker marks a section the role drafted and no person made theirs.
var DraftMarker = forge.Marker("draft")

// DraftLine opens a drafted section; deleting it makes the section a person's.
func DraftLine(role string) string {
	return fmt.Sprintf("*Draft by the %s: edit it, then delete this line to make it yours.* %s", strings.ReplaceAll(role, "-", " "), DraftMarker)
}

// Refine is an issue's body with the sections it lacks written: one there
// but empty (an issue form's "_No response_") filled in place, one missing
// added after the text, which stays as it is; added and kept name the
// sections written and those already there, left alone.
func Refine(body string, c Proposal, role string) (string, []string, []string) {
	have := work.Sections(body)
	given := map[string]string{"Need": c.Need, "Verification": c.Verification, "Validation": c.Validation, "Scope": c.Scope}
	body = strings.TrimRight(body, " \r\n")
	var added, kept []string
	for _, name := range Sections {
		text := strings.TrimSpace(given[name])
		if text == "" {
			continue
		}
		if slices.Contains(drafted, name) {
			text = DraftLine(role) + "\n\n" + text
		}
		old, ok := have[name]
		switch {
		case ok && strings.TrimSpace(old) != "":
			kept = append(kept, name)
			continue
		case ok:
			body = fill(body, name, text)
		default:
			body += "\n\n## " + name + "\n\n" + text
		}
		added = append(added, name)
	}
	return body, added, kept
}

// fill writes text under the heading of an empty section, in place.
func fill(body, name, text string) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if h, ok := work.Heading(l); !ok || h != name {
			continue
		}
		end := i + 1
		for end < len(lines) {
			if _, ok := work.Heading(lines[end]); ok || strings.HasPrefix(lines[end], "# ") {
				break
			}
			end++
		}
		tail := ""
		if end < len(lines) {
			tail = "\n\n" + strings.Join(lines[end:], "\n")
		}
		return strings.Join(lines[:i+1], "\n") + "\n\n" + text + tail
	}
	return body
}

// The labels of an issue on its way to ready (docs/spec/routing.md).
const (
	LabelToRefine = "workline:to-refine"
	LabelDraft    = "workline:draft"    // the role drafted its Need or Validation
	LabelAccepted = "workline:accepted" // a person accepted the drafts: only who may triage sets a label
	LabelReady    = "workline:ready"
)

// Accepted says whether a person accepted an issue's drafts, by its label
// — read on the issue, never taken from a proposal.
func Accepted(is forge.Issue) bool { return slices.Contains(is.Labels, LabelAccepted) }

// StripDrafts takes the draft lines out of a body: the drafts are a
// person's once accepted.
func StripDrafts(body string) string {
	var out []string
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], DraftMarker) {
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				i++ // and the blank line after it
			}
			continue
		}
		out = append(out, lines[i])
	}
	return strings.Join(out, "\n")
}

// NotReady says what keeps an issue's body from ready: a section missing or
// empty, Need or Validation still a draft no person accepted.
func NotReady(body string, accepted bool) []string {
	have := work.Sections(body)
	var out []string
	for _, name := range Sections {
		text := strings.TrimSpace(have[name])
		switch {
		case text == "":
			out = append(out, "## "+name+" is missing or empty")
		case !accepted && slices.Contains(drafted, name) && strings.Contains(text, DraftMarker):
			out = append(out, "## "+name+" is a draft, not a person's yet")
		}
	}
	return out
}

// BodyDigest is what the state keeps of a body, to tell when a person
// changed it.
func BodyDigest(body string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(body)))
	return fmt.Sprintf("%x", sum[:6])
}

// AskMarker marks the comment asking an issue's reporter: the first round's
// as it always was, a later one's with its round.
func AskMarker(role string, round int) string {
	if round <= 1 {
		return forge.Marker(role + "/ask")
	}
	return forge.Marker(fmt.Sprintf("%s/ask=%d", role, round))
}

// ProposalMarker marks the comment proposing an outsider's issue refined.
func ProposalMarker(role string, round int) string {
	return forge.Marker(fmt.Sprintf("%s/proposal=%d", role, round))
}

// RoundsMax is how many times an issue's reporter is written to — asked, or
// proposed a refined text — before a person takes it: acts.ask.rounds,
// three when it is not set (ADR-0021).
func RoundsMax(settings map[string]Setting) int {
	if n := settings["ask"].Rounds; n > 0 {
		return n
	}
	return 3
}

// Exchange is the conversation with an issue's reporter, read from its
// comments: the engine's asks and proposals, in order, and whether a
// person wrote after the last one.
type Exchange struct {
	Rounds   int      // the engine's comments to the reporter
	Answered bool     // a person's comment after the last of them
	Asked    []string // each one's text, its marker left out
}

// ReadExchange reads the conversation with an issue's reporter.
func ReadExchange(comments []string, role string) Exchange {
	var e Exchange
	for _, c := range comments {
		_, engine := EngineMarker(c)
		switch {
		case Round(c, role):
			e.Rounds++
			e.Answered = false
			e.Asked = append(e.Asked, askedIn(c))
		case !engine:
			e.Answered = true
		}
	}
	return e
}

var engineMarker = regexp.MustCompile(`<!-- workline:[^>]*-->`)

// EngineMarker says whether the engine wrote a comment, and its marker:
// the engine ends each of its comments with one; a person quoting one
// holds it inside, and is a person's all the same.
func EngineMarker(comment string) (string, bool) {
	last := strings.TrimSpace(comment)
	if i := strings.LastIndex(last, "\n"); i >= 0 {
		last = strings.TrimSpace(last[i+1:])
	}
	return last, engineMarker.FindString(last) == last
}

// Round says whether a comment is one of the engine's rounds with an
// issue's reporter: an ask or a proposal.
func Round(comment, role string) bool {
	last, engine := EngineMarker(comment)
	return engine && (strings.HasPrefix(last, "<!-- workline:"+role+"/ask") || strings.HasPrefix(last, "<!-- workline:"+role+"/proposal="))
}

// The leads the engine writes before the questions: an ask's (Ask), and a
// proposal's paragraph of what it still needs (ProposalComment).
var askLead = regexp.MustCompile(`(?is)^.*?to refine this issue(?:, still)?: `)

const needsLead = "**What it still needs:** "

// askedIn is the questions an engine comment asked, its lead and marker
// left out: in a proposal, only what it still needs.
func askedIn(comment string) string {
	c := strings.TrimSpace(engineMarker.ReplaceAllString(comment, ""))
	if _, rest, ok := strings.Cut(c, needsLead); ok {
		line, _, _ := strings.Cut(rest, "\n\n")
		return line
	}
	if strings.Contains(comment, "/proposal=") {
		return "" // a proposal that needed nothing asked nothing
	}
	return askLead.ReplaceAllString(c, "")
}

// conversation checks an ask, or a refine proposed to an outsider, against
// the conversation so far: never twice without an answer, never the same
// question twice; it sets the round the comment would be.
func conversation(role string, comments []string, c *Proposal) (rule, why string) {
	e := ReadExchange(comments, role)
	if e.Rounds > 0 && !e.Answered {
		if c.Do == "ask" {
			return "already-asked", "its reporter was asked and has not answered since: an issue is asked again only after an answer"
		}
		return "already-proposed", "its reporter was proposed a text, or asked, and has not answered since: nothing more is written before an answer"
	}
	for _, q := range questions(c.Questions) {
		// The same question, whole: the leads the engine wrote are not in
		// e.Asked (ReadExchange).
		if slices.ContainsFunc(e.Asked, func(a string) bool { return slices.Contains(questions(a), q) }) {
			return "asked-before", fmt.Sprintf("%q was asked before: a question is never asked twice", q)
		}
	}
	c.Round = e.Rounds + 1
	return "", ""
}

// sentenceEnd is where a sentence ends before a question.
var sentenceEnd = regexp.MustCompile(`[.!] (?:[-*•] +|\d+[.)] +)?\p{Lu}`)

// bullet is a list's bullet or number before a question.
var bullet = regexp.MustCompile(`^\s*(?:[-*•]|\d+[.)])\s+`)

// questions cuts a text into its questions, each squeezed and lowercased,
// to tell one asked before.
func questions(text string) []string {
	var out []string
	for _, part := range strings.SplitAfter(text, "?") {
		if !strings.HasSuffix(part, "?") {
			continue
		}
		part = squeeze(part)
		// The question alone, not the sentence before it: a sentence ends
		// at a "." or "!", a space, then a capital — perhaps after a bullet —,
		// not at a file's or a version's dot, nor at "e.g.".
		if m := sentenceEnd.FindAllStringIndex(part, -1); len(m) > 0 {
			part = part[m[len(m)-1][0]+2:]
		}
		part = bullet.ReplaceAllString(part, "") // a list's bullet or number aside
		if q := strings.ToLower(strings.Trim(part, " *")); len(q) > 3 {
			out = append(out, q)
		}
	}
	return out
}

// Ask is the comment asking the reporter what is missing; a later round
// says it follows their answer.
func Ask(author, questions string, round int) string {
	lead := "to refine this issue"
	if round > 1 {
		lead = "thank you; to refine this issue, still"
	}
	if author == "" {
		return strings.ToUpper(lead[:1]) + lead[1:] + ": " + strings.TrimSpace(questions)
	}
	return fmt.Sprintf("@%s, %s: %s", author, lead, strings.TrimSpace(questions))
}

// ProposalComment is the comment proposing an outsider's issue refined: what the
// role understood, its sections as it would write them, what it needs, and
// how to agree. Nothing is written in the body before (ADR-0021). The
// sections are kept in a YAML block, read back when it is agreed to.
func ProposalComment(author string, c Proposal, role string) string {
	// No fence of the agent's: it would end the engine's block, or be read
	// for it, when the text is agreed to.
	for _, s := range []*string{&c.Why, &c.Need, &c.Verification, &c.Validation, &c.Scope, &c.Questions} {
		*s = strings.ReplaceAll(*s, "```", "'''")
	}
	var b strings.Builder
	who := ""
	if author != "" {
		who = "@" + author + ", "
	}
	name := strings.ReplaceAll(role, "-", " ")
	fmt.Fprintf(&b, "%sthe %s read this issue and would refine it as below. Nothing is changed in your issue until you agree.\n\n", who, name)
	if w := strings.TrimSpace(c.Why); w != "" {
		fmt.Fprintf(&b, "**What it understood:** %s\n\n", w)
	}
	given := map[string]string{"Need": c.Need, "Verification": c.Verification, "Validation": c.Validation, "Scope": c.Scope}
	for _, name := range Sections {
		if !slices.Contains(c.Added, name) {
			continue
		}
		draft := ""
		if slices.Contains(drafted, name) {
			draft = " (yours to state: a draft from your words)"
		}
		fmt.Fprintf(&b, "**%s**%s\n\n%s\n\n", name, draft, strings.TrimSpace(given[name]))
	}
	if q := strings.TrimSpace(c.Questions); q != "" {
		fmt.Fprintf(&b, "**What it still needs:** %s\n\n", q)
	}
	fmt.Fprintf(&b, "To agree, copy these sections into your issue (edit it), changing what is wrong; or reply, and it reads your answer at its next run. A maintainer may agree for the project with the label `%s`: the sections are then written in the issue.\n\n", LabelAccepted)
	kept := struct {
		Need         string   `yaml:"need,omitempty"`
		Verification string   `yaml:"verification,omitempty"`
		Validation   string   `yaml:"validation,omitempty"`
		Scope        string   `yaml:"scope,omitempty"`
		Sources      []string `yaml:"sources,flow,omitempty"`
	}{}
	for _, name := range c.Added {
		switch name {
		case "Need":
			kept.Need = c.Need
		case "Verification":
			kept.Verification = c.Verification
		case "Validation":
			kept.Validation = c.Validation
		case "Scope":
			kept.Scope, kept.Sources = c.Scope, c.Sources
		}
	}
	data, _ := yaml.Marshal(kept)
	b.WriteString(engineBlock + "\n\n```yaml\n" + string(data) + "```\n</details>")
	return b.String()
}

// engineBlock opens the block of a proposal the engine reads back.
const engineBlock = "<details><summary>As the engine reads it</summary>"

// LastProposal reads the sections of the last refined text proposed to an
// issue's reporter; nil when none was, or it does not read.
func LastProposal(comments []string, role string) *Proposal {
	prefix := "<!-- workline:" + role + "/proposal="
	for i := len(comments) - 1; i >= 0; i-- {
		if last, _ := EngineMarker(comments[i]); !Round(comments[i], role) || !strings.HasPrefix(last, prefix) {
			continue
		}
		// The engine's own block, after its summary: the agent's words come
		// before it, their fences neutralised (ProposalComment).
		at := strings.LastIndex(comments[i], engineBlock)
		if at < 0 {
			return nil
		}
		m := fenced.FindStringSubmatch(comments[i][at:])
		if m == nil {
			return nil
		}
		var p Proposal
		if yaml.Unmarshal([]byte(m[1]), &p) != nil {
			return nil
		}
		return &p
	}
	return nil
}

// found says whether a quote is there, as written but for spaces: in a file
// at the commit the run is on, or in an issue's body or comments.
func (p *Plan) found(f forge.Backlog, repo string, q Quote) bool {
	want := squeeze(q.Text)
	var where []string
	switch {
	case q.Path != "":
		out, err := exec.Command("git", "-C", repo, "show", "HEAD:"+q.Path).Output()
		if err != nil {
			return false
		}
		where = []string{string(out)}
	case q.Issue > 0:
		fi, ok := f.(forge.Forge)
		if !ok {
			return false
		}
		is, err := fi.Issue(q.Issue)
		if err != nil {
			return false
		}
		comments, _ := f.Comments(forge.Target{Kind: "issue", ID: q.Issue})
		where = append([]string{is.Title, is.Body}, comments...)
	}
	for _, w := range where {
		if strings.Contains(squeeze(w), want) {
			return true
		}
	}
	return false
}

func squeeze(s string) string { return strings.Join(strings.Fields(s), " ") }

// ReportBody is the report issue's body: what the run did and proposes.
func (p *Plan) ReportBody() string {
	var did, proposed []string
	for _, q := range p.Record.Proposed {
		proposed = append(proposed, "- [ ] "+q.Line)
	}
	for _, d := range p.Decisions {
		switch d.Mode {
		case Act:
			undo := " Reopen it to undo."
			switch d.Act.Do {
			case "sources":
				undo = ""
			case "milestone", "order":
				undo = " Put it back as it was before this run (below) to undo."
			case "refine":
				undo = " Edit its body to undo."
				if d.Act.ToReporter {
					undo = ""
				}
			case "ready":
				undo = " Remove the label workline:ready to undo."
			case "ask":
				undo = ""
			case "rename":
				undo = " Rename it back to undo."
			case "split":
				undo = fmt.Sprintf(" Close the issues opened from #%d to undo: its own text was left as it was.", d.Act.Issue)
			}
			did = append(did, "- "+describe(d.Act, "Closed")+undo)
		}
	}
	var b strings.Builder
	b.WriteString("What the product owner did on its last run, and what it proposes until a person settles it. A closing undone (the issue reopened) puts that kind of act back to a person.\n")
	if len(p.Record.Propose) > 0 {
		fmt.Fprintf(&b, "\nBack to propose after a wrong closing: %s.\n", strings.Join(p.Record.Propose, ", "))
	}
	for _, w := range p.Record.Wrong {
		fmt.Fprintf(&b, "- #%d, closed as %s, was reopened.\n", w.Issue, w.Act)
	}
	if len(did) > 0 {
		b.WriteString("\n## Done\n\n" + strings.Join(did, "\n") + "\n")
	}
	var before []string
	listed := map[int]bool{}
	for _, d := range p.Decisions {
		if d.Mode == Act && slices.Contains(moves, d.Act.Do) && !listed[d.Act.Issue] {
			listed[d.Act.Issue] = true
			before = append(before, fmt.Sprintf("- #%d: %s", d.Act.Issue, d.Act.Before))
		}
	}
	if len(before) > 0 {
		b.WriteString("\n## Before this run\n\nThe issues this run moved, as they were: to put the order back, set their priority label and milestone to these.\n\n" + strings.Join(before, "\n") + "\n")
	}
	if len(proposed) > 0 {
		b.WriteString("\n## Proposed\n\nFor a person: do what a line says if you agree — close the issue, write the section, set the label; an issue to open is opened by running the import again.\n\n" + strings.Join(proposed, "\n") + "\n")
	}
	return b.String()
}

// describe says one closing in a line, its evidence quoted.
func describe(c Proposal, verb string) string {
	if c.Do == "open" {
		v := "Opened"
		if verb == "Close" {
			v = "Open"
		}
		return fmt.Sprintf("%s %q, from %s.", v, c.Title, c.Quote.Path)
	}
	if c.Do == "milestone" && c.Milestone == "" {
		return fmt.Sprintf("Move #%d out of the milestone %q, released: no open milestone to move it to. %s", c.Issue, c.From, strings.TrimSpace(c.Why))
	}
	if c.Do == "milestone" {
		return fmt.Sprintf("Put #%d in the milestone %q (was: %s): %s", c.Issue, c.Milestone, c.Before, strings.TrimSpace(c.Why))
	}
	if c.Do == "order" {
		return fmt.Sprintf("Set #%d's priority to %d (was: %s): %s", c.Issue, c.Priority, c.Before, strings.TrimSpace(c.Why))
	}
	done := verb != "Close"
	if !done && c.Spent {
		// The rounds spent: a person's to settle, with the reporter.
		what := "would ask: " + strings.TrimSpace(c.Questions)
		if c.Do == "refine" {
			what = "would propose its refined text again. " + strings.TrimSpace(c.Why)
		}
		return fmt.Sprintf("Settle #%d with its reporter, written to %d times already: answer them, refine it, or close it. The product owner %s", c.Issue, c.Round-1, what)
	}
	switch c.Do {
	case "rename":
		return fmt.Sprintf("%s #%d from %q to %q: %s", map[bool]string{true: "Renamed", false: "Rename"}[done], c.Issue, c.Before, strings.TrimSpace(c.Title), strings.TrimSpace(c.Why))
	case "split":
		var titles []string
		for _, ch := range c.Into {
			titles = append(titles, fmt.Sprintf("%q", strings.TrimSpace(ch.Title)))
		}
		return fmt.Sprintf("Split #%d into %d issues: %s. %s", c.Issue, len(c.Into), strings.Join(titles, ", "), strings.TrimSpace(c.Why))
	case "refine":
		if c.ToReporter {
			return fmt.Sprintf("%s #%d's reporter (an outsider) the sections %s, in a comment: nothing written in the issue until they or a maintainer agree. %s", map[bool]string{true: "Proposed to", false: "Propose to"}[done], c.Issue, strings.Join(c.Added, ", "), strings.TrimSpace(c.Why))
		}
		var names []string
		for _, n := range c.Added {
			if slices.Contains(drafted, n) {
				n += " (draft)"
			}
			names = append(names, n)
		}
		return fmt.Sprintf("%s #%d: %s. %s", map[bool]string{true: "Refined", false: "Refine"}[done], c.Issue, strings.Join(names, ", "), strings.TrimSpace(c.Why))
	case "ready":
		return fmt.Sprintf("%s #%d to ready: %s", map[bool]string{true: "Moved", false: "Move"}[done], c.Issue, strings.TrimSpace(c.Why))
	case "ask":
		again := ""
		if c.Round > 1 {
			again = fmt.Sprintf(" again, after their answer (round %d)", c.Round)
		}
		return fmt.Sprintf("%s #%d's reporter%s: %s", map[bool]string{true: "Asked", false: "Ask"}[done], c.Issue, again, strings.TrimSpace(c.Questions))
	}
	if c.Do == "sources" {
		v := "Named"
		if verb == "Close" {
			v = "Name"
		}
		return fmt.Sprintf("%s the code #%d is about: %s. %s %s", v, c.Issue, strings.Join(c.Sources, ", "), cite(*c.Quote), strings.TrimSpace(c.Why))
	}
	what := fmt.Sprintf("%s #%d as obsolete", verb, c.Issue)
	if c.Reason == "duplicate" {
		what = fmt.Sprintf("%s #%d as a duplicate of #%d", verb, c.Issue, c.DuplicateOf)
	}
	return fmt.Sprintf("%s: %s %s", what, cite(*c.Quote), strings.TrimSpace(c.Why))
}

func cite(q Quote) string {
	where := q.Path
	if q.Issue > 0 {
		where = fmt.Sprintf("#%d", q.Issue)
	}
	return fmt.Sprintf("%s says `%s` —", where, strings.ReplaceAll(squeeze(q.Text), "`", "'"))
}

// Comment is what the closed issue is told.
func Comment(c Proposal, role string) string {
	head := "Obsolete: the code it is about changed."
	if c.Reason == "duplicate" {
		head = fmt.Sprintf("Duplicate of #%d", c.DuplicateOf)
	}
	return fmt.Sprintf("%s\n\n%s %s\n\nClosed by the %s role. Reopen it to undo: a closing undone puts this kind of act back to a person.",
		head, cite(*c.Quote), strings.TrimSpace(c.Why), role)
}

// FormatRecord is the body of the report's record comment.
func FormatRecord(r Record) string {
	data, _ := yaml.Marshal(r)
	return "What the role did, read by the engine on its next run; not to be edited by hand.\n\n```yaml\n" + string(data) + "```"
}

// Decision is what becomes of the act at index i, nil when it is none.
func (p *Plan) Decision(i int) *Decision {
	if p == nil {
		return nil
	}
	for k := range p.Decisions {
		if p.Decisions[k].Index == i {
			return &p.Decisions[k]
		}
	}
	return nil
}

// Acts says whether the act at index i is done, not proposed nor dropped.
func (p *Plan) Acts(i int) bool {
	d := p.Decision(i)
	return d != nil && d.Mode == Act
}

// ImportKey is the marker key of an issue opened from a file's text: the
// file and a digest of the text, so the same text is never opened twice.
func ImportKey(q Quote) string {
	sum := sha256.Sum256([]byte(squeeze(q.Text)))
	return fmt.Sprintf("import=%s:%x", q.Path, sum[:6])
}

// Locate finds a quote in a file, as written but for spaces: the lines it
// spans, from 1, and those lines as the file has them.
func Locate(repo, path, text string) (from, to int, original string, ok bool) {
	out, err := exec.Command("git", "-C", repo, "show", "HEAD:"+path).Output()
	if err != nil {
		return 0, 0, "", false
	}
	lines := strings.Split(string(out), "\n")
	want := squeeze(text)
	if want == "" {
		return 0, 0, "", false
	}
	first := strings.Fields(want)[0]
	for i := range lines {
		if !strings.Contains(lines[i], first) {
			continue
		}
		acc := ""
		for j := i; j < len(lines) && len(squeeze(acc)) <= len(want)+len(lines[j]); j++ {
			acc += lines[j] + "\n"
			if strings.Contains(squeeze(acc), want) {
				// The first line holding the quote's first word may come
				// before it: start at the last line that still holds it all.
				k := i
				for k < j && strings.Contains(squeeze(strings.Join(lines[k+1:j+1], "\n")), want) {
					k++
				}
				return k + 1, j + 1, strings.Join(lines[k:j+1], "\n"), true
			}
		}
	}
	return 0, 0, "", false
}

// PeopleComments counts an issue's comments that are not the engine's.
func PeopleComments(comments []string) int {
	n := 0
	for _, c := range comments {
		if !strings.Contains(c, "<!-- workline:") {
			n++
		}
	}
	return n
}

// CappedByRole lists the issues an act was proposed on only because a run's
// cap was reached: they are read again, though nothing changed on them.
func CappedByRole(f forge.Backlog, role string, open []forge.Issue) []int {
	r := record(f, role, open)
	if r == nil {
		return nil
	}
	var ids []int
	for _, q := range r.Proposed {
		if q.Capped && q.Issue > 0 {
			ids = append(ids, q.Issue)
		}
	}
	return ids
}

// record reads the role's record from its report issue; nil when there is
// none or it does not read.
func record(f forge.Backlog, role string, open []forge.Issue) *Record {
	for _, is := range open {
		if is.Title != ReportTitle(role) {
			continue
		}
		comments, err := f.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil
		}
		var r Record
		if _, err := readBlock(comments, RecordMarker(role), &r); err != nil {
			return nil
		}
		return &r
	}
	return nil
}

// ClosedByRole lists the issues the role's record says it closed, read from
// its report issue's comments; nil when there is none or it does not read.
func ClosedByRole(f forge.Backlog, role string, open []forge.Issue) []int {
	for _, is := range open {
		if is.Title != ReportTitle(role) {
			continue
		}
		comments, err := f.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil
		}
		var r Record
		if _, err := readBlock(comments, RecordMarker(role), &r); err != nil {
			return nil
		}
		var ids []int
		for _, c := range r.Closed {
			ids = append(ids, c.Issue)
		}
		return ids
	}
	return nil
}
