// Package gitrange reads the range of commits a run is given: `base..head`,
// a single commit, or `head ^base…` when the commits stand on several
// commits already pushed — a branch that merged main (internal/hooks).
package gitrange

import "strings"

// Args are the range as git log takes it: one argument a word.
func Args(rng string) []string { return strings.Fields(rng) }

// Head is the commit the range ends at.
func Head(rng string) string {
	if _, head, ok := strings.Cut(rng, ".."); ok {
		return head
	}
	for _, w := range strings.Fields(rng) {
		if !strings.HasPrefix(w, "^") {
			return w
		}
	}
	return rng
}

// Base is the commit the range starts after, the first one left out; ""
// for a range of the whole history.
func Base(rng string) string {
	if base, _, ok := strings.Cut(rng, ".."); ok {
		return base
	}
	for _, w := range strings.Fields(rng) {
		if strings.HasPrefix(w, "^") {
			return strings.TrimPrefix(w, "^")
		}
	}
	return ""
}
