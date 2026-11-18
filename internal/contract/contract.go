// Package contract is how a role's built-in code joins the engine: the
// role registers its steps by name, and the engine calls them through the
// registry, never by importing the role (ADR-0070 §6 e). It imports no
// engine, line nor sample package.
package contract

import (
	"fmt"
	"slices"
	"sync"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// Step is a role's built-in pre or post: it reads and writes the run
// folder, and returns the exit code the engine reads from a step's file.
type Step func(runDir, repo string) int

// Role is what a role's built-in code offers: either step may be nil;
// Code, offered by the role whose code the engine merges.
type Role struct {
	Pre, Post Step
	Code      *Code
}

// Code is what the engine and the line ask of the role whose code they
// merge, beside its steps: the developer's.
type Code struct {
	// ProveMerge runs the project's tests whole on at, a merge of the base
	// into a request made by the engine: the proof that two changes green
	// apart are green together (#1370).
	ProveMerge func(rolesDir, repo, at string) devmark.Proof
	// Guarded says, for a file, the rule that keeps a change to it from
	// being trusted to its own run, the rules read at the commit at; ""
	// for any other file.
	Guarded func(repo, at string) func(string) string
	// RoundDue says whether a round of the role is due on mr, its open
	// request: what the line's reconciler wakes it for.
	RoundDue func(f forge.Forge, repo string, mr forge.MergeRequest) (bool, error)
	// Held are the files each claimed issue holds, by its open requests'
	// changes (changed) or its Scope's paths; Overlapping, the first held
	// issue whose files is's Scope overlaps, and the path.
	Held        func(issues []forge.Issue, mrs []forge.MergeRequest, claimLabel string, letGo map[int]bool, files []string, changed Diffs) map[int][]string
	Overlapping func(is forge.Issue, held map[int][]string, files []string) (int, string)
	// Changed reads open requests' changed files with git in repo; RepoFiles,
	// the files the repository's index holds.
	Changed   func(repo string, keeps bool) Diffs
	RepoFiles func(repo string) []string
	// ClaimHours: the hours a claim with no pull request is kept, read
	// from the role's settings.
	ClaimHours func(settings map[string]any) int
}

// Registry holds roles by name.
type Registry struct {
	mu    sync.RWMutex
	roles map[string]Role
}

// NewRegistry returns an empty registry: a test fills its own with
// fixture roles, apart from the shipped ones.
func NewRegistry() *Registry { return &Registry{roles: map[string]Role{}} }

// Join adds a role; a name joined twice, or a second role offering
// Code, is a programming error.
func (r *Registry) Join(name string, role Role) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.roles[name]; ok {
		panic(fmt.Sprintf("contract: the role %q joined twice", name))
	}
	if role.Code != nil {
		for n, other := range r.roles {
			if other.Code != nil {
				panic(fmt.Sprintf("contract: the role %q offers the code the engine merges, as %q does", name, n))
			}
		}
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

// Code is what the role whose code the engine merges offers, and that
// role's name, one at most (Join); nil when none joined offers it.
func (r *Registry) Code() (*Code, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, n := range r.names() {
		if c := r.roles[n].Code; c != nil {
			return c, n
		}
	}
	return nil, ""
}

// Names lists the roles joined, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.names()
}

func (r *Registry) names() []string {
	names := make([]string, 0, len(r.roles))
	for n := range r.roles {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

var builtins = NewRegistry()

// UseForTest makes r the registry Builtins and BuiltinCode read, until
// restore puts the shipped one back: an in-package test of the engine,
// the line or the sample runs on fixture roles, never the real ones
// (docs/spec/layout.md, Tests).
func UseForTest(r *Registry) (restore func()) {
	old := builtins
	builtins = r
	return func() { builtins = old }
}

// Join adds a compiled-in role to the shipped registry, from its init().
func Join(name string, role Role) { builtins.Join(name, role) }

// Builtins is the registry the compiled-in roles join.
func Builtins() *Registry { return builtins }

// BuiltinCode is what the compiled-in role whose code the engine merges
// offers; none joined is a programming error: the binary links the
// developer (cmd/workline/builtins.go), a test its role or a fixture.
func BuiltinCode() *Code {
	c, _ := builtins.Code()
	if c == nil {
		panic("contract: no compiled-in role offers the code the engine merges")
	}
	return c
}

// Diffs gives the files an open merge request changes, its diff from the
// merge base with its base branch, and whether they were read (#1256).
// nil reads none: each held issue then holds its Scope's prediction.
type Diffs func(forge.MergeRequest) ([]string, bool)

// Backstop says why issue's runs reached its backstop, what a person
// does about it; "" when they did not. Set by the engine's command to
// the line's reading (line.BackstopSaid), which keeps the record and the
// normals; nil, never checked.
var Backstop func(f forge.Forge, repo string, issue int) (string, error)
