package engine

import "testing"

// A merge queue group's branch, as GitHub names it, gives the pull request
// it ends with; anything else names none, never a guess.
func TestGroupOf(t *testing.T) {
	for _, c := range []struct {
		ref string
		n   int
	}{
		{"gh-readonly-queue/main/pr-12-7f122f4acd601f5afc1a76b31fe0640ec2174b39", 12},
		{"gh-readonly-queue/release/pr-x/pr-7-93389eb", 7}, // a base holding "/pr-"
		{"gh-readonly-queue/main/pr-7x-93389eb", 0},
		{"gh-readonly-queue/main/pr-0-93389eb", 0},
		{"gh-readonly-queue/main/pr--3-93389eb", 0},
		{"gh-readonly-queue/main/pr-99999999999999999999-93389eb", 0},
		{"gh-readonly-queue/main/pr-7-", 0},
		{"gh-readonly-queue/main/pr-7", 0},
		{"feat/pr-7-93389eb", 0},
		{"", 0},
	} {
		n, ok := groupOf(c.ref)
		if n != c.n || ok != (c.n > 0) {
			t.Errorf("groupOf(%q) = %d, %v; want %d", c.ref, n, ok, c.n)
		}
	}
}
