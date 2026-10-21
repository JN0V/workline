package verdict

import (
	"slices"
	"testing"
)

func TestEnforce(t *testing.T) {
	story := Finding{Rule: "bug-story", Where: "a.go:3"}
	code := Finding{Rule: "internal-code", Where: "a.go:4"}
	unread := Finding{Rule: "record-unread", Level: "warn"}
	warned := func(f Finding) Finding { f.Level = "warn"; return f }
	for _, c := range []struct {
		name     string
		status   string
		findings []Finding
		enforce  map[string]string
		want     string
		kept     []Finding
	}{
		{"a warning already never counts as blocking (#550)", Block, []Finding{story, unread},
			map[string]string{"bug-story": "warn"}, Pass, []Finding{warned(story), unread}},
		{"a rule left at block still blocks", Block, []Finding{story, code, unread},
			map[string]string{"bug-story": "warn"}, Block, []Finding{warned(story), code, unread}},
		{"off drops the finding, and its block", Block, []Finding{story, unread},
			map[string]string{"bug-story": "off"}, Pass, []Finding{unread}},
		{"off on the only finding lifts the block", Block, []Finding{story},
			map[string]string{"bug-story": "off"}, Pass, nil},
		{"a block with no finding is not enforcement's to lift", Block, nil,
			map[string]string{"bug-story": "warn"}, Block, nil},
		{"a block with warnings only, none lowered, stands", Block, []Finding{unread},
			map[string]string{"record-unread": "warn"}, Block, []Finding{unread}},
		{"block keeps the level its role gave (raising: #99)", Block, []Finding{story, unread},
			map[string]string{"bug-story": "warn", "record-unread": "block"}, Pass, []Finding{warned(story), unread}},
		{"a pass stays a pass", Pass, []Finding{unread}, nil, Pass, []Finding{unread}},
	} {
		t.Run(c.name, func(t *testing.T) {
			v := &Verdict{Status: c.status, Findings: slices.Clone(c.findings)}
			Enforce(v, c.enforce)
			if v.Status != c.want {
				t.Errorf("status %s, want %s", v.Status, c.want)
			}
			if !slices.Equal(v.Findings, c.kept) {
				t.Errorf("findings %v, want %v", v.Findings, c.kept)
			}
		})
	}
}
