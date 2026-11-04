package work

import (
	"path"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
)

var (
	// scopeWord is a word in backticks: what a Scope names.
	scopeWord = regexp.MustCompile("`([^`\\s]+)`")
	// lineRef is a line or a span of lines after a file: `parts.go:943`,
	// `x.go:12-30`.
	lineRef = regexp.MustCompile(`:\d+(-\d+)?$`)
)

// guessMark is a Scope's line that says its paths are a guess: `Guessed:`
// at its start, bold or in a bullet as well (#970).
var guessMark = regexp.MustCompile(`(?mi)^\s*(?:[-*]\s+)?[*_]*guessed[*_]*\s*:`)

// codeUnread is a Scope's own word that no code was given when it was
// written, as Scopes said it before the mark (#998): "no `init` code
// was given in this run", "the code was not read".
var codeUnread = regexp.MustCompile("(?i)\\bno\\b[^.\\n]{0,40}\\bcode\\b[^.\\n]{0,20}\\b(?:was|were|has been|had been)\\s+(?:given|read|shown|seen)\\b" +
	"|\\bcode\\b[^.\\n]{0,30}\\b(?:was|were)\\s+not\\s+(?:given|read|shown|seen)\\b")

// Guessed says whether a Scope's text marks its paths a guess — written
// before the code they name was read: a line starting `Guessed:` (#970),
// or, from before that mark, its words saying no code was given (#998).
// Such paths hold no other issue back.
func Guessed(scope string) bool {
	return guessMark.MatchString(scope) || codeUnread.MatchString(scope)
}

// ScopeWords are the words a Scope's text writes in backticks, each with
// a trailing line or span of lines taken off: `parts.go:943` names
// parts.go, as the product owner's writing rules ask a Scope to write it.
func ScopeWords(scope string) []string {
	var out []string
	for _, m := range scopeWord.FindAllStringSubmatch(scope, -1) {
		out = append(out, LineOff(m[1]))
	}
	return out
}

// LineOff is a path with a trailing line or span of lines taken off:
// `parts.go:943` names parts.go.
func LineOff(p string) string {
	return lineRef.ReplaceAllString(p, "")
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

// Pattern says whether a Scope's word names its files by a pattern,
// `src/calc/*.sh`: `*` within a folder, `**` across folders, `?` and
// `[...]` as in a shell (pathglob).
func Pattern(word string) bool {
	return strings.ContainsAny(word, "*?[")
}

// Files are the files of files a Scope's word names: the one InRepo
// names, or, a pattern, every file it matches, in files' order (#801:
// `src/calc/*.sh` named, nothing given). None when it names no file: a
// folder is the caller's to read.
func Files(word string, files []string) []string {
	if f := InRepo(word, files); f != "" {
		return []string{f}
	}
	if !Pattern(word) {
		return nil
	}
	p := path.Clean(strings.ReplaceAll(word, `\`, "/"))
	if strings.HasPrefix(p, "../") || path.IsAbs(p) {
		return nil
	}
	var out []string
	for _, f := range files {
		if f != "" && pathglob.Match(p, f) {
			out = append(out, f)
		}
	}
	return out
}
