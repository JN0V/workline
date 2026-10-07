package documentalist

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// The reader checks (#193): whether a user page serves its reader, beside
// whether it is true. Computed with no agent, when gardening: a decision,
// doc or issue named without a link, a paragraph or a table cell too long
// to be read at a glance, a page explaining a flow with no diagram, a user
// page no navigation reaches. Reported, never blocking: a proposal for the
// page, not a verdict on the change.

// Reader says which docs are user pages and how much a reader takes at once
// (roles/documentalist/role.yaml, `reader`).
type Reader struct {
	// Pages: the user pages, among the docs the role reads; Skip: those left
	// out (specs). Records — decisions, research, history — are always left
	// out: they are read for what was true then, not as a guide.
	Pages []string `json:"pages"`
	Skip  []string `json:"skip"`
	// Navigation: the pages a reader starts from; every user page must be
	// reached from one, link after link through user pages.
	Navigation []string `json:"navigation"`
	// Flow: pages that explain a flow and need a diagram, beside any page
	// with a "How it works" heading.
	Flow           []string `json:"flow"`
	ParagraphWords int      `json:"paragraph-words"`
	CellWords      int      `json:"cell-words"`
}

// userPages are the docs the reader checks look at, sorted.
func userPages(t Tree, r Reader) []string {
	var out []string
	for p := range t.Docs {
		if matchAny(r.Pages, p) && !matchAny(r.Skip, p) && !isHistory(p) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// ReaderChecks runs the reader checks on the user pages. A threshold that is
// not set is said, never skipped silently.
func ReaderChecks(t Tree, r Reader) []Problem {
	var out []Problem
	for _, set := range []struct {
		name string
		v    int
	}{{"paragraph-words", r.ParagraphWords}, {"cell-words", r.CellWords}} {
		if name, v := set.name, set.v; v <= 0 {
			out = append(out, Problem{Rule: "setting-missing", Key: "setting-missing reader." + name,
				Message: fmt.Sprintf("reader.%s is not set, so this size was not checked; set it, or turn the rule off with `enforce`", name)})
		}
	}
	pages := userPages(t, r)
	for _, p := range pages {
		lines := scan(t.Docs[p])
		out = append(out, unlinkedRefs(t, p, lines)...)
		if r.ParagraphWords > 0 {
			out = append(out, longParagraphs(p, lines, r.ParagraphWords)...)
		}
		if r.CellWords > 0 {
			out = append(out, longCells(p, lines, r.CellWords)...)
		}
		if why := explainsFlow(p, lines, r.Flow); why != "" && !hasDiagram(lines) {
			out = append(out, Problem{Rule: "flow-without-diagram", Where: p, Key: "flow-without-diagram " + p,
				Message: fmt.Sprintf("%s, and holds no diagram: open it with one — a few blocks, a few words each — and keep the detail in the text below", why)})
		}
	}
	out = append(out, unreachable(t, r, pages)...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Where < out[j].Where })
	return out
}

var (
	// A link, its text included: what it says is linked; the end of one
	// whose text began on the line above.
	wholeLink = regexp.MustCompile(`!?\[[^\]]*\]\([^)]*\)|\[[^\]]*\]\[[^\]]*\]|\]\([^)]*\)|<[a-z]+://[^>]*>|https?://\S+`)
	adrRef    = regexp.MustCompile(`\bADR[- ]?\d+\b`)
	issueRef  = regexp.MustCompile(`(?:^|[\s(,;:])(#\d+)\b`)
	docRef    = regexp.MustCompile(`[\w./-]*\w\.md\b`)
)

// unlinkedRefs are the decisions, docs and issues a page names without a
// link: a reader cannot follow ADR-0026, docs/x.md or #128. A doc is named
// when the name is a file of the repository, from the page or the root; a
// name in a code span is an example, not a reference.
func unlinkedRefs(t Tree, p string, lines []line) []Problem {
	var found []string
	n := 0
	var hidden hiddenText
	for _, l := range lines {
		if l.code || refLink.MatchString(l.text) { // a link's definition is the link
			continue
		}
		text := wholeLink.ReplaceAllString(hidden.strip(l.text), " ")
		var refs []string
		refs = append(refs, adrRef.FindAllString(text, -1)...)
		for _, m := range issueRef.FindAllStringSubmatch(text, -1) {
			refs = append(refs, m[1])
		}
		for _, m := range docRef.FindAllString(text, -1) {
			for _, f := range []string{path.Join(path.Dir(p), m), path.Clean(m)} {
				if f != p && t.Files[f] {
					refs = append(refs, m)
					break
				}
			}
		}
		for _, ref := range refs {
			n++
			if len(found) < 5 {
				found = append(found, fmt.Sprintf("line %d `%s`", l.n, ref))
			}
		}
	}
	if n == 0 {
		return nil
	}
	more := ""
	if n > len(found) {
		more = fmt.Sprintf(" and %d more", n-len(found))
	}
	return []Problem{{Rule: "reference-unlinked", Where: p, Key: "reference-unlinked " + p, Size: n,
		Message: fmt.Sprintf("%d decision, doc or issue named without a link: %s%s; make each a link the reader can follow", n, strings.Join(found, ", "), more)}}
}

// hiddenText takes out of prose, line after line, what a reader does not
// read as a reference: HTML comments and code spans (examples), either of
// which may run over several lines. A code span ends with its paragraph.
type hiddenText struct {
	comment bool
	span    int // the backticks the open code span closes with; 0 when none is open
}

func (h *hiddenText) strip(text string) string {
	if strings.TrimSpace(text) == "" {
		h.span = 0
	}
	var out strings.Builder
	for text != "" {
		switch {
		case h.comment:
			i := strings.Index(text, "-->")
			if i < 0 {
				return out.String()
			}
			text, h.comment = text[i+3:], false
		case h.span > 0:
			i := closingRun(text, h.span)
			if i < 0 {
				return out.String()
			}
			text, h.span = text[i+h.span:], 0
		default:
			c, b := strings.Index(text, "<!--"), strings.IndexByte(text, '`')
			switch {
			case c < 0 && b < 0:
				out.WriteString(text)
				return out.String()
			case c >= 0 && (b < 0 || c < b):
				out.WriteString(text[:c] + " ")
				text, h.comment = text[c+4:], true
			default:
				out.WriteString(text[:b] + " ")
				n := len(text[b:]) - len(strings.TrimLeft(text[b:], "`"))
				text, h.span = text[b+n:], n
			}
		}
	}
	return out.String()
}

// closingRun is where a run of exactly n backticks starts in text, or -1.
func closingRun(text string, n int) int {
	for i := 0; i < len(text); {
		if text[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(text) && text[j] == '`' {
			j++
		}
		if j-i == n {
			return i
		}
		i = j
	}
	return -1
}

var (
	listItem  = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s`)
	linkURL   = regexp.MustCompile(`\]\([^)]*\)`)
	tableLine = regexp.MustCompile(`^\s*\|`)
	tableRule = regexp.MustCompile(`^\s*\|?[\s:|-]+$`)
)

// longParagraphs are the paragraphs, and the list items, over max words: a
// list item is a paragraph of its own.
func longParagraphs(p string, lines []line, max int) []Problem {
	type block struct{ from, words int }
	var over []block
	cur := block{}
	var text strings.Builder
	flush := func() {
		if w := words(linkURL.ReplaceAllString(text.String(), "]")); w > max {
			over = append(over, block{cur.from, w})
		}
		text.Reset()
		cur = block{}
	}
	for _, l := range lines {
		t := strings.TrimSpace(l.text)
		if l.code || l.heading > 0 || t == "" || tableLine.MatchString(l.text) || strings.HasPrefix(t, "<") {
			flush()
			continue
		}
		if listItem.MatchString(l.text) {
			flush()
		}
		if text.Len() == 0 {
			cur.from = l.n
		}
		text.WriteString(strings.TrimLeft(t, "> ") + "\n")
	}
	flush()
	if len(over) == 0 {
		return nil
	}
	var at []string
	for _, b := range over {
		at = append(at, fmt.Sprintf("line %d (%d words)", b.from, b.words))
	}
	return []Problem{{Rule: "paragraph-too-long", Where: p, Key: "paragraph-too-long " + p, Size: len(over),
		Message: fmt.Sprintf("%d paragraph(s) over %d words: %s; break each into bullets, or move the detail to a page below", len(over), max, strings.Join(at, ", "))}}
}

// longCells are the table cells over max words: a table is read across, one
// short sentence a cell.
func longCells(p string, lines []line, max int) []Problem {
	var at []string
	for _, l := range lines {
		if l.code || !tableLine.MatchString(l.text) || tableRule.MatchString(l.text) {
			continue
		}
		row := strings.ReplaceAll(strings.Trim(strings.TrimSpace(l.text), "|"), `\|`, "")
		for _, cell := range strings.Split(row, "|") {
			if w := words(linkURL.ReplaceAllString(cell, "]")); w > max {
				at = append(at, fmt.Sprintf("line %d (%d words)", l.n, w))
			}
		}
	}
	if len(at) == 0 {
		return nil
	}
	return []Problem{{Rule: "cell-too-long", Where: p, Key: "cell-too-long " + p, Size: len(at),
		Message: fmt.Sprintf("%d table cell(s) over %d words: %s; keep one short sentence and a link, the detail on the page it links to", len(at), max, strings.Join(at, ", "))}}
}

var howItWorks = regexp.MustCompile(`(?i)\bhow it works\b`)

// explainsFlow says why a page explains a flow, or "" when it does not.
func explainsFlow(p string, lines []line, flow []string) string {
	if matchAny(flow, p) {
		return "it explains a flow (`reader.flow`)"
	}
	for _, l := range lines {
		if l.heading > 0 && howItWorks.MatchString(l.text) {
			return fmt.Sprintf("its heading at line %d explains how it works", l.n)
		}
	}
	return ""
}

var (
	diagramFence = regexp.MustCompile("^\\s*(?:```|~~~)\\s*\\{?\\s*(?:mermaid|plantuml|puml|dot|graphviz|d2|ditaa|svgbob|pikchr|nomnoml)\\b")
	imageRef     = regexp.MustCompile(`(?i)!\[[^\]]*\]\([^)]*\.(?:svg|png|jpe?g|gif|webp)[^)]*\)|<img\s`)
)

