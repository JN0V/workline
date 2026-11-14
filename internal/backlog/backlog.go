// Package backlog decides what becomes of a role's acts on a project's
// issues (docs/spec/backlog-acts.md): each is checked against the code and
// the forge, then done, proposed to a person, or dropped. Closing is the
// first act built.
package backlog

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// HiddenLevel is the level an issue's state records for a value the role
// set in a field visible to organization members only, on a public
// repository: set, never shown (ADR-0040).
const HiddenLevel = -1

// axisOf is where the project keeps what an act sets: a priority (order)
// or an effort; false for another act.
func axisOf(c Proposal) (scale.Source, bool) {
	switch c.Do {
	case "order":
		return scale.Used.Priority, true
	case "effort":
		return scale.Used.Effort, true
	}
	return scale.Source{}, false
}

// onceSaid says a level the role set, as say does; one recorded hidden,
// never shown.
func onceSaid(l int, say func(int) string) string {
	if l == HiddenLevel {
		return scale.HiddenSaid
	}
	return say(l)
}

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
	Effort    int      `yaml:"effort,omitempty"`   // the effort the role set, once: another on the issue is a person's (ADR-0040); HiddenLevel, one never shown
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
	// another review, never taken for one answered. Revised are the
	// sections it was given to rewrite then (Revisable): one the rule
	// gives it since is read again once (Unrevised, ADR-0060).
	Wrote    map[string]string `yaml:"wrote,omitempty"`
	Answered string            `yaml:"answered,omitempty"`
	Revised  []string          `yaml:"revised,flow,omitempty"`
	// Plain is the digest of the description the role rewrote in plain
	// words (ADR-0038, amended): another text there is a person's.
	Plain string `yaml:"plain,omitempty"`
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
	// Rules is the role's way of reading an issue when it last read it
	// (Reading): an issue it left with nothing is read again once they
	// change.
	Rules int `yaml:"rules,omitempty"`
	// Left is who wrote last when the role left the issue to a person,
	// its proposal revised as often as allowed: said on the issue while
	// it waits on them (ADR-0059, decision 1).
	Left string `yaml:"left,omitempty"`
	// Proof is the kind the role picked when the tester found no test
	// possible (ADR-0059, decision 4): the developer's proof in place of
	// a Draft, while it answers the tester's last word (ReadProof).
	Proof *Proof `yaml:"proof,omitempty"`
	// NoTest is the tester's "no test possible" the role was last given,
	// by its digest (NoTest): read once, not again until it changes.
	NoTest string `yaml:"no-test,omitempty"`
	// Retest is the tester's "no test possible" the role sent the issue
	// back to its tests unchanged for, by its key (RetestKey: the word and
	// the body it read, #1053): the tester tries again, once (Retested).
	Retest string `yaml:"retest,omitempty"`
	// Silent is why the role read the issue and wrote nothing on it,
	// though it lacks a section: said on the issue until the role acts
	// on it (ADR-0059, decision 1).
	Silent string `yaml:"silent,omitempty"`
	// Unread is the body, by its digest, whose Validation the role was
	// told to rewrite in a form the tester reads (#855): told once, not
	// again until the body changes.
	Unread string `yaml:"unread,omitempty"`
	// Assumed is the round of the question to its reporter the role took
	// an assumption on, unanswered past acts.ask.wait (ADR-0060), and
	// AssumedAt when, RFC 3339: a person's comment after it answers it.
	// Told is the round the agent was told is unanswered past the delay:
	// told once, not again until a new question.
	Assumed   int    `yaml:"assumed,omitempty"`
	AssumedAt string `yaml:"assumed-at,omitempty"`
	Told      int    `yaml:"told,omitempty"`
	// Kind is the kind label the role set, once (ADR-0039 (c)): another
	// kind on the issue, or none, is a person's, never set again.
	Kind string `yaml:"kind,omitempty"`
	// Size is the tier the role granted, once (ADR-0064, §3): its label
	// taken off is a person's, never granted again.
	Size string `yaml:"size,omitempty"`
	// SplitTests: the role predicted this bug past its normal at refine,
	// and readied it for its tests first (#1212): once the tester's
	// Draft holds them, the line asks it its split before the developer.
	SplitTests bool `yaml:"split-after-tests,omitempty"`
	// AnswerHeld is how many comments of a person who counts the issue
	// held when the role's `ready` on it was held (#1331): a person's
	// comment read in that run, nothing written into the spec. Those
	// comments stay unheard, read once more; its next `ready` is taken.
	AnswerHeld int `yaml:"answer-held,omitempty"`
}

// StateMarker marks the comment holding an issue's state.
func StateMarker(role string) string { return forge.Marker("sticky=" + role + "/state") }

// RecordMarker marks the report's comment holding what the role did.
func RecordMarker(role string) string { return forge.Marker("sticky=" + role + "/acts") }

// ReportTitle is the title of the role's report issue.
func ReportTitle(role string) string { return "Backlog — " + strings.ReplaceAll(role, "-", " ") }

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
		Effort    int               `yaml:"effort,omitempty"`
		Title     string            `yaml:"title,omitempty"`
		Split     []int             `yaml:"split,flow,omitempty"`
		After     map[int][]int     `yaml:"after,flow,omitempty"`
		Kept      []string          `yaml:"kept,flow,omitempty"`
		Sections  map[string]string `yaml:"sections,omitempty"`
		Wrote     map[string]string `yaml:"wrote,omitempty"`
		Answered  string            `yaml:"answered,omitempty"`
		Revised   []string          `yaml:"revised,flow,omitempty"`
		Plain     string            `yaml:"plain,omitempty"`
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
		Rules     int               `yaml:"rules,omitempty"`
		Left      string            `yaml:"left,omitempty"`
		Proof     *Proof            `yaml:"proof,omitempty"`
		NoTest    string            `yaml:"no-test,omitempty"`
		Retest    string            `yaml:"retest,omitempty"`
		Silent    string            `yaml:"silent,omitempty"`
		Unread    string            `yaml:"unread,omitempty"`
		Assumed   int               `yaml:"assumed,omitempty"`
		AssumedAt string            `yaml:"assumed-at,omitempty"`
		Told      int               `yaml:"told,omitempty"`
		Kind      string            `yaml:"kind,omitempty"`
		Size      string            `yaml:"size,omitempty"`
		SplitTest bool              `yaml:"split-after-tests,omitempty"`
		Held      int               `yaml:"answer-held,omitempty"`
	}{s.Sources, s.Confirmed, s.Judged, s.Comments, s.Body, s.Priority, s.Effort, s.Title, s.Split, s.After, s.Kept, s.Sections, s.Wrote, s.Answered, s.Revised, s.Plain, s.Deleted,
		s.Proposed, s.Done, s.Undone, s.Did, s.Closed, s.Label, s.Aside, s.Heard, s.Revisions, s.Rules, s.Left, s.Proof, s.NoTest, s.Retest, s.Silent, s.Unread,
		s.Assumed, s.AssumedAt, s.Told, s.Kind, s.Size, s.SplitTests, s.AnswerHeld})
	// The record is kept out of sight, in an HTML comment (#548): a forge
	// mails what a comment shows, and a person reads there only what the
	// role tells them, never its record.
	block := hidden(recordTitle, string(data))
	say := s.say()
	if say == "" {
		return "What workline knows of this issue is kept in this comment, out of sight; edited by the engine, not by hand.\n\n" + block
	}
	return Inert(strings.ReplaceAll(say, "```", "'''")) + "\n\n" + block
}

// recordTitle heads the record in an issue's state comment.
const recordTitle = "What workline knows of this issue, edited by the engine, not by hand."

// hidden is a YAML block in an HTML comment: no forge shows it, nor mails
// it, and the engine reads it back (yamlBlock). Its fence is longer than
// any run of backticks in data, so nothing in it ends the block; what
// would end the HTML comment is escaped, and read back as it was.
func hidden(title, data string) string {
	fence := FenceFor(data)
	return "<!-- " + title + "\n" + fence + "yaml\n" + hide.Replace(data) + fence + "\n-->"
}

// hide escapes what ends an HTML comment, and the escape itself; unhide
// reads it back.
var (
	hide   = strings.NewReplacer("&", "&amp;", "-->", "--&gt;", "--!>", "--!&gt;")
	unhide = strings.NewReplacer("--&gt;", "-->", "--!&gt;", "--!>", "&amp;", "&")
)

