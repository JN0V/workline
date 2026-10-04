package reviewer

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Line is a line a change added, or removed, in a file.
type Line struct {
	Path string `json:"path"`
	At   int    `json:"at"` // added: its line in the new file; removed: the new file's line where it was
	Text string `json:"text"`
}

// Change is what a range of commits did to the files: the lines it added and
// removed, by file.
type Change struct {
	Added   map[string][]Line `json:"added"`
	Removed map[string][]Line `json:"removed"`
}

var hunk = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// readChange reads `git diff -U0 from to -- files`, every line it adds and
// removes placed in the new file.
func readChange(repo, from, to string, files []string) (Change, error) {
	if len(files) == 0 {
		return Change{Added: map[string][]Line{}, Removed: map[string][]Line{}}, nil
	}
	out, err := git(repo, append([]string{"diff", "-U0", "--no-color", "--no-ext-diff", "--no-renames", from, to, "--"}, files...)...)
	if err != nil {
		return Change{}, err
	}
	return parseDiff(out), nil
}

// parseDiff reads a unified diff without context. A file's `---` and `+++`
// lines are read only in its header, before its first hunk: inside one, a
// line removed that read `-- x` is `--- x`, and is a line, not a file.
func parseDiff(out string) Change {
	c := Change{Added: map[string][]Line{}, Removed: map[string][]Line{}}
	path, old, at, header := "", "", 0, false
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			header = true
		case header && strings.HasPrefix(l, "--- "):
			old = strings.TrimPrefix(strings.TrimPrefix(l, "--- "), "a/")
		case header && strings.HasPrefix(l, "+++ "):
			path = strings.TrimPrefix(strings.TrimPrefix(l, "+++ "), "b/")
			if path == "/dev/null" {
				path = old // deleted: its removed lines are the file's
			}
		case hunk.MatchString(l):
			header = false
			at, _ = strconv.Atoi(hunk.FindStringSubmatch(l)[1])
		case header:
		case strings.HasPrefix(l, "+"):
			c.Added[path] = append(c.Added[path], Line{path, at, l[1:]})
			at++
		case strings.HasPrefix(l, "-"):
			c.Removed[path] = append(c.Removed[path], Line{path, at, l[1:]})
		}
	}
	return c
}

// addedAt says whether the change added line n of path.
func (c Change) addedAt(path string, n int) bool {
	for _, l := range c.Added[path] {
		if l.At == n {
			return true
		}
	}
	return false
}

func git(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.quotePath=off"}, args...)...)
	var errOut strings.Builder
	cmd.Stderr = &errOut
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(errOut.String()))
	}
	return string(out), nil
}

// fileAt is a file as a commit holds it; "" and false when it holds none.
func fileAt(repo, commit, path string) (string, bool) {
	out, err := git(repo, "show", commit+":"+path)
	return out, err == nil
}

// norm is a line as quotes are compared: spaces at its ends dropped, runs
// of spaces inside made one.
func norm(s string) string { return strings.Join(strings.Fields(s), " ") }

// Place is where a quote is found: its first and last lines, 1-based.
type Place struct{ From, To int }

// locate finds a quote in a file's lines, spaces and line breaks aside: an
// agent often quotes two lines as one, or one line as two. It returns every
// place it is found.
func locate(lines []string, quote string) []Place {
	want := norm(quote)
	if want == "" {
		return nil
	}
	var text strings.Builder
	var lineAt []int // the line each character of text comes from
	for i, l := range lines {
		n := norm(l)
		if n == "" {
			continue
		}
		if text.Len() > 0 {
			text.WriteByte(' ')
			lineAt = append(lineAt, i+1)
		}
		text.WriteString(n)
		for range len(n) {
			lineAt = append(lineAt, i+1)
		}
	}
	all := text.String()
	var found []Place
	for start := 0; ; {
		i := strings.Index(all[start:], want)
		if i < 0 {
			return found
		}
		at := start + i
		found = append(found, Place{lineAt[at], lineAt[at+len(want)-1]})
		start = at + 1
	}
}
