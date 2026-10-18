// Command gen writes trunk.json, the extract of trunk-io/plugins (MIT)
// internal/linters reads, regenerated, never edited (docs/languages.md):
//
//	go run ./gen -commit <sha> [-from <a checkout of that commit>]
//
// Left out: formatters, tools that read every file, deprecated linters.
package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// The table as written; internal/linters reads the same shape.
type table struct {
	Source  string  `json:"source"`
	Licence string  `json:"licence"`
	Types   []ftype `json:"types"`
	Tools   []tool  `json:"tools"`
}

type ftype struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions,omitempty"`
	Filenames  []string `json:"filenames,omitempty"`
	Shebangs   []string `json:"shebangs,omitempty"`
	Regexes    []string `json:"regexes,omitempty"`
	Inherit    []string `json:"inherit,omitempty"`
	Keys       []string `json:"keys,omitempty"` // top-level YAML keys a file must hold: CloudFormation's
}

type tool struct {
	Name    string              `json:"name"`
	Reads   map[string][]string `json:"reads"`   // kind -> file types
	Suggest string              `json:"suggest"` // files | config | never
	Configs []string            `json:"configs,omitempty"`
	Version string              `json:"version,omitempty"`
	SeenAs  []string            `json:"seen-as"`
	Run     string              `json:"run,omitempty"`
	OnFiles bool                `json:"on-files,omitempty"`
}

// What trunk's plugin.yaml files hold, of what is read.
type plugin struct {
	Tools struct {
		Definitions []struct {
			Name    string `yaml:"name"`
			Shims   []any  `yaml:"shims"`
			Version any    `yaml:"known_good_version"`
		} `yaml:"definitions"`
	} `yaml:"tools"`
	Lint struct {
		Files []struct {
			Name             string   `yaml:"name"`
			Extensions       []string `yaml:"extensions"`
			Filenames        []string `yaml:"filenames"`
			Shebangs         []string `yaml:"shebangs"`
			Regexes          []string `yaml:"regexes"`
			Inherit          []string `yaml:"inherit"`
			RequiredYAMLKeys []string `yaml:"required_yaml_keys"`
		} `yaml:"files"`
		Definitions []linter `yaml:"definitions"`
	} `yaml:"lint"`
}

type linter struct {
	Name       string    `yaml:"name"`
	Files      []string  `yaml:"files"`
	Tools      []string  `yaml:"tools"`
	Commands   []command `yaml:"commands"`
	SuggestIf  string    `yaml:"suggest_if"`
	Configs    []string  `yaml:"direct_configs"`
	Version    any       `yaml:"known_good_version"`
	Deprecated any       `yaml:"deprecated"`
}

type command struct {
	Files      []string `yaml:"files"`
	Run        string   `yaml:"run"`
	Target     string   `yaml:"target"`
	Formatter  bool     `yaml:"formatter"`
	IsSecurity bool     `yaml:"is_security"`
	Configs    []string `yaml:"direct_configs"`
	Enabled    *bool    `yaml:"enabled"`
}

// whole: the command trunk runs on one file each, or once at the root,
// cannot be cut from its definition into a gate check: written here,
// from each tool's own documentation.
var whole = map[string]string{
	"osv-scanner": "osv-scanner scan source -r .",
	"grype":       "grype dir:.",
	"trivy":       "trivy config .",
	"pinact":      "pinact run --check",
}

