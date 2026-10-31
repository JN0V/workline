package work

import (
	"path"
	"regexp"
	"strings"
)

var (
	// scopeWord is a word in backticks: what a Scope names.
	scopeWord = regexp.MustCompile("`([^`\\s]+)`")
	// lineRef is a line or a span of lines after a file: `parts.go:943`,
	// `x.go:12-30`.
	lineRef = regexp.MustCompile(`:\d+(-\d+)?$`)
)

// ScopeWords are the words a Scope's text writes in backticks, each with
// a trailing line or span of lines taken off: `parts.go:943` names
// parts.go, as the product owner's writing rules ask a Scope to write it.
func ScopeWords(scope string) []string {
	var out []string
	for _, m := range scopeWord.FindAllStringSubmatch(scope, -1) {
		out = append(out, lineRef.ReplaceAllString(m[1], ""))
	}
	return out
}

// InRepo is the file of files a Scope's word names: the word itself,
// cleaned, or else the one file whose path ends with it, so `parts.go`
// or `documentalist/parts.go` names internal/builtin/documentalist/parts.go
// when no other file ends so. "" when none does, or several.
func InRepo(word string, files []string) string {
	p := path.Clean(strings.ReplaceAll(word, `\`, "/"))
	if p == "." || strings.HasPrefix(p, "../") || path.IsAbs(p) {
		return ""
	}
	found := ""
	for _, f := range files {
		switch {
		case f == p:
			return f
		case strings.HasSuffix(f, "/"+p):
			if found != "" {
				found = "\x00" // several: none named
			} else {
				found = f
			}
		}
	}
	if found == "\x00" {
		return ""
	}
	return found
}
