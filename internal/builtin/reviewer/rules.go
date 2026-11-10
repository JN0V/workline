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

// Hint is a comment a guess flags for a lens to judge (ADR-0065, Rule
// A): where it starts, the words that made the guess, the comment whole.
type Hint struct {
	Rule    string `json:"rule"`
	Where   string `json:"where"`
	Word    string `json:"word"`
	Comment string `json:"comment"`
}

// storyRule is the rule whose guess, a word of story-words in a comment,
// a lens judges when an agent is there (its `hints`); with none, it warns.
const storyRule = "bug-story"

// mechanical runs the rules no judgement is needed for on the comments a
// change adds: a code internal to the project, a block too long; and the
// guess of a bug's story, which only warns. A line the change did not add
// is not looked at.
func mechanical(c Change, s Settings, codes committer.Settings) ([]verdict.Finding, error) {
	out, _, err := rules(c, s, codes)
	return out, err
}

// rules is mechanical, with the comments the story guess flags as hints.
func rules(c Change, s Settings, codes committer.Settings) ([]verdict.Finding, []Hint, error) {
	var story []*regexp.Regexp
	for _, w := range s.StoryWords {
		re, err := regexp.Compile("(?i)" + w)
		if err != nil {
			return nil, nil, fmt.Errorf("story-words %q: %w", w, err)
		}
		story = append(story, re)
	}
	files := make([]string, 0, len(c.Added))
	for f := range c.Added {
		files = append(files, f)
	}
	sort.Strings(files)
	var out []verdict.Finding
	var hints []Hint
	// A story may run over the lines of one comment ("Fields used / to
	// drop"): it is looked for in the comment's lines joined.
	tells := func(f string, lines []Line) {
		joined, starts := "", []int{}
		for _, l := range lines {
			starts = append(starts, len(joined))
			joined += l.Text + " "
		}
		for _, re := range story {
			if loc := re.FindStringIndex(joined); loc != nil {
				at := lines[0].At
				for i, st := range starts {
					if st <= loc[0] {
						at = lines[i].At
					}
				}
				word := strings.TrimSpace(joined[loc[0]:loc[1]])
				where := fmt.Sprintf("%s:%d", f, at)
				out = append(out, verdict.Finding{Rule: storyRule, Level: "warn", Where: where,
					Message: fmt.Sprintf("the comment may tell the code's history (%q), a guess by a word list (story-words) that no agent judged: if it does, say what the code does now; how it was, and the bug, belong in the commit message", word)})
				hints = append(hints, Hint{Rule: storyRule, Where: where, Word: word, Comment: strings.TrimSpace(joined)})
				return
			}
		}
	}
	for _, f := range files {
		var block []Line // whole-line comments, one after the other
		flush := func() {
			if s.CommentBlockMax > 0 && len(block) > s.CommentBlockMax {
				out = append(out, verdict.Finding{Rule: "long-comment", Level: "warn", Where: fmt.Sprintf("%s:%d", f, block[0].At),
					Message: fmt.Sprintf("a comment of %d lines; the limit is %d: say what the code does in a few lines, and put the reasons in the docs", len(block), s.CommentBlockMax)})
			}
			if len(block) > 0 {
				tells(f, block)
			}
			block = nil
		}
		for _, l := range c.Added[f] {
			text, ok, whole := comment(f, l.Text)
			if !whole || len(block) > 0 && block[len(block)-1].At != l.At-1 {
				flush()
			}
			if !ok {
				continue
			}
			if whole {
				block = append(block, Line{f, l.At, text})
			} else {
				tells(f, []Line{{f, l.At, text}})
			}
			found, err := committer.InternalCodes(text, codes)
			if err != nil {
				return nil, nil, err
			}
			for _, code := range found {
				out = append(out, verdict.Finding{Rule: "internal-code", Where: fmt.Sprintf("%s:%d", f, l.At),
					Message: fmt.Sprintf("%q means nothing to a reader of the code: say what it refers to", code)})
			}
		}
		flush()
	}
	return out, hints, nil
}
