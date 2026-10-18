package scale

import (
	"slices"
	"strings"
	"testing"
)

// A setting reads as unset, auto, or a family of labels with names mapped;
// anything else is refused, never read as unset.
func TestReadSetting(t *testing.T) {
	for _, c := range []struct {
		v    any
		want Setting
	}{
		{nil, Setting{}},
		{"auto", Setting{In: InAuto}},
		{map[string]any{"in": "auto"}, Setting{In: InAuto}},
		{map[string]any{"in": "labels", "prefix": "P"}, Setting{In: InLabels, Prefix: "P"}},
		{map[string]any{"in": "labels", "prefix": "sev-", "names": map[string]any{"sev-blocker": "Urgent", "sev-minor": "low"}},
			Setting{In: InLabels, Prefix: "sev-", Names: map[string]int{"sev-blocker": 1, "sev-minor": 4}}},
	} {
		got, err := ReadSetting(Priority, c.v)
		if err != nil || got.In != c.want.In || got.Prefix != c.want.Prefix || len(got.Names) != len(c.want.Names) {
			t.Errorf("%v = %+v, %v; want %+v", c.v, got, err, c.want)
		}
		for k, l := range c.want.Names {
			if got.Names[k] != l {
				t.Errorf("%v: %s = %d, want %d", c.v, k, got.Names[k], l)
			}
		}
	}
	for _, v := range []any{"labels", "field", 3, true, map[string]any{"in": "field"}, map[string]any{"in": "labels"},
		map[string]any{"prefix": "P"}, map[string]any{"in": "auto", "prefix": "P"}, map[string]any{"in": "auto", "rank": 1},
		map[string]any{"in": "auto", "names": map[string]any{"sev-1": "blocker"}}, map[string]any{"in": "labels", "prefix": " "}} {
		if s, err := ReadSetting(Priority, v); err == nil || !strings.HasPrefix(err.Error(), "priority: ") {
			t.Errorf("%v = %+v, %v; want refused, the axis named", v, s, err)
		}
	}
}

// Unset, the engine reads as before the detection: workline's own
// priority labels, and no effort.
func TestDetectUnsetIsBefore(t *testing.T) {
	p, err := Detect(Priority, Setting{}, []string{"P0", "P1"}, []string{"P0"}, false)
	if err != nil || p.Family.Prefix != Own.Prefix || p.Words || len(p.Others) != 0 {
		t.Errorf("priority = %+v, %v", p, err)
	}
	e, err := Detect(Effort, Setting{}, []string{"effort/small", "effort/large"}, []string{"effort/small"}, false)
	if err != nil || len(e.Family.Labels) != 0 || e.Said != "" {
		t.Errorf("effort = %+v, %v", e, err)
	}
}

// Auto: the project's own family first, two values or more and one on an
// open issue, numbered from the project's labels, not the issues'.
func TestDetectProjectsFamily(t *testing.T) {
	project := []string{"bug", "P0", "P1", "P2", "P3", "priority/low", "workline:priority/2"}
	p, err := Detect(Priority, Setting{In: InAuto}, project, []string{"bug", "P2"}, false)
	if err != nil || p.Family.Prefix != "P" || p.Words || p.Said != "labels P0 to P3, the project's" {
		t.Fatalf("source = %+v, %v", p, err)
	}
	if l, _ := p.Family.Label(1); l.Name != "P0" || p.Family.Of("P2") != 3 {
		t.Errorf("P0 is Urgent though on no open issue: %+v", p.Family.Labels)
	}
	if got := p.Elsewhere([]string{"bug", "P1", "priority/low", "workline:priority/2"}); !slices.Equal(got, []string{"priority/low", "workline:priority/2"}) {
		t.Errorf("elsewhere = %v; want the other families' labels", got)
	}
	if got := p.Elsewhere([]string{"P1", "bug"}); len(got) != 0 {
		t.Errorf("elsewhere = %v; want none", got)
	}
	// One value, or none on an open issue, is not a family the project uses.
	q, _ := Detect(Priority, Setting{In: InAuto}, []string{"P1", "P-high", "P-low"}, []string{"P1"}, false)
	if q.Family.Prefix != "priority/" || !q.Words {
		t.Errorf("source = %+v; want the forge's words", q)
	}
	if got := q.Elsewhere([]string{"P1", "P-low"}); !slices.Equal(got, []string{"P1", "P-low"}) {
		t.Errorf("elsewhere = %v", got)
	}
}

// Workline's own labels, one on an open issue, are read before the
// forge's words: a project that used them keeps them.
func TestDetectWorklinesOwn(t *testing.T) {
	p, err := Detect(Priority, Setting{In: InAuto}, []string{"workline:priority/3"}, []string{"workline:priority/3"}, false)
	if err != nil || p.Family.Prefix != Own.Prefix || p.Said != "labels workline:priority/*, workline's own" {
		t.Fatalf("source = %+v, %v", p, err)
	}
	if l, ok := p.Family.Label(1); !ok || l.Color == "" {
		t.Errorf("Urgent is written %+v, created as the role's own", l)
	}
	if e, _ := Detect(Effort, Setting{In: InAuto}, []string{"workline:priority/3"}, []string{"workline:priority/3"}, false); e.Family.Prefix != "effort/" {
		t.Errorf("effort = %+v; workline's own are priorities", e)
	}
}

