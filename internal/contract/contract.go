// Package contract is how a role's built-in code joins the engine: the
// role registers its steps by name, and the engine calls them through the
// registry, never by importing the role (ADR-0070 §6 e). It imports no
// engine, line nor sample package.
package contract

import (
	"fmt"
	"slices"
	"sync"
)

// Step is a role's built-in pre or post: it reads and writes the run
// folder, and returns the exit code the engine reads from a step's file.
type Step func(runDir, repo string) int

// Role is what a role's built-in code offers: either step may be nil.
type Role struct {
	Pre, Post Step
}

// Registry holds roles by name.
type Registry struct {
	mu    sync.RWMutex
	roles map[string]Role
}

// NewRegistry returns an empty registry: a test fills its own with
// fixture roles, apart from the shipped ones.
func NewRegistry() *Registry { return &Registry{roles: map[string]Role{}} }

// Join adds a role; a name joined twice is a programming error.
func (r *Registry) Join(name string, role Role) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.roles[name]; ok {
		panic(fmt.Sprintf("contract: the role %q joined twice", name))
	}
	r.roles[name] = role
}

// Step returns a role's built-in step, "pre" or "post"; false when the
// role has none.
func (r *Registry) Step(name, step string) (Step, bool) {
	r.mu.RLock()
	role, ok := r.roles[name]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	var s Step
	switch step {
	case "pre":
		s = role.Pre
	case "post":
		s = role.Post
	}
	return s, s != nil
}

// Names lists the roles joined, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.roles))
	for n := range r.roles {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

var builtins = NewRegistry()

// Join adds a compiled-in role to the shipped registry, from its init().
func Join(name string, role Role) { builtins.Join(name, role) }

// Builtins is the registry the compiled-in roles join.
func Builtins() *Registry { return builtins }
