package documentalist

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Why a fix takes words out (ADR-0014, step 2). The agent removed a true
// claim it could not see backed (workline d43b3f2), and followed a stale
// code comment over the code's setting (#29). So every word a fix takes out
// of a doc's body is cited: a `claim` beside the patch, giving the doc's
// lines and either a source's words, quoted, that the engine finds in a file
// of the doc's sources outside its comments, or a name the engine finds gone
// from the code. A quote found only in a comment is not evidence: the fix is
// refused, and the disagreement reported for a person. Words only added need
// no claim. Condensing, deduplicating, merging and splitting are judged
// elsewhere: they move text, and their judge checks where it went.

// citeTask is what a task judging docs tells the agent of it.
const citeTask = `Every word a patch takes out of a doc's body — a line removed, a value
replaced, a phrase dropped — is cited, or the patch is refused: beside the
patch, one ` + "`claim`" + ` for each place, giving the doc's lines as numbered here and
why the words go — a source's words, quoted exactly, from a file under the
doc's sources, or a name the code no longer has:

    - claim:
        doc: docs/example.md
        lines: "12"
        status: contradicted
        source:
          path: src/example.go
          quote: "const TTL = 7200"
    - claim:
        doc: docs/example.md
        lines: "15"
        status: gone
        name: RefreshToken

A comment in the code is not evidence: a quote found only in a comment is
refused, and the disagreement reported for a person. With nothing else to
cite, leave the words as they are and say in a ` + "`note`" + ` what you could not
confirm. A line a diff shown here adds is in the file as it is now: quote it.
Words only added need no claim. Nor does a line count the engine lists as off
for a doc, a table's total among them, brought to the engine's number: the
engine's count is the evidence; the words around it may then be said anew,
but not a fact of the line's own: another number, a version, a name, a "not",
an "all", an "or", a tense taken out still need a claim. Nor do words like
"the" or "per", taken out of a line whose other words stay.

`

// citedBlock is one run of changed lines of a doc's body: the old lines it
// takes out, from..to (to < from for lines only added, after from-1), and
// the lines it writes in their place.
type citedBlock struct {
	from, to       int
	removed, added []string
	hunk           int   // the hunk it is in, by index
	at             []int // its lines in that hunk, by index
}

// bodyBlocks splits a placed diff of one doc into runs of changed lines of
// its body, by the doc's old line numbers; the header (`checked`, `judged`)
// is left out.
func bodyBlocks(old string, f fileDiff) []citedBlock {
	_, head := header(old)
	var out []citedBlock
	for hi, h := range f.hunks {
		n := h.oldStart
		if h.oldCount == 0 {
			n++ // inserted after that line
		}
		var cur *citedBlock
		flush := func() {
			if cur != nil && len(cur.removed)+len(cur.added) > 0 {
				out = append(out, *cur)
			}
			cur = nil
		}
		for li, l := range h.lines {
			if l[0] == ' ' {
				flush()
				n++
				continue
			}
			if n <= head { // the header: who checked the doc, not what it says
				if l[0] == '-' {
					n++
				}
				continue
			}
			if cur == nil {
				cur = &citedBlock{from: n, to: n - 1, hunk: hi}
			}
			cur.at = append(cur.at, li)
			switch l[0] {
			case '-':
				cur.removed = append(cur.removed, l[1:])
				cur.to = n
				n++
			case '+':
				cur.added = append(cur.added, l[1:])
			}
		}
		flush()
	}
	return out
}

// wordOf is a word as the removal rule counts it: letters and digits, with
// the dots, dashes and apostrophes inside a version or a name.
var wordOf = regexp.MustCompile(`[\p{L}\p{N}]+(?:[._'’-][\p{L}\p{N}]+)*`)

// wordsTakenOut are the words a block's removed lines say more often than
// its added lines, case aside: what the fix takes out. Lines rewrapped, or a
// sentence moved within the block, take nothing out.
func wordsTakenOut(b citedBlock) []string {
	count := map[string]int{}
	for _, l := range b.added {
		for _, w := range wordOf.FindAllString(l, -1) {
			count[strings.ToLower(w)]++
		}
	}
	var out []string
	for _, l := range b.removed {
		for _, w := range wordOf.FindAllString(l, -1) {
			if lw := strings.ToLower(w); count[lw] > 0 {
				count[lw]--
			} else {
				out = append(out, w)
			}
		}
	}
	return out
}

