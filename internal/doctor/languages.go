package doctor

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/linters"
)

// languages says, for each language the repository holds, which tools
// read it, by kind, and which it lacks, with the gate check to add
// (ADR-0049, step 1). A warning, never an error: blocking on it is the
// project's choice, a rule still to come (#244).
func languages(r *Report, o Options) {
	rep, err := linters.Read(o.Repo)
	if err != nil {
		r.add(Check{Area: "repository", Rule: "languages-unread", Level: Warn,
			Message: "the languages here could not be told, so what reads them is not said: " + err.Error()})
		return
	}
	var checks []Check
	for i, l := range rep.Languages {
		if len(l.Running) == 0 && len(l.Missing) == 0 && within(l, rep.Languages, i) {
			continue // go.mod and go.sum, said under lockfile already
		}
		checks = append(checks, languageCheck(l))
	}
	msg := fmt.Sprintf("%d tracked files, %d languages a known tool reads", rep.Tracked, len(checks))
	if rep.Skipped > 0 {
		msg += fmt.Sprintf("; %d vendored or generated files left out", rep.Skipped)
	}
	r.add(Check{Area: "repository", Rule: "languages", Level: OK, Message: msg})
	for _, c := range checks {
		r.add(c)
	}
}

// within says whether another language holds all the files of one.
func within(l linters.Language, all []linters.Language, self int) bool {
	for i, o := range all {
		if i == self || len(o.Files) < len(l.Files) {
			continue
		}
		held := map[string]bool{}
		for _, f := range o.Files {
			held[f] = true
		}
		every := true
		for _, f := range l.Files {
			every = every && held[f]
		}
		if every {
			return true
		}
	}
	return false
}

func languageCheck(l linters.Language) Check {
	c := Check{Area: "repository", Where: l.Name}
	var parts []string
	// The first tool that runs, for each kind; the first three suggested.
	for _, g := range byTools(l.Running, 1, func(s linters.Seen) string { return s.Tool + " (" + s.Where + ")" }) {
		parts = append(parts, g.kinds+" by "+g.tools)
	}
	for _, g := range byTools(l.Missing, 3, said) {
		parts = append(parts, g.kinds+" by nothing: "+g.tools)
	}
	switch {
	case len(l.Missing) > 0:
		c.Rule, c.Level = "language-gaps", Warn
		c.Fix = fix(l)
	case len(l.Running) > 0:
		c.Rule, c.Level = "language-covered", OK
	default:
		c.Rule, c.Level = "language-unchecked", Note
		if len(l.Unsuggested) == 0 {
			parts = append(parts, "nothing reads it, and no tool is suggested for it")
		} else {
			var names []string
			for i, s := range l.Unsuggested {
				if i == 3 {
					names = append(names, fmt.Sprintf("and %d more", len(l.Unsuggested)-3))
					break
				}
				names = append(names, s.Tool+" ("+s.Config+")")
			}
			parts = append(parts, "nothing reads it; suggested only with their config: "+strings.Join(names, ", "))
		}
	}
	c.Message = filesOf(l) + ": " + strings.Join(parts, "; ")
	return c
}

// group is kinds read by the same tools: "lint and types by go vet".
type group struct{ kinds, tools string }

func byTools[T any](m map[string][]T, most int, name func(T) string) []group {
	var out []group
	index := map[string]int{}
	for _, kind := range linters.Kinds {
		l, ok := m[kind]
		if !ok {
			continue
		}
		var names []string
		for _, x := range l[:min(most, len(l))] {
			names = append(names, name(x))
		}
		tools := strings.Join(names, ", ")
		if i, ok := index[tools]; ok {
			out[i].kinds += " and " + kind
			continue
		}
		index[tools] = len(out)
		out = append(out, group{kind, tools})
	}
	return out
}

func said(s linters.Suggestion) string {
	out := s.Tool
	if s.Version != "" {
		out += " " + s.Version
	}
	if s.Config != "" {
		out += " (configured in " + s.Config + ")"
	}
	return out
}

// filesOf is the language and its files: their names when three or fewer.
func filesOf(l linters.Language) string {
	n := len(l.Files)
	switch {
	case n <= 3:
		return l.Name + " (" + strings.Join(l.Files, ", ") + ")"
	case l.Shebang == 1:
		return fmt.Sprintf("%s (%d files, 1 told by its #! line)", l.Name, n)
	case l.Shebang > 1:
		return fmt.Sprintf("%s (%d files, %d told by their #! line)", l.Name, n, l.Shebang)
	}
	return fmt.Sprintf("%s (%d files)", l.Name, n)
}

// fix is the gate checks to add, the first tool suggested for each kind
// read by nothing.
func fix(l linters.Language) string {
	lines := []string{"add to a gate's checks in .workline/config.yaml (docs/languages.md):"}
	done := map[string]bool{}
	for _, kind := range linters.Kinds {
		s, ok := l.Missing[kind]
		if !ok || done[s[0].Tool] {
			continue
		}
		done[s[0].Tool] = true
		if s[0].Check == "" {
			lines = append(lines, "# "+s[0].Tool+": its command is not known here, see its documentation")
			continue
		}
		lines = append(lines, "- "+s[0].Check)
		if s[0].Each != "" {
			lines = append(lines, "# from each folder holding a "+s[0].Each+": none is at the root")
		}
	}
	if l.Shebang > 0 && strings.Contains(strings.Join(lines, "\n"), "git ls-files") {
		lines = append(lines, fmt.Sprintf("# and the %d told by their #! line, by name: git ls-files names files by extension", l.Shebang))
	}
	return strings.Join(lines, "\n      ")
}