// No family: the forge's words — GitHub's the fields' option names,
// GitLab's ranked priority::1 to ::4 —, created when first set.
func TestDetectForgesWords(t *testing.T) {
	gh, _ := Detect(Priority, Setting{In: InAuto}, []string{"bug"}, nil, false)
	var names []string
	for _, l := range gh.Family.Labels {
		names = append(names, l.Name)
		if l.Rank != 0 || l.Color == "" || l.About == "" {
			t.Errorf("GitHub's %+v", l)
		}
	}
	if !gh.Words || !slices.Equal(names, []string{"priority/urgent", "priority/high", "priority/medium", "priority/low"}) || gh.Said != "labels priority/*, created when first set" {
		t.Errorf("GitHub's words = %v, %q", names, gh.Said)
	}
	gl, _ := Detect(Priority, Setting{In: InAuto}, []string{"bug", "priority::2"}, nil, true)
	names = nil
	for _, l := range gl.Family.Labels {
		names = append(names, l.Name)
		if l.Rank != l.Level {
			t.Errorf("GitLab's %+v: ranked by its level", l)
		}
	}
	if !gl.Words || !slices.Equal(names, []string{"priority::1", "priority::2", "priority::3", "priority::4"}) || gl.Said != "labels priority::*, created" {
		t.Errorf("GitLab's words = %v, %q", names, gl.Said)
	}
	e, _ := Detect(Effort, Setting{In: InAuto}, nil, nil, true)
	if l, _ := e.Family.Label(3); l.Name != "effort::high" || l.Rank != 0 || !strings.Contains(l.About, "weeks") {
		t.Errorf("GitLab's effort High = %+v", l)
	}
	// Words already there, two on open issues, are still the role's: one
	// missing is created when first set.
	w, _ := Detect(Priority, Setting{In: InAuto}, []string{"priority/high", "priority/low"}, []string{"priority/low"}, false)
	if !w.Words || len(w.Family.Labels) != 4 || w.Said != "labels priority/*, created" {
		t.Errorf("source = %+v", w)
	}
}

// The setting's family is read, whatever the others; one the project
// lacks stops the run; a name no table knows is said, unless mapped.
func TestDetectSetting(t *testing.T) {
	project := []string{"sev-1", "sev-2", "sev-blocker", "P0", "P1"}
	p, err := Detect(Priority, Setting{In: InLabels, Prefix: "sev-"}, project, nil, false)
	if err != nil || p.Family.Prefix != "sev-" || p.Said != "labels sev-*, as the settings say" {
		t.Fatalf("source = %+v, %v", p, err)
	}
	if p.Unknown == nil || !strings.Contains(p.Unknown.Error(), `"sev-blocker"`) {
		t.Errorf("unknown = %v; want sev-blocker named", p.Unknown)
	}
	if len(p.Others) != 1 || p.Others[0].Prefix != "P" {
		t.Errorf("others = %+v", p.Others)
	}
	m, err := Detect(Priority, Setting{In: InLabels, Prefix: "sev-", Names: map[string]int{"sev-blocker": 1}}, project, nil, false)
	if err != nil || m.Unknown != nil || m.Family.Of("sev-blocker") != 1 {
		t.Errorf("mapped = %+v, %v", m, err)
	}
	if _, err := Detect(Priority, Setting{In: InLabels, Prefix: "prio-"}, project, nil, false); err == nil || !strings.Contains(err.Error(), "prio-*") {
		t.Errorf("err = %v; want the family missing said", err)
	}
	e, err := Detect(Effort, Setting{In: InLabels, Prefix: "effort"}, []string{"effort1: hours", "effort2: days", "effort3: weeks"}, nil, false)
	if err != nil || e.Said != "labels effort1: hours to effort3: weeks, as the settings say" {
		t.Errorf("effort = %+v, %v", e, err)
	}
}

// Next says each issue's cell of the value-vs-effort grid, its four
// corners named, a value missing said (ADR-0040, "The order").
func TestCells(t *testing.T) {
	for _, c := range []struct {
		p, e int
		want string
	}{
		{2, 1, "High · Effort Low → quick win"},
		{2, 2, "High · Effort Medium"},
		{2, 3, "High · Effort High → big bet"},
		{4, 1, "Low · Effort Low → fill-in"},
		{4, 3, "Low · Effort High → time sink"},
		{1, 0, "Urgent · effort not estimated"},
		{0, 1, "no priority · Effort Low"},
		{0, 0, "no priority, effort not estimated"},
	} {
		if got := Cell(c.p, c.e); got != c.want {
			t.Errorf("Cell(%d, %d) = %q, want %q", c.p, c.e, got, c.want)
		}
	}
	if EffortAnchor(1) != "hours" || EffortAnchor(3) != "weeks" || EffortAnchor(0) != "" || EffortValue(0) != "not estimated" {
		t.Error("anchors and none")
	}
}

// Past 10 values, one over 60% of them is said; ten, or a spread, is not.
func TestStuck(t *testing.T) {
	ten := []int{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	if got := Stuck(Effort, ten); got != "" {
		t.Errorf("ten values: %q", got)
	}
	if got := Stuck(Effort, append(ten, 3)); got != "Medium on 10 of the 11" {
		t.Errorf("eleven: %q", got)
	}
	// Exactly 60% is not over it: 9 of 15.
	if got := Stuck(Effort, []int{2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 1, 1, 3, 3, 3}); got != "" {
		t.Errorf("60%%: %q", got)
	}
	spread := []int{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3}
	if got := Stuck(Effort, spread); got != "" {
		t.Errorf("spread: %q", got)
	}
	if got := Stuck(Priority, []int{1, 1, 1, 1, 1, 1, 1, 2, 3, 4, 4}); got != "Urgent on 7 of the 11" {
		t.Errorf("too many Urgent: %q", got)
	}
}
