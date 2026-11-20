package tester

import "sort"

// Manifest maps each module to the modules it depends on, as the build
// manifest says.
type Manifest map[string][]string

// SelectSets returns the modules whose tests run. mode is "local" (the
// changed modules only), "pr" (them and every module depending on them,
// directly or not) or "release" (all modules).
func SelectSets(changed []string, manifest Manifest, mode string) []string {
	picked := map[string]bool{}
	switch mode {
	case "release":
		for m := range manifest {
			picked[m] = true
		}
	default:
		for _, m := range changed {
			picked[m] = true
		}
		if mode == "pr" {
			for grew := true; grew; {
				grew = false
				for m, deps := range manifest {
					if picked[m] {
						continue
					}
					for _, d := range deps {
						if picked[d] {
							picked[m], grew = true, true
							break
						}
					}
				}
			}
		}
	}
	out := make([]string, 0, len(picked))
	for m := range picked {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
