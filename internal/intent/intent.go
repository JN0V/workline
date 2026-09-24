// Package intent reads the proposals an agent writes and checks them against
// the closed catalogue (docs/spec/role-contract.md, "Intentions").
package intent

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

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

// Merge returns the fallback proposals with the agent's in place of those of
// the same kind (docs/spec/role-contract.md, "One run"). A patch replaces only
// the fallback patches of the files it touches: the agent fixing one doc does
// not drop what the role regenerated in another.
func Merge(fallback, agent []Intention) []Intention {
	proposed := map[string]bool{}
	patched := map[string]bool{}
	for _, a := range agent {
		proposed[a.Kind] = true
		if a.Kind == "patch" {
			for _, f := range PatchFiles(a.Value) {
				patched[f] = true
			}
		}
	}
	var out []Intention
	for _, f := range fallback {
		switch {
		case f.Kind == "patch":
			kept := true
			for _, file := range PatchFiles(f.Value) {
				kept = kept && !patched[file]
			}
			if kept {
				out = append(out, f)
			}
		case !proposed[f.Kind]:
			out = append(out, f)
		}
	}
	return append(out, agent...)
}

// PatchFiles lists the files a patch names: its `file`, or the `+++` lines of
// its diff. It reads names only; whether the diff applies is checked later.
func PatchFiles(v any) []string {
	if m, ok := v.(map[string]any); ok {
		if f, ok := m["file"].(string); ok {
			return []string{path.Clean(f)}
		}
		return nil
	}
	diff, _ := v.(string)
	var files []string
	for _, l := range strings.Split(diff, "\n") {
		if p, ok := strings.CutPrefix(l, "+++ "); ok {
			p, _, _ = strings.Cut(strings.TrimSpace(p), "\t")
			files = append(files, path.Clean(strings.TrimPrefix(p, "b/")))
		}
	}
	return files
}

// Write saves proposals in the same shape Read expects. No proposals, no file.
func Write(path string, in []Intention) error {
	if len(in) == 0 {
		return nil
	}
	list := make([]map[string]any, len(in))
	for i, x := range in {
		list[i] = map[string]any{x.Kind: x.Value}
	}
	data, err := yaml.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// applyOrder is the order intentions are applied in, whatever order they were
// proposed in: files first, then what depends on them (a release commits and
// tags what the patches wrote), then what only informs.
var applyOrder = map[string]int{
	"commit-message": 0, "patch": 1, "release": 2, "label": 3, "comment": 4, "issue": 5, "handoff": 6, "note": 7,
}

// SortForApply puts intentions in apply order, keeping the proposed order within a kind.
func SortForApply(in []Intention) {
	sort.SliceStable(in, func(i, j int) bool { return applyOrder[in[i].Kind] < applyOrder[in[j].Kind] })
}
