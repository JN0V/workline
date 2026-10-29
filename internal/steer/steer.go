// Package steer tells the files that steer agents: AGENTS.md, CLAUDE.md
// and their likes, a page one links to or imports, a file named a prompt,
// an instruction or a template. What the routine merge keeps for a
// person (ADR-0042) and what the developer may never touch (ADR-0054).
package steer

import (
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
)

// agentNames are the files agents read before anything, by name, wherever
// they are; a page one links to or imports (`@path`) is one agents are
// told to read.
var agentNames = []string{"agents.md", "claude.md", "gemini.md", "copilot-instructions.md"}

// steersByName are words in a file's name that make it one agents follow:
// a prompt, an instruction, a template.
var steersByName = []string{"prompt", "instruction", "template"}

var (
	mdLink    = regexp.MustCompile(`\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	refLink   = regexp.MustCompile(`(?m)^\s*\[[^\]]+\]:\s*<?(\S+?)>?(?:\s|$)`)
	importDoc = regexp.MustCompile(`(?m)(?:^|\s)@([\w./-]+)`)
)

// Files is, at tip, what says whether a page steers agents: an agent
// file itself, a page one links to or imports — a folder linked, all of it
// —, followed through the files they import, or a file named a prompt, an
// instruction or a template.
func Files(repo, tip string) (func(path string) bool, error) {
	out, err := git(repo, "ls-tree", "-r", "--name-only", tip)
	if err != nil {
		return nil, err
	}
	var queue []string
	for _, p := range strings.Split(out, "\n") {
		if slices.Contains(agentNames, strings.ToLower(path.Base(p))) {
			queue = append(queue, p)
		}
	}
	read := map[string]bool{}
	linked := map[string]bool{}
	for len(queue) > 0 && len(read) < 50 {
		p := queue[0]
		queue = queue[1:]
		if read[p] {
			continue
		}
		read[p] = true
		linked[p] = true
		text, ok := blob(repo, tip, p)
		if !ok {
			continue
		}
		dir := path.Dir(p)
		add := func(target string, imported bool) {
			if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
				return
			}
			target, _, _ = strings.Cut(target, "#")
			target = strings.TrimPrefix(target, "/")
			if target == "" {
				return
			}
			at := path.Clean(path.Join(dir, target))
			if strings.HasPrefix(at, "../") || at == ".." {
				return
			}
			linked[at] = true
			if imported {
				queue = append(queue, at)
			}
		}
		for _, m := range mdLink.FindAllStringSubmatch(text, -1) {
			add(m[1], false)
		}
		for _, m := range refLink.FindAllStringSubmatch(text, -1) {
			add(m[1], false)
		}
		for _, m := range importDoc.FindAllStringSubmatch(text, -1) {
			add(m[1], true)
		}
	}
	return func(p string) bool {
		name := strings.ToLower(path.Base(p))
		if slices.Contains(agentNames, name) || slices.ContainsFunc(steersByName, func(w string) bool { return strings.Contains(name, w) }) {
			return true
		}
		for l := range linked {
			if p == l || l == "." || strings.HasPrefix(p, l+"/") {
				return true
			}
		}
		return false
	}, nil
}

func git(repo string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	return string(out), err
}

func blob(repo, commit, p string) (string, bool) {
	out, err := exec.Command("git", "-C", repo, "cat-file", "blob", commit+":"+p).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}
