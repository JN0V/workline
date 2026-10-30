// Package board matches a forge's board to the steps workline marks
// (ADR-0057): which column is ready, doing, done, found by their names,
// loosely, or named by the `board:` setting; and which board, among the
// ones the forge holds.
package board

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
)

// The names each slot takes, by Key, first match wins.
var names = map[string][]string{
	"ready": {"ready", "todo", "backlog"},
	"doing": {"inprogress", "doing", "wip", "indevelopment", "started"},
	"done":  {"done", "closed", "complete", "completed"},
}

// Slots are the columns matched, by slot; "" for a slot with no column.
type Slots struct{ Ready, Doing, Done string }

// Key is a column's name as matched: its scope's key dropped
// (`workflow::In progress` → `In progress`), lowercase, letters only.
func Key(name string) string {
	if i := strings.LastIndex(name, "::"); i >= 0 {
		name = name[i+2:]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Match finds each slot's column among columns; a slot the setting names
// takes that column, by Key too. Unmatched are the setting's names no
// column holds: that slot stays empty.
func Match(columns []string, s role.Board) (m Slots, unmatched []string) {
	find := func(slot, set string) string {
		if set != "" {
			if i := slices.IndexFunc(columns, func(c string) bool { return Key(c) == Key(set) }); i >= 0 {
				return columns[i]
			}
			unmatched = append(unmatched, slot+": "+set)
			return ""
		}
		for _, n := range names[slot] {
			if i := slices.IndexFunc(columns, func(c string) bool { return Key(c) == n }); i >= 0 {
				return columns[i]
			}
		}
		return ""
	}
	m.Ready = find("ready", s.Ready)
	m.Doing = find("doing", s.Doing)
	m.Done = find("done", "")
	return m, unmatched
}

// Move is a card moved on the forge's board (ADR-0057): on GitLab, its
// column's label put on the issue, Off the labels taken off in the same
// call. Kept in the role's record.
//
// On GitHub, Project is the project's number, whose Status takes Column.
// Left says why the role writes it not: "workflow", the project's
// *Pull request linked to issue* moves it; "user", a user's project, on
// which GitHub gives an App no write. Said, a finding of the run's, its
// Rule: the workflow that moves no card to Done, off
// (`board-workflow-off`, the rule when none is kept); a user's project,
// what does not move and the workflows to turn on (`board-user-project`).
type Move struct {
	Board   string   `json:"board" yaml:"board"`
	Column  string   `json:"column" yaml:"column"`
	Off     []string `json:"off,omitempty" yaml:"off,omitempty"`
	Project int      `json:"project,omitempty" yaml:"project,omitempty"`
	Left    string   `json:"left,omitempty" yaml:"left,omitempty"`
	Said    string   `json:"said,omitempty" yaml:"said,omitempty"`
	Rule    string   `json:"rule,omitempty" yaml:"rule,omitempty"`
	URL     string   `json:"url,omitempty" yaml:"url,omitempty"`
}

// The GitHub project's built-in workflows workline relies on (ADR-0057).
const (
	LinkedWorkflow = "Pull request linked to issue"
	ClosedWorkflow = "Item closed"
)

// Findings are what the role that moves the card says once a run: its
// Said, an info.
func (m *Move) Findings() []verdict.Finding {
	if m == nil || m.Said == "" {
		return nil
	}
	rule := m.Rule
	if rule == "" {
		rule = "board-workflow-off"
	}
	return []verdict.Finding{{Rule: rule, Level: "info", Where: m.URL, Message: m.Said}}
}

// Labels: the move puts a label on, GitLab's column.
func (m *Move) Labels() bool { return m != nil && m.Project == 0 }

// Writes: the move sets a GitHub project's Status, the role's to write.
func (m *Move) Writes() bool { return m != nil && m.Project != 0 && m.Left == "" }

// Told is the move as the role's record says it to a person.
func (m *Move) Told() string {
	switch m.Left {
	case "workflow":
		return fmt.Sprintf("Its card goes to `%s` by the workflow *%s* of the board %q, when its pull request opens.", m.Column, LinkedWorkflow, m.Board)
	case "user":
		return fmt.Sprintf("Its card is not moved to `%s` on the board %q: a user's project, which GitHub lets no App write.", m.Column, m.Board)
	}
	return fmt.Sprintf("Its card moved to `%s` on the board %q.", m.Column, m.Board)
}

// To is the move of an issue to slot (ready or doing) on f's board; nil
// with no board, no column for the slot, or a board out of reach, even in
// part — which err says, changing no verdict. On GitLab, a column is a
// label: the board's other columns' labels come off in the same call, as
// a Free board's lists do not (a scoped label's board, Premium, has
// already). On GitHub, a column is an option of the project's Status:
// left to its workflow at the claim when that is on, never written on a
// user's project.
func To(f any, slot string, s role.Board) (*Move, error) {
	boards, ok, err := forge.Boards(f)
	if err != nil || !ok || len(boards) == 0 {
		return nil, err
	}
	chosen, _ := Choose(boards, s)
	if chosen == nil {
		return nil, nil
	}
	m, _ := Match(chosen.Columns, s)
	mv := &Move{Board: chosen.Name}
	switch slot {
	case "ready":
		mv.Column = m.Ready
	case "doing":
		mv.Column = m.Doing
	}
	if mv.Column == "" {
		return nil, nil
	}
	if chosen.Forge == "github" {
		mv.Project, mv.URL = chosen.Number, chosen.URL
		switch {
		case slot == "doing" && chosen.Workflows[LinkedWorkflow]:
			mv.Left = "workflow"
		case chosen.Owner == "user":
			mv.Left = "user"
		}
		off := func(name string) bool {
			on, read := chosen.Workflows[name]
			return chosen.WorkflowsUnread == "" && (!read || !on)
		}
		if chosen.Owner == "user" {
			mv.Rule, mv.Said = "board-user-project", userSaid(*chosen, m, off)
		} else if off(ClosedWorkflow) {
			mv.Said = fmt.Sprintf("turn on the workflow *%s* of the GitHub project %q (#%d): without it, no card reaches `%s` at the close",
				ClosedWorkflow, chosen.Name, chosen.Number, doneName(m.Done))
		}
		return mv, nil
	}
	for _, c := range chosen.Columns {
		if c != mv.Column {
			mv.Off = append(mv.Off, c)
		}
	}
	return mv, nil
}

// userSaid is what a user's GitHub project moves (ADR-0057): GitHub
// gives an App no right to write it, so a ready moves no card; its
// built-in workflows move the rest, those off named to turn on.
func userSaid(b forge.Board, m Slots, off func(string) bool) string {
	said := fmt.Sprintf("the GitHub project %q (#%d) is a user's, which GitHub lets no App write: a ready moves no card", b.Name, b.Number)
	if b.WorkflowsUnread != "" {
		// Never read as on: what moves the other cards is not known.
		return said + fmt.Sprintf("; its workflows not read (%s): *%s* and *%s* move the cards to the other columns when on",
			b.WorkflowsUnread, LinkedWorkflow, ClosedWorkflow)
	}
	var flows, cols []string
	if m.Doing != "" && off(LinkedWorkflow) {
		flows, cols = append(flows, "*"+LinkedWorkflow+"*"), append(cols, "`"+m.Doing+"`")
	}
	if off(ClosedWorkflow) {
		flows, cols = append(flows, "*"+ClosedWorkflow+"*"), append(cols, "`"+doneName(m.Done)+"`")
	}
	switch len(flows) {
	case 0:
		return said + "; its workflows move the cards to the other columns"
	case 1:
		return said + fmt.Sprintf("; turn on its workflow %s to move cards to %s", flows[0], cols[0])
	}
	return said + fmt.Sprintf("; turn on its workflows %s: they move the cards to %s", strings.Join(flows, ", "), strings.Join(cols, " and "))
}

func doneName(column string) string {
	if column == "" {
		return "Done"
	}
	return column
}

// Choose takes the board workline marks, among boards: on GitHub, the
// project the setting numbers; else the only one with a column —
// GitLab's boards listing the same labels count as one, a label being
// the same move on each. Several left: none chosen, candidates names
// them.
func Choose(boards []forge.Board, s role.Board) (chosen *forge.Board, candidates []forge.Board) {
	seen := map[string]bool{}
	for _, b := range boards {
		if len(b.Columns) == 0 || (s.Project != 0 && b.Forge == "github" && b.Number != s.Project) {
			continue
		}
		if b.Forge == "gitlab" {
			labels := slices.Clone(b.Columns)
			slices.Sort(labels)
			key := strings.Join(labels, "\x00")
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		candidates = append(candidates, b)
	}
	if len(candidates) == 1 {
		return &candidates[0], nil
	}
	return nil, candidates
}
