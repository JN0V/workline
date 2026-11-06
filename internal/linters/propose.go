package linters

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Proposal is a gate check proposed for a language no linter reads: the
// usual linter, pinned where the table knows its version, and the check
// that runs it (#244).
type Proposal struct {
	Language string `json:"language"`
	Tool     string `json:"tool"`
	Version  string `json:"version,omitempty"`
	ID       string `json:"id"`
	Check    string `json:"check"` // the gate check, as YAML
}

// Propose is the lint checks to add for the languages the repository holds
// that no linter reads: the first suggested tool with a known command, once
// per tool. Nothing is written.
func Propose(repo string) ([]Proposal, error) {
	rep, err := Read(repo)
	if err != nil {
		return nil, err
	}
	var out []Proposal
	done := map[string]bool{}
	for _, l := range rep.Languages {
		for _, s := range l.Missing[Lint] {
			if s.Check == "" || s.Each != "" || done[s.Tool] {
				continue
			}
			done[s.Tool] = true
			out = append(out, Proposal{Language: l.Name, Tool: s.Tool, Version: s.Version,
				ID: strings.ReplaceAll(s.Tool, " ", "-"), Check: s.Check})
			break
		}
	}
	return out, nil
}

// Unproposed is the languages that no linter reads and for which Propose has
// no check to write: every tool suggested lacks a known command, or runs
// once from each folder. They are said, never silently left unlinted.
func Unproposed(repo string, ps []Proposal) ([]string, error) {
	rep, err := Read(repo)
	if err != nil {
		return nil, err
	}
	proposed := map[string]bool{}
	for _, p := range ps {
		proposed[p.Language] = true
	}
	var out []string
	for _, l := range rep.Languages {
		if len(l.Missing[Lint]) == 0 || proposed[l.Name] {
			continue
		}
		var tools []string
		for _, s := range l.Missing[Lint] {
			tools = append(tools, s.Tool)
		}
		out = append(out, l.Name+" ("+strings.Join(tools, ", ")+")")
	}
	return out, nil
}

// Adopt writes the proposals as checks of the gate named, in the project's
// .workline/config.yaml, skipping a check whose id the gate holds already.
// It returns the ids added.
func Adopt(repo, gate string, ps []Proposal) ([]string, error) {
	file := filepath.Join(repo, ".workline", "config.yaml")
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	var kept []byte
	if len(doc.Content) == 0 {
		kept = bytes.TrimRight(data, " \n")
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	node := doc.Content[0]
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf(".workline/config.yaml: not a mapping")
	}
	child := func(m *yaml.Node, key string, kind yaml.Kind) (*yaml.Node, error) {
		for i := 0; i+1 < len(m.Content); i += 2 {
			if m.Content[i].Value == key {
				v := m.Content[i+1]
				if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
					*v = yaml.Node{Kind: kind}
				}
				if v.Kind != kind {
					return nil, fmt.Errorf(".workline/config.yaml: %s is not what a check can be added to", key)
				}
				return v, nil
			}
		}
		v := &yaml.Node{Kind: kind}
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
		return v, nil
	}
	gates, err := child(node, "gates", yaml.MappingNode)
	if err != nil {
		return nil, err
	}
	g, err := child(gates, gate, yaml.MappingNode)
	if err != nil {
		return nil, err
	}
	checks, err := child(g, "checks", yaml.SequenceNode)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, c := range checks.Content {
		for i := 0; i+1 < len(c.Content); i += 2 {
			if c.Content[i].Value == "id" {
				have[c.Content[i+1].Value] = true
			}
		}
	}
	var added []string
	for _, p := range ps {
		if have[p.ID] {
			continue
		}
		var c yaml.Node
		if err := yaml.Unmarshal([]byte(p.Check), &c); err != nil || len(c.Content) == 0 {
			return added, fmt.Errorf("the check for %s is not YAML: %v", p.Tool, err)
		}
		checks.Content = append(checks.Content, c.Content[0])
		added = append(added, p.ID)
	}
	if len(added) == 0 {
		return nil, nil
	}
	var out bytes.Buffer
	if len(kept) > 0 && !bytes.Equal(kept, []byte("---")) {
		out.Write(append(kept, '\n'))
	}
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return nil, err
	}
	return added, os.WriteFile(file, out.Bytes(), 0o644)
}
