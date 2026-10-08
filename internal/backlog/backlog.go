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
	"time"
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
	// After are the blockers the role's split set on each child, among
	// its siblings: the role's own links, taken off once a blocker closes
	// (ADR-0028).
	After map[int][]int `yaml:"after,flow,omitempty"`
	Kept  []string      `yaml:"kept,omitempty"` // the evidence an announcement as obsolete rested on, kept open: not announced again for it
	// Sections are its Need and Scope as last read or written (Basis): a
	// person's change to them touches the issues built on it (ADR-0032).
	Sections map[string]string `yaml:"sections,omitempty"`
	// Wrote are the sections the role wrote, each the digest of its text:
	// another text there is a person's (SectionDigest). Answered is the
	// review of its spec the role was last given, by the digest of the
	// body that review read (#128): a review restarting at round 1 is
	// another review, never taken for one answered.
	Wrote    map[string]string `yaml:"wrote,omitempty"`
	Answered string            `yaml:"answered,omitempty"`
	// Deleted are the sections the role wrote that a person took out of
	// the body: a "no" for each, never written again (ADR-0038).
	Deleted []string `yaml:"deleted,flow,omitempty"`
	// What the role holds on this issue, in place of a report (ADR-0038):
	// the acts it proposes here; those it did alone a person may undo;
	// those a person undid — proposed here from then on —; what it did,
	// each with its day; the kind of act it closed it with, should it open
	// again.
	Proposed []Pending `yaml:"proposed,omitempty"`
	Done     []Done    `yaml:"done,omitempty"`
	Undone   []Undo    `yaml:"undone,omitempty"`
	Did      []Did     `yaml:"did,omitempty"`
	Closed   string    `yaml:"closed,omitempty"`
	// Label is the label the role put on it while it waits on a person
	// (workline:proposed); Aside, the body's digest when a person took it
	// off: nothing proposed until the issue changes.
	Label string `yaml:"label,omitempty"`
	Aside string `yaml:"aside,omitempty"`
	// Heard are the comments of its reporter and the project's people the
	// role read; Revisions, the times it revised its drafts for them.
	Heard     *int `yaml:"heard,omitempty"`
	Revisions int  `yaml:"revisions,omitempty"`
}

// StateMarker marks the comment holding an issue's state.
func StateMarker(role string) string { return forge.Marker("sticky=" + role + "/state") }

// RecordMarker marks the report's comment holding what the role did.
func RecordMarker(role string) string { return forge.Marker("sticky=" + role + "/acts") }

// ReportTitle is the title of the role's report issue.
func ReportTitle(role string) string { return "Backlog — " + strings.ReplaceAll(role, "-", " ") }

var fenced = regexp.MustCompile("(?s)```yaml\n(.*?)```")

// readBlock decodes the fenced YAML block of the comment carrying marker
// into v: the last, when another user's left behind could not be edited
// (GitLab: a note is its author's). found is false when no comment carries
// it; err when it does not read.
func readBlock(comments []string, marker string, v any) (found bool, err error) {
	for i := len(comments) - 1; i >= 0; i-- {
		c := comments[i]
		if !strings.Contains(c, marker) {
			continue
		}
		text, ok := yamlBlock(c)
		if !ok {
			return true, fmt.Errorf("no YAML block")
		}
		dec := yaml.NewDecoder(strings.NewReader(text))
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
		Sources   []string          `yaml:"sources,flow"`
		Confirmed string            `yaml:"confirmed"`
		Judged    string            `yaml:"judged,omitempty"`
		Comments  int               `yaml:"comments,omitempty"`
		Body      string            `yaml:"body,omitempty"`
		Priority  int               `yaml:"priority,omitempty"`
		Title     string            `yaml:"title,omitempty"`
		Split     []int             `yaml:"split,flow,omitempty"`
		After     map[int][]int     `yaml:"after,flow,omitempty"`
		Kept      []string          `yaml:"kept,flow,omitempty"`
		Sections  map[string]string `yaml:"sections,omitempty"`
		Wrote     map[string]string `yaml:"wrote,omitempty"`
		Answered  string            `yaml:"answered,omitempty"`
		Deleted   []string          `yaml:"deleted,flow,omitempty"`
		Proposed  []Pending         `yaml:"proposed,omitempty"`
		Done      []Done            `yaml:"done,omitempty"`
		Undone    []Undo            `yaml:"undone,omitempty"`
		Did       []Did             `yaml:"did,omitempty"`
		Closed    string            `yaml:"closed,omitempty"`
		Label     string            `yaml:"label,omitempty"`
		Aside     string            `yaml:"aside,omitempty"`
		Heard     *int              `yaml:"heard,omitempty"`
		Revisions int               `yaml:"revisions,omitempty"`
	}{s.Sources, s.Confirmed, s.Judged, s.Comments, s.Body, s.Priority, s.Title, s.Split, s.After, s.Kept, s.Sections, s.Wrote, s.Answered, s.Deleted,
		s.Proposed, s.Done, s.Undone, s.Did, s.Closed, s.Label, s.Aside, s.Heard, s.Revisions})
	// A fence in a text the agent wrote must not end the block: the block's
	// is longer than any run of backticks in it.
	fence := fenceFor(string(data))
	block := fence + "yaml\n" + string(data) + fence
	say := s.say()
	if say == "" {
		return "What workline knows of this issue; edited by the engine, not by hand.\n\n" + block
	}
	return Inert(strings.ReplaceAll(say, "```", "'''")) + "\n\n<details><summary>What workline knows of this issue, edited by the engine</summary>\n\n" + block + "\n\n</details>"
}

// fenceFor is a code fence longer than any run of backticks in text: three,
// or one more than its longest run, so nothing in it closes the block.
func fenceFor(text string) string {
	n, run := 3, 0
	for _, r := range text {
		if r == '`' {
			run++
			n = max(n, run+1)
		} else {
			run = 0
		}
	}
	return strings.Repeat("`", n)
}

// yamlBlock is the text of the first YAML code block in a comment: its
// fence three backticks or more, closed by the same at a line's start.
func yamlBlock(c string) (string, bool) {
	m := openYAML.FindStringSubmatchIndex(c)
	if m == nil {
		return "", false
	}
	fence := c[m[2]:m[3]]
	rest := c[m[1]:]
	for at := 0; ; {
		k := strings.Index(rest[at:], fence)
		if k < 0 {
			return "", false
		}
		k += at
		after := k + len(fence)
		if (k == 0 || rest[k-1] == '\n') && (after == len(rest) || rest[after] != '`') {
			return rest[:k], true
		}
		at = after
	}
}