// citeClaim is a claim given beside a patch: a part's claim, and the doc it
// speaks of when the answer patches several, or the name it finds gone.
type citeClaim struct {
	claim `yaml:",inline"`
	Doc   string `yaml:"doc"`
	Name  string `yaml:"name"`
}

// readCitations reads the claims of an answer; those that cannot be read
// are counted.
func readCitations(intents []intent.Intention) (out []citeClaim, unread int) {
	for _, in := range intents {
		if in.Kind != "claim" {
			continue
		}
		data, _ := yaml.Marshal(in.Value)
		var c citeClaim
		if err := yaml.Unmarshal(data, &c); err != nil || c.Lines.from == 0 {
			unread++
			continue
		}
		c.Doc = path.Clean(strings.TrimPrefix(c.Doc, "./"))
		out = append(out, c)
	}
	return out, unread
}

// citeContext is what the engine checks a claim against.
type citeContext struct {
	repo  string
	files map[string]bool // tracked in the repository
	pl    *places
	shown map[string]string // file -> content, read once
}

// evidence is what one claim shows: "" when it is evidence, else why not;
// comment is set when its quote was found, but only in a comment.
type citeCheck struct {
	ok      bool
	why     string
	comment string // path:line of the comment its quote is in
}

// check reads one claim against the doc's sources: a quote outside any
// comment of a file under them, or a name the code had when the doc was
// last edited and has no more, said by the lines the block removes.
func (cc *citeContext) check(c citeClaim, d *Doc, b citedBlock) citeCheck {
	switch c.Status {
	case "gone":
		name := strings.Trim(c.Name, "`() ")
		if name == "" {
			return citeCheck{why: "a `gone` claim names the name the code no longer has, in `name`"}
		}
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(strings.Join(b.removed, "\n")) {
			return citeCheck{why: fmt.Sprintf("`%s` is not in the lines the fix removes", name)}
		}
		now, err := inCode(cc.repo, "HEAD", map[string]bool{name: true})
		if err != nil || now[name] {
			return citeCheck{why: fmt.Sprintf("`%s` is still in the code", name)}
		}
		last, err := git(cc.repo, "log", "-1", "--format=%H", "--", d.Path)
		if err != nil || last == "" {
			return citeCheck{why: fmt.Sprintf("`%s` cannot be found in the code when the doc was last edited", name)}
		}
		then, err := inCode(cc.repo, last, map[string]bool{name: true})
		if err != nil || !then[name] {
			return citeCheck{why: fmt.Sprintf("`%s` was not in the code when the doc was last edited (%.7s): nothing says it went", name, last)}
		}
		return citeCheck{ok: true}
	case "contradicted":
		if c.Source == nil || strings.TrimSpace(c.Source.Quote) == "" || c.Source.Path == "" {
			return citeCheck{why: "a `contradicted` claim gives its source's `path` and `quote`"}
		}
		p := strings.TrimPrefix(path.Clean(strings.TrimPrefix(c.Source.Path, "./")), "b/")
		content, ok := cc.source(d, p)
		if !ok {
			return citeCheck{why: fmt.Sprintf("%s is not a file under the doc's sources (%s)", c.Source.Path, strings.Join(d.Sources, ", "))}
		}
		found, inCodeToo, line := quoteIn(p, content, c.Source.Quote)
		switch {
		case !found && countIn.MatchString(c.Source.Quote):
			return citeCheck{why: fmt.Sprintf("%q is not in %s: a line count is not words a file holds; a count the engine reports off needs no claim, its count being the evidence", c.Source.Quote, p)}
		case !found:
			return citeCheck{why: fmt.Sprintf("%q is not in %s, as it is now", c.Source.Quote, p)}
		case !inCodeToo:
			return citeCheck{why: fmt.Sprintf("%q is a comment (%s:%d), not the code", c.Source.Quote, p, line), comment: fmt.Sprintf("%s:%d", p, line)}
		}
		return citeCheck{ok: true}
	}
	return citeCheck{why: fmt.Sprintf("status %q cites nothing: a claim beside a patch is `contradicted`, with a source, or `gone`, with a name", c.Status)}
}

