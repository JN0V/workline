// Package steer tells the files that never merge by the policy alone
// (ADR-0042, amended by ADR-0068 §4) and that the developer may never
// touch (ADR-0054): the paths the project's CODEOWNERS lists, the files
// that define oversight and authority; with no CODEOWNERS, the files that
// steer agents, AGENTS.md, CLAUDE.md and their likes, and the pages one
// links to or imports.
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

// Agent says whether a path is an agent file, by its name: AGENTS.md,
// CLAUDE.md and their likes, wherever it is.
func Agent(p string) bool {
	return slices.Contains(agentNames, strings.ToLower(path.Base(p)))
}

var (
	mdLink    = regexp.MustCompile(`\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	refLink   = regexp.MustCompile(`(?m)^\s*\[[^\]]+\]:\s*<?(\S+?)>?(?:\s|$)`)
	importDoc = regexp.MustCompile(`(?m)(?:^|\s)@([\w./-]+)`)
)

// What a held path is, said after it: "it changes X, which …".
const (
	ListedWhy = "which CODEOWNERS lists"
	AgentsWhy = "which agents are told to read"
)

// Files is, at tip, what says whether a path never merges by the policy
// alone, and why ("" for any other path). The project's CODEOWNERS, the
// first found of CodeownersPlaces, is the list: a path it gives an owner,
// and a CODEOWNERS file itself, never anything else (ADR-0068 §4). With
// none, the files that steer agents: an agent file itself, a page one
// links to or imports — a folder linked, all of it —, followed through
// the files they import.
func Files(repo, tip string) (func(path string) string, error) {
	for _, p := range CodeownersPlaces {
		if text, ok := blob(repo, tip, p); ok {
			co := ParseCodeowners(p, []byte(text))
			return func(f string) string {
				if owned, _ := co.Owns(f); owned || slices.Contains(CodeownersPlaces, path.Clean(f)) {
					return ListedWhy
				}
				return ""
			}, nil
		}
	}
	out, err := git(repo, "ls-tree", "-r", "--name-only", tip)
	if err != nil {
		return nil, err
	}
	var queue []string
	for _, p := range strings.Split(out, "\n") {
		if Agent(p) {
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
	return func(p string) string {
		if Agent(p) {
			return AgentsWhy
		}
		for l := range linked {
			if p == l || l == "." || strings.HasPrefix(p, l+"/") {
				return AgentsWhy
			}
		}
		return ""
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
