package backlog

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/work"
)

// How far the role goes (ADR-0026): one setting, `autonomy`, a preset of
// the acts' modes and caps the role ships (role.yaml, `levels`); a kind the
// project sets wins over it, field by field; a kind a person undid on an
// issue is proposed there from then on, and everywhere once undone
// undone-max times (ADR-0038, amended).

// The autonomy levels, from the least to the most the role does alone.
const (
	Cautious     = "cautious"
	Normal       = "normal"
	Enterprising = "enterprising"
)

// AutonomyLevels are the autonomy levels a project may pick.
var AutonomyLevels = []string{Cautious, Normal, Enterprising}

// GoneSettings are the settings the role no longer reads, each with why:
// a project still setting one is told so.
var GoneSettings = map[string]string{
	// The pause lived on the report, gone (ADR-0038).
	"ignored-runs-max": "the role keeps no report, and never pauses (ADR-0038)",
	// Its own weekly issue, folded into every role's (ADR-0046).
	"weekly-sample": "the weekly sample of every role's acts draws the product owner's, and says the level they suggest (ADR-0046)",
}

// byLevel is the setting the engine adds beside a role's own: its settings
// as the level alone gives them (role.ByLevel).
const byLevel = "by-level"

// projectSet is the project's own settings for the role, as written
// (role.ProjectSet).
const projectSet = "project-set"

// levelFrom is the setting the engine adds beside it: where the level
// comes from (role.LevelFrom).
const levelFrom = "level-from"

// Where a kind's mode comes from, as the task and the report say it.
const (
	FromLevel   = "level"   // the autonomy level's preset
	FromSetting = "setting" // the project set this kind itself
	FromDemoted = "demoted" // undone undone-max times across the issues: proposed everywhere
)

