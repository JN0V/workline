package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/verdict"
)

// On is what an issue carries where an axis may be kept: its labels, its
// fields' values, its weight.
func On(is forge.Issue) scale.On {
	return scale.On{Labels: is.Labels, Fields: is.Fields, Weight: is.Weight}
}

// PriorityLabels lists the levels an issue's priority gives, read by name
// where the project keeps it (scale.Used) — its labels, or its field —;
// more than one is a person's doing.
func PriorityLabels(is forge.Issue) []int { return scale.Used.Priority.Read(On(is)) }

// Priority is an issue's level, the most pressing it carries; 0 when it
// has none.
func Priority(is forge.Issue) int { return first(PriorityLabels(is)) }

// EffortLabels lists the levels an issue's effort gives, where the
// project keeps it — its labels, its field, its GitLab weight read in
// bands —; none while it keeps none (ADR-0040).
func EffortLabels(is forge.Issue) []int { return scale.Used.Effort.Read(On(is)) }

// Effort is an issue's effort, the lightest it carries; 0 when it has
// none, or the project keeps none.
func Effort(is forge.Issue) int { return first(EffortLabels(is)) }

// first is the lowest of levels, the first in the order's direction; 0
// for none.
func first(levels []int) int {
	if len(levels) == 0 {
		return 0
	}
	return slices.Min(levels)
}

// Where reads where the project keeps each axis (ADR-0040, "Where the
// values live"), with no agent, as its settings ask; unset, as before
// (scale.Before), nothing read of the forge. With it, what the run says
// once: where each axis was found, and the names of a family no table
// knows. A setting naming a family the project lacks is an error: the
// run stops.
func Where(f forge.Backlog, cfg Config, open []forge.Issue) (scale.Scheme, []verdict.Finding, error) {
	if !cfg.Detects() {
		return scale.Before, nil, nil
	}
	labels, err := forge.ProjectLabels(f, open)
	if err != nil {
		return scale.Before, nil, err
	}
	var on []string
	for _, is := range open {
		on = append(on, is.Labels...)
	}
	// The forge's issue fields, GitHub's: a value visible to members only
	// on a public repository is hidden, a visibility not read counting as
	// hidden; a token that reads none of them is said blind (ADR-0040).
	fs, err := forge.IssueFields(f)
	if err != nil {
		return scale.Before, nil, err
	}
	// GitLab's weight, Premium's and Ultimate's: their issues carry the key.
	weights, err := forge.Weights(f)
	if err != nil {
		return scale.Before, nil, err
	}
	var fields []scale.Field
	for _, x := range fs.List {
		sf := scale.Field{ID: x.ID, Name: x.Name, Options: x.Options, Hidden: !x.Public && fs.Public}
		if !x.Public {
			sf.Blind = fs.Blind
		}
		fields = append(fields, sf)
	}
	s := scale.Scheme{Detected: true}
	var said []string
	var unknown []verdict.Finding
	for _, a := range []struct {
		scale scale.Scale
		set   scale.Setting
		to    *scale.Source
	}{{scale.Priority, cfg.Priority, &s.Priority}, {scale.Effort, cfg.Effort, &s.Effort}} {
		src, err := scale.Detect(a.scale, a.set, labels, on, forge.ScopedLabels(f), fields, weights)
		if err != nil {
			return scale.Before, nil, err
		}
		*a.to = src
		where := src.Said
		if where == "" {
			where = "read nowhere, its setting unset"
		}
		if src.Field.Blind != "" {
			where += " — " + src.Field.Blind + ": nothing written there"
		}
		said = append(said, strings.ToUpper(a.scale.Axis[:1])+a.scale.Axis[1:]+": "+where+".")
		if src.Unknown != nil {
			unknown = append(unknown, verdict.Finding{Rule: "level-unknown", Level: "info",
				Message: src.Unknown.Error() + "; a name is mapped to a level under `names` in the settings"})
		}
	}
	return s, append([]verdict.Finding{{Rule: "values-where", Level: "info", Message: strings.Join(said, " ")}}, unknown...), nil
}