func main() {
	commit := flag.String("commit", "", "the trunk-io/plugins commit read")
	from := flag.String("from", "", "a checkout of that commit, instead of downloading it")
	out := flag.String("out", "trunk.json", "the file written")
	flag.Parse()
	if *commit == "" {
		fmt.Fprintln(os.Stderr, "gen: -commit is needed: the table names what it was read from")
		os.Exit(64)
	}
	files, err := read(*commit, *from)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	t, err := build(files)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	t.Source = "https://github.com/trunk-io/plugins/tree/" + *commit
	if err := os.WriteFile(*out, write(t), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	fmt.Printf("gen: %d file types, %d tools, from %s\n", len(t.Types), len(t.Tools), t.Source)
}

// write is the table as JSON, a type or a tool a line: a refresh's diff
// shows what changed, tool by tool.
func write(t *table) []byte {
	line := func(v any) string {
		data, _ := json.Marshal(v)
		return string(data)
	}
	var types, tools []string
	for _, x := range t.Types {
		types = append(types, "  "+line(x))
	}
	for _, x := range t.Tools {
		tools = append(tools, "  "+line(x))
	}
	return []byte("{\n" +
		` "source": ` + line(t.Source) + ",\n" +
		` "licence": ` + line(t.Licence) + ",\n" +
		" \"types\": [\n" + strings.Join(types, ",\n") + "\n ],\n" +
		" \"tools\": [\n" + strings.Join(tools, ",\n") + "\n ]\n}\n")
}

// read returns LICENSE and each linters/**/plugin.yaml, by path from the
// repository's root.
func read(commit, from string) (map[string][]byte, error) {
	files := map[string][]byte{}
	keep := func(p string) bool {
		return p == "LICENSE" || (strings.HasPrefix(p, "linters/") && path.Base(p) == "plugin.yaml")
	}
	if from != "" {
		err := filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(from, p)
			if rel = filepath.ToSlash(rel); keep(rel) {
				data, err := os.ReadFile(p)
				files[rel] = data
				return err
			}
			return nil
		})
		return files, err
	}
	resp, err := http.Get("https://codeload.github.com/trunk-io/plugins/tar.gz/" + commit)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trunk-io/plugins at %s: %s", commit, resp.Status)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		_, rel, _ := strings.Cut(h.Name, "/") // the archive's top folder is plugins-<commit>
		if h.Typeflag == tar.TypeReg && keep(rel) {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			files[rel] = data
		}
	}
}

func build(files map[string][]byte) (*table, error) {
	t := &table{Licence: strings.TrimSpace(string(files["LICENSE"]))}
	if !strings.HasPrefix(t.Licence, "MIT License") {
		return nil, fmt.Errorf("LICENSE is no longer MIT's: read it before borrowing again")
	}
	var plugins []plugin
	names := make([]string, 0, len(files))
	for p := range files {
		names = append(names, p)
	}
	sort.Strings(names)
	for _, p := range names {
		if p == "LICENSE" {
			continue
		}
		var pl plugin
		if err := yaml.Unmarshal(files[p], &pl); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		plugins = append(plugins, pl)
	}

	types := map[string]bool{}
	for _, pl := range plugins {
		for _, f := range pl.Lint.Files {
			if f.Name == "ALL" || types[f.Name] {
				continue
			}
			types[f.Name] = true
			t.Types = append(t.Types, ftype{Name: f.Name, Extensions: f.Extensions, Filenames: f.Filenames,
				Shebangs: f.Shebangs, Regexes: f.Regexes, Inherit: f.Inherit, Keys: f.RequiredYAMLKeys})
		}
	}
	for i := range t.Types { // an inherited type left out is dropped
		t.Types[i].Inherit = slices.DeleteFunc(t.Types[i].Inherit, func(n string) bool { return !types[n] })
	}
	sort.Slice(t.Types, func(i, j int) bool { return t.Types[i].Name < t.Types[j].Name })

	shims, versions := map[string][]string{}, map[string]string{}
	for _, pl := range plugins {
		for _, d := range pl.Tools.Definitions {
			for _, s := range d.Shims {
				if name, ok := s.(string); ok {
					shims[d.Name] = append(shims[d.Name], name)
				} else if m, ok := s.(map[string]any); ok {
					if name, ok := m["name"].(string); ok {
						shims[d.Name] = append(shims[d.Name], name)
					}
				}
			}
			versions[d.Name] = version(d.Version)
		}
	}
	for _, pl := range plugins {
		for _, l := range pl.Lint.Definitions {
			if l.Deprecated != nil {
				continue
			}
			x, ok := linterTool(l, types, shims, versions)
			if !ok {
				continue
			}
			// One entry a tool: golangci-lint's v1 and v2 definitions run
			// the same command; the one trunk suggests is kept.
			if i := slices.IndexFunc(t.Tools, func(y tool) bool { return y.Name == x.Name }); i >= 0 {
				if rank[x.Suggest] > rank[t.Tools[i].Suggest] {
					x.SeenAs = union(x.SeenAs, t.Tools[i].SeenAs)
					t.Tools[i] = x
				} else {
					t.Tools[i].SeenAs = union(t.Tools[i].SeenAs, x.SeenAs)
				}
				continue
			}
			t.Tools = append(t.Tools, x)
		}
	}
	sort.Slice(t.Tools, func(i, j int) bool { return t.Tools[i].Name < t.Tools[j].Name })
	return t, nil
}

var rank = map[string]int{"never": 0, "config": 1, "files": 2}