// Config is how far the role goes in a run, read from its settings.
type Config struct {
	Level        string             // cautious, normal or enterprising
	LevelFrom    string             // where Level comes from (ADR-0060, §3): role setting, project default, role default; "" unknown
	Acts         map[string]Setting // each kind's mode and cap, the project's laid over the level's
	Origins      map[string]string  // each kind's: level or setting
	MovedPercent int                // the share of the open issues a run moves
	NextMax      int                // the ready issues the job's summary lists first; 0 none (ADR-0031)
	StuckDays    int                // the days an issue waits on a person before the summary says it stuck
	// RepeatedPercent: the share of a draft's words already said above it
	// past which it is not written (ADR-0039, amended).
	RepeatedPercent int
	// UndoneMax: the acts of a kind a person undoes, across the issues,
	// before that kind is proposed everywhere; ProposalsMax, the issues
	// waiting on a person before the role reads only those answered
	// (ADR-0038).
	UndoneMax    int
	ProposalsMax int
	// ChangedNeeds: a person's change to a Need or a Scope, or to an
	// imported file's lines, flags the issues built on it (ADR-0032); off
	// unless set (ADR-0038).
	ChangedNeeds bool
	// Gone are the settings the project still sets that the role no
	// longer reads: said, never silently ignored.
	Gone []string
	// Archived are the files no longer a source, as globs: an issue
	// imported from one is not flagged when its lines change (ADR-0032).
	Archived []string
	// SpecReview: the project's line runs the reviewer after the role, so
	// ready is held while the reviewer's important findings on the spec are
	// open (#128); set by the caller, from the routing.
	SpecReview bool
	// Ready: the move a ready makes on the forge's board (ADR-0057): to
	// its ready column, a label on GitLab, a Status option on GitHub; nil
	// none. Set by the caller, from the board.
	Ready *board.Move
	// Tests: the move of a ready that sends the issue to its tests, to
	// the board's tests column (ADR-0059, decision 2), for the issues
	// ToTests says go there; nil, the ready column's. Set by the caller.
	Tests   *board.Move
	ToTests func(forge.Issue) bool
	// Tester: the project's line routes the tester, so a ready holds an
	// issue whose Validation no test can name (#855); set by the caller,
	// from the routing.
	Tester bool
	// Read is what the run read the backlog with, kept with each act it
	// does (ADR-0046); set by the caller, from the run.
	Read Read
	// Priority and Effort: where the project keeps each axis, as its
	// settings say (ADR-0040); unset, workline's own priority labels, and
	// no effort read.
	Priority, Effort scale.Setting
	// CodeRead are the issues whose code this run gave the agent: an
	// effort is estimated on them only (ADR-0040); set by the caller,
	// from what pre wrote (CodeReadFile).
	CodeRead []int
	// CodeGiven are the files whose code this run gave the agent, by
	// issue (the code picker, ADR-0063 §8): sources naming only those
	// leave nothing to read again (#1089). Set by the caller, from what
	// pre wrote (CodeGivenFile).
	CodeGiven map[int][]string
	// CodeWhole are those of CodeGiven given whole, by issue (#1440):
	// sources naming only those, and the issue's own, lack nothing. Set
	// by the caller, from what pre wrote (CodeWholeFile).
	CodeWhole map[int][]string
	// RunAgain: the run is one the line started again at once for the
	// code its agent said it lacked (the input `needs`, ADR-0063 §6),
	// those files given to the issue the event is about (#1146). Set by
	// the caller, from the run's input.
	RunAgain bool
	// Notes are the agent's notes of this run, as it wrote them: the
	// reason said on an issue it left with nothing (ADR-0059, decision 1).
	Notes []string
	// Scope are the issues an event woke the run for (ADR-0059): only
	// their records are read and written (#804); none, every open
	// issue's. Set by the caller, from what pre wrote (ScopeFile).
	Scope []int
	// RevisitFrom: on a blocked issue the line woke the role on, the
	// state it was blocked from, "" unknown: where an unblock may send it
	// besides refine (ADR-0059, amended). Set by the caller, from the
	// run's input.
	RevisitFrom string
	// Wait: how long a question to a person, or a draft no person
	// accepted, waits before the role goes on with its assumption
	// (acts.ask.wait, ADR-0060): never at cautious, 24 h at normal, at
	// once at enterprising.
	Wait Delay
	// Kinds: the project's labels per kind, the first that exists set by
	// the kind act (`kinds`, ADR-0039 (c)); none for a kind, KindLabels'.
	Kinds map[string][]string
	// Sized: the issue the event names, 0 for none; SizeAsked, the line
	// asked the role on its size (the input `size`). A split `too-big`
	// holds on that issue alone, asked or in refine, its size predicted
	// at ready (ADR-0064, §2). Set by the caller, from the run.
	Sized     int
	SizeAsked bool
	// Symptom reads the tester's proof of a bug's parent's symptom, on
	// the issue (ReadSymptom, #1212); nil, none is read. Tested says
	// whether the tester's Draft of an issue holds its tests: a bug is
	// split only then; nil, not known, as before. Set by the caller.
	Symptom func(issue int) (Symptom, bool, error)
	Tested  func(issue int) (bool, error)
	// ModeSet are the kinds whose mode the project wrote itself, even
	// equal to the level's: a `split` it turned off is never turned on for
	// an issue's size (ADR-0064, §3, the panel of 2026-11-08).
	ModeSet []string
}

// ScopeFile is where, in the run's folder, pre lists the issues an event
// woke the run for: none on a run over the whole backlog.
const ScopeFile = "scope"

// CodeReadFile is where, in the run's folder, pre lists the issues whose
// code it gave the agent.
const CodeReadFile = "code-read"

// CodeGivenFile is where, in the run's folder, pre lists the files whose
// code it gave the agent: a line an issue, its number then the paths.
const CodeGivenFile = "code-given"

// CodeWholeFile is where pre lists, the same way, the files of
// CodeGivenFile it gave whole (#1440).
const CodeWholeFile = "code-whole"

