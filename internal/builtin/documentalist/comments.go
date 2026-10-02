package documentalist

import (
	"path"
	"strings"
	"unicode"
)

// Comments, by a file's type (ADR-0014, step 2: a comment is not evidence).
// Fitted at first to C-like code, the test read as code Python docstrings,
// extensionless scripts, SQL, Handlebars and Django templates
// (docs/research/documentalist-genericity.md); a type it does not know is
// now said, not silently taken as code.

// commentStyle is how a file writes its comments.
type commentStyle struct {
	known bool     // the type is known, comments or none
	line  []string // a line comment starts with one of these
	// afterSpace are the line markers that start a comment only at the
	// start of a line or after white space: `#` in shell, `//` in a Vue
	// template, where "http://" is no comment.
	afterSpace []string
	block      [][2]string // longest opening first where two share a prefix
	backtick   bool        // backticks quote, over several lines (Go, JavaScript)
	docstrings bool        // Python: a string standing as a statement is a comment
}

var (
	cBlock    = [2]string{"/*", "*/"}
	htmlBlock = [2]string{"<!--", "-->"}
	hashLine  = commentStyle{known: true, line: []string{"#"}, afterSpace: []string{"#"}}
	slashes   = commentStyle{known: true, line: []string{"//"}, block: [][2]string{cBlock}}
	noComment = commentStyle{known: true}
)

// styles are the comment styles by extension.
var styles = map[string]commentStyle{}

func init() {
	set := func(st commentStyle, exts ...string) {
		for _, e := range exts {
			styles[e] = st
		}
	}
	set(commentStyle{known: true, line: []string{"//"}, block: [][2]string{cBlock}, backtick: true},
		"go", "js", "mjs", "cjs", "ts", "tsx", "jsx", "mts", "cts")
	set(slashes, "c", "h", "cpp", "hpp", "cc", "cxx", "hh", "hxx", "ino", "m", "mm", "java", "kt", "kts",
		"rs", "swift", "cs", "scala", "sc", "dart", "proto", "groovy", "gradle", "zig", "sol", "v", "d",
		"glsl", "hlsl", "vert", "frag", "jsonc", "json5", "mod", "work", "scss", "less", "styl")
	set(commentStyle{known: true, line: []string{"//", "#"}, block: [][2]string{cBlock}}, "php")
	set(commentStyle{known: true, block: [][2]string{cBlock}}, "css")
	set(commentStyle{known: true, block: [][2]string{htmlBlock}},
		"md", "markdown", "mdx", "xml", "svg", "xsd", "xsl", "xslt", "plist", "csproj", "props", "targets", "xaml", "resx", "mjml")
	// HTML, and the templates written in it: Django, Jinja, Nunjucks, Twig, Liquid.
	set(commentStyle{known: true, block: [][2]string{htmlBlock, {"{#", "#}"}, {"{% comment %}", "{% endcomment %}"}}},
		"html", "htm", "xhtml", "jinja", "jinja2", "j2", "njk", "twig", "liquid", "djhtml", "jsp", "erb", "ejs")
	set(commentStyle{known: true, block: [][2]string{{"{{!--", "--}}"}, {"{{!", "}}"}, htmlBlock}},
		"hbs", "handlebars", "mustache")
	// A single-file component: its template, script and style.
	set(commentStyle{known: true, line: []string{"//"}, afterSpace: []string{"//"}, block: [][2]string{htmlBlock, cBlock}},
		"vue", "svelte", "astro")
	set(hashLine, "yaml", "yml", "sh", "bash", "zsh", "fish", "ksh", "rb", "rake", "gemspec", "toml", "cfg",
		"conf", "pl", "pm", "r", "mk", "cmake", "env", "properties", "dockerfile", "tcl", "awk", "cr",
		"ex", "exs", "gitignore", "dockerignore", "containerfile", "bzl", "star", "bazel", "pp", "nim",
		"graphql", "gql", "in", "mailmap", "snyk")
	set(commentStyle{known: true, line: []string{"#"}, afterSpace: []string{"#"}, docstrings: true}, "py", "pyi", "pyw", "pyx", "pxd")
	set(commentStyle{known: true, line: []string{"#"}, afterSpace: []string{"#"}, block: [][2]string{{"<#", "#>"}}}, "ps1", "psm1", "psd1")
	set(commentStyle{known: true, line: []string{"#"}, afterSpace: []string{"#"}, block: [][2]string{{"#=", "=#"}}}, "jl")
	set(commentStyle{known: true, line: []string{"#"}, afterSpace: []string{"#"}, block: [][2]string{cBlock}}, "nix")
	set(commentStyle{known: true, line: []string{"#", ";"}, afterSpace: []string{"#"}}, "ini", "editorconfig", "npmrc", "gitconfig", "coveragerc", "pylintrc", "flake8")
	set(commentStyle{known: true, line: []string{"#", "//"}, afterSpace: []string{"#"}, block: [][2]string{cBlock}}, "tf", "hcl", "tfvars")
	set(commentStyle{known: true, line: []string{"--"}, block: [][2]string{cBlock}}, "sql", "psql", "pgsql", "plsql", "ddl", "hql")
	set(commentStyle{known: true, line: []string{"--"}, block: [][2]string{{"--[[", "]]"}}}, "lua")
	set(commentStyle{known: true, line: []string{"--"}, block: [][2]string{{"{-", "-}"}}}, "hs", "elm", "purs")
	set(commentStyle{known: true, line: []string{"--"}}, "ada", "adb", "ads", "vhd", "vhdl")
	set(commentStyle{known: true, line: []string{";"}}, "lisp", "el", "clj", "cljs", "cljc", "edn", "scm", "rkt", "asm", "s")
	set(commentStyle{known: true, line: []string{"%"}}, "tex", "sty", "cls", "erl", "hrl")
	set(noComment, "json", "txt", "csv", "tsv", "lock", "sum", "snap", "rst", "adoc", "ipynb", "geojson", "typed", "patch", "diff", "pem")
}

