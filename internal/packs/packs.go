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
	"regexp"
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
	// BuiltinNames are the language's own names (its built-in exceptions
	// and standard library's): a Scope naming one is no stale name, the
	// code never holds it.
	BuiltinNames []string `yaml:"builtin-names"`
	// Code are its languages' code navigation rows, read by
	// internal/codenav: how a file's functions and declarations are found
	// with no build.
	Code []Code `yaml:"code-navigation"`
	// BuildErrors are the rows the tester reads a build broken by a new
	// test with, the tester's build-errors setting when the project sets
	// none: the shape of that setting (docs/packs.md).
	BuildErrors []BuildError `yaml:"build-errors"`
}

// BuildError is one build-errors row: an error line of the test command's
// output, the messages naming what the code lacks, and the tests a file
// declares.
type BuildError struct {
	Error       string   `yaml:"error"`
	MissingName []string `yaml:"missing-name,omitempty"`
	NameLine    string   `yaml:"name-line,omitempty"`
	MessageLine string   `yaml:"message-line,omitempty"`
	TestName    string   `yaml:"test-name,omitempty"`
}

func (b BuildError) row() map[string]any {
	m := map[string]any{"error": b.Error}
	if len(b.MissingName) > 0 {
		names := make([]any, len(b.MissingName))
		for i, n := range b.MissingName {
			names[i] = n
		}
		m["missing-name"] = names
	}
	if b.NameLine != "" {
		m["name-line"] = b.NameLine
	}
	if b.MessageLine != "" {
		m["message-line"] = b.MessageLine
	}
	if b.TestName != "" {
		m["test-name"] = b.TestName
	}
	return m
}

// Code is how one language's functions are found, and how its
// declarations are outlined, with no build
// (docs/research/code-navigation.md, docs/research/code-outline.md).
type Code struct {
	Name       string   `yaml:"name"`
	Extensions []string `yaml:"extensions"`
	// Definition is a definition's pattern, NAME standing for its name,
	// or "c-like" for CLike.
	Definition string `yaml:"definition"`
	// Body is how a function's bounds are found: go (Go's parser), braces,
	// indent (Python), end (Ruby, Lua).
	Body string `yaml:"body"`
	// Outline is how its declarations are outlined: braces, indent, or
	// none: whole or cut, never a body shown.
	Outline string `yaml:"outline"`
	// Containers are the words opening a block whose members are
	// declarations (`class`, `struct`, `namespace`): kept in an outline.
	Containers []string `yaml:"containers"`
	// Bodies, for an indented language, are the words opening a body cut
	// from an outline (`def`); another block (`if TYPE_CHECKING:`) is kept.
	Bodies []string `yaml:"bodies"`
	// Ignore is the pattern of what a block's header holds that says
	// nothing of the block: annotations, attributes, a template's head.
	Ignore string `yaml:"ignore"`
	// Preproc: a line opening with `#` is the preprocessor's, kept.
	Preproc bool `yaml:"preproc"`
	// LineEnds: a line ends a statement, no `;` needed, unless a
	// parenthesis is still open.
	LineEnds bool `yaml:"line-ends"`
	// Bases is the pattern of the types a type derives from or embeds,
	// each group a name.
	Bases string `yaml:"bases"`
	// ParamHeaders: a type's header may hold its parameters: a container
	// word before a parenthesis keeps the block.
	ParamHeaders bool `yaml:"param-headers"`
	// ShellQuotes: a single quote is raw, a backquote no string, a `#`
	// after a space opens a comment.
	ShellQuotes bool `yaml:"shell-quotes"`
	// HashComments: a `#` opens a comment.
	HashComments bool `yaml:"hash-comments"`
	// BareCalls: a function is called with no parenthesis, any word may
	// be one.
	BareCalls bool `yaml:"bare-calls"`
	// Keywords are the language's words that name no definition worth a
	// search when a text writes them as code, space-separated.
	Keywords string `yaml:"keywords"`
}