// Detects says whether the settings ask where the project keeps an axis:
// the forge's labels are then read to find it (Where).
func (c Config) Detects() bool { return c.Priority.Set() || c.Effort.Set() }

// Read is what a run of the role read: the commit and each open issue's
// body, as its pre step wrote them (ReadFile), and the agents and models
// the run asked, the engine's to say.
type Read struct {
	At     string         `yaml:"at"`
	Bodies map[int]string `yaml:"bodies,omitempty"` // each open issue's body, by its digest (BodyDigest)
	Model  string         `yaml:"-"`
}

// ReadFile is where, in the run's folder, the role's pre step writes what
// it read.
const ReadFile = "read.yaml"

// ReadConfig reads the role's settings: the level, the acts, the moved
// share, what the summary lists first, the undoing, the proposals waiting
// and how much a draft may repeat. A level or a number out of range is an error, never read as a
// default (principle 12).
func ReadConfig(settings map[string]any) (Config, error) {
	c := Config{Level: Normal, Acts: Settings(settings), Origins: map[string]string{},
		MovedPercent: MovedPercent(settings), NextMax: NextMax, StuckDays: StuckDays,
		UndoneMax: UndoneMax, ProposalsMax: ProposalsMax, RepeatedPercent: RepeatedPercentMax}
	if v, ok := settings["autonomy"]; ok && v != nil {
		s, _ := v.(string)
		if !slices.Contains(AutonomyLevels, s) {
			return c, fmt.Errorf("autonomy: %v is not one of %s", v, strings.Join(AutonomyLevels, ", "))
		}
		c.Level = s
	}
	if from, ok := settings[levelFrom].(map[string]any); ok {
		c.LevelFrom, _ = from["autonomy"].(string)
	}
	for _, name := range slices.Sorted(maps.Keys(GoneSettings)) {
		if v, ok := settings[name]; ok && v != nil {
			// A project still setting one is told, never silently ignored
			// (principle 12).
			c.Gone = append(c.Gone, name)
		}
	}
	var err error
	if c.Priority, err = scale.ReadSetting(scale.Priority, settings[scale.Priority.Axis]); err != nil {
		return c, err
	}
	if c.Effort, err = scale.ReadSetting(scale.Effort, settings[scale.Effort.Axis]); err != nil {
		return c, err
	}
	if v, ok := settings["changed-needs"]; ok && v != nil {
		b, isBool := v.(bool)
		if !isBool {
			return c, fmt.Errorf("changed-needs: %v is not true or false", v)
		}
		c.ChangedNeeds = b
	}
	for _, b := range []struct {
		name     string
		min, max int
		to       *int
	}{{"next-max", 0, NextMaxLimit, &c.NextMax}, {"stuck-days", 1, StuckDaysLimit, &c.StuckDays},
		{"undone-max", 1, UndoneMaxLimit, &c.UndoneMax}, {"proposals-max", 1, ProposalsMaxLimit, &c.ProposalsMax},
		{"repeated-percent-max", 1, 100, &c.RepeatedPercent}} {
		if v, ok := settings[b.name]; ok && v != nil {
			n, isNumber := whole(v)
			if !isNumber || n < b.min || n > b.max {
				return c, fmt.Errorf("%s: %v is not a number from %d to %d", b.name, v, b.min, b.max)
			}
			*b.to = n
		}
	}
	if v, ok := settings["archived"]; ok && v != nil {
		list, isList := v.([]any)
		for _, x := range list {
			s, isText := x.(string)
			if !isText || strings.TrimSpace(s) == "" {
				isList = false
				break
			}
			c.Archived = append(c.Archived, s)
		}
		if !isList {
			return c, fmt.Errorf("archived: %v is not a list of file paths or globs", v)
		}
	}
	acts, _ := settings["acts"].(map[string]any)
	ask, _ := acts["ask"].(map[string]any)
	if c.Wait, err = ReadWait(ask["wait"]); err != nil {
		return c, err
	}
	c.Kinds = KindNames(settings)
	own, _ := settings[projectSet].(map[string]any)
	ownActs, _ := own["acts"].(map[string]any)
	for kind, v := range ownActs {
		if m, ok := v.(map[string]any); ok && m["mode"] != nil {
			c.ModeSet = append(c.ModeSet, kind)
		}
	}
	level, _ := settings[byLevel].(map[string]any)
	preset := Settings(level)
	for kind, s := range c.Acts {
		c.Origins[kind] = FromLevel
		if p, ok := preset[kind]; level != nil && (!ok || !same(s, p)) {
			c.Origins[kind] = FromSetting
		}
	}
	return c, nil
}

