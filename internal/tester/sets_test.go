package tester

import (
	"slices"
	"testing"
)

// sortedSets is SelectSets' answer, sorted: the order of the modules is
// not what a scenario says.
func sortedSets(changed []string, m Manifest, mode string) []string {
	got := slices.Clone(SelectSets(changed, m, mode))
	slices.Sort(got)
	return got
}

// One module changed: Given a repository with two modules and a change
// in one, When the tester runs locally, Then only that module's tests run.
func TestSelectSetsLocalChangedModuleOnly(t *testing.T) {
	m := Manifest{"a": {"b"}, "b": nil}
	got := sortedSets([]string{"b"}, m, "local")
	if !slices.Equal(got, []string{"b"}) {
		t.Errorf("local, change in b: %q, want only [b]", got)
	}
	got = sortedSets([]string{"a"}, m, "local")
	if !slices.Equal(got, []string{"a"}) {
		t.Errorf("local, change in a: %q, want only [a]", got)
	}
}

// Pull request: Given module A depends on module B in the build manifest,
// and a change in B, When CI runs on a pull request, Then all tests of B
// and of A run, and the tests of modules that do not depend on B do not.
func TestSelectSetsPullRequestAddsDependents(t *testing.T) {
	m := Manifest{"a": {"b"}, "b": nil, "d": nil}
	got := sortedSets([]string{"b"}, m, "pr")
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("pr, change in b: %q, want [a b] and not d", got)
	}
}

// Indirect dependency: Given module C depends on A, A depends on B, and
// a change in B, When CI runs on a pull request, Then the tests of C run
// too.
func TestSelectSetsPullRequestIndirectDependent(t *testing.T) {
	m := Manifest{"c": {"a"}, "a": {"b"}, "b": nil, "d": nil}
	got := sortedSets([]string{"b"}, m, "pr")
	if !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("pr, change in b: %q, want [a b c] and not d", got)
	}
}

// Release: Given the same repository, When CI runs on a release, Then
// all tests of all modules run.
func TestSelectSetsReleaseAllModules(t *testing.T) {
	m := Manifest{"c": {"a"}, "a": {"b"}, "b": nil, "d": nil}
	got := sortedSets([]string{"b"}, m, "release")
	if !slices.Equal(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("release: %q, want every module [a b c d]", got)
	}
}
