package testmark

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/JN0V/workline/internal/work"
)

// Answered is an answer to an objection, as its reply records it.
type Answered struct {
	Act      string        `json:"act"`             // kept or rewritten
	At       string        `json:"at"`              // the commit the test was read at
	Model    string        `json:"model,omitempty"` // the agents and models asked, <agent>:<model>, a comma between two
	Scenario work.Scenario `json:"scenario"`        // the scenario, as the run read it
	Test     string        `json:"test"`            // the test objected to
	Blob     string        `json:"blob,omitempty"`  // a rewrite's: the test as rewritten
	Whys     bool          `json:"whys,omitempty"`  // the reply holds the five whys, the third answer's
	// Missing: an answer to a test the review finds missing (#1265):
	// kept, none needed; rewritten, the test added.
	Missing bool `json:"missing,omitempty"`
}

var answeredData = regexp.MustCompile(`<!-- workline:tester-act (\{.*\}) -->`)

// Line is the hidden line a reply keeps it in.
func (a Answered) Line() string {
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
