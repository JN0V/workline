package role

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// FlatPrompt is a prompt file an override left at its old, flat path, in a
// role whose prompts have moved under prompts/.
type FlatPrompt struct {
	Old, New string // the file's path, and where it now goes
}

// Moved says whether a role's shipped prompts are under prompts/.
func Moved(roleDir string) bool {
	st, err := os.Stat(filepath.Join(roleDir, "prompts"))
	return err == nil && st.IsDir()
}

// FlatPrompts lists the prompt files an override folder holds at their old
// path (persona.md, policy.md, instruction*.md, knowledge/, skills/*/SKILL.md,
// or any file the role ships under prompts/), each with the path it now has.
// roleDir is the role's own folder; override, the folder of an override of it.
// Nothing for a role whose prompts are not under prompts/.
func FlatPrompts(roleDir, override string) []FlatPrompt {
	if !Moved(roleDir) {
		return nil
	}
	shipped := map[string]bool{}
	_ = filepath.WalkDir(filepath.Join(roleDir, "prompts"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if rel, err := filepath.Rel(filepath.Join(roleDir, "prompts"), p); err == nil {
				shipped[filepath.ToSlash(rel)] = true
			}
		}
		return nil
	})
	var out []FlatPrompt
	_ = filepath.WalkDir(override, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(override, p)
		if rerr != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "prompts" {
				return fs.SkipDir
			}
			return nil
		}
		if shipped[rel] || isPromptPath(rel) {
			out = append(out, FlatPrompt{Old: p, New: filepath.Join(override, "prompts", filepath.FromSlash(rel))})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Old < out[j].Old })
	return out
}

func isPromptPath(rel string) bool {
	switch {
	case !strings.Contains(rel, "/"):
		return rel == "persona.md" || rel == "policy.md" || (strings.HasPrefix(rel, "instruction") && strings.HasSuffix(rel, ".md"))
	case strings.HasPrefix(rel, "knowledge/"):
		return strings.HasSuffix(rel, ".md")
	case strings.HasPrefix(rel, "skills/"):
		return path.Base(rel) == "SKILL.md"
	}
	return false
}
