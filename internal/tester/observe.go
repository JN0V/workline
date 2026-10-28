package tester

// What the auditor reads of the tester's acts (ADR-0046, the tester's row):
// each act where it is done, completed only where a field lacked. Its
// tests: the commit (`Workline-Role: tester`, `Workline-Model`) and the
// record on the Draft, which keeps the scenarios as the run read them and
// the base it proved them red on. An answer kept or rewritten: a hidden
// line in its reply — the commit the test was read at, the models asked,
// the scenario, the test, a rewrite's blob. A removal: the curator's
// commit and its pull request's proof.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/forge"
)

// Record is what the tester keeps on its Draft, as the auditor reads it.
type Record = record

// ReadRecord reads the record of the tester's Draft id from workline's own
// comment; ok is false when there is none it can read.
func ReadRecord(f forge.Forge, id int) (Record, bool, error) {
	r, missing, err := readRecord(f, id)
	return r, err == nil && missing == nil, err
}

// The tester's branches: an issue's tests, a folder curated.
const (
	TestsBranch  = branchPrefix
	CurateBranch = "workline/tester/" + curateKey
)

// TestsSubject and CurateSubject open the subjects of the tester's
// commits of its tests and of a removal: its merge request's title.
const (
	TestsSubject  = "test: the acceptance tests of #"
	CurateSubject = "test: curate "
)

// Answered is an answer to an objection, as its reply records it.
type Answered struct {
	Act      string   `json:"act"`             // kept or rewritten
	At       string   `json:"at"`              // the commit the test was read at
	Model    string   `json:"model,omitempty"` // the agents and models asked, <agent>:<model>, a comma between two
	Scenario Scenario `json:"scenario"`        // the scenario, as the run read it
	Test     string   `json:"test"`            // the test objected to
	Blob     string   `json:"blob,omitempty"`  // a rewrite's: the test as rewritten
}

var answeredData = regexp.MustCompile(`<!-- workline:tester-act (\{.*\}) -->`)

// line is the hidden line a reply keeps it in.
func (a Answered) line() string {
	data, _ := json.Marshal(a)
	return fmt.Sprintf("<!-- workline:tester-act %s -->", data)
}

// ReadAnswered reads an answer's record from a reply of the tester's.
func ReadAnswered(body string) (Answered, bool) {
	m := answeredData.FindStringSubmatch(body)
	var a Answered
	if m == nil || json.Unmarshal([]byte(m[1]), &a) != nil || a.Act == "" {
		return Answered{}, false
	}
	return a, true
}

// IsAnswered says whether a comment is the tester's answer to an objection.
func IsAnswered(body string) bool { return strings.Contains(body, answerMark) }

// asked are the agents and models the run asked, as <agent>:<model>,
// from its calls (out/calls.jsonl); a judge is not one.
func asked(runDir string) string {
	data, err := os.ReadFile(filepath.Join(runDir, "out", "calls.jsonl"))
	if err != nil {
		return ""
	}
	var out []string
	for _, l := range strings.Split(string(data), "\n") {
		var c agent.Call
		if json.Unmarshal([]byte(l), &c) != nil || c.Task == "judge" || c.Model == "" {
			continue
		}
		if m := c.Agent + ":" + c.Model; !slices.Contains(out, m) {
			out = append(out, m)
		}
	}
	return strings.Join(out, ", ")
}
