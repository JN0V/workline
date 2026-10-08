package backlog

import (
	"fmt"
	"slices"
	"strings"
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

// byLevel is the setting the engine adds beside a role's own: its settings
// as the level alone gives them (role.ByLevel).
const byLevel = "by-level"

// Where a kind's mode comes from, as the task and the report say it.
const (
	FromLevel   = "level"   // the autonomy level's preset
	FromSetting = "setting" // the project set this kind itself
	FromDemoted = "demoted" // undone undone-max times across the issues: proposed everywhere
)

// Config is how far the role goes in a run, read from its settings.
type Config struct {
	Level        string             // cautious, normal or enterprising
	Acts         map[string]Setting // each kind's mode and cap, the project's laid over the level's
	Origins      map[string]string  // each kind's: level or setting
	MovedPercent int                // the share of the open issues a run moves
	NextMax      int                // the ready issues the job's summary lists first; 0 none (ADR-0031)
	StuckDays    int                // the days an issue waits on a person before the summary says it stuck
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
}

// ReadConfig reads the role's settings: the level, the acts, the moved
// share, what the summary lists first, the undoing and the proposals
// waiting. A level or a number out of range is an error, never read as a
// default (principle 12).
func ReadConfig(settings map[string]any) (Config, error) {
	c := Config{Level: Normal, Acts: Settings(settings), Origins: map[string]string{},
		MovedPercent: MovedPercent(settings), NextMax: NextMax, StuckDays: StuckDays,
		UndoneMax: UndoneMax, ProposalsMax: ProposalsMax}
	if v, ok := settings["autonomy"]; ok && v != nil {
		s, _ := v.(string)
		if !slices.Contains(AutonomyLevels, s) {
			return c, fmt.Errorf("autonomy: %v is not one of %s", v, strings.Join(AutonomyLevels, ", "))
		}
		c.Level = s
	}
	if v, ok := settings["ignored-runs-max"]; ok && v != nil {
		// The pause lived on the report, gone (ADR-0038): a project still
		// setting it is told, never silently ignored (principle 12).
		c.Gone = append(c.Gone, "ignored-runs-max")
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
		{"undone-max", 1, UndoneMaxLimit, &c.UndoneMax}, {"proposals-max", 1, ProposalsMaxLimit, &c.ProposalsMax}} {
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
var ActKinds = []string{"sources", "close-duplicate", "close-obsolete", "milestone", "order", "refine", "ready", "ask", "split", "depend", "rename", "open"}

// Say says one kind's mode in a few words: "act, 5 a run (level)".
func (m KindMode) Say() string {
	what := m.Mode
	if m.Mode == Act && m.Max > 0 {
		what = fmt.Sprintf("act, %d a run", m.Max)
	}
	if m.Kind == "refine" && m.Mode == Act && m.Drafts == Propose {
		what += ", Need and Validation drafts proposed"
	}
	return what + " (" + m.Origin + ")"
}

// ModesLine says every kind's mode on one line, for the report.
func ModesLine(modes []KindMode) string {
	var parts []string
	for _, m := range modes {
		parts = append(parts, m.Kind+": "+m.Say())
	}
	return strings.Join(parts, "; ")
}
