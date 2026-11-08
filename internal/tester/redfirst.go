package tester

// Which compiler errors in a test of the change are red for the right
// reason, beyond a missing name: a signature the issue gives anew, a name
// it adds, and the errors that follow from those in their file (#1192).
// The rule, its reasons and its limits: roles/tester/docs/missing-name.md.

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// quotedPart is a stretch between quotes in a message, spaces in it or
// not: 'fun restoreFromExisting(existingCount: Int): Unit'. The opening
// quote follows no letter: the apostrophe of `Can't` opens nothing.
var quotedPart = regexp.MustCompile("(?:^|[^\\p{L}\\p{N}])(['‘\"`])([^'‘’\"`\n]+)['’\"`]")

// beforeParen is the name before a quoted signature's first parenthesis.
var beforeParen = regexp.MustCompile(`([\p{L}_][\p{L}\p{N}_]*)\s*\(`)

// word is a name-shaped word of a message.
var word = regexp.MustCompile(`[\p{L}_][\p{L}\p{N}_]*`)

// inference is a message saying a type could not be inferred: it follows
// from a name the compiler could not resolve before it.
var inference = regexp.MustCompile(`(?i)\b(?:cannot|can't|could not|not enough information to) infer\b`)

// namesOf is the names an error's message gives: the quoted ones (a
// signature's own name, before its parenthesis; a lone word, its last
// part), never a double-quoted value; quoting none, its words the test
// file calls (`Add` of Go's `not enough arguments in call to Add`,
// called `Add(` in the test).
func namesOf(msg, test string) []string {
	var out []string
	for _, q := range quotedPart.FindAllStringSubmatch(msg, -1) {
		if q[1] == `"` {
			continue // a value, as Go quotes a string literal: not a name
		}
		part := q[2]
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

// patterns keeps each name's patterns compiled once.
var patterns sync.Map

// pattern is the regular expression a format makes of a name, compiled
// once: %s is the name, quoted.
func pattern(format, name string) *regexp.Regexp {
	key := format + "\x00" + name
	if re, ok := patterns.Load(key); ok {
		return re.(*regexp.Regexp)
	}
	re := regexp.MustCompile(strings.ReplaceAll(format, "%s", regexp.QuoteMeta(name)))
	patterns.Store(key, re)
	return re
}

// member is a name written after a dot.
func member(name, text string) bool {
	last := lastPart.FindString(name)
	return last != "" && pattern(`\.\s*%s(?:$|[^\p{L}\p{N}_])`, last).MatchString(text)
}

// calledIn says whether text writes name, or its last part, as a call:
// `name(`.
func calledIn(name, text string) bool {
	last := lastPart.FindString(name)
	return last != "" && pattern(`(?:^|[^\p{L}\p{N}_.])%s\s*\(`, last).MatchString(text)
}

// declaredIn says whether text declares name: a class, an object, an
// interface, a type, a function or a value, never an override of one.
func declaredIn(name, text string) bool {
	last := lastPart.FindString(name)
	return last != "" && pattern(`(?m)(?:^|[^\p{L}\p{N}_])(?:class|object|interface|type|struct|enum|fun|func|def|val|var|let|const)\s+%s(?:$|[^\p{L}\p{N}_])`, last).
		MatchString(overrides.ReplaceAllString(text, "override"))
}

// overrides is an override's keyword and what it overrides: not a
// declaration of the test's own.
var overrides = regexp.MustCompile(`\boverride\s+(?:fun|val|var)\b`)

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

// redFirst sorts the build errors of the run where the change's tests
// were laid on the base: those red for the right reason, those broken
// otherwise, and the missing names the issue never names.
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

// declared says whether a test file of the change declares name.
func (r *redFirst) declared(name string) bool {
	for _, f := range r.tests {
		if declaredIn(name, r.test(f)) {
			return true
		}
	}
	return false
}

// adds says whether the change adds or reshapes name: the issue writes it
// as a call; the change's code touches it; or the base lacks it and the
// issue names it, or, no issue known, no test of the change declares it.
func (r *redFirst) adds(name string) bool {
	if (r.issue != "" && calledIn(name, r.issue)) || namedIn(name, r.code()) {
		return true
	}
	if !r.lacks(name) {
		return false
	}
	if r.issue != "" {
		return namedIn(name, r.issue)
	}
	return !r.declared(name)
}

// sort is errs, each in a test of the change red first or its knock-on
// left out of broken and unnamed; byShape says an error other than a
// missing name made a test red.
func (r *redFirst) sort(errs []buildError) (broken, unnamed []buildError, byShape bool) {
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
				if r.adds(n) {
					red[i], byShape = true, true
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
		case anchored[e.file] && !e.missing && len(names[i]) == 0 && !quotedPart.MatchString(e.msg) && inference.MatchString(e.msg):
			// a knock-on: a type not inferred past a name the base lacks
		case anchored[e.file] && e.missing && knockOn(e.name, r.test(e.file)):
		case e.missing:
			unnamed = append(unnamed, e)
		default:
			broken = append(broken, e) // a wrong call, a parse error: the test's own
		}
	}
	return broken, unnamed, byShape
}
