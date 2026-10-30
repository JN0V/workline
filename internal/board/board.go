// Package board matches a forge's board to the steps workline marks
// (ADR-0057): which column is ready, doing, done, found by their names,
// loosely, or named by the `board:` setting; and which board, among the
// ones the forge holds.
package board

import (
	"slices"
	"strings"
	"unicode"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
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
type Move struct {
	Board  string   `json:"board" yaml:"board"`
	Column string   `json:"column" yaml:"column"`
	Off    []string `json:"off,omitempty" yaml:"off,omitempty"`
}

// To is the move of an issue to slot (ready or doing) on f's board; nil
// with no board, no column for the slot, or a board out of reach, even in
// part — which err says, changing no verdict. On GitLab, a column is a
// label: the board's other columns' labels come off in the same call, as
// a Free board's lists do not (a scoped label's board, Premium, has
// already). On GitHub, nothing moves yet.
func To(f any, slot string, s role.Board) (*Move, error) {
	boards, ok, err := forge.Boards(f)
	if err != nil || !ok || len(boards) == 0 {
		return nil, err
	}
	chosen, _ := Choose(boards, s)
	if chosen == nil || chosen.Forge != "gitlab" {
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
	for _, c := range chosen.Columns {
		if c != mv.Column {
			mv.Off = append(mv.Off, c)
		}
	}
	return mv, nil
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
