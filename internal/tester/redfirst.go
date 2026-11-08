package tester

// Which compiler errors in a test of the change are red for the right
// reason (#1192, round 6's RC1): a test of code not written yet may fail
// to build by more than a missing name. A signature the issue changes
// (`Too many arguments for 'fun restoreFromExisting(…)'`), a method the
// change adds that a fake overrides (`'nextIndex' overrides nothing`),
// and the errors that follow from those (`Cannot infer type`) are red
// first too; a wrong call on what the base already has is not.
//
//   - A missing name (a missing-name row) is red first when the issue
//     names it, as before (#1111); with no issue, always.
//   - Any other error is red first when a name it quotes is one the base
//     lacks, or one the issue writes as a call (`name(`), giving its new
//     shape, or, beside code, one the change's code touches.
//   - In a file holding an error red first, the other errors are its
//     knock-ons: an error quoting nothing, a missing name the test
//     reaches as a member (`grid.vertical`), or a type it does not call
//     (`GridType.THIRDS`) of what the change adds. Not knock-ons: an
//     error quoting only names the base has, none said above (a wrong
//     call, back to the tester), nor a missing helper the test calls,
//     the issue never naming it (#1111).

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
)

// quotedPart is a stretch between quotes in a message, spaces in it or
// not: 'fun restoreFromExisting(existingCount: Int): Unit'.
var quotedPart = regexp.MustCompile("['‘\"`]([^'‘’\"`\n]+)['’\"`]")

// beforeParen is the name before a quoted signature's first parenthesis.
var beforeParen = regexp.MustCompile(`([\p{L}_][\p{L}\p{N}_]*)\s*\(`)

// word is a name-shaped word of a message.
var word = regexp.MustCompile(`[\p{L}_][\p{L}\p{N}_]*`)

// namesOf is the names an error's message gives: the quoted ones (a
// signature's own name, before its parenthesis; a lone word, its last
// part), never a double-quoted value; quoting none, its words the test
// file calls (`Add` of Go's `not enough arguments in call to Add`,
// called `Add(` in the test).
func namesOf(msg, test string) []string {
	var out []string
	for _, q := range quotedPart.FindAllStringSubmatch(msg, -1) {
		part := q[1]
		if strings.HasPrefix(q[0], `"`) {
			continue // a value, as Go quotes a string literal: not a name
		}
		if m := beforeParen.FindStringSubmatch(part); m != nil {
			out = append(out, m[1])
			continue
		}
		if !strings.ContainsAny(part, " \t") {
			if last := lastPart.FindString(part); last != "" {
				out = append(out, last)
			}
		}
	}
	if len(out) > 0 || test == "" {
		return out
	}
	for _, w := range word.FindAllString(msg, -1) {
		if calledIn(w, test) {
			out = append(out, w)
		}
	}
	return out
}

// member is a name written after a dot.
func member(name, text string) bool {
	last := lastPart.FindString(name)
	return last != "" && regexp.MustCompile(`\.\s*`+regexp.QuoteMeta(last)+`(?:$|[^\p{L}\p{N}_])`).MatchString(text)
}

// knockOn says whether a missing name the issue never names follows from
// another in its file: the test reaches it as a member, or it is a type
// (a capital first) the test does not call; not a helper it calls.
func knockOn(name, test string) bool {
	last := lastPart.FindString(name)
	if last == "" {
		return true
	}
	if member(last, test) {
		return true
	}
	return unicode.IsUpper([]rune(last)[0]) && !calledIn(last, test)
}

// calledIn says whether text writes name, or its last part, as a call:
// `name(`.
func calledIn(name, text string) bool {
	last := lastPart.FindString(name)
	if last == "" {
		return false
	}
	return regexp.MustCompile(`(?:^|[^\p{L}\p{N}_.])` + regexp.QuoteMeta(last) + `\s*\(`).MatchString(text)
}

// redFirst sorts the build errors of the run where the change's tests
// were laid on the base: those red for the right reason, those broken
// otherwise, and the missing names the issue never names.
type redFirst struct {
	repo, base string
	issue      string              // the issue's text, "" when not known
	code       func() string       // the change's code diff, "" when tests alone
	test       func(string) string // a test file of the change, as the head holds it
	inBase     map[string]bool
}

// lacks says whether the base holds no word name (its last part) in any
// file: a name the change adds. A search that fails says it holds it.
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
	cmd := exec.Command("git", "-C", r.repo, "grep", "-q", "-w", "-F", "-e", last, r.base, "--")
	err := cmd.Run()
	var exit *exec.ExitError
	has := !errors.As(err, &exit) || exit.ExitCode() != 1
	r.inBase[last] = has
	return !has
}

// said says whether the issue writes name as a call, or the change's code
// touches it.
func (r *redFirst) said(name string) bool {
	return (r.issue != "" && calledIn(name, r.issue)) || namedIn(name, r.code())
}

// sort is errs, each in a test of the change red first or its knock-on
// left out of broken and unnamed.
func (r *redFirst) sort(errs []buildError) (broken, unnamed []buildError) {
	anchored := map[string]bool{}
	red := make([]bool, len(errs))
	names := make([][]string, len(errs))
	for i, e := range errs {
		switch {
		case !e.inTest:
		case e.missing:
			red[i] = r.issue == "" || e.name == "" || namedIn(e.name, r.issue)
		default:
			names[i] = namesOf(e.msg, r.test(e.file))
			for _, n := range names[i] {
				if r.said(n) || r.lacks(n) {
					red[i] = true
					break
				}
			}
		}
		if red[i] {
			anchored[e.file] = true
		}
	}
	for i, e := range errs {
		switch {
		case red[i]:
		case !e.inTest:
			broken = append(broken, e)
		case anchored[e.file] && !e.missing && len(names[i]) == 0 && !quotedPart.MatchString(e.msg):
			// a knock-on: a type the compiler could not infer past a
			// name the base lacks; one quoting a value (Go's `cannot use
			// "Kitchen" … as int value`) is the test's own
		case anchored[e.file] && e.missing && knockOn(e.name, r.test(e.file)):
		case e.missing:
			unnamed = append(unnamed, e)
		default:
			broken = append(broken, e) // a wrong call on what the base has
		}
	}
	return broken, unnamed
}
