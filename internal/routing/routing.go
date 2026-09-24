// Package routing reads which roles and gates run on which event, and which
// handoffs are allowed (docs/spec/routing.md). It decides nothing by itself.
package routing

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JN0V/workline"
	"go.yaml.in/yaml/v3"
)

// Edge is one handoff a role may ask for.
type Edge struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// Config is the line: steps per event, and the allowed handoffs.
type Config struct {
	Events      map[string][]string `yaml:"events"`
	Handoffs    *[]Edge             `yaml:"handoffs"`
	MaxHandoffs int                 `yaml:"max-handoffs"`
}

// Load returns the shipped routing with the project's changes applied.
func Load(repo string) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(workline.DefaultRouting, &c); err != nil {
		return nil, fmt.Errorf("default routing: %w", err)
	}
	var project struct {
		Routing *Config `yaml:"routing"`
	}
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	if p := project.Routing; p != nil {
		for event, steps := range p.Events {
			c.Events[event] = steps
		}
		if p.Handoffs != nil {
			c.Handoffs = p.Handoffs
		}
		if p.MaxHandoffs > 0 {
			c.MaxHandoffs = p.MaxHandoffs
		}
	}
	return &c, nil
}

// Allowed reports whether role from may hand over to role to.
func (c *Config) Allowed(from, to string) bool {
	if c.Handoffs == nil {
		return false
	}
	for _, e := range *c.Handoffs {
		if e.From == from && e.To == to {
			return true
		}
	}
	return false
}