// names are the styles of files known by name rather than by extension.
var names = map[string]commentStyle{
	"makefile": hashLine, "gnumakefile": hashLine, "dockerfile": hashLine, "containerfile": hashLine,
	"cmakelists.txt": hashLine, "gemfile": hashLine, "rakefile": hashLine, "podfile": hashLine,
	"vagrantfile": hashLine, "brewfile": hashLine, "procfile": hashLine, "justfile": hashLine,
	"codeowners": hashLine, "pipfile": hashLine, "build": hashLine, "workspace": hashLine,
	"license": noComment, "copying": noComment, "notice": noComment, "authors": noComment,
	"version": noComment, "owners": noComment, "maintainers": noComment,
}

// interpreters are the styles of scripts without an extension, by the
// program their first line `#!` names.
var interpreters = map[string]string{
	"sh": "sh", "bash": "sh", "zsh": "sh", "dash": "sh", "ksh": "sh", "fish": "sh", "ash": "sh",
	"python": "py", "python2": "py", "python3": "py", "pypy": "py", "pypy3": "py",
	"node": "js", "deno": "js", "bun": "js", "tsx": "ts", "ts-node": "ts",
	"ruby": "rb", "perl": "pl", "lua": "lua", "php": "php", "rscript": "r", "tclsh": "tcl",
	"awk": "awk", "gawk": "awk", "pwsh": "ps1", "julia": "jl", "make": "mk",
}

// styleOf is how a file writes its comments: by its extension, its name,
// or, for a script, the interpreter its first line names. kind names its
// type for a person: ".ext", or the file's name.
func styleOf(p, content string) (st commentStyle, kind string) {
	base := strings.ToLower(path.Base(p))
	ext := strings.TrimPrefix(path.Ext(base), ".")
	if s, ok := styles[ext]; ok {
		return s, "." + ext
	}
	if s, ok := names[base]; ok {
		return s, base
	}
	if strings.HasPrefix(base, ".git") || strings.HasPrefix(base, ".env") || strings.HasSuffix(base, "ignore") {
		return hashLine, base
	}
	if strings.HasPrefix(base, "license") || strings.HasPrefix(base, "unlicense") || strings.HasPrefix(base, "copying") {
		return noComment, base
	}
	if prog := shebang(content); prog != "" {
		for _, v := range []string{prog, strings.TrimRight(prog, "0123456789.")} {
			if e, ok := interpreters[v]; ok {
				return styles[e], "#!" + prog
			}
		}
		return commentStyle{}, "#!" + prog
	}
	if ext == "" {
		return commentStyle{}, base
	}
	return commentStyle{}, "." + ext
}

