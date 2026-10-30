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

// Choose takes the board workline marks, among boards: the project the
// setting numbers; else the only one with a column — GitLab's boards
// matching the same labels count as one, a label being the same move on
// each. Several left: none chosen, candidates names them.
func Choose(boards []forge.Board, s role.Board) (chosen *forge.Board, candidates []forge.Board) {
	seen := map[Slots]bool{}
	for _, b := range boards {
		if len(b.Columns) == 0 || (s.Project != 0 && (b.Forge != "github" || b.Number != s.Project)) {
			continue
		}
		if b.Forge == "gitlab" {
			m, _ := Match(b.Columns, s)
			if seen[m] {
				continue
			}
			seen[m] = true
		}
		candidates = append(candidates, b)
	}
	if len(candidates) == 1 {
		return &candidates[0], nil
	}
	return nil, candidates
}
