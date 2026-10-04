package reviewer

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/verdict"
)

// commentStyles are the line-comment openers of a file, by its extension;
// a file not listed is read with both `//` and `#`.
var commentStyles = map[string][]string{
	"slash": {"//", "/*", "*"},
	"hash":  {"#"},
	"dash":  {"--"},
}

var styleOf = map[string]string{
	".go": "slash", ".js": "slash", ".mjs": "slash", ".cjs": "slash", ".ts": "slash", ".tsx": "slash", ".jsx": "slash",
	".java": "slash", ".kt": "slash", ".kts": "slash", ".scala": "slash", ".swift": "slash", ".c": "slash", ".h": "slash",
	".cc": "slash", ".cpp": "slash", ".hpp": "slash", ".cs": "slash", ".rs": "slash", ".php": "slash", ".dart": "slash",
	".ino": "slash", ".proto": "slash",
	".py": "hash", ".sh": "hash", ".bash": "hash", ".rb": "hash", ".pl": "hash", ".r": "hash", ".yaml": "hash",
	".yml": "hash", ".toml": "hash", ".cfg": "hash", ".conf": "hash", ".mk": "hash", ".ps1": "hash",
	".sql": "dash", ".lua": "dash", ".hs": "dash",
}

// comment returns the text of the comment a line holds, whether it holds
// one, and whether the comment is the whole line rather than what follows
// code after `//` or ` #`.
func comment(file, line string) (text string, ok, whole bool) {
	openers := []string{"//", "#"}
	style := styleOf[strings.ToLower(path.Ext(file))]
	if base := path.Base(file); base == "Makefile" || base == "Dockerfile" {
		style = "hash"
	}
	if style != "" {
		openers = commentStyles[style]
	}
	t := strings.TrimSpace(line)
	for _, o := range openers {
		if strings.HasPrefix(t, o) {
			if o == "*" && strings.HasPrefix(t, "*/") {
				return "", false, false
			}
			return strings.TrimSpace(strings.TrimPrefix(t, o)), true, true
		}
	}
	// A comment after code: its opener outside any string.
	for _, o := range openers {
		if o == "*" || o == "/*" {
			continue
		}
		if i := strings.Index(line, " "+o+" "); i > 0 && strings.Count(line[:i], `"`)%2 == 0 && strings.Count(line[:i], "'")%2 == 0 {
			return strings.TrimSpace(line[i+len(o)+1:]), true, false
		}
	}
	return "", false, false
}

// mechanical runs the rules no judgement is needed for on the comments a
// change adds: a bug's story, a code internal to the project, a block too
// long. A line the change did not add is not looked at.
func mechanical(c Change, s Settings, codes committer.Settings) ([]verdict.Finding, error) {
	var story []*regexp.Regexp
	for _, w := range s.StoryWords {
		re, err := regexp.Compile("(?i)" + w)
		if err != nil {
			return nil, fmt.Errorf("story-words %q: %w", w, err)
		}
		story = append(story, re)
	}
	files := make([]string, 0, len(c.Added))
	for f := range c.Added {
		files = append(files, f)
	}
	sort.Strings(files)
	var out []verdict.Finding
	for _, f := range files {
		blockStart, blockLen, last := 0, 0, -1
		flush := func() {
			if s.CommentBlockMax > 0 && blockLen > s.CommentBlockMax {
				out = append(out, verdict.Finding{Rule: "long-comment", Level: "warn", Where: fmt.Sprintf("%s:%d", f, blockStart),
					Message: fmt.Sprintf("a comment of %d lines; the limit is %d: say what the code does in a few lines, and put the reasons in the docs", blockLen, s.CommentBlockMax)})
			}
			blockLen = 0
		}
		for _, l := range c.Added[f] {
			text, ok, whole := comment(f, l.Text)
			if !whole { // a block is whole-line comments, one after the other
				flush()
				last = -1
			} else {
				if last != l.At-1 {
					flush()
				}
				if blockLen == 0 {
					blockStart = l.At
				}
				blockLen++
				last = l.At
			}
			if !ok {
				continue
			}
			where := fmt.Sprintf("%s:%d", f, l.At)
			for _, re := range story {
				if m := re.FindString(text); m != "" {
					out = append(out, verdict.Finding{Rule: "bug-story", Where: where,
						Message: fmt.Sprintf("the comment tells the code's history (%q): say what the code does now; how it was, and the bug, belong in the commit message", m)})
					break
				}
			}
			found, err := committer.InternalCodes(text, codes)
			if err != nil {
				return nil, err
			}
			for _, code := range found {
				out = append(out, verdict.Finding{Rule: "internal-code", Where: where,
					Message: fmt.Sprintf("%q means nothing to a reader of the code: say what it refers to", code)})
			}
		}
		flush()
	}
	return out, nil
}
