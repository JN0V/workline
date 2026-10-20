package gate

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
)

// change is what a merge request changes, against its merge base: the
// files, those renamed, the lines added and removed.
type change struct {
	repo  string
	base  string            // the merge base
	files map[string]bool   // changed: false when deleted
	moved map[string]string // renamed: the base's name -> the head's
	added []line
	gone  map[string]int // removed lines' texts, spaces left out, how many times
	tree  string         // the base checked out, once a check reruns there
}

type line struct {
	file string
	n    int
	text string
}

// changeSince reads the change from the merge base of since and HEAD to
// the working tree, as the checks see it.
func changeSince(repo, since string) (*change, error) {
	if _, err := git(repo, "rev-parse", "--verify", "--quiet", since+"^{commit}"); err != nil {
		return nil, fmt.Errorf("%s is not a commit here (misspelled, or not fetched), so the findings the change adds cannot be told", since)
	}
	mb, err := git(repo, "merge-base", since, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("the merge base of %s and HEAD is not known (a shallow clone?), so the findings the change adds cannot be told: %v", since, err)
	}
	c := &change{repo: repo, base: strings.TrimSpace(mb), files: map[string]bool{}, moved: map[string]string{}, gone: map[string]int{}}
	status, err := git(repo, "-c", "core.quotePath=false", "diff", "-M", "--name-status", "-z", "--no-relative", c.base, "--")
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimSuffix(status, "\x00"), "\x00")
	for i := 0; i+1 < len(fields); {
		st := fields[i]
		if st != "" && (st[0] == 'R' || st[0] == 'C') && i+2 < len(fields) {
			from, to := fields[i+1], fields[i+2]
			if st[0] == 'R' {
				c.moved[from] = to
				c.files[from] = false
			}
			c.files[to] = true
			i += 3
			continue
		}
		c.files[fields[i+1]] = st != "D"
		i += 2
	}
	patch, err := git(repo, "-c", "core.quotePath=false", "diff", "-M", "-U0", "--no-color", "--no-ext-diff", "--no-relative",
		"--src-prefix=a/", "--dst-prefix=b/", c.base, "--")
	if err != nil {
		return nil, err
	}
	c.readPatch(patch)
	return c, nil
}

// readPatch takes the lines a patch adds, where the head holds them, and
// those it removes.
func (c *change) readPatch(patch string) {
	file := ""
	old, now, n := 0, 0, 0
	s := bufio.NewScanner(strings.NewReader(patch))
	s.Buffer(make([]byte, 1<<20), 1<<26)
	for s.Scan() {
		t := s.Text()
		if old > 0 || now > 0 { // in a hunk
			switch {
			case strings.HasPrefix(t, "+"):
				c.added = append(c.added, line{file, n, t[1:]})
				n++
				now--
			case strings.HasPrefix(t, "-"):
				c.gone[squeeze(t[1:])]++
				old--
			case strings.HasPrefix(t, " "):
				n++
				old--
				now--
			}
			continue
		}
		switch {
		case strings.HasPrefix(t, "+++ "):
			file = strings.TrimPrefix(unquote(strings.TrimPrefix(t, "+++ ")), "b/")
		case strings.HasPrefix(t, "@@ "):
			// @@ -a[,b] +c[,d] @@
			parts := strings.Fields(t)
			if len(parts) < 3 {
				continue
			}
			_, old = span(parts[1])
			n, now = span(parts[2])
		}
	}
}

// span reads -a,b or +c,d: where the hunk starts, and how many lines.
func span(s string) (start, count int) {
	s = s[1:]
	a, b, ok := strings.Cut(s, ",")
	start, _ = strconv.Atoi(a)
	count = 1
	if ok {
		count, _ = strconv.Atoi(b)
	}
	return start, count
}

func unquote(s string) string {
	if strings.HasPrefix(s, `"`) {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	return s
}

// squeeze is a line without its spaces and tabs: a line moved, indented
// again, is the same line.
func squeeze(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\r' {
			return -1
		}
		return r
	}, s)
}

// checkout puts the merge base in a tree of its own, outside the
// repository, the head's config copied over it: the files the change
// touched that config says, the gate's tools' configs by their names, so
// both runs read the same config (ADR-0049, section 6).
func (c *change) checkout(config func(file string) bool) error {
	if c.tree != "" {
		return nil
	}
	dir, err := os.MkdirTemp("", "workline-base-")
	if err != nil {
		return err
	}
	if _, err := git(c.repo, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", dir, c.base); err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("the merge base %s could not be checked out: %v", short(c.base), err)
	}
	c.tree = dir
	for file, kept := range c.files {
		if !config(file) {
			continue
		}
		at := filepath.Join(dir, filepath.FromSlash(file))
		if !kept {
			os.Remove(at)
			continue
		}
		data, err := os.ReadFile(filepath.Join(c.repo, filepath.FromSlash(file)))
		if err != nil {
			return err
		}
		os.MkdirAll(filepath.Dir(at), 0o755)
		if err := os.WriteFile(at, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// cleanUp removes the base's tree.
func (c *change) cleanUp() {
	if c == nil || c.tree == "" {
		return
	}
	if _, err := git(c.repo, "worktree", "remove", "--force", c.tree); err != nil {
		os.RemoveAll(c.tree)
		git(c.repo, "worktree", "prune")
	}
	c.tree = ""
}

// configFor says whether a file is config a check's tools read: a file
// named as one of their configs, or matching the check's own `config`.
func configFor(checks []Check) func(string) bool {
	return func(file string) bool {
		for _, ch := range checks {
			for _, pattern := range ch.Config {
				if pathglob.Match(pattern, file) {
					return true
				}
			}
			for _, name := range ch.configNames() {
				if path.Base(file) == name {
					return true
				}
			}
		}
		return false
	}
}

func git(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
