package tester

// A tool a test needs, missing where the tests ran (#760): the developer's
// proof on a runner without gitleaks turned ten of the committer's cases
// red, a red that named no tool. The test command's output says it the
// way the shell, Go's exec or the tool's own caller does; the proof names
// it, so a red that comes from the environment reads as one. The shell
// says the same of a function the code lacks: a name not found, a tool or
// not.

import (
	"regexp"
	"slices"
)

var missingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`exec: "([^"\s]+)": executable file not found in \$PATH`), // Go
	regexp.MustCompile(`(?m)(?:^|\s)([\w.+-]+): command not found`),              // bash
	regexp.MustCompile(`(?m)(?:^|\s)\S*: (?:line )?\d+: ([\w.+-]+): not found`),  // dash: "x.sh: 1: name: not found"
	regexp.MustCompile(`(?m)(?:^|[\s"'(])([\w.+-]+) is not installed`),           // workline's own: "gitleaks is not installed"
}

// MissingTools are the names a test command's output says were not
// found, each once, sorted: none when it says none.
func MissingTools(output string) []string {
	var out []string
	for _, re := range missingPatterns {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			out = append(out, m[1])
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
