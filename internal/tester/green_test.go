package tester

import (
	"slices"
	"testing"
)

func TestUncached(t *testing.T) {
	for _, c := range []struct {
		env  []string
		want string
	}{
		{[]string{"HOME=/h"}, "GOFLAGS=-count=1"},
		{[]string{"GOFLAGS=-mod=mod", "HOME=/h"}, "GOFLAGS=-mod=mod -count=1"},
		{[]string{"GOFLAGS=-count=2"}, "GOFLAGS=-count=2"},
	} {
		got := uncached(c.env)
		if n := len(slices.DeleteFunc(slices.Clone(got), func(e string) bool { return e != c.want })); n != 1 {
			t.Errorf("uncached(%q) = %q, want one %q", c.env, got, c.want)
		}
		if slices.Contains(c.env, "HOME=/h") && !slices.Contains(got, "HOME=/h") {
			t.Errorf("uncached(%q) lost HOME: %q", c.env, got)
		}
	}
}