// shebang is the program a script's first line `#!` runs, "" if none:
// `#!/usr/bin/env python3` is python3.
func shebang(content string) string {
	if !strings.HasPrefix(content, "#!") {
		return ""
	}
	first, _, _ := strings.Cut(content[2:], "\n")
	fields := strings.Fields(first)
	if len(fields) == 0 {
		return ""
	}
	prog := path.Base(fields[0])
	if prog == "env" {
		prog = ""
		for _, f := range fields[1:] {
			if !strings.HasPrefix(f, "-") && !strings.Contains(f, "=") {
				prog = path.Base(f)
				break
			}
		}
	}
	return strings.ToLower(prog)
}

// commentMask says, for each byte of a file, whether it is in a comment,
// by the file's type. Strings are read as strings: a `//` in a URL between
// quotes is no comment. A Python string standing as a statement — a
// docstring — is a comment.
func commentMask(p, content string) []bool {
	st, _ := styleOf(p, content)
	mask := make([]bool, len(content))
	mark := func(from, to int) {
		for k := from; k < to && k < len(mask); k++ {
			mask[k] = true
		}
	}
	lineStart := true
	var last byte // the last byte of code before i, white space aside
	for i := 0; i < len(content); {
		c := content[i]
		if c == '\n' {
			lineStart = true
			i++
			continue
		}
		// A block comment.
		opened := false
		for _, b := range st.block {
			if strings.HasPrefix(content[i:], b[0]) {
				end := strings.Index(content[i+len(b[0]):], b[1])
				stop := len(content)
				if end >= 0 {
					stop = i + len(b[0]) + end + len(b[1])
				}
				mark(i, stop)
				i, opened = stop, true
				break
			}
		}
		if opened {
			continue
		}
		// A line comment.
		for _, l := range st.line {
			if !strings.HasPrefix(content[i:], l) {
				continue
			}
			if contains(st.afterSpace, l) && !lineStart && i > 0 && !unicode.IsSpace(rune(content[i-1])) {
				continue
			}
			end := strings.IndexByte(content[i:], '\n')
			stop := len(content)
			if end >= 0 {
				stop = i + end
			}
			mark(i, stop)
			i, opened = stop, true
			break
		}
		if opened {
			continue
		}
		// A Python string over three quotes: a docstring when it stands as
		// a statement — first on its line, not continuing an expression.
		if st.docstrings {
			if q, n := tripleQuote(content, i); n > 0 {
				end := strings.Index(content[i+n:], q)
				stop := len(content)
				if end >= 0 {
					stop = i + n + end + len(q)
				}
				if lineStart && !strings.ContainsRune("([{,=+\\%", rune(last)) {
					mark(i, stop)
				} else {
					last = q[0]
				}
				i, lineStart = stop, false
				continue
			}
		}
		// A string: skipped whole when it closes, on its line (or, for a
		// backtick, anywhere); an apostrophe that closes nothing is a letter.
		if len(st.line)+len(st.block) > 0 && (c == '"' || c == '\'' || c == '`' && st.backtick) {
			if end := closing(content, i, c); end > 0 {
				i, lineStart, last = end+1, false, c
				continue
			}
		}
		if !unicode.IsSpace(rune(c)) {
			lineStart, last = false, c
		}
		i++
	}
	return mask
}

// tripleQuote says whether a Python string over three quotes opens at i,
// a prefix (r, b, u, f) aside: the quote that closes it and the length of
// its opening, or 0.
func tripleQuote(content string, i int) (string, int) {
	j := i
	for j < len(content) && j-i < 2 && strings.ContainsRune("rRbBuUfF", rune(content[j])) {
		j++
	}
	if j > i && i > 0 && (unicode.IsLetter(rune(content[i-1])) || unicode.IsDigit(rune(content[i-1])) || content[i-1] == '_') {
		return "", 0 // a name ending in r, b, u or f
	}
	for _, q := range []string{`"""`, `'''`} {
		if strings.HasPrefix(content[j:], q) {
			return q, j - i + 3
		}
	}
	return "", 0
}

// closing finds where a string opened at i ends, or 0: on its line, a
// backslash escaping, but for a backtick, which may span lines.
func closing(content string, i int, q byte) int {
	for j := i + 1; j < len(content); j++ {
		switch content[j] {
		case '\\':
			if q != '`' {
				j++
			}
		case '\n':
			if q != '`' {
				return 0
			}
		case q:
			return j
		}
	}
	return 0
}
