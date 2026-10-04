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
	c := Change{Added: map[string][]Line{}, Removed: map[string][]Line{}}
	if len(files) == 0 {
		return c, nil
	}
	out, err := git(repo, append([]string{"diff", "-U0", "--no-color", "--no-ext-diff", "--no-renames", from, to, "--"}, files...)...)
	if err != nil {
		return c, err
	}
	path, old, at := "", "", 0
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "--- "):
			old = strings.TrimPrefix(strings.TrimPrefix(l, "--- "), "a/")
		case strings.HasPrefix(l, "+++ "):
			path = strings.TrimPrefix(strings.TrimPrefix(l, "+++ "), "b/")
			if path == "/dev/null" {
				path = old // deleted: its removed lines are the file's
			}
		case hunk.MatchString(l):
			at, _ = strconv.Atoi(hunk.FindStringSubmatch(l)[1])
		case strings.HasPrefix(l, "+"):
			c.Added[path] = append(c.Added[path], Line{path, at, l[1:]})
			at++
		case strings.HasPrefix(l, "-"):
			c.Removed[path] = append(c.Removed[path], Line{path, at, l[1:]})
		}
	}
	return c, nil
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

// locate finds a quote in a file's lines: each line of the quote, spaces
// aside, held by consecutive lines of the file. It returns the line numbers
// of every place it is found, 1-based, each the first line of the place.
func locate(lines []string, quote string) []int {
	var want []string
	for _, q := range strings.Split(quote, "\n") {
		if n := norm(q); n != "" {
			want = append(want, n)
		}
	}
	if len(want) == 0 {
		return nil
	}
	var found []int
	for i := 0; i+len(want) <= len(lines); i++ {
		ok := true
		for k, w := range want {
			if !strings.Contains(norm(lines[i+k]), w) {
				ok = false
				break
			}
		}
		if ok {
			found = append(found, i+1)
		}
	}
	return found
}

// quoteLines counts the lines a quote spans, blank ones left out.
func quoteLines(quote string) int {
	n := 0
	for _, q := range strings.Split(quote, "\n") {
		if norm(q) != "" {
			n++
		}
	}
	return n
}
