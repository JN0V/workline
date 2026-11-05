package codenav

import (
	"regexp"
	"slices"
	"strings"
)

// Elided marks the lines of a body left out of an outline.
const Elided = "…"

// Containers are the words that open a block whose members are
// declarations — a type, a namespace, a module — in any language: the
// outline keeps such a block's lines, and leaves out every other block,
// a function's body above all (docs/research/code-outline.md). Data, not
// a parser: a language whose word is missing here loses its members to
// `…`, never shows a body.
var Containers = []string{"class", "struct", "union", "namespace", "interface", "enum", "trait", "impl",
	"mod", "module", "object", "protocol", "extension", "record", "type", "extern", "mixin"}

var (
	// containerRe is a container word opening a declaration: a name, a
	// brace or a type's parameters after it (`class A`, `struct {`,
	// `impl<T>`), never `module.exports` nor `var record =`.
	containerRe = regexp.MustCompile(`\b(?:` + strings.Join(Containers, "|") + `)(?:\s+[A-Za-z_]|\s*[{<]|\s*$)`)
	// annotation is a Java or Kotlin annotation, a C++ attribute: its
	// parenthesis says nothing of the block after it.
	annotation = regexp.MustCompile(`@[\w.]+(?:\([^)]*\))?|\[\[[^\]]*\]\]`)
)

// Outlined is a file's declarations, its bodies left out: the lines kept,
// and each body cut, from the line it opens on to the line it ends on.
type Outlined struct {
	Keep   []bool      // Keep[n-1]: line n is kept
	Bodies map[int]int // a body opening on line n ends on line Bodies[n]
	Braces bool        // a body ends on a brace, shown after its `…`
}

// Outline is a file's declarations, its bodies left out: each block that
// is not a container — a function's body, an initializer — is cut to its
// opening line and `…`; a line holding only a comment is left out too.
// Nil when the file's language has no outline (Ruby and Lua close their
// blocks by a word; shell's are few). By braces for C-like languages and
// Go, by indentation for Python: one rule a family, no parser.
func Outline(file string, lines []string) *Outlined {
	l := Of(file)
	if l == nil {
		return nil
	}
	switch {
	case l.Name == "shell":
		return nil
	case l.Body == "go" || l.Body == "braces":
		return braceOutline(lines)
	case l.Body == "indent":
		return indentOutline(lines)
	}
	return nil
}

// braceOutline keeps the lines outside any body: a block opened by a
// header holding a parenthesis (a function, a call, a control word) or
// no container word (an initializer) is a body.
func braceOutline(lines []string) *Outlined {
	o := &Outlined{Keep: make([]bool, len(lines)), Bodies: map[int]int{}, Braces: true}
	var stack []bool // true: a body
	bodies := 0      // bodies open
	header := ""     // the code since the last `;`, `{` or `}`
	open := 0        // the line the outermost body opened on
	inComment := false
	for i, raw := range lines {
		bare, still := stripBlockComments(raw, inComment)
		inComment = still
		code := StripCode(bare, false)
		startIn := bodies > 0
		pre := strings.HasPrefix(strings.TrimSpace(code), "#")
		if pre {
			code = "" // a preprocessor line: kept, no header
		}
		for _, c := range code {
			switch c {
			case '{':
				body := bodies > 0 || isBody(header)
				stack = append(stack, body)
				if body {
					bodies++
				}
				header = ""
			case '}':
				if n := len(stack); n > 0 {
					if stack[n-1] {
						bodies--
					}
					stack = stack[:n-1]
				}
				header = ""
			case ';':
				header = ""
			default:
				header += string(c)
			}
		}
		header += " "
		// A line ending a call or a group, or blank, ends a header too:
		// Go writes no `;`, and `import (…)` is not the next type's.
		if t := strings.TrimSpace(code); strings.TrimSpace(raw) == "" || !pre && strings.HasSuffix(t, ")") {
			header = ""
		}
		switch {
		case startIn:
			if bodies == 0 {
				o.Bodies[open] = i + 1
			}
		case strings.TrimSpace(bare) == "" && strings.TrimSpace(raw) != "":
			// a comment's line: the declarations, not their prose
		default:
			o.Keep[i] = true
			if bodies > 0 {
				open = i + 1
			}
		}
	}
	if bodies > 0 && open > 0 { // a body the file does not close
		o.Bodies[open] = len(lines)
	}
	return o
}

