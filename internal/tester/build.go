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
	// MissingName matches a message naming what the code lacks.
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
	missing    bool // in a test file of the change, naming what the code lacks
	rules      int  // the ecosystem it was read by
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
					if re.MatchString(msg) {
						e.missing = true
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
