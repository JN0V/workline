package backlog

// Questions time out, as lazy consensus (ADR-0060, decision 1): a
// question to an issue's reporter left unanswered for the level's delay
// (acts.ask.wait) is taken as the role's best assumption, written on the
// issue — "Assumption taken: …" — and the issue goes on. A Validation or
// Need draft no person accepted is a question too. The person's Need is
// never rewritten: the assumptions go in a section of the role's own,
// `## Assumptions`, at the end of the body. A person corrects one by
// answering on the issue, or by editing its line.
//
// The delay is counted from the question's comment as the forge dates it
// (its created_at), never from the issue's last update; for drafts, from
// when the forge says the issue got workline:draft, or a person's last
// comment after it.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// Delay is how long a question to a person waits for an answer before the
// role goes on with its assumption: D, or Never, the answer waited for.
type Delay struct {
	D     time.Duration
	Never bool
}

// WaitNever is the setting's word for waiting for the answer.
const WaitNever = "never"

// DefaultWait is the delay when acts.ask.wait is not set: normal's.
var DefaultWait = Delay{D: 24 * time.Hour}

// ReadWait reads acts.ask.wait: `never`, `0`, a Go duration (`24h`,
// `90m`) or days (`2d`). Anything else is an error, never a default
// (principle 12).
func ReadWait(v any) (Delay, error) {
	switch n := v.(type) {
	case nil:
		return DefaultWait, nil
	case int:
		if n == 0 {
			return Delay{}, nil
		}
	case float64:
		if n == 0 {
			return Delay{}, nil
		}
	case string:
		s := strings.TrimSpace(n)
		switch {
		case s == WaitNever:
			return Delay{Never: true}, nil
		case s == "0":
			return Delay{}, nil
		case strings.HasSuffix(s, "d"):
			if days, err := strconv.Atoi(strings.TrimSuffix(s, "d")); err == nil && days >= 0 {
				return Delay{D: time.Duration(days) * 24 * time.Hour}, nil
			}
		default:
			if d, err := time.ParseDuration(s); err == nil && d >= 0 {
				return Delay{D: d}, nil
			}
		}
	}
	return Delay{}, fmt.Errorf("acts.ask.wait: %v is not %s, 0, a duration such as 24h or 30m, nor days such as 2d", v, WaitNever)
}

// Say says the delay as a person reads it: "24 h", "30 min", "at once",
// "until it is answered".
func (w Delay) Say() string {
	switch {
	case w.Never:
		return "until it is answered"
	case w.D == 0:
		return "at once"
	case w.D%time.Hour == 0:
		return fmt.Sprintf("%d h", int(w.D.Hours()))
	case w.D%time.Minute == 0:
		return fmt.Sprintf("%d min", int(w.D.Minutes()))
	}
	return w.D.String()
}

// Passed says whether a silence since since has lasted the delay at now.
func (w Delay) Passed(since, now time.Time) bool {
	return !w.Never && !now.Before(since.Add(w.D))
}

// AskWait reads the delay of a role's settings, as merged by the engine:
// its default when unset or unreadable — ReadConfig says the error.
func AskWait(settings map[string]any) Delay {
	acts, _ := settings["acts"].(map[string]any)
	ask, _ := acts["ask"].(map[string]any)
	w, err := ReadWait(ask["wait"])
	if err != nil {
		return DefaultWait
	}
	return w
}

// Silence is a question to an issue's reporter no comment followed: its
// round, when the forge dated it (Dated false when it does not say), its
// questions.
type Silence struct {
	Round     int
	Since     time.Time
	Dated     bool
	Questions string
}

// Unanswered is the role's last question to an issue's reporter when no
// counted comment came after it (a bot's, an outsider's or a "+1" does
// not answer, ADR-0038); false when there is none, or the last round was
// a proposal to an outsider, whose issue is theirs.
func Unanswered(notes []forge.Note, role string, is forge.Issue) (Silence, bool) {
	var s Silence
	open, round := false, 0
	for _, n := range notes {
		last, engine := EngineMarker(n.Body)
		switch {
		case Round(n.Body, role):
			round++
			open = strings.HasPrefix(last, "<!-- workline:"+role+"/ask")
			s = Silence{Round: round, Questions: askedIn(n.Body)}
			s.Since, s.Dated = readTime(n.Created)
		case !engine && !strings.Contains(n.Body, "<!-- workline:") && countsAnswer(n, is):
			open = false
		}
	}
	return s, open
}

// Due says whether a silence is due for the role's assumption at now:
// past the delay, dated by the forge, not taken already (State.Assumed).
func (s Silence) Due(w Delay, st *State, now time.Time) bool {
	if !s.Dated || st != nil && st.Assumed >= s.Round {
		return false
	}
	return w.Passed(s.Since, now)
}

