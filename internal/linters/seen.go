package linters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ranBy are the files a project runs its tools from, as globs from its
// root: CI, hooks, task runners. Each one's text is read, comments left
// out; a tool named there runs.
var ranBy = []string{
	".github/workflows/*.yml", ".github/workflows/*.yaml",
	".github/actions/*/action.yml", ".github/actions/*/action.yaml",
	".gitlab-ci.yml", ".gitlab/ci/*.yml", ".gitlab/ci/*.yaml",
	".forgejo/workflows/*.yml", ".forgejo/workflows/*.yaml",
	".gitea/workflows/*.yml", ".gitea/workflows/*.yaml",
	".pre-commit-config.yaml", "lefthook.yml", ".lefthook.yml", ".husky/*",
	"Makefile", "makefile", "GNUmakefile", "justfile", "Justfile", "Taskfile.yml", "Taskfile.yaml",
}

var (
	// comment is a comment of YAML, a Makefile or shell: from a # at the
	// start of a line or after a space.
	comment = regexp.MustCompile(`(?m)(^|\s)#.*$`)
	// separator splits a line into its commands.
	separator = regexp.MustCompile(`&&|\|\||;|\|`)
	// lead is what comes before a command: a list's dash, a key
	// (`run:`, `script:`), make's @.
	lead = regexp.MustCompile(`^[\s@-]*(?:([A-Za-z_-]+):\s+)?[\s-]*`)
	// prose are the keys whose value is words, not a command.
	prose = map[string]bool{"name": true, "description": true, "title": true}
	// notARun is a command that names a tool without running it: an
	// install, a download, a file moved, a word printed.
	notARun = regexp.MustCompile(`^(?:sudo\s+)?(?:(?:pip3?|pipx|uv(?:\s+pip|\s+tool)?|npm|pnpm|yarn|bun|go|cargo|brew|apt-get|apt|gem|apk|dnf|yum|choco|scoop|mise|asdf|aqua|conda|poetry)\s+(?:install|add|get|use|i)\b|(?:curl|wget|install|chmod|mv|cp|ln|tar|unzip|rm|echo|printf)\b)`)
)

// runs is a text's commands that run something: comments, installs,
// downloads and printed words left out.
func runs(text string) string {
	var kept []string
	for _, line := range strings.Split(comment.ReplaceAllString(text, "$1"), "\n") {
		if m := lead.FindStringSubmatch(line); prose[m[1]] {
			continue // name: Lint with shellcheck
		}
		for _, c := range separator.Split(line, -1) {
			if c = lead.ReplaceAllString(c, ""); c != "" && !notARun.MatchString(c) {
				kept = append(kept, c)
			}
		}
	}
	return strings.Join(kept, "\n")
}

// running says, for each tool the project runs, where it was found:
// the files above, package.json's scripts, and the gate checks of
// .workline/config.yaml.
func running(repo string, k *known) map[string][]string {
	type text struct{ where, body string }
	var texts []text
	for _, g := range ranBy {
		matches, _ := filepath.Glob(filepath.Join(repo, g))
		sort.Strings(matches)
		for _, m := range matches {
			if st, err := os.Stat(m); err != nil || st.IsDir() {
				continue
			}
			data, err := os.ReadFile(m)
			if err != nil {
				continue
			}
			rel, _ := filepath.Rel(repo, m)
			texts = append(texts, text{filepath.ToSlash(rel), runs(string(data))})
		}
	}
	if data, err := os.ReadFile(filepath.Join(repo, "package.json")); err == nil {
		var p struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(data, &p) == nil {
			var lines []string
			for _, s := range p.Scripts {
				lines = append(lines, s)
			}
			sort.Strings(lines)
			texts = append(texts, text{"package.json", runs(strings.Join(lines, "\n"))})
		}
	}
	if data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml")); err == nil {
		var cfg struct {
			Gates map[string]struct {
				Checks []struct {
					Run string `yaml:"run"`
				} `yaml:"checks"`
			} `yaml:"gates"`
		}
		if yaml.Unmarshal(data, &cfg) == nil {
			names := make([]string, 0, len(cfg.Gates))
			for n := range cfg.Gates {
				names = append(names, n)
			}
			sort.Strings(names)
			var gates []text // said first: a gate check is what the line reads
			for _, n := range names {
				var lines []string
				for _, c := range cfg.Gates[n].Checks {
					lines = append(lines, c.Run)
				}
				gates = append(gates, text{".workline/config.yaml, gate " + n, runs(strings.Join(lines, "\n"))})
			}
			texts = append(gates, texts...)
		}
	}
	found := map[string][]string{}
	for _, x := range k.tools {
		for _, t := range texts {
			for _, re := range k.seen[x.Name] {
				if re.MatchString(t.body) {
					found[x.Name] = append(found[x.Name], t.where)
					break
				}
			}
		}
	}
	return found
}
