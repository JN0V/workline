package tester

import (
	"encoding/xml"
	"os"
	"strings"
)

// result is how a test ended in a report.
type result int

const (
	passed result = iota
	skipped
	failed // a failure or an error: red
)

// suite is a JUnit XML <testsuite>, or <testsuites> holding them: each
// runner nests them its own way.
type suite struct {
	Name   string     `xml:"name,attr"`
	Time   string     `xml:"time,attr"`
	Suites []suite    `xml:"testsuite"`
	Cases  []testcase `xml:"testcase"`
}

type testcase struct {
	Class   string    `xml:"classname,attr"`
	Name    string    `xml:"name,attr"`
	Time    string    `xml:"time,attr"`
	Failure *struct{} `xml:"failure"`
	Error   *struct{} `xml:"error"`
	Skipped *struct{} `xml:"skipped"`
}

// readJUnit adds the tests of a JUnit XML report to results, by id —
// classname.name, or the name alone, as its file writes it: the `()`
// JUnit 5 on Gradle puts after a method's name dropped (#1178) — and, when suites is not nil, the
// name of the testsuite holding each one (a Go package, a pytest run), or
// its classname when the suite has none. A test run more than once (a
// parameter, a retry) is red when one of its runs is. classless, when not
// nil, holds the ids of the cases written with no classname; erred, when
// not nil, the name of each case red by an error alone, no failure: the
// JUnit mark of a test that did not end by its assertion, or of a
// runner's own record of a suite it could not build or run.
func readJUnit(file string, results map[string]result, suites map[string]string, classless map[string]bool, erred map[string]string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var root suite
	if err := xml.Unmarshal(data, &root); err != nil {
		return err
	}
	var walk func(s suite)
	walk = func(s suite) {
		for _, c := range s.Cases {
			name := strings.TrimSuffix(c.Name, "()")
			// A name ending in `()`, or written as words (a display
			// name, spaces in it), is a method's, whatever text it
			// carries: a runner's record of a suite is one token.
			method := name != c.Name || strings.ContainsAny(name, " \t")
			id := name
			if c.Class != "" {
				id = c.Class + "." + name
			} else if classless != nil {
				classless[id] = true
			}
			r := passed
			switch {
			case c.Failure != nil || c.Error != nil:
				r = failed
				// Erred only while no run of it failed: a run in
				// error after a failure stays a failure.
				if prev, seen := results[id]; erred != nil && c.Failure == nil && !method && (!seen || prev != failed || erred[id] != "") {
					erred[id] = name
				} else if erred != nil {
					delete(erred, id)
				}
			case c.Skipped != nil:
				r = skipped
			}
			if prev, ok := results[id]; !ok || r > prev {
				results[id] = r
			}
			if suites != nil {
				suites[id] = s.Name
				if s.Name == "" {
					suites[id] = c.Class
				}
			}
		}
		for _, sub := range s.Suites {
			walk(sub)
		}
	}
	walk(root)
	return nil
}
