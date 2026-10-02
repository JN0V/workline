package role

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ConfigError is a .workline/config.yaml the engine cannot trust: unreadable,
// or holding a key it does not know. A key that is ignored is a setting
// someone believes in and nothing applies, so it blocks (principle 12).
type ConfigError struct{ msg string }

func (e *ConfigError) Error() string { return ".workline/config.yaml: " + e.msg }

// IsConfigError reports whether err, or an error it wraps, is a ConfigError.
func IsConfigError(err error) bool {
	var c *ConfigError
	return errors.As(err, &c)
}

// shape is what a part of the config may hold: named keys, keys of any name
// (roles, gates, repositories), or anything (a role's settings, the steps of an
// event: they belong to the role or the routing, which check them).
type shape struct {
	keys map[string]*shape
	any  *shape
	free bool
}

var (
	free   = &shape{free: true}
	config = &shape{keys: map[string]*shape{
		"ai":    free,
		"forge": free,
		"roles": {any: &shape{keys: map[string]*shape{"settings": free, "enforce": free}}},
		"routing": {keys: map[string]*shape{
			"events":       free,
			"handoffs":     {keys: map[string]*shape{"from": free, "to": free}}, // a list: each item
			"max-handoffs": free,
		}},
		"gates": {any: &shape{keys: map[string]*shape{
			"criteria": free,
			"checks":   {keys: map[string]*shape{"id": free, "run": free, "output": free, "max": free, "optional": free, "timeout": free}},
			"enforce":  free,
		}}},
		"repos": {any: &shape{keys: map[string]*shape{"url": free, "branch": free}}},
	}}
)

// hints say what to do instead, for keys the specs describe and the engine
// does not read yet.
var hints = map[string]string{
	"roles.*.from": "a role taken from elsewhere is not built yet; point --roles or WORKLINE_ROLES to a folder of roles instead",
}

// CheckConfig refuses a config holding a key the engine does not know.
func CheckConfig(data []byte) error {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return &ConfigError{err.Error()}
	}
	if len(doc.Content) == 0 {
		return nil
	}
	if err := check(doc.Content[0], config, nil, nil); err != nil {
		return err
	}
	return asWritten(doc.Content[0])
}

// asWritten refuses an unquoted value written as a commit is (hex digits)
// that YAML reads as a number other than written: 0123456 (octal), 1234e56
// (a float), forty digits (past an integer). Read as text, it would be a
// commit mangled; quoted, it is the commit. A number YAML reads as written
// (7515148) is let through: a setting holding a commit reads it as text.
func asWritten(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		if n.Style != 0 || !hexDigits(n.Value) || (n.ShortTag() != "!!int" && n.ShortTag() != "!!float") {
			return nil
		}
		var v any
		if err := n.Decode(&v); err == nil && fmt.Sprint(v) == n.Value {
			return nil
		}
		return &ConfigError{fmt.Sprintf("line %d: %s reads as a number, not as written: quote it ('%s') if it is a commit or a text", n.Line, n.Value, n.Value)}
	}
	for _, c := range n.Content {
		if err := asWritten(c); err != nil {
			return err
		}
	}
	return nil
}

func hexDigits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789abcdefABCDEF") == ""
}

func check(n *yaml.Node, s *shape, path, pattern []string) error {
	if s.free {
		return nil
	}
	switch n.Kind {
	case yaml.SequenceNode:
		for _, item := range n.Content {
			if err := check(item, s, path, pattern); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			next, pat := s.keys[k.Value], k.Value
			if next == nil && s.any != nil {
				next, pat = s.any, "*"
			}
			if next == nil {
				return unknown(k, s, path, pattern)
			}
			if err := check(n.Content[i+1], next, append(path, k.Value), append(pattern, pat)); err != nil {
				return err
			}
		}
	}
	return nil
}

func unknown(k *yaml.Node, s *shape, path, pattern []string) error {
	where := "at the top level"
	if len(path) > 0 {
		where = "in " + strings.Join(path, ".")
	}
	msg := fmt.Sprintf("line %d: unknown key %q %s", k.Line, k.Value, where)
	if hint, ok := hints[strings.Join(append(pattern, k.Value), ".")]; ok {
		return &ConfigError{msg + ": " + hint}
	}
	known := make([]string, 0, len(s.keys))
	for name := range s.keys {
		known = append(known, name)
	}
	sort.Strings(known)
	return &ConfigError{fmt.Sprintf("%s (known: %s)", msg, strings.Join(known, ", "))}
}
