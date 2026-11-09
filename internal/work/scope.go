package work

import (
	"path"
	"regexp"
	"slices"
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

// newMark is a Scope's line that names files to create: `New:` at its
// start, bold or in a bullet as well (ADR-0065, change 2).
var newMark = regexp.MustCompile(`(?mi)^\s*(?:[-*]\s+)?[*_]*new[*_]*\s*:`)

// Guessed says whether a Scope's text marks its paths a guess — written
// before the code they name was read: a line starting `Guessed:` (#970).
// Such paths hold no other issue back.
func Guessed(scope string) bool {
	return guessMark.MatchString(scope)
}

// MarksNew says whether a Scope's text names a file to create, on a line
// starting `New:`.
func MarksNew(scope string) bool {
	return newMark.MatchString(scope)
}

// fileExt is a file's extension, a letter first: never an Action's
// version, `@v4.1`.
var fileExt = regexp.MustCompile(`^\.[A-Za-z][A-Za-z0-9]{0,7}$`)

// NewPath says whether a Scope's word reads as a path to create: a
// file's extension or a trailing slash (`pkg/__init__.py`, `src/`), no
// `@` (`actions/checkout@v4.1` is an Action).
func NewPath(w string) bool {
	return !strings.Contains(w, "@") && (strings.HasSuffix(w, "/") || fileExt.MatchString(path.Ext(w)))
}

// Unfound are the paths a Scope writes in backticks, a folder in them,
// that files hold nothing of — no file, no folder, no file a glob
// matches — and no line `New:` names (ADR-0065, change 2): the product
// owner's writing check. A word with no `/` is no path to check (a
// name, `os.Getenv`); one whose first folder files lack, and with no
// file's extension, names something else (`dorny/paths-filter`).
func Unfound(scope string, files []string) []string {
	var out []string
	for _, line := range strings.Split(scope, "\n") {
		if newMark.MatchString(line) {
			continue
		}
		for _, w := range ScopeWords(line) {
			p := path.Clean(strings.ReplaceAll(w, `\`, "/"))
			if !strings.Contains(p, "/") || strings.HasPrefix(p, "../") || path.IsAbs(p) || slices.Contains(out, p) {
				continue
			}
			first := strings.SplitN(p, "/", 2)[0]
			if !Pattern(p) && !NewPath(w) && !held(files, first) {
				continue
			}
			if len(Files(p, files)) == 0 && !held(files, p) {
				out = append(out, p)
			}
		}
	}
	return out
}

// held says whether files hold p, a file or a folder.
func held(files []string, p string) bool {
	return slices.ContainsFunc(files, func(f string) bool { return f == p || strings.HasPrefix(f, p+"/") })
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

// Plain is a Scope's text with its words in backticks taken out: what
// it says in plain words, where a backticked glob's folder,
// `core/session/*.kts`, is not read as the whole folder.
func Plain(scope string) string {
	return scopeWord.ReplaceAllString(scope, " ")
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
