package routing

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// adopted is the pre-push line a repository adopting workline gets.
var adopted = []string{"committer", "documentalist"}

// RoutePrePush has the project's .workline/config.yaml route pre-push to the
// committer and the documentalist, unless it routes pre-push already: then it
// is the project's choice, left as it is. It returns the steps pre-push runs
// afterwards, and whether the file changed. The file is appended to when it
// has no routing yet, so what a person wrote stays as they wrote it.
func RoutePrePush(repo string) (steps []string, changed bool, err error) {
	file := filepath.Join(repo, ".workline", "config.yaml")
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, false, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	var top *yaml.Node
	if len(doc.Content) > 0 {
		top = doc.Content[0]
	}
	if top != nil && top.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf(".workline/config.yaml: not a mapping")
	}
	routing := value(top, "routing")
	events := value(routing, "events")
	if prePush := value(events, "pre-push"); prePush != nil {
		if err := prePush.Decode(&steps); err != nil {
			return nil, false, fmt.Errorf(".workline/config.yaml: routing.events.pre-push: %w", err)
		}
		return steps, false, nil
	}
	line := "{pre-push: [" + strings.Join(adopted, ", ") + "]}"
	const why = "# Before a push, the committer checks the commits leaving the machine, and\n# the documentalist has the docs they made suspect judged.\n"
	switch {
	case routing == nil:
		if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
			data = append(data, '\n')
		}
		data = append(data, []byte("routing:\n  "+strings.ReplaceAll(strings.TrimSuffix(why, "\n"), "\n", "\n  ")+"\n  events: "+line+"\n")...)
	default:
		// Inside a routing section already there: the node is edited, and the
		// file written again from it.
		if events == nil {
			events = &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
			routing.Content = append(routing.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "events"}, events)
		}
		list := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
		for _, s := range adopted {
			list.Content = append(list.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: s})
		}
		events.Content = append(events.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "pre-push"}, list)
		var out bytes.Buffer
		enc := yaml.NewEncoder(&out)
		enc.SetIndent(2)
		if err := enc.Encode(&doc); err != nil {
			return nil, false, err
		}
		data = out.Bytes()
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(file, data, 0o644); err != nil {
		return nil, false, err
	}
	return adopted, true, nil
}

// AddReviewer has the project's merge requests reviewed (ADR-0020): the
// reviewer is added at the end of the project's merge-request line, or of
// the shipped one when the project routes none. It returns the steps the
// line runs afterwards, and whether the file changed.
func AddReviewer(repo string) (steps []string, changed bool, err error) {
	file := filepath.Join(repo, ".workline", "config.yaml")
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, false, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	top := doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf(".workline/config.yaml: not a mapping")
	}
	child := func(m *yaml.Node, key string) *yaml.Node {
		if v := value(m, key); v != nil {
			return v
		}
		v := &yaml.Node{Kind: yaml.MappingNode}
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
		return v
	}
	events := child(child(top, "routing"), "events")
	line := value(events, "merge-request")
	if line == nil {
		c, err := Load(repo)
		if err != nil {
			return nil, false, err
		}
		line = &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
		for _, s := range c.Events["merge-request"] {
			line.Content = append(line.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: s})
		}
		events.Content = append(events.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "merge-request"}, line)
	}
	if err := line.Decode(&steps); err != nil {
		return nil, false, fmt.Errorf(".workline/config.yaml: routing.events.merge-request: %w", err)
	}
	for _, s := range steps {
		if s == "reviewer" {
			return steps, false, nil
		}
	}
	line.Content = append(line.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "reviewer"})
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, false, err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(file, out.Bytes(), 0o644); err != nil {
		return nil, false, err
	}
	return append(steps, "reviewer"), true, nil
}

// value returns the value of key in a mapping node, or nil.
func value(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// SetAutonomy sets the product owner's autonomy level in the project's
// .workline/config.yaml (ADR-0026), unless the project set one already:
// then it is the project's choice, returned as it is. It returns the level
// in force afterwards, and whether the file changed.
func SetAutonomy(repo, level string) (string, bool, error) {
	file := filepath.Join(repo, ".workline", "config.yaml")
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", false, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	top := doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return "", false, fmt.Errorf(".workline/config.yaml: not a mapping")
	}
	child := func(m *yaml.Node, key string) *yaml.Node {
		if v := value(m, key); v != nil {
			return v
		}
		v := &yaml.Node{Kind: yaml.MappingNode}
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
		return v
	}
	settings := child(child(child(top, "roles"), "product-owner"), "settings")
	if v := value(settings, "autonomy"); v != nil {
		return v.Value, false, nil
	}
	settings.Content = append(settings.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "autonomy",
		HeadComment: "How far the product owner goes alone (ADR-0026): cautious, normal or enterprising."},
		&yaml.Node{Kind: yaml.ScalarNode, Value: level})
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(file, out.Bytes(), 0o644); err != nil {
		return "", false, err
	}
	return level, true, nil
}
