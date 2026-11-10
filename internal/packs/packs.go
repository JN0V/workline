// Package packs reads the language packs (ADR-0065, "Language packs"): one
// YAML file an ecosystem, what workline knows of it as data, never engine
// code. Built-in packs are embedded in the binary; a project adds one, or
// replaces a built-in one whole, with a file of the same name under
// .workline/packs/. The project's config lists the packs it uses, written
// by `workline init` from their markers. A pack never decides a verdict:
// its fields are the defaults of a role's settings, a setting set wins.
package packs

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

// Generic is the built-in pack read where no pack is listed, and for a
// field no listed pack gives.
const Generic = "generic"

// Dir is where a project's own packs live, from its root.
const Dir = ".workline/packs"

// Home maps the folder of a test file onto the folder of its code: each
// `**` of Tests fills the next one of Code, the rest dropped.
type Home struct {
	Tests string `json:"tests" yaml:"tests"`
	Code  string `json:"code" yaml:"code"`
}

// Detect says when a pack fits a repository: a file of one of these
// names, at any depth outside test and aside folders; Extensions are the
// languages it knows, which `workline doctor` reads.
type Detect struct {
	Files      []string `yaml:"files"`
	Extensions []string `yaml:"extensions"`
}

// Pack is what workline knows of one ecosystem. Read strictly: a field
// workline does not read is refused, never ignored (principle 12).
type Pack struct {
	Name   string `yaml:"-"`
	Source string `yaml:"-"` // "built-in", or the project's file
	About  string `yaml:"about"`
	Detect Detect `yaml:"detect"`
	// TestPaths and TestHomes are the tester's test-paths and test-homes
	// when the project sets none: the one definition of a test file every
	// role reads (ADR-0065, change 3).
	TestPaths []string `yaml:"test-paths"`
	TestHomes []Home   `yaml:"test-homes"`
	// Aside are folders beside a project's own code, ranked after it by
	// the code picker: read from the generic pack only, until the code
	// picker's tables move to packs.
	Aside []string `yaml:"aside"`
	// TestCommand is a suggested test-command, `workline doctor` gives it
	// when the tester has none; never run unless the person sets it.
	TestCommand string `yaml:"test-command"`
}

//go:embed builtin/*.yaml
var builtinFS embed.FS

var builtin = sync.OnceValues(func() (map[string]Pack, error) {
	out := map[string]Pack{}
	entries, err := fs.ReadDir(builtinFS, "builtin")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		data, err := builtinFS.ReadFile("builtin/" + e.Name())
		if err != nil {
			return nil, err
		}
		p, err := parse(strings.TrimSuffix(e.Name(), ".yaml"), "built-in", data)
		if err != nil {
			return nil, err
		}
		out[p.Name] = p
	}
	return out, nil
})

// Builtin is the packs embedded in the binary, by name.
func Builtin() map[string]Pack {
	b, err := builtin()
	if err != nil {
		panic(fmt.Sprintf("packs: built-in: %v", err)) // embedded: a test reads them
	}
	return b
}

// parse reads one pack's file.
func parse(name, source string, data []byte) (Pack, error) {
	p := Pack{Name: name, Source: source}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil && !errors.Is(err, io.EOF) { // empty: a pack with nothing in it
		return p, fmt.Errorf("pack %s (%s): %w", name, source, err)
	}
	p.Name, p.Source = name, source
	if len(p.Aside) > 0 && name != Generic {
		return p, fmt.Errorf("pack %s (%s): aside is read from the generic pack only, for now", name, source)
	}
	return p, nil
}

// Available is every pack a project can list: the built-in ones, a
// project's file under .workline/packs/ replacing one of the same name
// whole, or adding one.
func Available(repo string) (map[string]Pack, error) {
	out := map[string]Pack{}
	for n, p := range Builtin() {
		out[n] = p
	}
	dir := filepath.Join(repo, filepath.FromSlash(Dir))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".yaml")
		if e.IsDir() || !ok {
			continue
		}
		p, err := own(repo, name)
		if err != nil {
			return out, err
		}
		out[name] = p
	}
	return out, nil
}

// own reads the project's pack file of that name.
func own(repo, name string) (Pack, error) {
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(Dir), name+".yaml"))
	if err != nil {
		return Pack{}, err
	}
	return parse(name, Dir+"/"+name+".yaml", data)
}

