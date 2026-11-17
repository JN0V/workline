package routing

// The line's timed jobs (ADR-0069 §4): when each is due, read by
// `workline tick`. A job sets `at` (a cron, UTC) or `gap` (minutes since
// its last run started), never both; `off` turns it off. Not set, a job
// keeps the times the shipped workflows give it; none is invented.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Timed is one job's time: At, a cron in UTC; Gap, minutes; Off, not
// timed. Neither At nor Gap: due on its own rule (reconcile: every tick;
// backlog: Ready below its target).
type Timed struct {
	At  string `yaml:"at,omitempty"`
	Gap int    `yaml:"gap,omitempty"`
	Off bool   `yaml:"-"`
}

// UnmarshalYAML takes `off` (or false) for a job turned off, else a
// mapping of at and gap.
func (t *Timed) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		if v := strings.ToLower(n.Value); v == "off" || v == "false" {
			*t = Timed{Off: true}
			return nil
		}
		return fmt.Errorf("line %d: %q: a job's time is {at: <cron>}, {gap: <minutes>} or off", n.Line, n.Value)
	}
	type plain Timed
	var p plain
	if err := n.Decode(&p); err != nil {
		return err
	}
	*t = Timed(p)
	return nil
}

// TimedJobs are the jobs `routing.timed` sets, in the order the tick
// lists them, each with its shipped time.
var TimedJobs = []struct {
	Name string
	Timed
}{
	{"reconcile", Timed{}},              // every tick (ADR-0069 §2)
	{"backlog", Timed{}},                // Ready below its target (#1549)
	{"nightly", Timed{Off: true}},       // no shipped time: a project sets one
	{"docs", Timed{At: "7 18 * * 0"}},   // the gardening template's weekly docs
	{"merge", Timed{Gap: 60}},           // merge's safety net, hourly in the template
	{"sample", Timed{At: "43 5 * * 1"}}, // the weekly sample
	{"seeds", Timed{Off: true}},         // the simulator's new seeds: workline's own
}

// Timing is job's time: the project's, else the shipped one.
func (c *Config) Timing(job string) Timed {
	if t, ok := c.Timed[job]; ok {
		return t
	}
	for _, j := range TimedJobs {
		if j.Name == job {
			return j.Timed
		}
	}
	return Timed{Off: true}
}

// checkTimed refuses a job workline does not know, one setting both at
// and gap, a gap under a minute, a cron it cannot read, and a time on a
// job decided by its own rule (backlog).
func checkTimed(timed map[string]Timed) error {
	names := make([]string, 0, len(TimedJobs))
	for _, j := range TimedJobs {
		names = append(names, j.Name)
	}
	for job, t := range timed {
		switch {
		case !slices.Contains(names, job):
			return fmt.Errorf("routing.timed.%s: no such job; the jobs are %s", job, strings.Join(names, ", "))
		case t.Off:
		case t.At != "" && t.Gap != 0:
			return fmt.Errorf("routing.timed.%s sets both at and gap: a job sets one", job)
		case t.Gap < 0:
			return fmt.Errorf("routing.timed.%s.gap is %d: a number of minutes above 0", job, t.Gap)
		case job == "backlog" && (t.At != "" || t.Gap != 0):
			return fmt.Errorf("routing.timed.backlog: due when Ready is below its target, never on a time: off, or nothing")
		case t.At == "" && t.Gap == 0 && job != "reconcile" && job != "backlog":
			return fmt.Errorf("routing.timed.%s sets neither at nor gap: a job sets one, or is off", job)
		case t.At != "":
			if _, err := ParseCron(t.At); err != nil {
				return fmt.Errorf("routing.timed.%s.at: %w", job, err)
			}
		}
	}
	return nil
}

// Cron is a cron expression of five fields, minute hour day month
// weekday, in UTC: each a set of the values it allows.
type Cron struct {
	fields [5][]bool
	// anyDay, anyWeekday: the day of the month, of the week, is `*`: cron
	// matches either when both are restricted.
	anyDay, anyWeekday bool
}

var cronRanges = [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}

// ParseCron reads a cron expression: `*`, numbers, ranges `a-b`, lists
// `a,b` and steps `*/n`, `a-b/n`; no names, no `@daily`.
func ParseCron(s string) (Cron, error) {
	f := strings.Fields(s)
	if len(f) != 5 {
		return Cron{}, fmt.Errorf("%q: a cron is 5 fields, minute hour day month weekday", s)
	}
	var c Cron
	for i, field := range f {
		lo, hi := cronRanges[i][0], cronRanges[i][1]
		set := make([]bool, hi+1)
		for _, part := range strings.Split(field, ",") {
			rng, step := part, 1
			if r, st, ok := strings.Cut(part, "/"); ok {
				n, err := strconv.Atoi(st)
				if err != nil || n < 1 {
					return Cron{}, fmt.Errorf("%q: step %q is not a number above 0", s, st)
				}
				rng, step = r, n
			}
			a, b := lo, hi
			if rng != "*" {
				x, y, isRange := strings.Cut(rng, "-")
				var err1, err2 error
				a, err1 = strconv.Atoi(x)
				b = a
				if isRange {
					b, err2 = strconv.Atoi(y)
				}
				if err1 != nil || err2 != nil || a < lo || b > hi || a > b {
					return Cron{}, fmt.Errorf("%q: %q is not within %d-%d", s, rng, lo, hi)
				}
				if !isRange && step > 1 {
					b = hi // `5/15`: from 5, every 15
				}
			}
			for v := a; v <= b; v += step {
				set[v] = true
			}
		}
		c.fields[i] = set
	}
	if c.fields[4][7] {
		c.fields[4][0] = true // Sunday, 0 or 7
	}
	c.anyDay, c.anyWeekday = f[2] == "*", f[4] == "*"
	return c, nil
}

// matches says whether the minute t is one of the cron's.
func (c Cron) matches(t time.Time) bool {
	if !c.fields[0][t.Minute()] || !c.fields[1][t.Hour()] || !c.fields[3][int(t.Month())] {
		return false
	}
	day, wd := c.fields[2][t.Day()], c.fields[4][int(t.Weekday())]
	if c.anyDay || c.anyWeekday {
		return day && wd
	}
	return day || wd
}

// Last is the cron's last time at or before now, in UTC, looked for over
// the past year and a day; false when none is found there (a 31 February).
func (c Cron) Last(now time.Time) (time.Time, bool) {
	t := now.UTC().Truncate(time.Minute)
	for end := t.AddDate(-1, 0, -1); !t.Before(end); t = t.Add(-time.Minute) {
		if c.matches(t) {
			return t, true
		}
	}
	return time.Time{}, false
}
