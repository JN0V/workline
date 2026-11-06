package tester

import (
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
)

// runnerFilters are the files a project's test runner reads which tests
// to run from, and the keys there that leave some out: a test written
// where one of them does not reach is never run (#1149).
var runnerFilters = map[string][]string{
	"platformio.ini":   {"test_filter", "test_ignore"},
	"pytest.ini":       {"testpaths", "python_files"},
	"tox.ini":          {"testpaths", "python_files"},
	"setup.cfg":        {"testpaths", "python_files"},
	"pyproject.toml":   {"testpaths", "python_files"},
	"package.json":     {"testMatch", "testRegex", "testPathIgnorePatterns", "roots"},
	"jest.config.js":   {"testMatch", "testRegex", "testPathIgnorePatterns", "roots"},
	"jest.config.ts":   {"testMatch", "testRegex", "testPathIgnorePatterns", "roots"},
	"jest.config.cjs":  {"testMatch", "testRegex", "testPathIgnorePatterns", "roots"},
	"jest.config.mjs":  {"testMatch", "testRegex", "testPathIgnorePatterns", "roots"},
	"jest.config.json": {"testMatch", "testRegex", "testPathIgnorePatterns", "roots"},
}

// notRun says why a report names no test of the files the tester wrote:
// they were written, and the project's test command did not run them.
// The filters found on the base in the folders above them are named, the
// line that sets each, so the agent writes them where the command runs
// them (trial #55: a PlatformIO test_filter listing its suites by name).
func notRun(repo, base string, files []string, command string) string {
	msg := fmt.Sprintf("the tests were written (%s), but the report names no test of them: the project's test command (`%s`) did not run them, "+
		"so nothing is proven red", strings.Join(files, ", "), command)
	if found := filtersAbove(repo, base, files); len(found) > 0 {
		msg += "; it may leave them out by " + strings.Join(found, ", ")
	}
	return msg + ". Write them where that command runs them, or say in a note why no test it runs can check the scenario"
}

// filtersAbove are the runner filters set on the base in a file's folder
// or above it, each as "`file`'s `key` (`its line`)", once.
func filtersAbove(repo, base string, files []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range files {
		for dir := path.Dir(f); ; dir = path.Dir(dir) {
			for _, name := range slices.Sorted(maps.Keys(runnerFilters)) {
				keys := runnerFilters[name]
				p := path.Join(dir, name)
				if seen[p] {
					continue
				}
				seen[p] = true
				text, err := git(repo, "show", base+":"+p)
				if err != nil {
					continue
				}
				for _, line := range strings.Split(text, "\n") {
					l := strings.TrimSpace(line)
					for _, k := range keys {
						if m := filterLine.FindStringSubmatch(l); m != nil && m[1] == k {
							out = append(out, fmt.Sprintf("`%s`'s `%s` (`%s`)", p, k, clip(l, 200)))
						}
					}
				}
			}
			if dir == "." || dir == "/" {
				break
			}
		}
	}
	return out
}

// filterLine is a line setting a key, its whole name: `key = …`,
// `key: …`, `"key": …`.
var filterLine = regexp.MustCompile(`^"?([A-Za-z_]+)"?\s*[=:]`)

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
