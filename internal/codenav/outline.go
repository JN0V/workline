package codenav

import (
	"slices"
	"strings"
)

// Elided marks the lines of a body left out of an outline.
const Elided = "…"

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
// By braces or by indentation, the language's row of Languages saying
// its containers, what a header holds that says nothing, and the rest
// (docs/research/code-outline.md). Nil when the language has none
// (shell, Ruby, Lua), or when its braces do not balance — a regular
// expression's, a branch of `#if` each opening one: the file is then
// given whole or cut, never a body shown for a declaration.
func Outline(file string, lines []string) *Outlined {
	l := Of(file)
	if l == nil {
		return nil
	}
	switch l.Outline {
	case "braces":
		return braceOutline(l, lines)
	case "indent":
		return indentOutline(l, lines)
	}
	return nil
}

// braceOutline keeps the lines outside any body: a block whose header
// holds no container word before its parameters — a function, a call, a
// control word, an initializer — is a body.
func braceOutline(l *Language, lines []string) *Outlined {
	o := &Outlined{Keep: make([]bool, len(lines)), Bodies: map[int]int{}, Braces: true}
	var stack []bool // true: a body
	bodies := 0      // bodies open
	header := ""     // the code since the last `;`, `{` or `}`
	parens := 0      // parentheses open in the header
	open := 0        // the line the outermost body opened on
	inComment := false
	for i, raw := range lines {
		bare, still := stripBlockComments(raw, inComment)
		inComment = still
		code := StripCode(bare, false)
		startIn := bodies > 0
		pre := l.Preproc && strings.HasPrefix(strings.TrimSpace(code), "#")
		if pre {
			code = "" // a preprocessor line: kept, no header
		}
		for _, c := range code {
			switch c {
			case '{':
				body := bodies > 0 || l.isBody(header)
				stack = append(stack, body)
				if body {
					bodies++
				}
				header, parens = "", 0
			case '}':
				n := len(stack)
				if n == 0 {
					return nil // more closed than opened: not read right
				}
				if stack[n-1] {
					bodies--
				}
				stack = stack[:n-1]
				header, parens = "", 0
			case ';':
				header, parens = "", 0
			default:
				switch c {
				case '(':
					parens++
				case ')':
					parens = max(parens-1, 0)
				}
				header += string(c)
			}
		}
		header += " "
		// A blank line, a line ending a call or a group, or, where a line
		// ends a statement, any line with no parenthesis open, ends a
		// header: `import (…)` is not the next type's.
		t := strings.TrimSpace(code)
		switch {
		case strings.TrimSpace(raw) == "":
			header, parens = "", 0
		case pre || parens > 0:
		case strings.HasSuffix(t, ")"), l.LineEnds && !strings.HasSuffix(t, ",") && !strings.HasSuffix(t, "="):
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
	if len(stack) > 0 || inComment {
		return nil // a block the file does not close: not read right
	}
	return o
}

// isBody says whether a block's header opens a body rather than a
// container: no container word before its first closed parenthesis or
// arrow (`class P(val x: Int) : B()` is a container, `void f(struct S
// s)` a body), what says nothing (an annotation, an attribute, a
// template's head) left out first.
func (l *Language) isBody(header string) bool {
	h := header
	if l.ignoreRe != nil {
		h = l.ignoreRe.ReplaceAllString(h, " ")
	}
	at := len(h) // the first parenthesis closed in the header, or an arrow
	if i := strings.Index(h, "=>"); i >= 0 {
		at = i
	}
	depth, from := 0, -1
	for i, c := range h {
		if c == '(' {
			if depth == 0 {
				from = i
			}
			depth++
		} else if c == ')' && depth > 0 {
			if depth--; depth == 0 && from < at {
				at = from
				break
			}
		}
	}
	if l.containerRe == nil {
		return true
	}
	m := l.containerRe.FindStringIndex(h)
	return m == nil || m[0] > at
}

// indentOutline keeps the lines outside any body by their indentation: a
// header ending in `:` whose first word opens a body (Python's `def`) is
// cut to its header and `…`; a container (`class`) or another block
// (`if TYPE_CHECKING:`) is kept. A docstring's lines are never headers.
func indentOutline(l *Language, lines []string) *Outlined {
	o := &Outlined{Keep: make([]bool, len(lines)), Bodies: map[int]int{}}
	depth := 0  // brackets open
	start := -1 // the first line of a header spread over lines
	doc := false
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		t := strings.TrimSpace(raw)
		inDoc := doc
		doc = doc != (tripleQuotes(raw)%2 == 1)
		if !inDoc && strings.HasPrefix(t, "#") && depth == 0 {
			continue // a comment's line: the declarations, not their prose
		}
		o.Keep[i] = true
		if inDoc || doc {
			continue
		}
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
		if len(word) == 0 || !slices.Contains(l.Bodies, strings.TrimSuffix(word[0], ":")) {
			continue
		}
		hi := len(indentOf(lines[start]))
		end, inBody := i, false
		for n := i + 1; n < len(lines); n++ {
			was := inBody
			inBody = inBody != (tripleQuotes(lines[n])%2 == 1)
			if s := strings.TrimSpace(lines[n]); s != "" {
				if !was && len(indentOf(lines[n])) <= hi {
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

// tripleQuotes counts a line's triple quotes, double and single.
func tripleQuotes(line string) int {
	return strings.Count(line, strings.Repeat(`"`, 3)) + strings.Count(line, strings.Repeat("'", 3))
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