// FenceFor is a code fence longer than any run of backticks in text: three,
// or one more than its longest run, so nothing in it closes the block.
func FenceFor(text string) string {
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
	before := strings.TrimSuffix(c[:m[0]], "\n")
	inComment := strings.HasPrefix(before[strings.LastIndex(before, "\n")+1:], "<!-- ") // hidden's
	for at := 0; ; {
		k := strings.Index(rest[at:], fence)
		if k < 0 {
			return "", false
		}
		k += at
		after := k + len(fence)
		if (k == 0 || rest[k-1] == '\n') && (after == len(rest) || rest[after] != '`') {
			if inComment {
				return unhide.Replace(rest[:k]), true
			}
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
	proposed := ForPerson(s.Proposed) // a question held for a later run is not shown
	for _, q := range proposed {
		line := q.Line
		if q.Proposal != nil {
			line = offer(*q.Proposal)
		}
		lines = append(lines, "- **Proposes**: "+line)
	}
	for _, q := range proposed {
		if c := q.Proposal; c != nil && c.Do == "refine" && !c.ToReporter {
			lines = append(lines, fold("The sections it proposes", drafts(*c)))
		}
	}
	switch {
	case s.Aside != "":
		lines = append(lines, "Set aside by a person: nothing more is proposed here until the issue changes.")
	case s.Label != "":
		if len(proposed) == 0 && len(s.Did) == 0 {
			var names []string
			for _, m := range layout() {
				if name, wrote := s.wrote(m); wrote != "" && drafted(m) {
					names = append(names, name)
				}
			}
			what := "sections of this issue"
			if len(names) > 0 {
				what = "the " + and(names) + " of this issue"
			}
			lines = append(lines, "**The product owner** drafted "+what+" from its own words: they wait on your answer.")
		}
		lines = append(lines, s.answers())
		if s.Left != "" {
			lines = append(lines, strings.TrimSpace(Said("The product owner", "leaves this issue to a person",
				fmt.Sprintf("@%s wrote on it, and its proposal was revised %d times already, as often as allowed", s.Left, s.Revisions),
				fmt.Sprintf("a person answers @%s, edits the issue or gives the answer above; the product owner proposes nothing more here until then", s.Left))))
		}
	case strings.HasPrefix(s.Silent, NoTestSilent):
		lines = append(lines, strings.TrimSpace(Said("The product owner", "leaves this issue in refine, for a person", s.Silent,
			"a person of the project names its proof in a comment, `proof: <kind> — why`, or edits its "+work.Name(work.Validation)+
				" so a test checks it; it does not go to its tests again as it stands")))
	case HeldForScenarios(s.Silent):
		lines = append(lines, strings.TrimSpace(scenariosUnreadSaid(s.Silent)))
	case s.Silent != "":
		lines = append(lines, strings.TrimSpace(Said("The product owner", "read this issue and did not refine it", s.Silent,
			"it reads it again when the line wakes it on this issue, or once the issue changes or a person comments: complete it, or name the files it is about in a comment")))
	}
	return strings.Join(lines, "\n\n")
}

// answers is what a person may answer on an issue waiting on them: one
// gesture a line, read on every proposal (ADR-0038).
func (s State) answers() string {
	if proposed := ForPerson(s.Proposed); len(proposed) > 0 && !slices.ContainsFunc(proposed, func(q Pending) bool { return q.Proposal == nil || !q.Proposal.Advice }) {
		// Only what a person does themselves: no yes to give.
		return fmt.Sprintf("**Your answer:**\n\n- close the issue: yes\n- take `%s` off: keep it open as it is\n- comment: tell the role why not", s.Label)
	}
	list := []string{fmt.Sprintf("- label `%s`: yes, done at its next run", strings.Replace(s.Label, "proposed", "accepted", 1))}
	if s.drafts() {
		list = append(list, "- edit a draft: it is yours")
	}
	list = append(list, "- comment: it revises what it proposes", fmt.Sprintf("- take `%s` off: not now", s.Label), "- close the issue: it should not exist")
	return "**Your answer:**\n\n" + strings.Join(list, "\n")
}

// drafts says whether the role drafted on the issue, or proposes to: a
// draft is there for a person to edit.
func (s State) drafts() bool {
	for _, m := range layout() {
		if _, wrote := s.wrote(m); wrote != "" && drafted(m) {
			return true
		}
	}
	return slices.ContainsFunc(ForPerson(s.Proposed), func(q Pending) bool { return q.Proposal != nil && q.Proposal.Do == "refine" })
}

// drafts says the sections a refine would write, each under its name.
func drafts(c Proposal) string {
	var b strings.Builder
	write := func(name, text string) {
		if text = strings.TrimSpace(text); text != "" {
			fmt.Fprintf(&b, "**%s**\n\n%s\n\n", name, strings.ReplaceAll(text, "```", "'''"))
		}
	}
	for _, m := range layout() {
		if len(c.Added) == 0 || work.Has(c.Added, m) {
			write(named(c.Added, m), c.Text(m))
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

// Held says whether a pending act is a question past a run's cap: the
// role's to ask at a later run, its issue read again first, never a
// person's to answer (#430). It adds no label, is not shown on the issue,
// and waits on no one.
func (q Pending) Held() bool { return q.Capped && q.Act == "ask" }

// ForPerson are the acts proposed on an issue that wait on a person: those
// held for a later run left out.
func ForPerson(proposed []Pending) []Pending {
	var out []Pending
	for _, q := range proposed {
		if !q.Held() {
			out = append(out, q)
		}
	}
	return out
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
	Native []int `yaml:"native,flow,omitempty"`
	// Card, the engine's: the move a ready makes on the board, to its
	// ready column (ADR-0057); never the agent's.
	Card *board.Move `yaml:"card,omitempty"`
	// Proof, on a ready: the kind the role picks where the tester found
	// no test possible (ADR-0059, decision 4); the issue goes to ready,
	// not back to its tests.
	Proof *Proof `yaml:"proof,omitempty"`
	// Retest, on a ready: the role sends the issue back to its tests
	// unchanged after the tester's "no test possible", the tester's
	// reason its own (#1053); Retested, the engine's: the word it answers.
	Retest   bool   `yaml:"retest,omitempty"`
	Retested string `yaml:"retested,omitempty"`
	// To, on an unblock: the state a blocked issue goes back to, the one
	// it came from or refine; Person, on a blocked: only a person can
	// lift the block (ADR-0059, amended).
	To     string `yaml:"to,omitempty"`
	Person bool   `yaml:"person,omitempty"`
	Quote  *Quote `yaml:"quote"`
	Why    string `yaml:"why"`
	// Refining: the sections written, Need and Validation as drafts, with
	// a real example of the need or a bug's steps to reproduce, drafts too,
	// never required; Added, the engine's, says which the body did not have
	// yet.
	Scope        string `yaml:"scope,omitempty"`
	Verification string `yaml:"verification,omitempty"`
	Need         string `yaml:"need,omitempty"`
	Example      string `yaml:"example,omitempty"`
	Steps        string `yaml:"steps,omitempty"`
	Validation   string `yaml:"validation,omitempty"`
	// Description, the text above its sections rewritten in plain words by
	// an earlier engine: read in a proposal kept from then, never written —
	// a role's text on top is now taken into the Need (Lift).
	Description string   `yaml:"description,omitempty"`
	Added       []string `yaml:"added,omitempty"`
	// Lift, the engine's: the issue's text on top is a role's no person
	// edited (RolesTop), taken from the top as this refine writes its Need
	// draft, the engine's lines there kept (ADR-0039, amended).
	Lift bool `yaml:"lift,omitempty"`
	// Repeated, the engine's: the drafts not written because they said
	// again what is above them (Repeated), by heading; said on the issue.
	Repeated []string `yaml:"repeated,omitempty"`
	// Revise, the engine's: the sections with text this refine may rewrite,
	// to answer the reviewer's findings on the spec (#128) — those it may
	// rewrite (Revisable); never taken from the agent.
	Revise []string `yaml:"revise,omitempty"`
	// Refused, the engine's: the sections the role wrote that a person
	// deleted (State.Deleted), never written again (ADR-0038).
	Refused   []string `yaml:"refused,omitempty"`
	Questions string   `yaml:"questions,omitempty"` // asking the reporter; for an outsider's refine, what it still needs
	// Assume (ADR-0060): the role's best assumption for a question left
	// unanswered past acts.ask.wait, or asked in the same run at a level
	// that goes on at once; Asked, the engine's, the day the question was
	// asked, "" in the same run. Answered, on the engine's own: who
	// answered after the assumptions, their lines marked so.
	Assumption string `yaml:"assumption,omitempty"`
	Asked      string `yaml:"asked,omitempty"`
	Answered   string `yaml:"answered,omitempty"`
	// Assumed, the engine's, on a ready: the drafts no person accepted,
	// taken as assumptions past acts.ask.wait (ADR-0060), by heading.
	// Written, the engine's: the line an assume, or such a ready, adds
	// under the issue's Assumptions.
	Assumed []string `yaml:"assumed,omitempty"`
	Written string   `yaml:"written,omitempty"`
	// The engine's, for the conversation with the reporter: the round this
	// comment would be, and whether a refine is proposed to the reporter in
	// a comment rather than written in the body (an outsider's issue).
	Round      int  `yaml:"round,omitempty"`
	ToReporter bool `yaml:"to-reporter,omitempty"`
	// LeftOut, the engine's: the questions asked before, taken out of it;
	// said once the act is decided, never when it is dropped.
	LeftOut []string `yaml:"-"`
	// Unquoted, the engine's: the sources it left out of a sources named
	// with no quote, their code given (#1151); said once it is decided.
	Unquoted []string `yaml:"-"`
	// Agreed, the engine's: who agreed in a reply to the text last proposed
	// to the reporter, which this refine writes; checked again on the forge
	// (Agreement), never taken from the agent.
	Agreed string `yaml:"agreed,omitempty"`
	Spent  bool   `yaml:"spent,omitempty"` // the rounds spent: proposed to a person
	// Hearsay, the engine's: the words it quotes are only in a comment by
	// someone outside the project — proposed, never done alone (ADR-0038).
	Hearsay bool `yaml:"hearsay,omitempty"`
	// Own, the engine's: an act pre falls back on, never the agent's.
	Own bool `yaml:"-"`
	// Ordering: the priority set (1 to 4); a milestone left because it is
	// released (From, the engine's); and, the engine's, the issue's
	// priority and milestone before the run, written in the report.
	Priority int    `yaml:"priority,omitempty"`
	From     string `yaml:"from,omitempty"`
	Before   string `yaml:"before,omitempty"`
	// Effort: the effort set (1 to 3, ADR-0040); Had, the engine's, the
	// level the act's axis held as the plan read it: one changed since is
	// a person's, and the act is dropped when applied.
	Effort int `yaml:"effort,omitempty"`
	Had    int `yaml:"had,omitempty"`
	// IssueKind: the kind set, bug, feature or task (ADR-0039 (c));
	// KindLabel, the engine's, the project's label it sets for it.
	IssueKind string `yaml:"kind,omitempty"`
	KindLabel string `yaml:"kind-label,omitempty"`
	// Tier: the size tier granted, large alone (ADR-0064, §3). TooBig,
	// on a split: the issue past its size, measured by a run that
	// crossed it or predicted at ready; split then turns on, off as it
	// is by default, and a part is never split again (§3, §4). On a
	// ready, a bug's predicted past it with no Draft yet: its tests
	// first, then its split asked (State.SplitTests, #1212).
	Tier   string `yaml:"tier,omitempty"`
	TooBig bool   `yaml:"too-big,omitempty"`
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
	// Advice, the engine's: a closing whose kind is off (close-obsolete,
	// by default), said on the issue for a person to close, never done by
	// the role nor by a yes (ADR-0038: off, the code kept).
	Advice bool `yaml:"advice,omitempty"`
	// Parent, the engine's: closing a parent as completed, all its parts
	// closed (ADR-0029, amended): proposed, never done alone; a person's
	// yes (workline:accepted) closes it. No quote: its parts' comment is
	// the evidence.
	Parent bool `yaml:"parent,omitempty"`
	// Symptom, the engine's, on a bug's parent's closing: its
	// reproduction test green on the default branch, as the tester's
	// comment says it (#1212); the parent is then closed alone past
	// cautious, the test named. "" otherwise: proposed, as any parent.
	Symptom string `yaml:"symptom,omitempty"`
	// Ticked, the engine's: who ticked this proposal's box in the report, a
	// person of the project — checked again on the forge, the act done as
	// the record says, never as the intention does (ADR-0025).
	Ticked string `yaml:"ticked,omitempty"`
	// IfRefined, the engine's, on an unblock: done only beside a refine
	// of the same issue done in the run, and no unblock, blocked, close
	// nor split of the agent's on it (#1206). A blocked issue read again
	// whose text the role rewrote, deciding nothing on it, goes back to
	// refine: the read it was blocked on is gone.
	IfRefined bool `yaml:"if-refined,omitempty"`
}

// Child is one part of a split need: an issue of its own, with its
// sections (ADR-0022); its Verification the tester's (ADR-0066), a
// child's given by the product owner left out.
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
var Kinds = []string{"open", "close", "keep", "sources", "milestone", "order", "effort", "kind", "size", "refine", "ready", "unready", "unblock", "blocked", "ask", "assume", "split", "rename", "depend", "undepend"}

// theirs are the acts an outsider's issue is proposed for, not done: it is
// theirs (ADR-0018).
var theirs = map[string]string{"ready": "moving it to ready", "split": "splitting it", "rename": "renaming it"}

// moves are the acts that move an issue in the backlog's order: capped
// together, at a share of the open issues a run (ADR-0018).
var moves = []string{"milestone", "order", "effort"}

// bookkeeping are the acts that file an issue without refining it: one
// left with these alone is said on it, as one left with nothing (#951).
var bookkeeping = []string{"milestone", "order", "effort", "kind", "rename", "depend", "undepend"}

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
	// codeFirst are the issues this run names the code of: no ask on them
	// (#548).
	codeFirst map[int]bool
	// firstNamed are the sources each codeFirst issue was named this
	// run: all given by the task, its Scope written now stands (#1089).
	firstNamed map[int][]string
	// readAgain are the issues whose Scope, written at a run again for
	// the code they named, stands on the code given (scopeRead, #1433).
	readAgain map[int]bool
	// drops are the agent's acts this run dropped on each issue, each with
	// why: said on an issue nothing else refined (#951).
	drops map[int][]string
	// Visit are the open issues with a state, written back at the end of
	// the run (Persist); SetAside, those a person took workline:proposed
	// off; Answers, a person's comments read as "revise" on each.
	Visit    []int           `yaml:"visit,flow,omitempty"`
	SetAside map[int]bool    `yaml:"set-aside,omitempty"`
	Answers  map[int]*Answer `yaml:"answers,omitempty"`
	// ProposalsMax is how many issues may bear the role's label at once:
	// one more waits for room (#428).
	ProposalsMax int `yaml:"proposals-max,omitempty"`
	// Silent are the issues read and left with nothing though they lack
	// a section, each with why: said on the issue (ADR-0059, decision 1).
	Silent map[int]string `yaml:"silent,omitempty"`
	// Unaccepted are the parents a person's yes to close was dropped on,
	// a part open again (checkParent), each with why: the yes taken off,
	// said on the issue, so it never stands for a later act (#1208).
	Unaccepted map[int]string `yaml:"unaccepted,omitempty"`
	// Held are the issues whose `ready` was held for a person's comment
	// read in this run, nothing written into the spec (#1331): the
	// comments the role had heard before it, and those counted now.
	Held map[int][2]int `yaml:"held,omitempty"`
	// Elsewhere are the open issues a run woken for one issue did not
	// visit that bear the role's label (#804): counted among those
	// waiting, from the listing, their records unread.
	Elsewhere     int          `yaml:"elsewhere,omitempty"`
	open          map[int]bool // the open issues, read once
	issues        map[int]forge.Issue
	judged        map[int]Judged // the second judge's answers on the issues announced obsolete
	settings      map[string]Setting
	config        Config
	bodies        []string         // their bodies, to find an import again
	seen          map[string]bool  // the imports this run decided
	read          []int            // the issues this run read
	unread        map[int]string   // the issues a ready was held for, no scenario read: why, said on it
	widenDropped  map[int][]string // the blocked issues whose Scope widening, for the developer's ask, was dropped: the files asked
	narrowDropped map[int]bool     // the blocked issues asked on their size whose Scope narrowing was dropped: an unblock beside it would cross the same
	ledger        *Ledger          // what the role holds, read from the issues
	ticked        map[string]bool  // the proposals a person accepted this run decided, done or dropped: settled
	added         map[int][]int    // the blockers this run's depend acts add, for the next act's cycle check
	dropped       map[int][]int    // the blockers this run's undepend acts take off
	revisited     map[int]string   // the blocked issues this run decided on: unblock or blocked, one a run
	labels        []string         // the project's labels, read once, at the first kind act
	askedNow      map[int]int      // the issues this run asks a question on, each its round: an assumption at once follows it (ADR-0060)
	hearsay       bool             // the last quote found was only in an outsider's comment
	// changedFor are the issues read again for a change to what they were
	// built on: every act on them proposed (ADR-0032).
	changedFor map[int]Change
	// What a refine's text is made readable with (readable): the forge's
	// file and issue pages, the decisions the commit holds, the issues
	// closed — each read once a run, when first needed.
	pages     *[2]string
	decisions map[int]string
	closed    map[int]bool
	// refining are the issues the agent drafts a refine on in this run.
	refining map[int]bool
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

// PathLike is a path a text names: `src/a.go`, `docs/x.md`, a folder
// `internal/doctor`, a file named alone `csv.go`, or a link's address.
var PathLike = regexp.MustCompile(`[\w.-]+(?:/[\w.-]+)+|[\w-]+\.[A-Za-z]{1,5}\b`)

// fileLink is the part of a forge's address that shows a file or a
// folder of the repository: `…/blob/main/`, GitLab's `…/-/tree/HEAD/`.
var fileLink = regexp.MustCompile(`/(?:-/)?(?:blob|tree)/[^/]+/(.+)$`)

// RepoPath is the path of the repository a name stands for: a link's
// address to a file on the forge read as that file, the punctuation
// around it left out.
func RepoPath(p string) string {
	p, _, _ = strings.Cut(p, "#")
	if m := fileLink.FindStringSubmatch(p); m != nil {
		p = m[1]
	}
	p = strings.TrimPrefix(strings.Trim(p, "`'\""), "./")
	return strings.TrimRight(p, ".,;:/")
}

// InCommit says whether the commit the run is on holds path, a file or a
// folder.
func InCommit(repo, path string) bool {
	return path != "" && exec.Command("git", "-C", repo, "cat-file", "-e", "HEAD:"+path).Run() == nil
}

// Ignored are those of paths the repository's ignore rules match,
// as git reads them: a build's outputs, `app/build/…`, none when they
// cannot be read.
func Ignored(repo string, paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	// The repository's rules alone, never a machine's own excludes: the
	// same on a laptop and in CI.
	cmd := exec.Command("git", "-C", repo, "-c", "core.excludesFile=/dev/null", "check-ignore", "-z", "--stdin")
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\x00") + "\x00")
	// Exit 1: none is ignored; any other failure reads as none as well.
	out, _ := cmd.Output()
	var ignored []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			ignored = append(ignored, p)
		}
	}
	return ignored
}

// Tracked are the files of the commit the run is on, nil when they
// cannot be read.
func Tracked(repo string) []string {
	out, err := exec.Command("git", "-C", repo, "ls-tree", "-r", "-z", "--name-only", "HEAD").Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files
}

// PathsIn are the files and folders a text names that the commit holds,
// in the order named, at most maxSources.
func PathsIn(repo, text string) []string {
	var out []string
	for _, m := range PathLike.FindAllString(text, -1) {
		if p := RepoPath(m); strings.Contains(p, "/") && !slices.Contains(out, p) && len(out) < maxSources && InCommit(repo, p) {
			out = append(out, p)
		}
	}
	return out
}

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
	p := &Plan{read: read, judged: judged, settings: settings, config: cfg, Answers: answers, ProposalsMax: cfg.ProposalsMax}
	if err := p.readRecord(f, role); err != nil {
		return nil, err
	}
	p.readChanges(changes)
	p.codeFirst = map[int]bool{}
	p.firstNamed = map[int][]string{}
	p.readAgain = map[int]bool{}
	p.askedNow = map[int]int{}
	p.drops = map[int][]string{}
	p.refining = map[int]bool{}
	for _, c := range closes {
		if c.Do == "refine" && !c.Own && strings.TrimSpace(c.Need+c.Example+c.Steps+c.Validation+c.Scope) != "" {
			p.refining[c.Issue] = true // a section of the spec drafted: not the tester's Verification alone
		}
	}
	dropped := func(c Proposal, rule, msg string) {
		p.Findings = append(p.Findings, verdict.Finding{Rule: rule, Where: c.where(), Message: msg})
		if c.Issue > 0 {
			// Nobody named in it is called: said, never a mention.
			why := strings.ReplaceAll(strings.TrimSuffix(strings.TrimSpace(msg), "."), "@", "")
			p.drops[c.Issue] = append(p.drops[c.Issue], fmt.Sprintf("`%s` dropped: %s", c.Do, why))
		}
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
	// A sources is decided before the asks: an ask waits only for code a
	// sources done in this run names (#548).
	// A blocked issue's unblock or keep after the acts on its text: a
	// narrowing or widening of its Scope is decided first, whatever the
	// agent's order, so one dropped holds the unblock (#1193, #1061).
	// An assumption after the rest: one taken at once follows an ask
	// done in this run (ADR-0060).
	slices.SortStableFunc(indexes, func(a, b int) int {
		first := func(i int) int {
			switch closes[i].Do {
			case "sources":
				return 0
			case "unblock", "blocked":
				return 2
			case "assume":
				return 3
			}
			return 1
		}
		return first(a) - first(b)
	})
	decided := map[string]bool{}
	for _, i := range indexes {
		c := closes[i]
		if c.IfRefined && !p.refinedAlone(c.Issue, closes) {
			continue // its text unchanged, or the agent decided on it: nothing to do
		}
		if c.Ticked != "" {
			// A proposal a person of the project accepted, by the label on
			// its issue: the act as its state holds it, checked again
			// against the forge's label.
			var why string
			if c, why = p.tickedAct(c); why != "" {
				if c.Do == "close" && c.Parent {
					// A yes left on a split parent with no closing
					// proposed (#1213): early, or one to an act done
					// already, it is never one to close it, from what its
					// parts delivered (ADR-0029): taken off, said.
					dropped(c, "parent-not-proposed", why)
					p.unaccept(c, why, "a yes to its closing is asked once all its parts are closed")
					p.Decisions = append(p.Decisions, Decision{Index: i, Mode: Off, Act: c})
					continue
				}
				dropped(c, "not-accepted", why)
				p.Decisions = append(p.Decisions, Decision{Index: i, Mode: Off, Act: c})
				continue
			}
		}
		if (c.Do == "ready" || c.Do == "unready" || c.Do == "unblock" || c.Do == "blocked" || c.Do == "assume" || c.Do == "keep" || c.Do == "milestone" || c.Do == "order" || c.Do == "effort" || c.Do == "kind" || c.Do == "size" || c.Do == "split" || c.Do == "rename" || p.ticked[c.key()]) && decided[c.key()] {
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
		// A narrowing of the Scope of an issue asked on its size (ADR-0064,
		// step 3): one not written holds the unblock beside it (#1193).
		narrowing := c.Do == "refine" && strings.TrimSpace(c.Scope) != "" && p.sizeAsked(c.Issue)
		mode, why := p.check(f, repo, role, &c)
		if narrowing {
			// Dropped until it is done, its Scope written: a refine
			// proposed, or held by a cap, leaves the Scope as it was.
			if p.narrowDropped == nil {
				p.narrowDropped = map[int]bool{}
			}
			p.narrowDropped[c.Issue] = true
		}
		narrowed := func(c Proposal) {
			if narrowing && !c.ToReporter && work.Has(c.Added, work.Scope) {
				p.narrowDropped[c.Issue] = false
			}
		}
		c.Card = nil
		if c.Do == "ready" {
			// Its card to the board's ready column (ADR-0057); to its
			// tests column when it goes to its tests (ADR-0059).
			c.Card = p.config.Ready
			if p.config.Tests != nil && c.Proof == nil && p.config.ToTests != nil && p.config.ToTests(p.issues[c.Issue]) {
				c.Card = p.config.Tests
			}
		}
		if once && why == "" {
			decided[c.key()] = true // the first that passes its check is kept
		}
		if c.Ticked != "" {
			decided[c.key()] = true // a person's yes: the agent's act on it is not done again
		}
		d := Decision{Index: i, Mode: Off, Act: c}
		var extra *Decision // a refine's drafts proposed beside what it writes
		switch {
		case why != "" && c.Ticked != "" && mode == "parent-part-open":
			// Their yes was to close it with all its parts closed: one is
			// open again, so the yes is taken off, said (#1208).
			dropped(c, mode, why)
			p.unaccept(c, why, "its closing is proposed again once all its parts are closed")
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
			narrowed(c)
			p.recordDid(c)
		default:
			s, ok := settings[c.Kind()]
			if !ok {
				s = Setting{Mode: Propose}
			}
			if c.Do == "split" && c.TooBig && s.Mode == Off && !p.config.SplitOff() {
				// Past the issue's size, split turns on (ADR-0064, §3):
				// done at normal and enterprising, proposed at cautious.
				s = Setting{Mode: Act, Max: max(s.Max, 1)}
				if p.config.Level == "cautious" {
					s.Mode = Propose
				}
			}
			if c.Do == "keep" {
				s = Setting{Mode: Act} // keeping an issue open is never held back
			}
			if c.Do == "close" && c.Parent {
				// A parent is accepted by a person (ADR-0029), but a bug's
				// whose reporter's symptom its own test proves gone on the
				// default branch, past cautious (#1212).
				s = Setting{Mode: Propose}
				if c.Symptom != "" && p.config.Level != "cautious" {
					s = Setting{Mode: Act}
				}
			}
			if c.Do == "unready" || c.Do == "unblock" || c.Do == "blocked" || c.Do == "assume" {
				// Back to refine, out of Blocked: cheap, undone by putting
				// the label back; kept blocked, only said — done alone, its
				// reason on the issue (ADR-0059, amended); the rules below
				// still hold it for a person.
				s = Setting{Mode: Act}
			}
			cleanup := c.Do == "undepend" && p.allClosed(c.BlockedBy)
			switch {
			case cleanup:
				s = Setting{Mode: Act} // its blockers closed: the link's reason is gone, the engine's to take off (ADR-0028)
			case c.Do == "undepend":
				s = Setting{Mode: Propose} // a blocker still open: whether its reason is gone is a person's to say
			}
			d.Mode = s.Mode
			if d.Mode == Off && c.Do == "close" && c.Ticked == "" {
				// Its kind is off: what the code shows is said on the issue,
				// for a person to close; the role closes nothing.
				c.Advice, c.Announce, c.Until = true, false, ""
				d.Mode, d.Act = Propose, c
			}
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
			if ch, ok := p.changedFor[c.Issue]; ok && d.Mode == Act && c.Do != "keep" && c.Do != "unready" && c.Do != "unblock" && c.Do != "blocked" && c.Do != "assume" && !cleanup {
				// Read again for a change to what it was built on: what the
				// change asks of it is a person's to decide (ADR-0032); back to
				// refine excepted, cheap to undo (ADR-0059, amended).
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "need-changed", Level: "info", Where: c.where(),
					Message: fmt.Sprintf("read again for %s: %s is proposed, not done", ch.said(), c.Kind())})
			}
			if what, ok := theirs[c.Do]; ok && d.Mode == Act && Outsider(p.issues[c.Issue]) && !Accepted(p.issues[c.Issue]) {
				// The reporter has no write access: the issue is theirs
				// (ADR-0018), the act is for a person to decide. A role's
				// finding, or a bot's issue — a split's children the
				// line's App opened (#1195) —, is the project's (#429).
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
			if d.Mode == Act && s.Max > 0 && done[c.Kind()] >= s.Max && c.Do == "ask" {
				// A question past the cap is the role's to ask at a later
				// run, never a person's to answer (#430): held, its issue
				// read again first; no label, nothing shown on it.
				d.Mode, d.Capped = Off, true
				p.Findings = append(p.Findings, verdict.Finding{Rule: "act-cap", Where: c.where(),
					Message: fmt.Sprintf("past this run's limit of %d (ask): asked at a later run, its issue read again first", s.Max)})
				p.Decisions = append(p.Decisions, d)
				continue
			}
			if d.Mode == Act && s.Max > 0 && done[c.Kind()] >= s.Max {
				d.Mode, d.Capped = Propose, c.From == ""
				p.Findings = append(p.Findings, verdict.Finding{Rule: "act-cap", Where: c.where(),
					Message: fmt.Sprintf("past this run's limit of %d (%s): proposed instead of done", s.Max, c.Kind())})
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
						Message: fmt.Sprintf("its %s and %s are proposed on the issue, not written in it: the role is set to propose them (autonomy cautious, or acts.refine.drafts)", work.Name(work.Need), work.Name(work.Validation))})
				}
			}
			if src, ok := axisOf(c); ok && src.Hidden() && d.Mode == Propose {
				// A proposal shows its value on the issue, and its state
				// keeps it: one visible to organization members only is
				// never proposed on a public repository (ADR-0040).
				d.Mode, d.Capped = Off, false
				p.Findings = append(p.Findings, verdict.Finding{Rule: src.Family.Scale.Axis + "-hidden", Where: c.where(),
					Message: "proposing it would show on the issue a value visible to organization members only: nothing proposed; turn the act to `act`, or set it yourself"})
			}
			if d.Mode == Act {
				done[c.Kind()]++
				narrowed(c)
				if c.Do == "refine" && !c.ToReporter {
					// A ready later in the run reads the body as this leaves it.
					is := p.issues[c.Issue]
					is.Body, _, _ = Refine(is.Body, c, role)
					p.issues[c.Issue] = is
					p.recordDrafted(c)
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
				if c.Do == "sources" && c.Ticked == "" {
					p.codeFirst[c.Issue] = true // its code read at the next run: no ask on it now
					p.firstNamed[c.Issue] = append(p.firstNamed[c.Issue], c.Sources...)
				}
				if c.Do == "ask" {
					p.askedNow[c.Issue] = c.Round
				}
				p.recordDone(c)
				p.recordDid(c)
			}
		}
		if d.Mode != Off && len(c.LeftOut) > 0 {
			// Written or proposed: the question asked before left out is
			// said; an act dropped writes nothing, and says only why.
			p.Findings = append(p.Findings, verdict.Finding{Rule: "question-left-out", Level: "info", Where: c.where(), Message: leftOut(c.LeftOut)})
		}
		if d.Mode != Off && len(c.Unquoted) > 0 {
			p.Findings = append(p.Findings, verdict.Finding{Rule: "sources-unquoted", Level: "info", Where: c.where(),
				Message: fmt.Sprintf("Named without a quote, so left out of its sources: %s, whose code was given; the rest, not given, is read at the next run.", strings.Join(c.Unquoted, ", "))})
		}
		p.Decisions = append(p.Decisions, d)
		if extra != nil {
			p.Decisions = append(p.Decisions, *extra)
		}
	}
	p.keepProposed()
	p.summary(role)
	return p, nil
}

// summary says, in the job's summary, what the run did alone and what it
// proposed, issue by issue (ADR-0038); what waits on a person is a filter
// on workline:proposed.
func (p *Plan) summary(role string) {
	for _, d := range p.Decisions {
		switch {
		case d.Mode == Act && d.Act.Ticked != "":
			p.Findings = append(p.Findings, verdict.Finding{Rule: "done-as-accepted", Level: "info", Where: d.Act.where(),
				Message: strings.TrimSpace(describe(d.Act, "Closed")) + " Accepted by " + d.Act.Ticked + "."})
		case d.Mode == Act:
			p.Findings = append(p.Findings, verdict.Finding{Rule: "done", Level: "info", Where: d.Act.where(),
				Message: strings.TrimSpace(describe(d.Act, "Closed"))})
			if d.Act.Do == "order" && d.Act.Priority == 1 && scale.Used.Detected && !scale.Used.Priority.Hidden() {
				// Urgent is set like any level, said apart for a person to
				// see at once (ADR-0040).
				p.Findings = append(p.Findings, verdict.Finding{Rule: "urgent-set", Level: "info", Where: d.Act.where(),
					Message: p.issues[d.Act.Issue].Title})
			}
		case d.Mode == Propose:
			p.Findings = append(p.Findings, verdict.Finding{Rule: "proposed", Level: "info", Where: d.Act.where(),
				Message: strings.TrimSpace(offer(d.Act))})
		}
	}
	// The code an issue is about named in this run, its Scope or question
	// left for that code (codeFirst): the agent lacked it, a needs-input
	// of the engine (ADR-0063, §6), which reads it at the run that
	// follows. None for an issue this run readied, closed or split.
	// A Scope written from code the task gave, the sources naming only
	// that code, settles it too: nothing is left to read (#1089).
	named, settled := map[int][]string{}, map[int]bool{}
	for _, d := range p.Decisions {
		switch {
		case d.Mode != Act:
		case d.Act.Do == "sources" && d.Act.Ticked == "" && p.codeFirst[d.Act.Issue] && p.given(d.Act.Issue, d.Act.Sources):
			// Only files the task gave: naming them records what the
			// issue is about, nothing is left to read (#1440).
		case d.Act.Do == "sources" && d.Act.Ticked == "" && p.codeFirst[d.Act.Issue]:
			named[d.Act.Issue] = append(named[d.Act.Issue], d.Act.Sources...)
		case d.Act.Do == "ready" || d.Act.Do == "close" || d.Act.Do == "split":
			settled[d.Act.Issue] = true
		case d.Act.Do == "refine" && strings.TrimSpace(d.Act.Scope) != "" && (p.given(d.Act.Issue, p.firstNamed[d.Act.Issue]) || p.readAgain[d.Act.Issue] && p.covered(d.Act.Issue, p.firstNamed[d.Act.Issue])):
			settled[d.Act.Issue] = true
		}
	}
	for _, id := range sortedKeys(named) {
		if !settled[id] && len(named[id]) > 0 {
			p.Findings = append(p.Findings, verdict.Finding{Rule: verdict.NeedsCode, Level: "info", Where: fmt.Sprintf("#%d", id),
				Message: strings.Join(named[id], ", ")})
		}
	}
	// An issue read that lacks a section and got nothing at all — no
	// draft, no question, no act dropped with its reason —: said, never
	// left in silence.
	acted, moved, refined := map[int]bool{}, map[int]bool{}, map[int]bool{}
	for _, d := range p.Decisions {
		acted[d.Act.Issue] = true
		moved[d.Act.Issue] = moved[d.Act.Issue] || d.Mode != Off
		// A kind, an order, a link or a title is no step of its refining:
		// one with nothing else left it as it was (#951).
		refined[d.Act.Issue] = refined[d.Act.Issue] || d.Mode != Off && !slices.Contains(bookkeeping, d.Act.Do)
	}
	for _, id := range p.read {
		// The tester found no test possible, and the run picked no proof
		// nor rewrote it (#815): it waits on a person, said on it, not
		// moved to its tests again unchanged.
		is, ok := p.issues[id]
		if !ok || moved[id] || p.aside(id) {
			continue
		}
		if reason, _, _ := NoTest(forge.Bodies(p.ledger.Notes[id])); NoTestStands(is, forge.Bodies(p.ledger.Notes[id]), role) {
			p.Findings = append(p.Findings, verdict.Finding{Rule: "no-test-waits", Level: "info", Where: fmt.Sprintf("#%d", id),
				Message: "the tester found no test possible, and no proof was picked nor its text rewritten: it waits in refine for a person"})
			if p.Silent == nil {
				p.Silent = map[int]string{}
			}
			p.Silent[id] = NoTestSilent + strings.Join(strings.Fields(reason), " ")
		}
	}
	for _, id := range sortedKeys(p.unread) {
		// A ready held, no scenario the tester reads (#855): said on it,
		// unless something else moved it in this run.
		if _, said := p.Silent[id]; said || moved[id] || p.aside(id) {
			continue
		}
		if p.Silent == nil {
			p.Silent = map[int]string{}
		}
		p.Silent[id] = p.unread[id]
	}
	for _, id := range p.read {
		is, ok := p.issues[id]
		if _, said := p.Silent[id]; said {
			continue
		}
		if !ok || refined[id] || acted[id] && !moved[id] && len(p.drops[id]) == 0 || p.aside(id) {
			continue
		}
		if p.broughtBack(id) {
			// Their comment asked for nothing more, or the agent proposed
			// nothing: the issue stays as they left it.
			p.Findings = append(p.Findings, verdict.Finding{Rule: "set-aside", Level: "info", Where: fmt.Sprintf("#%d", id),
				Message: "a person's comment brought it back and nothing was proposed on it (the agent's note says why): it stays set aside until it changes or someone comments"})
			continue
		}
		if lacks := missingSections(is.Body); len(lacks) > 0 {
			p.Findings = append(p.Findings, verdict.Finding{Rule: "nothing-proposed", Level: "info", Where: fmt.Sprintf("#%d", id),
				Message: fmt.Sprintf("read, and nothing was drafted or asked though it lacks %s: read again once it changes or a person comments", strings.Join(lacks, ", "))})
			if p.Silent == nil {
				p.Silent = map[int]string{}
			}
			p.Silent[id] = silentWhy(id, lacks, p.drops[id], p.config.Notes, len(p.read))
		} else if unread, own := ScenariosUnread(is, p.ledger.States[id]); unread && p.config.Tester && !Readied(is) {
			// Complete, but its Validation names no scenario the tester
			// reads, and nothing rewritten nor asked (#1036): held, said
			// on it, never left in refine with no word.
			p.Findings = append(p.Findings, verdict.Finding{Rule: "nothing-proposed", Level: "info", Where: fmt.Sprintf("#%d", id),
				Message: "read, and nothing was rewritten or asked though " + scenariosWhat(is) + ": it stays in refine, said on it"})
			if p.Silent == nil {
				p.Silent = map[int]string{}
			}
			p.Silent[id] = scenariosUnreadWhy(is, own)
		}
	}
	// An issue read in full that no kind tells, and the answer gave it
	// none (#895): said, never silent; the light read gives it one.
	if s, ok := p.settings["kind"]; !ok || s.Mode != Off {
		kinded, closed := map[int]bool{}, p.closedInRun()
		for _, d := range p.Decisions {
			if d.Act.Do == "kind" {
				kinded[d.Act.Issue] = true // done, proposed or dropped: said either way
			}
		}
		for _, id := range p.read {
			is, ok := p.issues[id]
			if !ok || kinded[id] || closed[id] || p.aside(id) {
				continue
			}
			if k, _ := KindOf(is.Labels); k != "" {
				continue
			}
			if st := p.ledger.States[id]; st != nil && st.Kind != "" {
				continue // set once; a person's change or removal stands
			}
			p.Findings = append(p.Findings, verdict.Finding{Rule: "kind-missing", Level: "warn", Where: fmt.Sprintf("#%d", id),
				Message: "read in full, and the agent gave it no kind: read for its kind alone, by its title and body, at a later scheduled run"})
		}
	}
	for _, id := range sortedKeys(p.SetAside) {
		p.Findings = append(p.Findings, verdict.Finding{Rule: "set-aside", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: p.ledger.States[id].labelOr() + " taken off without a yes: nothing more is proposed until it changes or someone comments"})
	}
	if p.Report != 0 && p.Moved {
		p.Findings = append(p.Findings, verdict.Finding{Rule: "report-closed", Level: "info", Where: fmt.Sprintf("#%d", p.Report),
			Message: "the old report's record moved to each issue's state; the report is closed: what waits on a person bears " + LabelProposed + " (GitLab: workline::proposed)"})
	}
}

// Missing names the required sections a body lacks, or has empty: the
// line's word on an issue left unrefined (#951).
func Missing(body string) []string { return missingSections(body) }

// missingSections names the required sections a body lacks, or has empty.
func missingSections(body string) []string {
	var lacks []string
	have := work.ByMeaning(body)
	for _, m := range work.Required {
		if strings.TrimSpace(have[m].Text) == "" {
			lacks = append(lacks, work.Name(m))
		}
	}
	return lacks
}

// silentMax bounds the agent's note said on an issue, in characters.
const silentMax = 600

// silentWhy is why an issue read was left with nothing: the sections it
// lacks, and the agent's note on it — one naming the issue, or any when
// the run read it alone —, its whitespace folded, clipped.
func silentWhy(id int, lacks, drops, notes []string, read int) string {
	why := fmt.Sprintf("it lacks %s, and the agent drafted nothing nor asked", and(lacks))
	if len(drops) > 0 {
		why += "; " + strings.Join(drops, "; ")
	}
	names := regexp.MustCompile(fmt.Sprintf(`#%d\b`, id))
	var said []string
	for _, n := range notes {
		if n = strings.Join(strings.Fields(n), " "); n != "" && (read == 1 || names.MatchString(n)) {
			said = append(said, n)
		}
	}
	if note := strings.Join(said, " "); note != "" {
		if r := []rune(note); len(r) > silentMax {
			note = string(r[:silentMax]) + "…"
		}
		why += "; its note: " + note
	}
	return why
}

// aside says whether an issue was set aside by a person — workline:proposed
// taken off without a yes — and has not changed since, its body nor a
// person's word on it: nothing is proposed nor done on it (ADR-0038). One
// the run read is not: read while aside, a comment brought it back under
// an older way of reading (BroughtBack) — pre reads no other.
func (p *Plan) aside(id int) bool {
	if p.SetAside[id] {
		return true
	}
	return StillAside(p.ledger.States[id], p.issues[id], p.ledger.Notes[id]) && p.Answers[id] == nil && !slices.Contains(p.read, id)
}

// broughtBack says whether an issue set aside was read only for a
// person's comment, its body as it was: nothing decided on it, it stays
// set aside.
func (p *Plan) broughtBack(id int) bool {
	st := p.ledger.States[id]
	return st != nil && st.Aside != "" && st.Aside == BodyDigest(p.issues[id].Body) && slices.Contains(p.read, id)
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
	if c.Do == "close" && c.Parent {
		return c, fmt.Sprintf("no closing of #%d is proposed, and a yes left on a parent never stands for one", c.Issue)
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
		held := d.Mode == Off && d.Capped && d.Act.Do == "ask" // asked at a later run (Held)
		if (d.Mode == Propose || held) && d.Act.Do != "open" && d.Act.Issue > 0 {
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
	// Where the project keeps each axis, as its settings ask (ADR-0040):
	// the values read and checked there; pre said it once.
	scheme, _, err := Where(f, p.config, open)
	if err != nil {
		return err
	}
	scale.Use(scheme)
	var scope []int // a run an event woke for one issue reads its record alone (#804)
	if len(p.config.Scope) > 0 {
		scope = p.config.Scope
	}
	if p.ledger, err = ReadLedger(f, role, open, scope); err != nil {
		return err
	}
	for _, is := range open {
		if scope != nil && !slices.Contains(scope, is.ID) && slices.Contains(is.Labels, ProposedLabel(f)) {
			p.Elsewhere++
		}
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
			Message: fmt.Sprintf("%s: undone %d times or more by people, across the issues: proposed on every issue, never done alone, until fewer stand (undone-max)", kind, max(p.config.UndoneMax, 1))})
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
	if is, ok := p.issues[c.Issue]; ok && c.Issue != p.Report && Tracking(is, role) {
		// The weekly sample, a role's issue kept in place: workline's own
		// work, never the backlog's (#429).
		return "workline-issue", fmt.Sprintf("#%d is one workline keeps for its own work: nothing is done nor proposed on it", c.Issue)
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
	if rule, why := p.checkBlocked(c); rule != "" {
		return rule, why
	}
	if rule, why := p.checkAssume(f, role, c); rule != "" {
		return rule, why
	}
	if c.Do == "unready" {
		is, ok := p.issues[c.Issue]
		switch {
		case !ok || c.Issue == p.Report:
			return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
		case !Readied(is):
			return "unready-not-ready", fmt.Sprintf("#%d is not %s: nothing to move back", c.Issue, LabelReady)
		}
	}
	if c.Do == "milestone" || c.Do == "order" || c.Do == "effort" {
		// Ordering says nothing of an issue's truth: no quote, its state
		// readable all the same.
		is, ok := p.issues[c.Issue]
		if !ok {
			return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
		}
		c.Before = Before(is)
		switch {
		case c.Do == "effort":
			// An effort is estimated only on code this run read (ADR-0040),
			// where the project keeps it.
			c.Had, c.Before = Effort(is), scale.EffortValue(Effort(is))
			switch {
			case !scale.Effort.Valid(c.Effort):
				return "effort-level", fmt.Sprintf("an effort is a level from 1, the lightest (hours), to %d (weeks)", scale.Effort.Count())
			case !scale.Used.Detected || scale.Used.Effort.Said == "":
				return "effort-nowhere", "the setting `effort` is unset: where the project keeps an effort is not known"
			case c.Ticked == "" && !slices.Contains(p.config.CodeRead, c.Issue):
				return "effort-unread", "its code was not read in this run: an effort is estimated only on the code it names"
			}
			if rule, why := Placeable(scale.Used.Effort, is, c.Effort); rule != "" {
				return rule, why
			}
		case c.Do == "order" && !scale.Priority.Valid(c.Priority):
			return "priority-level", fmt.Sprintf("a priority is a level from 1, the most pressing, to %d", scale.Priority.Count())
		case c.Do == "order":
			c.Had = Priority(is)
			if rule, why := Placeable(scale.Used.Priority, is, c.Priority); rule != "" {
				return rule, why
			}
		case c.From != "" && (c.From != is.Milestone || !Released(repo, c.From)):
			return "not-slipped", fmt.Sprintf("#%d is not in the released milestone %q", c.Issue, c.From)
		case c.From != "" && c.Milestone == "":
			// slipped, no milestone to move it to: proposed
		default:
			if t := strings.TrimSpace(c.Milestone); t == "" || len(t) > 60 || strings.ContainsAny(t, "\n") {
				return "milestone-title", "a milestone is a title of one line, 60 characters at most"
			}
		}
	} else if c.Do == "kind" {
		if rule, why := p.checkKind(f, c, nil); rule != "" {
			return rule, why
		}
	} else if c.Do == "size" {
		if rule, why := p.checkSize(c, nil); rule != "" {
			return rule, why
		}
	} else if c.Do == "sources" {
		if rule, why := checkSources(repo, *c); rule != "" {
			return rule, why
		}
		if st := p.ledger.States[c.Issue]; st != nil && len(st.Sources) > 0 && slices.Equal(st.Sources, c.Sources) {
			// The code it names already, given in this run: named again,
			// it would only be read again, unrefined (#951).
			return "sources-same", fmt.Sprintf("#%d names %s already, its code given in this run: refine it from that code, or ask", c.Issue, strings.Join(c.Sources, ", "))
		}
	} else if c.Do == "close" && !slices.Contains(closeReasons, c.Reason) && (!c.Parent || c.Reason != "completed") {
		return "close-reason", fmt.Sprintf("not closed: the role closes only a duplicate, or an issue the code made obsolete; closing one as %q is a person's call", c.Reason)
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
		case scale.Used.Detected && st.Priority != 0:
			// Set once, never estimated again, so Next does not move on a
			// night its issues did not (ADR-0040); a person's change or
			// clearing since stands.
			return "priority-once", fmt.Sprintf("the role set its priority once already (%s): never estimated again; a person's change or clearing stands", onceSaid(st.Priority, scale.Said))
		case !(len(set) == 0 && st.Priority == 0) && !(len(set) == 1 && set[0] == st.Priority):
			return "priority-kept", "its priority was set by a person (a value the role did not set, or took off): it is kept"
		case len(set) == 1 && set[0] == c.Priority:
			return "priority-same", fmt.Sprintf("it has %s already", scale.Said(c.Priority))
		}
	}
	if c.Do == "effort" {
		// Only an empty value is written, once: a person's value, or the
		// role's they changed or cleared, stands (ADR-0040).
		switch set := EffortLabels(p.issues[c.Issue]); {
		case st.Effort != 0:
			return "effort-once", fmt.Sprintf("the role estimated its effort once already (%s): never again; a person's change or clearing stands", onceSaid(st.Effort, scale.Effort.Name))
		case len(set) > 0:
			return "effort-kept", "its effort was set by a person: it is kept"
		}
	}
	if c.Do == "kind" {
		if rule, why := p.checkKind(f, c, st); rule != "" {
			return rule, why
		}
	}
	if c.Do == "size" {
		if rule, why := p.checkSize(c, st); rule != "" {
			return rule, why
		}
	}
	if c.Do == "ready" && p.config.SpecReview {
		// The reviewer reads the spec first (#128): held while it has not
		// read the body as it is, or an important finding is open.
		if rule, why := SpecHold(p.issues[c.Issue], comments); rule != "" {
			return rule, why
		}
	}
	if author := p.issues[c.Issue].Author; c.Do == "ask" && Bot(author) {
		// Nobody answers a bot's issue: what it lacks is the project's to
		// say (#429).
		return "reporter-bot", fmt.Sprintf("opened by a bot, @%s: nobody to ask", author)
	}
	if c.Do == "ask" || c.ToReporter {
		rule, why, left := conversation(role, comments, c)
		if rule != "" {
			return rule, why
		}
		// Only the question asked before is dropped, never the act: its
		// drafts, its other questions written — said once it is decided.
		c.LeftOut = left
	}
	switch {
	case c.Do == "close" && c.Ticked == "" && !c.Parent && len(SplitInto(p.issues[c.Issue], st)) > 0:
		return "parent-accepted-by-a-person", fmt.Sprintf("#%d is split into %s: the role never closes a parent; a person accepts it, from what its parts delivered (ADR-0029)", c.Issue, issueList(SplitInto(p.issues[c.Issue], st)))
	case c.Do == "split" && len(st.Split) > 0:
		return "already-split", fmt.Sprintf("it was split already, into %s: an issue is split once", issueList(st.Split))
	case c.Do == "rename" && st.Title != "" && st.Title != p.issues[c.Issue].Title:
		return "title-kept", "its title was set by a person after the role's: it is kept"
	}
	if c.Do == "keep" {
		return p.checkKeep(f, repo, role, st, c)
	}
	if c.Do == "close" && c.Parent {
		if rule, why := p.checkParent(c.Issue, st); rule != "" {
			return rule, why // its parts' comment is the evidence, a person's yes the act (ADR-0029, amended)
		}
		if c.Ticked == "" {
			p.bugParent(c, st)
		}
		return "", ""
	}
	if c.Do == "milestone" || c.Do == "order" || c.Do == "effort" || c.Do == "kind" || c.Do == "size" || c.Do == "refine" || c.Do == "ready" || c.Do == "unready" || c.Do == "unblock" || c.Do == "blocked" || c.Do == "ask" || c.Do == "assume" || c.Do == "split" || c.Do == "rename" || c.Do == "depend" || c.Do == "undepend" {
		return "", "" // writing a plan or asking says nothing of the issue's truth: no quote
	}
	if c.Do == "sources" && (c.Quote == nil || strings.TrimSpace(c.Quote.Text) == "") {
		// Named outside the code given, nothing of it to quote (#951,
		// #980): the issue is read again with that code, at the next run.
		// Each such file on its own (#1151): one its state names already
		// stays; one shown, new to it, wants its quote, and is left out
		// without one.
		if unshown, shown := p.unshown(repo, c.Issue, c.Sources); len(unshown) > 0 {
			var had []string
			if st := p.ledger.States[c.Issue]; st != nil {
				had = st.Sources
			}
			kept := slices.DeleteFunc(slices.Clone(c.Sources), func(s string) bool {
				return slices.Contains(shown, s) && !slices.ContainsFunc(had, func(h string) bool { return RepoPath(h) == RepoPath(s) })
			})
			c.Unquoted = slices.DeleteFunc(slices.Clone(shown), func(s string) bool { return slices.Contains(kept, s) })
			c.Quote, c.Sources = nil, kept
			return "", ""
		}
	}
	if c.Quote == nil || strings.TrimSpace(c.Quote.Text) == "" {
		return "no-quote", "nothing done: no evidence quoted from the code or an issue"
	}
	p.hearsay = false
	if !p.found(f, repo, *c.Quote) {
		return "no-quote", fmt.Sprintf("nothing done: the words quoted as evidence, %q, are not where it says", c.Quote.Text)
	}
	c.Hearsay = p.hearsay
	if c.Reason == "obsolete" {
		return p.checkObsolete(f, repo, role, st, c)
	}
	return "", ""
}

// checkParent holds a closing marked parent to what the mark claims, at
// its proposal and again at a person's yes (ADR-0029, amended): the
// issue was split — parts in the forge's relation, its task list or the
// role's split — and every part is closed now, none open again.
func (p *Plan) checkParent(id int, st *State) (rule, why string) {
	parts := SplitInto(p.issues[id], st)
	if len(parts) == 0 {
		return "parent-not-split", fmt.Sprintf("#%d has no parts: a closing for its parts closed is only a split parent's", id)
	}
	var open []int
	for _, n := range parts {
		if p.open[n] {
			open = append(open, n)
		}
	}
	if len(open) > 0 {
		return "parent-part-open", fmt.Sprintf("#%d's part %s is open: it is closed once all its parts are, a person's yes then", id, issueList(open))
	}
	return "", ""
}

// bugParent reads a bug's parent's closing against the tester's proof of
// its reporter's symptom (#1212): green on the default branch once all
// its parts were closed, c.Symptom says it, the closing done alone past
// cautious; otherwise why it stays a person's is added to c.Why. A
// feature's or a task's parent is left as it is: proposed.
func (p *Plan) bugParent(c *Proposal, st *State) {
	c.Symptom = ""
	if kind, _ := KindOf(p.issues[c.Issue].Labels); kind != KindBug {
		return
	}
	why := "no reproduction test of its own was run on the default branch: the tester writes it on the bug's Draft before its split"
	if p.config.Symptom != nil {
		s, found, err := p.config.Symptom(c.Issue)
		switch {
		case err != nil:
			why = "the tester's proof of its symptom could not be read (" + err.Error() + ")"
		case found:
			if why = s.Proves(SplitInto(p.issues[c.Issue], st)); why == "" {
				c.Symptom = s.Said()
				return
			}
		}
	}
	c.Why = strings.TrimSuffix(strings.TrimSpace(c.Why), ".") + "; its closing is a person's: " + why
}

// splitAfterTests says why a ready's `too-big` does not hold on is, ""
// when it does (#1212): a bug the run was told to size in refine, which
// may be split, its tests not yet written. Readied so, it goes to its
// tests, and the line asks its split once they are.
func (p *Plan) splitAfterTests(is forge.Issue, c Proposal) string {
	if kind, _ := KindOf(is.Labels); kind != KindBug || p.config.Tested == nil {
		return "only a bug goes to its tests before its split, where the line runs the tester"
	}
	if c.Issue != p.config.Sized || !slices.Contains(is.Labels, LabelToRefine) {
		return fmt.Sprintf("#%d's size was not asked in this run, in refine", c.Issue)
	}
	if p.config.SplitOff() {
		return "this project allows no splits"
	}
	if partOf(is.Body) > 0 {
		return "a part of a split is never split again"
	}
	tested, err := p.config.Tested(c.Issue)
	switch {
	case err != nil:
		return "the tester's Drafts could not be read (" + err.Error() + ")"
	case tested:
		return "its tests are written already: split it now, or grant it"
	}
	return ""
}

// unshown splits sources into the code the run did not give for an
// issue, each path neither in, nor holding, one its state or its title
// and body name, and the code it gave. A quote is asked only of code the
// agent was shown (#980): one given a file the issue is not about names
// its code from the repository's folders, as one given no code does;
// one shown a file that calls into another names that other unquoted
// (#1151).
func (p *Plan) unshown(repo string, id int, sources []string) (unshown, shown []string) {
	var given []string
	if st := p.ledger.States[id]; st != nil {
		given = append(given, st.Sources...)
	}
	is := p.issues[id]
	given = append(given, PathsIn(repo, is.Title+"\n"+is.Body)...)
	for _, src := range sources {
		s := RepoPath(src)
		if slices.ContainsFunc(given, func(g string) bool {
			g = RepoPath(g)
			return holds(s, g) || holds(g, s)
		}) {
			shown = append(shown, src)
		} else {
			unshown = append(unshown, src)
		}
	}
	return unshown, shown
}

// given says whether this run gave the agent the code of each source
// named for an issue, a file each (CodeGiven): naming it is no reason
// to read the issue again (#1089).
func (p *Plan) given(id int, sources []string) bool {
	files := p.config.CodeGiven[id]
	return len(sources) > 0 && !slices.ContainsFunc(sources, func(s string) bool {
		s, _, _ = strings.Cut(RepoPath(s), "#")
		return !slices.Contains(files, s)
	})
}

// scopeRead says whether a Scope written at a run again for the code its
// issue named stands on the code given (#1433): each file it names given,
// or its paths marked a guess (`Guessed:`), which holds nothing back
// (#970). A module folder named is given by its files, cut or not: the
// Scope's own files are what must have been read.
func (p *Plan) scopeRead(repo string, c Proposal) bool {
	files := p.config.CodeGiven[c.Issue]
	if !p.config.RunAgain || !slices.Contains(p.config.Scope, c.Issue) || len(files) == 0 {
		return false
	}
	if work.Guessed(c.Scope) {
		return true
	}
	read := 0
	for _, m := range PathLike.FindAllString(c.Scope, -1) { // every path, never PathsIn's first few
		if n := RepoPath(m); strings.Contains(n, "/") && InCommit(repo, n) {
			if !slices.Contains(files, n) {
				return false
			}
			read++
		}
	}
	return read > 0
}

// covered says whether each source named for an issue is a file this run
// gave, or a folder a file given lies below: what a Scope standing on the
// code given (scopeRead) settles, nothing left to read (#1433).
func (p *Plan) covered(id int, sources []string) bool {
	files := p.config.CodeGiven[id]
	return !slices.ContainsFunc(sources, func(s string) bool {
		s = RepoPath(s)
		return !slices.ContainsFunc(files, func(f string) bool { return holds(s, f) })
	})
}

// noCodeGiven says whether the run gave no code for an issue: its state
// names no sources, nor its title and body a file of the commit — the
// task listed the repository's folders, nothing to quote from.
func (p *Plan) noCodeGiven(repo string, id int) bool {
	if st := p.ledger.States[id]; st != nil && len(st.Sources) > 0 {
		return false
	}
	is := p.issues[id]
	return len(PathsIn(repo, is.Title+"\n"+is.Body)) == 0
}

// scopeUnread says why a refine's Scope is left out, "" when it is not:
// no code of the issue was given in this run, or the run names other code
// for it — a Scope written now would name paths not read (#970).
func (p *Plan) scopeUnread(repo string, c Proposal) string {
	if c.Ticked != "" || strings.TrimSpace(c.Scope) == "" {
		return ""
	}
	if p.config.RunAgain && slices.Contains(p.config.Scope, c.Issue) && p.codeFirst[c.Issue] && p.given(c.Issue, p.firstNamed[c.Issue]) {
		// Run again for the code it said it lacked, each file it names
		// given this time, whole or in part: its Scope is written from
		// that code (#1146). A first run's stays left out (#970).
		return ""
	}
	// The code pre gave whole (CodeRead) counts too: files a comment
	// names, as the developer's ask for its Scope does (#921); so does a
	// run again for the code it named whose Scope stands on the files
	// given, module folders named, cut or not, its state keeping no
	// source (#1433).
	again := p.scopeRead(repo, c)
	if again {
		if p.readAgain == nil {
			p.readAgain = map[int]bool{}
		}
		p.readAgain[c.Issue] = true
	}
	given := slices.Contains(p.config.CodeRead, c.Issue) || !p.noCodeGiven(repo, c.Issue) || again
	if !given || p.codeFirst[c.Issue] && !p.given(c.Issue, p.firstNamed[c.Issue]) && !again {
		return fmt.Sprintf("its %[1]s was left out: no code of #%[2]d was given in this run, so its paths would be a guess. The code they name is read at the next run, and its %[1]s is written from that code", work.Name(work.Scope), c.Issue)
	}
	return ""
}

// obsoleteOn reads where an issue's announcement as obsolete stands, from
// the forge as it is.
func (p *Plan) obsoleteOn(f forge.Backlog, repo, role string, st *State, id int) (Obsolete, error) {
	notes, err := f.Notes(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return Obsolete{}, err
	}
	return ReadObsolete(repo, p.issues[id], notes, st, role, p.settings["close-obsolete"], clock.Now()), nil
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
		c.Announce, c.Until = true, clock.Now().AddDate(0, 0, s.Delay()).Format(dateLayout)
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

// checkSources checks the code a `sources` names for an issue: 1 to
// maxSources files the commit holds, the line quoted from one of them.
func checkSources(repo string, c Proposal) (rule, why string) {
	if len(c.Sources) == 0 || len(c.Sources) > maxSources {
		return "sources-unknown", fmt.Sprintf("could not tell which code #%d is about (1 to %d files named): nothing changed; add a %s naming them, or leave it", c.Issue, maxSources, work.Name(work.Scope))
	}
	for _, s := range c.Sources {
		path, _, _ := strings.Cut(s, "#")
		if exec.Command("git", "-C", repo, "cat-file", "-e", "HEAD:"+path).Run() != nil {
			return "sources-unknown", fmt.Sprintf("could not tell which code #%d is about: %s, named for it, is not in the repository; nothing changed", c.Issue, path)
		}
	}
	if c.Quote != nil && c.Quote.Path != "" && !slices.ContainsFunc(c.Sources, func(s string) bool { return holds(s, c.Quote.Path) }) {
		return "no-quote", "the line quoted to show its code is from none of the files named: nothing changed"
	}
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
			// A Scope names the files or folders it touches: given as
			// sources, or else read from its text. One that names none the
			// commit holds is left out, the rest of the refine written —
			// never all of it dropped for its Scope.
			// The files the developer asked its Scope widened with, for a
			// blocked issue (#921): one its pull request adds is not in the
			// default branch yet, and is kept all the same (#1061).
			asked, err := p.scopeAskOf(f, is)
			if err != nil {
				return "no-state", err.Error()
			}
			// The new test files the developer kept beside the Scope
			// (ADR-0063, §9): folded into it, on its branch only yet.
			kept, err := p.scopeKeptOf(f, is, c)
			if err != nil {
				return "no-state", err.Error()
			}
			onBranch := append(slices.Clone(asked), kept...)
			if len(c.Sources) == 0 {
				c.Sources = PathsIn(repo, c.Scope)
				for _, m := range PathLike.FindAllString(c.Scope, -1) {
					// A path the Scope names whole, never a part of one.
					if a := RepoPath(m); slices.Contains(onBranch, a) && !slices.Contains(c.Sources, a) {
						c.Sources = append(c.Sources, a)
					}
				}
			}
			for _, k := range kept {
				if !slices.Contains(c.Sources, k) {
					c.Sources = append(c.Sources, k)
				}
			}
			missing := ""
			for _, s := range c.Sources {
				if path, _, _ := strings.Cut(s, "#"); !InCommit(repo, path) && !slices.Contains(onBranch, path) && missing == "" {
					missing = path
				}
			}
			// Each path the Scope writes, the developer's bounds (ADR-0065,
			// change 2): a file or folder the commit holds, a glob matching
			// one, or a new file on a line `New:`.
			// A path the repository ignores is one a build writes, a value
			// in the Scope, never a file to change (#1411).
			unfound := work.Unfound(c.Scope, append(Tracked(repo), onBranch...))
			ignored := Ignored(repo, unfound)
			for _, u := range unfound {
				if missing == "" && !slices.Contains(ignored, u) {
					missing = u
				}
			}
			switch {
			case len(c.Sources) == 0 && missing == "" && !work.MarksNew(c.Scope):
				p.Findings = append(p.Findings, verdict.Finding{Rule: "scope-without-files", Level: "info", Where: c.where(),
					Message: fmt.Sprintf("could not tell which files #%d is about, so its %[2]s was left out: add a %[2]s naming them, or leave it", c.Issue, work.Name(work.Scope))})
			case missing != "":
				p.Findings = append(p.Findings, verdict.Finding{Rule: "scope-without-files", Level: "info", Where: c.where(),
					Message: fmt.Sprintf("its %[1]s named %[2]s, which the repository does not hold, so it was left out: name a path it holds, a glob matching one, or a new file on a line `New:`", work.Name(work.Scope), missing)})
			}
			if len(c.Sources) == 0 && !work.MarksNew(c.Scope) || missing != "" {
				c.Scope, c.Sources = "", nil
				if len(asked) > 0 {
					// Its widening dropped: an unblock beside it would send
					// it back on a Scope that did not change (#1061).
					if p.widenDropped == nil {
						p.widenDropped = map[int][]string{}
					}
					p.widenDropped[c.Issue] = asked
				}
			} else {
				delete(p.widenDropped, c.Issue) // a later widening kept: the unblock stands
			}
			c.Sources = c.Sources[:min(len(c.Sources), maxSources)]
		}
		unread := p.scopeUnread(repo, *c)
		if unread != "" {
			// A Scope from code not read is a guess, wide enough to hold
			// the issue behind an unrelated claim (#970): its paths kept
			// as the code to read, the rest of the refine written.
			c.Scope = ""
		}
		if err := p.readable(f, repo, c); err != nil {
			return "no-state", err.Error()
		}
		// An outsider's issue is theirs (ADR-0021): the refined text is
		// proposed to its reporter in a comment until they, or a person of
		// the project, agree. A role's finding is the line's own draft; a
		// bot's issue the project's, never addressed to the bot (#429).
		c.ToReporter = c.Agreed == "" && Outsider(is) && !Accepted(is)
		// Answering the reviewer's findings on its spec (#128): the role
		// rewrites the sections they lie in it may rewrite (Revisable).
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
			for _, m := range []work.Meaning{work.Need, work.Validation} {
				// A section its writer marked `Draft:` (#1019): the role's
				// to rewrite, unless a person accepted it.
				if h := work.ByMeaning(is.Body)[m].Heading; MarkedDraft(is.Body, m) && !Accepted(is) && !c.ToReporter && !slices.Contains(c.Revise, h) {
					c.Revise = append(c.Revise, h)
				}
			}
			if s := work.ByMeaning(is.Body)[work.Scope]; work.Guessed(s.Text) && !c.ToReporter && !slices.Contains(c.Revise, s.Heading) {
				// A Scope its writer marked a guess (#970), the role's at
				// every level (ADR-0060): rewritten once its code is read
				// (#1126), never held back as drafted already.
				c.Revise = append(c.Revise, s.Heading)
			}
			if unread, own := ScenariosUnread(is, st); unread && own && p.config.Tester && !c.ToReporter {
				// Its own Validation, no test could name a scenario of:
				// rewritten in the form the tester reads (#855).
				if v := work.ByMeaning(is.Body)[work.Validation].Heading; !slices.Contains(c.Revise, v) {
					c.Revise = append(c.Revise, v)
				}
			}
			if NoTestStands(is, comments, role) && !c.ToReporter {
				// The tester found no test possible on this text (#1053):
				// the sections a test is written from, those the role may
				// rewrite, rewritten so one can be — never held back as
				// drafted already. A person's Validation is asked of them.
				for _, m := range []work.Meaning{work.Verification, work.Validation, work.Scope} {
					if h := work.ByMeaning(is.Body)[m].Heading; h != "" && Rewritable(is, st, m) && !slices.Contains(c.Revise, h) {
						c.Revise = append(c.Revise, h)
					}
				}
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
			if p.ledger != nil && len(Unheard(st, p.ledger.Notes[c.Issue], is)) > 0 && !c.ToReporter {
				// A person's comment it reads now, an answer to the
				// reviewer's question among them (#1331): what it settles
				// goes into the Scope, the role's at every level
				// (ADR-0060): the tester and the developer build from the
				// body, never from its comments.
				if s := work.ByMeaning(is.Body)[work.Scope].Heading; s != "" && !slices.Contains(c.Revise, s) {
					c.Revise = append(c.Revise, s)
				}
			}
			if (slices.Contains(is.Labels, LabelBlocked) || slices.Contains(is.Labels, LabelToRefine)) && len(ScopeAsked(comments)) > 0 && !c.ToReporter {
				// The developer blocked it, or sent it back to refine, to
				// ask for files outside its Scope (#921, ADR-0063 §9): the
				// Scope, the role's at every level (ADR-0060), rewritten
				// to widen it.
				if s := work.ByMeaning(is.Body)[work.Scope].Heading; s != "" && !slices.Contains(c.Revise, s) {
					c.Revise = append(c.Revise, s)
				}
			}
			if p.sizeAsked(c.Issue) && !c.ToReporter {
				// Asked on its size (ADR-0064, step 3): the Scope, the
				// role's at every level, rewritten to narrow it (#1193).
				if s := work.ByMeaning(is.Body)[work.Scope].Heading; s != "" && !slices.Contains(c.Revise, s) {
					c.Revise = append(c.Revise, s)
				}
			}
			p.plain(is, st, c)
			p.repeats(is, st, c)
		}
		_, added, kept := Refine(is.Body, *c, role)
		c.Added = added
		if len(added) == 0 && unread != "" {
			// Its Scope alone, left out: nothing keeps its paths. The
			// agent names them with a sources, no line quoted.
			return "scope-unread", fmt.Sprintf("nothing written: no code of #%d was given in this run, so its %s would be a guess. Name the code it is about with `sources`: its %[2]s is written from that code at the next run", c.Issue, work.Name(work.Scope))
		}
		if unread != "" {
			p.Findings = append(p.Findings, verdict.Finding{Rule: "scope-unread", Level: "info", Where: c.where(), Message: unread})
		}
		if len(added) == 0 {
			if len(c.Repeated) > 0 {
				return "nothing-to-refine", "nothing written: the " + strings.Join(c.Repeated, ", ") + " it drafted repeats the text above it, left out"
			}
			return "nothing-to-refine", "nothing written: the sections it drafted are in the issue already, never rewritten"
		}
		if len(kept) > 0 {
			p.Findings = append(p.Findings, verdict.Finding{Rule: "section-kept", Where: c.where(),
				Message: "already in the body, left as it is: " + strings.Join(kept, ", ")})
		}
	case "ready":
		c.Assumed, c.Written, c.Retested = nil, "", ""
		if missing := NotReady(is, p.ledger.States[c.Issue], Accepted(is)); len(missing) > 0 {
			// Drafts no person accepted, past the level's delay: taken as
			// assumptions, the issue goes on (ADR-0060).
			names, written, later := p.assumable(f, is, p.ledger.States[c.Issue])
			if len(names) == 0 {
				return "not-ready", "not moved to ready: " + strings.Join(missing, "; ") + later
			}
			c.Assumed, c.Written = names, written
		}
		if rule, why := p.answerUnwritten(is, c); rule != "" {
			return rule, why
		}
		if unread, own := ScenariosUnread(is, p.ledger.States[c.Issue]); unread && p.config.Tester && c.Proof == nil && !Readied(is) {
			// No test could name a scenario: ready, it would skip its
			// tests without a word (#855). Held, said on the issue.
			if p.unread == nil {
				p.unread = map[int]string{}
			}
			p.unread[c.Issue] = scenariosUnreadWhy(is, own)
			return "scenarios-unread", fmt.Sprintf("not moved to ready: %s (%s)", scenariosWhat(is), work.ScenarioForm)
		}
		if c.Proof != nil && !c.Retest {
			return p.checkProof(f, c)
		}
		if rule, why := p.checkNoTest(f, role, is, c); rule != "" {
			return rule, why
		}
		if c.TooBig {
			if why := p.splitAfterTests(is, *c); why != "" {
				c.TooBig = false // readied all the same, no split asked after its tests
				p.Findings = append(p.Findings, verdict.Finding{Rule: "too-big-ready-dropped", Level: "info", Where: c.where(),
					Message: "`too-big` left out of its `ready`: " + why})
			}
		}
	case "ask":
		if strings.TrimSpace(c.Questions) == "" {
			return "ask-empty", "nothing asked: the question was empty"
		}
		if p.codeFirst[c.Issue] {
			return "code-first", fmt.Sprintf("not asked: the code #%d is about, named in this run, is read first, at the next run; asked then only if it does not answer", c.Issue)
		}
	}
	return "", ""
}

// plain decides whether a refine takes the issue's text on top into the
// Need it drafts (ADR-0039, amended): a role's text no person edited
// (RolesTop), never on an outsider's issue proposed in a comment. A
// proposal a person said yes to is done as they read it: taken only if it
// said so, and the text still the role's. A description given apart, as
// an earlier engine asked, is left out, said.
func (p *Plan) plain(is forge.Issue, st *State, c *Proposal) {
	if strings.TrimSpace(c.Description) != "" {
		c.Description = ""
		p.Findings = append(p.Findings, verdict.Finding{Rule: "description-kept", Level: "info", Where: c.where(),
			Message: fmt.Sprintf("its description is left as it is: a role's text on top is taken into the %s it drafts, never rewritten apart", work.Name(work.Need))})
	}
	said := c.Lift
	c.Lift = false
	if strings.TrimSpace(c.Need) != "" && !c.ToReporter {
		roles, _ := RolesTop(is, st)
		c.Lift = roles && (said || !yes(*c))
	}
}

// yes says whether a refine is a person's yes to what was proposed: a
// label accepting it (Ticked), or the reporter's reply agreeing (Agreed).
// Its drafts are written as they read them.
func yes(c Proposal) bool { return c.Ticked != "" || c.Agreed != "" }

// repeats leaves out each draft of a refine that says again what is above
// it (ADR-0039, amended): a Need draft over a person's text on top; an
// example or a bug's steps against all above them, as the issue will read
// — the text on top, unless taken into the Need, and the Need. Past the
// role's repeated-percent-max, not written, said on the issue. How a
// person accepts it restates the case by design: never compared. A
// person's text is only read, and what they said yes to written whole.
func (p *Plan) repeats(is forge.Issue, st *State, c *Proposal) {
	if yes(*c) {
		return // measured when proposed: what a person said yes to is written whole
	}
	limit := p.config.RepeatedPercent
	if limit <= 0 {
		limit = RepeatedPercentMax
	}
	have := work.ByMeaning(is.Body)
	written := func(m work.Meaning) bool { // a draft this refine writes in the body
		return strings.TrimSpace(c.Text(m)) != "" && !work.Has(c.Refused, m) &&
			(strings.TrimSpace(have[m].Text) == "" || work.Has(c.Revise, m))
	}
	drop := func(m work.Meaning, n int) {
		name := named(nil, m)
		if s, ok := have[m]; ok {
			name = s.Heading
		}
		switch m {
		case work.Need:
			c.Need = ""
		case work.Example:
			c.Example = ""
		case work.Steps:
			c.Steps = ""
		}
		c.Repeated = append(c.Repeated, name)
		p.Findings = append(p.Findings, verdict.Finding{Rule: "repeated", Level: "info", Where: c.where(),
			Message: fmt.Sprintf("its %s draft was not written: %d%% of its words are said above it already, past %d (repeated-percent-max)", name, n, limit)})
	}
	c.Repeated = nil
	top := DescriptionOf(is.Body)
	if written(work.Need) && NeedOnTop(is, st) {
		if n := Repeated(c.Need, top); n > limit {
			drop(work.Need, n)
		}
	}
	need := have[work.Need].Text
	if written(work.Need) {
		need = c.Need
	}
	above := StripDrafts(need)
	if !c.Lift || !written(work.Need) {
		above = top + "\n\n" + above
	}
	for _, m := range []work.Meaning{work.Example, work.Steps} {
		if !written(m) {
			continue
		}
		if n := Repeated(c.Text(m), above); n > limit {
			drop(m, n)
		}
	}
}

// readable makes the sections a refine writes readable to a person, as
// the engine can tell (write.go): each decision cited bare linked to its
// page, each closed issue cited said closed.
func (p *Plan) readable(f forge.Backlog, repo string, c *Proposal) error {
	if f == nil {
		return nil
	}
	if p.pages == nil {
		p.pages = &[2]string{forge.FilePages(f), forge.IssuePages(f)}
		p.decisions = Decisions(repo)
	}
	files, issues := p.pages[0], p.pages[1]
	for _, t := range []*string{&c.Need, &c.Example, &c.Steps, &c.Verification, &c.Validation, &c.Scope} {
		*t = LinkDecisions(*t, files, p.decisions)
		if !slices.ContainsFunc(CitedIssues(*t, issues), func(n int) bool { return !p.open[n] }) {
			continue
		}
		if p.closed == nil {
			all, err := f.AllIssues()
			if err != nil {
				return err
			}
			p.closed = map[int]bool{}
			for _, is := range all {
				p.closed[is.ID] = is.Closed
			}
		}
		*t = MarkClosed(*t, issues, p.closed)
	}
	return nil
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
	if c.TooBig && (c.Issue != p.config.Sized || !p.config.SizeAsked && !slices.Contains(is.Labels, LabelToRefine)) {
		// Past its size only where the run was told to judge it (ADR-0064,
		// §2): the issue the event names, asked on its size or in refine.
		return "too-big-unasked", fmt.Sprintf("#%d's size was not asked in this run: `too-big` holds only on the issue the line woke the role on, asked on its size or predicted in refine; `split` keeps its mode", c.Issue)
	}
	if c.TooBig && p.config.SplitOff() {
		// The project turned split off itself: never on for a size
		// (ADR-0064, §3, the panel of 2026-11-08).
		return "split-off", fmt.Sprintf("this project set `split` off itself: no split, even past its size; grant #%d `size` `%s`, or leave it to a person", c.Issue, TierLarge)
	}
	if parent := partOf(is.Body); c.TooBig && parent > 0 {
		// Split depth 1 (ADR-0064, §4): a part may be granted a size,
		// never split again for it.
		return "split-depth", fmt.Sprintf("it is a part of #%d already: a part is never split for its size; grant it `size` `%s`, or leave it to a person", parent, TierLarge)
	}
	if kind, _ := KindOf(is.Labels); kind == KindBug && p.config.Tested != nil && c.Ticked == "" && !p.sizeAsked(c.Issue) {
		// A bug goes to its tests first (#1212): the tester writes the
		// reporter's symptom as a red test on its Draft, then it is
		// split; at its closing, that test is run on the default branch.
		// Blocked and asked on its size, it may be split with none: the
		// tester's own run crossed it, and each part gets its tests.
		tested, err := p.config.Tested(c.Issue)
		switch {
		case err != nil:
			return "bug-tests-unread", fmt.Sprintf("#%d is a bug, split once its reproduction test is written, and the tester's Drafts could not be read (%v): not split", c.Issue, err)
		case !tested:
			return "bug-tests-first", fmt.Sprintf("#%d is a bug: it is split once the tester wrote its reproduction test, on its Draft; `ready` it with `too-big: true`: it goes to its tests, and the line asks its split once they are written", c.Issue)
		}
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
		for _, sec := range []struct {
			field work.Meaning
			text  string
		}{{work.Need, ch.Need}, {work.Validation, ch.Validation}, {work.Scope, ch.Scope}} {
			if strings.TrimSpace(sec.text) == "" {
				return "split-sections", fmt.Sprintf("%q has no %s: each child has its %s, %s and %s", ch.Title, sec.field, work.Name(work.Need), work.Name(work.Validation), work.Name(work.Scope))
			}
		}
		if len(ch.Sources) == 0 || len(ch.Sources) > maxSources {
			return "sources-unknown", fmt.Sprintf("not split: the part %q names no file of the repository (1 to %d) for its %s", ch.Title, maxSources, work.Name(work.Scope))
		}
		for _, k := range ch.After {
			if k < 1 || k > len(c.Into) || k == i+1 {
				return "split-after", fmt.Sprintf("%q: `after` names the other children it waits on, by their place in the split, 1 to %d", ch.Title, len(c.Into))
			}
		}
		for _, s := range ch.Sources {
			path, _, _ := strings.Cut(s, "#")
			if out, err := exec.Command("git", "-C", repo, "cat-file", "-t", "HEAD:"+path).Output(); path == "" || err != nil || strings.TrimSpace(string(out)) != "blob" {
				return "sources-unknown", fmt.Sprintf("not split: %q, named for a part, is not in the repository", path)
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
// four sections, the need and how it is accepted as drafts; and the
// headings it wrote, for its state.
func ChildBody(parent int, ch Child, role string) (string, []string) {
	body, added, _ := Refine(fmt.Sprintf("Part of #%d.", parent), Proposal{Need: ch.Need, Verification: ch.Verification, Validation: ch.Validation, Scope: ch.Scope}, role)
	return body, added
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
	body, footer := work.Footer(body)
	if _, ok := work.Sections(body)["Sub-issues"]; ok {
		return work.WithFooter(fill(body, "Sub-issues", text), footer)
	}
	return work.WithFooter(strings.TrimRight(body, " \r\n")+"\n\n"+SubIssuesHeading+"\n\n"+text, footer)
}

// layout is the order the role writes an issue's sections in, the table's
// (work.Headings): the person's part above a thematic break, the builder's
// below it. The example and the steps are never required.
func layout() []work.Meaning { return work.Headings.Meanings() }

// builder says whether a section is of the builder's part, below the break.
func builder(m work.Meaning) bool { return work.Headings.Builder(m) }

// drafted says whether the role writes a section as a draft, a person's to
// state: the person's part.
func drafted(m work.Meaning) bool { return m != "" && !builder(m) }

// Drafted says whether the role writes a section, by its heading, as a
// draft, a person's to make theirs: the need and how it is accepted, an
// example, the steps to reproduce. Not marked a draft, it is a person's.
func Drafted(name string) bool { return drafted(work.MeaningOf(name)) }

// Text is the text a refine gives a section, by its meaning: the fields of
// a refine are named by meaning.
func (c Proposal) Text(m work.Meaning) string {
	switch m {
	case work.Need:
		return c.Need
	case work.Example:
		return c.Example
	case work.Steps:
		return c.Steps
	case work.Validation:
		return c.Validation
	case work.Verification:
		return c.Verification
	case work.Scope:
		return c.Scope
	}
	return ""
}

// named is the heading names give meaning m, or the one it is written under.
func named(names []string, m work.Meaning) string {
	if n, ok := work.Headings.In(names, m); ok {
		return n
	}
	return work.Name(m)
}

// DraftMarker marks a section the role drafted and no person made theirs.
var DraftMarker = forge.Marker("draft")

// DraftLine opens a drafted section: the marker alone, which a reader does
// not see. The role's one comment says the section is a draft and how to
// make it yours: workline:accepted, or an edit (ADR-0038). An issue an
// earlier engine drafted holds a visible line too; read the same.
func DraftLine(role string) string { return DraftMarker }

// IsDraft says whether an issue's section is still the role's draft: its
// text marked so, and as the role wrote it (the state's `wrote`) — one a
// person edited is theirs, the marker left in it or not. A draft written
// before the role kept its digest is told by its marker alone.
func IsDraft(body string, m work.Meaning, st *State) bool {
	text := work.ByMeaning(body)[m].Text
	if !drafted(m) || !strings.Contains(text, DraftMarker) {
		return false
	}
	_, wrote := st.wrote(m)
	return wrote == "" || wrote == BodyDigest(text)
}

// HasDraft says whether an issue's body holds a draft of the role's still.
func HasDraft(body string, st *State) bool {
	return slices.ContainsFunc(layout(), func(m work.Meaning) bool { return IsDraft(body, m, st) })
}

// Refine is an issue's body with the sections it lacks written: one there
// but empty (an issue form's "_No response_") filled in place, one missing
// added where the Layout puts it among those there, after the text, which
// stays as it is — a thematic break between the person's part and the
// builder's; added and kept name the sections written and those already
// there, left alone. A role's text on top (c.Lift, checked before) leaves
// the top as the Need draft is written: Description among added; what it
// folded there — a cause, its fix — goes to the end of the last builder's
// section written, kept on top when none is (#841).
func Refine(body string, c Proposal, role string) (string, []string, []string) {
	have := work.ByMeaning(body)
	body, footer := work.Footer(body) // a section added goes above the body's footer
	body = strings.TrimRight(body, " \r\n")
	type write struct {
		m           work.Meaning
		name, text  string
		ok, revised bool
	}
	var writes []write
	var kept []string
	for _, m := range layout() {
		text := Inert(strings.TrimSpace(c.Text(m))) // no quick action run on GitLab when the body is written
		if text == "" || work.Has(c.Refused, m) || work.Testers(m) && role != TesterRole {
			// The Verification is the tester's to write (ADR-0066): one
			// another role gives is left out.
			continue
		}
		if drafted(m) {
			text = DraftLine(role) + "\n\n" + text
		}
		// A section there is written under its heading in the body; one
		// missing, under the heading the table writes.
		old, ok := have[m]
		name := work.Name(m)
		if ok {
			name = old.Heading
		}
		revised := ok && strings.TrimSpace(old.Text) != "" && work.Has(c.Revise, m) && squeeze(StripDrafts(old.Text)) != squeeze(StripDrafts(text))
		if ok && strings.TrimSpace(old.Text) != "" && !revised {
			kept = append(kept, name)
			continue
		}
		writes = append(writes, write{m, name, text, ok, revised})
	}
	below := -1 // the last builder's section written: what the top folded goes there
	for i, w := range writes {
		if builder(w.m) {
			below = i
		}
	}
	var added []string
	for i, w := range writes {
		if w.m == work.Need && c.Lift && DescriptionOf(body) != "" {
			// Its text on top, a role's, is this draft now: said once.
			if folded := Folded(body); folded != "" && below >= 0 {
				writes[below].text += "\n\n" + folded
			}
			body = withoutTop(body, below < 0)
			added = append(added, DescriptionName)
		}
		switch {
		case w.revised:
			body = fill(body, w.name, writes[i].text) // the role's own, rewritten to answer the reviewer (#128)
		case w.ok:
			body = fill(body, w.name, writes[i].text)
		default:
			body = insert(body, w.m, writes[i].text)
		}
		added = append(added, w.name)
	}
	if len(added) == 0 {
		return work.WithFooter(body, footer), added, kept
	}
	if f := strings.TrimSpace(footer); work.Break(f) && slices.ContainsFunc(added, isBuilder) {
		footer = "" // a break alone after the person's part: the one put above the builder's
	}
	return work.WithFooter(withBreak(body), footer), added, kept
}

// isBuilder says whether a section, by its heading, is of the builder's part.
func isBuilder(name string) bool { return builder(work.MeaningOf(name)) }

// insert adds a section where the layout puts it: after the nearest one
// before it in the body — above the break when it is the person's —, else
// before the nearest one after it, else after the body's text; at the
// level of the heading it goes by.
func insert(body string, m work.Meaning, text string) string {
	lines := strings.Split(body, "\n")
	at := map[work.Meaning]int{} // the line of each section of the layout, the last when written twice
	for i, l := range lines {
		if h, ok := work.Heading(l); ok && work.MeaningOf(h) != "" {
			at[work.MeaningOf(h)] = i
		}
	}
	name := work.Name(m)
	section := func(i int) string { // at the level of heading i
		return lines[i][:strings.Index(lines[i], " ")+1] + name + "\n\n" + text
	}
	order := layout()
	k := slices.Index(order, m)
	for _, before := range slices.Backward(order[:k]) {
		i, ok := at[before]
		if !ok {
			continue
		}
		next := i + 1
		for next < len(lines) {
			if _, ok := work.Heading(lines[next]); ok || strings.HasPrefix(lines[next], "# ") {
				break
			}
			next++
		}
		if b := breakAbove(lines, next); b >= 0 && !builder(m) {
			next = b
		}
		return splice(lines, next, section(i))
	}
	for _, after := range order[k+1:] {
		i, ok := at[after]
		if !ok {
			continue
		}
		to := i
		if b := breakAbove(lines, i); b >= 0 && !builder(m) {
			to = b
		}
		return splice(lines, to, section(i))
	}
	return body + "\n\n## " + name + "\n\n" + text
}

// breakAbove is the line of the thematic break just above line i, blank
// lines between, or -1.
func breakAbove(lines []string, i int) int {
	for j := i - 1; j >= 0; j-- {
		switch {
		case strings.TrimSpace(lines[j]) == "":
			continue
		case work.Break(lines[j]) && j > 0 && strings.TrimSpace(lines[j-1]) == "":
			return j
		}
		return -1
	}
	return -1
}

// splice puts a section at line i of a body, blank lines around it.
func splice(lines []string, i int, section string) string {
	head := strings.TrimRight(strings.Join(lines[:i], "\n"), " \r\n")
	tail := strings.TrimLeft(strings.Join(lines[i:], "\n"), " \r\n")
	if tail != "" {
		tail = "\n\n" + tail
	}
	return head + "\n\n" + section + tail
}

// withBreak puts a thematic break between the person's part of a body and
// the builder's, when each section of the one comes before each of the
// other and none is between them.
func withBreak(body string) string {
	lines := strings.Split(body, "\n")
	person, builders := -1, -1
	for i, l := range lines {
		h, ok := work.Heading(l)
		switch {
		case !ok || work.MeaningOf(h) == "":
		case isBuilder(h) && builders < 0:
			builders = i
		case !isBuilder(h) && builders >= 0:
			return body // the person's part and the builder's mixed: left as they are
		case !isBuilder(h):
			person = i
		}
	}
	if person < 0 || builders < 0 || breakAbove(lines, builders) > person {
		return body
	}
	return splice(lines, builders, "---")
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
	// The tester's state, between refined and ready (ADR-0059): set by
	// the product owner where the project routes the tester.
	LabelNeedsTests = "workline:needs-tests"
	// A state no role leaves alone: its reason on the issue (ADR-0059).
	LabelBlocked = "workline:blocked"
)

// Readied says whether an issue went past refining: ready, or waiting
// for its tests on the way there (ADR-0059).
func Readied(is forge.Issue) bool {
	return slices.Contains(is.Labels, LabelReady) || slices.Contains(is.Labels, LabelNeedsTests)
}

// AcceptedSays is the description of a person's yes, as the forge's list of
// labels shows it: on an issue, to what the product owner proposes
// (ADR-0038); on a pull request, to merge it, by the project's merge policy
// (ADR-0042). GitHub keeps 100 characters.
const AcceptedSays = "A person's yes: to what the product owner proposes; on a pull request, to merge it"

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
// empty, Need or Validation still a draft no person accepted nor edited. A
// person's text on top, with no Need section, is its Need (NeedOnTop).
func NotReady(is forge.Issue, st *State, accepted bool) []string {
	body := is.Body
	have := work.ByMeaning(body)
	var out []string
	for _, m := range work.Required {
		text := strings.TrimSpace(have[m].Text)
		switch {
		case m == work.Need && text == "" && NeedOnTop(is, st):
		case text == "":
			out = append(out, work.Name(m)+" is missing")
		case !accepted && IsDraft(body, m, st):
			out = append(out, have[m].Heading+" is still a draft: accept it ("+LabelAccepted+") or edit it")
		case !Accepted(is) && MarkedDraft(body, m):
			// Its writer's own `Draft:` (#1019): never taken as written,
			// nor as an assumption; rewritten first, or a person's yes.
			out = append(out, have[m].Heading+" opens with `Draft:`: rewrite it, or accept it ("+LabelAccepted+")")
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
// question twice; it sets the round the comment would be. A question
// asked before is left out, the rest of the act kept (left): an ask with
// no question left is dropped, a refine keeps its drafts.
func conversation(role string, comments []string, c *Proposal) (rule, why string, left []string) {
	e := ReadExchange(comments, role)
	if e.Rounds > 0 && !e.Answered {
		if c.Do == "ask" {
			return "already-asked", "its reporter was asked and has not answered since: an issue is asked again only after an answer", nil
		}
		return "already-proposed", "its reporter was proposed a text, or asked, and has not answered since: nothing more is written before an answer", nil
	}
	// The same question, whole: the leads the engine wrote are not in
	// e.Asked (ReadExchange).
	var asked []string
	for _, a := range e.Asked {
		asked = append(asked, questions(a)...)
	}
	kept, left := withoutAsked(c.Questions, asked)
	if len(left) > 0 && c.Do == "ask" && len(questions(kept)) == 0 {
		return "asked-before", fmt.Sprintf("%q was asked before: a question is never asked twice", left[0]), nil
	}
	c.Questions = kept
	c.Round = e.Rounds + 1
	return "", "", left
}

// withoutAsked is a text of questions with those asked before left out,
// each from where it starts — its sentence or its list item — to its
// "?"; what comes before it stays. No question left, nothing is. left
// are the questions taken out, as written.
func withoutAsked(text string, asked []string) (kept string, left []string) {
	var b strings.Builder
	for _, part := range strings.SplitAfter(text, "?") {
		start, q := question(part)
		if q == "" || !slices.Contains(asked, q) {
			b.WriteString(part)
			continue
		}
		b.WriteString(part[:start])
		left = append(left, bullet.ReplaceAllString(squeeze(part[start:]), ""))
	}
	if len(left) == 0 {
		return text, nil
	}
	if kept = strings.TrimSpace(b.String()); len(questions(kept)) == 0 {
		kept = ""
	}
	return kept, left
}

// leftOut says, in plain words, the questions a refine or an ask was
// written without.
func leftOut(left []string) string {
	quoted := make([]string, len(left))
	for i, q := range left {
		quoted[i] = `"` + q + `"`
	}
	if len(left) == 1 {
		return fmt.Sprintf("The question %s was asked before, so it was left out; the rest was kept.", quoted[0])
	}
	return fmt.Sprintf("The questions %s were asked before, so they were left out; the rest was kept.", and(quoted))
}

// sentenceEnd is where a sentence ends before a question, spaces as
// written.
var sentenceEnd = regexp.MustCompile(`[.!]\s+(?:[-*•]\s+|\d+[.)]\s+)?\p{Lu}`)

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
		if _, q := question(part); q != "" {
			out = append(out, q)
		}
	}
	return out
}

// question is where the question a part of a text ends with starts, and
// that question squeezed and lowercased; "" when the part ends with none.
func question(part string) (start int, q string) {
	if !strings.HasSuffix(part, "?") {
		return 0, ""
	}
	// A list's item on a line of its own is a question of its own.
	if m := listItem.FindAllStringIndex(part, -1); len(m) > 0 {
		start = m[len(m)-1][0]
	}
	// The question alone, not the sentence before it: a sentence ends at a
	// "." or "!", spaces, then a capital — perhaps after a bullet —, not at
	// a file's or a version's dot, nor at "e.g.".
	if m := sentenceEnd.FindAllStringIndex(part[start:], -1); len(m) > 0 {
		start += m[len(m)-1][0] + 1
	}
	text := bullet.ReplaceAllString(squeeze(part[start:]), "") // a list's bullet or number aside
	if q = strings.ToLower(strings.Trim(text, " *")); len(q) > 3 {
		return start, q
	}
	return start, ""
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
	for _, m := range layout() {
		name, ok := work.Headings.In(c.Added, m)
		if !ok {
			continue
		}
		draft := ""
		if drafted(m) {
			draft = " (yours to state: a draft from your words)"
		}
		fmt.Fprintf(&b, "**%s**%s\n\n%s\n\n", name, draft, strings.TrimSpace(c.Text(m)))
	}
	if q := strings.TrimSpace(c.Questions); q != "" {
		fmt.Fprintf(&b, "**What it still needs:** %s\n\n", q)
	}
	fmt.Fprintf(&b, "To agree, reply `%s` alone: the sections are then written in your issue at its next run. Or copy them into your issue (edit it), changing what is wrong; or reply what is wrong, and it reads your answer at its next run. A maintainer may agree for the project with the label `%s`: the sections are then written in the issue, and it moves to ready.\n\n", AgreeWord, LabelAccepted)
	kept := struct {
		Need         string   `yaml:"need,omitempty"`
		Example      string   `yaml:"example,omitempty"`
		Steps        string   `yaml:"steps,omitempty"`
		Verification string   `yaml:"verification,omitempty"`
		Validation   string   `yaml:"validation,omitempty"`
		Scope        string   `yaml:"scope,omitempty"`
		Sources      []string `yaml:"sources,flow,omitempty"`
	}{}
	for _, name := range c.Added {
		switch work.MeaningOf(name) {
		case work.Need:
			kept.Need = c.Need
		case work.Example:
			kept.Example = c.Example
		case work.Steps:
			kept.Steps = c.Steps
		case work.Verification:
			kept.Verification = c.Verification
		case work.Validation:
			kept.Validation = c.Validation
		case work.Scope:
			kept.Scope, kept.Sources = c.Scope, c.Sources
		}
	}
	data, _ := yaml.Marshal(kept)
	b.WriteString(engineBlock + "\n\n" + yamlFenced(string(data)) + "\n</details>")
	return b.String()
}

// yamlFenced puts YAML in a code block whose fence is longer than any run
// of backticks it holds: a fence in a bug's steps or an example neither
// ends it nor is read for it (#391); yamlBlock reads it back.
func yamlFenced(data string) string {
	fence := FenceFor(data)
	return fence + "yaml\n" + data + fence
}

// engineBlock opens the block of a proposal the engine reads back.
const engineBlock = "<details><summary>As the engine reads it</summary>"

// LastProposal reads the sections of the last refined text proposed to an
// issue's reporter; nil when none was, or it does not read
// (ProposalUnread says so).
func LastProposal(comments []string, role string) *Proposal {
	p, _ := lastProposal(comments, role)
	return p
}

// ProposalUnread says why the engine's block of the last text proposed to
// an issue's reporter does not read; nil when it reads, or none was
// proposed. An agreement to it writes nothing: said (#302), never left
// for a person to wonder at.
func ProposalUnread(comments []string, role string) error {
	_, err := lastProposal(comments, role)
	return err
}

func lastProposal(comments []string, role string) (*Proposal, error) {
	prefix := "<!-- workline:" + role + "/proposal="
	for i := len(comments) - 1; i >= 0; i-- {
		if last, _ := EngineMarker(comments[i]); !Round(comments[i], role) || !strings.HasPrefix(last, prefix) {
			continue
		}
		// The engine's own block, after its summary: the agent's words come
		// before it, their fences neutralised (ProposalComment).
		at := strings.LastIndex(comments[i], engineBlock)
		if at < 0 {
			return nil, fmt.Errorf("no %q block in it", "As the engine reads it")
		}
		text, ok := yamlBlock(comments[i][at:])
		if !ok {
			return nil, fmt.Errorf("no YAML in its %q block", "As the engine reads it")
		}
		var p Proposal
		if err := yaml.Unmarshal([]byte(text), &p); err != nil {
			return nil, fmt.Errorf("its YAML: %v", err)
		}
		return &p, nil
	}
	return nil, nil
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
		last.Need != c.Need || last.Example != c.Example || last.Steps != c.Steps ||
		last.Verification != c.Verification || last.Validation != c.Validation || last.Scope != c.Scope {
		return "not-agreed", "no reply agreeing to the text last proposed to its reporter: nothing of it is written in the body"
	}
	return "", ""
}

// holds says whether a source named for an issue's code holds the file a
// quote is from: that file, with an anchor or not, or a folder it is in
// (#392).
func holds(source, file string) bool {
	s, _, _ := strings.Cut(source, "#")
	s = strings.TrimSuffix(s, "/")
	return s != "" && (s == file || strings.HasPrefix(file, s+"/"))
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
		return fmt.Sprintf("Set #%d's priority to %s (was: %s): %s", c.Issue, scale.Value(c.Priority), c.Before, strings.TrimSpace(c.Why))
	}
	if c.Do == "effort" {
		return fmt.Sprintf("Set #%d's effort to %s (was: %s): %s", c.Issue, scale.EffortValue(c.Effort), c.Before, strings.TrimSpace(c.Why))
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
	case "size":
		return fmt.Sprintf("%s #%d `%s`: %s Every size target of its runs ×%d. Wrong? Take the label off; the role does not grant it again.", map[bool]string{true: "Granted", false: "Grant"}[done], c.Issue, TierLabel(c.Tier), strings.TrimSuffix(strings.TrimSpace(c.Why), ".")+".", TierFactor(c.Tier))
	case "kind":
		return fmt.Sprintf("%s #%d `%s`, a %s: %s Wrong? Change the label; the role does not set it again.", map[bool]string{true: "Labelled", false: "Label"}[done], c.Issue, c.KindLabel, c.IssueKind, strings.TrimSpace(c.Why))
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
		lifted := slices.Contains(c.Added, DescriptionName)
		for _, n := range c.Added {
			switch {
			case n == DescriptionName:
				continue
			case lifted && work.MeaningOf(n) == work.Need:
				n += " (draft, from the text on top)"
			case Drafted(n):
				n += " (draft)"
			}
			names = append(names, n)
		}
		return fmt.Sprintf("%s #%d: %s. %s", map[bool]string{true: "Refined", false: "Refine"}[done], c.Issue, strings.Join(names, ", "), strings.TrimSpace(c.Why))
	case "ready":
		board := ""
		switch {
		case c.Card == nil:
		case c.Card.Left == "user":
			board = fmt.Sprintf(", its card not moved to `%s`: a user's project, which GitHub lets no App write", c.Card.Column)
		default:
			board = fmt.Sprintf(", its card to `%s` on the board", c.Card.Column)
		}
		if c.Proof != nil {
			// No test possible, said the tester: the kind picked in place
			// of its Draft, and why.
			board += fmt.Sprintf(", proved by %s in place of new tests (%s)", c.Proof.Kind, strings.TrimSuffix(strings.TrimSpace(c.Proof.Reason), "."))
		}
		if len(c.Assumed) > 0 {
			// Drafts no person accepted past the delay (ADR-0060).
			board += fmt.Sprintf(", its %s taken as written after no answer, said under %s", and(c.Assumed), AssumptionsName)
		}
		return fmt.Sprintf("%s #%d to ready%s: %s", map[bool]string{true: "Moved", false: "Move"}[done], c.Issue, board, strings.TrimSpace(c.Why))
	case "unready":
		if done {
			return fmt.Sprintf("Moved #%d back to refine (%s off, %s on; put %s back to undo): %s", c.Issue, LabelReady, LabelToRefine, LabelReady, strings.TrimSpace(c.Why))
		}
		return fmt.Sprintf("Move #%d back to refine (%s off, %s on): %s", c.Issue, LabelReady, LabelToRefine, strings.TrimSpace(c.Why))
	case "unblock":
		label := StateLabel(c.To)
		if done {
			return fmt.Sprintf("Moved #%d out of Blocked, to %s (%s off, %s on; put %s back to undo): %s", c.Issue, c.To, LabelBlocked, label, LabelBlocked, strings.TrimSpace(c.Why))
		}
		return fmt.Sprintf("Move #%d out of Blocked, to %s (%s off, %s on): %s", c.Issue, c.To, LabelBlocked, label, strings.TrimSpace(c.Why))
	case "blocked":
		who := ""
		if c.Person {
			who = ", for a person to lift"
		}
		return fmt.Sprintf("Kept #%d blocked%s, read again: %s", c.Issue, who, strings.TrimSpace(c.Why))
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
	case "assume":
		if c.Answered != "" {
			return fmt.Sprintf("Marked the assumptions on #%d as answered by @%s: their answer stands over them.", c.Issue, c.Answered)
		}
		return fmt.Sprintf("%s on #%d: %s Answer here, or edit its line under %s in the issue, to change it.",
			AssumedLead, c.Issue, strings.TrimSuffix(strings.Join(strings.Fields(c.Assumption), " "), ".")+".", AssumptionsName)
	}
	if c.Do == "sources" {
		v := "Named"
		if verb == "Close" {
			v = "Name"
		}
		if c.Quote == nil {
			return fmt.Sprintf("%s the code #%d is about: %s. %s", v, c.Issue, strings.Join(c.Sources, ", "), strings.TrimSpace(c.Why))
		}
		return fmt.Sprintf("%s the code #%d is about: %s. %s %s", v, c.Issue, strings.Join(c.Sources, ", "), cite(*c.Quote), strings.TrimSpace(c.Why))
	}
	if c.Do == "keep" {
		return fmt.Sprintf("Kept #%d open, announced obsolete on %s: %s.", c.Issue, c.Announced, strings.TrimSuffix(strings.TrimSpace(c.Why), "."))
	}
	if c.Parent {
		why := strings.TrimSpace(c.Why)
		if c.Symptom != "" {
			why = strings.TrimSuffix(why, ".") + "; " + c.Symptom // a bug's, its symptom gone (#1212)
		}
		return fmt.Sprintf("%s #%d as completed: %s", map[bool]string{true: "Closed", false: "Close"}[done], c.Issue, why)
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

// refined says on the issue, in one line, what a refine did to its body:
// the sections added, those it rewrote, its description
// rewritten in plain words, and where the text before is — the forge's edit history, which no forge links to.
// A Verification or Scope a person wrote, rewritten, is in that history too
// (ADR-0060): never quoted here.
func refined(c Proposal) string {
	var added, changed []string
	var said []string
	for _, n := range c.Added {
		name := n
		if n == DescriptionName {
			said = append(said, fmt.Sprintf("Rewrote: the text on top as the %s, in plain words.", work.Name(work.Need)))
			continue
		}
		if Drafted(n) {
			name += " (draft)"
		}
		if work.Has(c.Revise, work.MeaningOf(n)) {
			changed = append(changed, name)
		} else {
			added = append(added, name)
		}
	}
	list := func(names []string) string {
		if len(names) == 0 {
			return "—"
		}
		return strings.Join(names, ", ")
	}
	if len(added) > 0 || len(changed) > 0 || len(said) == 0 {
		said = append([]string{fmt.Sprintf("Added: %s. Changed: %s.", list(added), list(changed))}, said...)
	}
	said = append(said, notWritten(c)...)
	return strings.Join(said, " ") + " See the issue's edit history."
}

// notWritten says each draft a refine left out for saying again what is
// above it, in a sentence of its own (ADR-0039, amended).
func notWritten(c Proposal) []string {
	var out []string
	for _, n := range c.Repeated {
		out = append(out, fmt.Sprintf("Not written: the %s, which said again what is above it.", n))
	}
	return out
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
	if c.Parent {
		by := "Closed by the " + role + " role"
		if c.Ticked != "" {
			by += ", " + c.Ticked + " having accepted it (" + LabelAccepted + ")"
		}
		if c.Symptom != "" && c.Ticked == "" {
			// A bug's, its reporter's symptom proven gone (#1212).
			return fmt.Sprintf("Fixed: its parts are closed, and %s; what each part delivered is said in the comment on its parts.\n\nClosed by the %s role. Reopen it to undo.", c.Symptom, role)
		}
		return fmt.Sprintf("Accepted: its parts are closed, what each delivered said in the comment on its parts.\n\n%s. Reopen it to undo.", by)
	}
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

// recordDrafted notes, in the plan's copy of an issue's state, the drafts a
// refine just wrote into its body as the role's own, so that a ready later
// in the run reads them as apply will: drafts no person accepted.
func (p *Plan) recordDrafted(c Proposal) {
	if p.ledger == nil {
		return
	}
	if p.ledger.States == nil {
		p.ledger.States = map[int]*State{}
	}
	is := p.issues[c.Issue]
	have := work.ByMeaning(is.Body)
	st := State{}
	if old := p.ledger.States[c.Issue]; old != nil {
		st = *old
		st.Wrote = maps.Clone(old.Wrote)
	}
	for _, m := range layout() {
		if !drafted(m) || work.Has(c.Refused, m) || strings.TrimSpace(c.Text(m)) == "" {
			continue
		}
		if sec, ok := have[m]; ok {
			st.SetWrote(sec.Heading, BodyDigest(sec.Text))
		}
	}
	p.ledger.States[c.Issue] = &st
}