// isBody says whether a block's header opens a body rather than a
// container: a parenthesis (a function's parameters, a condition), or
// no container word (`= {`, an initializer).
func isBody(header string) bool {
	h := annotation.ReplaceAllString(header, "")
	if strings.Contains(h, "(") || strings.Contains(h, "=>") {
		return true
	}
	return !containerRe.MatchString(h)
}

// indentOutline keeps the lines outside any body by their indentation: a
// header ending in `:` opens a block, a container when its first word is
// one (`class`), else a body cut to its header and `…`.
func indentOutline(lines []string) *Outlined {
	o := &Outlined{Keep: make([]bool, len(lines)), Bodies: map[int]int{}}
	depth := 0  // brackets open
	start := -1 // the first line of a header spread over lines
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		t := strings.TrimSpace(raw)
		if strings.HasPrefix(t, "#") && depth == 0 {
			continue // a comment's line: the declarations, not their prose
		}
		o.Keep[i] = true
		if depth == 0 {
			start = i
		}
		code := StripCode(raw, true)
		for _, c := range code {
			switch c {
			case '(', '[', '{':
				depth++
			case ')', ']', '}':
				depth = max(depth-1, 0)
			}
		}
		if depth > 0 || !strings.HasSuffix(strings.TrimSpace(code), ":") {
			continue
		}
		word := strings.Fields(strings.TrimSpace(lines[start]))
		if len(word) > 0 && word[0] != "type" && slices.Contains(Containers, word[0]) {
			continue
		}
		hi := len(indentOf(lines[start]))
		end := i
		for n := i + 1; n < len(lines); n++ {
			if s := strings.TrimSpace(lines[n]); s != "" {
				if len(indentOf(lines[n])) <= hi {
					break
				}
				end = n
			}
		}
		if end > i {
			o.Bodies[i+1] = end + 1
			i = end
		}
	}
	return o
}

// Render shows the outline: each line kept, a body as its opening line,
// `…`, and its closing brace; full(n) lines shown whole, wherever they
// lie, and a body holding one shown in full.
func (o *Outlined) Render(lines []string, full func(n int) bool) []string {
	var out []string
	blank := false
	emit := func(s string) {
		if strings.TrimSpace(s) == "" {
			if blank || len(out) == 0 {
				return
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, s)
	}
	for n := 1; n <= len(lines); n++ {
		switch {
		case full(n):
			end, ok := o.Bodies[n]
			if !ok || !o.Keep[n-1] {
				emit(lines[n-1])
				continue
			}
			for k := n; k <= end; k++ { // a body whose opening is shown whole: shown whole
				emit(lines[k-1])
			}
			n = end
		case !o.Keep[n-1]:
		default:
			end, ok := o.Bodies[n]
			if !ok {
				emit(lines[n-1])
				continue
			}
			if holds(full, n+1, end) {
				for k := n; k <= end; k++ { // a body holding a line shown whole: shown whole
					emit(lines[k-1])
				}
				n = end
				continue
			}
			line := strings.TrimRight(lines[n-1], " \t") + " " + Elided
			if c := StripCode(lines[end-1], false); o.Braces && strings.Contains(c, "}") {
				line += " " + strings.TrimSpace(lines[end-1][strings.LastIndex(lines[end-1], "}"):])
			}
			emit(line)
			n = end
		}
	}
	return out
}

// holds says whether full holds a line from to.
func holds(full func(int) bool, from, to int) bool {
	for n := from; n <= to; n++ {
		if full(n) {
			return true
		}
	}
	return false
}

func indentOf(s string) string { return s[:len(s)-len(strings.TrimLeft(s, " \t"))] }

// stripBlockComments drops a line's comments, `//` and `/* … */`, its
// strings kept; in says whether
// the line starts inside one, still whether it ends inside one.
func stripBlockComments(line string, in bool) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case in:
			if strings.HasPrefix(line[i:], "*/") {
				in = false
				i++
			}
		case strings.HasPrefix(line[i:], "/*"):
			in = true
			i++
		case strings.HasPrefix(line[i:], "//"):
			return b.String(), false
		case line[i] == '"' || line[i] == '\'' || line[i] == '`':
			q := line[i]
			b.WriteByte(q)
			for i++; i < len(line); i++ {
				b.WriteByte(line[i])
				if line[i] == '\\' && i+1 < len(line) {
					i++
					b.WriteByte(line[i])
					continue
				}
				if line[i] == q {
					break
				}
			}
		default:
			b.WriteByte(line[i])
		}
	}
	return b.String(), in
}
