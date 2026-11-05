package gate

import (
	"fmt"
	"sort"

	"github.com/JN0V/workline/internal/linters"
	"github.com/JN0V/workline/internal/verdict"
)

// lintedRule is the gate's "linted" rule (#244, ADR-0049 section 7): set in
// the gate's `enforce` (`linted: block`, or `warn`), it refuses a change
// that touches a language no linter reads. A language is exempted by
// `linted/<language>: exempt`; the exemption is said, and passes.
const lintedRule = "linted"

// linted tells, for a change, the languages it touches that no linter of the
// project reads. Nothing is said when the project did not set the rule.
func linted(repo string, g *Gate, ch *change) []verdict.Finding {
	if _, set := g.Enforce[lintedRule]; !set || ch == nil {
		return nil
	}
	rep, err := linters.Read(repo)
	if err != nil {
		return []verdict.Finding{{Rule: lintedRule, Level: "warn",
			Message: "the languages here could not be told, so the \"linted\" rule was not applied: " + err.Error()}}
	}
	var out []verdict.Finding
	for _, l := range rep.Languages {
		if len(l.Running[linters.Lint]) > 0 || len(l.Missing[linters.Lint]) == 0 {
			continue
		}
		var touched []string
		for _, f := range l.Files {
			if ch.files[f] {
				touched = append(touched, f)
			}
		}
		if len(touched) == 0 {
			continue
		}
		sort.Strings(touched)
		s := l.Missing[linters.Lint][0]
		where := lintedRule + "/" + l.Name
		if g.Enforce[where] == "exempt" {
			out = append(out, verdict.Finding{Rule: lintedRule, Where: where, Level: "warn",
				Message: fmt.Sprintf("%s (%s) is read by no linter; exempted by `enforce: %s: exempt`", l.Name, touched[0], where)})
			continue
		}
		fix := fmt.Sprintf("add a linter for %s to a gate's checks in .workline/config.yaml (%s), or exempt it with `enforce: %s: exempt`", l.Name, s.Tool, where)
		if s.Check != "" {
			fix = fmt.Sprintf("add to a gate's checks in .workline/config.yaml: - %s; or exempt %s with `enforce: %s: exempt`", s.Check, l.Name, where)
		}
		out = append(out, verdict.Finding{Rule: lintedRule, Where: where,
			Message: fmt.Sprintf("the change touches %s (%s), which no linter reads; %s would: %s", l.Name, touched[0], s.Tool, fix)})
	}
	return out
}
