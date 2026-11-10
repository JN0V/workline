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
	return setLevel(repo, []string{"roles", "product-owner", "settings"}, level,
		"How far the product owner goes alone (ADR-0026): cautious, normal or enterprising.")
}

// SetProjectAutonomy sets the project's default level, the top-level
// `autonomy` (ADR-0060, §3), unless the project set one already: as
// SetAutonomy.
func SetProjectAutonomy(repo, level string) (string, bool, error) {
	return setLevel(repo, nil, level,
		"Each role's autonomy unless it sets its own (ADR-0060): cautious, normal or enterprising. The merge keeps merge: {autonomy: …}.")
}

// setLevel sets `autonomy` under the keys named, made when missing.
func setLevel(repo string, under []string, level, comment string) (string, bool, error) {
	c, err := readConfig(repo)
	if err != nil {
		return "", false, err
	}
	top := c.doc.Content[0]
	// A key there with nothing under it (`roles:`, `roles: ~`) becomes a
	// mapping; one holding a value of its own is the person's: refused.
	child := func(m *yaml.Node, key, where string) (*yaml.Node, error) {
		v := value(m, key)
		switch {
		case v == nil:
			v = &yaml.Node{Kind: yaml.MappingNode}
			m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
		case v.Kind == yaml.ScalarNode && v.Tag == "!!null":
			// Its comments go to its key: a block mapping holds no line comment.
			for i := 0; i+1 < len(m.Content); i += 2 {
				if k := m.Content[i]; m.Content[i+1] == v {
					k.LineComment = strings.TrimSpace(k.LineComment + " " + v.LineComment)
					k.HeadComment = strings.TrimSpace(k.HeadComment + "\n" + v.HeadComment)
					k.FootComment = strings.TrimSpace(k.FootComment + "\n" + v.FootComment)
				}
			}
			*v = yaml.Node{Kind: yaml.MappingNode, Line: v.Line, Column: v.Column}
		case v.Kind != yaml.MappingNode:
			return nil, fmt.Errorf(".workline/config.yaml: %s is not a mapping: autonomy not set", where+key)
		}
		return v, nil
	}
	settings, where := top, ""
	for _, key := range under {
		if settings, err = child(settings, key, where); err != nil {
			return "", false, err
		}
		where += key + "."
	}
	switch v := value(settings, "autonomy"); {
	case v == nil:
		settings.Content = append(settings.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "autonomy", HeadComment: comment},
			&yaml.Node{Kind: yaml.ScalarNode, Value: level})
	case v.Kind == yaml.ScalarNode && v.Tag == "!!null":
		v.Tag, v.Value, v.Style = "", level, 0 // `autonomy:` with nothing set: set now
	case v.Kind != yaml.ScalarNode:
		return "", false, fmt.Errorf(".workline/config.yaml: %sautonomy is not a level: autonomy not set", where)
	default:
		return v.Value, false, nil // the project's choice
	}
	if err := c.write(); err != nil {
		return "", false, err
	}
	return level, true, nil
}

// SetPacks lists the language packs in the project's
// .workline/config.yaml (ADR-0065), unless `packs:` holds a list
// already, `packs: []` too (the person's choice: the generic pack
// alone), or none were given: then the file is left as it is. The
// packs listed are empty, not nil, for `packs: []`. It returns the packs
// listed afterwards, and whether the file changed.
func SetPacks(repo string, names []string) ([]string, bool, error) {
	c, err := readConfig(repo)
	if err != nil {
		return nil, false, err
	}
	top := c.doc.Content[0]
	switch v := value(top, "packs"); {
	case v != nil && v.Kind == yaml.SequenceNode:
		listed := []string{}
		for _, n := range v.Content {
			listed = append(listed, n.Value)
		}
		return listed, false, nil // the project's choice
	case v != nil && (v.Kind != yaml.ScalarNode || v.Tag != "!!null") && v.Kind != yaml.SequenceNode:
		return nil, false, fmt.Errorf(".workline/config.yaml: packs is not a list: packs not set")
	case len(names) == 0:
		return nil, false, nil
	}
	list := &yaml.Node{Kind: yaml.SequenceNode}
	for _, n := range names {
		list.Content = append(list.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: n})
	}
	if v := value(top, "packs"); v != nil {
		*v = *list // `packs:` or `packs: ~`, nothing written
	} else {
		top.Content = append(top.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "packs",
			HeadComment: "The language packs every role reads (ADR-0065), detected by `workline init`: docs/packs.md."}, list)
	}
	if err := c.write(); err != nil {
		return nil, false, err
	}
	return names, true, nil
}

// config is the project's .workline/config.yaml, read to be edited.
type config struct {
	file string
	doc  yaml.Node
	kept []byte // a file of comments alone, which the parser drops: kept above
}

// readConfig reads the project's config, a mapping, empty when missing.
func readConfig(repo string) (*config, error) {
	c := &config{file: filepath.Join(repo, ".workline", "config.yaml")}
	data, err := os.ReadFile(c.file)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &c.doc); err != nil {
		return nil, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	if len(c.doc.Content) == 0 {
		c.kept = bytes.TrimRight(data, " \n") // `---` or comments
		c.doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	if c.doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf(".workline/config.yaml: not a mapping")
	}
	return c, nil
}

// write writes the config back.
func (c *config) write() error {
	var out bytes.Buffer
	if len(c.kept) > 0 && !bytes.Equal(c.kept, []byte("---")) {
		out.Write(append(c.kept, '\n'))
	}
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&c.doc); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.file, out.Bytes(), 0o644)
}
