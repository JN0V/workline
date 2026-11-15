package backlog

import (
	"path"
	"strings"

	"github.com/JN0V/workline/internal/work"
)

// CIFiles are the files and folders a CI runs from: GitHub's workflows and
// actions, GitLab's pipeline, workline's own settings and gates. Only a
// person changes them: the developer's bounds refuse a patch reaching one.
var CIFiles = []string{".workline", ".github/workflows", ".github/actions", ".gitlab-ci.yml", ".gitlab/ci"}

// Protected says whether p is a CI file, or under a CI folder.
func Protected(p string) bool {
	p = path.Clean(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
	for _, c := range CIFiles {
		if p == c || strings.HasPrefix(p, c+"/") {
			return true
		}
	}
	return false
}

// ScopePaths are the paths a spec's Scope names, in backticks or in plain
// words, read with no repository at hand.
func ScopePaths(body string) []string {
	text := work.ByMeaning(body)[work.Scope].Text
	var out []string
	add := func(p string) {
		p = path.Clean(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
		if p == "." || p == "" || strings.HasPrefix(p, "../") || path.IsAbs(p) {
			return
		}
		for _, q := range out {
			if q == p {
				return
			}
		}
		out = append(out, p)
	}
	for _, w := range work.ScopeWords(text) {
		if strings.ContainsAny(w, "/.") {
			add(w)
		}
	}
	for _, m := range PathLike.FindAllString(work.Plain(text), -1) {
		add(work.LineOff(RepoPath(m)))
	}
	return out
}

// PersonOnly says whether a spec's Scope names paths and every one is a CI
// file: nothing the developer may change.
func PersonOnly(body string) bool {
	paths := ScopePaths(body)
	if len(paths) == 0 {
		return false
	}
	// A file with no slash nor dot (Makefile, Dockerfile) is no CI file, and
	// ScopePaths drops it: its presence means the Scope is not all CI.
	text := work.ByMeaning(body)[work.Scope].Text
	for _, w := range work.ScopeWords(text) {
		if !strings.ContainsAny(w, "/.") {
			return false
		}
	}
	// Any plain word left once the paths are taken out must be known prose:
	// a word that may name a file (Makefile, justfile, Taskfile) means the
	// Scope is not known to be all CI.
	for _, w := range strings.FieldsFunc(PathLike.ReplaceAllString(work.Plain(text), " "), func(r rune) bool {
		return r != '_' && r != '-' && (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
	}) {
		if !proseWords[strings.ToLower(w)] && strings.Trim(w, "0123456789") != "" {
			return false
		}
	}
	for _, p := range paths {
		if !Protected(p) {
			return false
		}
	}
	return true
}

// proseWords are words a Scope's sentence is made of, none a file.
var proseWords = map[string]bool{"the": true, "and": true, "only": true, "a": true, "an": true, "in": true, "on": true, "of": true, "to": true, "for": true, "its": true, "this": true, "that": true, "these": true, "those": true, "ci": true, "github": true, "gitlab": true, "workflow": true, "workflows": true, "scope": true, "file": true, "files": true, "change": true, "step": true, "is": true, "are": true, "names": true, "named": true, "one": true, "pipeline": true, "job": true, "jobs": true, "action": true, "actions": true, "yaml": true, "yml": true, "line": true, "lines": true, "see": true, "with": true, "by": true, "at": true, "as": true}