// source returns a file under one of the doc's sources, as it is now; for
// another repository's source, the path is `name:file`.
func (cc *citeContext) source(d *Doc, p string) (string, bool) {
	repoName, file := "", p
	if n, f, ok := strings.Cut(p, ":"); ok && !strings.Contains(n, "/") {
		repoName, file = n, f
	}
	under := false
	for _, src := range d.Sources {
		name, sp, _ := splitSource(src)
		sp = strings.TrimSuffix(path.Clean(sp), "/")
		if name == repoName && (file == sp || sp == "." || strings.HasPrefix(file, sp+"/")) {
			under = true
		}
	}
	if !under {
		return "", false
	}
	if c, ok := cc.shown[p]; ok {
		return c, true
	}
	var content string
	if repoName == "" {
		if !cc.files[file] {
			return "", false
		}
		data, err := os.ReadFile(filepath.Join(cc.repo, filepath.FromSlash(file)))
		if err != nil {
			return "", false
		}
		content = string(data)
	} else {
		where, err := cc.pl.get(repoName)
		if err != nil {
			return "", false
		}
		if content, err = git(where.dir, "show", where.rev+":"+file); err != nil {
			return "", false
		}
	}
	cc.shown[p] = content
	return content, true
}

// judgeCitations checks that each run of a doc's body a fix changes is
// cited where it takes words out, and that no change rests on a comment
// alone. It returns the refusal, if any, and the comment it rests on.
func (cc *citeContext) judgeCitations(docPath, old string, f fileDiff, claims []citeClaim, unread int, alone bool) (rule, msg string, comments []verdict.Finding) {
	refused, held, off, comments := cc.citationRefusals(docPath, old, f, claims, unread, alone)
	if len(refused) == 0 {
		return "", "", nil
	}
	rule, msg = citationMessage(docPath, refused, held, off)
	return rule, msg, comments
}

// citationMessage says why a doc's fix is refused by the removal rule, as
// the agent is told: each place refused, the counts the engine fixes with
// no claim, and the places that hold.
func citationMessage(docPath string, refused []placeRefusal, held []string, off []countOff) (rule, msg string) {
	var refusals []string
	for _, r := range refused {
		if rule == "" || r.rule == "comment-not-evidence" {
			rule = r.rule
		}
		refusals = append(refusals, r.why)
	}
	msg = strings.Join(refusals, "; ") + "; give a `claim` beside the patch for each place, with the doc's lines and a source's words quoted from a file under its sources (`status: contradicted`, `source: {path, quote}`), or a name the code no longer has (`status: gone`, `name`); with nothing to cite, leave the words, and say in a note what you could not confirm"
	if len(off) > 0 {
		var said []string
		for _, c := range off {
			said = append(said, fmt.Sprintf("line %d, %s to %d", c.line, c.stated, c.real))
		}
		msg += ". A line count the engine reported off needs no claim: the engine's count is the evidence, so bring each to its number (" + strings.Join(said, "; ") + ") and keep that change"
	}
	// Only what is refused is named: the agent once withdrew a right fix
	// with the refused one beside it (16c660b, ADR-0014 step 3).
	if len(held) > 0 {
		msg += fmt.Sprintf(". Only the places named are refused: the rest of this patch of %s holds (%s); send it again unchanged", docPath, strings.Join(held, ", "))
	}
	return rule, msg
}

// placeRefusal is one run of a doc's changed lines refused by the removal
// rule, and why: the rest of the doc's fix may hold without it.
type placeRefusal struct {
	block     citedBlock
	place     string // its lines, as "line 12" or "lines 12-14"
	rule, why string
}

