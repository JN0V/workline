package backlog

import (
	"fmt"
	"slices"
	"strings"
)

// How far the role goes (ADR-0026): one setting, `autonomy`, a preset of
// the acts' modes and caps the role ships (role.yaml, `levels`); a kind the
// project sets wins over it, field by field; a kind demoted by a person's
// undoing stays proposed whatever the level, until a person's tick.

// The autonomy levels, from the least to the most the role does alone.
const (
	Cautious     = "cautious"
	Normal       = "normal"
	Enterprising = "enterprising"
)

// AutonomyLevels are the autonomy levels a project may pick.
var AutonomyLevels = []string{Cautious, Normal, Enterprising}

// IgnoredRunsMax bounds `ignored-runs-max`; 0 never pauses.
const IgnoredRunsMax = 20

// byLevel is the setting the engine adds beside a role's own: its settings
// as the level alone gives them (role.ByLevel).
const byLevel = "by-level"

// Where a kind's mode comes from, as the task and the report say it.
const (
	FromLevel   = "level"   // the autonomy level's preset
	FromSetting = "setting" // the project set this kind itself
	FromDemoted = "demoted" // a person undid one of its acts: proposed until a tick sets it back
)

// Config is how far the role goes in a run, read from its settings.
type Config struct {
	Level        string             // cautious, normal or enterprising
	Acts         map[string]Setting // each kind's mode and cap, the project's laid over the level's
	Origins      map[string]string  // each kind's: level or setting
	MovedPercent int                // the share of the open issues a run moves
	IgnoredMax   int                // runs nobody answered before the role pauses; 0 never
	NextMax      int                // the ready issues the report lists first; 0 none (ADR-0031)
	StuckDays    int                // the days an issue waits on a person before the report says it stuck
	// SpecReview: the project's line runs the reviewer after the role, so
	// ready is held while the reviewer's important findings on the spec are
	// open (#128); set by the caller, from the routing.
	SpecReview bool
}

// ReadConfig reads the role's settings: the level, the acts, the moved
// share, the runs before a pause, and what the report lists first. A level or a number out of range is an
// error, never read as a default (principle 12).
func ReadConfig(settings map[string]any) (Config, error) {
	c := Config{Level: Normal, Acts: Settings(settings), Origins: map[string]string{},
		MovedPercent: MovedPercent(settings), IgnoredMax: 3, NextMax: NextMax, StuckDays: StuckDays}
	if v, ok := settings["autonomy"]; ok && v != nil {
		s, _ := v.(string)
		if !slices.Contains(AutonomyLevels, s) {
			return c, fmt.Errorf("autonomy: %v is not one of %s", v, strings.Join(AutonomyLevels, ", "))
		}
		c.Level = s
	}
	if v, ok := settings["ignored-runs-max"]; ok && v != nil {
		n, isNumber := whole(v)
		if !isNumber || n < 0 || n > IgnoredRunsMax {
			return c, fmt.Errorf("ignored-runs-max: %v is not a number from 0 (never pause) to %d", v, IgnoredRunsMax)
		}
		c.IgnoredMax = n
	}
	for _, b := range []struct {
		name     string
		min, max int
		to       *int
	}{{"next-max", 0, NextMaxLimit, &c.NextMax}, {"stuck-days", 1, StuckDaysLimit, &c.StuckDays}} {
		if v, ok := settings[b.name]; ok && v != nil {
			n, isNumber := whole(v)
			if !isNumber || n < b.min || n > b.max {
				return c, fmt.Errorf("%s: %v is not a number from %d to %d", b.name, v, b.min, b.max)
			}
			*b.to = n
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

// SuggestAfter is the proposals a person settled at a level before the
// report suggests another; SuggestTicked, the share ticked, in percent,
// past which a cautious role is told it could do more alone.
const (
	SuggestAfter  = 10
	SuggestTicked = 80
)

// Measure is what a person did with the proposals at the level in force:
// the report suggests another level from it, never changes the setting.
type Measure struct {
	Level  string `yaml:"level"`
	Ticked int    `yaml:"ticked,omitempty"` // proposals a person of the project ticked: done as proposed
	Other  int    `yaml:"other,omitempty"`  // proposals settled otherwise: their issue closed, or opened by hand
}

// Suggest is the level the report suggests, and why; "" when none.
func (m *Measure) Suggest() (string, string) {
	if m == nil {
		return "", ""
	}
	total := m.Ticked + m.Other
	if m.Level == Cautious && total >= SuggestAfter && m.Ticked*100 > SuggestTicked*total {
		return Normal, fmt.Sprintf("%d of the %d proposals settled at cautious were ticked as proposed (more than %d%%)", m.Ticked, total, SuggestTicked)
	}
	return "", ""
}
