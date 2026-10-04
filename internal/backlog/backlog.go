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
	}{s.Sources, s.Confirmed, s.Judged, s.Comments, s.Body})
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
	Title       string   `yaml:"title,omitempty"` // an issue to open
	Quote       *Quote   `yaml:"quote"`
	Why         string   `yaml:"why"`
	// Refining: the sections written, Need and Validation as drafts; Added,
	// the engine's, says which the body did not have yet.
	Scope        string   `yaml:"scope,omitempty"`
	Verification string   `yaml:"verification,omitempty"`
	Need         string   `yaml:"need,omitempty"`
	Validation   string   `yaml:"validation,omitempty"`
	Added        []string `yaml:"added,omitempty"`
	Questions    string   `yaml:"questions,omitempty"` // asking the reporter
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
var Kinds = []string{"open", "close", "sources", "milestone", "refine", "ready", "ask"}

// Decision is what becomes of one act.
type Decision struct {
	Index int      `yaml:"index"` // the intention's place in the run
	Mode  string   `yaml:"mode"`  // act, propose, or off (dropped)
	Act   Proposal `yaml:"act"`
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
}

// Setting is a kind of act's mode and cap.
type Setting struct {
	Mode string
	Max  int
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
		out[kind] = s
	}
	return out
}

var closeReasons = []string{"duplicate", "obsolete"}

// maxSources bounds the files an issue names.
const maxSources = 5

// Decide plans the acts proposed, given at their place in the run.
func Decide(f forge.Backlog, repo, role string, settings map[string]Setting, closes map[int]Proposal) (*Plan, error) {
	p := &Plan{}
	if err := p.readRecord(f, role); err != nil {
		return nil, err
	}
	dropped := func(c Proposal, rule, msg string) {
		p.Findings = append(p.Findings, verdict.Finding{Rule: rule, Where: c.where(), Message: msg})
	}
	done := map[string]int{}
	var indexes []int
	for i := range closes {
		indexes = append(indexes, i)
	}
	slices.Sort(indexes)
	for _, i := range indexes {
		c := closes[i]
		mode, why := p.check(f, repo, role, &c)
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
			if d.Mode == Act && c.Do == "ready" && !p.issues[c.Issue].Insider {
				// The reporter has no write access: the issue is theirs
				// (ADR-0018), moving it is for a person to decide.
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "reporter-outside", Where: c.where(),
					Message: "opened by someone without write access to the project: moving it to ready is proposed, not done"})
			}
			if d.Mode == Act && s.Max > 0 && done[c.Kind()] >= s.Max {
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "act-cap", Where: c.where(),
					Message: fmt.Sprintf("%s: at most %d a run; this one is proposed", c.Kind(), s.Max)})
			}
			if d.Mode == Act {
				done[c.Kind()]++
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
		if !settled && !decided[key] {
			kept = append(kept, q)
		}
	}
	for _, d := range p.Decisions {
		if d.Mode == Propose {
			q := Pending{Issue: d.Act.Issue, Act: d.Act.Kind(), Line: describe(d.Act, "Close")}
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
	if c.Do == "milestone" {
		// Ordering says nothing of an issue's truth: no quote, its state
		// readable all the same.
		if t := strings.TrimSpace(c.Milestone); t == "" || len(t) > 60 || strings.ContainsAny(t, "\n") {
			return "milestone-title", "a milestone is a title of one line, 60 characters at most"
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
	if _, found, err := ReadState(comments, role); !found {
		return "no-state", "the issue has no state comment yet: it is not acted on before the engine has one"
	} else if err != nil {
		return "state-broken", "the issue's state comment does not read (" + err.Error() + "): nothing is written on it"
	}
	if c.Do == "ask" && slices.ContainsFunc(comments, func(s string) bool { return strings.Contains(s, AskMarker(role)) }) {
		return "already-asked", "its reporter was asked already: an issue is asked once"
	}
	if c.Do == "milestone" || c.Do == "refine" || c.Do == "ready" || c.Do == "ask" {
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
		if missing := NotReady(is.Body); len(missing) > 0 {
			return "not-ready", "it stays to refine: " + strings.Join(missing, "; ")
		}
	case "ask":
		if strings.TrimSpace(c.Questions) == "" {
			return "ask-empty", "an ask holds the questions"
		}
	}
	return "", ""
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

// Refine is an issue's body with the sections it lacks added after its
// text, which stays as it is; added and kept name the sections written and
// those already there, left alone.
func Refine(body string, c Proposal, role string) (string, []string, []string) {
	have := work.Sections(body)
	given := map[string]string{"Need": c.Need, "Verification": c.Verification, "Validation": c.Validation, "Scope": c.Scope}
	var b strings.Builder
	b.WriteString(strings.TrimRight(body, " \n"))
	var added, kept []string
	for _, name := range Sections {
		text := strings.TrimSpace(given[name])
		if text == "" {
			continue
		}
		if _, ok := have[name]; ok {
			kept = append(kept, name)
			continue
		}
		added = append(added, name)
		b.WriteString("\n\n## " + name + "\n\n")
		if slices.Contains(drafted, name) {
			b.WriteString(DraftLine(role) + "\n\n")
		}
		b.WriteString(text)
	}
	return b.String(), added, kept
}

// NotReady says what keeps an issue's body from ready: a section missing or
// empty, Need or Validation still a draft.
func NotReady(body string) []string {
	have := work.Sections(body)
	var out []string
	for _, name := range Sections {
		text := strings.TrimSpace(have[name])
		switch {
		case text == "":
			out = append(out, "## "+name+" is missing or empty")
		case slices.Contains(drafted, name) && strings.Contains(text, DraftMarker):
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

// AskMarker marks the comment asking an issue's reporter, once.
func AskMarker(role string) string { return forge.Marker(role + "/ask") }

// Ask is the comment asking the reporter what is missing.
func Ask(author, questions string) string {
	if author == "" {
		return "To refine this issue: " + strings.TrimSpace(questions)
	}
	return fmt.Sprintf("@%s, to refine this issue: %s", author, strings.TrimSpace(questions))
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
			case "milestone":
				undo = " Move it back to undo."
			case "refine":
				undo = " Edit its body to undo."
			case "ready":
				undo = " Remove the label workline:ready to undo."
			case "ask":
				undo = ""
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
	if c.Do == "milestone" {
		return fmt.Sprintf("Put #%d in the milestone %q: %s", c.Issue, c.Milestone, strings.TrimSpace(c.Why))
	}
	done := verb != "Close"
	switch c.Do {
	case "refine":
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
		return fmt.Sprintf("%s #%d's reporter: %s", map[bool]string{true: "Asked", false: "Ask"}[done], c.Issue, strings.TrimSpace(c.Questions))
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
