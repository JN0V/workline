package tester

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// lastTests is the corpus's test count at the latest measure.
func lastTests(h History) int {
	if len(h.Measures) == 0 {
		return 0
	}
	return h.Measures[len(h.Measures)-1].Tests
}

// mergeText is the merge proposals for the curator's task: from the
// per-test coverage map (test-coverage-map). With no coverage tool, it says
// so and suggests adding one; it never guesses redundancy. Proposals only:
// the curator or a person decides.
func mergeText(repo string, s Settings) string {
	if strings.TrimSpace(s.TestCoverage) == "" {
		return "\n## Merge proposals\n\nNone: no per-test coverage (`test-coverage-map`). Suggestion: add a coverage tool that records the lines each test covers; redundancy is never guessed.\n"
	}
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(s.TestCoverage)))
	if err != nil {
		return fmt.Sprintf("\n## Merge proposals\n\nNone: %s is unreadable (%v). Suggestion: add a coverage tool that writes it.\n", s.TestCoverage, err)
	}
	var lines map[string][]string
	if err := json.Unmarshal(data, &lines); err != nil {
		return fmt.Sprintf("\n## Merge proposals\n\nNone: %s is not a JSON map of test to lines (%v).\n", s.TestCoverage, err)
	}
	cov := map[string]map[string]bool{}
	for t, ls := range lines {
		cov[t] = map[string]bool{}
		for _, l := range ls {
			cov[t][l] = true
		}
	}
	ms := MergeCandidates(cov)
	if len(ms) == 0 {
		return "\n## Merge proposals\n\nNone: no test's covered lines lie within another's.\n"
	}
	var b strings.Builder
	b.WriteString("\n## Merge proposals\n\nProposed only, never decided: the curator or a person decides. Never write a merge.\n\n")
	for _, m := range ms {
		fmt.Fprintf(&b, "- `%s` covers only lines `%s` covers: merge into it?\n", m.Narrow, m.Wide)
	}
	return b.String()
}

// SetsLines says the modules whose tests run, one a line. changed are
// files, each placed in the manifest's module whose name is its longest
// path prefix; manifestFile is a JSON map, module to the modules it
// depends on.
func SetsLines(manifestFile, mode string, changed []string) ([]string, error) {
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", manifestFile, err)
	}
	if mode != "local" && mode != "pr" && mode != "release" {
		return nil, fmt.Errorf("mode %q: local, pr or release", mode)
	}
	mods := map[string]bool{}
	for _, f := range changed {
		best := ""
		for mod := range m {
			if (f == mod || strings.HasPrefix(f, strings.TrimSuffix(mod, "/")+"/")) && len(mod) > len(best) {
				best = mod
			}
		}
		if best != "" {
			mods[best] = true
		}
	}
	var ch []string
	for k := range mods {
		ch = append(ch, k)
	}
	sort.Strings(ch)
	return SelectSets(ch, m, mode), nil
}
