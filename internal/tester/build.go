package tester

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// BuildErrors is how one ecosystem's compiler says, in the test
// command's output, that a build broke, and when that is only a name the
// code lacks: a test of code not written yet, red for the right reason
// (ADR-0055, the proof).
type BuildErrors struct {
	// Error matches one error line; its groups `file` (from the
	// repository's root) and `message`.
	Error string `json:"error"`
	// MissingName matches a message naming what the code lacks; the
	// name itself in its group `name`, else the first quoted word of the
	// message (#1111).
	MissingName []string `json:"missing-name"`
	// TestName finds the tests a test file declares, its first group the
	// name: their ids while the file does not build.
	TestName string `json:"test-name"`
}

type buildRules struct {
	error    *regexp.Regexp
	missing  []*regexp.Regexp
	testName *regexp.Regexp
}

// compileBuildErrors reads the build-errors setting; a pattern that does
// not compile, or lacks its groups, is said.
func compileBuildErrors(list []BuildErrors) ([]buildRules, error) {
	var out []buildRules
	for i, b := range list {
		var r buildRules
		var err error
		if r.error, err = regexp.Compile(b.Error); err != nil {
			return nil, fmt.Errorf("build-errors[%d].error: %v", i, err)
		}
		if r.error.SubexpIndex("file") < 0 || r.error.SubexpIndex("message") < 0 {
			return nil, fmt.Errorf("build-errors[%d].error: the groups (?P<file>…) and (?P<message>…) are needed", i)
		}
		for j, m := range b.MissingName {
			re, err := regexp.Compile(m)
			if err != nil {
				return nil, fmt.Errorf("build-errors[%d].missing-name[%d]: %v", i, j, err)
			}
			r.missing = append(r.missing, re)
		}
		if r.testName, err = regexp.Compile(b.TestName); err != nil || r.testName.NumSubexp() < 1 {
			if err == nil {
				err = fmt.Errorf("a group holding the name is needed")
			}
			return nil, fmt.Errorf("build-errors[%d].test-name: %v", i, err)
		}
		out = append(out, r)
	}
	return out, nil
}

// buildError is one error line of a broken build.
type buildError struct {
	file, line string
	missing    bool   // in a test file of the change, naming what the code lacks
	name       string // the name it lacks, when the message gives it
	rules      int    // the ecosystem it was read by
}

// buildErrorsIn reads the error lines of output by each ecosystem's
// pattern. Each is a missing name only when it lies in one of tests (the
// change's test files) and its message names what the code lacks.
func buildErrorsIn(output string, tests []string, rules []buildRules) []buildError {
	inTests := map[string]bool{}
	for _, t := range tests {
		inTests[t] = true
	}
	var errs []buildError
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		for i, r := range rules {
			m := r.error.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			file := path.Clean(strings.TrimPrefix(m[r.error.SubexpIndex("file")], "./"))
			msg := m[r.error.SubexpIndex("message")]
			e := buildError{file: file, line: line, rules: i}
			if inTests[file] {
				for _, re := range r.missing {
					if m := re.FindStringSubmatch(msg); m != nil {
						e.missing, e.name = true, nameIn(re, m, msg)
						break
					}
				}
			}
			errs = append(errs, e)
			break
		}
	}
	return errs
}

// testNames is the tests content declares, by the rules' test-name.
func testNames(content string, r buildRules) []string {
	var names []string
	for _, m := range r.testName.FindAllStringSubmatch(content, -1) {
		names = append(names, m[1])
	}
	return names
}

// quotedName is a word between quotes, as compilers quote the name they
// lack: 'x', ‘x’, "x", `x`.
var quotedName = regexp.MustCompile("['‘\"`]([^\\s'‘’\"`]+)['’\"`]")

// nameIn is the name a missing-name message gives: its group `name`,
// else its first quoted word; "" when it gives none.
func nameIn(re *regexp.Regexp, m []string, msg string) string {
	if i := re.SubexpIndex("name"); i >= 0 && m[i] != "" {
		return m[i]
	}
	if q := quotedName.FindStringSubmatch(msg); q != nil {
		return q[1]
	}
	return ""
}

// lastPart is a name's last word, past its qualifiers: `lostCount` of
// `MQTTStatistics::lostCount`, `Mul` of `calc.Mul`.
var lastPart = regexp.MustCompile(`[\p{L}\p{N}_]+$`)

// namedIn says whether text names name as a whole word, or its last
// part: a missing name the issue names is what the change adds; one it
// never names, a guess of the test's (#1111).
func namedIn(name, text string) bool {
	last := lastPart.FindString(name)
	if last == "" {
		return false
	}
	return regexp.MustCompile(`(?:^|[^\p{L}\p{N}_])` + regexp.QuoteMeta(last) + `(?:$|[^\p{L}\p{N}_])`).MatchString(text)
}