// openYAML opens a YAML code block: a fence of three backticks or more.
var openYAML = regexp.MustCompile("(`{3,})yaml\n")

// say is what the role's one comment on an issue tells a person, above its
// state (ADR-0038): what it did there last, what it proposes, what it
// wants from them — a few lines; "" when it has nothing to say.
func (s State) say() string {
	var lines []string
	if n := len(s.Did); n > 0 {
		day := s.Did[n-1].Day
		var did []string
		for _, d := range s.Did {
			if d.Day == day && d.Line != "" {
				if d.Yes != "" {
					d.Line += " Accepted by " + d.Yes + "."
				}
				did = append(did, "- "+d.Line)
			}
		}
		if len(did) > 0 {
			lines = append(lines, fmt.Sprintf("**The product owner**, on %s:\n\n%s", day, strings.Join(did, "\n")))
		}
	}
	for _, q := range s.Proposed {
		line := q.Line
		if q.Proposal != nil {
			line = offer(*q.Proposal)
		}
		lines = append(lines, "- **Proposes**: "+line)
	}
	for _, q := range s.Proposed {
		if c := q.Proposal; c != nil && c.Do == "refine" && !c.ToReporter {
			lines = append(lines, fold("The sections it proposes", drafts(*c)))
		}
	}
	switch {
	case s.Aside != "":
		lines = append(lines, "Set aside by a person: nothing more is proposed here until the issue changes.")
	case s.Label != "":
		if len(s.Proposed) == 0 && len(s.Did) == 0 {
			lines = append(lines, "**The product owner** drafted sections of this issue, marked as drafts in its body: they wait on your answer.")
		}
		accepted := strings.Replace(s.Label, "proposed", "accepted", 1)
		lines = append(lines, fmt.Sprintf("**Your answer**: label `%s` to agree — what it proposes is done at its next run, its drafts become yours, and the issue moves to ready once complete; comment to have it revise; take `%s` off for not now; close the issue if it should not exist.", accepted, s.Label))
	}
	return strings.Join(lines, "\n\n")
}

// drafts says the sections a refine would write, each under its name.
func drafts(c Proposal) string {
	given := map[string]string{"Need": c.Need, "Verification": c.Verification, "Validation": c.Validation, "Scope": c.Scope}
	var b strings.Builder
	for _, name := range Sections {
		if text := strings.TrimSpace(given[name]); text != "" && (len(c.Added) == 0 || slices.Contains(c.Added, name)) {
			fmt.Fprintf(&b, "**%s**\n\n%s\n\n", name, strings.ReplaceAll(text, "```", "'''"))
		}
	}
	return strings.TrimSpace(b.String())
}

// Record is what the role holds on the backlog, read from each issue's
// state (Ledger) and written back to each at the end of a run (Persist):
// never kept in one place (ADR-0038).
type Record struct {
	Closed  []Closing `yaml:"closed,omitempty"`
	Wrong   []Closing `yaml:"wrong,omitempty"`        // closings found wrong: their issue open again
	Propose []string  `yaml:"propose,flow,omitempty"` // kinds of act proposed on every issue: undone too often (UndoneMax)
	// Done are the role's own acts a person may undo, other than closings:
	// a rename, a priority, a milestone, ready, a split (ADR-0026); Undone,
	// those a person undid, with what shows it.
	Done     []Done    `yaml:"done,omitempty"`
	Undone   []Undo    `yaml:"undone,omitempty"`
	Proposed []Pending `yaml:"proposed,omitempty"` // acts proposed, kept until their issue is closed, set aside or proposed again
	// Did are the acts the role did, with their day and level, for its
	// comment on each issue and the weekly sample (ADR-0033).
	Did []Did `yaml:"did,omitempty"`
}

// Pending is an act proposed to a person, on its issue.
type Pending struct {
	Issue int    `yaml:"issue,omitempty"` // left out in the issue's own state
	Act   string `yaml:"act"`
	Line  string `yaml:"line"`
	Key   string `yaml:"key,omitempty"` // an issue to open: the text it would be opened from (ImportKey)
	// Capped: proposed only because the run's cap was reached, not left to
	// a person by its mode: its issue is read again at the next run.
	Capped bool `yaml:"capped,omitempty"`
	// Proposal is what the engine would do, as decided: done as it says
	// when a person of the project labels the issue workline:accepted
	// (ADR-0038).
	Proposal *Proposal `yaml:"proposal,omitempty"`
	// Since is the day it was first proposed, YYYY-MM-DD: the job's summary
	// says it stuck past stuck-days (ADR-0031).
	Since string `yaml:"since,omitempty"`
}

// Closing is one issue the role closed.
type Closing struct {
	Issue int    `yaml:"issue"`
	Act   string `yaml:"act"`
	Level string `yaml:"level,omitempty"` // the autonomy level it was done at
}

