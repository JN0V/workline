package tester

import "sort"

// Merge proposes folding the test Narrow, whose covered lines all lie in
// Wide's, into Wide.
type Merge struct {
	Wide   string
	Narrow string
}

// MergeCandidates proposes each test whose covered lines are a subset of
// another test's. It only proposes and never changes cov. Tests with equal
// coverage propose the later name into the earlier one, once.
func MergeCandidates(cov map[string]map[string]bool) []Merge {
	names := make([]string, 0, len(cov))
	for n := range cov {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []Merge
	for _, n := range names {
		if len(cov[n]) == 0 {
			continue
		}
		for _, w := range names {
			if w == n || !subset(cov[n], cov[w]) {
				continue
			}
			if len(cov[n]) == len(cov[w]) && w > n {
				continue
			}
			out = append(out, Merge{Wide: w, Narrow: n})
			break
		}
	}
	return out
}

func subset(a, b map[string]bool) bool {
	for l, on := range a {
		if on && !b[l] {
			return false
		}
	}
	return true
}
