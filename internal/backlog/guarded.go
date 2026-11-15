package backlog

// A Scope of guarded files only (#1505): what CODEOWNERS lists never
// merges on the developer's edit (ADR-0068 §4: the guard refuses it), so
// an issue whose Scope names nothing else would spend the tester's and
// the developer's tokens on a change nobody on the line may write. The
// product owner keeps it in refine and asks a person to write those
// files; a Scope with other files goes on, the guarded ones said as left
// to a person.

import (
	"path"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/steer"
	"github.com/JN0V/workline/internal/work"
)

// GuardedSilent opens the product owner's word on an issue it keeps in
// refine for a Scope of guarded files only.
var GuardedSilent = "its " + work.Name(work.Scope) + " names only files the line may not write, which a person writes: "

// ScopeSplit is, of the files an issue's Scope names, the guarded ones
// (steer.Files at HEAD of repo) and the others. Files in backticks, or
// written plain when the repository holds them.
func ScopeSplit(repo string, is forge.Issue) (guarded, others []string, err error) {
	guard, err := steer.Files(repo, "HEAD")
	if err != nil {
		return nil, nil, err
	}
	text := work.ByMeaning(is.Body)[work.Scope].Text
	var paths []string
	add := func(p string) {
		p = path.Clean(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
		if p != "." && p != ".." && !strings.HasPrefix(p, "../") && !path.IsAbs(p) && !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	for _, w := range work.ScopeWords(text) {
		// A file the repository holds, or a path to create (a folder and
		// a file's extension): never a version or a domain name.
		// A name the guard lists counts too, held or not yet: `AGENTS.md`,
		// `CODEOWNERS` at the root.
		cw := path.Clean(strings.ReplaceAll(strings.TrimSpace(w), `\`, "/"))
		if (strings.ContainsAny(w, "/.") && (InCommit(repo, cw) || (strings.Contains(w, "/") && !strings.Contains(w, "://") && work.NewPath(w)))) || guard(cw) != "" {
			add(w)
		}
	}
	for _, m := range PathLike.FindAllString(work.Plain(text), -1) {
		if p := work.LineOff(RepoPath(m)); InCommit(repo, p) {
			add(p)
		}
	}
	for _, p := range paths {
		if guard(p) != "" {
			guarded = append(guarded, p)
		} else {
			others = append(others, p)
		}
	}
	return guarded, others, nil
}

// guardedScope is, for a ready of an issue not ready yet, the guarded
// files its Scope names when it names nothing else ("" none): held in
// refine. A mixed Scope goes on, its guarded files kept on c.
func (p *Plan) guardedScope(repo string, is forge.Issue, c *Proposal) (hold []string) {
	c.Guarded = nil
	if repo == "" || Readied(is) {
		return nil
	}
	guarded, others, err := ScopeSplit(repo, is)
	if err != nil {
		// The guard unread: nothing says the Scope is writable, held.
		return []string{"(the guarded files could not be read: " + err.Error() + ")"}
	}
	switch {
	case len(guarded) == 0:
		return nil
	case len(others) == 0:
		return guarded
	}
	c.Guarded = guarded
	return nil
}