// Quote is the evidence an act cites: a text in a file, or in an issue.
type Quote struct {
	Path  string `yaml:"path,omitempty"`
	Issue int    `yaml:"issue,omitempty"`
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
	Title       string   `yaml:"title,omitempty"`           // an issue to open, or an issue's new title (rename)
	Into        []Child  `yaml:"into,omitempty"`            // the children of a split
	BlockedBy   []int    `yaml:"blocked-by,flow,omitempty"` // depend: the issues it waits on; undepend: those taken off (ADR-0028)
	// Native, the engine's: the blockers an undepend takes off the forge's
	// own relation, the rest being in the engine's line; never the agent's.
	Native []int  `yaml:"native,flow,omitempty"`
	Quote  *Quote `yaml:"quote"`
	Why    string `yaml:"why"`
	// Refining: the sections written, Need and Validation as drafts; Added,
	// the engine's, says which the body did not have yet.
	Scope        string   `yaml:"scope,omitempty"`
	Verification string   `yaml:"verification,omitempty"`
	Need         string   `yaml:"need,omitempty"`
	Validation   string   `yaml:"validation,omitempty"`
	Added        []string `yaml:"added,omitempty"`
	// Revise, the engine's: the sections with text this refine may rewrite,
	// to answer the reviewer's findings on the spec (#128) — the role's own
	// only (Revisable); never taken from the agent.
	Revise []string `yaml:"revise,omitempty"`
	// Refused, the engine's: the sections the role wrote that a person
	// deleted (State.Deleted), never written again (ADR-0038).
	Refused   []string `yaml:"refused,omitempty"`
	Questions string   `yaml:"questions,omitempty"` // asking the reporter; for an outsider's refine, what it still needs
	// The engine's, for the conversation with the reporter: the round this
	// comment would be, and whether a refine is proposed to the reporter in
	// a comment rather than written in the body (an outsider's issue).
	Round      int  `yaml:"round,omitempty"`
	ToReporter bool `yaml:"to-reporter,omitempty"`
	// Agreed, the engine's: who agreed in a reply to the text last proposed
	// to the reporter, which this refine writes; checked again on the forge
	// (Agreement), never taken from the agent.
	Agreed string `yaml:"agreed,omitempty"`
	Spent  bool   `yaml:"spent,omitempty"` // the rounds spent: proposed to a person
	// Hearsay, the engine's: the words it quotes are only in a comment by
	// someone outside the project — proposed, never done alone (ADR-0038).
	Hearsay bool `yaml:"hearsay,omitempty"`
	// Ordering: the priority set (1 to 4); a milestone left because it is
	// released (From, the engine's); and, the engine's, the issue's
	// priority and milestone before the run, written in the report.
	Priority int    `yaml:"priority,omitempty"`
	From     string `yaml:"from,omitempty"`
	Before   string `yaml:"before,omitempty"`
	// Obsolete (ADR-0024): Announced, on the engine's closing, the day of
	// the announcement it follows; Announce and Until, the engine's, that
	// this act announces rather than closes, and the day from which it may
	// close; Judge, the engine's, the second judge's yes and its level;
	// Say, on a keep, that the issue is told why.
	Announced string `yaml:"announced,omitempty"`
	Announce  bool   `yaml:"announce,omitempty"`
	Until     string `yaml:"until,omitempty"`
	Judge     string `yaml:"judge,omitempty"`
	Say       bool   `yaml:"say,omitempty"`
	// Ticked, the engine's: who ticked this proposal's box in the report, a
	// person of the project — checked again on the forge, the act done as
	// the record says, never as the intention does (ADR-0025).
	Ticked string `yaml:"ticked,omitempty"`
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
	After        []int    `yaml:"after,flow,omitempty"` // the children it waits on, by their place in the split, from 1 (ADR-0028)
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
var Kinds = []string{"open", "close", "keep", "sources", "milestone", "order", "refine", "ready", "unready", "ask", "split", "rename", "depend", "undepend"}

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
	// Report is an earlier engine's report issue, still open: its record
	// moved to the issues, and Moved, it is closed at the end of the run
	// (ADR-0038).
	Report int  `yaml:"report,omitempty"`
	Moved  bool `yaml:"moved,omitempty"`
	// Visit are the open issues with a state, written back at the end of
	// the run (Persist); SetAside, those a person took workline:proposed
	// off; Answers, a person's comments read as "revise" on each.
	Visit    []int           `yaml:"visit,flow,omitempty"`
	SetAside map[int]bool    `yaml:"set-aside,omitempty"`
	Answers  map[int]*Answer `yaml:"answers,omitempty"`
	open     map[int]bool    // the open issues, read once
	issues   map[int]forge.Issue
	judged   map[int]Judged // the second judge's answers on the issues announced obsolete
	settings map[string]Setting
	config   Config
	bodies   []string        // their bodies, to find an import again
	seen     map[string]bool // the imports this run decided
	read     []int           // the issues this run read
	ledger   *Ledger         // what the role holds, read from the issues
	ticked   map[string]bool // the proposals a person accepted this run decided, done or dropped: settled
	added    map[int][]int   // the blockers this run's depend acts add, for the next act's cycle check
	dropped  map[int][]int   // the blockers this run's undepend acts take off
	hearsay  bool            // the last quote found was only in an outsider's comment
	// changedFor are the issues read again for a change to what they were
	// built on: every act on them proposed (ADR-0032).
	changedFor map[int]Change
}

// Setting is a kind of act's mode and cap.
type Setting struct {
	Mode   string
	Max    int
	Rounds int      // asking: the times an issue's reporter is written to, then a person (ask only)
	Days   *int     // close-obsolete: the days an announcement waits; ObsoleteDays when nil
	Exempt []string // close-obsolete: labels that keep an issue from it; DefaultExempt when nil
	Drafts string   // refine: propose has Need and Validation drafts proposed, Scope and Verification written (ADR-0026)
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
		if n, ok := number(m["max"]); ok {
			s.Max = n
		}
		if n, ok := number(m["rounds"]); ok {
			s.Rounds = n
		}
		if n, ok := number(m["days"]); ok {
			s.Days = &n
		}
		if d, ok := m["drafts"].(string); ok {
			s.Drafts = d
		}
		if l, ok := m["exempt"].([]any); ok {
			s.Exempt = []string{}
			for _, v := range l {
				s.Exempt = append(s.Exempt, fmt.Sprint(v))
			}
		}
		out[kind] = s
	}
	return out
}

// number reads a setting's number, from YAML or JSON.
func number(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	}
	return 0, false
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