// SilenceDue is the question of an issue due for an assumption at now,
// read from its comments; false when none is.
func SilenceDue(notes []forge.Note, role string, is forge.Issue, w Delay, now time.Time) (Silence, bool) {
	s, open := Unanswered(notes, role, is)
	if !open {
		return s, false
	}
	st, _, _ := ReadState(forge.Bodies(notes), role)
	return s, s.Due(w, st, now)
}

// OnlyDrafts says whether all that keeps an issue from ready is a Need or
// Validation draft of the role's no person accepted: its sections there.
func OnlyDrafts(is forge.Issue, st *State) bool {
	return !Accepted(is) && len(NotReady(is, st, true)) == 0 && len(NotReady(is, st, false)) > 0
}

// DraftsSince is when the drafts of an issue started waiting on a
// person: the forge's time of its workline:draft label, or a counted
// comment after it — a person who wrote since is waited for again. With
// no time for the label — drafts whose marks came in the body, never
// labelled (#1021) —, from the product owner's first comment on it.
func DraftsSince(labeled string, notes []forge.Note, is forge.Issue) (time.Time, bool) {
	at, ok := readTime(labeled)
	if !ok {
		at, ok = firstRead(notes)
	}
	if !ok {
		return time.Time{}, false
	}
	for _, n := range Counted(notes, is) {
		if t, known := readTime(n.Created); known && t.After(at) {
			at = t
		}
	}
	return at, true
}

