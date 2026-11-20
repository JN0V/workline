package tester

import (
	"slices"
	"testing"
)

// sortedSets returns the sets SelectSets chose, in a fixed order.
func sortedSets(got []string) []string {
	out := slices.Clone(got)
	slices.Sort(out)
	return out
}

// One module changed: Given a repository with two modules and a change
// in one, When the tester runs locally, Then only that module's tests run.
func TestSelectSetsLocalOneModule(t *testing.T) {
	manifest := Manifest{"a": {}, "b": {}}
	got := sortedSets(SelectSets([]string{"a"}, manifest, "local"))
	if !slices.Equal(got, []string{"a"}) {
		t.Errorf("local, change in a: got %q, want [a]", got)
	}
}

// Pull request: Given module A depends on module B in the build manifest,
// and a change in B, When CI runs on a pull request, Then all tests of B
// and of A run, and the tests of modules that do not depend on B do not.
func TestSelectSetsPullRequestDependents(t *testing.T) {
	manifest := Manifest{"a": {"b"}, "b": {}, "d": {}}
	got := sortedSets(SelectSets([]string{"b"}, manifest, "pr"))
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("pr, change in b: got %q, want [a b]", got)
	}
	// The same change run locally stays on b alone: the dependents are
	// the pull request's.
	if local := sortedSets(SelectSets([]string{"b"}, manifest, "local")); !slices.Equal(local, []string{"b"}) {
		t.Errorf("local, change in b: got %q, want [b]", local)
	}
}

// Indirect dependency: Given module C depends on A, A depends on B, and a
// change in B, When CI runs on a pull request, Then the tests of C run too.
func TestSelectSetsPullRequestIndirect(t *testing.T) {
	manifest := Manifest{"a": {"b"}, "b": {}, "c": {"a"}, "d": {}}
	got := sortedSets(SelectSets([]string{"b"}, manifest, "pr"))
	if !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("pr, change in b: got %q, want [a b c]", got)
	}
}

// Release: Given the same repository, When CI runs on a release, Then all
// tests of all modules run.
func TestSelectSetsRelease(t *testing.T) {
	manifest := Manifest{"a": {"b"}, "b": {}, "c": {"a"}, "d": {}}
	got := sortedSets(SelectSets([]string{"b"}, manifest, "release"))
	if !slices.Equal(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("release: got %q, want [a b c d]", got)
	}
}
