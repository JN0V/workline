package documentalist

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Budgets are the size limits of the docs (roles/documentalist/role.yaml).
type Budgets struct {
	DocLines     int `json:"doc-lines"`
	SectionWords int `json:"section-words"`
	CardWords    struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"card-words"`
	FolderLines        int `json:"folder-lines"`
	RootAgentFileLines int `json:"root-agent-file-lines"`
}

// Duplicates says which repeated passages are worth reporting.
type Duplicates struct {
	MinWords   int     `json:"min-words"`
	Similarity float64 `json:"similarity"`
}

// rootAgentFiles are the entry points coding agents load at the root.
var rootAgentFiles = []string{"AGENTS.md", "CLAUDE.md"}

// Problem is what a hygiene check found. Key names the problem without its
// size, so the same problem is recognised before and after a patch; Size says
// how bad it is, so a patch that makes it worse is recognised too.
type Problem struct {
	Rule, Where, Message, Key string
	Size                      int
}

// Tree is what the hygiene checks look at: the docs' contents, and every file
// the repository tracks (links may point to any of them).
type Tree struct {
	Docs  map[string]string
	Files map[string]bool
}

// exists reports whether p is a tracked file or a folder holding one.
func (t Tree) exists(p string) bool {
	if t.Files[p] || p == "." {
		return true
	}
	for f := range t.Files {
		if strings.HasPrefix(f, p+"/") {
			return true
		}
	}
	return false
}

// Hygiene runs the checks that need no git history: budgets, duplicates and
// links. A budget that is not set is said, never skipped silently.
func Hygiene(t Tree, b Budgets, d Duplicates) []Problem {
	var out []Problem
	out = append(out, budgets(t, b)...)
	out = append(out, duplicates(t, d)...)
	out = append(out, links(t)...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Where < out[j].Where })
	return out
}

// line is one line of a doc, as the checks see it.
type line struct {
	n       int    // 1-based, in the whole file
	text    string // as written
	heading int    // 1-6 for a heading, else 0
	code    bool   // inside a fenced code block, or a fence itself
}

// scan splits a doc into lines, skipping its frontmatter.
func scan(content string) []line {
	all := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	_, start := header(content)
	var out []line
	fence := ""
	for i := start; i < len(all); i++ {
		l := line{n: i + 1, text: all[i]}
		t := strings.TrimSpace(all[i])
		switch {
		case fence != "":
			l.code = true
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			l.code, fence = true, t[:3]
		default:
			if lv := len(t) - len(strings.TrimLeft(t, "#")); lv > 0 && lv <= 6 && strings.HasPrefix(t[lv:], " ") {
				l.heading = lv
			}
		}
		out = append(out, l)
	}
	return out
}

// words counts the words of prose: tokens holding a letter or a digit, so
// list markers and table bars are not words.
func words(s string) int {
	n := 0
	for _, f := range strings.Fields(s) {
		if strings.IndexFunc(f, func(r rune) bool { return r > 127 || r >= '0' && r <= '9' || r|0x20 >= 'a' && r|0x20 <= 'z' }) >= 0 {
			n++
		}
	}
	return n
}

func lineCount(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(content, "\n"), "\n") + 1
}

func docType(content string) string {
	block, n := header(content)
	if n == 0 {
		return ""
	}
	var fm struct {
		Type string `yaml:"type"`
	}
	_ = yaml.Unmarshal([]byte(block), &fm) // a broken header is reported by ParseDoc
	return fm.Type
}

func budgets(t Tree, b Budgets) []Problem {
	var out []Problem
	missing := func(name string) {
		out = append(out, Problem{Rule: "setting-missing", Key: "setting-missing " + name,
			Message: fmt.Sprintf("budgets.%s is not set, so this size was not checked; set it, or turn the rule off with `enforce`", name)})
	}
	for name, v := range map[string]int{"doc-lines": b.DocLines, "section-words": b.SectionWords,
		"card-words.min": b.CardWords.Min, "card-words.max": b.CardWords.Max,
		"folder-lines": b.FolderLines, "root-agent-file-lines": b.RootAgentFileLines} {
		if v <= 0 {
			missing(name)
		}
	}
	folders := map[string]int{}
	for p, content := range t.Docs {
		n := lineCount(content)
		folders[path.Dir(p)] += n
		if b.DocLines > 0 && n > b.DocLines {
			out = append(out, Problem{Rule: "doc-too-long", Where: p, Key: "doc-too-long " + p, Size: n,
				Message: fmt.Sprintf("%d lines; the budget is %d: move whole parts into cards or a companion doc, and link to them", n, b.DocLines)})
		}
		lines := scan(content)
		if b.SectionWords > 0 {
			for _, s := range sections(lines) {
				if w := words(s.text); w > b.SectionWords {
					where := p
					if s.slug != "" {
						where += "#" + s.slug
					}
					out = append(out, Problem{Rule: "section-too-long", Where: where, Key: "section-too-long " + where, Size: w,
						Message: fmt.Sprintf("%d words; the budget is %d per section", w, b.SectionWords)})
				}
			}
		}
		if docType(content) == "card" {
			var all strings.Builder
			for _, l := range lines {
				if !l.code {
					all.WriteString(l.text + "\n")
				}
			}
			w := words(all.String())
			if b.CardWords.Min > 0 && w < b.CardWords.Min {
				out = append(out, Problem{Rule: "card-too-short", Where: p, Key: "card-too-short " + p, Size: b.CardWords.Min - w,
					Message: fmt.Sprintf("%d words; a card holds at least %d — merge it into the card it belongs with", w, b.CardWords.Min)})
			}
			if b.CardWords.Max > 0 && w > b.CardWords.Max {
				out = append(out, Problem{Rule: "card-too-long", Where: p, Key: "card-too-long " + p, Size: w,
					Message: fmt.Sprintf("%d words; a card holds at most %d — it covers more than one concept", w, b.CardWords.Max)})
			}
		}
	}
	if b.FolderLines > 0 {
		for dir, n := range folders {
			if n > b.FolderLines {
				out = append(out, Problem{Rule: "folder-too-long", Where: dir, Key: "folder-too-long " + dir, Size: n,
					Message: fmt.Sprintf("its docs hold %d lines; the budget is %d per folder", n, b.FolderLines)})
			}
		}
	}
	if b.RootAgentFileLines > 0 {
		for _, f := range rootAgentFiles {
			if content, ok := t.Docs[f]; ok {
				if n := lineCount(content); n > b.RootAgentFileLines {
					out = append(out, Problem{Rule: "agent-file-too-long", Where: f, Key: "agent-file-too-long " + f, Size: n,
						Message: fmt.Sprintf("%d lines; the budget is %d: an agent's entry point routes to docs, it does not hold them", n, b.RootAgentFileLines)})
				}
			}
		}
	}
	return out
}

type section struct{ slug, text string }

// sections splits a doc at every heading. A section's text is its own, up to
// the next heading of any level: a long chapter made of short sections is fine.
func sections(lines []line) []section {
	var out []section
	cur := section{}
	var b strings.Builder
	flush := func() {
		cur.text = b.String()
		if strings.TrimSpace(cur.text) != "" {
			out = append(out, cur)
		}
		b.Reset()
	}
	seen := map[string]int{}
	for _, l := range lines {
		if l.heading > 0 {
			flush()
			cur = section{slug: uniqueSlug(seen, strings.TrimSpace(strings.TrimSpace(l.text)[l.heading:]))}
			continue
		}
		if !l.code {
			b.WriteString(l.text + "\n")
		}
	}
	flush()
	return out
}

// uniqueSlug is slug with the suffix GitHub and GitLab add to repeated headings.
func uniqueSlug(seen map[string]int, heading string) string {
	s := slug(heading)
	n := seen[s]
	seen[s]++
	if n > 0 {
		return fmt.Sprintf("%s-%d", s, n)
	}
	return s
}

// anchors lists the anchors a doc's headings give.
func anchors(content string) map[string]bool {
	out := map[string]bool{}
	seen := map[string]int{}
	for _, l := range scan(content) {
		if l.heading > 0 {
			out[uniqueSlug(seen, strings.TrimSpace(strings.TrimSpace(l.text)[l.heading:]))] = true
		}
	}
	return out
}

// paragraph is a run of prose lines between blank lines, headings or code.
type paragraph struct {
	doc        string
	from, to   int
	shingles   map[string]bool
	wordsCount int
	text       string // its words, normalized: what identifies it wherever it moves
}

// duplicates finds passages written twice. Paragraphs are compared by the
// sets of three-word sequences they hold (Jaccard similarity), which catches a
// passage copied and then lightly edited; at the size of a project's docs,
// comparing every pair is fast enough.
func duplicates(t Tree, d Duplicates) []Problem {
	if d.MinWords <= 0 || d.Similarity <= 0 {
		return []Problem{{Rule: "setting-missing", Key: "setting-missing duplicates",
			Message: "duplicates.min-words or duplicates.similarity is not set, so repeated passages were not looked for"}}
	}
	var paras []paragraph
	docs := make([]string, 0, len(t.Docs))
	for p := range t.Docs {
		docs = append(docs, p)
	}
	sort.Strings(docs)
	for _, p := range docs {
		var cur []line
		flush := func() {
			if len(cur) > 0 {
				var text strings.Builder
				for _, l := range cur {
					text.WriteString(l.text + " ")
				}
				if ws := normalWords(text.String()); len(ws) >= d.MinWords {
					paras = append(paras, paragraph{doc: p, from: cur[0].n, to: cur[len(cur)-1].n, shingles: shingles(ws), wordsCount: len(ws), text: strings.Join(ws, " ")})
				}
			}
			cur = nil
		}
		for _, l := range scan(t.Docs[p]) {
			if l.code || l.heading > 0 || strings.TrimSpace(l.text) == "" {
				flush()
				continue
			}
			cur = append(cur, l)
		}
		flush()
	}
	var out []Problem
	for i := range paras {
		for j := i + 1; j < len(paras); j++ {
			a, b := paras[i], paras[j]
			if sim := jaccard(a.shingles, b.shingles); sim >= d.Similarity {
				out = append(out, Problem{Rule: "duplicate", Where: a.doc,
					Key: "duplicate " + pairKey(a.text, b.text),
					Message: fmt.Sprintf("lines %d-%d repeat %s lines %d-%d (%.0f%% alike): keep the passage in the one place it belongs, and link to it",
						a.from, a.to, b.doc, b.from, b.to, sim*100)})
			}
		}
	}
	return out
}

func normalWords(s string) []string {
	var out []string
	for _, f := range strings.Fields(strings.ToLower(s)) {
		w := strings.TrimFunc(f, func(r rune) bool { return !(r > 127 || r >= '0' && r <= '9' || r >= 'a' && r <= 'z') })
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

func shingles(ws []string) map[string]bool {
	out := map[string]bool{}
	for i := 0; i+3 <= len(ws); i++ {
		out[strings.Join(ws[i:i+3], " ")] = true
	}
	return out
}

func jaccard(a, b map[string]bool) float64 {
	inter := 0
	for s := range a {
		if b[s] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

var (
	inlineLink = regexp.MustCompile(`\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	refLink    = regexp.MustCompile(`^\s{0,3}\[[^\]]+\]:\s*<?(\S+?)>?(?:\s|$)`)
	codeSpan   = regexp.MustCompile("`[^`]*`")
	scheme     = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// links checks every link between files of the repository: the file must be
// tracked, and the heading an anchor names must exist. Links to other sites
// need the network; they are counted and said to be unchecked.
func links(t Tree) []Problem {
	var out []Problem
	for p, content := range t.Docs {
		external := 0
		for _, l := range scan(content) {
			if l.code {
				continue
			}
			text := codeSpan.ReplaceAllString(l.text, "")
			var targets []string
			for _, m := range inlineLink.FindAllStringSubmatch(text, -1) {
				targets = append(targets, m[1])
			}
			if m := refLink.FindStringSubmatch(text); m != nil {
				targets = append(targets, m[1])
			}
			for _, target := range targets {
				if scheme.MatchString(target) {
					if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
						external++
					}
					continue
				}
				if why := deadLink(t, p, target); why != "" {
					out = append(out, Problem{Rule: "dead-link", Where: p, Key: "dead-link " + p + " " + target,
						Message: fmt.Sprintf("line %d links to %s: %s", l.n, target, why)})
				}
			}
		}
		if external > 0 {
			out = append(out, Problem{Rule: "links-not-checked", Where: p, Key: "links-not-checked " + p, Size: external,
				Message: fmt.Sprintf("%d link(s) to other sites not checked: that needs the network, and no link checker is wired in yet", external)})
		}
	}
	return out
}

// deadLink says why a link inside the repository leads nowhere; empty if it is fine.
func deadLink(t Tree, from, target string) string {
	file, anchor, _ := strings.Cut(target, "#")
	file, _, _ = strings.Cut(file, "?")
	if u, err := url.PathUnescape(file); err == nil {
		file = u
	}
	switch {
	case file == "":
		file = from
	case strings.HasPrefix(file, "/"):
		file = path.Clean(strings.TrimPrefix(file, "/"))
	default:
		file = path.Join(path.Dir(from), file)
	}
	if strings.HasPrefix(file, "../") || file == ".." {
		return "it points outside the repository"
	}
	if !t.exists(file) {
		return "no such file"
	}
	if anchor == "" || !strings.HasSuffix(file, ".md") {
		return ""
	}
	content, ok := t.Docs[file]
	if !ok {
		return "" // not a doc the role reads; its headings are not checked
	}
	if !anchors(content)[anchor] {
		return "no heading gives the anchor #" + anchor
	}
	return ""
}

// pairKey names a pair of passages by their text, in a stable order, so a
// duplicate moved to another doc is still the same duplicate.
func pairKey(a, b string) string {
	if b < a {
		a, b = b, a
	}
	return a + " | " + b
}
