// Package intent reads the proposals an agent writes and checks them against
// the closed catalogue (docs/spec/role-contract.md, "Intentions").
package intent

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Catalogue is every kind of intention the engine knows how to apply.
var Catalogue = map[string]bool{
	"commit-message": true,
	"patch":          true,
	"comment":        true,
	"label":          true,
	"issue":          true,
	"release":        true,
	"handoff":        true,
	"note":           true,
}

// Intention is one proposal: its kind and its value.
type Intention struct {
	Kind  string
	Value any
}

// Read loads out/intentions.yaml: a list of one-key maps, e.g.
// `- commit-message: "fix: ..."`. A missing file means no intentions.
func Read(path string) ([]Intention, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("intentions: %w", err)
	}
	var out []Intention
	for i, m := range raw {
		if len(m) != 1 {
			return nil, fmt.Errorf("intentions: entry %d must have exactly one kind", i+1)
		}
		for k, v := range m {
			out = append(out, Intention{Kind: k, Value: v})
		}
	}
	return out, nil
}

// Kinds lists the kinds of a set of intentions, in order.
func Kinds(in []Intention) []string {
	var k []string
	for _, i := range in {
		k = append(k, i.Kind)
	}
	return k
}
