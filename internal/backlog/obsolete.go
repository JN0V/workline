package backlog

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
)

// An issue the code made obsolete is announced on it first, then closed at
// a later run if nobody wrote, the label is still there and a second judge
// agrees (ADR-0024).

// LabelObsolete marks an issue announced as obsolete: taking it off keeps
// the issue open.
const LabelObsolete = "workline:obsolete"

// ObsoleteDays is how long an announcement waits, in days, when the
// role's settings do not say.
const ObsoleteDays = 7

// DefaultExempt are the labels that keep an issue from being announced
// obsolete, when the role's settings do not say.
var DefaultExempt = []string{"pinned", "security"}

// dateLayout is how an announcement's date is written.
const dateLayout = "2006-01-02"

// Announcement is what the engine wrote when it announced an issue
// obsolete, read back from its comment's block.
type Announcement struct {
	Quote     Quote  `yaml:"quote"`
	Why       string `yaml:"why"`
	Commit    string `yaml:"commit"`
	Announced string `yaml:"announced"`    // the day it was announced, YYYY-MM-DD
	By        string `yaml:"by,omitempty"` // the model that proposed it
}

// Key is the evidence an announcement rests on: kept open once, not
// announced again for it.
func (a Announcement) Key() string { return ObsoleteKey(a.Quote) }

// ObsoleteKey names a quote's evidence: its file or issue, and a digest of
// its text, spaces aside.
func ObsoleteKey(q Quote) string {
	sum := sha256.Sum256([]byte(squeeze(q.Text)))
	where := q.Path
	if q.Issue > 0 {
		where = fmt.Sprintf("#%d", q.Issue)
	}
	return fmt.Sprintf("%s:%x", where, sum[:6])
}

// AnnounceMarker marks the comment announcing an issue obsolete.
func AnnounceMarker(role string) string { return forge.Marker(role + "/obsolete") }

// KeptMarker marks the comment saying why an announced issue stays open.
func KeptMarker(role string) string { return forge.Marker(role + "/obsolete-kept") }

// Delay reads close-obsolete's delay; ObsoleteDays when it is not set.
func (s Setting) Delay() int {
	if s.Days == nil {
		return ObsoleteDays
	}
	return max(*s.Days, 0)
}

// exempt reads close-obsolete's exempt labels; DefaultExempt when not set.
func (s Setting) exempt() []string {
	if s.Exempt == nil {
		return DefaultExempt
	}
	return s.Exempt
}

// ExemptLabel is the first of an issue's labels that keeps it from being
// announced or closed as obsolete, or "".
func ExemptLabel(is forge.Issue, s Setting) string {
	for _, l := range is.Labels {
		if slices.Contains(s.exempt(), l) {
			return l
		}
	}
	return ""
}

// AnnouncementComment is the comment announcing an issue obsolete: to its
// reporter and its watchers, why, the code quoted, from when it may be
// closed and how to keep it open; then the block the engine reads back.
func AnnouncementComment(author string, c Proposal, role string, a Announcement, days int) string {
	who := ""
	if author != "" {
		who = "@" + author + ", "
	}
	when, _ := time.Parse(dateLayout, a.Announced)
	from := when.AddDate(0, 0, days).Format(dateLayout)
	why := strings.ReplaceAll(strings.TrimSpace(c.Why), "```", "'''")
	var b strings.Builder
	fmt.Fprintf(&b, "%sthis issue looks obsolete: the code it is about changed (commit %s).\n\n", who, a.Commit)
	fmt.Fprintf(&b, "%s %s\n\n", cite(a.Quote), why)
	fmt.Fprintf(&b, "The %s will close it at a run from %s, if a second judge agrees, unless someone writes here or takes the label `%s` off. "+
		"If it is not solved, say so, or take the label off: it stays open, and is not announced again for this code. Closed, it can be reopened.\n\n",
		strings.ReplaceAll(role, "-", " "), from, LabelObsolete)
	kept := a
	kept.Why = why
	data, _ := yaml.Marshal(kept)
	b.WriteString(engineBlock + "\n\n```yaml\n" + string(data) + "```\n</details>")
	return b.String()
}

// LastAnnouncement reads the last announcement on an issue, and the notes
// written after it; nil when there is none, or it does not read.
func LastAnnouncement(notes []forge.Note, role string) (*Announcement, []forge.Note) {
	for i := len(notes) - 1; i >= 0; i-- {
		if last, engine := EngineMarker(notes[i].Body); !engine || last != AnnounceMarker(role) {
			continue
		}
		at := strings.LastIndex(notes[i].Body, engineBlock)
		if at < 0 {
			return nil, nil
		}
		m := fenced.FindStringSubmatch(notes[i].Body[at:])
		if m == nil {
			return nil, nil
		}
		var a Announcement
		if yaml.Unmarshal([]byte(m[1]), &a) != nil || a.Quote.Text == "" {
			return nil, nil
		}
		return &a, notes[i+1:]
	}
	return nil, nil
}