// Decide plans the acts proposed, given at their place in the run, as far
// as the config lets the role go (ADR-0026). read lists the issues the run
// read: a proposal made only for a cap is dropped once its issue was read
// again, decided again or not. A run moves at most cfg.MovedPercent of the
// open issues (milestone and order). judged holds the second judge's
// answers on the issues announced obsolete (ADR-0024); changes, what open
// issues were built on that pre found changed, and the issues it read
// again for them (ADR-0032); answers, a person's comments pre read as
// "revise" (ADR-0038).
func Decide(f forge.Backlog, repo, role string, cfg Config, closes map[int]Proposal, read []int, judged map[int]Judged, changes []Change, answers map[int]*Answer) (*Plan, error) {
	settings, movedPercent := cfg.Acts, cfg.MovedPercent
	p := &Plan{read: read, judged: judged, settings: settings, config: cfg, Answers: answers}
	if err := p.readRecord(f, role); err != nil {
		return nil, err
	}
	p.readChanges(changes)
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
		if c.Ticked != "" {
			// A proposal a person of the project accepted, by the label on
			// its issue: the act as its state holds it, checked again
			// against the forge's label.
			var why string
			if c, why = p.tickedAct(c); why != "" {
				dropped(c, "not-accepted", why)
				p.Decisions = append(p.Decisions, Decision{Index: i, Mode: Off, Act: c})
				continue
			}
		}
		if (c.Do == "ready" || c.Do == "keep" || c.Do == "milestone" || c.Do == "order" || c.Do == "split" || c.Do == "rename" || p.ticked[c.key()]) && decided[c.key()] {
			p.Decisions = append(p.Decisions, Decision{Index: i, Mode: Off, Act: c}) // the engine's own and the agent's: one
			if c.Do == "split" || c.Do == "rename" {
				// Both checked against the state before either is applied:
				// a second would split or rename it again.
				dropped(c, "once-a-run", c.Do+": one an issue a run; the first that passed its check is kept")
			}
			continue
		}
		if c.Ticked == "" && p.aside(c.Issue) && c.Do != "keep" && !(c.Do == "undepend" && p.allClosed(c.BlockedBy)) && !(c.Do == "milestone" && c.From != "") {
			// A person took workline:proposed off, without a yes: not now,
			// on that issue, until it changes (ADR-0038).
			dropped(c, "set-aside", "a person took "+p.ledger.States[c.Issue].labelOr()+" off this issue: nothing is proposed nor done on it until it changes")
			p.Decisions = append(p.Decisions, Decision{Index: i, Mode: Off, Act: c})
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
		if c.Ticked != "" {
			decided[c.key()] = true // a person's yes: the agent's act on it is not done again
		}
		d := Decision{Index: i, Mode: Off, Act: c}
		var extra *Decision // a refine's drafts proposed beside what it writes
		switch {
		case why != "":
			dropped(c, mode, why)
		case c.Ticked != "":
			// Their yes: done, whatever its mode, a kind back to propose or a
			// cap — a person decided it, not the role.
			d.Mode = Act
			done[c.Kind()]++
			if slices.Contains(moves, c.Do) {
				moved[c.Issue] = true
			}
			if c.Do == "close" {
				p.Record.Closed = append(p.Record.Closed, Closing{Issue: c.Issue, Act: c.Kind(), Level: p.config.Level})
			}
			if c.Do == "undepend" {
				p.recordUndepend(c)
			}
			p.recordDid(c)
		default:
			s, ok := settings[c.Kind()]
			if !ok {
				s = Setting{Mode: Propose}
			}
			if c.Do == "keep" {
				s = Setting{Mode: Act} // keeping an issue open is never held back
			}
			if c.Do == "unready" {
				s = Setting{Mode: Propose} // a ready issue is moved back by a person only (ADR-0032)
			}
			cleanup := c.Do == "undepend" && p.allClosed(c.BlockedBy)
			switch {
			case cleanup:
				s = Setting{Mode: Act} // its blockers closed: the link's reason is gone, the engine's to take off (ADR-0028)
			case c.Do == "undepend":
				s = Setting{Mode: Propose} // a blocker still open: whether its reason is gone is a person's to say
			}
			d.Mode = s.Mode
			if d.Mode == Act && slices.Contains(p.Record.Propose, c.Kind()) {
				d.Mode = Propose
			}
			if d.Mode == Act && c.Do != "keep" && !cleanup && slices.Contains(p.ledger.UndoneOn(c.Issue), c.Kind()) {
				// A person undid an act of this kind on this issue: proposed
				// here from then on (ADR-0038, amended).
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "undone-here", Level: "info", Where: c.where(),
					Message: fmt.Sprintf("a person undid a %s of the role's on this issue: proposed here, not done", c.Kind())})
			}
			if d.Mode == Act && c.Hearsay {
				// Its only evidence is the word of someone outside the
				// project: proposed, never done alone (ADR-0038).
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "outsider-evidence", Where: c.where(),
					Message: "the words it quotes are only in a comment by someone outside the project: proposed, not done"})
			}
			if ch, ok := p.changedFor[c.Issue]; ok && d.Mode == Act && c.Do != "keep" && !cleanup {
				// Read again for a change to what it was built on: what the
				// change asks of it is a person's to decide (ADR-0032).
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "need-changed", Level: "info", Where: c.where(),
					Message: fmt.Sprintf("read again for %s: %s is proposed, not done", ch.said(), c.Kind())})
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
					Message: fmt.Sprintf("its reporter was written to %d times (acts.ask.rounds): what is left is proposed to a person on the issue", rounds)})
			}
			if d.Mode == Act && c.Announced != "" && s.Max > 0 && done[c.Kind()] >= s.Max {
				// The engine's closing after an announcement: it stays
				// announced, and is closed at a later run.
				d.Mode = Off
				p.Findings = append(p.Findings, verdict.Finding{Rule: "act-cap", Where: c.where(),
					Message: fmt.Sprintf("%s: at most %d a run; it stays announced, closed at a later run", c.Kind(), s.Max)})
				p.Decisions = append(p.Decisions, d)
				continue
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
			if d.Mode == Act && c.Do == "refine" && s.Drafts == Propose {
				// Cautious (ADR-0026): what the code shows is written, the
				// Need and Validation drafts proposed to a person.
				if facts, drafts, ok := p.splitDrafts(c, role); ok {
					if len(facts.Added) == 0 {
						c, d.Mode = drafts, Propose
					} else {
						c, extra = facts, &Decision{Index: -1, Mode: Propose, Act: drafts}
					}
					d.Act = c
					p.Findings = append(p.Findings, verdict.Finding{Rule: "drafts-proposed", Level: "info", Where: c.where(),
						Message: "Need and Validation drafts proposed on the issue, not written (acts.refine.drafts: propose)"})
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
				if c.Do == "close" && !c.Announce {
					p.Record.Closed = append(p.Record.Closed, Closing{Issue: c.Issue, Act: c.Kind(), Level: p.config.Level})
				}
				if c.Do == "depend" {
					p.added[c.Issue] = append(p.added[c.Issue], c.BlockedBy...)
				}
				if c.Do == "undepend" {
					p.recordUndepend(c)
				}
				p.recordDone(c)
				p.recordDid(c)
			}
		}
		p.Decisions = append(p.Decisions, d)
		if extra != nil {
			p.Decisions = append(p.Decisions, *extra)
		}
	}
	p.keepProposed()
	p.summary()
	return p, nil
}

