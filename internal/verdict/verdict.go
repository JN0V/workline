// Package verdict reads and writes the verdict a run ends with.
package verdict

import (
	"os"

	"go.yaml.in/yaml/v3"
)

// The statuses a run can end with (docs/spec/role-contract.md, "Verdict").
const (
	Pass            = "pass"
	Block           = "block"
	Human           = "human"
	BlockedExternal = "blocked-external"
)

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

// Enforce applies a project's per-rule enforcement: "off" drops a finding,
// "warn" keeps it without blocking. A block whose findings are all warnings
// becomes a pass that still shows them.
func Enforce(v *Verdict, enforce map[string]string) {
	var kept []Finding
	blocking := 0
	for _, f := range v.Findings {
		switch enforce[f.Rule] {
		case "off":
			continue
		case "warn":
			f.Level = "warn"
		default:
			blocking++
		}
		kept = append(kept, f)
	}
	v.Findings = kept
	if v.Status == Block && blocking == 0 && len(kept) > 0 {
		v.Status = Pass
	}
}
