package tester

// Whether a build the change's tests broke on the base is red for the
// right reason (ADR-0065, change 1). Kept mechanical: the errors the
// build-errors rows read, the names the base lacks (`git grep -w`). A
// fast path that may only pass: each error a missing name the base lacks
// and the issue names verbatim. Anything else is not guessed here: the
// judge reads the raw facts (judged.go). The rule and its limits:
// roles/tester/docs/missing-name.md.

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// redFirst reads the build errors of the run where the change's tests
// were laid on the base.
type redFirst struct {
	repo, base string
	issue      string              // the issue's text, "" when not known
	code       func() string       // the lines the change's code adds or removes, "" when tests alone
	test       func(string) string // a test file of the change, as the head holds it
	tests      []string            // the change's test files
	inBase     map[string]bool
}

// lacks says whether the base holds no word name (its last part) in any
// file. A search that fails says it holds it.
func (r *redFirst) lacks(name string) bool {
	last := lastPart.FindString(name)
	if last == "" {
		return false
	}
	if r.inBase == nil {
		r.inBase = map[string]bool{}
	}
	if has, ok := r.inBase[last]; ok {
		return !has
	}
	err := exec.Command("git", "-C", r.repo, "grep", "-q", "-w", "-F", "-e", last, r.base, "--").Run()
	var exit *exec.ExitError
	has := !errors.As(err, &exit) || exit.ExitCode() != 1
	r.inBase[last] = has
	return !has
}

// named says whether the issue names name verbatim, as a whole word; no
// issue known (`workline test check`), whether the change's code does.
func (r *redFirst) named(name string) bool {
	if r.issue != "" {
		return namedIn(name, r.issue)
	}
	return namedIn(name, r.code())
}

// fast says each error is red by the fast path: in a test of the change,
// a missing name the build-errors rows read, the base lacks it, and the
// issue names it. Sure of a yes, or false: never a no.
func (r *redFirst) fast(errs []buildError) bool {
	if len(errs) == 0 {
		return false
	}
	for _, e := range errs {
		if !e.inTest || !e.missing || e.name == "" || !r.lacks(e.name) || !r.named(e.name) {
			return false
		}
	}
	return true
}

// facts are what the judge reads of a broken build, raw: the issue, the
// change's test files, the compiler's lines and the output's end, clipped
// and masked, and whether the base holds each name the errors quote.
func (r *redFirst) facts(errs []buildError, output string, list []BuildErrors) string {
	var b strings.Builder
	issue := r.issue
	if issue == "" {
		issue = "(not known: the tests are proven on their own)"
	}
	fmt.Fprintf(&b, "## The issue\n\n%s\n", clipRunes(forge.Mask(issue), factsIssueMax))
	b.WriteString("\n## The change's test files\n")
	left := factsTestsMax
	for _, f := range r.tests {
		c := clipRunes(forge.Mask(r.test(f)), max(left, 200))
		left -= len([]rune(c))
		fmt.Fprintf(&b, "\n### `%s`\n\n````\n%s\n````\n", f, strings.TrimRight(c, "\n"))
	}
	b.WriteString("\n## The build's errors, on the base with these tests\n\n```\n")
	var lines []string
	for _, e := range errs {
		lines = append(lines, clipRunes(e.line, quoteLine))
	}
	if len(lines) > factsLinesMax {
		lines = append(lines[:factsLinesMax], fmt.Sprintf("(and %d more)", len(lines)-factsLinesMax))
	}
	fmt.Fprintf(&b, "%s\n```\n", forge.Mask(strings.Join(lines, "\n")))
	fmt.Fprintf(&b, "\nThe output, as the project's lines read it: %s\n", forge.Mask(errorLines(output, list, 15)))
	var names []string
	for _, e := range errs {
		for _, n := range quotedNames(e) {
			if !slices.Contains(names, n) {
				names = append(names, n)
			}
		}
	}
	if len(names) > 0 {
		b.WriteString("\n## The names the errors give, in the base\n\n")
		for _, n := range names {
			held := "held somewhere in the base's files"
			if r.lacks(n) {
				held = "in none of the base's files"
			}
			fmt.Fprintf(&b, "- `%s`: %s\n", n, held)
		}
	}
	return b.String()
}

// quotedNames is the names an error gives: the one its row reads, and
// each word it quotes. Context for the judge, never a verdict.
func quotedNames(e buildError) []string {
	var out []string
	if e.name != "" {
		out = append(out, e.name)
	}
	for _, q := range quotedName.FindAllStringSubmatch(e.msg, -1) {
		if !slices.Contains(out, q[1]) && lastPart.MatchString(q[1]) {
			out = append(out, q[1])
		}
	}
	return out
}

// The judge's material, bounded: the issue's runes, the test files' all
// together, the compiler's lines.
const (
	factsIssueMax = 6000
	factsTestsMax = 12000
	factsLinesMax = 40
)