// whole reads a setting's whole number: 2.5, from JSON, is none —
// never cut to 2.
func whole(v any) (int, bool) {
	if f, ok := v.(float64); ok && f != float64(int(f)) {
		return 0, false
	}
	return number(v)
}

// same says whether two settings of a kind say the same.
func same(a, b Setting) bool {
	days := func(s Setting) int {
		if s.Days == nil {
			return -1
		}
		return *s.Days
	}
	return a.Mode == b.Mode && a.Max == b.Max && a.Rounds == b.Rounds && a.Drafts == b.Drafts &&
		days(a) == days(b) && slices.Equal(a.Exempt, b.Exempt)
}

// KindMode is one kind of act's mode in a run, and where it comes from.
type KindMode struct {
	Kind   string
	Mode   string
	Max    int
	Drafts string // refine: Need and Validation proposed when "propose"
	Origin string
}

// Modes lists each kind of act's mode, the kinds demoted proposed whatever
// the level or the setting.
func (c Config) Modes(demoted []string) []KindMode {
	var out []KindMode
	for _, kind := range ActKinds {
		if kind == "effort" && !c.Effort.Set() {
			continue // read nowhere, said nowhere (ADR-0040)
		}
		s, ok := c.Acts[kind]
		if !ok {
			s = Setting{Mode: Propose}
		}
		m := KindMode{Kind: kind, Mode: s.Mode, Max: s.Max, Drafts: s.Drafts, Origin: c.Origins[kind]}
		if m.Origin == "" {
			m.Origin = FromSetting // not in the role's settings: the project removed it
		}
		if slices.Contains(demoted, kind) && m.Mode == Act {
			m.Mode, m.Origin = Propose, FromDemoted
		}
		out = append(out, m)
	}
	return out
}

// ActKinds are the kinds of act as the settings name them, in the order the
// task and the report list them.
var ActKinds = []string{"sources", "close-duplicate", "close-obsolete", "milestone", "order", "effort", "kind", "size", "refine", "ready", "ask", "split", "depend", "rename", "open"}

// Say says one kind's mode in a few words: "act, 5 a run (level)".
func (m KindMode) Say() string {
	what := m.Mode
	if m.Mode == Act && m.Max > 0 {
		what = fmt.Sprintf("act, %d a run", m.Max)
	}
	if m.Kind == "refine" && m.Mode == Act && m.Drafts == Propose {
		what += fmt.Sprintf(", %s and %s drafts proposed", work.Name(work.Need), work.Name(work.Validation))
	}
	what += " (" + m.Origin + ")"
	if strings.HasPrefix(m.Kind, "close-") && m.Mode == Off {
		what += "; write one all the same when the evidence settles it: said on the issue for a person to close, never closed by the role"
	}
	if m.Kind == "split" && m.Mode == Off {
		what += "; on, `too-big: true`, for an issue the task says is past its size"
	}
	return what
}

// ModesLine says every kind's mode on one line, for the report.
func ModesLine(modes []KindMode) string {
	var parts []string
	for _, m := range modes {
		parts = append(parts, m.Kind+": "+m.Say())
	}
	return strings.Join(parts, "; ")
}
