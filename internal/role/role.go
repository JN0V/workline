// Package role loads a role's contract (role.yaml) and the project's settings
// for it, as described in docs/spec/role-contract.md.
package role

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Role is a parsed role.yaml, plus the folder it was loaded from.
type Role struct {
	Contract   int      `yaml:"contract"`
	Name       string   `yaml:"name"`
	Mission    string   `yaml:"mission"`
	Events     []string `yaml:"events"`
	Requires   []string `yaml:"requires"`
	Uses       []string `yaml:"uses"`
	Intentions []string `yaml:"intentions"`
	Model      Model    `yaml:"model"`
	Duties     struct {
		Reads  []string `yaml:"reads"`
		Writes []string `yaml:"writes"`
	} `yaml:"duties"`
	Context struct {
		Knowledge []string `yaml:"knowledge"`
		Budget    int      `yaml:"budget"`
	} `yaml:"context"`
	WithoutAI string         `yaml:"without-ai"`
	Settings  map[string]any `yaml:"settings"`

	Dir string `yaml:"-"`
}

// Model is what kind of thinking a role needs (docs/spec/model-grid.md).
type Model struct {
	Capability   string `yaml:"capability"`
	Tier         string `yaml:"tier"`
	Effort       string `yaml:"effort"`
	PromoteAfter int    `yaml:"promote-after"`
	Timeout      string `yaml:"timeout"` // how long one answer may take, e.g. "10m"; default 3m
	// Tasks gives a kind of task, named by pre in in/task-kind, other needs
	// than the role's: condensing a doc needs more than judging one.
	Tasks map[string]Model `yaml:"tasks"`
}

// For returns the needs of a task of that kind: the role's, with what
// model.tasks sets for the kind in their place.
func (m Model) For(kind string) Model {
	t, ok := m.Tasks[kind]
	if !ok {
		return m
	}
	out := m
	out.Tasks = nil
	if t.Capability != "" {
		out.Capability = t.Capability
	}
	if t.Tier != "" {
		out.Tier = t.Tier
	}
	if t.Effort != "" {
		out.Effort = t.Effort
	}
	if t.PromoteAfter != 0 {
		out.PromoteAfter = t.PromoteAfter
	}
	if t.Timeout != "" {
		out.Timeout = t.Timeout
	}
	return out
}

// AnswerTimeout is how long the agent may take for one answer.
func (m Model) AnswerTimeout() (time.Duration, error) {
	if m.Timeout == "" {
		return 3 * time.Minute, nil
	}
	d, err := time.ParseDuration(m.Timeout)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("model.timeout %q is not a duration like 10m", m.Timeout)
	}
	return d, nil
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

// ProjectConfig is the part of .workline/config.yaml the engine reads today.
type ProjectConfig struct {
	AI    string `yaml:"ai"`    // default agent for this project: none, claude...
	Forge string `yaml:"forge"` // github, gitlab, or none
	Roles map[string]struct {
		Settings map[string]any    `yaml:"settings"`
		Enforce  map[string]string `yaml:"enforce"`
	} `yaml:"roles"`
}

// LoadProjectConfig reads <repo>/.workline/config.yaml. A missing file is an
// empty config; an unreadable or invalid one is an error.
func LoadProjectConfig(repo string) (*ProjectConfig, error) {
	var c ProjectConfig
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if os.IsNotExist(err) {
		return &c, nil
	}
	if err != nil {
		return nil, err
	}
	if err := CheckConfig(data); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, &ConfigError{err.Error()}
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

// Writes returns the path patterns the role may write, with `$settings.<name>`
// replaced by the setting's value (a string or a list of strings).
func (r *Role) Writes(settings map[string]any) []string {
	var out []string
	for _, w := range r.Duties.Writes {
		name, ok := strings.CutPrefix(w, "$settings.")
		if !ok {
			out = append(out, w)
			continue
		}
		switch v := settings[name].(type) {
		case string:
			out = append(out, v)
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out
}
