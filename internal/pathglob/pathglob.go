// Package pathglob matches repository paths against patterns where `*` stays
// within one folder and `**` spans any number of folders.
package pathglob

import (
	"path"
	"strings"
)

// Match reports whether a slash-separated path matches pattern.
func Match(pattern, name string) bool {
	return match(strings.Split(pattern, "/"), strings.Split(path.Clean(name), "/"))
}

func match(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			if len(pat) == 1 {
				return true
			}
			for i := range name {
				if match(pat[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, _ := path.Match(pat[0], name[0]); !ok {
			return false
		}
		pat, name = pat[1:], name[1:]
	}
	return len(name) == 0
}

// Any reports whether name matches one of the patterns.
func Any(patterns []string, name string) bool {
	for _, p := range patterns {
		if Match(p, name) {
			return true
		}
	}
	return false
}
