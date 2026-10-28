package tester

import (
	"encoding/xml"
	"os"
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
	Suites []suite    `xml:"testsuite"`
	Cases  []testcase `xml:"testcase"`
}

type testcase struct {
	Class   string    `xml:"classname,attr"`
	Name    string    `xml:"name,attr"`
	Failure *struct{} `xml:"failure"`
	Error   *struct{} `xml:"error"`
	Skipped *struct{} `xml:"skipped"`
}

// readJUnit adds the tests of a JUnit XML report to results, by id —
// classname.name, or the name alone — and, when suites is not nil, the
// name of the testsuite holding each one (a Go package, a pytest run), or
// its classname when the suite has none. A test run more than once (a
// parameter, a retry) is red when one of its runs is.
func readJUnit(file string, results map[string]result, suites map[string]string) error {
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
			id := c.Name
			if c.Class != "" {
				id = c.Class + "." + c.Name
			}
			r := passed
			switch {
			case c.Failure != nil || c.Error != nil:
				r = failed
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
