package linters

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ranBy are the files a project runs its tools from, as globs from its
// root: CI, hooks, task runners. A YAML one is read as YAML, the values
// of its command keys only; another is read as shell.
var ranBy = []string{
	".github/workflows/*.yml", ".github/workflows/*.yaml",
	".github/actions/*/action.yml", ".github/actions/*/action.yaml",
	".gitlab-ci.yml", ".gitlab/ci/*.yml", ".gitlab/ci/*.yaml",
	".forgejo/workflows/*.yml", ".forgejo/workflows/*.yaml",
	".gitea/workflows/*.yml", ".gitea/workflows/*.yaml",
	".pre-commit-config.yaml", "lefthook.yml", ".lefthook.yml", ".husky/*",
	"Makefile", "makefile", "GNUmakefile", "justfile", "Justfile", "Taskfile.yml", "Taskfile.yaml",
}

// commandKeys are the keys whose values are commands: GitHub's, Forgejo's
// and Gitea's steps (run, uses), GitLab's jobs, lefthook's and Taskfile's.
// A step's name, a description, a `with:` are not.
var commandKeys = set("run", "uses", "script", "before_script", "after_script", "entry", "cmd", "cmds", "command", "commands")

// hookKeys are pre-commit's: a hook's id is the tool it runs.
var hookKeys = set("id", "entry")

// running says, for each tool the project runs, where it was found:
// the files above, package.json's scripts, and the gate checks of
// .workline/config.yaml, said first.
func running(repo string, k *known) map[string][]string {
	type text struct {
		where string
		runs  []string
	}
	var texts []text
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
			for _, n := range names {
				var runs []string
				for _, c := range cfg.Gates[n].Checks {
					runs = append(runs, commands(c.Run)...)
				}
				texts = append(texts, text{".workline/config.yaml, gate " + n, runs})
			}
		}
	}
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
			texts = append(texts, text{filepath.ToSlash(rel), commandsOf(rel, data)})
		}
	}
	if data, err := os.ReadFile(filepath.Join(repo, "package.json")); err == nil {
		var p struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(data, &p) == nil {
			var runs []string
			for _, s := range p.Scripts {
				runs = append(runs, commands(s)...)
			}
			sort.Strings(runs)
			texts = append(texts, text{"package.json", runs})
		}
	}
	found := map[string][]string{}
	for _, x := range k.tools {
		for _, t := range texts {
			body := strings.Join(t.runs, "\n")
			for _, re := range k.seen[x.Name] {
				if re.MatchString(body) {
					found[x.Name] = append(found[x.Name], t.where)
					break
				}
			}
		}
	}
	return found
}

// commandsOf are the commands a file runs: a YAML file's command values,
// each read as shell; another file, or YAML that does not parse, as
// shell whole.
func commandsOf(name string, data []byte) []string {
	ext := filepath.Ext(name)
	if ext != ".yml" && ext != ".yaml" {
		return commands(string(data))
	}
	keys := commandKeys
	if filepath.Base(name) == ".pre-commit-config.yaml" {
		keys = hookKeys
	}
	values, err := yamlValues(data, keys)
	if err != nil {
		return commands(string(data))
	}
	var out []string
	for _, v := range values {
		out = append(out, commands(v)...)
	}
	return out
}

// yamlValues are the scalars under the keys, in every document: block,
// quoted and flow styles alike, anchors followed.
func yamlValues(data []byte, keys map[string]bool) ([]string, error) {
	var out []string
	var walk func(n *yaml.Node, take bool)
	walk = func(n *yaml.Node, take bool) {
		switch n.Kind {
		case yaml.DocumentNode, yaml.SequenceNode:
			for _, c := range n.Content {
				walk(c, take)
			}
		case yaml.MappingNode:
			for i := 0; i+1 < len(n.Content); i += 2 {
				walk(n.Content[i+1], keys[n.Content[i].Value])
			}
		case yaml.ScalarNode:
			if take {
				out = append(out, n.Value)
			}
		case yaml.AliasNode:
			if n.Alias != nil {
				walk(n.Alias, take)
			}
		}
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		walk(&doc, false)
	}
}
