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
)

// Levels are the priorities an issue takes: 1, the most pressing, to 4
// (docs/spec/backlog-acts.md, "Ordering").
const Levels = 4

// PriorityPrefix starts each priority label: workline:priority/1 to /4.
const PriorityPrefix = "workline:priority/"

// PriorityLabel is the label of a priority level.
func PriorityLabel(n int) string { return PriorityPrefix + strconv.Itoa(n) }

// PriorityLabels lists the levels an issue's priority labels give, in the
// order of its labels; more than one is a person's doing.
func PriorityLabels(is forge.Issue) []int {
	var out []int
	for _, l := range is.Labels {
		if rest, ok := strings.CutPrefix(l, PriorityPrefix); ok {
			if n, err := strconv.Atoi(rest); err == nil && n >= 1 && n <= Levels {
				out = append(out, n)
			}
		}
	}
	return out
}

// Priority is an issue's level, the most pressing of its labels; 0 when
// it has none.
func Priority(is forge.Issue) int {
	set := PriorityLabels(is)
	if len(set) == 0 {
		return 0
	}
	return slices.Min(set)
}

// Before says an issue's place in the order, to put it back: its
// priority and its milestone.
func Before(is forge.Issue) string {
	p, m := "no priority", "no milestone"
	if n := Priority(is); n > 0 {
		p = fmt.Sprintf("priority %d", n)
	}
	if is.Milestone != "" {
		m = "milestone " + is.Milestone
	}
	return p + ", " + m
}

// Released says whether a tag named after the milestone exists: the
// release it was for is out.
func Released(repo, milestone string) bool {
	if milestone == "" {
		return false
	}
	return exec.Command("git", "-C", repo, "rev-parse", "-q", "--verify", "refs/tags/"+milestone).Run() == nil
}

// NextMilestone is the nearest open milestone not released, in version
// order; "" when there is none.
func NextMilestone(repo string, open []string) string {
	var left []string
	for _, m := range open {
		if !Released(repo, m) {
			left = append(left, m)
		}
	}
	sort.SliceStable(left, func(i, j int) bool { return CompareMilestones(left[i], left[j]) < 0 })
	if len(left) == 0 {
		return ""
	}
	return left[0]
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
// the nearest milestone first, an issue in none after every one in one;
// then the priority, an issue with none after priority 4; then the
// lowest number.
func Less(a, b forge.Issue) bool {
	switch {
	case a.Milestone != b.Milestone && (a.Milestone == "" || b.Milestone == ""):
		return b.Milestone == ""
	case a.Milestone != b.Milestone:
		if c := CompareMilestones(a.Milestone, b.Milestone); c != 0 {
			return c < 0
		}
	}
	pa, pb := Priority(a), Priority(b)
	if pa == 0 {
		pa = Levels + 1
	}
	if pb == 0 {
		pb = Levels + 1
	}
	if pa != pb {
		return pa < pb
	}
	return a.ID < b.ID
}

// Order sorts issues in the backlog's order.
func Order(issues []forge.Issue) {
	sort.SliceStable(issues, func(i, j int) bool { return Less(issues[i], issues[j]) })
}