// CLike is a C-like language's definition, a row's `definition: c-like`:
// a function by its keyword, a function bound to a name, or a name before
// its parameters.
const CLike = `(?:\b(?:function|fn|fun|func|def)\s+(NAME)\b|\b(?:const|let|var)\s+(NAME)\s*=\s*(?:async\s+)?(?:function\b|\([^)]*\)\s*=>|[A-Za-z_$][\w$]*\s*=>)|^\s*(?:[\w$:<>,*&\[\]~.]+\s+)*(?:[\w$]+(?:<(?:[^<>()]|<[^<>()]*>)*>)?::)*(NAME)\s*\([^;{]*(?:\{.*)?$)`

// wordRe is a word of a language: a container or a body's opening word.
var wordRe = regexp.MustCompile(`^\w+$`)

// check reads a code navigation row: a word or a pattern it cannot read is
// refused, said with the row's name.
func (c *Code) check() error {
	if c.Name == "" {
		return errors.New("code-navigation: a row has no name")
	}
	if len(c.Extensions) == 0 {
		return fmt.Errorf("code-navigation %s: no extensions", c.Name)
	}
	for _, e := range c.Extensions {
		if !strings.HasPrefix(e, ".") || e != strings.ToLower(e) {
			return fmt.Errorf("code-navigation %s: extension %q: a dot, then lower case", c.Name, e)
		}
	}
	if c.Definition == "c-like" {
		c.Definition = CLike
	}
	if !strings.Contains(c.Definition, "NAME") {
		return fmt.Errorf("code-navigation %s: definition holds no NAME", c.Name)
	}
	for _, w := range append(slices.Clone(c.Containers), c.Bodies...) {
		if !wordRe.MatchString(w) {
			return fmt.Errorf("code-navigation %s: %q in containers or bodies: a word", c.Name, w)
		}
	}
	if !slices.Contains([]string{"go", "braces", "indent", "end"}, c.Body) {
		return fmt.Errorf("code-navigation %s: body is %q: go, braces, indent or end", c.Name, c.Body)
	}
	if c.Outline == "none" {
		c.Outline = ""
	}
	if !slices.Contains([]string{"", "braces", "indent"}, c.Outline) {
		return fmt.Errorf("code-navigation %s: outline is %q: braces, indent, or none", c.Name, c.Outline)
	}
	for what, re := range map[string]string{"definition": strings.ReplaceAll(c.Definition, "NAME", "x"), "ignore": c.Ignore, "bases": c.Bases} {
		if _, err := regexp.Compile(re); err != nil {
			return fmt.Errorf("code-navigation %s: %s: %w", c.Name, what, err)
		}
	}
	return nil
}

// CodeRows are the code navigation rows a project reads: the listed
// packs' first, a project's own row winning on its extensions, then
// every built-in pack's, by name. Navigation decides no verdict: a Go
// project's shell scripts are read too, whatever its list.
func CodeRows(listed []Pack) []Code {
	var out []Code
	done := map[string]bool{}
	for _, p := range listed {
		out = append(out, p.Code...)
		if p.Source == "built-in" {
			done[p.Name] = true
		}
	}
	b := Builtin()
	for _, n := range Names(b) {
		if !done[n] {
			out = append(out, b[n].Code...)
		}
	}
	return out
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
	for i := range p.Code {
		if err := p.Code[i].check(); err != nil {
			return p, fmt.Errorf("pack %s (%s): %w", name, source, err)
		}
	}
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

// OfProject are the packs a project's .workline/config.yaml lists, read:
// none when it lists none, or has no config. For a role reading packs
// with no settings to carry them, the code navigation rows; the config's
// other keys are role.LoadProjectConfig's to check.
func OfProject(repo string) ([]Pack, error) {
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c struct {
		Packs []string `yaml:"packs"`
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	return Listed(repo, c.Packs)
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
	var errs []any
	seenE := map[string]bool{}
	seenP, seenH := map[string]bool{}, map[Home]bool{}
	all := slices.Clone(listed)
	if !slices.ContainsFunc(listed, func(p Pack) bool { return p.Name == Generic }) {
		all = append(all, Builtin()[Generic])
	}
	for _, p := range all {
		for _, e := range p.BuildErrors {
			row := e.row()
			if k := fmt.Sprint(row); !seenE[k] {
				seenE[k] = true
				errs = append(errs, row)
			}
		}
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
	if len(errs) > 0 {
		out["build-errors"] = errs
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
