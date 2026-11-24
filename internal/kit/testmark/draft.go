package testmark

import (
	"encoding/json"
	"regexp"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// DraftRecordKey is the sticky key of the record the tester keeps on its
// Draft: its frozen tests and what it proved them on.
const DraftRecordKey = "tests"

// DraftData is the hidden line that record is kept in.
var DraftData = regexp.MustCompile(`<!-- workline:tester-record (\{.*\}) -->`)

// Draft is what another role reads of the tester's record on its Draft:
// the issue its tests are for, the base it proved them red on, each
// frozen test file's blob, the issue's scenarios as the run that wrote
// them read them, each scenario's tests (Covers), and why its red proof
// was not settled (RedUnproven); none of the last three in a record
// written before. The rest of the record is the tester's own.
type Draft struct {
	Issue       int                 `json:"issue"`
	Base        string              `json:"base"`
	Files       map[string]string   `json:"files"`
	Scenarios   []work.Scenario     `json:"scenarios,omitempty"`
	Covers      map[string][]string `json:"covers,omitempty"`
	RedUnproven string              `json:"red-unproven,omitempty"`
	// Unproven: the scenarios no test proves, each with the tester's
	// reason; none in a record written before.
	Unproven []Unproven `json:"unproven,omitempty"`
}

// Unproven is a scenario no test proves, and why, as the tester said it:
// the reviewer's tests lens weighs whether a test can prove it.
type Unproven struct {
	Scenario string `json:"scenario"`
	Why      string `json:"why,omitempty"`
}

// ReadDraft reads the tester's record on its Draft id from workline's own
// comment; ok is false when there is none it can read: no such comment,
// no record line in it that parses, or one naming no file.
func ReadDraft(f forge.Forge, id int) (Draft, bool, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		return Draft{}, false, err
	}
	body, ok := kept.Last(forge.Marker("sticky=tester/" + DraftRecordKey))
	if !ok {
		return Draft{}, false, nil
	}
	m := DraftData.FindStringSubmatch(body)
	var d Draft
	if m == nil || json.Unmarshal([]byte(m[1]), &d) != nil || len(d.Files) == 0 {
		return Draft{}, false, nil
	}
	return d, true, nil
}