// citationRefusals reads each run of a doc's body a fix changes against
// the claims given: the runs refused, with their rule and why; the places
// that hold; the line counts the engine reports off in the doc; and the
// comments given as the only reason for a fix, for a person.
func (cc *citeContext) citationRefusals(docPath, old string, f fileDiff, claims []citeClaim, unread int, alone bool) (refused []placeRefusal, held []string, off []countOff, comments []verdict.Finding) {
	d, _ := ParseDoc(docPath, []byte(old))
	if d == nil {
		d = &Doc{Path: docPath}
	}
	// A line count the engine counted off is cited by the engine itself,
	// when the fix brings it to the engine's number.
	counted := map[int][]countOff{}
	off = countsOff(cc.repo, cc.files, d, old)
	for _, c := range off {
		counted[c.line] = append(counted[c.line], c)
	}
	for _, b := range bodyBlocks(old, f) {
		out := uncited(b, counted)
		at := lineRange{from: b.from, to: max(b.from, b.to)}
		var why []string
		var commentAt []string
		cited, any := false, false
		for _, c := range claims {
			if c.Doc != docPath && (c.Doc != "." || !alone) {
				continue
			}
			wide := lineRange{from: max(1, c.Lines.from-placeWithin), to: c.Lines.to + placeWithin}
			if !wide.overlaps(at) {
				continue
			}
			any = true
			r := cc.check(c, d, b)
			if r.ok {
				cited = true
				break
			}
			why = append(why, r.why)
			if r.comment != "" {
				commentAt = append(commentAt, r.comment)
			}
		}
		place := at.String()
		if cited || len(out) == 0 && (!any || len(commentAt) == 0) {
			held = append(held, place)
			continue
		}
		r := placeRefusal{block: b, place: place}
		switch {
		case len(commentAt) > 0:
			r.rule = "comment-not-evidence"
			r.why = fmt.Sprintf("the fix of %s rests on a comment only (%s): a comment is not evidence, the code is; keep the doc's words unless the code itself says otherwise — the disagreement between the comment and the code is reported for a person", place, strings.Join(commentAt, ", "))
			for _, ca := range commentAt {
				comments = append(comments, verdict.Finding{Rule: "comment-disagrees", Where: strings.SplitN(ca, ":", 2)[0],
					Message: fmt.Sprintf("the comment at %s was given as the only reason to change %s %s; a comment is not evidence: if the code says otherwise, the comment is stale — fix it", ca, docPath, place)})
			}
		case any:
			r.rule = "citation-unchecked"
			r.why = fmt.Sprintf("the claim for %s does not hold: %s", place, strings.Join(why, "; "))
		default:
			r.rule = "removal-uncited"
			verb := "takes"
			if at.from != at.to {
				verb = "take"
			}
			r.why = fmt.Sprintf("%s %s out %s with no claim saying why", place, verb, quoteWords(out))
			if unread > 0 {
				r.why += fmt.Sprintf(" (%d claims could not be read)", unread)
			}
		}
		refused = append(refused, r)
	}
	return refused, held, off, comments
}

// quoteWords says the words taken out, a few of them.
func quoteWords(w []string) string {
	if len(w) > 8 {
		return fmt.Sprintf("%q and %d more words", strings.Join(w[:8], " "), len(w)-8)
	}
	return fmt.Sprintf("%q", strings.Join(w, " "))
}

// quoteIn finds a quote in a file, whitespace aside: found anywhere, found
// with at least one character of it outside a comment, and the line its
// first finding starts on.
func quoteIn(p, content, quote string) (found, code bool, line int) {
	q := strings.Join(strings.Fields(quote), " ")
	if q == "" {
		return false, false, 0
	}
	mask := commentMask(p, content)
	// The content with each run of white space made one space, each byte
	// remembering whether it is in a comment, and its line.
	var norm []byte
	var inComment []bool
	var lineOf []int
	ln, space := 1, true
	for i := 0; i < len(content); i++ {
		c := content[i]
		if c == '\n' || c == ' ' || c == '\t' || c == '\r' {
			if !space {
				norm, inComment, lineOf = append(norm, ' '), append(inComment, true), append(lineOf, ln)
				space = true
			}
			if c == '\n' {
				ln++
			}
			continue
		}
		space = false
		norm, inComment, lineOf = append(norm, c), append(inComment, mask[i]), append(lineOf, ln)
	}
	s := string(norm)
	for from := 0; ; {
		i := strings.Index(s[from:], q)
		if i < 0 {
			return found, code, line
		}
		i += from
		if !found {
			found, line = true, lineOf[i]
		}
		for j := i; j < i+len(q); j++ {
			if !inComment[j] {
				return true, true, line
			}
		}
		from = i + 1
	}
}

// commentStyle is how a file writes its comments, by its name.
type commentStyle struct {
	line           []string // a line comment starts with one of these
	block          [][2]string
	hashAfterSpace bool // `#` starts a comment only at the start or after white space
	backtick       bool // backticks quote, over several lines (Go, JavaScript)
}

