package steer

import (
	"path"
	"regexp"
	"strings"
)

// CodeownersPlaces are where the forges look for CODEOWNERS, in the order
// GitHub reads them: .github/, the root, docs/; then GitLab's .gitlab/.
// The engine reads the first found.
var CodeownersPlaces = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS", ".gitlab/CODEOWNERS"}

// Codeowners is a CODEOWNERS file, read as GitHub and GitLab read it.
type Codeowners struct {
	// Path is where it was read.
	Path string
	// Unread is the first line read past, "" none: an exclusion
	// (`!pattern`), never taken to unlist a path, or a section header
	// that cannot be read, its rules then a section of their own.
	Unread   string
	rules    []coRule
	optional map[int]bool
}

// coRule is a CODEOWNERS rule: its pattern, its section, whether it
// names an owner (its own, or its section's default ones).
type coRule struct {
	pattern string
	section int
	owned   bool
}

var sectionHeader = regexp.MustCompile(`^\^?\[[^\]]*\](\[\d+\])?\s*(.*)$`)

// ParseCodeowners reads a CODEOWNERS file's text, read at p.
func ParseCodeowners(p string, data []byte) *Codeowners {
	c := &Codeowners{Path: p, optional: map[int]bool{}}
	unread := func(why string) {
		if c.Unread == "" {
			c.Unread = why
		}
	}
	section, defaults := 0, false
	for l := range strings.Lines(string(data)) {
		l = strings.TrimSpace(l)
		switch {
		case l == "" || strings.HasPrefix(l, "#"):
			continue
		case strings.HasPrefix(l, "!"):
			unread("excludes paths (`!`)")
			continue
		case strings.HasPrefix(l, "[") || strings.HasPrefix(l, "^["):
			section++
			c.optional[section] = strings.HasPrefix(l, "^")
			m := sectionHeader.FindStringSubmatch(l)
			if m == nil {
				unread("holds a section header that cannot be read: " + l)
				defaults = false
				continue
			}
			defaults = strings.TrimSpace(m[2]) != ""
			continue
		}
		f := strings.Fields(l)
		c.rules = append(c.rules, coRule{pattern: f[0], section: section, owned: len(f) > 1 || defaults})
	}
	return c
}

// Owns says whether a file has an owner: in each section, the last rule
// that matches it decides. Required, an owner whose approval is
// required: not only in an optional section (`^[…]`).
func (c *Codeowners) Owns(file string) (owned, required bool) {
	return c.owns(file, func(pattern, f string) bool { return codeownersMatch(pattern, f) })
}

// Holds says whether a file has an owner, a rule naming a folder with
// no trailing `/` taking everything under it too, as GitHub reads it
// (gitignore's rules): the engine's guard reads a pattern the widest
// way a forge does, holding more, never less.
func (c *Codeowners) Holds(file string) bool {
	owned, _ := c.owns(file, func(pattern, f string) bool {
		for p := path.Clean(f); p != "." && p != "/"; p = path.Dir(p) {
			if codeownersMatch(pattern, p) {
				return true
			}
		}
		return false
	})
	return owned
}

func (c *Codeowners) owns(file string, match func(pattern, file string) bool) (owned, required bool) {
	last := map[int]bool{} // by section, the last matching rule's owner
	for _, r := range c.rules {
		if match(r.pattern, file) {
			last[r.section] = r.owned
		}
	}
	for s, o := range last {
		owned = owned || o
		required = required || o && !c.optional[s]
	}
	return owned, required
}

// codeownersMatch says whether a CODEOWNERS pattern takes the file: a
// leading `/` anchors it at the root, a trailing `/` takes a folder's
// files, a name without `/` matches at any depth, `*` and `**` glob.
func codeownersMatch(pattern, file string) bool {
	if pattern == "*" || pattern == "/**" || pattern == "**" {
		return true
	}
	anchored := strings.HasPrefix(pattern, "/")
	p := strings.TrimPrefix(pattern, "/")
	if strings.HasSuffix(p, "/") {
		p += "**"
	}
	if !anchored && !strings.Contains(strings.TrimSuffix(p, "/**"), "/") {
		p = "**/" + p
	}
	return globMatch(p, file)
}

// globMatch matches a path to a pattern whose `**` spans folders.
func globMatch(pattern, name string) bool {
	if !strings.Contains(pattern, "**") {
		ok, _ := path.Match(pattern, name)
		return ok
	}
	before, after, _ := strings.Cut(pattern, "**")
	if !strings.HasPrefix(name, before) {
		return false
	}
	rest := strings.TrimPrefix(name, before)
	after = strings.TrimPrefix(after, "/")
	if after == "" {
		return true
	}
	for i := 0; i <= len(rest); i++ {
		if i == 0 || rest[i-1] == '/' {
			if globMatch(after, rest[i:]) {
				return true
			}
		}
	}
	return false
}