// Obsolete is where an issue's announcement stands.
type Obsolete struct {
	Announcement *Announcement // nil: none waiting
	Due          bool          // its delay passed, nothing cancels it: a judge decides
	From         string        // the day it may be closed from
	Keep         string        // why it is kept open, when it is
	Say          bool          // the issue is told why it is kept: no person did it
}

// ReadObsolete says where an issue's announcement stands: none waiting
// (none, or settled already — its evidence in the state's kept), kept open
// (someone wrote, the label taken off, an exempt label, the code quoted
// gone), waiting for its day, or due for the judge.
func ReadObsolete(repo string, is forge.Issue, notes []forge.Note, st *State, role string, s Setting, now time.Time) Obsolete {
	a, after := LastAnnouncement(notes, role)
	if a == nil || slices.Contains(st.Kept, a.Key()) {
		return Obsolete{}
	}
	o := Obsolete{Announcement: a}
	when, err := time.Parse(dateLayout, a.Announced)
	if err != nil {
		o.Keep, o.Say = "its announcement has no date that reads", true
		return o
	}
	o.From = when.AddDate(0, 0, s.Delay()).Format(dateLayout)
	for _, n := range after {
		if _, engine := EngineMarker(n.Body); engine || n.Bot {
			continue
		}
		who := "someone"
		if n.Author != "" {
			who = "@" + n.Author
		}
		o.Keep = who + " wrote on it after it was announced obsolete"
		return o
	}
	switch {
	case !slices.Contains(is.Labels, LabelObsolete):
		o.Keep = "the label " + LabelObsolete + " was taken off"
	case ExemptLabel(is, s) != "":
		o.Keep = "it bears the label " + ExemptLabel(is, s) + ", exempt"
	case !quoteInRepo(repo, a.Quote):
		o.Keep, o.Say = "the code quoted is no longer there: the evidence changed", true
	case now.Format(dateLayout) >= o.From:
		o.Due = true
	}
	return o
}

// quoteInRepo says whether a file quote is still in the commit the run is
// on; a quote from an issue is taken as there.
func quoteInRepo(repo string, q Quote) bool {
	if q.Path == "" {
		return true
	}
	return (&Plan{}).found(nil, repo, q)
}

// Judged is a second judge's answer on an announced issue (ADR-0005),
// written by the engine beside the question pre asked.
type Judged struct {
	Yes    *bool  `yaml:"yes"`
	Why    string `yaml:"why"`
	Model  string `yaml:"model"`
	Level  string `yaml:"level"`
	Author string `yaml:"author"`
	Error  string `yaml:"error"`
}

// Says is the judge's answer as a line: yes or no, why, and how far it
// stood from the model that proposed it.
func (j Judged) Says() string {
	v := "no"
	if j.Yes != nil && *j.Yes {
		v = "yes"
	}
	who := j.Model
	if j.Author != "" {
		who = j.Author + " → " + j.Model
	}
	return fmt.Sprintf("%s: %s (independence: %s, %s)", v, strings.TrimSuffix(strings.TrimSpace(j.Why), "."), j.Level, who)
}

// JudgeKeyPrefix names the folder of a judge's question on an announced
// issue, its number after it: in/judge/obsolete-12/.
const JudgeKeyPrefix = "obsolete-"

// JudgeQuestion is what the second judge is asked of an announced issue.
const JudgeQuestion = "Is this issue solved, or made moot, by the code as it is now, so that closing it as obsolete is right? Answer no if any part of what it asks or reports is still true, if the code quoted only looks related, or if the material does not settle it."

// JudgeMaterial is what the second judge reads: the issue, the
// announcement's evidence, and the code quoted as it is now.
func JudgeMaterial(repo string, is forge.Issue, a Announcement, code string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The issue: #%d %s\n\n%s\n\n", is.ID, is.Title, strings.TrimSpace(is.Body))
	fmt.Fprintf(&b, "## Why it was announced obsolete\n\n%s %s\n\nAnnounced on %s, at commit %s.\n\n", cite(a.Quote), a.Why, a.Announced, a.Commit)
	if code != "" {
		fmt.Fprintf(&b, "## %s, as it is now\n\n```\n%s\n```\n", a.Quote.Path, code)
	}
	return b.String()
}

// KeptComment tells an announced issue why it stays open, when no person
// kept it.
func KeptComment(why string) string {
	return fmt.Sprintf("Kept open: %s. The label `%s` is taken off; it is not announced obsolete again for this code.", strings.TrimSpace(why), LabelObsolete)
}

// ClosingComment is what an obsolete issue is told as it is closed after
// its announcement.
func ClosingComment(c Proposal, role string) string {
	return fmt.Sprintf("Obsolete: the code it is about changed.\n\n%s %s\n\nAnnounced on %s; nobody wrote since, and a second judge agreed — %s.\n\nClosed by the %s role. Reopen it to undo: a closing undone puts this kind of act back to a person.",
		cite(*c.Quote), strings.TrimSpace(c.Why), c.Announced, c.Judge, role)
}