func styleOf(p string) commentStyle {
	base := strings.ToLower(path.Base(p))
	ext := strings.TrimPrefix(path.Ext(base), ".")
	cBlock := [][2]string{{"/*", "*/"}}
	switch ext {
	case "go", "js", "mjs", "cjs", "ts", "tsx", "jsx":
		return commentStyle{line: []string{"//"}, block: cBlock, backtick: true}
	case "c", "h", "cpp", "hpp", "cc", "cxx", "hh", "ino", "java", "kt", "kts", "rs", "swift", "cs", "scala", "dart", "proto", "groovy", "gradle":
		return commentStyle{line: []string{"//"}, block: cBlock}
	case "php":
		return commentStyle{line: []string{"//", "#"}, block: cBlock}
	case "css", "scss", "less":
		return commentStyle{block: cBlock}
	case "md", "markdown", "html", "htm", "xml", "svg", "vue":
		return commentStyle{block: [][2]string{{"<!--", "-->"}}}
	case "yaml", "yml", "sh", "bash", "zsh", "py", "rb", "toml", "cfg", "conf", "pl", "r", "mk", "cmake", "env", "properties", "dockerfile":
		return commentStyle{line: []string{"#"}, hashAfterSpace: true}
	case "ini":
		return commentStyle{line: []string{"#", ";"}, hashAfterSpace: true}
	case "tf", "hcl":
		return commentStyle{line: []string{"#", "//"}, block: cBlock, hashAfterSpace: true}
	}
	switch {
	case base == "makefile" || base == "dockerfile" || base == "cmakelists.txt" || strings.HasPrefix(base, ".git") || strings.HasPrefix(base, ".env"):
		return commentStyle{line: []string{"#"}, hashAfterSpace: true}
	}
	return commentStyle{} // JSON, text: no comment
}

// commentMask says, for each byte of a file, whether it is in a comment,
// by the file's type: `//` and `/* */`, `#`, `<!-- -->`. Strings are read
// as strings: a `//` in a URL between quotes is no comment.
func commentMask(p, content string) []bool {
	st := styleOf(p)
	mask := make([]bool, len(content))
	mark := func(from, to int) {
		for k := from; k < to && k < len(mask); k++ {
			mask[k] = true
		}
	}
	lineStart := true
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
			if l == "#" && st.hashAfterSpace && !lineStart && i > 0 && !unicode.IsSpace(rune(content[i-1])) {
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
		// A string: skipped whole when it closes, on its line (or, for a
		// backtick, anywhere); an apostrophe that closes nothing is a letter.
		if len(st.line)+len(st.block) > 0 && (c == '"' || c == '\'' || c == '`' && st.backtick) {
			if end := closing(content, i, c); end > 0 {
				i, lineStart = end+1, false
				continue
			}
		}
		if !unicode.IsSpace(rune(c)) {
			lineStart = false
		}
		i++
	}
	return mask
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

// sortedFindings keeps the comment reports in one order, without repeats.
func sortedFindings(f []verdict.Finding) []verdict.Finding {
	seen := map[string]bool{}
	var out []verdict.Finding
	for _, x := range f {
		if k := x.Rule + x.Where + x.Message; !seen[k] {
			seen[k] = true
			out = append(out, x)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Where < out[j].Where })
	return out
}

// newCiteContext reads what claims are checked against: the repository's
// files, and where the other repositories it names are read.
func newCiteContext(repo string, files map[string]bool) (*citeContext, error) {
	var cfg projectRepos
	if data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}
	gitDir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, err
	}
	return &citeContext{repo: repo, files: files, shown: map[string]string{},
		pl: &places{repo: repo, gitDir: gitDir, cfg: cfg, known: map[string]place{}}}, nil
}

// uncited are the words a block takes out that need a claim. None do that
// the engine's count stands for (countedOn). Where the block replaces line
// for line, none on a line reworded taking out only words that carry no fact
// (glue); and on a line whose count the engine found off and the block
// brings to its number, none said around the count — the count changed what
// the line says, and the words beside it follow (DomoticsCore, ADR-0014
// step 4: "Watch the 800-line limit" became "Over the 800-line hard limit"
// beside 930 lines) — but for a fact of the line's own (factOn): the count
// stands for itself, not for a version or a "not" beside it.
func uncited(b citedBlock, counted map[int][]countOff) []string {
	var out []string
	for _, w := range wordsTakenOut(b) {
		if !countedOn(counted, b, w) {
			out = append(out, w)
		}
	}
	if len(out) == 0 || len(b.removed) != len(b.added) {
		return out
	}
	free := map[string]int{}
	for i := range b.removed {
		line := citedBlock{from: b.from + i, to: b.from + i, removed: b.removed[i : i+1], added: b.added[i : i+1]}
		words := wordsTakenOut(line)
		if onlyGlue(words) {
			for _, w := range words {
				free[w]++
			}
			continue
		}
		if !countFixedOn(counted, line) {
			continue
		}
		for _, w := range words {
			if !factOn(b.removed[i], w) {
				free[w]++
			}
		}
	}
	var kept []string
	for _, w := range out {
		if free[w] > 0 {
			free[w]--
			continue
		}
		kept = append(kept, w)
	}
	return kept
}

