package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/verdict"
)

// PriorityLabels lists the levels an issue's priority labels give, read
// by name where the project keeps them (scale.Used), in the order of its
// labels; more than one is a person's doing.
func PriorityLabels(is forge.Issue) []int { return scale.Used.Priority.Family.Levels(is.Labels) }

// Priority is an issue's level, the most pressing of its labels; 0 when
// it has none.
func Priority(is forge.Issue) int { return scale.Used.Priority.Family.First(is.Labels) }

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
	var s scale.Scheme
	var said []string
	var unknown []verdict.Finding
	for _, a := range []struct {
		scale scale.Scale
		set   scale.Setting
		to    *scale.Source
	}{{scale.Priority, cfg.Priority, &s.Priority}, {scale.Effort, cfg.Effort, &s.Effort}} {
		src, err := scale.Detect(a.scale, a.set, labels, on, forge.ScopedLabels(f))
		if err != nil {
			return scale.Before, nil, err
		}
		*a.to = src
		where := src.Said
		if where == "" {
			where = "read nowhere, its setting unset"
		}
		said = append(said, strings.ToUpper(a.scale.Axis[:1])+a.scale.Axis[1:]+": "+where+".")
		if src.Unknown != nil {
			unknown = append(unknown, verdict.Finding{Rule: "level-unknown", Level: "info",
				Message: src.Unknown.Error() + "; a name is mapped to a level under `names` in the settings"})
		}
	}
	return s, append([]verdict.Finding{{Rule: "values-where", Level: "info", Message: strings.Join(said, " ")}}, unknown...), nil
}

// Placeable says why a priority cannot be written on an issue where the
// project keeps it, src (ADR-0040): the issue holds one in another place
// — never written over, nor mirrored —, or no label of the family carries
// the level; "" when it can. The plan asks it, and the applying job again
// before it writes.
func Placeable(src scale.Source, is forge.Issue, l int) (rule, why string) {
	if there := src.Elsewhere(is.Labels); len(there) > 0 {
		var here []string
		for _, n := range is.Labels {
			if src.Family.Of(n) > 0 {
				here = append(here, n)
			}
		}
		if len(here) > 0 {
			return "priority-elsewhere", fmt.Sprintf("its priority is in two places, %s and %s: nothing is written on it; a person keeps one",
				strings.Join(here, ", "), strings.Join(there, ", "))
		}
		return "priority-elsewhere", fmt.Sprintf("its priority is in %s, not in %s where the project keeps it: nothing is written on it, never in two places",
			strings.Join(there, ", "), src.Family.Span())
	}
	if _, ok := src.Family.Label(l); !ok {
		return "priority-no-label", fmt.Sprintf("no label of %s carries %s: a person creates one, or maps one to it under `names` in the settings",
			src.Family.Span(), scale.Priority.Name(l))
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
// the lowest number.
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
	return a.ID < b.ID
}