// summary says, in the job's summary, what the run did alone and what it
// proposed, issue by issue (ADR-0038); what waits on a person is a filter
// on workline:proposed.
func (p *Plan) summary() {
	for _, d := range p.Decisions {
		switch {
		case d.Mode == Act && d.Act.Ticked != "":
			p.Findings = append(p.Findings, verdict.Finding{Rule: "done-as-accepted", Level: "info", Where: d.Act.where(),
				Message: strings.TrimSpace(describe(d.Act, "Closed")) + " Accepted by " + d.Act.Ticked + "."})
		case d.Mode == Act:
			p.Findings = append(p.Findings, verdict.Finding{Rule: "done", Level: "info", Where: d.Act.where(),
				Message: strings.TrimSpace(describe(d.Act, "Closed"))})
		case d.Mode == Propose:
			p.Findings = append(p.Findings, verdict.Finding{Rule: "proposed", Level: "info", Where: d.Act.where(),
				Message: strings.TrimSpace(offer(d.Act))})
		}
	}
	for _, id := range sortedKeys(p.SetAside) {
		p.Findings = append(p.Findings, verdict.Finding{Rule: "set-aside", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: "a person took the role's label off, without accepting: nothing is proposed on it until it changes"})
	}
	if p.Report != 0 && p.Moved {
		p.Findings = append(p.Findings, verdict.Finding{Rule: "report-closed", Level: "info", Where: fmt.Sprintf("#%d", p.Report),
			Message: "the old report's record moved to each issue's state; the report is closed: what waits on a person bears " + LabelProposed + " (GitLab: workline::proposed)"})
	}
}

// aside says whether an issue was set aside by a person — workline:proposed
// taken off without a yes — and has not changed since: nothing is proposed
// nor done on it (ADR-0038).
func (p *Plan) aside(id int) bool {
	if p.SetAside[id] {
		return true
	}
	st := p.ledger.States[id]
	return st != nil && st.Aside != "" && st.Aside == BodyDigest(p.issues[id].Body) && p.Answers[id] == nil
}

// labelOr is the label the role set, or its usual name.
func (s *State) labelOr() string {
	if s != nil && s.Label != "" {
		return s.Label
	}
	return LabelProposed
}

// tickedAct is the act an accepted intention stands for: the proposal the
// issue's state holds under its key, its issue bearing workline:accepted,
// as the forge says; or why it is not done.
func (p *Plan) tickedAct(c Proposal) (Proposal, string) {
	key := c.key()
	if !Accepted(p.issues[c.Issue]) {
		return c, "its issue does not bear " + LabelAccepted + ", as the forge says"
	}
	for _, q := range p.Record.Proposed {
		if q.ID() == key && q.Doable() {
			act := *q.Proposal
			act.Ticked = c.Ticked
			p.ticked[key] = true
			return act, ""
		}
	}
	if c.Do == "ready" {
		return c, "" // the drafts accepted: ready, as the engine checks it
	}
	return c, "no such proposal in its issue's state"
}

// keepProposed carries over the acts proposed by earlier runs whose issue
// is still open and not set aside, that this run did not decide again nor
// a person accept, then adds this run's: a proposal stays on its issue
// until a person settles it. An issue to open (an import's, past its cap)
// is not kept: the import finds it again.
func (p *Plan) keepProposed() {
	decided := map[string]bool{}
	for _, d := range p.Decisions {
		if d.Mode != Off {
			decided[d.Act.key()] = true
		}
	}
	// The day each was first proposed: kept while it stays, carried to the
	// same act decided again; one with none gets today's.
	since := map[string]string{}
	for _, q := range p.Record.Proposed {
		since[q.ID()] = q.Since
	}
	var kept []Pending
	for _, q := range p.Record.Proposed {
		if q.Since == "" {
			q.Since = today()
		}
		key := q.ID()
		settled := q.Key != "" || q.Issue <= 0 || !p.open[q.Issue] || p.SetAside[q.Issue]
		switch {
		case p.ticked[key]:
			settled = true // a person's yes, done or said why not
		case Accepted(p.issues[q.Issue]):
			settled = true // a yes to the issue: its proposals are done, or left
		case (q.Capped || q.Undrafted()) && slices.Contains(p.read, q.Issue):
			settled = true // read again: the agent decided it anew, or not at all
		}
		if !settled && !decided[key] {
			kept = append(kept, q)
		}
	}
	for _, d := range p.Decisions {
		if d.Mode == Propose && d.Act.Do != "open" && d.Act.Issue > 0 {
			act := d.Act
			q := Pending{Issue: d.Act.Issue, Act: d.Act.Kind(), Line: describe(d.Act, "Close"), Capped: d.Capped, Proposal: &act}
			if q.Since = since[q.ID()]; q.Since == "" {
				q.Since = today()
			}
			kept = append(kept, q)
		}
	}
	p.Record.Proposed = kept
}