// countFixedOn says whether a one-line block brings a count the engine found
// off on its line to the engine's number.
func countFixedOn(counted map[int][]countOff, line citedBlock) bool {
	for _, c := range counted[line.from] {
		for _, a := range numberIn.FindAllString(line.added[0], -1) {
			if strings.ReplaceAll(a, ",", "") == strconv.Itoa(c.real) {
				return true
			}
		}
	}
	return false
}

// glue are words that carry no fact of their own: taken out of a line, with
// every other word kept, the line says the same. Never a fact word.
var glue = map[string]bool{"a": true, "an": true, "the": true, "of": true, "to": true, "in": true, "on": true,
	"at": true, "for": true, "per": true, "with": true, "by": true, "from": true, "as": true, "it": true,
	"its": true, "this": true, "that": true, "these": true, "those": true, "which": true, "currently": true}

// onlyGlue says whether words, some, are all glue.
func onlyGlue(words []string) bool {
	for _, w := range words {
		if !glue[strings.ToLower(w)] {
			return false
		}
	}
	return len(words) > 0
}

// factWords change what a line says when taken out, whatever words stand
// around them: a negation, a quantifier, a conjunction, a tense or a mood.
var factWords = wordSet(`not no never none nor nothing nobody neither cannot without
	all every each some any many much few fewer more most less least only both either several
	always often sometimes rarely usually once twice again also still yet already just
	and or but if unless except than because
	is are was were be been being has have had do does did will would shall should can could
	may might must`)

// wordSet is the set of the words given.
func wordSet(words string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.Fields(words) {
		set[w] = true
	}
	return set
}

// factOn says whether a word taken out of a line is a fact of its own, not
// words said around a count: a number or a version (the count is not taken
// out, countedOn having let it go), a fact word or one negated ("isn't"), or
// a name — quoted as code, shaped as one ("ClockWebUI", "Clock.h"), or
// capitalised past a sentence's start ("Arduino").
func factOn(line, w string) bool {
	lw := strings.ToLower(w)
	if factWords[lw] || strings.HasSuffix(lw, "n't") || strings.HasSuffix(lw, "n’t") ||
		strings.ContainsAny(w, "0123456789.") {
		return true
	}
	for _, r := range []rune(w)[1:] {
		if unicode.IsUpper(r) {
			return true
		}
	}
	for _, at := range wordOf.FindAllStringIndex(line, -1) {
		if line[at[0]:at[1]] != w {
			continue
		}
		if strings.Count(line[:at[0]], "`")%2 == 1 {
			return true
		}
		if first, _ := utf8.DecodeRuneInString(w); unicode.IsUpper(first) && !sentenceStart(line[:at[0]]) {
			return true
		}
	}
	return false
}

// sentenceStart says whether a word after these words of its line begins a
// sentence: nothing before it but markup, or a full stop, a colon or a
// cell's bar.
func sentenceStart(before string) bool {
	before = strings.TrimRight(before, " \t*_#>([\"'-~")
	if before == "" {
		return true
	}
	last, _ := utf8.DecodeLastRuneInString(before)
	return strings.ContainsRune(".!?:;|", last)
}

// numberIn is a number as a doc writes a count: "569", "1,008".
var numberIn = regexp.MustCompile(`\d{1,3}(?:,\d{3})+|\d+`)

// countedOn says whether a word a block takes out is a line count the
// engine found off on one of the lines it removes, or the word making it a
// rough one ("about", "approximately"), the block writing the engine's
// number in its place: the engine's count is the evidence, no claim needed.
func countedOn(counted map[int][]countOff, b citedBlock, w string) bool {
	added := map[string]bool{}
	for _, l := range b.added {
		for _, a := range numberIn.FindAllString(l, -1) {
			added[strings.ReplaceAll(a, ",", "")] = true
		}
	}
	for n := b.from; n <= b.to; n++ {
		for _, c := range counted[n] {
			if !added[strconv.Itoa(c.real)] {
				continue
			}
			for _, x := range wordOf.FindAllString(c.stated+" "+c.approx, -1) {
				if x == w {
					return true
				}
			}
		}
	}
	return false
}