// firstRead is when the product owner first wrote on an issue, as the
// forge dates it: its state, an ask or a proposal; false when the forge
// dates none.
func firstRead(notes []forge.Note) (time.Time, bool) {
	for _, n := range notes {
		if Round(n.Body, Keeper) || strings.Contains(n.Body, "<!-- workline:sticky="+Keeper+"/") {
			if t, ok := readTime(n.Created); ok {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// DraftNames are the headings of an issue's drafts no person accepted.
func DraftNames(is forge.Issue, st *State) []string {
	have := work.ByMeaning(is.Body)
	var out []string
	for _, m := range []work.Meaning{work.Need, work.Validation} {
		if strings.TrimSpace(have[m].Text) != "" && IsDraft(is.Body, m, st) {
			out = append(out, have[m].Heading)
		}
	}
	return out
}

// AssumptionsName heads the section the role writes its assumptions in,
// its own, at the end of the body.
const AssumptionsName = "Assumptions"

// AssumedLead starts each assumption's line, and its line in the role's
// comment.
const AssumedLead = "Assumption taken"

// WithAssumption is a body with one more line under its Assumptions
// section, the section added at the end — above a footer — when it has
// none. The issue's other sections are left as they are.
func WithAssumption(body, line string) string {
	body, footer := work.Footer(body)
	if text, ok := work.Sections(body)[AssumptionsName]; ok {
		return work.WithFooter(fill(body, AssumptionsName, strings.TrimRight(strings.TrimSpace(text)+"\n"+line, "\n")), footer)
	}
	return work.WithFooter(strings.TrimRight(body, " \r\n")+"\n\n## "+AssumptionsName+"\n\n"+line, footer)
}

// Assumptions are the lines of an issue's Assumptions section.
func Assumptions(body string) []string {
	body, _ = work.Footer(body)
	var out []string
	for _, l := range strings.Split(work.Sections(body)[AssumptionsName], "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// AssumptionLine is one line of the Assumptions section: when, why, what.
func AssumptionLine(day, how, text string) string {
	text = strings.Join(strings.Fields(text), " ")
	return fmt.Sprintf("- %s on %s, %s: %s", AssumedLead, day, how, strings.TrimSuffix(text, ".")+".")
}

// answeredMark ends an assumption's line a person answered after.
const answeredMark = " Answered by @"

// WithAnswered is a body whose assumptions not answered yet say who
// answered after them, and when: their word stands over them.
func WithAnswered(body, who, day string) string {
	lines := Assumptions(body)
	changed := false
	for i, l := range lines {
		if strings.HasPrefix(l, "- "+AssumedLead) && !strings.Contains(l, answeredMark) {
			lines[i] = l + fmt.Sprintf("%s%s on %s: their answer stands over it.", answeredMark, who, day)
			changed = true
		}
	}
	if !changed {
		return body
	}
	main, footer := work.Footer(body)
	return work.WithFooter(fill(main, AssumptionsName, strings.Join(lines, "\n")), footer)
}

// AnsweredAfter says who answered an issue's assumptions: a counted
// comment written after the role took its last one, the author of the
// last; "" when none did, or none is open.
func AnsweredAfter(is forge.Issue, st *State, notes []forge.Note) string {
	if st == nil || st.AssumedAt == "" || !hasOpenAssumption(is.Body) {
		return ""
	}
	at, ok := readTime(st.AssumedAt)
	if !ok {
		return ""
	}
	who := ""
	for _, n := range Counted(notes, is) {
		if t, known := readTime(n.Created); known && t.After(at) {
			who = n.Author
		}
	}
	return who
}

// hasOpenAssumption says whether a body holds an assumption no person
// answered after.
func hasOpenAssumption(body string) bool {
	for _, l := range Assumptions(body) {
		if strings.HasPrefix(l, "- "+AssumedLead) && !strings.Contains(l, answeredMark) {
			return true
		}
	}
	return false
}

// checkAssume checks an assumption against the issue as it is: its
// question unanswered past the delay — or asked in this run at a level
// that goes on at once —, not taken already; the engine's own mark of a
// person's answer after the assumptions, that one came.
func (p *Plan) checkAssume(f forge.Backlog, role string, c *Proposal) (rule, why string) {
	if c.Do != "assume" {
		return "", ""
	}
	is, ok := p.issues[c.Issue]
	if !ok || c.Issue == p.Report {
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	}
	notes, err := f.Notes(forge.Target{Kind: "issue", ID: c.Issue})
	if err != nil {
		return "no-state", err.Error()
	}
	st, _, _ := ReadState(forge.Bodies(notes), role)
	if c.Answered != "" {
		if AnsweredAfter(is, st, notes) == "" {
			return "assume-not-answered", "no person wrote after its assumptions: they stand"
		}
		return "", ""
	}
	if strings.TrimSpace(c.Assumption) == "" {
		return "assume-empty", "nothing assumed: the assumption was empty"
	}
	w, now := p.config.Wait, clock.Now()
	s, open := Unanswered(notes, role, is)
	switch round := p.askedNow[c.Issue]; {
	case round > 0 && !w.Never && w.D == 0:
		// Asked in this run, at a level that goes on at once.
		c.Round, c.Asked = round, ""
		c.Written = AssumptionLine(today(), "asked in the same run, the role going on at once (acts.ask.wait 0)", c.Assumption)
		return "", ""
	case round > 0:
		return "assume-early", fmt.Sprintf("asked in this run: its answer is waited for %s (acts.ask.wait) before an assumption is taken", w.Say())
	case !open:
		return "assume-nothing-asked", "no question to its reporter waits for an answer: nothing to assume"
	case w.Never:
		return "assume-waits", "the role waits for the answer (autonomy cautious, or acts.ask.wait never)"
	case !s.Dated:
		return "assume-undated", "the forge does not say when the question was asked: its answer is waited for"
	case st != nil && st.Assumed >= s.Round:
		return "assumed-already", "an assumption was taken on this question already"
	case !w.Passed(s.Since, now):
		return "assume-early", fmt.Sprintf("asked on %s: an assumption is taken from %s (acts.ask.wait %s)",
			s.Since.UTC().Format("2006-01-02 15:04 UTC"), s.Since.Add(w.D).UTC().Format("2006-01-02 15:04 UTC"), w.Say())
	}
	c.Round, c.Asked = s.Round, s.Since.UTC().Format(dateLayout)
	c.Written = AssumptionLine(today(), fmt.Sprintf("the question of %s unanswered after %s", c.Asked, w.Say()), c.Assumption)
	return "", ""
}

// assumable are the drafts of an issue a ready takes as assumptions —
// all that keeps it from ready, no person's answer for the level's delay
// —, and the line it writes; none, and when they will be, said after a
// not-ready.
func (p *Plan) assumable(f forge.Backlog, is forge.Issue, st *State) (names []string, written, later string) {
	w := p.config.Wait
	if w.Never || !OnlyDrafts(is, st) {
		return nil, "", ""
	}
	if names = DraftNames(is, st); len(names) == 0 {
		return nil, "", ""
	}
	drafts := "draft"
	if len(names) > 1 {
		drafts = "drafts"
	}
	how := fmt.Sprintf("the %s %s, the role going on at once (acts.ask.wait 0)", and(names), drafts)
	if w.D > 0 {
		if f == nil {
			return nil, "", ""
		}
		trail, err := f.Trail(is.ID, LabelDraft)
		if err != nil {
			return nil, "", ""
		}
		notes, err := f.Notes(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil, "", ""
		}
		since, known := DraftsSince(trail.Labeled, notes, is)
		switch {
		case !known:
			return nil, "", "; the forge does not say when its drafts were written: they wait for a person"
		case !w.Passed(since, clock.Now()):
			return nil, "", fmt.Sprintf("; its drafts are taken as assumptions from %s, with no answer (acts.ask.wait %s)",
				since.Add(w.D).UTC().Format("2006-01-02 15:04 UTC"), w.Say())
		}
		how = fmt.Sprintf("the %s %s unanswered after %s", and(names), drafts, w.Say())
	}
	return names, AssumptionLine(today(), how, "written as the role drafted it, for a person to correct"), ""
}
