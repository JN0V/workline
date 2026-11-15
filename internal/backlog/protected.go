package backlog

import (
	"path"
	"strings"

	"github.com/JN0V/workline/internal/work"
)

// CIFiles are the files and folders a CI runs from: GitHub's workflows and
// actions, GitLab's pipeline, workline's own settings and gates. Only a
// person changes them (docs/spec/gates.md); the developer's bounds refuse
// a patch reaching one.
var CIFiles = []string{".workline", ".github/workflows", ".github/actions", ".gitlab-ci.yml", ".gitlab/ci"}

// IsCIFile says whether p is, or lies under, a file only a person changes.
func IsCIFile(p string) bool {
	p = path.Clean(strings.ReplaceAll(p, `\`, "/"))
	for _, r := range CIFiles {
		if p == r || strings.HasPrefix(p, r+"/") {
			return true
		}
	}
	return false
}

// ScopePaths are the paths a Scope's text names, in backticks or plain.
func ScopePaths(scope string) []string {
	var out []string
	for _, w := range work.ScopeWords(scope) {
		if p := RepoPath(w); p != "" {
			out = append(out, p)
		}
	}
	for _, m := range PathLike.FindAllString(work.Plain(scope), -1) {
		if p := RepoPath(m); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ScopeOnlyCI says whether a Scope names paths and every one is a file only
// a person changes: no tester nor developer run can do it.
func ScopeOnlyCI(scope string) bool {
	paths := ScopePaths(scope)
	for _, p := range paths {
		if !IsCIFile(p) {
			return false
		}
	}
	return len(paths) > 0
}
