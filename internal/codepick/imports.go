package codepick

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
)

// importRule is how a family of files names the files it includes or
// imports: a pattern whose first group is the name imported, and how a
// name becomes a path. Data, one row a family, never a parser
// (docs/research/code-outline.md); a family missing here imports
// nothing the picker follows.
type importRule struct {
	exts []string
	re   *regexp.Regexp
	// dots: `a.b.c` is `a/b/c`; leading dots climb from the file's folder.
	dots bool
	// relative: only `./` and `../` names are the project's; a bare name
	// is a package from outside (JavaScript's `react`).
	relative bool
	// tries are the endings tried after the name; "" the name as written.
	tries []string
	// folder: the name is a folder, its files imported (Go's packages).
	folder bool
	// why is said of a file it finds: "included or imported".
	why string
}

var (
	cExts  = []string{".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".hh", ".hxx", ".ipp", ".inl", ".m", ".mm", ".ino"}
	jsExts = []string{".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts", ".vue", ".svelte"}
	jsTry  = []string{"", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".d.ts", "/index.ts", "/index.tsx", "/index.js"}
	jvm    = []string{".java", ".kt", ".kts", ".scala", ".groovy"}
)

// importRules are the families known.
var importRules = []importRule{
	{exts: cExts, re: regexp.MustCompile(`^\s*#\s*(?:include|import)\s*[<"]([^>"]+)[>"]`), tries: []string{""}, why: "included or imported"},
	{exts: []string{".py"}, re: regexp.MustCompile(`^\s*from\s+(\.*[\w.]*)\s+import\b`), dots: true, tries: []string{".py", "/__init__.py"}, why: "imported"},
	{exts: []string{".py"}, re: regexp.MustCompile(`^\s*import\s+([\w.]+)(?:\s+as\s+\w+)?\s*$`), dots: true, tries: []string{".py", "/__init__.py"}, why: "imported"},
	{exts: jsExts, re: regexp.MustCompile(`\bfrom\s+['"]([^'"]+)['"]`), relative: true, tries: jsTry, why: "included or imported"},
	{exts: jsExts, re: regexp.MustCompile(`^\s*import\s+['"]([^'"]+)['"]`), relative: true, tries: jsTry, why: "included or imported"},
	{exts: jsExts, re: regexp.MustCompile(`\b(?:require|import)\(\s*['"]([^'"]+)['"]\s*\)`), relative: true, tries: jsTry, why: "included or imported"},
	{exts: []string{".go"}, re: regexp.MustCompile(`^\s*(?:import\s+)?(?:[\w.]+\s+)?"([\w.~-]+(?:/[\w.~-]+)+)"\s*$`), folder: true, why: "imported"},
	{exts: jvm, re: regexp.MustCompile(`^\s*import\s+(?:static\s+)?([\w.]+?)(?:\.\*)?\s*;?\s*$`), dots: true, tries: jvm, why: "imported"},
	{exts: []string{".rs"}, re: regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?mod\s+(\w+)\s*;`), tries: []string{".rs", "/mod.rs"}, why: "imported"},
	{exts: []string{".rb"}, re: regexp.MustCompile(`^\s*require(?:_relative)?\s*\(?\s*['"]([^'"]+)['"]`), tries: []string{".rb", ""}, why: "required"},
	{exts: []string{".php"}, re: regexp.MustCompile(`\b(?:require|include)(?:_once)?\s*\(?\s*['"]([^'"]+)['"]`), tries: []string{""}, why: "included"},
	{exts: []string{".lua"}, re: regexp.MustCompile(`\brequire\s*\(?\s*['"]([\w.]+)['"]`), dots: true, tries: []string{".lua", "/init.lua"}, why: "required"},
	{exts: []string{".sh", ".bash", ".zsh"}, re: regexp.MustCompile(`^\s*(?:source|\.)\s+['"]?([\w./-]+\.\w+)`), tries: []string{""}, why: "sourced"},
}

// link is one file a file includes or imports, the project's own.
type link struct {
	file, why string
}

// imports are the project's files a file includes or imports, in their
// order, each once; none when its family is not known.
func (p *Picker) imports(from string, tests []string) []link {
	ext := strings.ToLower(path.Ext(from))
	var out []link
	seen := map[string]bool{from: true}
	for _, l := range p.Lines(from) {
		for _, r := range importRules {
			if !slices.Contains(r.exts, ext) {
				continue
			}
			m := r.re.FindStringSubmatch(l)
			if m == nil || m[1] == "" {
				continue
			}
			for _, f := range p.resolveImport(from, m[1], r, tests) {
				if !seen[f] {
					seen[f] = true
					out = append(out, link{f, r.why})
				}
			}
		}
	}
	return out
}

// resolveImport is the files a name imported from a file stands for:
// beside it, from the root, or the one file whose path ends with it; a
// folder's files of the same family for a rule naming folders.
func (p *Picker) resolveImport(from, name string, r importRule, tests []string) []string {
	dir := path.Dir(from)
	rel := strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../")
	if r.dots {
		up := len(name) - len(strings.TrimLeft(name, "."))
		name = strings.ReplaceAll(strings.TrimLeft(name, "."), ".", "/")
		if up > 0 {
			rel = true
			for range up - 1 {
				dir = path.Dir(dir)
			}
			if name == "" {
				name = "."
			}
		}
	}
	if r.relative && !rel {
		return nil // a package from outside
	}
	if r.folder {
		return p.folderFiles(name, path.Ext(from), tests)
	}
	bases := []string{path.Join(dir, name)}
	if !rel {
		bases = append(bases, path.Clean(name))
	}
	if e := path.Ext(name); e != "" && r.relative { // `./a.js` written for `a.ts`
		bases = append(bases, path.Join(dir, strings.TrimSuffix(name, e)))
	}
	for _, b := range bases {
		for _, t := range r.tries {
			if p.tracked[b+t] {
				return []string{b + t}
			}
		}
	}
	if rel {
		return nil
	}
	for _, t := range r.tries {
		found := ""
		for _, f := range p.files {
			if strings.HasSuffix(f, "/"+path.Clean(name)+t) {
				if found != "" {
					return nil // several: none named
				}
				found = f
			}
		}
		if found != "" {
			return []string{found}
		}
	}
	return nil
}

// folderFiles are the files of the folder a path ends with, of the
// family ext, its tests aside; the longest such ending of the path wins
// (Go's `github.com/o/r/internal/x` is the project's `internal/x`).
func (p *Picker) folderFiles(name, ext string, tests []string) []string {
	parts := strings.Split(path.Clean(name), "/")
	for i := range parts {
		dir := strings.Join(parts[i:], "/")
		var out []string
		for _, f := range p.files {
			if path.Dir(f) == dir && path.Ext(f) == ext && (len(tests) == 0 || !pathglob.Any(tests, f)) {
				out = append(out, f)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}