// hasDiagram says whether a page holds a diagram: a diagram's fenced code,
// or an image.
func hasDiagram(lines []line) bool {
	for _, l := range lines {
		if diagramFence.MatchString(l.text) || !l.code && imageRef.MatchString(l.text) {
			return true
		}
	}
	return false
}

// unreachable are the user pages no navigation reaches: from each entry
// point, link after link through user pages, as a reader clicks.
func unreachable(t Tree, r Reader, pages []string) []Problem {
	if len(pages) == 0 {
		return nil
	}
	user := map[string]bool{}
	for _, p := range pages {
		user[p] = true
	}
	var queue []string
	reached := map[string]bool{}
	for _, entry := range r.Navigation {
		if _, ok := t.Docs[entry]; ok && !reached[entry] {
			reached[entry] = true
			queue = append(queue, entry)
		}
	}
	if len(queue) == 0 {
		return []Problem{{Rule: "navigation-missing", Key: "navigation-missing",
			Message: fmt.Sprintf("no page of reader.navigation %v is a doc of the repository, so the pages no navigation reaches were not looked for", r.Navigation)}}
	}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, l := range scan(t.Docs[p]) {
			if l.code {
				continue
			}
			for _, target := range linkTargets(codeSpan.ReplaceAllString(l.text, "")) {
				if scheme.MatchString(target) {
					continue
				}
				file, _ := resolve(p, target)
				for _, f := range []string{file, path.Join(file, "README.md"), path.Join(file, "index.md")} {
					if user[f] && !reached[f] {
						reached[f] = true
						queue = append(queue, f)
					}
				}
			}
		}
	}
	var out []Problem
	for _, p := range pages {
		if !reached[p] {
			out = append(out, Problem{Rule: "page-unreachable", Where: p, Key: "page-unreachable " + p,
				Message: fmt.Sprintf("no link reaches this page from %s through the user pages: link it from the page a reader would look for it on, or leave it out of reader.pages", strings.Join(r.Navigation, ", "))})
		}
	}
	return out
}
