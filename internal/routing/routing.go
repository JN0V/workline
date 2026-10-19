// Package routing reads which roles and gates run on which event, and which
// handoffs are allowed (docs/spec/routing.md). It decides nothing by itself.
package routing

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/JN0V/workline"
	"github.com/JN0V/workline/internal/role"
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
	// FailFast says, per event, whether the first step that does not pass
	// stops the line (true, the default) or every step runs and the line
	// takes the worst verdict (false: the schedule's upkeep, ADR-0037).
	FailFast map[string]bool `yaml:"fail-fast"`
	// Budgets: each moment's wait budget, in seconds (ADR-0050, section
	// 1): the shipped ones, a project's laid over them moment by moment.
	Budgets map[string]float64 `yaml:"budgets"`
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
	if err := role.CheckConfig(data); err != nil {
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
		for event, stop := range p.FailFast {
			if c.FailFast == nil {
				c.FailFast = map[string]bool{}
			}
			c.FailFast[event] = stop
		}
		for moment, seconds := range p.Budgets {
			if seconds <= 0 {
				return nil, role.ConfigErrorf("routing.budgets.%s is %v: a budget is a number of seconds above 0", moment, seconds)
			}
			if c.Budgets == nil {
				c.Budgets = map[string]float64{}
			}
			c.Budgets[moment] = seconds
		}
	}
	return &c, nil
}

// Budget is the moment's wait budget, in seconds; false when it has none
// (main after a merge, the night, the release wait for nobody).
func (c *Config) Budget(moment string) (float64, bool) {
	seconds, ok := c.Budgets[moment]
	return seconds, ok && seconds > 0
}

// StopsAtFirst reports whether the first step of event that does not pass
// stops the line: it does, unless the routing sets `fail-fast: false` for it.
func (c *Config) StopsAtFirst(event string) bool {
	stop, set := c.FailFast[event]
	return stop || !set
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

// Follows reports whether a line runs role then after role first, in the
// same event's steps: the reviewer after the product owner reads the specs
// it refined (#128).
func (c *Config) Follows(first, then string) bool {
	for _, steps := range c.Events {
		i := slices.Index(steps, first)
		if i >= 0 && slices.Contains(steps[i+1:], then) {
			return true
		}
	}
	return false
}