// Placeable says why a priority or an effort cannot be written on an
// issue where the project keeps it, src (ADR-0040): a token that reads
// none of the field's values, never read as empty; a parent's GitLab
// weight, summed from its parts'; a value in another place — never
// written over, nor mirrored —; no label of the family, nor option of the
// field, carrying the level; "" when it can. The plan asks it, and the
// applying job again before it writes.
func Placeable(src scale.Source, is forge.Issue, l int) (rule, why string) {
	axis := src.Family.Scale.Axis
	where := src.Place()
	if src.Field.Blind != "" {
		return axis + "-blind", fmt.Sprintf("%s: its values are never read as empty, and nothing is written in %s", src.Field.Blind, where)
	}
	if parts := Parts(is); src.Weight && len(parts) > 0 {
		return axis + "-parent", fmt.Sprintf("a parent of %s: GitLab sums its parts' weights into its own, never written", issueList(parts))
	}
	if there := src.Elsewhere(On(is)); len(there) > 0 {
		var here []string
		for _, n := range is.Labels {
			if src.Family.Of(n) > 0 && !src.Native() {
				here = append(here, n)
			}
		}
		if len(src.Read(On(is))) > 0 && src.Native() {
			here = append(here, where)
		}
		if len(here) > 0 {
			return axis + "-elsewhere", fmt.Sprintf("its %s is in two places, %s and %s: nothing is written on it; a person keeps one",
				axis, strings.Join(here, ", "), strings.Join(there, ", "))
		}
		return axis + "-elsewhere", fmt.Sprintf("its %s is in %s, not in %s where the project keeps it: nothing is written on it, never in two places",
			axis, strings.Join(there, ", "), where)
	}
	if _, ok := src.Family.Label(l); !ok {
		what := "no label of " + where
		if src.Native() {
			what = "no option of " + where
		}
		return axis + "-no-label", fmt.Sprintf("%s carries %s: a person creates one, or maps one to it under `names` in the settings",
			what, src.Family.Scale.Name(l))
	}
	return "", ""
}

// Before says an issue's place in the order, to put it back: its
// priority and its milestone.
func Before(is forge.Issue) string {
	m := "no milestone"
	if is.Milestone != "" {
		m = "milestone " + is.Milestone
	}
	return scale.Said(Priority(is)) + ", " + m
}

// Released says whether a tag named after the milestone exists: the
// release it was for is out.
func Released(repo, milestone string) bool {
	if milestone == "" {
		return false
	}
	return exec.Command("git", "-C", repo, "rev-parse", "-q", "--verify", "refs/tags/"+milestone).Run() == nil
}

// NextMilestone is the nearest open milestone not released, in the
// milestones' order (CompareDue); "" when there is none.
func NextMilestone(repo string, open []forge.Milestone) string {
	var left []forge.Milestone
	for _, m := range open {
		if !Released(repo, m.Title) {
			left = append(left, m)
		}
	}
	sort.SliceStable(left, func(i, j int) bool { return CompareDue(left[i], left[j]) < 0 })
	if len(left) == 0 {
		return ""
	}
	return left[0].Title
}

// CompareDue orders two milestones: by their due date, the earlier first,
// one with a date before one without; then by title, as versions
// (CompareMilestones). GitHub and GitLab both give a milestone a due date.
func CompareDue(a, b forge.Milestone) int {
	switch {
	case a.Due != b.Due && (a.Due == "" || b.Due == ""):
		if a.Due == "" {
			return 1
		}
		return -1
	case a.Due != b.Due:
		return strings.Compare(a.Due, b.Due)
	}
	return CompareMilestones(a.Title, b.Title)
}

var chunk = regexp.MustCompile(`\d+|\D+`)

// CompareMilestones orders two milestone titles as versions: their
// numbers compared as numbers (v1.9 before v1.10), the rest as text.
func CompareMilestones(a, b string) int {
	x, y := chunk.FindAllString(a, -1), chunk.FindAllString(b, -1)
	for i := 0; i < len(x) && i < len(y); i++ {
		n, errA := strconv.Atoi(x[i])
		m, errB := strconv.Atoi(y[i])
		switch {
		case errA == nil && errB == nil && n != m:
			return n - m
		case errA == nil && errB == nil:
		case x[i] != y[i]:
			return strings.Compare(x[i], y[i])
		}
	}
	return len(x) - len(y)
}

// Less is the backlog's order (docs/spec/backlog-acts.md, "Ordering"):
// the nearest milestone first — by its due date, then its title
// (CompareDue) —, an issue in none after every one in one; then the
// priority, an issue with none after the last (scale.Priority.Rank); then
// the effort, the lightest first, none as Medium; then the lowest number.
func Less(a, b forge.Issue) bool {
	switch {
	case a.Milestone != b.Milestone && (a.Milestone == "" || b.Milestone == ""):
		return b.Milestone == ""
	case a.Milestone != b.Milestone:
		if c := CompareDue(forge.Milestone{Title: a.Milestone, Due: a.MilestoneDue}, forge.Milestone{Title: b.Milestone, Due: b.MilestoneDue}); c != 0 {
			return c < 0
		}
	}
	pa, pb := scale.Priority.Rank(Priority(a)), scale.Priority.Rank(Priority(b))
	if pa != pb {
		return pa < pb
	}
	// Then the effort, the lightest first, none as Medium: never a score,
	// so a noisy effort never carries an issue across priorities
	// (ADR-0040, the value-vs-effort grid).
	if ea, eb := scale.Effort.Rank(Effort(a)), scale.Effort.Rank(Effort(b)); ea != eb {
		return ea < eb
	}
	return a.ID < b.ID
}
