// Package testdef is the one definition of a test file every role reads
// (ADR-0065, change 3): the project's test-paths and test-homes, the
// tester's settings, else the built-in generic ones, embedded data
// (generic.yaml), never engine code.
package testdef

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/pathglob"
)

// The settings the definition is read from, the tester's.
const (
	PathsSetting = "test-paths"
	HomesSetting = "test-homes"
)

// Home maps the folder of a test file onto the folder of its code: each
// `**` of Tests fills the next one of Code, the rest dropped.
type Home struct {
	Tests string `json:"tests" yaml:"tests"`
	Code  string `json:"code" yaml:"code"`
}

// Def says which files are tests and where a test's code lives. Its zero
// value is the generic definition.
type Def struct {
	Paths  []string
	Homes  []Home
	Asides []string
	// PathsSet, HomesSet: the project set them; else the generic ones.
	PathsSet, HomesSet bool
}

//go:embed generic.yaml
var genericYAML []byte

var generic = sync.OnceValue(func() Def {
	var g struct {
		Paths []string `yaml:"test-paths"`
		Homes []Home   `yaml:"test-homes"`
		Aside []string `yaml:"aside"`
	}
	if err := yaml.Unmarshal(genericYAML, &g); err != nil {
		panic(fmt.Sprintf("testdef: generic.yaml: %v", err)) // embedded: a test reads it
	}
	return Def{Paths: g.Paths, Homes: g.Homes, Asides: g.Aside}
})

// Generic is the built-in generic definition.
func Generic() Def { return generic() }

// From is the definition from a project's test-paths and test-homes:
// each unset, the generic one.
func From(paths []string, homes []Home) Def {
	d := Generic()
	if len(paths) > 0 {
		d.Paths, d.PathsSet = paths, true
	}
	if len(homes) > 0 {
		d.Homes, d.HomesSet = homes, true
	}
	return d
}

// FromSettings reads the definition from a role's settings, the
// tester's: test-paths, test-homes.
func FromSettings(m map[string]any) (Def, error) {
	var s struct {
		Paths []string `json:"test-paths"`
		Homes []Home   `json:"test-homes"`
	}
	data, err := json.Marshal(map[string]any{PathsSetting: m[PathsSetting], HomesSetting: m[HomesSetting]})
	if err == nil {
		err = json.Unmarshal(data, &s)
	}
	if err != nil {
		return Generic(), fmt.Errorf("the tester's test-paths and test-homes: %w", err)
	}
	if err := Check(s.Homes); err != nil {
		return From(s.Paths, nil), err
	}
	return From(s.Paths, s.Homes), nil
}

// ErrHomes marks test-homes that cannot be read: every role then ignores
// them, the generic homes in their place, the same answer everywhere.
var ErrHomes = errors.New("test-homes ignored")

// Check says what is wrong with test-homes: a `code` asking for more `**`
// than its `tests` gives, or a wildcard other than `**` in it.
func Check(homes []Home) error {
	for _, h := range homes {
		if h.Tests == "" || h.Code == "" {
			return fmt.Errorf("%w: %+v: both tests and code are needed", ErrHomes, h)
		}
		n := 0
		for _, seg := range strings.Split(h.Code, "/") {
			switch {
			case seg == "**":
				n++
			case strings.ContainsAny(seg, "*?["):
				return fmt.Errorf("%w: code %q: only ** may stand for folders", ErrHomes, h.Code)
			}
		}
		if n > strings.Count("/"+h.Tests+"/", "/**/") {
			return fmt.Errorf("%w: code %q has more ** than tests %q", ErrHomes, h.Code, h.Tests)
		}
	}
	return nil
}

func (d Def) orGeneric() Def {
	if d.Paths == nil && d.Homes == nil && d.Asides == nil {
		return Generic()
	}
	return d
}

// Patterns are the test files' patterns.
func (d Def) Patterns() []string { return d.orGeneric().Paths }

// Test says whether a file, by its path from the repository's root, is a
// test.
func (d Def) Test(p string) bool { return pathglob.Any(d.Patterns(), clean(p)) }

// Aside says whether a file is beside the project's own code: a test, or
// in a folder that holds neither code nor tests (examples, tools).
func (d Def) Aside(p string) bool {
	p = clean(p)
	return d.Test(p) || pathglob.Any(d.orGeneric().Asides, p) || pathglob.Any(d.orGeneric().Asides, strings.ToLower(p))
}

// Home is the folder a test file's code lives in: its folder mapped by
// the first test-home whose tests match it, else its own folder.
func (d Def) Home(p string) string {
	dir := path.Dir(clean(p))
	for _, h := range d.orGeneric().Homes {
		if caps, ok := capture(split(h.Tests), split(dir)); ok {
			return fill(h.Code, caps)
		}
	}
	return dir
}

func clean(p string) string { return path.Clean(strings.ReplaceAll(p, "\\", "/")) }

func split(p string) []string {
	if p == "." || p == "" {
		return nil
	}
	return strings.Split(path.Clean(p), "/")
}

// capture matches name against pat, as pathglob does, and returns what
// each `**` spans, the shortest first.
func capture(pat, name []string) ([]string, bool) {
	if len(pat) == 0 {
		return nil, len(name) == 0
	}
	if pat[0] == "**" {
		for i := 0; i <= len(name); i++ {
			if rest, ok := capture(pat[1:], name[i:]); ok {
				return append([]string{strings.Join(name[:i], "/")}, rest...), true
			}
		}
		return nil, false
	}
	if len(name) == 0 {
		return nil, false
	}
	if ok, _ := path.Match(pat[0], name[0]); !ok {
		return nil, false
	}
	return capture(pat[1:], name[1:])
}

// fill writes code with each `**` the next capture.
func fill(code string, caps []string) string {
	var out []string
	for _, seg := range strings.Split(code, "/") {
		if seg == "**" {
			if len(caps) > 0 {
				if caps[0] != "" {
					out = append(out, caps[0])
				}
				caps = caps[1:]
			}
			continue
		}
		out = append(out, seg)
	}
	if len(out) == 0 {
		return "."
	}
	return path.Clean(strings.Join(out, "/"))
}