// Listed is the packs a project's config lists, in its order: a name
// with no pack is an error, never skipped. Only the listed packs are
// read: a draft pack file the project does not list stops no run.
func Listed(repo string, names []string) ([]Pack, error) {
	var out []Pack
	for _, n := range names {
		p, err := own(repo, n)
		if errors.Is(err, fs.ErrNotExist) {
			var ok bool
			if p, ok = Builtin()[n]; !ok {
				return nil, fmt.Errorf("packs lists %q: no built-in pack nor %s/%s.yaml has that name", n, Dir, n)
			}
			err = nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// TesterDefaults are the tester's settings the listed packs give, each
// field the union of theirs, then the generic pack's: test-paths,
// test-homes. A pack adds what its ecosystem knows to what most share,
// never narrows it: a Go project's tests/ folder stays tests. The
// listed packs' homes come first, the first matching one wins; a listed
// generic pack, the project's own file, stands for the built-in one.
func TesterDefaults(listed []Pack) map[string]any {
	var paths []any
	var homes []any
	seenP, seenH := map[string]bool{}, map[Home]bool{}
	all := slices.Clone(listed)
	if !slices.ContainsFunc(listed, func(p Pack) bool { return p.Name == Generic }) {
		all = append(all, Builtin()[Generic])
	}
	for _, p := range all {
		for _, t := range p.TestPaths {
			if !seenP[t] {
				seenP[t] = true
				paths = append(paths, t)
			}
		}
		for _, h := range p.TestHomes {
			if !seenH[h] {
				seenH[h] = true
				homes = append(homes, map[string]any{"tests": h.Tests, "code": h.Code})
			}
		}
	}
	out := map[string]any{}
	if len(paths) > 0 {
		out["test-paths"] = paths
	}
	if len(homes) > 0 {
		out["test-homes"] = homes
	}
	return out
}

// Files are the repository's files, from its root: tracked, and the
// untracked ones git does not ignore.
func Files(repo string) ([]string, error) {
	out, err := exec.Command("git", "-C", repo, "-c", "core.quotePath=off", "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		return nil, fmt.Errorf("the repository's files (git ls-files): %w", err)
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

// Detected are the packs among all whose marker files the repository
// holds, sorted, each with the first marker found; files in test or
// aside folders are left out (a fixture's go.mod is no module of the
// project's).
func Detected(files []string, all map[string]Pack, aside func(string) bool) map[string]string {
	out := map[string]string{}
	for _, f := range files {
		if aside(f) {
			continue
		}
		base := path.Base(f)
		for n, p := range all {
			if _, done := out[n]; !done && slices.Contains(p.Detect.Files, base) {
				out[n] = f
			}
		}
	}
	return out
}

// Names are a map's keys, sorted.
func Names[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// Language is a programming language, by its file extensions; File is
// the name a pack of it would take.
type Language struct {
	Name       string   `yaml:"name"`
	File       string   `yaml:"file"`
	Extensions []string `yaml:"extensions"`
}

//go:embed languages.yaml
var languagesYAML []byte

var languages = sync.OnceValue(func() []Language {
	var l []Language
	if err := yaml.Unmarshal(languagesYAML, &l); err != nil {
		panic(fmt.Sprintf("packs: languages.yaml: %v", err)) // embedded: a test reads it
	}
	return l
})

// Unknown are the languages the files hold that no pack of packs knows,
// each with how many files hold it, in languages.yaml's order; files in
// test or aside folders are left out.
func Unknown(files []string, packs []Pack, aside func(string) bool) []Count {
	known := map[string]bool{}
	for _, p := range packs {
		for _, e := range p.Detect.Extensions {
			known[e] = true
		}
	}
	byExt := map[string]int{}
	for _, f := range files {
		if !aside(f) {
			byExt[path.Ext(f)]++
		}
	}
	var out []Count
	for _, l := range languages() {
		n := 0
		for _, e := range l.Extensions {
			if known[e] {
				n = 0
				break
			}
			n += byExt[e]
		}
		if n > 0 {
			out = append(out, Count{Language: l, Files: n})
		}
	}
	return out
}

// Count is a language and how many files hold it.
type Count struct {
	Language
	Files int
}

// Knowing are the packs among all whose detect.extensions hold one of
// the language's, sorted by name.
func Knowing(l Language, all map[string]Pack) []string {
	var out []string
	for _, n := range Names(all) {
		for _, e := range all[n].Detect.Extensions {
			if slices.Contains(l.Extensions, e) {
				out = append(out, n)
				break
			}
		}
	}
	return out
}
