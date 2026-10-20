package gate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// sarifFile is a SARIF file a check wrote, kept whole so it can be written
// back with each result's baselineState.
type sarifFile struct {
	name    string
	doc     map[string]any
	results []*result
	tools   []string // each run's tool and version, when it says one
}

// result is one SARIF result, with what tells it from another.
type result struct {
	raw         map[string]any
	tool        string
	rule, level string
	text        string
	uri         string // from the tree's root, slashes
	line        int
	fingerprint string // the tool's own, or ""
	state       string // new or unchanged, once compared
	posted      string // the fingerprint it is shown by (added.go)
}

// readSARIF reads the SARIF files a check wrote in out since it started,
// each result's file made relative to root, the tree the check ran in. A
// file read is renamed, counted once even if another check writes SARIF
// later.
func readSARIF(out, root string) ([]sarifFile, error) {
	names, _ := filepath.Glob(filepath.Join(out, "*.sarif"))
	if len(names) == 0 {
		return nil, errors.New("the check wrote no SARIF file in {out}")
	}
	sort.Strings(names)
	var files []sarifFile
	for _, name := range names {
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("%s is not readable SARIF: %v", filepath.Base(name), err)
		}
		f := sarifFile{name: filepath.Base(name), doc: doc}
		runs, _ := doc["runs"].([]any)
		for _, r := range runs {
			run, _ := r.(map[string]any)
			driver := obj(obj(run, "tool"), "driver")
			tool, version := str(driver, "name"), str(driver, "semanticVersion")
			if version == "" {
				version = str(driver, "version")
			}
			if version != "" {
				f.tools = append(f.tools, tool+" "+version)
			}
			results, _ := run["results"].([]any)
			for _, x := range results {
				raw, ok := x.(map[string]any)
				if !ok {
					continue
				}
				f.results = append(f.results, parse(raw, tool, root))
			}
		}
		files = append(files, f)
		os.Rename(name, name+".read")
	}
	return files, nil
}

func parse(raw map[string]any, tool, root string) *result {
	r := &result{raw: raw, tool: tool, rule: str(raw, "ruleId"), level: str(raw, "level"), text: str(obj(raw, "message"), "text")}
	if r.level == "" {
		r.level = "warning" // as the SARIF standard says
	}
	if locs, _ := raw["locations"].([]any); len(locs) > 0 {
		loc, _ := locs[0].(map[string]any)
		phys := obj(loc, "physicalLocation")
		r.uri = relative(str(obj(phys, "artifactLocation"), "uri"), root)
		if n, ok := obj(phys, "region")["startLine"].(float64); ok {
			r.line = int(n)
		}
	}
	r.fingerprint = fingerprint(obj(raw, "fingerprints"))
	if r.fingerprint == "" {
		r.fingerprint = fingerprint(obj(raw, "partialFingerprints"))
	}
	return r
}

// fingerprint joins a result's fingerprints, in the order of their keys;
// workline's own is left out, as it holds the finding's line
// (docs/research/inspector.md).
func fingerprint(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != "workline/v1" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v\x00", k, m[k])
	}
	return b.String()
}

// relative is a result's file from the root of the tree the check ran in:
// a file URI or an absolute path made relative, a relative one cleaned.
func relative(uri, root string) string {
	if uri == "" {
		return ""
	}
	p := uri
	if u, err := url.Parse(uri); err == nil && u.Scheme == "file" {
		p = u.Path
	} else if s, err := url.PathUnescape(uri); err == nil {
		p = s
	}
	if filepath.IsAbs(p) {
		for _, r := range []string{root, evalSymlinks(root)} {
			if rel, err := filepath.Rel(r, p); err == nil && !strings.HasPrefix(rel, "..") {
				return filepath.ToSlash(rel)
			}
		}
		return filepath.ToSlash(p)
	}
	return path.Clean(filepath.ToSlash(p))
}

func evalSymlinks(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func obj(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// lines reads the lines of a tree's files, each read once.
type lines struct {
	root  string
	files map[string][]string
}

// at is a file's line, its spaces and tabs left out, as GitHub's line hash
// reads it: a finding moved keeps it, one whose line changed does not.
func (l *lines) at(file string, n int) string {
	if l.files == nil {
		l.files = map[string][]string{}
	}
	text, ok := l.files[file]
	if !ok {
		data, _ := os.ReadFile(filepath.Join(l.root, filepath.FromSlash(file)))
		text = strings.Split(string(data), "\n")
		l.files[file] = text
	}
	if n < 1 || n > len(text) {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\r' {
			return -1
		}
		return r
	}, text[n-1])
}

// identity tells a finding from another without its line number (SARIF's
// appendix B): the tool, the rule and the file, named as the head names
// it, then the tool's own fingerprints when it writes some, else the
// message and the text of the line it is on, read in its own tree.
func (r *result) identity(l *lines, name string) string {
	return identity(r.tool, r.rule, name, r.rest(l))
}

// rest is what tells a finding apart beside its tool, rule and file: the
// tool's own fingerprints, else its message and the text of its line.
func (r *result) rest(l *lines) string {
	if r.fingerprint != "" {
		return r.fingerprint
	}
	return r.text + "\x00" + l.at(r.uri, r.line)
}

func identity(tool, rule, name, rest string) string {
	return tool + "\x00" + rule + "\x00" + name + "\x00" + rest
}

// compare marks each of the head's results new or unchanged, one for one:
// a result the base holds as many times as the head is not new, wherever
// its line moved to; a file the change renamed is followed.
func compare(head, base []*result, headLines, baseLines *lines, renamed map[string]string) {
	compareWith(head, keptResults(base, baseLines), headLines, renamed)
}

// compareWith is compare, the base's results as kept: read in the base's
// tree, or on main when it was analysed there (kept.go).
func compareWith(head []*result, base []keptResult, headLines *lines, renamed map[string]string) {
	there := map[string]int{}
	for _, r := range base {
		name := r.Path
		if to, ok := renamed[name]; ok {
			name = to
		}
		there[identity(r.Tool, r.Rule, name, r.Rest)]++
	}
	for _, r := range head {
		id := r.identity(headLines, r.uri)
		if there[id] > 0 {
			there[id]--
			r.state = "unchanged"
		} else {
			r.state = "new"
		}
	}
}

// writeCompared writes the head's SARIF files again, each result's
// baselineState said (SARIF 2.1.0, 3.27.24), for what posts them.
func writeCompared(dir string, files []sarifFile) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		for _, r := range f.results {
			r.raw["baselineState"] = r.state
		}
		data, err := json.MarshalIndent(f.doc, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, f.name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
