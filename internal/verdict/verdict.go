// Package verdict reads and writes the verdict a run ends with.
package verdict

import (
	"os"
	"slices"

	"go.yaml.in/yaml/v3"
)

// The statuses a run can end with (docs/spec/role-contract.md, "Verdict").
const (
	Pass            = "pass"
	Block           = "block"
	Human           = "human"
	BlockedExternal = "blocked-external"
)

// WaitsForReady is the rule of the finding a role leaves on a draft where
// its agent would have judged (ADR-0050): it waits for the merge request
// to be marked ready. The merge policy reads a head holding it as a
// review to come, never as one given.
const WaitsForReady = "ai-waits-for-ready"

// Finding is one thing a check noticed.
type Finding struct {
	Rule    string `yaml:"rule" json:"rule"`
	Where   string `yaml:"where,omitempty" json:"where,omitempty"`
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
	Level   string `yaml:"level,omitempty" json:"level,omitempty"` // "warn" once enforcement downgraded it
}

// Verdict is the content of out/verdict.yaml.
type Verdict struct {
	Status   string    `yaml:"status" json:"status"`
	Summary  string    `yaml:"summary,omitempty" json:"summary,omitempty"`
	Findings []Finding `yaml:"findings,omitempty" json:"findings,omitempty"`
	// Final: a block that does not come from the agent's answer — a check
	// that could not run — so asking the agent again would change nothing.
	Final bool `yaml:"final,omitempty" json:"final,omitempty"`
}

// Read loads a verdict file.
func Read(path string) (*Verdict, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v Verdict
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Write saves a verdict file.
func Write(path string, v *Verdict) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Blocks says whether a finding blocks: one at no level, or at block.
func (f Finding) Blocks() bool { return f.Level == "" || f.Level == "block" }

// Levels applies a project's per-rule enforcement to findings: "off" drops
// one, "warn" keeps it without blocking; a rule the project does not name
// keeps the level its role gave it. A role that decides on its findings
// before its verdict, the reviewer's rules holding its lenses, reads them so.
func Levels(fs []Finding, enforce map[string]string) []Finding {
	var kept []Finding
	for _, f := range fs {
		switch enforce[f.Rule] {
		case "off":
			continue
		case "warn":
			f.Level = "warn"
		}
		kept = append(kept, f)
	}
	return kept
}

// Enforce applies a project's per-rule enforcement to a verdict (Levels).
// A block that enforcement leaves no finding to block on becomes a pass
// that still shows its warnings; a block no finding of which it lowered —
// none at all, or warnings only — is not its to lift.
func Enforce(v *Verdict, enforce map[string]string) {
	lowered := slices.ContainsFunc(v.Findings, func(f Finding) bool {
		return f.Blocks() && (enforce[f.Rule] == "off" || enforce[f.Rule] == "warn")
	})
	v.Findings = Levels(v.Findings, enforce)
	if v.Status == Block && lowered && !slices.ContainsFunc(v.Findings, Finding.Blocks) {
		v.Status = Pass
	}
}