// readRecord reads what the role holds from each open issue's state, and
// an earlier engine's report the first time; it looks for wrong closings
// — an issue the role closed, open again — and the acts a person undid:
// each proposed on its issue from then on, and a kind undone too often
// proposed everywhere (ADR-0038, amended).
func (p *Plan) readRecord(f forge.Backlog, role string) error {
	open, err := f.Issues()
	if err != nil {
		return err
	}
	p.open, p.seen, p.issues, p.ticked, p.added, p.dropped = map[int]bool{}, map[string]bool{}, map[int]forge.Issue{}, map[string]bool{}, map[int][]int{}, map[int][]int{}
	for _, is := range open {
		p.open[is.ID] = true
		p.issues[is.ID] = is
		p.bodies = append(p.bodies, is.Body)
	}
	if p.ledger, err = ReadLedger(f, role, open); err != nil {
		return err
	}
	p.Report = p.ledger.Report
	p.Visit = sortedKeys(p.ledger.States)
	p.SetAside = map[int]bool{}
	for id := range p.ledger.Aside {
		p.SetAside[id] = true
	}
	if p.Report != 0 {
		if err := p.ledger.Broken; err != nil {
			// What the old report recorded cannot be read: it stays open,
			// for a person; nothing of it moves.
			p.Findings = append(p.Findings, verdict.Finding{Rule: "record-broken", Level: "warn", Where: fmt.Sprintf("#%d", p.Report),
				Message: "the old report's record does not read (" + err.Error() + "): it is left open, nothing of it moved to the issues"})
		} else {
			p.Moved = true
		}
	}
	p.Record = p.ledger.Record
	p.Record.Proposed = slices.Clone(p.ledger.Record.Proposed)
	var kept []Closing
	for _, c := range p.Record.Closed {
		if !p.open[c.Issue] {
			kept = append(kept, c)
			continue
		}
		p.Record.Wrong = append(p.Record.Wrong, c)
		p.Findings = append(p.Findings, verdict.Finding{Rule: "wrong-closing", Where: fmt.Sprintf("#%d", c.Issue),
			Message: fmt.Sprintf("closed by the role (%s), open again: %s is proposed on it from now on", c.Act, c.Act)})
	}
	p.Record.Closed = kept
	// Any other act of the role's a person undid: that kind proposed on
	// its issue, as a closing reopened (ADR-0026; ADR-0038, amended).
	p.Record.Done = slices.Clone(p.ledger.Standing)
	for _, u := range p.ledger.Undone {
		if u.Day == "" {
			u.Day = today()
		}
		p.Record.Undone = append(p.Record.Undone, u)
		p.Findings = append(p.Findings, verdict.Finding{Rule: "undone", Where: fmt.Sprintf("#%d", u.Issue),
			Message: fmt.Sprintf("%s: %s is proposed on #%d from now on", u.Evidence, u.Act, u.Issue)})
	}
	p.Record.Propose = demoted(p.Record.Undone, p.Record.Wrong, p.config.UndoneMax)
	for _, kind := range p.Record.Propose {
		p.Findings = append(p.Findings, verdict.Finding{Rule: "demoted", Level: "warn",
			Message: fmt.Sprintf("%s: undone %d times or more across the issues (undone-max): proposed on every issue until the undone acts are fewer", kind, max(p.config.UndoneMax, 1))})
	}
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
	if rule, why := p.checkAgreed(f, role, c); rule != "" {
		return rule, why
	}
	if rule, why := p.checkRefining(f, repo, role, c); rule != "" {
		return rule, why
	}
	if rule, why := p.checkSplitRename(repo, c); rule != "" {
		return rule, why
	}
	if rule, why := p.checkDepend(c); rule != "" {
		return rule, why
	}
	if rule, why := p.checkUndepend(c); rule != "" {
		return rule, why
	}
	if c.Do == "unready" {
		is, ok := p.issues[c.Issue]
		switch {
		case !ok || c.Issue == p.Report:
			return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
		case !slices.Contains(is.Labels, LabelReady):
			return "unready-not-ready", fmt.Sprintf("#%d is not %s: nothing to move back", c.Issue, LabelReady)
		}
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
	switch {
	case !found && slices.Contains(p.read, c.Issue):
		// A person's issue opened since the role's last run, read in this
		// one: pre wrote its first state, applied before any act
		// (ADR-0018, amended).
		st, err = &State{}, nil
	case !found:
		return "no-state", "the issue has no state comment yet: it is not acted on before the engine has one"
	}
	if err != nil {
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
	if c.Do == "ready" && p.config.SpecReview {
		// The reviewer reads the spec first (#128): held while it has not
		// read the body as it is, or an important finding is open.
		if rule, why := SpecHold(p.issues[c.Issue], comments); rule != "" {
			return rule, why
		}
	}
	if c.Do == "ask" || c.ToReporter {
		if rule, why := conversation(role, comments, c); rule != "" {
			return rule, why
		}
	}
	switch {
	case c.Do == "close" && c.Ticked == "" && len(SplitInto(p.issues[c.Issue], st)) > 0:
		return "parent-accepted-by-a-person", fmt.Sprintf("#%d is split into %s: the role never closes a parent; a person accepts it, from what its parts delivered (ADR-0029)", c.Issue, issueList(SplitInto(p.issues[c.Issue], st)))
	case c.Do == "split" && len(st.Split) > 0:
		return "already-split", fmt.Sprintf("it was split already, into %s: an issue is split once", issueList(st.Split))
	case c.Do == "rename" && st.Title != "" && st.Title != p.issues[c.Issue].Title:
		return "title-kept", "its title was set by a person after the role's: it is kept"
	}
	if c.Do == "keep" {
		return p.checkKeep(f, repo, role, st, c)
	}
	if c.Do == "milestone" || c.Do == "order" || c.Do == "refine" || c.Do == "ready" || c.Do == "unready" || c.Do == "ask" || c.Do == "split" || c.Do == "rename" || c.Do == "depend" || c.Do == "undepend" {
		return "", "" // writing a plan or asking says nothing of the issue's truth: no quote
	}
	if c.Quote == nil || strings.TrimSpace(c.Quote.Text) == "" {
		return "no-quote", "no quote: an act cites the code or the issue it rests on"
	}
	p.hearsay = false
	if !p.found(f, repo, *c.Quote) {
		return "no-quote", fmt.Sprintf("the quote %q is not found where it says", c.Quote.Text)
	}
	c.Hearsay = p.hearsay
	if c.Reason == "obsolete" {
		return p.checkObsolete(f, repo, role, st, c)
	}
	return "", ""
}

// obsoleteOn reads where an issue's announcement as obsolete stands, from
// the forge as it is.
func (p *Plan) obsoleteOn(f forge.Backlog, repo, role string, st *State, id int) (Obsolete, error) {
	notes, err := f.Notes(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return Obsolete{}, err
	}
	return ReadObsolete(repo, p.issues[id], notes, st, role, p.settings["close-obsolete"], time.Now()), nil
}

// checkObsolete checks a closing as obsolete (ADR-0024): the agent's is an
// announcement, unless the issue is exempt, announced already, or kept
// open on that evidence; the engine's (Announced) closes only an
// announcement due, unanswered, that the second judge agreed to.
func (p *Plan) checkObsolete(f forge.Backlog, repo, role string, st *State, c *Proposal) (rule, why string) {
	o, err := p.obsoleteOn(f, repo, role, st, c.Issue)
	if err != nil {
		return "no-state", err.Error()
	}
	s := p.settings["close-obsolete"]
	if c.Ticked != "" {
		// A person of the project ticked it in the report: their yes is the
		// judge, it is closed now, not announced (ADR-0025).
		c.Announce, c.Until, c.Announced = false, "", ""
		return "", ""
	}
	if c.Announced == "" {
		switch {
		case ExemptLabel(p.issues[c.Issue], s) != "":
			return "exempt", fmt.Sprintf("it bears the label %s: never announced obsolete", ExemptLabel(p.issues[c.Issue], s))
		case o.Announcement != nil:
			return "announced", fmt.Sprintf("announced obsolete already, on %s: closed or kept open from there", o.Announcement.Announced)
		case slices.Contains(st.Kept, ObsoleteKey(*c.Quote)):
			return "obsolete-kept", "kept open after an announcement resting on this code: not announced again for it"
		}
		c.Announce, c.Until = true, time.Now().AddDate(0, 0, s.Delay()).Format(dateLayout)
		return "", ""
	}
	a := o.Announcement
	switch {
	case a == nil || o.Keep != "":
		return "not-due", "no announcement waiting on it: it is not closed"
	case a.Announced != c.Announced || a.Key() != ObsoleteKey(*c.Quote):
		return "not-due", "not the evidence it was announced with: it is not closed"
	case !o.Due:
		return "not-due", fmt.Sprintf("announced on %s: closed at a run from %s", a.Announced, o.From)
	}
	j, ok := p.judged[c.Issue]
	if !ok || j.Yes == nil || !*j.Yes {
		return "not-judged", "no second judge agreed: it is not closed"
	}
	c.Judge = j.Says()
	return "", ""
}

// checkKeep checks keeping an announced issue open: an announcement must
// be waiting on it.
func (p *Plan) checkKeep(f forge.Backlog, repo, role string, st *State, c *Proposal) (rule, why string) {
	o, err := p.obsoleteOn(f, repo, role, st, c.Issue)
	if err != nil {
		return "no-state", err.Error()
	}
	if o.Announcement == nil {
		return "not-announced", "no announcement as obsolete waits on it"
	}
	if strings.TrimSpace(c.Why) == "" {
		c.Why = o.Keep
	}
	q := o.Announcement.Quote
	c.Quote, c.Announced = &q, o.Announcement.Announced
	return "", ""
}

// checkRefining checks a refine, a ready or an ask against the issue as it
// is: what refining would add, whether it is ready, whether its reporter
// was asked already.
func (p *Plan) checkRefining(f forge.Backlog, repo, role string, c *Proposal) (rule, why string) {
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
		c.ToReporter = c.Agreed == "" && !is.Insider && !Accepted(is) && OpenedBy(is.Body) == ""
		// Answering the reviewer's findings on its spec (#128): the role
		// rewrites the sections they lie in that are still its own.
		c.Revise, c.Refused = nil, nil
		if f != nil {
			comments, err := f.Comments(forge.Target{Kind: "issue", ID: c.Issue})
			if err != nil {
				return "no-state", err.Error()
			}
			st, _, _ := ReadState(comments, role)
			// A section the role wrote and a person deleted is their "no"
			// for it: never written again (ADR-0038).
			c.Refused = DeletedSections(st, is.Body)
			if p.config.SpecReview && !c.ToReporter {
				c.Revise = Revisable(is, st, SpecOpen(is, comments))
			}
			if p.Answers[c.Issue] != nil && !c.ToReporter {
				// A person's comment on its proposal: the role rewrites
				// its own sections, never a person's (ADR-0038).
				for _, name := range Own(is, st) {
					if !slices.Contains(c.Revise, name) {
						c.Revise = append(c.Revise, name)
					}
				}
			}
		}
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
	for i, ch := range c.Into {
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
		for _, k := range ch.After {
			if k < 1 || k > len(c.Into) || k == i+1 {
				return "split-after", fmt.Sprintf("%q: `after` names the other children it waits on, by their place in the split, 1 to %d", ch.Title, len(c.Into))
			}
		}
		for _, s := range ch.Sources {
			path, _, _ := strings.Cut(s, "#")
			if out, err := exec.Command("git", "-C", repo, "cat-file", "-t", "HEAD:"+path).Output(); path == "" || err != nil || strings.TrimSpace(string(out)) != "blob" {
				return "sources-unknown", fmt.Sprintf("%q is not a file in the commit the run is on", path)
			}
		}
	}
	edges := map[int][]int{}
	for i, ch := range c.Into {
		edges[i+1] = ch.After
	}
	for i := range c.Into {
		for _, k := range c.Into[i].After {
			if reaches(edges, k, i+1) {
				return "split-after", fmt.Sprintf("%q and %q wait on each other: a cycle", c.Into[i].Title, c.Into[k-1].Title)
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
		text := Inert(strings.TrimSpace(given[name])) // no quick action run on GitLab when the body is written
		if text == "" || slices.Contains(c.Refused, name) {
			continue
		}
		if slices.Contains(drafted, name) {
			text = DraftLine(role) + "\n\n" + text
		}
		old, ok := have[name]
		switch {
		case ok && strings.TrimSpace(old) != "" && slices.Contains(c.Revise, name) && squeeze(old) != squeeze(text):
			body = fill(body, name, text) // the role's own, rewritten to answer the reviewer (#128)
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
func Accepted(is forge.Issue) bool {
	return slices.ContainsFunc(acceptedLabels, func(l string) bool { return slices.Contains(is.Labels, l) })
}

// ReadyRemoves are the labels an issue moved to ready loses: those of its
// way there, and a person's yes, in either spelling, with the role's
// proposal label (ADR-0038).
func ReadyRemoves(is forge.Issue) []string {
	out := []string{LabelToRefine, LabelDraft, LabelAccepted}
	for _, l := range []string{scopedAccepted, LabelProposed, scopedProposed} {
		if slices.Contains(is.Labels, l) {
			out = append(out, l)
		}
	}
	return out
}

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

// listItem starts a list's item on a line of its own, a sentence of its
// own: an option of one question ("- the CLI or") starts in lower case.
var listItem = regexp.MustCompile(`\n\s*(?:[-*•]|\d+[.)])\s+\p{Lu}`)

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
		// A list's item on a line of its own is a question of its own.
		if m := listItem.FindAllStringIndex(part, -1); len(m) > 0 {
			part = part[m[len(m)-1][0]:]
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
	fmt.Fprintf(&b, "To agree, reply `%s` alone: the sections are then written in your issue at its next run. Or copy them into your issue (edit it), changing what is wrong; or reply what is wrong, and it reads your answer at its next run. A maintainer may agree for the project with the label `%s`: the sections are then written in the issue, and it moves to ready.\n\n", AgreeWord, LabelAccepted)
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

// AgreeWord is the reply that agrees to a text proposed to a reporter: its
// first line, case, spaces and a final "." or "!" aside (ADR-0021).
const AgreeWord = "agreed"

// Agreement is who agreed, in a reply, to the text last proposed to an
// issue's reporter (ADR-0021): after the last round, that round a
// proposal, the last comment of the reporter or of a person of the
// project — never a bot's, nor someone's the forge does not name — has
// AgreeWord as its first line. "" when there is none.
func Agreement(notes []forge.Note, is forge.Issue, role string) string {
	last := -1
	for i, n := range notes {
		if Round(n.Body, role) {
			last = i
		}
	}
	if last < 0 {
		return ""
	}
	if m, _ := EngineMarker(notes[last].Body); !strings.Contains(m, "/proposal=") {
		return "" // the last round asked: a question is not agreed to
	}
	// The last word of those who may agree: a stranger's "+1" or a bot's
	// note after it neither gives nor takes back an agreement; a later
	// word of the reporter's or the project's does.
	var reply *forge.Note
	for i := last + 1; i < len(notes); i++ {
		n := &notes[i]
		if _, engine := EngineMarker(n.Body); engine || n.Bot || n.Author == "" || (n.Author != is.Author && !n.Insider) {
			continue
		}
		reply = n
	}
	if reply == nil {
		return ""
	}
	first, _, _ := strings.Cut(strings.TrimSpace(reply.Body), "\n")
	first = strings.ToLower(strings.TrimSpace(first))
	if first != AgreeWord && first != AgreeWord+"." && first != AgreeWord+"!" {
		return ""
	}
	return reply.Author
}

// checkAgreed checks a refine written on a reply's agreement against the
// forge itself: the agreement there, by the same person, and the sections
// those last proposed. Anything else is not written.
func (p *Plan) checkAgreed(f forge.Backlog, role string, c *Proposal) (rule, why string) {
	if c.Agreed == "" {
		return "", ""
	}
	if c.Do != "refine" {
		c.Agreed = ""
		return "", ""
	}
	is, ok := p.issues[c.Issue]
	if !ok {
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	}
	notes, err := f.Notes(forge.Target{Kind: "issue", ID: c.Issue})
	if err != nil {
		return "no-state", err.Error()
	}
	last := LastProposal(forge.Bodies(notes), role)
	if who := Agreement(notes, is, role); who == "" || who != c.Agreed || last == nil ||
		last.Need != c.Need || last.Verification != c.Verification || last.Validation != c.Validation || last.Scope != c.Scope {
		return "not-agreed", "no reply agreeing to the text last proposed to its reporter: nothing of it is written in the body"
	}
	return "", ""
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
		notes, _ := f.Notes(forge.Target{Kind: "issue", ID: q.Issue})
		where = []string{is.Title, is.Body}
		for _, n := range Counted(notes, *is) {
			where = append(where, n.Body)
		}
		if !slices.ContainsFunc(where, func(w string) bool { return strings.Contains(squeeze(w), want) }) {
			// Only in a comment by someone outside the project, a bot's
			// or a stranger's: their word alone (ADR-0038).
			p.hearsay = slices.ContainsFunc(forge.Bodies(notes), func(w string) bool { return strings.Contains(squeeze(w), want) })
			return p.hearsay
		}
	}
	for _, w := range where {
		if strings.Contains(squeeze(w), want) {
			return true
		}
	}
	return false
}

func squeeze(s string) string { return strings.Join(strings.Fields(s), " ") }

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
	case "unready":
		return fmt.Sprintf("%s #%d back to refine (%s off, %s on): %s", map[bool]string{true: "Moved", false: "Move"}[done], c.Issue, LabelReady, LabelToRefine, strings.TrimSpace(c.Why))
	case "depend":
		return fmt.Sprintf("%s #%d as waiting on %s: %s", map[bool]string{true: "Marked", false: "Mark"}[done], c.Issue, issueList(c.BlockedBy), strings.TrimSpace(c.Why))
	case "undepend":
		return fmt.Sprintf("%s the link the role set from #%d to %s: %s", map[bool]string{true: "Took off", false: "Take off"}[done], c.Issue, issueList(c.BlockedBy), strings.TrimSpace(c.Why))
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
	if c.Do == "keep" {
		return fmt.Sprintf("Kept #%d open, announced obsolete on %s: %s.", c.Issue, c.Announced, strings.TrimSuffix(strings.TrimSpace(c.Why), "."))
	}
	what := fmt.Sprintf("%s #%d as obsolete", verb, c.Issue)
	switch {
	case c.Announce && done:
		return fmt.Sprintf("Announced #%d as obsolete, to be closed at a run from %s if nobody writes on it and a second judge agrees: %s %s", c.Issue, c.Until, cite(*c.Quote), strings.TrimSpace(c.Why))
	case c.Announced != "":
		judge := ""
		if c.Judge != "" {
			judge = " (" + c.Judge + ")"
		}
		what += fmt.Sprintf(", announced on %s: nobody wrote since, a second judge agreed%s", c.Announced, judge)
	}
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
	by := "Closed by the " + role + " role"
	if c.Ticked != "" {
		by += ", " + c.Ticked + " having accepted it (" + LabelAccepted + ")"
	}
	return fmt.Sprintf("%s\n\n%s %s\n\n%s. Reopen it to undo: a closing undone puts this kind of act back to a person.",
		head, cite(*c.Quote), strings.TrimSpace(c.Why), by)
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
	return LocateIn(strings.Split(string(out), "\n"), text)
}

// LocateIn finds a quote in a file's lines, as Locate does.
func LocateIn(lines []string, text string) (from, to int, original string, ok bool) {
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

