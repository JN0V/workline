package linters

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// Ran names the tools a gate check's command runs, as the table names
// them: what a gate rerun at the merge base copies the configs of, and
// whose suppressions it counts (ADR-0049, section 6).
func Ran(command string) []string {
	k, err := load()
	if err != nil {
		return nil
	}
	body := strings.Join(commands(command), "\n")
	var out []string
	for _, x := range k.tools {
		for _, re := range k.seen[x.Name] {
			if re.MatchString(body) {
				out = append(out, x.Name)
				break
			}
		}
	}
	return out
}

// Configs are the names of the files a tool reads its config from, as
// trunk's table gives them.
func Configs(tool string) []string {
	k, err := load()
	if err != nil {
		return nil
	}
	for _, x := range k.tools {
		if x.Name == tool {
			return x.Configs
		}
	}
	return nil
}

// Suppression is a comment that hides a tool's finding from it, in the
// files of its language.
type Suppression struct {
	Name  string   // as it is written: //nolint
	Tools []string // the tools it silences, by the table's names
	Exts  []string // the files it is read in
	re    *regexp.Regexp
}

// suppressions are those ADR-0049 names, each in its comment syntax, so a
// doc or a string naming one is none.
var suppressions = []Suppression{
	{Name: "//nolint", Tools: []string{"golangci-lint"}, Exts: []string{".go"}, re: regexp.MustCompile(`//\s*nolint\b`)},
	{Name: "# noqa", Tools: []string{"ruff", "flake8"}, Exts: []string{".py", ".pyi"}, re: regexp.MustCompile(`#\s*(?i:noqa)\b`)},
	{Name: "eslint-disable", Tools: []string{"eslint"}, Exts: []string{".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts", ".vue", ".svelte"},
		re: regexp.MustCompile(`(//|/\*)\s*eslint-disable`)},
	{Name: "NOLINT", Tools: []string{"clang-tidy"}, Exts: []string{".c", ".h", ".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx", ".m", ".mm"},
		re: regexp.MustCompile(`(//|/\*)\s*NOLINT`)},
}

// Suppressed says the suppression a line of a file holds, for one of the
// tools given; false when it holds none.
func Suppressed(file, text string, tools []string) (Suppression, bool) {
	ext := strings.ToLower(path.Ext(file))
	for _, s := range suppressions {
		if slices.Contains(s.Exts, ext) && slices.ContainsFunc(s.Tools, func(t string) bool { return slices.Contains(tools, t) }) && s.re.MatchString(text) {
			return s, true
		}
	}
	return Suppression{}, false
}
