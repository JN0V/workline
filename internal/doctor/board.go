package doctor

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
)

// boardDoc is where a person reads how workline finds a board.
const boardDoc = "docs/boards.md"

// boards says the forge's board workline found, its columns matched to
// the steps the product owner and the developer mark (ADR-0057), read
// only: workline moves no card yet. Only when one of them is routed.
func boards(r *Report, o Options, cfg *role.ProjectConfig, line *routing.Config) {
	if !routesRole(line, "product-owner") && !routesRole(line, "developer") {
		return
	}
	spec := o.Forge
	if spec == "" {
		spec = cfg.Forge
	}
	switch {
	case spec != "":
	case onGitHub(o.Repo):
		spec = "github"
	case onGitLab(o.Repo):
		spec = "gitlab"
	default:
		return
	}
	f, err := forge.Open(spec, o.Repo)
	if err != nil || f == nil {
		return // said by findingsOnMergeRequests
	}
	const labels = "workline's roles mark the work with their labels"
	found, ok, err := forge.Boards(f)
	switch {
	case !ok:
		r.add(Check{Area: "repository", Rule: "board-none", Level: Note,
			Message: "the forge " + spec + " holds no board: " + labels, Fix: boardDoc})
		return
	case err != nil:
		r.add(Check{Area: "repository", Rule: "board-unread", Level: Warn,
			Message: "could not read the forge's boards: " + forge.Said(err) + "; " + labels, Fix: boardDoc})
		return
	}
	set := cfg.Board
	chosen, candidates := board.Choose(found, set)
	switch {
	case chosen == nil && len(candidates) > 1:
		var named []string
		for _, b := range candidates {
			named = append(named, title(b))
		}
		fix := "name a column workline sets in `board:` (.workline/config.yaml), or keep one board with these columns (" + boardDoc + ")"
		if candidates[0].Forge == "github" {
			fix = fmt.Sprintf("`board: {project: %d}` in .workline/config.yaml (%s)", candidates[0].Number, boardDoc)
		}
		r.add(Check{Area: "repository", Rule: "board-ambiguous", Level: Note,
			Message: fmt.Sprintf("%d boards: %s; workline takes the one an issue sits in, else none", len(candidates), strings.Join(named, ", ")),
			Fix:     fix})
		return
	case chosen == nil && set.Project != 0:
		r.add(Check{Area: "repository", Rule: "board-setting-unmatched", Level: Warn,
			Message: fmt.Sprintf("`board: {project: %d}` names no open project linked to the repository with a Status field: %s", set.Project, labels),
			Fix:     boardDoc})
		return
	case chosen == nil:
		r.add(Check{Area: "repository", Rule: "board-none", Level: Note,
			Message: "no board with a column workline can set (a project's Status field, a board's label list): " + labels, Fix: boardDoc})
		return
	}
	m, unmatched := board.Match(chosen.Columns, set)
	if len(unmatched) > 0 {
		r.add(Check{Area: "repository", Rule: "board-setting-unmatched", Where: chosen.URL, Level: Warn,
			Message: "`board:` names columns " + title(*chosen) + " lacks (" + strings.Join(unmatched, ", ") + "): those steps move nothing",
			Fix:     "the columns are " + strings.Join(chosen.Columns, ", ") + " (" + boardDoc + ")"})
	}
	slot := func(name string) string {
		if name == "" {
			return "no column"
		}
		return name
	}
	done := slot(m.Done)
	if chosen.Forge == "gitlab" {
		done = "the Closed list, at the close"
		if !chosen.ClosedList {
			done = "the close (the Closed list is hidden)"
		}
	}
	msg := fmt.Sprintf("%s: ready → %s, doing → %s, done → %s", title(*chosen), slot(m.Ready), slot(m.Doing), done)
	if chosen.Forge == "github" {
		msg += "; " + projectWorkflows(*chosen)
	}
	r.add(Check{Area: "repository", Rule: "board-found", Where: chosen.URL, Level: OK,
		Message: msg + ". No card is moved yet: " + labels})
}

// title names a board as a person finds it.
func title(b forge.Board) string {
	if b.Forge == "github" {
		return fmt.Sprintf("GitHub project %q (#%d, %s's)", b.Name, b.Number, b.Owner)
	}
	return fmt.Sprintf("GitLab board %q (the %s's)", b.Name, b.Owner)
}

// projectWorkflows says a GitHub project's built-in workflows on and off.
func projectWorkflows(b forge.Board) string {
	if b.Workflows == nil {
		return "its workflows unread: " + b.WorkflowsUnread
	}
	var on, off []string
	for name, enabled := range b.Workflows {
		if enabled {
			on = append(on, name)
		} else {
			off = append(off, name)
		}
	}
	slices.Sort(on)
	slices.Sort(off)
	return "its workflows on: " + orNone(on) + "; off: " + orNone(off)
}

func orNone(l []string) string {
	if len(l) == 0 {
		return "none"
	}
	return strings.Join(l, ", ")
}
