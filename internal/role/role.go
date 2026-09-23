// Package role loads a role's contract (role.yaml) and the project's settings
// for it, as described in docs/spec/role-contract.md.
package role

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Role is a parsed role.yaml, plus the folder it was loaded from.
type Role struct {
	Contract   int            `yaml:"contract"`
	Name       string         `yaml:"name"`
	Mission    string         `yaml:"mission"`
	Events     []string       `yaml:"events"`
	Requires   []string       `yaml:"requires"`
	Uses       []string       `yaml:"uses"`
	Intentions []string       `yaml:"intentions"`
	WithoutAI  string         `yaml:"without-ai"`
	Settings   map[string]any `yaml:"settings"`

	Dir string `yaml:"-"`
}

// Load reads roles/<name>/role.yaml from rolesDir.
func Load(rolesDir, name string) (*Role, error) {
	dir := filepath.Join(rolesDir, name)
	data, err := os.ReadFile(filepath.Join(dir, "role.yaml"))
	if err != nil {
		return nil, fmt.Errorf("role %q: %w", name, err)
	}
	var r Role
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("role %q: role.yaml: %w", name, err)
	}
	if r.Contract != 1 {
		return nil, fmt.Errorf("role %q: contract version %d is not supported", name, r.Contract)
	}
	if r.Name != name {
		return nil, fmt.Errorf("role %q: role.yaml names itself %q", name, r.Name)
	}
	r.Dir = dir
	return &r, nil
}

// Accepts reports whether the role can run on event.
func (r *Role) Accepts(event string) bool {
	for _, e := range r.Events {
		if e == event {
			return true
		}
	}
	return false
}

// Allows reports whether the role may emit an intention of this kind.
func (r *Role) Allows(kind string) bool {
	for _, k := range r.Intentions {
		if k == kind {
			return true
		}
	}
	return false
}

// ProjectConfig is the part of .assembly/config.yaml the engine reads today.
type ProjectConfig struct {
	Roles map[string]struct {
		Settings map[string]any   `yaml:"settings"`
		Enforce  map[string]string `yaml:"enforce"`
	} `yaml:"roles"`
}

// LoadProjectConfig reads <repo>/.assembly/config.yaml. A missing file is an
// empty config; an unreadable or invalid one is an error.
func LoadProjectConfig(repo string) (*ProjectConfig, error) {
	var c ProjectConfig
	data, err := os.ReadFile(filepath.Join(repo, ".assembly", "config.yaml"))
	if os.IsNotExist(err) {
		return &c, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf(".assembly/config.yaml: %w", err)
	}
	return &c, nil
}

// MergedSettings returns the role's defaults overridden by the project's settings.
func (r *Role) MergedSettings(c *ProjectConfig) map[string]any {
	out := map[string]any{}
	for k, v := range r.Settings {
		out[k] = v
	}
	if rc, ok := c.Roles[r.Name]; ok {
		for k, v := range rc.Settings {
			out[k] = v
		}
	}
	return out
}

// Enforcement returns how hard each rule bites for this role in this project.
func (r *Role) Enforcement(c *ProjectConfig) map[string]string {
	if rc, ok := c.Roles[r.Name]; ok && rc.Enforce != nil {
		return rc.Enforce
	}
	return map[string]string{}
}
