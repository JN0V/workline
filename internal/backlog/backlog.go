// Package backlog decides what becomes of a role's acts on a project's
// issues (docs/spec/backlog-acts.md): each is checked against the code and
// the forge, then done, proposed to a person, or dropped. Closing is the
// first act built.
package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
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
	Judged    string   `yaml:"judged,omitempty"` // the commit the role last read it at
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
	}{s.Sources, s.Confirmed, s.Judged})
	return "What workline knows of this issue; edited by the engine, not by hand.\n\n```yaml\n" + string(data) + "```"
}

// Record is what the role did, kept on its report issue.
type Record struct {
	Closed  []Closing `yaml:"closed,omitempty"`
	Wrong   []Closing `yaml:"wrong,omitempty"`        // closings found wrong: their issue open again
	Propose []string  `yaml:"propose,flow,omitempty"` // kinds of act back to propose, until the person says
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
	Quote       *Quote   `yaml:"quote"`
	Why         string   `yaml:"why"`
}

// Kind is the kind of act, as settings name it.
func (c Proposal) Kind() string {
	if c.Do == "sources" {
		return "sources"
	}
	return "close-" + c.Reason
}

// Kinds are the intentions that are acts on the backlog.
var Kinds = []string{"close", "sources"}

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
	Changed   bool              `yaml:"changed"` // the record changed: wrong closings found
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
		p.Findings = append(p.Findings, verdict.Finding{Rule: rule, Where: fmt.Sprintf("#%d", c.Issue), Message: msg})
	}
	done := map[string]int{}
	var indexes []int
	for i := range closes {
		indexes = append(indexes, i)
	}
	slices.Sort(indexes)
	for _, i := range indexes {
		c := closes[i]
		d := Decision{Index: i, Mode: Off, Act: c}
		mode, why := p.check(f, repo, role, c)
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
			if d.Mode == Act && s.Max > 0 && done[c.Kind()] >= s.Max {
				d.Mode = Propose
				p.Findings = append(p.Findings, verdict.Finding{Rule: "act-cap", Where: fmt.Sprintf("#%d", c.Issue),
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
	return p, nil
}

// readRecord finds the report issue and what it records, and looks for
// wrong closings: an issue the role closed, open again, puts that kind of
// act back to propose.
func (p *Plan) readRecord(f forge.Backlog, role string) error {
	open, err := f.Issues()
	if err != nil {
		return err
	}
	for _, is := range open {
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
	isOpen := map[int]bool{}
	for _, is := range open {
		isOpen[is.ID] = true
	}
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
func (p *Plan) check(f forge.Backlog, repo, role string, c Proposal) (rule, why string) {
	if c.Do == "sources" {
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
	} else if !slices.Contains(closeReasons, c.Reason) {
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
	if c.Quote == nil || strings.TrimSpace(c.Quote.Text) == "" {
		return "no-quote", "no quote: an act cites the code or the issue it rests on"
	}
	if !p.found(f, repo, *c.Quote) {
		return "no-quote", fmt.Sprintf("the quote %q is not found where it says", c.Quote.Text)
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
	for _, d := range p.Decisions {
		if d.Act.Do == "sources" && d.Mode == Propose {
			proposed = append(proposed, "- [ ] "+describe(d.Act, "Close"))
			continue
		}
		switch d.Mode {
		case Act:
			undo := " Reopen it to undo."
			if d.Act.Do == "sources" {
				undo = ""
			}
			did = append(did, "- "+describe(d.Act, "Closed")+undo)
		case Propose:
			proposed = append(proposed, "- [ ] "+describe(d.Act, "Close"))
		}
	}
	var b strings.Builder
	b.WriteString("What the product owner did on its last run, and what it proposes. A closing undone (the issue reopened) puts that kind of act back to a person.\n")
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
		b.WriteString("\n## Proposed\n\nFor a person: close the issue if you agree.\n\n" + strings.Join(proposed, "\n") + "\n")
	}
	return b.String()
}

// describe says one closing in a line, its evidence quoted.
func describe(c Proposal, verb string) string {
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