func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, s := range b {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func linterTool(l linter, types map[string]bool, shims map[string][]string, versions map[string]string) (tool, bool) {
	x := tool{Reads: map[string][]string{}, Configs: slices.Clone(l.Configs), Version: version(l.Version)}
	var own []string
	for _, n := range l.Tools {
		own = append(own, shims[n]...)
		if x.Version == "" {
			x.Version = versions[n]
		}
	}
	x.Name = l.Name
	for _, s := range own { // golangci-lint2 is run as golangci-lint
		if s != l.Name && strings.HasPrefix(l.Name, s) {
			x.Name = s
		}
	}
	switch l.SuggestIf {
	case "files_present":
		x.Suggest = "files"
	case "config_present":
		x.Suggest = "config"
	default:
		x.Suggest = "never"
	}
	var first *command
	for i, c := range l.Commands {
		if c.Formatter || (c.Enabled != nil && !*c.Enabled) {
			continue
		}
		files := c.Files
		if len(files) == 0 {
			files = l.Files
		}
		files = slices.DeleteFunc(slices.Clone(files), func(n string) bool { return !types[n] })
		if len(files) == 0 {
			continue
		}
		kind := "lint"
		if c.IsSecurity {
			kind = "dependencies"
			for _, f := range files {
				if !strings.Contains(f, "lockfile") {
					kind = "security"
				}
			}
		}
		for _, f := range files {
			if !slices.Contains(x.Reads[kind], f) {
				x.Reads[kind] = append(x.Reads[kind], f)
			}
		}
		for _, cf := range c.Configs {
			if !slices.Contains(x.Configs, cf) {
				x.Configs = append(x.Configs, cf)
			}
		}
		if first == nil {
			first = &l.Commands[i]
		}
	}
	if first == nil {
		return x, false
	}
	for k := range x.Reads {
		sort.Strings(x.Reads[k])
	}
	x.Run, x.OnFiles = gateRun(*first, x.Name, l.Name, own)
	if r, ok := whole[x.Name]; ok {
		x.Run, x.OnFiles = r, false
	}
	x.SeenAs = seenAs(x.Name, l.Name, own, x.Run)
	return x, true
}

var (
	leadingWord = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.+-]*$`)
	// fileFlag: a flag naming the files after it, not one setting the
	// output (pyright --outputjson, swiftlint --quiet).
	fileFlag = regexp.MustCompile(`^(-[a-z]|--(path|file|files))$`)
)

// gateRun cuts the command a gate check runs from trunk's: its words up
// to the first flag or variable, the flag naming the files kept; on the
// files when trunk runs it on each, else once at the root.
func gateRun(c command, name, linter string, shims []string) (string, bool) {
	words := strings.Fields(c.Run)
	var lead []string
	for _, w := range words {
		if strings.HasPrefix(w, "-") || strings.Contains(w, "${") || !leadingWord.MatchString(w) {
			break
		}
		lead = append(lead, w)
	}
	if len(lead) == 0 {
		return "", false
	}
	lead[0] = strings.TrimSuffix(lead[0], ".cmd") // trunk's shim on Windows
	known := lead[0] == name || lead[0] == linter || slices.Contains(shims, lead[0])
	if !known && len(lead) < 2 {
		return "", false // a wrapper trunk ships: python3 ${plugin}/…, pwsh -f …
	}
	onFiles := c.Target == "" && strings.Contains(c.Run, "${target}")
	if onFiles {
		if i := slices.Index(words, "${target}"); i == len(lead)+1 && fileFlag.MatchString(words[i-1]) {
			lead = append(lead, words[i-1]) // checkov -f, cfn-lint -t, buf lint --path
		}
	}
	return strings.Join(lead, " "), onFiles
}

// seenAs are the words a project runs the tool by: its names of three
// letters or more, and its command when it has two words (cargo clippy).
func seenAs(name, linter string, shims []string, run string) []string {
	var out []string
	add := func(s string) {
		if len(s) >= 3 && !strings.Contains(s, ".") && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	add(name)
	add(linter)
	for _, s := range shims {
		add(s)
	}
	if w := strings.Fields(run); len(w) >= 2 && !strings.HasPrefix(w[1], "-") && !strings.Contains(w[1], ".") && !strings.Contains(w[1], ":") {
		add(w[0] + " " + w[1])
	}
	return out
}

func version(v any) string {
	switch v := v.(type) {
	case nil, bool:
		return ""
	default:
		return fmt.Sprint(v)
	}
}
