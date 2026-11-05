package codepick

// The names a text points to, read as CommonMark reads it (#425): moved
// from the reviewer's judge, which reads a finding's names by it, so the
// product owner, the tester and the reviewer read an issue's alike.

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
	"github.com/JN0V/workline/internal/work"
)

// PathWord is a word that may name a file: letters, digits and `._-/`,
// the punctuation closing a sentence left.
var PathWord = regexp.MustCompile(`[\w.\-/]+`)

// NamedIn are the names a text's words point to, its code read as
// CommonMark reads it (#425). code, the words written as code: every word
// of a code span; a word of a fenced block called or holding a capital or
// an underscore; in the prose, one followed by a parenthesis, or holding
// an underscore or a capital past its first letter. capital, a capitalised
// word inside a sentence; weak, one opening it, a line or a bold label:
// both English as often as a name. A path, a file's name or a flag names
// none.
func NamedIn(text string) (code, capital, weak []string) {
	add := func(to *[]string, w string) {
		if len(w) > 2 && !slices.Contains(*to, w) {
			*to = append(*to, w)
		}
	}
	text, spans, blocks := markdownCode(text)
	for _, s := range spans {
		for _, w := range codeWords(s) {
			add(&code, w.name)
		}
	}
	for _, s := range blocks {
		for _, w := range codeWords(s) {
			if w.call || strings.ToLower(w.name) != w.name || strings.Contains(w.name, "_") {
				add(&code, w.name)
			}
		}
	}
	text = PathWord.ReplaceAllStringFunc(text, func(t string) string {
		if p := strings.TrimRight(t, "."); APath(p) {
			return "``" + t[len(p):] // code, as a span is
		}
		return t
	})
	for _, at := range regexp.MustCompile(`\b([A-Za-z_]\w*)(\()?`).FindAllStringSubmatchIndex(text, -1) {
		w := text[at[2]:at[3]]
		before := strings.TrimRight(text[:at[0]], " \t*_#>+-") // a list's mark, a heading's, a bold label's
		opens := before == "" || strings.ContainsAny(before[len(before)-1:], ".:;!?\n")
		switch {
		case at[4] >= 0 || strings.Contains(w, "_") || strings.ToLower(w[1:]) != w[1:]:
			add(&code, w)
		case strings.ToLower(w) == w:
		case opens:
			add(&weak, w)
		default:
			add(&capital, w)
		}
	}
	capital = slices.DeleteFunc(capital, func(w string) bool { return slices.Contains(code, w) })
	weak = slices.DeleteFunc(weak, func(w string) bool { return slices.Contains(code, w) || slices.Contains(capital, w) })
	return code, capital, weak
}

// markdownCode splits a text as CommonMark reads it: its fenced blocks,
// its code spans, and its prose, a block left as a blank line, a span as
// two backticks. A span opens on a run of backticks and closes on the next
// run of as many, in its paragraph; a run left unclosed is prose. Lines
// indented as code, read as prose: a list item's lines are indented too.
func markdownCode(text string) (prose string, spans, blocks []string) {
	var out, para, block []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		p, s := codeSpans(strings.Join(para, "\n"))
		out, spans, para = append(out, p), append(spans, s...), nil
	}
	fence := ""
	for _, l := range strings.Split(text, "\n") {
		was := fence
		fence = work.Fenced(l, fence)
		switch {
		case was == "" && fence == "":
			if strings.TrimSpace(l) == "" {
				flush()
				out = append(out, "")
				continue
			}
			para = append(para, l)
		case was == "": // its opening line: an info string, not code
			flush()
			out = append(out, "")
		case fence == "":
			blocks, block = append(blocks, strings.Join(block, "\n")), nil
		default:
			block = append(block, l)
		}
	}
	if fence != "" { // open to the text's end
		blocks = append(blocks, strings.Join(block, "\n"))
	}
	flush()
	return strings.Join(out, "\n"), spans, blocks
}

// codeSpans are a paragraph's code spans, and the paragraph each left as
// two backticks.
func codeSpans(p string) (string, []string) {
	run := func(i int) int {
		n := 0
		for i+n < len(p) && p[i+n] == '`' {
			n++
		}
		return n
	}
	var b strings.Builder
	var spans []string
	for i := 0; i < len(p); {
		if p[i] != '`' {
			b.WriteByte(p[i])
			i++
			continue
		}
		n, end := run(i), -1
		for j := i + n; j < len(p) && end < 0; {
			switch m := run(j); m {
			case 0:
				j++
			case n:
				end = j
			default:
				j += m
			}
		}
		if end < 0 {
			b.WriteString(p[i : i+n])
			i += n
			continue
		}
		spans = append(spans, p[i+n:end])
		b.WriteString("``")
		i = end + n
	}
	return b.String(), spans
}

// codeWord is a word of code, and whether it is called.
type codeWord struct {
	name string
	call bool
}

var identRe = regexp.MustCompile(`^[A-Za-z_]\w*$`)

// codeWords are the words of some code that may name a function: each part
// of `a.b`, the last one called when a parenthesis follows. Not called, a
// path, a file's name and a word holding `-` name none: a flag (`--all`),
// a key (`code-lines-max`), a header (`Content-Type`); called, a function
// (`size-count(v)` names `count`).
func codeWords(code string) []codeWord {
	var out []codeWord
	for _, m := range PathWord.FindAllStringIndex(code, -1) {
		t := strings.TrimRight(code[m[0]:m[1]], ".")
		call := strings.HasPrefix(strings.TrimLeft(code[m[1]:], " \t"), "(")
		if !call && (APath(t) || strings.Contains(t, "-")) {
			continue
		}
		parts := strings.FieldsFunc(t, func(r rune) bool { return r == '.' || r == '-' || r == '/' })
		for k, w := range parts {
			if identRe.MatchString(w) {
				out = append(out, codeWord{w, call && k == len(parts)-1})
			}
		}
	}
	return out
}

// docFile is a file's name by an extension a text names often and holds no
// function.
var docFile = regexp.MustCompile(`\w\.(?:md|txt|ya?ml|json|toml|xml|html?|css|csv)$`)

// APath says whether a word names a file or a folder, not a function: it
// holds a `/`, or ends in a file's extension, in lower case (`a.C` is a
// field).
func APath(w string) bool {
	ext := path.Ext(w)
	return strings.Contains(w, "/") || docFile.MatchString(w) || (ext != "" && ext == strings.ToLower(ext) && codenav.Of(w) != nil)
}
